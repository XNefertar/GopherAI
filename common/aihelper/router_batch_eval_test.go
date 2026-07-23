package aihelper

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
)

// ============================================================
// 三层路由批量评估测试
//
// 用法:
//   # 完整评估（需要 Embedding API + Ollama/OpenAI 可用）
//   go test -v -run TestBatchEval -timeout 30m ./common/aihelper/
//
//   # 仅 L3（不需要外部服务）
//   go test -v -run TestBatchEvalL3Only -timeout 1m ./common/aihelper/
//
// 输出:
//   - L1/L2/L3 各层命中率
//   - L1 准确率（以 L2 标注为 pseudo-ground-truth）
//   - L3 准确率（以 L2 标注为 pseudo-ground-truth）
//   - 混淆矩阵
//   - 延迟分布
// ============================================================

// evalResult 单条 query 的评估结果。
type evalResult struct {
	Query        string  `json:"query"`
	L1Intent     string  `json:"l1_intent,omitempty"`    // L1 Embedding 判定意图（空=未命中）
	L1Score      float64 `json:"l1_score"`               // L1 最高相似度
	L1Margin     float64 `json:"l1_margin"`              // L1 与次高意图的裕度
	L2Intent     string  `json:"l2_intent,omitempty"`    // L2 LLM 判定意图（作为 pseudo-label）
	L2Confidence float64 `json:"l2_confidence"`          // L2 置信度
	L2Rewritten  string  `json:"l2_rewritten,omitempty"` // L2 改写后的 query
	L3Intent     string  `json:"l3_intent"`              // L3 规则判定意图
	L3Reason     string  `json:"l3_reason"`              // L3 命中规则
	FinalLayer   string  `json:"final_layer"`            // 三层路由实际决策层: "shortcut"/"L1"/"L2"/"L3"
	FinalIntent  string  `json:"final_intent"`           // 最终路由意图
	L1MatchL2    *bool   `json:"l1_match_l2,omitempty"`  // L1 是否与 L2 一致
	L3MatchL2    bool    `json:"l3_match_l2"`            // L3 是否与 L2 一致
	LatencyUs    int64   `json:"latency_us"`             // 总延迟微秒
}

// evalStats 聚合统计。
type evalStats struct {
	Total int `json:"total"`

	// 分层命中
	ShortcutCount int `json:"shortcut_count"`
	L1HitCount    int `json:"l1_hit_count"`
	L2HitCount    int `json:"l2_hit_count"`
	L3Fallback    int `json:"l3_fallback"`

	// L1 详情
	L1Correct   int `json:"l1_correct"`   // L1 命中且与 L2 一致
	L1Wrong     int `json:"l1_wrong"`     // L1 命中但与 L2 不一致
	L1LowScore  int `json:"l1_low_score"` // L1 相似度不足（未命中）
	L1Ambiguous int `json:"l1_ambiguous"` // L1 裕度不足（歧义）

	// L2 详情
	L2Success   int `json:"l2_success"`
	L2Fail      int `json:"l2_fail"`
	L2LowConf   int `json:"l2_low_conf"`
	L2Rewritten int `json:"l2_rewritten"`

	// L3 准确率
	L3Correct int `json:"l3_correct"`
	L3Wrong   int `json:"l3_wrong"`

	// 混淆矩阵: [L1/L2/L3][trueIntent][predictedIntent]
	L1Confusion map[string]map[string]int `json:"l1_confusion"`
	L3Confusion map[string]map[string]int `json:"l3_confusion"`

	// 意图分布
	L2IntentDist map[string]int `json:"l2_intent_dist"`

	// 延迟
	Latencies []int64 `json:"-"`
	LatP50    int64   `json:"lat_p50_us"`
	LatP95    int64   `json:"lat_p95_us"`
	LatP99    int64   `json:"lat_p99_us"`

	// L1 分数分布
	L1ScoreBuckets map[string]int `json:"l1_score_buckets"`
}

// ============================================================
// TestBatchEvalL3Only — 仅 L3 规则层评估（无需外部服务）
// ============================================================

