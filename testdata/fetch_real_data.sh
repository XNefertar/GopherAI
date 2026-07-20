#!/bin/bash
# 下载公开真实对话数据集，随机采样 200 条做路由器压测
# 来源: BELLE (中文指令) + Dolly (英文指令)
# 用法: bash testdata/fetch_real_data.sh

set -e
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6MywidXNlcm5hbWUiOiIxMzgyOTI3MTA0OCIsImlzcyI6Imh1YW5oZWFydCIsInN1YiI6IkdvcGhlckFJIiwiZXhwIjoxODE2MDgyMjU2LCJpYXQiOjE3ODQ1NDYyNTZ9.3wdoSvDiV-sRjyTYWOYyZHEPaqHwSx_MKw4TjOeQ_v8"
BASE="http://localhost:9090/api/v1/AI/chat/send-new-session"
SAMPLES=200

echo "=== 1. 下载数据集 ==="

# BELLE 中文指令数据集（3.5M+ 条，取 school_math 子集最快）
mkdir -p testdata/datasets
cd testdata/datasets

# 下载 BELLE 子集（约 20MB，包含各类真实提问）
if [ ! -f belle_10k.jsonl ]; then
  # 用 HuggingFace 下载（不需要安装 Python 库，直接 curl）
  curl -L -o belle_sample.zip \
    "https://huggingface.co/datasets/BelleGroup/train_3.5M_CN/resolve/main/Belle_open_source_1M.json?download=true" 2>/dev/null || true
  
  # 备选：直接生成随机合法采样数据
  echo "Downloading samples from HuggingFace..."
  pip3 install datasets huggingface_hub -q 2>/dev/null || true
  
  python3 -c "
import json, random
try:
    from datasets import load_dataset
    # 下载 BELLE 中文指令（1000 条样本）
    ds = load_dataset('BelleGroup/train_3.5M_CN', split='train', streaming=True)
    samples = []
    for i, item in enumerate(ds):
        if i >= 1000:
            break
        q = item.get('instruction', '').strip()
        if len(q) > 3 and len(q) < 200:
            samples.append(q)
    with open('belle_1k_queries.txt', 'w') as f:
        for s in samples:
            f.write(s + '\n')
    print(f'Saved {len(samples)} Belle queries')
except Exception as e:
    print(f'Dataset download failed, generating fallback data: {e}')
    # 降级：生成涵盖多种意图的随机中文 query 模板
    templates = [
        '请帮我{}', '怎么{}', '{}是什么意思', '能不能{}', '{}应该怎么做',
        '{}和{}有什么区别', '为什么{}', '{}的最佳实践是什么',
        '参考文档里的{}', '帮我查一下{}', '{}的步骤是什么',
        '{}有什么风险', '{}怎么优化', '解释一下{}', '{}怎么写'
    ]
    topics = ['代码','数据库','架构','性能','安全','部署','测试','API','缓存','消息队列',
              '微服务','容器','监控','日志','文档','协议','接口','算法','数据结构','网络',]
    random.seed(42)
    queries = []
    for _ in range(500):
        t = random.choice(templates)
        topic = random.choice(topics)
        q = t.format(topic, random.choice(topics)) if '{}' in t else t.format(topic)
        queries.append(q)
    with open('belle_1k_queries.txt', 'w') as f:
        for q in queries:
            f.write(q + '\n')
    print(f'Saved {len(queries)} fallback queries')
" 2>/dev/null || true

  wc -l belle_1k_queries.txt 2>/dev/null && mv belle_1k_queries.txt ../belle_1k_queries.txt
fi

cd ../..

echo ""
echo "=== 2. 如果 Belle 下载失败，用模板生成后备数据 ==="
if [ ! -f testdata/belle_1k_queries.txt ] || [ $(wc -l < testdata/belle_1k_queries.txt) -lt 50 ]; then
  python3 -c "
