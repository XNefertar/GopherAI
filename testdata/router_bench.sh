#!/bin/bash
# 路由器压力测试：60 条真实风格 query，覆盖四类意图 + 边缘 case
# 用法: bash testdata/router_bench.sh

TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6MywidXNlcm5hbWUiOiIxMzgyOTI3MTA0OCIsImlzcyI6Imh1YW5oZWFydCIsInN1YiI6IkdvcGhlckFJIiwiZXhwIjoxODE2MDgyMjU2LCJpYXQiOjE3ODQ1NDYyNTZ9.3wdoSvDiV-sRjyTYWOYyZHEPaqHwSx_MKw4TjOeQ_v8"
BASE="http://localhost:9090/api/v1/AI/chat/send-new-session"

echo "=== 路由器压力测试开始（共 60 条 query）==="
echo ""

send() { curl -s -X POST "$BASE" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "{\"modelType\":\"auto\",\"question\":\"$1\"}" > /dev/null; }

# ═══════════════════════════════════════════
# 1. 闲聊类 (15条) - 期望 Step0 短路或 L1 chat
# ═══════════════════════════════════════════
echo ">>> 闲聊类 (15条)"
send "你好"
send "晚上好"
send "谢谢你的帮助"
send "再见"
send "hello"
send "thank you"
send "good morning"
send "你是谁"
send "今天星期几"
send "讲个笑话"
send "你能做什么"
send "最近怎么样"
send "好无聊啊"
send "有什么好玩的"
send "没事"

# ═══════════════════════════════════════════
# 2. 推理/分析/编码类 (18条) - 期望 L1 reasoning 或 L2
# ═══════════════════════════════════════════
echo ">>> 推理/编码类 (18条)"
send "分析一下这段 SQL 为什么慢"
send "对比 React 和 Vue 的响应式原理"
send "帮我设计一个秒杀系统的架构"
send "这段代码的时间复杂度是多少"
send "重构这个函数让它更简洁"
send "为什么这个接口在高并发下会超时"
send "解释一下 CAP 定理"
send "如果我要实现一个 LRU 缓存，应该怎么设计"
send "微服务和单体架构各自的优缺点是什么"
send "这个 bug 的根因是什么"
send "怎么写一个线程安全的单例模式"
send "解释一下 Go 的 goroutine 调度原理"
send "MySQL 索引为什么用 B+ 树而不是哈希"
send "帮我 review 一下这段代码"
send "有没有更好的算法解决这个问题"
send "TCP 和 UDP 的区别是什么，什么时候用哪个"
send "怎么优化这个 API 的响应时间"
send "这个正则表达式为什么匹配不到"

# ═══════════════════════════════════════════
# 3. 文档/知识库类 (15条) - 期望 L1 knowledge 或 L2
# ═══════════════════════════════════════════
echo ">>> 知识库类 (15条)"
send "根据上传的文档回答这个问题"
send "白皮书里关于数据安全的条款是什么"
send "查一下资料里的相关记录"
send "这个在手册第几页"
send "上传的文件中有没有提到审批流程"
send "参考知识库里的内容，解释一下"
send "这个规范是怎么要求的"
send "文档里对接口定义是怎么写的"
send "根据参考文档，这个字段的含义是什么"
send "之前上传的那个 pdf 里怎么说的"
send "看看资料里有没有这个问题的答案"
send "技术方案文档里提到了哪些风险"
send "根据产品需求文档，这个功能要怎么实现"
send "查一下标准操作流程里的步骤"
send "运维手册里对这个报错怎么处理的"

# ═══════════════════════════════════════════
# 4. 工具调用类 (7条) - 期望 L1 tool 或 L2
# ═══════════════════════════════════════════
echo ">>> 工具调用类 (7条)"
send "帮我查一下北京的天气"
send "创建一个新的工单"
send "拉取最新的监控数据"
send "执行数据库备份命令"
send "帮我查一下这个用户的订单信息"
send "调用短信接口发送验证码"
send "查询服务器当前 CPU 使用率"

# ═══════════════════════════════════════════
# 5. 边缘/歧义类 (5条) - 考验 L1→L2 降级能力
# ═══════════════════════════════════════════
echo ">>> 边缘/歧义类 (5条)"
send "这个怎么回事"           # 很模糊，没有上下文
send "帮我看看"               # 极短，意图不明
send "为什么"                 # 只有一个词
send "怎么说"                 # 口语化，无明确指向
send "然后呢"                 # 依赖多轮对话上下文

echo ""
echo "=== 测试完成，查看统计 ==="
sleep 1
curl -s http://localhost:9090/debug/router/stats | python3 -m json.tool
