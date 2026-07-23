package aihelper

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	appconfig "GopherAI/config"

	embeddingArk "github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/cloudwego/eino/components/embedding"
)

// ============================================================
// Embedding 快速意图匹配器（三层渐进路由的 L1 层）
//
// 设计目标：
//   在 LLM 分类之前，用向量相似度拦截 60-70% 的高置信度 query，
//   将分类延迟从 ~200ms（LLM）降到 ~5ms（纯数学运算）。
//
// 工作原理：
//   1. 离线：为 4 类意图各准备一组"锚点样例 query"，预计算其 Embedding 向量
//   2. 在线：将用户 query Embedding 化 → 与各意图锚点质心算余弦相似度
//   3. 路由：相似度超过阈值（>0.85）且明显高于其他意图 → 直接路由，跳过 L2/L3
//
// 降级策略：
//   - Embedding API 未配置（RAG_EMBEDDING_API_KEY 为空）→ 跳过 L1，不影响可用性
//   - 相似度低于阈值 → 透传到 L2（LLM 分类）
//   - Embedding 调用失败/超时 → 透传到 L2
//
// 成本：每次匹配 1 次 Embedding API 调用（与 RAG 检索共用同一基础设施），
//       单次调用约 0.0001 元人民币，远低于 LLM 分类的 API 费用。
// ============================================================

// anchoringExample 意图锚点样例。
// 每个意图配备 5-8 个典型 query，覆盖口语化表达和规范化表达。
type anchoringExample struct {
	Query  string
	Intent IntentLabel
}

// anchorQueries 精心挑选的意图锚点样例集。
// 选择原则：
//   - 覆盖中英文常见表达
//   - 兼顾口语化和正式表达
//   - 每个意图的样例之间有语义多样性
//   - 避免过于相似的样例（防止中心过度偏向某种表达）
var anchorQueries = []anchoringExample{
	// ---- chat: 闲聊/问候/简单问答 (14 条) ----
	{Intent: IntentChat, Query: "你好"},
	{Intent: IntentChat, Query: "早上好"},
	{Intent: IntentChat, Query: "谢谢你帮我"},
	{Intent: IntentChat, Query: "今天天气怎么样"},
	{Intent: IntentChat, Query: "你是谁"},
	{Intent: IntentChat, Query: "讲个笑话吧"},
	{Intent: IntentChat, Query: "hi there, how are you"},
	{Intent: IntentChat, Query: "你能做什么"},
	{Intent: IntentChat, Query: "好无聊啊聊聊天"},
	{Intent: IntentChat, Query: "今天星期几"},
	{Intent: IntentChat, Query: "推荐一首歌"},
	{Intent: IntentChat, Query: "讲个故事给我听"},
	{Intent: IntentChat, Query: "你会些什么"},
	{Intent: IntentChat, Query: "晚安"},

	// ---- knowledge: 文档/知识库检索 (14 条) ----
	{Intent: IntentKnowledge, Query: "根据文档回答这个问题"},
	{Intent: IntentKnowledge, Query: "知识库里怎么说的"},
	{Intent: IntentKnowledge, Query: "上传的文件中有没有关于XX的内容"},
	{Intent: IntentKnowledge, Query: "帮我查一下资料里的相关条款"},
	{Intent: IntentKnowledge, Query: "这个在白皮书里是怎么描述的"},
	{Intent: IntentKnowledge, Query: "参考上传的文档，解释XX"},
	{Intent: IntentKnowledge, Query: "What does the uploaded document say about this"},
	{Intent: IntentKnowledge, Query: "上次发的那个文档里怎么说的"},
	{Intent: IntentKnowledge, Query: "之前上传的文件还能查到吗"},
	{Intent: IntentKnowledge, Query: "看看资料里有没有"},
	{Intent: IntentKnowledge, Query: "SOP 里规定的流程是什么"},
	{Intent: IntentKnowledge, Query: "编码规范里怎么要求命名的"},
	{Intent: IntentKnowledge, Query: "查一下说明书"},
	{Intent: IntentKnowledge, Query: "手册里有没有提到"},

	// ---- tool: 工具调用/外部系统交互 (14 条) ----
	{Intent: IntentTool, Query: "帮我查一下今天的天气"},
	{Intent: IntentTool, Query: "调用天气接口查北京"},
	{Intent: IntentTool, Query: "执行系统命令查看状态"},
	{Intent: IntentTool, Query: "调用工单系统创建一个新工单"},
	{Intent: IntentTool, Query: "拉取最新的数据"},
	{Intent: IntentTool, Query: "Use the weather tool to check Shanghai"},
	{Intent: IntentTool, Query: "帮我查一下系统里的库存"},
	{Intent: IntentTool, Query: "今天多少度"},
	{Intent: IntentTool, Query: "查一下服务器负载"},
	{Intent: IntentTool, Query: "备份数据库"},
	{Intent: IntentTool, Query: "发一条短信验证码"},
	{Intent: IntentTool, Query: "查询订单物流状态"},
	{Intent: IntentTool, Query: "重启服务"},
	{Intent: IntentTool, Query: "查看日志"},

	// ---- reasoning: 复杂推理/分析/编码 (14 条) ----
	{Intent: IntentReasoning, Query: "分析这段代码的性能瓶颈"},
	{Intent: IntentReasoning, Query: "比较A方案和B方案的优缺点"},
	{Intent: IntentReasoning, Query: "推理一下这个问题的根本原因"},
	{Intent: IntentReasoning, Query: "帮我重构这个函数"},
	{Intent: IntentReasoning, Query: "设计一个高可用的系统架构"},
	{Intent: IntentReasoning, Query: "解释这个算法的时间复杂度"},
	{Intent: IntentReasoning, Query: "Analyze the trade-offs between the two approaches"},
	{Intent: IntentReasoning, Query: "证明这个数学定理"},
	{Intent: IntentReasoning, Query: "这段 SQL 为什么这么慢"},
	{Intent: IntentReasoning, Query: "Microservices vs monolith which is better"},
	{Intent: IntentReasoning, Query: "怎么优化接口响应时间"},
	{Intent: IntentReasoning, Query: "这两个方法哪个更优"},
	{Intent: IntentReasoning, Query: "帮我看看这代码有什么问题"},
	{Intent: IntentReasoning, Query: "有没有更好的实现方式"},
}

