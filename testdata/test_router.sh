# 保存为 test_router.sh
#!/bin/bash
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6MiwidXNlcm5hbWUiOiIzNDM4MTEwNDMxMCIsImlzcyI6Imh1YW5oZWFydCIsInN1YiI6IkdvcGhlckFJIiwiZXhwIjoxODE1ODE2NjQwLCJpYXQiOjE3ODQyODA2NDB9.8Q-W2gJStmwCyIhTeciod66ngp3IOIiA-taKgDrPkeM"
BASE="http://localhost:9090/api/v1/AI/chat/send-new-session"

# ===== 闲聊类（期望命中 Step0 或 L1 chat）=====
echo "=== 闲聊类 ==="
curl -s -X POST $BASE -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"modelType":"auto","question":"你好"}' | grep -c status_code
curl -s -X POST $BASE -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"modelType":"auto","question":"谢谢"}' | grep -c status_code
curl -s -X POST $BASE -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"modelType":"auto","question":"早上好"}' | grep -c status_code

# ===== 推理类（期望命中 L1 reasoning）=====
echo "=== 推理类 ==="
curl -s -X POST $BASE -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"modelType":"auto","question":"分析这段代码的性能瓶颈"}' | grep -c status_code
curl -s -X POST $BASE -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"modelType":"auto","question":"比较A方案和B方案的优缺点"}' | grep -c status_code
curl -s -X POST $BASE -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"modelType":"auto","question":"设计一个高可用系统架构"}' | grep -c status_code

# ===== 知识库类（期望命中 L1 knowledge）=====
echo "=== 知识库类 ==="
curl -s -X POST $BASE -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"modelType":"auto","question":"根据文档回答这个问题"}' | grep -c status_code
curl -s -X POST $BASE -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"modelType":"auto","question":"上传的文件里怎么说的"}' | grep -c status_code

# ===== 模糊类（考验 L1 能否拦截，不行的掉到 L2）=====
echo "=== 模糊类 ==="
curl -s -X POST $BASE -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"modelType":"auto","question":"帮我看看这个怎么回事"}' | grep -c status_code
curl -s -X POST $BASE -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"modelType":"auto","question":"这个为什么不对"}' | grep -c status_code
curl -s -X POST $BASE -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"modelType":"auto","question":"上一次那个怎么弄的"}' | grep -c status_code

# ===== 工具类（期望命中 L1 tool）=====
echo "=== 工具类 ==="
curl -s -X POST $BASE -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"modelType":"auto","question":"帮我查一下今天的天气"}' | grep -c status_code
curl -s -X POST $BASE -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"modelType":"auto","question":"创建一个工单"}' | grep -c status_code

echo ""
echo "=== 查看路由器统计 ==="
curl -s http://localhost:9090/debug/router/stats | python3 -m json.tool
