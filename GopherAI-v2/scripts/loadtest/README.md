# GopherAI v2 授权压测与限流熔断验证

本目录提供一份基于 `k6` 的授权压测脚本，用于在你**已授权、可控的测试环境**中验证 `GopherAI v2` 的会话列表、普通聊天和流式聊天接口在不同并发下的表现，以及未来接入限流和熔断机制后的保护效果。

## 1. 前置条件

压测前请确认以下条件已经满足：

1. 后端已启动，默认地址为 `http://localhost:9090`。
2. 已存在一个可正常登录的测试账号。
3. 该账号具备正常访问 `/api/v1/AI/*` 接口的 JWT 权限。
4. 若你要压测 `modelType=2` 或 `modelType=3`，请确保对应 RAG 或 MCP 依赖已启动；若只是验证限流本身，建议先使用 `modelType=1`。
5. 压测只在本地开发环境或独立测试环境执行，不要直接对生产环境执行。

## 2. 安装 k6

macOS 可使用以下命令安装：

```bash
brew install k6
```

安装完成后，在 `GopherAI-v2` 目录下执行脚本。

## 3. 环境变量

脚本通过环境变量读取压测配置，避免把账号、密码和地址写死到仓库中。

必填：

```bash
export K6_USERNAME='你的测试账号'
export K6_PASSWORD='你的测试密码'
```

可选：

```bash
export K6_BASE_URL='http://localhost:9090/api/v1'
export K6_MODEL_TYPE='1'
export K6_THINK_TIME_MS='300'
export K6_LIST_VUS='5'
export K6_LIST_DURATION='30s'
export K6_CHAT_VUS='5'
export K6_CHAT_DURATION='30s'
export K6_STREAM_VUS='3'
export K6_STREAM_DURATION='30s'
```

说明：

- `K6_MODEL_TYPE=1`：普通聊天模型，最适合做基线压测。
- `K6_MODEL_TYPE=2`：RAG 模型，适合验证 Redis 向量检索和下游依赖保护。
- `K6_MODEL_TYPE=3`：MCP 模型，适合验证工具调用链路上的熔断与快速失败。
- `K6_THINK_TIME_MS`：每次请求后的思考时间，避免脚本完全无停顿地压住服务。

## 4. 运行脚本

```bash
cd /Users/bytedance/MyProject/GopherAI/GopherAI-v2
k6 run scripts/loadtest/k6-authorized-chat.js
```

该脚本会自动执行以下三类授权场景：

1. 登录拿 Token（只在 `setup()` 执行一次）。
2. `GET /api/v1/AI/chat/sessions` 会话列表。
3. `POST /api/v1/AI/chat/send-new-session` 普通聊天新建会话。
4. `POST /api/v1/AI/chat/send-stream-new-session` 流式聊天新建会话。

流式接口在 `k6` 中会以完整响应体形式被读取，因此脚本会校验返回体里是否含有 `data:`、`sessionId` 或 `[DONE]`，用来确认 SSE 链路至少成功返回。

## 5. 推荐压测分阶段

建议不要一开始就把并发拉高，而是采用下面的分阶段方式：

### 阶段 A：基线验证

目标是确认环境正常、账号可用、脚本与接口格式匹配。

```bash
export K6_LIST_VUS='1'
export K6_CHAT_VUS='1'
export K6_STREAM_VUS='1'
export K6_LIST_DURATION='10s'
export K6_CHAT_DURATION='10s'
export K6_STREAM_DURATION='10s'
k6 run scripts/loadtest/k6-authorized-chat.js
```

预期：

- `checks` 接近 `100%`。
- `http_req_failed` 接近 `0`。
- 聊天接口业务成功码应主要是 `status_code=1000`。
- 流式接口响应体中可以看到 `data:` 和 `[DONE]`。

### 阶段 B：中等并发压测

目标是观察 Gin 服务、MySQL、Redis、RabbitMQ 的基础承压能力。

```bash
export K6_LIST_VUS='10'
export K6_CHAT_VUS='10'
export K6_STREAM_VUS='5'
export K6_LIST_DURATION='1m'
export K6_CHAT_DURATION='1m'
export K6_STREAM_DURATION='1m'
k6 run scripts/loadtest/k6-authorized-chat.js
```

建议同时观察：

- 后端 CPU、内存、Goroutine 数量。
- MySQL 连接数与慢查询。
- Redis RTT、连接数和内存。
- RabbitMQ 队列深度、消费速率、未确认消息数。
- 若启用了 SSE 并发限制，还要关注当前活跃连接数。