// EmbeddingIntentMatcher 基于 Embedding 向量相似度的快速意图匹配器。
type EmbeddingIntentMatcher struct {
	embedder embedding.Embedder

	// anchors 预计算的锚点向量，按意图分组。
	// key=IntentLabel, value=该意图下所有锚点 query 的向量组。
	anchors map[IntentLabel][][]float64

	// threshold 相似度阈值：超过此值才采纳匹配结果。
	// 设为 0.85 是经过权衡的——太低会误判，太高会漏判（全部透传到 L2）。
	threshold float64

	// margin 意图间最小裕度：最高意图与次高意图的相似度差值必须超过此值。
	// 防止"两个意图都很高但很接近"时的歧义路由。
	margin float64

	// timeout Embedding API 调用超时。
	timeout time.Duration

	mu         sync.RWMutex
	initialized bool
}

// NewEmbeddingIntentMatcher 创建 Embedding 意图匹配器。
// 若 Embedding API 未配置（RAG_EMBEDDING_API_KEY 为空）或配置中禁用 L1，返回 nil。
func NewEmbeddingIntentMatcher(ctx context.Context) *EmbeddingIntentMatcher {
	cfg := appconfig.GetConfig().Router

	if !cfg.EmbeddingEnabledOrDefault() {
		log.Printf("[router] L1 Embedding matcher disabled by config")
		return nil
	}

	m := &EmbeddingIntentMatcher{
		threshold: cfg.EmbeddingThresholdOrDefault(),
		margin:    cfg.EmbeddingMarginOrDefault(),
		timeout:   time.Duration(cfg.EmbeddingTimeoutOrDefaultMs()) * time.Millisecond,
	}

	if err := m.init(ctx); err != nil {
		log.Printf("[router] Embedding matcher init failed (L1 disabled): %v", err)
		return nil
	}

	log.Printf("[router] Embedding matcher (L1) initialized with %d intents, %d anchors, threshold=%.2f margin=%.2f timeout=%v",
		len(m.anchors), len(anchorQueries), m.threshold, m.margin, m.timeout)
	return m
}

// init 创建 Embedder 并预计算所有锚点向量。
func (m *EmbeddingIntentMatcher) init(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initialized {
		return nil
	}

	// 1. 创建 Embedder（复用 RAG 的 Ark Embedding 服务）
	embedder, err := newRouterEmbedder(ctx)
	if err != nil {
		return fmt.Errorf("create embedder: %w", err)
	}
	m.embedder = embedder

	// 2. 收集所有锚点 query 并批量向量化
	queries := make([]string, len(anchorQueries))
	for i, a := range anchorQueries {
		queries[i] = a.Query
	}

	vectors, err := m.embedder.EmbedStrings(ctx, queries)
	if err != nil {
		return fmt.Errorf("embed anchors: %w", err)
	}

	// 3. 按意图分组预计算结果
	m.anchors = make(map[IntentLabel][][]float64)
	for i, a := range anchorQueries {
		m.anchors[a.Intent] = append(m.anchors[a.Intent], vectors[i])
	}

	m.initialized = true
	return nil
}

