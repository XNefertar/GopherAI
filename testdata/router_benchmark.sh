#!/bin/bash
# 路由器 QPS 与延迟测试
# 用法: bash testdata/router_benchmark.sh [并发数] [请求总数]
# 示例: bash testdata/router_benchmark.sh 5 20

CONCURRENCY=${1:-3}
TOTAL=${2:-15}
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6MywidXNlcm5hbWUiOiIxMzgyOTI3MTA0OCIsImlzcyI6Imh1YW5oZWFydCIsInN1YiI6IkdvcGhlckFJIiwiZXhwIjoxODE2MDgyMjU2LCJpYXQiOjE3ODQ1NDYyNTZ9.3wdoSvDiV-sRjyTYWOYyZHEPaqHwSx_MKw4TjOeQ_v8"
BASE="http://localhost:9090/api/v1/AI/chat/send-new-session"

# 测试 query 池（覆盖 4 类意图）
QUERIES=(
  "分析这段 SQL 的性能瓶颈"
  "你好"
  "根据文档回答审批流程"
  "比较微服务和单体架构"
  "帮我查一下天气"
  "谢谢"
  "设计一个高可用系统"
  "知识库里怎么说的"
  "创建工单"
  "为什么接口会超时"
  "上传的文件里怎么说的"
  "这个怎么回事"
  "解释 CAP 定理"
  "今天星期几"
  "帮我看看"
)

RESULT_FILE="/tmp/router_bench_$$.txt"
> $RESULT_FILE

echo "╔════════════════════════════════════╗"
echo "║  路由器性能测试                     ║"
echo "╠════════════════════════════════════╣"
echo "║  并发数: $CONCURRENCY                       ║"
echo "║  请求数: $TOTAL                         ║"
echo "╚════════════════════════════════════╝"
echo ""

# 发送单个请求并记录耗时
send_one() {
  local q="$1"
  local start=$(python3 -c "import time; print(int(time.time()*1000))")
  local status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"modelType\":\"auto\",\"question\":\"$q\"}" 2>/dev/null)
  local end=$(python3 -c "import time; print(int(time.time()*1000))")
  local elapsed=$((end - start))
  echo "$elapsed $status" >> $RESULT_FILE
  echo -n "."
}

echo ">>> 发送请求..."
START_TIME=$(date +%s)

# 批量发送：每次 $CONCURRENCY 条并发，等这批完成再发下一批
SENT=0
while [ $SENT -lt $TOTAL ]; do
  BATCH=0
  while [ $BATCH -lt $CONCURRENCY ] && [ $SENT -lt $TOTAL ]; do
    idx=$((SENT % ${#QUERIES[@]}))
    send_one "${QUERIES[$idx]}" &
    BATCH=$((BATCH + 1))
    SENT=$((SENT + 1))
  done
  wait
done

END_TIME=$(date +%s)
TOTAL_TIME=$((END_TIME - START_TIME))

echo ""
echo ""

# 统计延迟
LATENCIES=$(awk '{print $1}' $RESULT_FILE | sort -n)
COUNT=$(echo "$LATENCIES" | wc -l | tr -d ' ')
TOTAL_LAT=$(echo "$LATENCIES" | awk '{sum+=$1} END{print sum}')
AVG=$(python3 -c "print(f'{0 if $COUNT==0 else $TOTAL_LAT//$COUNT}')")

# P50, P90, P99
P50=$(echo "$LATENCIES" | awk -v n="$COUNT" 'NR == int(n * 0.50) {print $1}')
P90=$(echo "$LATENCIES" | awk -v n="$COUNT" 'NR == int(n * 0.90) {print $1}')
P99=$(echo "$LATENCIES" | awk -v n="$COUNT" 'NR == int(n * 0.99) {print $1}')
MIN=$(echo "$LATENCIES" | head -1)
MAX=$(echo "$LATENCIES" | tail -1)

# QPS = 总请求数 / 总耗时（秒）
QPS=$(python3 -c "print(f'{0 if $TOTAL_TIME==0 else $COUNT / $TOTAL_TIME:.2f}')")

# HTTP 状态码分布
HTTP_OK=$(awk '$2==200{count++} END{print count+0}' $RESULT_FILE)
HTTP_ERR=$(awk '$2!=200{count++} END{print count+0}' $RESULT_FILE)

echo "╔════════════════════════════════════╗"
echo "║  测试结果                           ║"
echo "╠════════════════════════════════════╣"
printf "║  总请求:  %3d 条                     ║\n" "$COUNT"
printf "║  总耗时:  %3d 秒                     ║\n" "$TOTAL_TIME"
printf "║  QPS:     %6.2f  req/s             ║\n" "$QPS"
echo "╠════════════════════════════════════╣"
printf "║  延迟 (ms):                         ║\n"
printf "║    Min:  %6d                       ║\n" "$MIN"
printf "║    Avg:  %6d                       ║\n" "$AVG"
printf "║    P50:  %6d                       ║\n" "$P50"
printf "║    P90:  %6d                       ║\n" "$P90"
printf "║    P99:  %6d                       ║\n" "$P99"
printf "║    Max:  %6d                       ║\n" "$MAX"
echo "╠════════════════════════════════════╣"
printf "║  HTTP:  200=%d err=%d                 ║\n" "$HTTP_OK" "$HTTP_ERR"
echo "╚════════════════════════════════════╝"

echo ""
echo "--- 原始延迟分布（前 20 条）---"
echo "$LATENCIES" | head -20 | awk '{printf "  %4d ms  ", $1; for(i=0;i<$1/200;i++) printf "#"; print ""}'

rm -f $RESULT_FILE
