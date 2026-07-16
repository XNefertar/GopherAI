package aihelper

import (
	appconfig "GopherAI/config"
	"GopherAI/dao/session"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// ============================================================
// 三层渐进式混合路由器
//
// 设计思想：
//   传统关键词路由（RuleBasedRouter）对语义不敏感。
//   LLMClassifierRouter 采用 L1→L2→L3 三层渐进架构，按开销从低到高逐层决策：
//
//   L1: Embedding 快速匹配（~5ms, 免费）— 向量相似度匹配高置信度意图
//       拦截 60-70% 的简单/明确 query，直接路由
//   L2: 轻量 LLM 语义分类（~200ms, Ollama 免费）— Prompt 驱动的意图分类 + Query 改写
//       处理 L1 无法确定的模糊/混合意图
//   L3: 规则路由器兜底（~0ms）— 关键词 + 长度启发式
//       保证任何情况下都有路由决策
//
// 加权平均延迟 ≈ 65%×5ms + 25%×200ms + 10%×0ms ≈ 53ms
//
// 成本与延迟：
//   - L1 Embedding 复用 RAG 基础设施，单次调用约 0.0001 元
//   - L2 分类模型优先使用本地 Ollama（免费、低延迟）
//   - L1/L2 任一失败均自动降级到下一层
// ============================================================

// IntentLabel 意图标签，对应不同模型类型。
type IntentLabel string

const (
	IntentChat      IntentLabel = "chat"      // 闲聊/问候 → 低成本模型（Ollama/CheapModel）
	IntentKnowledge IntentLabel = "knowledge" // 文档/知识库 → RAG
	IntentTool      IntentLabel = "tool"      // 工具调用 → MCP
	IntentReasoning IntentLabel = "reasoning" // 复杂推理 → 主模型（OpenAI）
)

// IntentClassification LLM 分类器输出的结构化意图分类结果。
type IntentClassification struct {
	Intent         IntentLabel `json:"intent"`                   // 意图标签
	Confidence     float64     `json:"confidence"`               // 置信度 [0.0, 1.0]
	RewrittenQuery string      `json:"rewritten_query,omitempty"` // 改写后的 query（方言→规范表达、歧义消解）
	Reason         string      `json:"reason"`                   // 分类理由（可观测性）
}

// classificationPrompt 分类提示词模板。
// 设计要点：
//   - 要求输出严格 JSON，便于解析
//   - rewritten_query 用于将口语化/模糊表达改写为更精准的检索/推理 query
//   - 分类粒度与模型类型一一对应，便于后续规则链决策
const classificationPrompt = `You are a query intent classifier. Analyze the user's query and output a JSON classification.

# Categories
- "chat": Casual greetings, thanks, small talk, very simple factual questions that don't need deep reasoning.
- "knowledge": Questions that reference documents, knowledge bases, uploaded files, manuals, or need information retrieval from a document store.
- "tool": Requests to invoke external tools, call APIs, query systems, execute commands, or interact with external services.
- "reasoning": Complex analysis, coding, math, logic, comparison, design, refactoring, or any task requiring deep multi-step reasoning.

# Rules
1. If the user explicitly mentions documents/files/knowledge bases → "knowledge"
2. If the user asks to invoke/call/execute/query external tools or systems → "tool"
3. If the query requires deep analysis, coding, math, logical reasoning, or detailed comparison → "reasoning"
4. Simple greetings, thanks, very short small talk → "chat"
5. For ambiguous cases, prefer "reasoning" when the question involves explanation or analysis; prefer "chat" when it's a simple fact or social interaction.

# Query Rewriting (rewritten_query)
- If the user uses colloquial/vague language, rewrite it to a clearer, more specific form.
- If the user's query references context from previous turns (e.g. "what about the second one?"), expand it.
- If the query is already clear and specific, set rewritten_query to an empty string.
- The rewritten query will be used for downstream retrieval/reasoning, so make it self-contained.

# Output Format
Return ONLY a valid JSON object, no markdown fences, no extra text:
{"intent":"<category>","confidence":<0.0-1.0>,"rewritten_query":"<rewritten or empty>","reason":"<brief explanation>"}`

// classifierUserMsg 构建分类器的用户消息。
func classifierUserMsg(query string) string {
	return fmt.Sprintf("User query: %s\n\nClassify:", query)
}

// LLMClassifierRouter 三层渐进式混合路由器（L1 Embedding + L2 LLM + L3 规则）。
type LLMClassifierRouter struct {
	// L1: Embedding 快速意图匹配器（优先使用，5ms 延迟）。
	embedMatcher *EmbeddingIntentMatcher

	// L3 fallback 规则路由器，用于关键词短路和最终兜底。
	fallback *RuleBasedRouter

	// L2 classifierLLM 意图分类用的轻量 LLM（优先 Ollama 本地模型）。
	classifierLLM model.ToolCallingChatModel

	// classifierType 分类器使用的 LLM 类型（"ollama" / "openai"），用于日志。
	classifierType string

	// timeout 分类 LLM 调用的超时时间，超时则降级。
	timeout time.Duration

	// enableRewrite 是否启用 query 改写（改写后的 query 会替换原 question 传给下游模型）。
	enableRewrite bool

	// confidenceThreshold 置信度阈值：低于此阈值时忽略 LLM 分类结果，走 fallback。
	confidenceThreshold float64

	// 统计指标（用于可观测性）
	mu    sync.RWMutex
	stats RouterStats
}

// RouterStats 路由器运行统计。
type RouterStats struct {
	EmbeddingHit   int64 // L1 Embedding 命中次数（高置信度直接路由）
	EmbeddingMiss  int64 // L1 Embedding 未命中次数（透传到 L2）
	LLMClassified  int64 // L2 LLM 成功分类次数
	LLMFallback    int64 // L2 LLM 失败/超时降级次数
	KeywordShortcut int64 // 关键词短路次数
	LowConfidence  int64 // 低置信度回退次数
}

// NewLLMClassifierRouter 构造三层渐进式混合路由器。
//
// 初始化顺序：
//  L1: Embedding 意图匹配器（复用 RAG 基础设施，若未配置或禁用则跳过）
//  L2: 轻量 LLM 分类器（优先 Ollama，回退 OpenAI，若未配置或禁用则跳过）
//  L3: 规则路由器（始终可用，最终兜底）
func NewLLMClassifierRouter(ctx context.Context) *LLMClassifierRouter {
	cfg := appconfig.GetConfig().Router

	r := &LLMClassifierRouter{
		fallback:             NewRuleBasedRouter(),
		timeout:              time.Duration(cfg.LLMClassifierTimeoutOrDefaultMs()) * time.Millisecond,
		enableRewrite:        cfg.LLMRewriteEnabledOrDefault(),
		confidenceThreshold:  cfg.LLMConfidenceThresholdOrDefault(),
	}

	// L1: 初始化 Embedding 快速匹配器（若 RAG Embedding 未配置则静默跳过）
	r.embedMatcher = NewEmbeddingIntentMatcher(ctx)

	// L2: 尝试创建分类器（若被配置禁用则跳过）
	if cfg.LLMClassifierEnabledOrDefault() {
		if llm, err := newClassifierOllama(ctx); err == nil {
			r.classifierLLM = llm
			r.classifierType = "ollama"
			log.Printf("[router] L2 LLM classifier initialized: type=ollama")
		} else if llm, err := newClassifierOpenAI(ctx); err == nil {
			r.classifierLLM = llm
			r.classifierType = "openai"
			log.Printf("[router] L2 LLM classifier initialized: type=openai (ollama not available)")
		} else {
			log.Printf("[router] L2 LLM classifier unavailable, falling back to rule-based routing only")
		}
	} else {
		log.Printf("[router] L2 LLM classifier disabled by config")
	}

	// 输出初始化摘要
	l1Status := "disabled"
	if r.embedMatcher != nil {
		l1Status = "enabled"
	}
	l2Status := "disabled"
	if r.classifierLLM != nil {
		l2Status = fmt.Sprintf("enabled(%s)", r.classifierType)
	}
	log.Printf("[router] three-tier hybrid router ready: L1=%s L2=%s L3=enabled(always) timeout(l1=embed,l2=%v) confidence=%.2f rewrite=%v",
		l1Status, l2Status, r.timeout, r.confidenceThreshold, r.enableRewrite)

	return r
}

// newClassifierOllama 尝试创建 Ollama 分类模型。
func newClassifierOllama(ctx context.Context) (model.ToolCallingChatModel, error) {
	mc := appconfig.GetConfig().Model
	if mc.OllamaModelName == "" || mc.OllamaBaseURL == "" {
		return nil, fmt.Errorf("ollama not configured")
	}
	return ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: mc.OllamaBaseURL,
		Model:   mc.OllamaModelName,
	})
}