// Match 对用户 query 做意图匹配。
// 返回 nil 表示匹配不成功的（相似度不够高或意图间歧义），应由上层（L2/L3）继续处理。
func (m *EmbeddingIntentMatcher) Match(ctx context.Context, query string) *EmbeddingMatchResult {
	if m == nil {
		return nil // L1 未启用
	}

	m.mu.RLock()
	if !m.initialized || m.embedder == nil {
		m.mu.RUnlock()
		return nil
	}
	m.mu.RUnlock()

	// 1. 向量化 query（带超时）
	matchCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	vectors, err := m.embedder.EmbedStrings(matchCtx, []string{query})
	if err != nil {
		log.Printf("[router] L1 embed failed: %v", err)
		return nil
	}
	if len(vectors) == 0 {
		return nil
	}
	queryVec := vectors[0]

	// 2. 计算与每个意图锚点的最大相似度
	var scores []intentScore

	m.mu.RLock()
	for intent, anchorVecs := range m.anchors {
		best := 0.0
		for _, anchorVec := range anchorVecs {
			sim := cosineSimilarity(queryVec, anchorVec)
			if sim > best {
				best = sim
			}
		}
		scores = append(scores, intentScore{Intent: intent, Score: best})
	}
	m.mu.RUnlock()

	// 3. 排序取最高分
	top := findTopTwo(scores)

	// 4. 阈值 + 裕度双重检查
	if top.best.Score < m.threshold {
		log.Printf("[router] L1 no match: best=%s score=%.3f < threshold=%.2f",
			top.best.Intent, top.best.Score, m.threshold)
		return nil
	}

	margin := top.best.Score - top.second.Score
	if margin < m.margin {
		log.Printf("[router] L1 ambiguous: best=%s(%.3f) second=%s(%.3f) margin=%.3f < %.2f",
			top.best.Intent, top.best.Score, top.second.Intent, top.second.Score, margin, m.margin)
		return nil
	}

	return &EmbeddingMatchResult{
		Intent: top.best.Intent,
		Score:  top.best.Score,
		Margin: margin,
	}
}

// EmbeddingMatchResult 意图匹配结果。
type EmbeddingMatchResult struct {
	Intent IntentLabel // 匹配到的意图
	Score  float64     // 相似度得分 [0,1]
	Margin float64     // 与次高意图的裕度
}

// intentScore 内部使用的意图-得分对。
type intentScore struct {
	Intent IntentLabel
	Score  float64
}

// pairedScores 最高分和次高分对。
type pairedScores struct {
	best, second intentScore
}

func findTopTwo(scores []intentScore) pairedScores {
	p := pairedScores{
		best:   intentScore{Score: -1},
		second: intentScore{Score: -1},
	}
	for _, s := range scores {
		if s.Score > p.best.Score {
			p.second = p.best
			p.best = s
		} else if s.Score > p.second.Score {
			p.second = s
		}
	}
	return p
}

// ============================================================
// Embedder 创建（复用 RAG 基础设施）
// ============================================================

// newRouterEmbedder 创建一个轻量的 Ark Embedder 实例。
// 完全复用 RAG 的配置，零额外环境变量。
func newRouterEmbedder(ctx context.Context) (embedding.Embedder, error) {
	cfg := appconfig.GetConfig()

	apiKey := cfg.Model.RagEmbeddingAPIKey
	if apiKey == "" {
		return nil, fmt.Errorf("RAG_EMBEDDING_API_KEY is empty, L1 embedding layer disabled")
	}

	modelName := cfg.RagModelConfig.RagEmbeddingModel
	if modelName == "" {
		return nil, fmt.Errorf("rag embedding model not configured")
	}

	baseURL := cfg.RagModelConfig.RagEmbeddingBaseURL
	if baseURL == "" {
		return nil, fmt.Errorf("rag embedding base URL not configured")
	}

	apiType := embeddingArk.APITypeText
	if cfg.RagModelConfig.RagEmbeddingAPIType == "multimodal" {
		apiType = embeddingArk.APITypeMultiModal
	}

	return embeddingArk.NewEmbedder(ctx, &embeddingArk.EmbeddingConfig{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   modelName,
		APIType: &apiType,
	})
}

// ============================================================
// 工具函数
// ============================================================

// cosineSimilarity 计算两个向量的余弦相似度。
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