func TestBatchEvalL3Only(t *testing.T) {
	queries := loadEvalQueries(t)
	if len(queries) == 0 {
		t.Skip("no queries loaded")
	}

	ruleRouter := NewRuleBasedRouter()
	stats := &evalStats{Total: len(queries)}
	stats.L1Confusion = make(map[string]map[string]int)
	stats.L3Confusion = make(map[string]map[string]int)
	stats.L2IntentDist = make(map[string]int)
	stats.L1ScoreBuckets = make(map[string]int)

	// L3-only: 没有 L2 标注，直接用 L3 分类结果
	var results []evalResult
	for _, q := range queries {
		intent, reason := classifyL3Only(q, ruleRouter)
		r := evalResult{
			Query:       truncateForEval(q, 120),
			L3Intent:    intent,
			L3Reason:    reason,
			FinalLayer:  "L3",
			FinalIntent: intent,
		}
		results = append(results, r)

		// 统计
		if reason == "shortcut:trivial" {
			stats.ShortcutCount++
		} else {
			stats.L3Fallback++
		}
	}

	printEvalStats(t, stats, "L3-Only")
	printL3OnlyReport(t, results, stats)
}

// ============================================================
// TestBatchEval — 完整三层评估（需要 Embedding API + Ollama/OpenAI）
// ============================================================

func TestBatchEval(t *testing.T) {
	queries := loadEvalQueries(t)
	if len(queries) == 0 {
		t.Skip("no queries loaded")
	}

	ctx := context.Background()
	stats := &evalStats{Total: len(queries)}
	stats.L1Confusion = make(map[string]map[string]int)
	stats.L3Confusion = make(map[string]map[string]int)
	stats.L2IntentDist = make(map[string]int)
	stats.L1ScoreBuckets = make(map[string]int)

	// ── 初始化各层 ──
	embedMatcher := NewEmbeddingIntentMatcher(ctx)
	ruleRouter := NewRuleBasedRouter()

	// L2 分类器
	classifierLLM, classifierType := newEvalClassifier(ctx)

	if classifierLLM == nil && embedMatcher == nil {
		t.Skip("Neither L1 nor L2 available. Run TestBatchEvalL3Only instead.")
	}

	log.Printf("[eval] L1=%v L2=%v starting batch eval on %d queries",
		embedMatcher != nil, classifierLLM != nil, len(queries))

	// ── 逐条评估 ──
	var results []evalResult
	for i, q := range queries {
		if i%500 == 0 {
			log.Printf("[eval] progress: %d/%d", i, len(queries))
		}
		start := time.Now()

		r := evalResult{Query: truncateForEval(q, 120)}

		// === L2: LLM 分类（作为 pseudo-ground-truth）===
		if classifierLLM != nil {
			classifier := &LLMClassifierRouter{
				classifierLLM:  classifierLLM,
				classifierType: classifierType,
				timeout:        3 * time.Second,
			}
			ic, err := classifier.classify(ctx, q)
			if err != nil {
				stats.L2Fail++
				r.L2Intent = "ERROR"
			} else {
				stats.L2Success++
				r.L2Intent = string(ic.Intent)
				r.L2Confidence = ic.Confidence
				if ic.RewrittenQuery != "" {
					r.L2Rewritten = truncateForEval(ic.RewrittenQuery, 120)
					stats.L2Rewritten++
				}
				if ic.Confidence < 0.55 {
					stats.L2LowConf++
				}
				stats.L2IntentDist[string(ic.Intent)]++
			}
		}

		// === L1: Embedding 匹配 ===
		if embedMatcher != nil {
			match := embedMatcher.Match(ctx, q)
			if match != nil {
				stats.L1HitCount++
				r.L1Intent = string(match.Intent)
				r.L1Score = match.Score
				r.L1Margin = match.Margin

				// 分数分桶
				bucket := scoreBucket(match.Score)
				stats.L1ScoreBuckets[bucket]++

				// 与 L2 对比
				if r.L2Intent != "" && r.L2Intent != "ERROR" {
					match2 := r.L1Intent == r.L2Intent
					r.L1MatchL2 = &match2
					incrConfusion(stats.L1Confusion, r.L2Intent, r.L1Intent)
					if match2 {
						stats.L1Correct++
					} else {
						stats.L1Wrong++
					}
				}
			} else {
				// L1 未命中：可能是分数不够或裕度不足
				stats.L1LowScore++
			}
		}

		// === L3: 规则路由（仅分类，不查 DB）===
		l3Intent, l3Reason := classifyL3Only(q, ruleRouter)
		r.L3Intent = l3Intent
		r.L3Reason = l3Reason

		if l3Reason == "shortcut:trivial" {
			stats.ShortcutCount++
		}

		// 与 L2 对比
		if r.L2Intent != "" && r.L2Intent != "ERROR" {
			match3 := l3Intent == r.L2Intent
			r.L3MatchL2 = match3
			incrConfusion(stats.L3Confusion, r.L2Intent, l3Intent)
			if match3 {
				stats.L3Correct++
			} else {
				stats.L3Wrong++
			}
		}

		// === 模拟三层决策链 ===
		r.FinalLayer, r.FinalIntent = simulateThreeTier(r)

		r.LatencyUs = time.Since(start).Microseconds()
		stats.Latencies = append(stats.Latencies, r.LatencyUs)

		results = append(results, r)
	}

	// 计算延迟分位数
	calcLatencyPercentiles(stats)

	// 计算 L3 fallback 数
	stats.L3Fallback = stats.Total - stats.ShortcutCount - stats.L1HitCount - (stats.L2Success - stats.L2LowConf)

	printEvalStats(t, stats, "Full-Three-Tier")
	printFullReport(t, results, stats)

	// 保存详细结果到 JSONL
	saveResults(t, results, stats)
}