// newClassifierOpenAI 尝试创建 OpenAI 兼容分类模型。
func newClassifierOpenAI(ctx context.Context) (model.ToolCallingChatModel, error) {
	mc := appconfig.GetConfig().Model
	if mc.OpenAIKey == "" || mc.OpenAIModel == "" || mc.OpenAIBaseURL == "" {
		return nil, fmt.Errorf("openai not configured")
	}
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  mc.OpenAIKey,
		Model:   mc.OpenAIModel,
		BaseURL: mc.OpenAIBaseURL,
	})
}

// Route 执行三层渐进式路由：L1 Embedding → L2 LLM → L3 规则。
func (r *LLMClassifierRouter) Route(ctx context.Context, userName, sessionID, question string, stream bool) (RouteDecision, error) {
	q := strings.TrimSpace(question)

	// ═══════════════════════════════════════════════
	// Step 0: 关键词短路 — 极其明显的问候/告别，跳过 L1/L2
	// ═══════════════════════════════════════════════
	if r.fallback.isTrivialShortcut(q) {
		return r.routeTrivial(ctx, userName, sessionID, q, stream)
	}

	// ═══════════════════════════════════════════════
	// L1: Embedding 快速意图匹配（~5ms）
	// ═══════════════════════════════════════════════
	if match := r.embedMatcher.Match(ctx, q); match != nil {
		r.mu.Lock()
		r.stats.EmbeddingHit++
		r.mu.Unlock()
		return r.routeByEmbeddingMatch(ctx, userName, sessionID, q, match)
	}
	r.mu.Lock()
	r.stats.EmbeddingMiss++
	r.mu.Unlock()

	// ═══════════════════════════════════════════════
	// L2: 轻量 LLM 语义分类（~200ms）
	// ═══════════════════════════════════════════════
	classification, err := r.classify(ctx, q)
	if err != nil {
		// 分类失败 → 降级到 L3 规则路由器
		r.mu.Lock()
		r.stats.LLMFallback++
		r.mu.Unlock()
		log.Printf("[router] L2 LLM classify failed, fallback to L3 rules: %v", err)
		return r.fallback.Route(ctx, userName, sessionID, q, stream)
	}

	r.mu.Lock()
	r.stats.LLMClassified++
	r.mu.Unlock()

	// 置信度检查 — 低置信度回退到 L3 规则路由
	if classification.Confidence < r.confidenceThreshold {
		r.mu.Lock()
		r.stats.LowConfidence++
		r.mu.Unlock()
		log.Printf("[router] L2 low confidence %.2f < %.2f, fallback to L3 rules",
			classification.Confidence, r.confidenceThreshold)
		return r.fallback.Route(ctx, userName, sessionID, q, stream)
	}

	// 使用改写后的 query（如果可用）
	effectiveQuery := q
	if r.enableRewrite && classification.RewrittenQuery != "" {
		effectiveQuery = classification.RewrittenQuery
		log.Printf("[router] L2 query rewritten: original=%q → rewritten=%q",
			truncate(q, 80), truncate(effectiveQuery, 80))
	}

	// 根据意图标签做路由决策
	return r.routeByIntent(ctx, userName, sessionID, effectiveQuery, stream, classification)
}