import random
random.seed(42)
templates = [
    '分析一下这段{}的性能瓶颈', '{}和{}有什么区别', '怎么优化{}',
    '{}的设计模式有哪些', '帮我写一个{}的实现', '{}的原理是什么',
    '为什么{}会失败', '有什么更好的方案替代{}', '解释一下{}算法',
    '{}的最佳实践', '如何保证{}的高可用', '{}和{}选哪个',
    '参考文档，{}要怎么做', '{}有什么安全风险', '如何排查{}的问题',
    '{}怎么测试', '设计一个{}系统', '重构这个{}',
    '{}是什么', '{}应该怎么学', '有没有{}的教程',
    '这个{}为什么不对', '帮我改一下{}', '{}怎么做性能测试',
    '{}的优缺点', '查一下{}的资料', '{}要怎么配置',
    '{}的版本怎么选', '{}报错了怎么办', '{}环境怎么搭建',
    '根据白皮书，{}', '{}需要哪些依赖', '{}的源码分析',
    '帮忙看一下这个{}', '{}怎么调试', '{}用哪种实现比较好',
    '{}的定义', '{}和{}可以同时用吗', '{}怎样部署',
    '{}的注意事项', '{}常见面试题', '{}怎么看日志',
    '调用接口获取{}', '执行命令查看{}', '{}监控要关注哪些指标',
]
topics = ['SQL查询','Redis缓存','消息队列','微服务','系统架构','API设计',
          '数据库','并发编程','内存泄漏','分布式事务','JWT鉴权','限流策略',
          '日志系统','Docker','K8s','负载均衡','数据库索引','线程池',
          '连接池','SSL/TLS','OAuth2','RESTful','WebSocket','gRPC',
          '正则表达式','排序算法','哈希表','二叉树','链表','网络协议',
          'Nginx','MySQL','RabbitMQ','ElasticSearch','Prometheus',
          '认证授权','数据同步','CI/CD','代码规范','设计模式','Python','Go','Java',
          'pytest','mock测试','单元测试','集成测试','端到端测试',
          '虚拟环境','包管理','依赖注入','配置中心','服务发现','熔断降级',
          '异地多活','数据分片','读写分离','数据库迁移','代码重构',
          '接口文档','Git','Linux','Shell','Docker Compose',
          '容器编排','日志收集','异常处理','错误码']
queries = []
for i in range(300):
    t = random.choice(templates)
    topics_sample = random.sample(topics, 2)
    q = t.format(*topics_sample[:t.count('{}')])
    queries.append(q)
# 再加一些闲聊和口语化表达
extras = ['你好','谢谢','hello','在吗','早上好','今天天气怎么样',
          '这个怎么回事','帮我看看','为什么','然后呢',
          '讲个笑话','最近好吗','今天心情不好',
          '推荐一本书','周末去哪里玩','你会写诗吗',
          '你的优势是什么','能帮我写周报吗']
queries.extend(extras)
with open('testdata/belle_1k_queries.txt','w') as f:
    for q in queries:
        f.write(q.strip() + '\n')
print(f'Generated {len(queries)} diverse test queries')
"
fi

TOTAL=$(wc -l < testdata/belle_1k_queries.txt | tr -d ' ')
echo "=== 3. 从 ${TOTAL} 条真实风格 query 中随机采样 ${SAMPLES} 条 ==="

# 随机采样（macOS 没有 shuf，用 sort -R）
sort -R testdata/belle_1k_queries.txt | head -$SAMPLES > testdata/sampled_queries.txt

echo "=== 4. 并行发送请求（每次 3 条并发，单条超时 20s）==="
START_TIME=$(date +%s)
COUNT=0
while IFS= read -r query; do
  query=$(echo "$query" | tr -d '\n\r')
  [ -z "$query" ] && continue
  
  # 发送请求（后台并行，单条超时 20s）
  curl -s --max-time 20 -X POST "$BASE" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"modelType\":\"auto\",\"question\":\"$query\"}" > /dev/null &
  
  COUNT=$((COUNT + 1))
  
  # 每 3 条等一批完成
  if [ $((COUNT % 3)) -eq 0 ]; then
    wait
    ELAPSED=$(($(date +%s) - START_TIME))
    echo "  [${ELAPSED}s] 已发送 $COUNT / $SAMPLES 条"
  fi
done < testdata/sampled_queries.txt

# 等最后一批完成
wait
ELAPSED=$(($(date +%s) - START_TIME))

echo ""
echo "=== 5. 测试完成（总耗时 ${ELAPSED}s），查看路由统计 ==="
sleep 2
curl -s http://localhost:9090/debug/router/stats | python3 -m json.tool

echo ""
echo "=== 采样 query 样本（前 10 条）==="
head -10 testdata/sampled_queries.txt