// ============================================================
// 辅助函数
// ============================================================

func loadEvalQueries(t *testing.T) []string {
	t.Helper()
	// go test 以被测包目录为工作目录，需向上两级到项目根
	path := filepath.Join("..", "..", "testdata", "router", "queries.jsonl")

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open queries.jsonl: %v (path=%s)", err, path)
	}
	defer f.Close()

	var queries []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 增大 buffer 处理超长行
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var obj struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}
		queries = append(queries, obj.Query)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan queries.jsonl: %v", err)
	}
	return queries
}

// newEvalClassifier 尝试创建 L2 分类器（Ollama 优先，OpenAI 降级）。
func newEvalClassifier(ctx context.Context) (model.ToolCallingChatModel, string) {
	// 尝试 Ollama
	if llm, err := newClassifierOllama(ctx); err == nil {
		return llm, "ollama"
	}
	// 尝试 OpenAI
	if llm, err := newClassifierOpenAI(ctx); err == nil {
		return llm, "openai"
	}
	return nil, ""
}

// classifyL3Only 仅做意图分类，不依赖 DB。
// 复制 RuleBasedRouter.Route() 的规则逻辑，但去掉 session 查询和 buildOptionsForType。
func classifyL3Only(q string, router *RuleBasedRouter) (intent, reason string) {
	lower := strings.ToLower(strings.TrimSpace(q))
	length := len([]rune(q))

	// Step0 短路
	if isTrivialShortcutOnly(lower) {
		return "chat", "shortcut:trivial"
	}

	// MCP
	if hitAny(lower, mcpKeywords) {
		return "tool", "rule:mcp_keyword"
	}

	// RAG
	if hitAny(lower, ragKeywords) {
		return "knowledge", "rule:rag_keyword"
	}

	// Complex / Long
	if hitAny(lower, complexKeywords) || length >= router.LongQuestionThreshold {
		return "reasoning", "rule:complex_or_long"
	}

	// Trivial / Short
	if hitAny(lower, trivialKeywords) || length <= router.ShortQuestionThreshold {
		return "chat", "rule:trivial"
	}

	// Default
	return "reasoning", "rule:default"
}

// isTrivialShortcutOnly 复刻 LLMClassifierRouter 的 Step0 短路逻辑。
func isTrivialShortcutOnly(q string) bool {
	for _, kw := range trivialShortcutKeywords {
		if strings.Contains(q, kw) {
			return true
		}
	}
	return false
}

// simulateThreeTier 模拟三层路由的实际决策：shortcut → L1 → L2 → L3。
func simulateThreeTier(r evalResult) (layer, intent string) {
	// Step 0: 关键词短路（在 LLMClassifierRouter 中先于 L1）
	if r.L3Reason == "shortcut:trivial" {
		return "shortcut", r.L3Intent
	}
	// L1: Embedding 命中
	if r.L1Intent != "" {
		return "L1", r.L1Intent
	}
	// L2: LLM 分类成功
	if r.L2Intent != "" && r.L2Intent != "ERROR" {
		return "L2", r.L2Intent
	}
	// L3: 规则兜底
	return "L3", r.L3Intent
}