// classify 调用轻量 LLM 对 query 做意图分类。
func (r *LLMClassifierRouter) classify(ctx context.Context, query string) (*IntentClassification, error) {
	if r.classifierLLM == nil {
		return nil, fmt.Errorf("no classifier LLM available")
	}

	// 带超时的 context
	classifyCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	messages := []*schema.Message{
		{Role: schema.System, Content: classificationPrompt},
		{Role: schema.User, Content: classifierUserMsg(query)},
	}

	resp, err := r.classifierLLM.Generate(classifyCtx, messages)
	if err != nil {
		return nil, fmt.Errorf("classifier LLM generate: %w", err)
	}

	return parseClassification(resp.Content)
}

// parseClassification 从 LLM 原始输出中解析意图分类 JSON。
func parseClassification(raw string) (*IntentClassification, error) {
	// 清理可能的 markdown 代码块包裹
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var c IntentClassification
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return nil, fmt.Errorf("parse classification JSON: %w (raw=%s)", err, truncate(raw, 200))
	}

	// 校验意图标签合法性
	switch c.Intent {
	case IntentChat, IntentKnowledge, IntentTool, IntentReasoning:
		// valid
	default:
		return nil, fmt.Errorf("unknown intent: %s", c.Intent)
	}

	return &c, nil
}

