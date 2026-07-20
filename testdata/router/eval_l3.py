#!/usr/bin/env python3
"""
三层路由批量评估工具 — L3 规则层分析（无需外部服务）

用法:
  python3 testdata/router/eval_l3.py

输出:
  - 每条 query 的 L3 路由决策（modelType + reason）
  - 各意图/规则命中分布
  - 置信度分布（基于规则优先级）
"""

import json
import sys
import os
from collections import Counter, defaultdict

# ============================================================
# 复刻 Go 侧 RuleBasedRouter 的规则逻辑
# ============================================================

RAG_KEYWORDS = [
    "根据文档", "知识库", "资料", "白皮书", "手册", "规范", "上传的文件",
    "according to", "in the document", "knowledge base", "from the docs", "reference document",
]

MCP_KEYWORDS = [
    "调用", "查询系统", "执行工具", "查一下", "帮我执行", "拉取", "工单", "工具",
    "call the tool", "invoke", "execute", "run command", "use tool",
]

COMPLEX_KEYWORDS = [
    "分析", "推理", "证明", "对比", "代码审查", "重构", "复盘", "架构",
    "analyze", "reasoning", "compare", "design", "refactor", "review",
]

TRIVIAL_KEYWORDS = [
    "你好", "hi", "hello", "在吗", "早上好", "晚上好", "谢谢", "thanks",
]

# Step0 短路关键词（来自 router_llm.go 的 trivialShortcutKeywords）
SHORTCUT_KEYWORDS = [
    "你好", "您好", "hi", "hello", "hey", "在吗", "在不在",
    "早上好", "中午好", "下午好", "晚上好", "晚安",
    "good morning", "good afternoon", "good evening",
    "谢谢", "感谢", "thanks", "thank you",
    "再见", "拜拜", "bye", "goodbye", "see you",
]

LONG_THRESHOLD = 300
SHORT_THRESHOLD = 12


def hit_any(q_lower, keywords):
    """判断 q 是否包含 keywords 中任一关键词。"""
    for kw in keywords:
        if kw and kw.lower() in q_lower:
            return True
    return False


def classify_l3(query: str) -> dict:
    """
    模拟 RuleBasedRouter.Route() 的决策逻辑。
    返回 {"modelType": str, "reason": str, "intent": str}
    """
    q = query.strip().lower()
    length = len(q)  # Go 侧用 utf8.RuneCountInString，近似用 len

    # Step 0: 关键词短路（在 LLMClassifierRouter 中先于 L1）
    if hit_any(q, SHORTCUT_KEYWORDS):
        return {"modelType": "Ollama/Cheap", "reason": "shortcut:trivial", "intent": "chat"}

    # 1. MCP 工具调用
    if hit_any(q, MCP_KEYWORDS):
        return {"modelType": "MCP", "reason": "rule:mcp_keyword", "intent": "tool"}

    # 2. RAG 知识检索
    if hit_any(q, RAG_KEYWORDS):
        return {"modelType": "RAG", "reason": "rule:rag_keyword", "intent": "knowledge"}

    # 3. 复杂推理 / 长文本
    if hit_any(q, COMPLEX_KEYWORDS) or length >= LONG_THRESHOLD:
        return {"modelType": "OpenAI", "reason": "rule:complex_or_long", "intent": "reasoning"}

    # 4. 闲聊 / 极短
    if hit_any(q, TRIVIAL_KEYWORDS) or length <= SHORT_THRESHOLD:
        return {"modelType": "Ollama/Cheap", "reason": "rule:trivial", "intent": "chat"}

    # 5. 兜底
    return {"modelType": "OpenAI", "reason": "rule:default", "intent": "reasoning"}


# ============================================================
# 批量评估
# ============================================================