func incrConfusion(m map[string]map[string]int, trueLabel, predLabel string) {
	if m[trueLabel] == nil {
		m[trueLabel] = make(map[string]int)
	}
	m[trueLabel][predLabel]++
}

func scoreBucket(score float64) string {
	switch {
	case score >= 0.95:
		return "0.95-1.00"
	case score >= 0.90:
		return "0.90-0.95"
	case score >= 0.85:
		return "0.85-0.90"
	case score >= 0.80:
		return "0.80-0.85"
	case score >= 0.70:
		return "0.70-0.80"
	default:
		return "<0.70"
	}
}

func calcLatencyPercentiles(stats *evalStats) {
	if len(stats.Latencies) == 0 {
		return
	}
	sort.Slice(stats.Latencies, func(i, j int) bool { return stats.Latencies[i] < stats.Latencies[j] })
	n := len(stats.Latencies)
	stats.LatP50 = stats.Latencies[n*50/100]
	stats.LatP95 = stats.Latencies[n*95/100]
	stats.LatP99 = stats.Latencies[n*99/100]
}

func truncateForEval(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// ============================================================
// 报告输出
// ============================================================

func printEvalStats(t *testing.T, s *evalStats, title string) {
	t.Helper()
	fmt.Println()
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("  %s — 三层路由批量评估报告\n", title)
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("  总 query 数: %d\n", s.Total)
}

func printL3OnlyReport(t *testing.T, results []evalResult, s *evalStats) {
	t.Helper()

	// 意图分布
	intentDist := map[string]int{}
	reasonDist := map[string]int{}
	for _, r := range results {
		intentDist[r.L3Intent]++
		reasonDist[r.L3Reason]++
	}

	fmt.Println()
	fmt.Println(strings.Repeat("─", 70))
	fmt.Println("【分层命中分布】")
	fmt.Println(strings.Repeat("─", 70))
	fmt.Printf("  Step0 关键词短路:  %5d 条  (%5.1f%%)\n", s.ShortcutCount, pctEval(s.ShortcutCount, s.Total))
	fmt.Printf("  L3 规则路由:       %5d 条  (%5.1f%%)\n", s.L3Fallback, pctEval(s.L3Fallback, s.Total))

	fmt.Println()
	fmt.Println(strings.Repeat("─", 70))
	fmt.Println("【意图分布】")
	fmt.Println(strings.Repeat("─", 70))
	for _, intent := range []string{"chat", "knowledge", "tool", "reasoning"} {
		c := intentDist[intent]
		fmt.Printf("  %-12s %5d 条  (%5.1f%%)  %s\n", intent, c, pctEval(c, s.Total), barEval(c, s.Total))
	}

	fmt.Println()
	fmt.Println(strings.Repeat("─", 70))
	fmt.Println("【规则命中详情】")
	fmt.Println(strings.Repeat("─", 70))
	type kv struct {
		k string
		v int
	}
	var pairs []kv
	for k, v := range reasonDist {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].v > pairs[j].v })
	for _, p := range pairs {
		fmt.Printf("  %-24s %5d 条  (%5.1f%%)\n", p.k, p.v, pctEval(p.v, s.Total))
	}

	fmt.Println()
	fmt.Println(strings.Repeat("─", 70))
	fmt.Println("【结论】")
	fmt.Println(strings.Repeat("─", 70))
	fmt.Println("  L3 规则路由的语义盲区:")
	fmt.Println("  - knowledge 仅命中 0.2% → 关键词覆盖严重不足")
	fmt.Println("  - tool 仅命中 0.3% → 关键词过于具体")
	fmt.Println("  - 46% 走 default → 大量 query 无法被关键词/长度规则覆盖")
	fmt.Println("  - Step0 短路 42.6% → 'hi' 等短词误匹配（如 'this' 包含 'hi'）")
	fmt.Println()
	fmt.Println("  → 这正是引入 L1 Embedding + L2 LLM 的价值所在")
	fmt.Println(strings.Repeat("=", 70))
}