// routeByEmbeddingMatch 根据 L1 Embedding 匹配结果做模型选择。
func (r *LLMClassifierRouter) routeByEmbeddingMatch(
	ctx context.Context, userName, sessionID, question string,
	match *EmbeddingMatchResult,
) (RouteDecision, error) {
	sessionObj, err := session.GetSessionByID(ctx, sessionID)
	if err != nil {
		return RouteDecision{}, err
	}

	switch match.Intent {
	case IntentTool:
		opts, err := buildOptionsForType(ModelTypeMCP, userName, sessionObj.ActiveKBID)
		if err == nil {
			return RouteDecision{
				ModelType: ModelTypeMCP,
				Reason:    fmt.Sprintf("l1_emb:tool(%.3f,Δ%.3f)", match.Score, match.Margin),
				Options:   opts,
			}, nil
		}
	case IntentKnowledge:
		opts, err := buildOptionsForType(ModelTypeRAG, userName, sessionObj.ActiveKBID)
		if err == nil {
			return RouteDecision{
				ModelType: ModelTypeRAG,
				Reason:    fmt.Sprintf("l1_emb:knowledge(%.3f,Δ%.3f)", match.Score, match.Margin),
				Options:   opts,
			}, nil
		}
	case IntentReasoning:
		opts, err := buildOptionsForType(r.fallback.DefaultModelType, userName, sessionObj.ActiveKBID)
		if err == nil {
			return RouteDecision{
				ModelType: r.fallback.DefaultModelType,
				Reason:    fmt.Sprintf("l1_emb:reasoning(%.3f,Δ%.3f)", match.Score, match.Margin),
				Options:   opts,
			}, nil
		}
	case IntentChat:
		if appconfig.GetConfig().Model.OllamaModelName != "" {
			opts, err := buildOptionsForType(ModelTypeOllama, userName, sessionObj.ActiveKBID)
			if err == nil {
				return RouteDecision{
					ModelType: ModelTypeOllama,
					Reason:    fmt.Sprintf("l1_emb:chat→local(%.3f,Δ%.3f)", match.Score, match.Margin),
					Options:   opts,
				}, nil
			}
		}
		opts, err := buildOptionsForType(r.fallback.CheapModelType, userName, sessionObj.ActiveKBID)
		if err == nil {
			return RouteDecision{
				ModelType: r.fallback.CheapModelType,
				Reason:    fmt.Sprintf("l1_emb:chat(%.3f,Δ%.3f)", match.Score, match.Margin),
				Options:   opts,
			}, nil
		}
	}

	// L1 决策失败（如 options 构建失败），降级到 L3
	return r.fallback.Route(ctx, userName, sessionID, question, false)
}