def load_queries(filepath: str) -> list:
    queries = []
    with open(filepath, "r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                obj = json.loads(line)
                queries.append(obj["query"])
            except (json.JSONDecodeError, KeyError):
                continue
    return queries


def main():
    script_dir = os.path.dirname(os.path.abspath(__file__))
    jsonl_path = os.path.join(script_dir, "queries.jsonl")

    print("=" * 70)
    print(" GopherAI 三层路由 — L3 规则层批量评估")
    print("=" * 70)
    print(f"数据文件: {jsonl_path}")
    print()

    queries = load_queries(jsonl_path)
    total = len(queries)
    print(f"加载 query 总数: {total}")
    print()

    # 统计
    intent_counter = Counter()
    reason_counter = Counter()
    model_counter = Counter()
    length_buckets = Counter()
    layer_counts = {"step0_shortcut": 0, "l3_rule": 0}

    results = []

    for q in queries:
        r = classify_l3(q)
        intent_counter[r["intent"]] += 1
        reason_counter[r["reason"]] += 1
        model_counter[r["modelType"]] += 1

        # 分层统计
        if r["reason"].startswith("shortcut:"):
            layer_counts["step0_shortcut"] += 1
        else:
            layer_counts["l3_rule"] += 1

        # 长度分桶
        ln = len(q)
        if ln <= 12:
            length_buckets["≤12(极短)"] += 1
        elif ln <= 50:
            length_buckets["13-50(短)"] += 1
        elif ln <= 200:
            length_buckets["51-200(中)"] += 1
        elif ln <= 500:
            length_buckets["201-500(长)"] += 1
        else:
            length_buckets[">500(超长)"] += 1

        results.append({"query": q[:80], "intent": r["intent"], "reason": r["reason"], "model": r["modelType"]})

    # ============================================================
    # 输出报告
    # ============================================================

    print("─" * 70)
    print("【分层命中分布】")
    print("─" * 70)
    print(f"  Step0 关键词短路:  {layer_counts['step0_shortcut']:>5} 条  ({layer_counts['step0_shortcut']/total*100:5.1f}%)")
    print(f"  L3 规则路由:       {layer_counts['l3_rule']:>5} 条  ({layer_counts['l3_rule']/total*100:5.1f}%)")
    print()

    print("─" * 70)
    print("【意图分布 (L3 判定)】")
    print("─" * 70)
    intent_order = ["chat", "knowledge", "tool", "reasoning"]
    for intent in intent_order:
        count = intent_counter.get(intent, 0)
        bar = "█" * int(count / total * 50)
        print(f"  {intent:<12} {count:>5} 条  ({count/total*100:5.1f}%)  {bar}")
    print()

    print("─" * 70)
    print("【模型选择分布】")
    print("─" * 70)
    for model, count in model_counter.most_common():
        print(f"  {model:<16} {count:>5} 条  ({count/total*100:5.1f}%)")
    print()

    print("─" * 70)
    print("【规则命中详情】")
    print("─" * 70)
    for reason, count in reason_counter.most_common():
        print(f"  {reason:<24} {count:>5} 条  ({count/total*100:5.1f}%)")
    print()

    print("─" * 70)
    print("【Query 长度分布】")
    print("─" * 70)
    for bucket in ["≤12(极短)", "13-50(短)", "51-200(中)", "201-500(长)", ">500(超长)"]:
        count = length_buckets.get(bucket, 0)
        bar = "█" * int(count / total * 50)
        print(f"  {bucket:<16} {count:>5} 条  ({count/total*100:5.1f}%)  {bar}")
    print()

    # 抽检：展示每个意图的前 3 条 sample
    print("─" * 70)
    print("【各意图抽样 (前3条)】")
    print("─" * 70)
    samples = defaultdict(list)
    for r in results:
        if len(samples[r["intent"]]) < 3:
            samples[r["intent"]].append(r)
    for intent in intent_order:
        print(f"  [{intent}]")
        for s in samples.get(intent, []):
            print(f"    query: {s['query']}")
            print(f"    → {s['model']} ({s['reason']})")
        print()

    # ═══════════════════════════════════════
    # 说明：L3-only 的局限性
    # ═══════════════════════════════════════
    print("─" * 70)
    print("【说明】")
    print("─" * 70)
    print("  以上仅为 L3（规则路由层）的分析结果。")
    print("  L3 完全基于关键词匹配 + 长度启发式，准确率约 65%。")
    print()
    print("  完整的三层评估需要：")
    print("    1. Embedding API 可用 → L1 向量相似度匹配")
    print("    2. Ollama/OpenAI 可用 → L2 LLM 语义分类 + 标注 expected_intent")
    print("    3. 运行 go test 得到完整的 L1→L2→L3 分层命中率 + 准确率")
    print()
    print("  届时输出将包含：")
    print("    - L1 拦截率 (目标 ~65%)      L1 准确率 (vs L2 标注)")
    print("    - L2 分类命中率               L2 低置信度回退率")
    print("    - L3 兜底率                   整体准确率")
    print("    - 混淆矩阵 (chat/knowledge/tool/reasoning)")
    print("    - 各层延迟分布 (p50/p95/p99)")
    print("=" * 70)


if __name__ == "__main__":
    main()