### 阶段 C：保护机制验证

目标是验证未来你加上的限流和熔断是否真的生效，而不是“代码写了但压测看不出来”。

先把某一类场景的 VU 快速提高，例如：

```bash
export K6_CHAT_VUS='50'
export K6_STREAM_VUS='20'
export K6_CHAT_DURATION='2m'
export K6_STREAM_DURATION='2m'
k6 run scripts/loadtest/k6-authorized-chat.js
```

预期：

- 如果启用了 Gin 限流，中高并发时会出现一部分 `429`，或者业务返回 `status_code=4001/429`。
- 如果启用了并发保护，流式接口会更早出现 `503` 或业务“服务繁忙”。
- 即使被限流，服务整体仍保持可响应，不应出现大量超时、崩溃或实例无响应。

## 6. 限流验证步骤

如果你已经在 Gin 中接入了限流中间件，可以按下面步骤验证：

1. 先记录一组“未触发限流”的结果，例如 `K6_CHAT_VUS=5`、`K6_STREAM_VUS=2`。
2. 逐步提升单类场景并发，例如把普通聊天提高到 `20 -> 50 -> 100 VUs`。
3. 每次只调整一个场景，避免多个维度同时变化导致结论不清晰。
4. 观察以下现象是否出现：
   - 返回 `429`，或返回你定义的限流业务码。
   - p95 延迟不再无限上升，而是被拒绝得更快。
   - CPU、内存、Goroutine 不会因为请求堆积而持续失控增长。
5. 记录“开始出现限流”的并发点，这就是当前配置的近似拐点。

如果你计划按用户维度限流，压测时建议只用一个账号先验证“单用户被限制”；如果要验证 IP 维度或多用户公平性，可以再复制多个测试账号，扩展脚本为多用户池。

## 7. 熔断验证步骤

熔断验证重点不是“压多少请求”，而是“下游异常时系统是否快速失败并恢复”。推荐按依赖分别验证：

### 验证 LLM 或兼容接口熔断

1. 先在正常依赖下运行脚本，确认聊天接口大多成功。
2. 人为制造下游失败，例如：
   - 把 `OPENAI_BASE_URL` 指到一个不可用地址；
   - 或用测试网关模拟大量 `5xx/超时`；
   - 或临时阻断对模型服务的网络访问。
3. 再运行脚本，观察：
   - 前几次请求会真实打到下游并失败；
   - 当失败率达到阈值后，应开始快速返回 `503` 或业务“服务繁忙/熔断打开”；
   - 此时接口 RT 应明显下降，因为请求不再长期阻塞在下游。
4. 恢复下游后等待熔断超时窗口结束，再重新运行低并发压测，观察是否进入半开并逐步恢复成功率。

### 验证 MCP 熔断

1. 设置 `K6_MODEL_TYPE=3`。
2. 确保问题文本会触发工具调用。
3. 停掉本地 `MCP Server` 或让其返回异常。
4. 重新运行脚本并确认：
   - 初始若干请求失败；
   - breaker 打开后，请求快速失败而不是持续等待；
   - 恢复 MCP 服务后，半开探测请求可重新成功。

### 验证 RAG 相关熔断或降级

1. 设置 `K6_MODEL_TYPE=2`。
2. 让 Redis Vector 或 Embedding 服务不可用。
3. 观察系统行为：
   - 如果你设计的是“检索失败降级为普通聊天”，则成功率应下降有限，但回答会失去知识增强；
   - 如果你设计的是“严格熔断返回错误”，则应尽快返回保护性失败，而不是把请求长时间挂死。

## 8. 如何判定通过

你可以把以下标准作为一轮压测的通过依据：

- 在正常依赖下，基线并发时 `checks > 95%`，`http_req_failed < 5%`。
- 在限流开启后，超额流量被明确拒绝，而不是演变为全局超时。
- 在熔断开启后，下游故障不会导致 API RT 持续飙升或实例雪崩。
- 降压后系统能够恢复，不需要手动重启才能继续服务。

## 9. 扩展建议

当前脚本优先覆盖最关键的授权聊天主链路。如果你后续还要验证：

- `/file/upload` 的上传限流与并发保护；
- `/AI/chat/tts` 与 `/AI/chat/tts/query` 的外部 TTS 依赖保护；
- 多用户池、公平性和租户隔离；

可以在现有脚本基础上继续扩展新的 `scenario`，并为每类接口设置独立阈值。
