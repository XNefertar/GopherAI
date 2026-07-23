package aihelper

import (
	"GopherAI/dao/session"
	appconfig "GopherAI/config"
	"context"
	"log"
	"strings"
	"sync"
	"unicode/utf8"
)

// RouteDecision 表示一次路由决策的结果，便于上层埋点与排障。
type RouteDecision struct {
	// ModelType 最终选择的模型类型（ModelTypeOpenAI / ModelTypeRAG / ...）
	ModelType string
	// Reason 命中的规则名称，用于日志/统计/可观测性
	Reason string
	// Options 工厂创建模型所需要的具体参数
	Options CreateOptions
}

// HybridRouter 定义混合路由器接口。
// 输入：用户名 / 会话 ID / 用户问题 / 是否流式。
// 输出：一次路由决策，决策结果可直接交给 AIHelperManager 创建或切换模型。
type HybridRouter interface {
	Route(ctx context.Context, userName, sessionID, question string, stream bool) (RouteDecision, error)
}

// RuleBasedRouter 基于关键词与启发式规则的混合路由器实现。
//
// 设计目标：
//  1. 低成本：高频简单问题（问候、闲聊、短问答）走低成本模型；
//  2. 高质量：长文/复杂推理问题走主模型；
//  3. 知识增强：明显依赖文档/资料的问题走 RAG；
//  4. 工具调用：明显需要外部系统操作的问题走 MCP；
//  5. 可回退：无法命中规则时使用默认模型。
type RuleBasedRouter struct {
	// DefaultModelType 兜底模型类型，建议设置为 OpenAI 主模型
	DefaultModelType string
	// CheapModelType 低成本模型类型，命中“简单问题”时使用
	CheapModelType string
	// LongQuestionThreshold 触发“高质量模型”的字符阈值（按 rune 计数）
	LongQuestionThreshold int
	// ShortQuestionThreshold 触发“低成本模型”的字符阈值（按 rune 计数）
	ShortQuestionThreshold int
}

// NewRuleBasedRouter 构造一个带有默认参数的规则路由器。
func NewRuleBasedRouter() *RuleBasedRouter {
	return &RuleBasedRouter{
		DefaultModelType:       ModelTypeOpenAI,
		CheapModelType:         ModelTypeOpenAI,
		LongQuestionThreshold:  300,
		ShortQuestionThreshold: 12,
	}
}

// ragKeywords 触发 RAG 路由的中英文关键词集合。
var ragKeywords = []string{
	"根据文档", "知识库", "资料", "白皮书", "手册", "规范", "上传的文件",
	"according to", "in the document", "knowledge base", "from the docs", "reference document",
}

// mcpKeywords 触发 MCP（工具调用）路由的关键词集合。
var mcpKeywords = []string{
	"调用", "查询系统", "执行工具", "查一下", "帮我执行", "拉取", "工单", "工具",
	"call the tool", "invoke", "execute", "run command", "use tool",
}

// complexKeywords 触发高质量主模型的关键词集合。
var complexKeywords = []string{
	"分析", "推理", "证明", "对比", "代码审查", "重构", "复盘", "架构",
	"analyze", "reasoning", "compare", "design", "refactor", "review",
}

// trivialKeywords 明显的低成本闲聊/问候类关键词集合。
var trivialKeywords = []string{
	"你好", "hi", "hello", "在吗", "早上好", "晚上好", "谢谢", "thanks",
}

