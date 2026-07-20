#!/bin/bash
# 路由器综合测试：60 条 query，覆盖 4 类意图 + 中英文 + 长短 + 口语/正式
# 用法: bash testdata/router_comprehensive_test.sh

TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6MywidXNlcm5hbWUiOiIxMzgyOTI3MTA0OCIsImlzcyI6Imh1YW5oZWFydCIsInN1YiI6IkdvcGhlckFJIiwiZXhwIjoxODE2MDgyMjU2LCJpYXQiOjE3ODQ1NDYyNTZ9.3wdoSvDiV-sRjyTYWOYyZHEPaqHwSx_MKw4TjOeQ_v8"
BASE="http://localhost:9090/api/v1/AI/chat/send-new-session"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'

PASS=0; FAIL=0

send() { curl -s -X POST "$BASE" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "{\"modelType\":\"auto\",\"question\":\"$1\"}" -o /dev/null & }

# ═══════════════════════════════════════════
# 测试函数
# ═══════════════════════════════════════════

# 发一批请求，等回答完成，看 LLMFallback 是否增加
batch_test() {
  local label="$1"; shift
  local before=$(curl -s http://localhost:9090/debug/router/stats | python3 -c "import json,sys; print(json.load(sys.stdin)['raw']['LLMFallback'])")
  local before_classified=$(curl -s http://localhost:9090/debug/router/stats | python3 -c "import json,sys; print(json.load(sys.stdin)['raw']['LLMClassified'])")

  for q in "$@"; do send "$q"; done
  wait
  sleep 10  # 等 Ollama 分类完成

  local after=$(curl -s http://localhost:9090/debug/router/stats | python3 -c "import json,sys; print(json.load(sys.stdin)['raw']['LLMFallback'])")
  local after_classified=$(curl -s http://localhost:9090/debug/router/stats | python3 -c "import json,sys; print(json.load(sys.stdin)['raw']['LLMClassified'])")
  local total=$#
  local classified=$((after_classified - before_classified))
  local fallback=$((after - before))

  if [ "$fallback" -eq 0 ]; then
    echo -e "  ${GREEN}[PASS]${NC} $label: ${classified}/${total} 条被 L2 成功分类，0 降级"
    PASS=$((PASS+1))
  else
    echo -e "  ${RED}[FAIL]${NC} $label: ${classified}/${total} 分类，${fallback} 条降级"
    FAIL=$((FAIL+1))
  fi
}

echo "╔══════════════════════════════════════════╗"
echo "║   GopherAI 路由器综合测试 (60 条)        ║"
echo "╚══════════════════════════════════════════╝"
echo ""

# ═══════════════════════════════════════════
# Category 1: 推理/分析/编码 (15 条)
#   - 架构设计、算法分析、代码 review、系统设计
#   - 中英文混合、正式/口语化
# ═══════════════════════════════════════════
echo -e "${CYAN}━━━ 1. 推理/分析/编码类 (15 条) ━━━${NC}"
batch_test "架构与系统设计" \
  "设计一个支持千万级用户的推送系统" \
  "微服务和单体架构各自的优缺点是什么" \
  "如果你要实现一个分布式锁，会怎么设计" \
  "CAP 理论在分布式系统中如何权衡"

batch_test "算法与性能分析" \
  "分析这段递归算法的时间复杂度" \
  "这个 SQL 查询为什么在大数据量下会变慢" \
  "有没有比快排更优的排序场景" \
  "如何优化这个接口从 2s 降到 200ms"

batch_test "代码与工程实践" \
  "帮我 review 这段 Python 代码有什么问题" \
  "这段 Go 代码的并发控制有什么隐患" \
  "解释一下 Redis 的渐进式 rehash 原理" \
  "TCP 三次握手的设计思想是什么"

batch_test "混合/口语推理" \
  "这个思路逻辑上有没有问题" \
  "帮我想想有没有更简单的办法" \
  "How would you design a rate limiter"

# ═══════════════════════════════════════════
# Category 2: 文档/知识库 (15 条)
#   - 文档引用、知识查询、规范查询
#   - 口语化变体
# ═══════════════════════════════════════════
echo -e "\n${CYAN}━━━ 2. 文档/知识库类 (15 条) ━━━${NC}"
batch_test "显式文档引用" \
  "根据上传的文档回答审批流程的问题" \
  "白皮书里关于数据脱敏的条款是怎么规定的" \
  "参考知识库里的内容，解释一下错误码 500" \
  "上传的那个 PDF 里面提到了哪些安全策略"

batch_test "知识库查询" \
  "查一下资料里关于接口协议的定义" \
  "技术方案文档里说的三个阶段分别是什么" \
  "这个在运维手册里有记录吗" \
  "标准操作流程 S.O.P 里怎么写的"

batch_test "口语化知识查询" \
  "上次发的那个方案文档里怎么说的来着" \
  "之前上传的文件第 8 页那个表还能找到吗" \
  "看看资料里有没有这块的说明" \
  "According to the uploaded document, what is the SLA"

batch_test "规范文档引用" \
  "根据接口文档，这个字段的校验规则是什么" \
  "编码规范里关于命名约定是怎么规定的" \
  "查一下安全规范里关于密码存储的要求"

# ═══════════════════════════════════════════
# Category 3: 工具调用 (10 条)
#   - 天气查询、系统操作、API 调用
# ═══════════════════════════════════════════
echo -e "\n${CYAN}━━━ 3. 工具调用类 (10 条) ━━━${NC}"
batch_test "系统操作" \
  "帮我查一下北京的天气" \
  "调用天气接口查上海未来三天" \
  "创建一个新的工单" \
  "执行数据库备份命令"

batch_test "数据查询" \
  "拉取最近一小时的监控数据" \
  "查询服务器当前 CPU 和内存使用率" \
  "查一下这个订单的物流状态" \
  "调用短信接口发一条验证码"

batch_test "英文工具调用" \
  "Use the weather tool to check tomorrow forecast" \
  "Check the system status via command"

# ═══════════════════════════════════════════
# Category 4: 闲聊/简单问答 (12 条)
#   - 问候、告别、感谢、简单事实查询、闲聊
# ═══════════════════════════════════════════
echo -e "\n${CYAN}━━━ 4. 闲聊/简单问答 (12 条) ━━━${NC}"
batch_test "问候/告别（Step0 短路）" \
  "下午好" "回见" "hello there" "goodbye"

batch_test "简单问询" \
  "今天是星期几" "1+1等于几" \
  "讲个冷笑话" "有什么好听的歌推荐"

batch_test "无上下文闲聊" \
  "好无聊啊聊天吧" "你擅长什么" \
  "今天心情不好" "讲个故事给我听"

# ═══════════════════════════════════════════
# Category 5: 边缘/歧义 (8 条)
#   - 极短 query、无上下文、模糊意图
#   - 考验 L1→L2 降级链路
# ═══════════════════════════════════════════
echo -e "\n${CYAN}━━━ 5. 边缘/歧义类 (8 条) ━━━${NC}"
batch_test "极短歧义" "这个怎么回事" "为什么" "怎么说"

batch_test "无上下文" "帮我看看" "然后呢" "那个是对的吗"

batch_test "混合意图" "根据文档帮我分析一下这个设计" "查一下这个知识点"

# ═══════════════════════════════════════════
# 汇总
# ═══════════════════════════════════════════
echo ""
echo "╔══════════════════════════════════════════╗"
TOTAL=$(curl -s http://localhost:9090/debug/router/stats | python3 -c "import json,sys; print(json.load(sys.stdin)['raw']['EmbeddingHit']+json.load(sys.stdin)['raw']['EmbeddingMiss']+json.load(sys.stdin)['raw']['KeywordShortcut'])")
HIT=$(curl -s http://localhost:9090/debug/router/stats | python3 -c "import json,sys; print(json.load(sys.stdin)['raw']['EmbeddingHit'])")
MISS=$(curl -s http://localhost:9090/debug/router/stats | python3 -c "import json,sys; print(json.load(sys.stdin)['raw']['EmbeddingMiss'])")
CLASS=$(curl -s http://localhost:9090/debug/router/stats | python3 -c "import json,sys; print(json.load(sys.stdin)['raw']['LLMClassified'])")
FB=$(curl -s http://localhost:9090/debug/router/stats | python3 -c "import json,sys; print(json.load(sys.stdin)['raw']['LLMFallback'])")
SHORT=$(curl -s http://localhost:9090/debug/router/stats | python3 -c "import json,sys; print(json.load(sys.stdin)['raw']['KeywordShortcut'])")

L1_TOTAL=$((HIT + MISS))
L1_PCT=$(python3 -c "print(f'{$HIT / max(1,$L1_TOTAL) * 100:.1f}')")
L2_PCT=$(python3 -c "print(f'{0 if $MISS==0 else $CLASS / $MISS * 100:.1f}')" 2>/dev/null)

echo "║  结果汇总                             ║"
echo "╠══════════════════════════════════════════╣"
printf "║  Step0 短路: %3d 条 (%.1f%%)              ║\n" "$SHORT" "$(python3 -c "print(f'{0 if $TOTAL==0 else $SHORT/$TOTAL*100:.1f}')")"
printf "║  L1 命中:    %3d 条 (${L1_PCT}%%)              ║\n" "$HIT"
printf "║  L2 分类:    %3d 条 (${L2_PCT}%%)              ║\n" "$CLASS"
printf "║  L2 降级:    %3d 条                      ║\n" "$FB"
echo "╠══════════════════════════════════════════╣"
printf "║  用例: ${GREEN}%2d 通过${NC} / ${RED}%2d 失败${NC}                       ║\n" "$PASS" "$FAIL"
echo "╚══════════════════════════════════════════╝"

echo ""
echo "--- 完整统计 ---"
curl -s http://localhost:9090/debug/router/stats | python3 -m json.tool