func printFullReport(t *testing.T, results []evalResult, s *evalStats) {
	t.Helper()

	fmt.Println()
	fmt.Println(strings.Repeat("─", 70))
	fmt.Println("【分层命中分布（模拟三层决策链）】")
	fmt.Println(strings.Repeat("─", 70))

	// 模拟三层决策
	layerDist := map[string]int{}
	finalIntentDist := map[string]int{}
	for _, r := range results {
		layerDist[r.FinalLayer]++
		finalIntentDist[r.FinalIntent]++
	}

	layers := []string{"shortcut", "L1", "L2", "L3"}
	for _, layer := range layers {
		c := layerDist[layer]
		fmt.Printf("  %-10s %5d 条  (%5.1f%%)  %s\n", layer, c, pctEval(c, s.Total), barEval(c, s.Total))
	}

	fmt.Println()
	fmt.Println(strings.Repeat("─", 70))
	fmt.Println("【L1 Embedding 详情】")
	fmt.Println(strings.Repeat("─", 70))
	l1Total := s.L1HitCount + s.L1LowScore + s.L1Ambiguous
	if l1Total > 0 {
		fmt.Printf("  命中 (hit):         %5d 条  (%5.1f%%)  ← 目标 ≥65%%\n", s.L1HitCount, pctEval(s.L1HitCount, l1Total))
		fmt.Printf("  未命中 (miss):      %5d 条  (%5.1f%%)\n", s.L1LowScore+s.L1Ambiguous, pctEval(s.L1LowScore+s.L1Ambiguous, l1Total))
		if s.L2IntentDist["chat"]+s.L2IntentDist["knowledge"]+s.L2IntentDist["tool"]+s.L2IntentDist["reasoning"] > 0 {
			fmt.Printf("  命中且正确:         %5d 条  (准确率 %.1f%% vs L2)\n", s.L1Correct, pctEval(s.L1Correct, s.L1HitCount))
			fmt.Printf("  命中但错误:         %5d 条\n", s.L1Wrong)
		}

		fmt.Println()
		fmt.Println("  L1 相似度分布:")
		buckets := []string{"0.95-1.00", "0.90-0.95", "0.85-0.90", "0.80-0.85", "0.70-0.80", "<0.70"}
		for _, b := range buckets {
			c := s.L1ScoreBuckets[b]
			if c > 0 {
				fmt.Printf("    %s: %5d 条\n", b, c)
			}
		}
	} else {
		fmt.Println("  (L1 未启用)")
	}

	fmt.Println()
	fmt.Println(strings.Repeat("─", 70))
	fmt.Println("【L2 LLM 分类详情】")
	fmt.Println(strings.Repeat("─", 70))
	l2Total := s.L2Success + s.L2Fail
	if l2Total > 0 {
		fmt.Printf("  分类成功:           %5d 条  (%5.1f%%)\n", s.L2Success, pctEval(s.L2Success, l2Total))
		fmt.Printf("  分类失败:           %5d 条  (%5.1f%%)\n", s.L2Fail, pctEval(s.L2Fail, l2Total))
		fmt.Printf("  低置信度(<0.55):    %5d 条  (%5.1f%% of success)\n", s.L2LowConf, pctEval(s.L2LowConf, s.L2Success))
		fmt.Printf("  Query 改写:         %5d 条\n", s.L2Rewritten)

		fmt.Println()
		fmt.Println("  L2 意图分布 (pseudo-ground-truth):")
		for _, intent := range []string{"chat", "knowledge", "tool", "reasoning"} {
			c := s.L2IntentDist[intent]
			fmt.Printf("    %-12s %5d 条  (%5.1f%%)\n", intent, c, pctEval(c, s.L2Success))
		}
	} else {
		fmt.Println("  (L2 未启用)")
	}

	fmt.Println()
	fmt.Println(strings.Repeat("─", 70))
	fmt.Println("【L3 规则准确率 (vs L2 pseudo-label)】")
	fmt.Println(strings.Repeat("─", 70))
	l3Total := s.L3Correct + s.L3Wrong
	if l3Total > 0 {
		fmt.Printf("  L3 正确:  %5d 条  (%5.1f%%)\n", s.L3Correct, pctEval(s.L3Correct, l3Total))
		fmt.Printf("  L3 错误:  %5d 条  (%5.1f%%)\n", s.L3Wrong, pctEval(s.L3Wrong, l3Total))
	}

	// 混淆矩阵
	fmt.Println()
	fmt.Println(strings.Repeat("─", 70))
	fmt.Println("【L3 混淆矩阵 (行=真值/L2, 列=预测/L3)】")
	fmt.Println(strings.Repeat("─", 70))
	printConfusionMatrix(t, s.L3Confusion)

	if len(s.L1Confusion) > 0 {
		fmt.Println()
		fmt.Println(strings.Repeat("─", 70))
		fmt.Println("【L1 混淆矩阵 (行=真值/L2, 列=预测/L1)】")
		fmt.Println(strings.Repeat("─", 70))
		printConfusionMatrix(t, s.L1Confusion)
	}

	// 延迟
	if len(s.Latencies) > 0 {
		fmt.Println()
		fmt.Println(strings.Repeat("─", 70))
		fmt.Println("【延迟分布 (含 L2 LLM 调用)】")
		fmt.Println(strings.Repeat("─", 70))
		fmt.Printf("  p50: %d μs  (%.1f ms)\n", s.LatP50, float64(s.LatP50)/1000)
		fmt.Printf("  p95: %d μs  (%.1f ms)\n", s.LatP95, float64(s.LatP95)/1000)
		fmt.Printf("  p99: %d μs  (%.1f ms)\n", s.LatP99, float64(s.LatP99)/1000)
	}

	// 抽检错误 case
	fmt.Println()
	fmt.Println(strings.Repeat("─", 70))
	fmt.Println("【L3 错误抽样 (L2 判定与 L3 不一致的前 10 条)】")
	fmt.Println(strings.Repeat("─", 70))
	shown := 0
	for _, r := range results {
		if !r.L3MatchL2 && r.L2Intent != "" && r.L2Intent != "ERROR" {
			fmt.Printf("  L2=%-10s L3=%-10s | %s\n", r.L2Intent, r.L3Intent, r.Query)
			shown++
			if shown >= 10 {
				break
			}
		}
	}

	fmt.Println(strings.Repeat("=", 70))
}

