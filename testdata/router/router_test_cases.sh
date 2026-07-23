#!/bin/bash
# 路由器功能测试用例
# 用法: bash testdata/router/router_test_cases.sh
# 前置: 项目已启动 (go run main.go), Ollama 已启动 (ollama serve &)

TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6MywidXNlcm5hbWUiOiIxMzgyOTI3MTA0OCIsImlzcyI6Imh1YW5oZWFydCIsInN1YiI6IkdvcGhlckFJIiwiZXhwIjoxODE2MDgyMjU2LCJpYXQiOjE3ODQ1NDYyNTZ9.3wdoSvDiV-sRjyTYWOYyZHEPaqHwSx_MKw4TjOeQ_v8"
BASE="http://localhost:9090/api/v1/AI/chat/send-new-session"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASS=0
FAIL=0

pass() { echo -e "${GREEN}[PASS]${NC} $1"; PASS=$((PASS+1)); }
fail() { echo -e "${RED}[FAIL]${NC} $1"; FAIL=$((FAIL+1)); }

send() {
  curl -s -X POST "$BASE" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"modelType\":\"auto\",\"question\":\"$1\"}" -o /dev/null &
}

# 获取 stats 某个字段
stat() { curl -s http://localhost:9090/debug/router/stats | python3 -c "import json,sys; print(json.load(sys.stdin)['raw']['$1'])"; }
pct()  { curl -s http://localhost:9090/debug/router/stats | python3 -c "import json,sys; print(json.load(sys.stdin)['derived']['$1'])"; }

echo "============================================"
echo " GopherAI 路由器功能测试"
echo "============================================"
echo ""

# ═══════════════════════════════════════════
# 测试 1: Step0 关键词短路
# ═══════════════════════════════════════════
echo "--- 测试 1: Step0 关键词短路 ---"
BEFORE=$(stat KeywordShortcut)

send "你好"
send "谢谢"
send "再见"
wait

AFTER=$(stat KeywordShortcut)
if [ "$AFTER" -gt "$BEFORE" ]; then
  pass "Step0 短路捕获了问候类 query ($AFTER > $BEFORE)"
else
  fail "Step0 短路未触发"
fi

# ═══════════════════════════════════════════
# 测试 2: L2 意图分类 - 推理类
# ═══════════════════════════════════════════
echo ""
echo "--- 测试 2: L2 意图分类 - 推理类 ---"
BEFORE=$(stat LLMClassified)

send "分析这段 SQL 查询的性能瓶颈"
send "比较微服务和单体架构的优缺点"
send "设计一个支持百万并发的消息队列"
wait
sleep 8

AFTER=$(stat LLMClassified)
FALLBACK=$(stat LLMFallback)
if [ "$AFTER" -gt "$BEFORE" ] && [ "$FALLBACK" -eq 0 ]; then
  pass "推理类 query 被 L2 成功分类 ($AFTER > $BEFORE)"
else
  fail "推理类 L2 分类失败 (classified=$AFTER fallback=$FALLBACK)"
fi

# ═══════════════════════════════════════════
# 测试 3: L2 意图分类 - 闲聊类
# ═══════════════════════════════════════════
echo ""
echo "--- 测试 3: L2 意图分类 - 闲聊类 ---"
BEFORE=$(stat LLMClassified)

send "今天天气怎么样"
send "你能做什么"
wait
sleep 8

AFTER=$(stat LLMClassified)
if [ "$AFTER" -gt "$BEFORE" ]; then
  pass "闲聊类 query 被 L2 成功分类"
else
  fail "闲聊类 L2 分类失败"
fi

# ═══════════════════════════════════════════
# 测试 4: L2 意图分类 - 工具类
# ═══════════════════════════════════════════
echo ""
echo "--- 测试 4: L2 意图分类 - 工具类 ---"
BEFORE=$(stat LLMClassified)

send "帮我查一下今天的天气"
send "创建一个新的工单"
wait
sleep 8

AFTER=$(stat LLMClassified)
if [ "$AFTER" -gt "$BEFORE" ]; then
  pass "工具类 query 被 L2 成功分类"
else
  fail "工具类 L2 分类失败"
fi

# ═══════════════════════════════════════════
# 测试 5: L2 分类稳定性（批量验证）
# ═══════════════════════════════════════════
echo ""
echo "--- 测试 5: L2 分类稳定性（5 条并发） ---"
BEFORE=$(stat LLMClassified)

send "解释一下这个正则表达式"
send "为什么 MySQL 用 B+ 树而不用哈希"
send "有没有更好的算法解决这个问题"
send "帮我 review 这段代码"
send "这个 bug 的根因可能是什么"
wait
sleep 10

AFTER=$(stat LLMClassified)
FALLBACK=$(stat LLMFallback)
TOTAL_CHANGE=$((AFTER - BEFORE))
if [ "$TOTAL_CHANGE" -ge 4 ] && [ "$FALLBACK" -eq 0 ]; then
  pass "批量分类稳定: 5 条中至少 4 条命中，0 降级"
else
  fail "批量分类不稳定: 命中 $TOTAL_CHANGE 条, 降级 $FALLBACK 条"
fi

# ═══════════════════════════════════════════
# 测试 6: 路径分布合理性
# ═══════════════════════════════════════════
echo ""
echo "--- 测试 6: 路径分布 ---"
SHORTCUT=$(pct step0_shortcut_pct)
CLASSIFIED=$(pct l2_llm_classified_pct)
FALLBACK_PCT=$(pct l2_llm_fallback_pct)

echo "  Step0 短路:  ${SHORTCUT}%"
echo "  L2 分类成功: ${CLASSIFIED}%"
echo "  L2 分类降级: ${FALLBACK_PCT}%"

if python3 -c "exit(0 if ${FALLBACK_PCT:-0} < 20 else 1)" 2>/dev/null; then
  pass "L2 降级率 < 20%"
else
  fail "L2 降级率过高: ${FALLBACK_PCT}%"
fi

# ═══════════════════════════════════════════
# 汇总
# ═══════════════════════════════════════════
echo ""
echo "============================================"
echo " 测试总结: ${GREEN}$PASS 通过${NC} / ${RED}$FAIL 失败${NC}"
echo "============================================"

echo ""
echo "--- 最终路由器统计 ---"
curl -s http://localhost:9090/debug/router/stats | python3 -m json.tool