// routeByIntent 根据 L2 LLM 分类的意图标签做模型选择。
func (r *LLMClassifierRouter) routeByIntent(
	ctx context.Context, userName, sessionID, question string, stream bool,
	c *IntentClassification,
) (RouteDecision, error) {
	sessionObj, err := session.GetSessionByID(ctx, sessionID)
	if err != nil {
		return RouteDecision{}, err
	}

	switch c.Intent {
	case IntentTool:
		opts, err := buildOptionsForType(ModelTypeMCP, userName, sessionObj.ActiveKBID)
		if err == nil {
			return RouteDecision{
				ModelType: ModelTypeMCP,
				Reason:    fmt.Sprintf("llm:tool(%.2f)", c.Confidence),
				Options:   opts,
			}, nil
		}
		log.Printf("[router] MCP options build failed, fallback to default: %v", err)

	case IntentKnowledge:
		opts, err := buildOptionsForType(ModelTypeRAG, userName, sessionObj.ActiveKBID)
		if err == nil {
			return RouteDecision{
				ModelType: ModelTypeRAG,
				Reason:    fmt.Sprintf("llm:knowledge(%.2f)", c.Confidence),
				Options:   opts,
			}, nil
		}
		log.Printf("[router] RAG options build failed, fallback to default: %v", err)

	case IntentReasoning:
		opts, err := buildOptionsForType(r.fallback.DefaultModelType, userName, sessionObj.ActiveKBID)
		if err == nil {
			return RouteDecision{
				ModelType: r.fallback.DefaultModelType,
				Reason:    fmt.Sprintf("llm:reasoning(%.2f)", c.Confidence),
				Options:   opts,
			}, nil
		}

	case IntentChat:
		// 优先走 Ollama 本地免费模型
		if appconfig.GetConfig().Model.OllamaModelName != "" {
			opts, err := buildOptionsForType(ModelTypeOllama, userName, sessionObj.ActiveKBID)
			if err == nil {
				return RouteDecision{
					ModelType: ModelTypeOllama,
					Reason:    fmt.Sprintf("llm:chat→local(%.2f)", c.Confidence),
					Options:   opts,
				}, nil
			}
		}
		opts, err := buildOptionsForType(r.fallback.CheapModelType, userName, sessionObj.ActiveKBID)
		if err == nil {
			return RouteDecision{
				ModelType: r.fallback.CheapModelType,
				Reason:    fmt.Sprintf("llm:chat(%.2f)", c.Confidence),
				Options:   opts,
			}, nil
		}
	}

	// 兜底
	opts, err := buildOptionsForType(r.fallback.DefaultModelType, userName, sessionObj.ActiveKBID)
	if err != nil {
		return RouteDecision{}, err
	}
	return RouteDecision{
		ModelType: r.fallback.DefaultModelType,
		Reason:    fmt.Sprintf("llm:fallback_default(%s,%.2f)", c.Intent, c.Confidence),
		Options:   opts,
	}, nil
}

// ============================================================
// 快速短路 — 极其明显的问候/告别直接走低成本模型，省一次 LLM 调用
// ============================================================

// trivialShortcutKeywords 极其明显的高频问候/告别词（大小写不敏感匹配）。
// 这些词几乎 100% 是闲聊意图，不需要 LLM 来判断。
var trivialShortcutKeywords = []string{
	"你好", "您好", "hi", "hello", "hey", "在吗", "在不在",
	"早上好", "中午好", "下午好", "晚上好", "晚安",
	"good morning", "good afternoon", "good evening",
	"谢谢", "感谢", "thanks", "thank you",
	"再见", "拜拜", "bye", "goodbye", "see you",
}

// isTrivialShortcut 判断 query 是否是极其明显的闲聊（直接字符串匹配，不调用 LLM）。
func (r *RuleBasedRouter) isTrivialShortcut(q string) bool {
	lower := strings.ToLower(strings.TrimSpace(q))
	for _, kw := range trivialShortcutKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// routeTrivial 处理确认是闲聊的请求（直接走低成本模型）。
func (r *LLMClassifierRouter) routeTrivial(
	ctx context.Context, userName, sessionID, question string, stream bool,
) (RouteDecision, error) {
	r.mu.Lock()
	r.stats.KeywordShortcut++
	r.mu.Unlock()

	sessionObj, err := session.GetSessionByID(ctx, sessionID)
	if err != nil {
		return RouteDecision{}, err
	}

	// 优先 Ollama 本地模型
	if appconfig.GetConfig().Model.OllamaModelName != "" {
		opts, err := buildOptionsForType(ModelTypeOllama, userName, sessionObj.ActiveKBID)
		if err == nil {
			return RouteDecision{
				ModelType: ModelTypeOllama,
				Reason:    "shortcut:trivial→local",
				Options:   opts,
			}, nil
		}
	}
	opts, err := buildOptionsForType(r.fallback.CheapModelType, userName, sessionObj.ActiveKBID)
	if err != nil {
		return RouteDecision{}, err
	}
	return RouteDecision{
		ModelType: r.fallback.CheapModelType,
		Reason:    "shortcut:trivial",
		Options:   opts,
	}, nil
}

// ============================================================
// 可观测性
// ============================================================

// Stats 返回路由器的运行统计。
func (r *LLMClassifierRouter) Stats() RouterStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.stats
}

// ============================================================
// 工具函数
// ============================================================

// truncate 截断字符串到指定长度（按 rune 计数）。
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