// Route 根据问题特征选择模型。
// 注意：本实现保持纯函数式判断，不依赖外部 IO，便于单测与压测复用。
func (r *RuleBasedRouter) Route(ctx context.Context, userName, sessionID, question string, stream bool) (RouteDecision, error) {
	q := strings.ToLower(strings.TrimSpace(question))
	length := utf8.RuneCountInString(q)
	sessionObj, err := session.GetSessionByID(ctx, sessionID)
	if err != nil {
		return RouteDecision{}, err
	}
	// 1. 工具调用类：优先级最高，命中即走 MCP
	if hitAny(q, mcpKeywords) {
		opts, err := buildOptionsForType(ModelTypeMCP, userName, sessionObj.ActiveKBID)
		if err == nil {
			return RouteDecision{ModelType: ModelTypeMCP, Reason: "rule:mcp_keyword", Options: opts}, nil
		}
		log.Printf("[router] MCP options build failed, fallback: %v", err)
	}

	// 2. 知识增强类：命中文档/知识库关键词走 RAG
	if hitAny(q, ragKeywords) {
		opts, err := buildOptionsForType(ModelTypeRAG, userName, sessionObj.ActiveKBID)
		if err == nil {
			return RouteDecision{ModelType: ModelTypeRAG, Reason: "rule:rag_keyword", Options: opts}, nil
		}
		log.Printf("[router] RAG options build failed, fallback: %v", err)
	}

	// 3. 复杂推理类：明确命中复杂关键词或问题特别长，走主模型
	if hitAny(q, complexKeywords) || length >= r.LongQuestionThreshold {
		opts, err := buildOptionsForType(r.DefaultModelType, userName, sessionObj.ActiveKBID)
		if err == nil {
			return RouteDecision{ModelType: r.DefaultModelType, Reason: "rule:complex_or_long", Options: opts}, nil
		}
	}

	// 4. 低成本闲聊类：问候/极短问题走低成本模型
	if hitAny(q, trivialKeywords) || length <= r.ShortQuestionThreshold {
		// 若配置了 Ollama 本地模型，则优先使用本地推理，进一步降低成本
		if appconfig.GetConfig().Model.OllamaModelName != "" {
			opts, err := buildOptionsForType(ModelTypeOllama, userName, sessionObj.ActiveKBID)
			if err == nil {
				return RouteDecision{ModelType: ModelTypeOllama, Reason: "rule:trivial_local", Options: opts}, nil
			}
		}
		opts, err := buildOptionsForType(r.CheapModelType, userName, sessionObj.ActiveKBID)
		if err == nil {
			return RouteDecision{ModelType: r.CheapModelType, Reason: "rule:trivial", Options: opts}, nil
		}
	}

	// 5. 兜底：默认主模型
	opts, err := buildOptionsForType(r.DefaultModelType, userName, sessionObj.ActiveKBID)
	if err != nil {
		return RouteDecision{}, err
	}
	return RouteDecision{ModelType: r.DefaultModelType, Reason: "rule:default", Options: opts}, nil
}

// hitAny 判断 q 是否包含 keywords 中的任意一个关键词（区分大小写已在 Route 中预处理）。
func hitAny(q string, keywords []string) bool {
	for _, kw := range keywords {
		if kw == "" {
			continue
		}
		if strings.Contains(q, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// buildOptionsForType 是 BuildSessionCreateOptions 的内部薄包装，避免 router 直接耦合错误处理细节。
func buildOptionsForType(modelType, userName, kbID string) (CreateOptions, error) {
	return BuildSessionCreateOptions(modelType, userName, kbID)
}

// 全局路由器单例：服务层可以直接使用，也可以通过 SetGlobalRouter 注入自定义实现做 A/B。
var (
	globalRouter     HybridRouter
	globalRouterOnce sync.Once
	globalRouterMu   sync.RWMutex
)

// GetGlobalRouter 获取全局混合路由器。
//
// 默认使用 LLMClassifierRouter（轻量 LLM 语义分类 + 规则降级），
// 在进程启动阶段通过 InitGlobalRouter(ctx) 确保分类器 LLM 已创建。
// 若进程未调用 InitGlobalRouter，sync.Once 内会使用 context.Background() 兜底创建。
func GetGlobalRouter() HybridRouter {
	globalRouterOnce.Do(func() {
		globalRouter = NewLLMClassifierRouter(context.Background())
	})
	globalRouterMu.RLock()
	defer globalRouterMu.RUnlock()
	return globalRouter
}

// InitGlobalRouter 在进程启动阶段主动初始化全局路由器。
// 应在 App.Run() 中、基础设施就绪后调用，确保分类器 LLM 尽早创建并暴露失败日志。
func InitGlobalRouter(ctx context.Context) {
	globalRouterOnce.Do(func() {
		globalRouter = NewLLMClassifierRouter(ctx)
	})
}

// SetGlobalRouter 注入自定义路由器（例如带成本统计、A/B 实验、模型质量评估的版本）。
func SetGlobalRouter(r HybridRouter) {
	if r == nil {
		return
	}
	// 触发一次默认初始化，保证 once 已经执行过
	GetGlobalRouter()
	globalRouterMu.Lock()
	globalRouter = r
	globalRouterMu.Unlock()
}