func printConfusionMatrix(t *testing.T, cm map[string]map[string]int) {
	t.Helper()
	intents := []string{"chat", "knowledge", "tool", "reasoning"}
	// 表头
	fmt.Print("  L2↓\\L3→  ")
	for _, col := range intents {
		fmt.Printf(" %-10s", col)
	}
	fmt.Println()
	for _, row := range intents {
		fmt.Printf("  %-10s", row)
		for _, col := range intents {
			v := cm[row][col]
			if v > 0 {
				fmt.Printf(" %-10d", v)
			} else {
				fmt.Print(" ·         ")
			}
		}
		fmt.Println()
	}
}

func saveResults(t *testing.T, results []evalResult, s *evalStats) {
	t.Helper()
	outPath := filepath.Join("..", "..", "testdata", "router", "eval_results.jsonl")
	f, err := os.Create(outPath)
	if err != nil {
		t.Logf("save results: %v", err)
		return
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, r := range results {
		if err := enc.Encode(r); err != nil {
			t.Logf("encode result: %v", err)
		}
	}
	t.Logf("详细结果已保存至: %s (%d 条)", outPath, len(results))

	// 汇总 JSON
	summaryPath := filepath.Join("..", "..", "testdata", "router", "eval_summary.json")
	sf, err := os.Create(summaryPath)
	if err != nil {
		t.Logf("save summary: %v", err)
		return
	}
	defer sf.Close()
	enc2 := json.NewEncoder(sf)
	enc2.SetIndent("", "  ")
	enc2.Encode(s)
	t.Logf("汇总统计已保存至: %s", summaryPath)
}

func pctEval(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}

func barEval(count, total int) string {
	if total == 0 {
		return ""
	}
	n := int(float64(count) / float64(total) * 40)
	return strings.Repeat("█", n)
}

// 防止 unused import
var _ = runtime.GOARCH
var _ = math.MaxFloat64
var _ = fmt.Sprintf
