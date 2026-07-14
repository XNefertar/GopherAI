# GopherAI：企业级 Go 语言 AI 应用服务平台

GopherAI 是一个基于 Go 语言构建的、高并发、可扩展且真正具备落地商用能力的 AI 应用服务平台。

本项目不仅仅是一个调用大模型 API 的简单 Demo，而是深入工程化实践，涵盖了**异步消息处理**、**多会话上下文流式输出**、**多模型工厂模式接入**以及**多模态（图像识别、TTS、RAG）**等复杂的真实业务场景。旨在提供一套从后端服务到前端交互（Vue3）的全栈 AI 产品级解决方案。

---

## 🚀 核心特性

- **🧠 智能多会话管理与上下文记忆**
  自定义 `AIHelperManager` 调度器，支持单用户多会话隔离。会话历史采用**按需惰性加载**：首次访问某会话时从 MySQL 拉取历史并重建内存上下文，无需启动时全量预热，启动复杂度从 O(全表消息) 降为 O(1)。内存通过 **LRU 容量上限 + 空闲 TTL 回收 + 淘汰前 flush** 封住无限增长，避免 OOM（详见 `ARCHITECTURE.md` §13.7）。
- **⚡ 流式输出 (SSE) 交互**
  后端通过 Server-Sent Events (SSE) 协议推送大模型数据流，前端实现极低延迟的打字机式响应效果。
- **🏗️ 模块化多模型接入 (工厂模式)**
  底层集成 ByteDance 的 EINO AI 框架，通过工厂模式（Factory Pattern）无缝支持 OpenAI 以及本地化部署的 Ollama 等多种 LLM，具备极强的扩展性。
- **🚀 高并发异步落库 (RabbitMQ)**
  为了保护数据库免受高频聊天请求的冲击，所有聊天记录先写入内存数组，并异步推送至 RabbitMQ，由后台消费者（Worker）平滑写入 MySQL，极大提升系统并发吞吐量。
- **👁️ 多模态与扩展能力 (GopherAI-v2)**
  - **图像识别**：集成 ONNXRuntime 与 MobileNetV2，支持图片上传后的本地端到端推理与分类。
  - **RAG (检索增强生成)**：结合知识库为大模型提供私有数据增强。
  - **语音合成 (TTS)**：将 AI 生成的文本转换为自然语音。
  - **MCP (Model Context Protocol)**：集成 MCP 客户端，便于大模型调用外部工具链。
- **🔐 健壮的用户体系**
  基于 Redis 缓存验证码的邮箱注册机制，配合 JWT (JSON Web Token) 进行接口安全鉴权，支持完整闭环的注册、登录与访问控制。

---

## 🛠️ 技术栈

### 后端 (Backend)
- **核心框架**: Go (Golang) + Gin (Web 框架)
- **AI 框架**: EINO (ByteDance 开源大模型开发框架)
- **数据库/ORM**: MySQL + GORM (自动迁移与连接池管理)
- **缓存**: Redis (验证码管理、防重放、RAG 向量检索)
- **消息队列**: RabbitMQ (异步日志落库)
- **机器学习/推理**: ONNXRuntime-Go (图像识别本地推理)
- **配置治理**: TOML(基础设施) + 环境变量(模型密钥) 统一收口，启动时 fail-fast 校验
- **其他组件**: JWT 鉴权、SMTP 邮件发送

### 前端 (Frontend)
- **核心框架**: Vue 3 + Vue Router
- **构建工具**: Webpack
- **主要页面**: 登录/注册、AI 聊天 (SSE 接收)、多模态功能展示页

---

## � 项目结构概览

本项目分为 `v1` 和 `v2` 两个版本：
- **`GopherAI-v1`**：实现了基础的核心链路，包括完整的用户体系、AI 聊天流式输出、Redis+RabbitMQ 异步高并发处理、以及基于 ONNX 的图像识别。
- **`GopherAI-v2`**：在 v1 基础上进行了架构重构和能力扩充，新增了 RAG (知识检索)、TTS (语音合成)、以及 MCP 协议集成，代码组织更趋近企业级微服务标准。

核心目录说明：

```text
GopherAI/
├── common/             # 公共基础设施层
│   ├── aihelper/       # 核心！AI 助手管理器、模型工厂与大模型调用逻辑
│   ├── mysql/          # GORM 数据库初始化与连接池配置
│   ├── rabbitmq/       # 消息队列初始化与发布/消费逻辑
│   ├── redis/          # Redis 缓存初始化与验证码校验
│   ├── rag/            # 检索增强生成（向量化 + 检索）
│   ├── titlesummary/   # 会话标题自动生成
│   └── mcp/            # Model Context Protocol 客户端/服务端实现
├── config/             # 配置治理层（TOML + 环境变量统一收口 + fail-fast 校验）
├── controller/         # HTTP 请求入口 (参数绑定与响应返回)
├── service/            # 核心业务逻辑层 (协调 Controller 与 Dao)
├── dao/                # 数据访问层 (封装数据库 CRUD)
├── middleware/         # Gin 中间件 (JWT 鉴权等)
├── model/              # 数据库结构体定义 (GORM Tags)
├── router/             # API 路由注册中心
├── vue-frontend/       # Vue3 前端工程代码
└── main.go             # 项目启动入口 (初始化资源、启动 Web 服务，会话历史按需惰性加载)
```

---

## 🏃 快速开始

### 1. 环境准备
确保你的开发环境已安装并运行以下中间件：
- Go 1.20+
- MySQL 5.7+ 或 8.0+
- Redis
- RabbitMQ
- Node.js & npm (用于前端)

### 2. 配置说明（两套来源，统一收口）

本项目采用**基础设施走 TOML、模型密钥走环境变量**的双源策略，二者在 `config` 包内统一收口，启动时一次性加载并校验。

**① 基础设施配置（`config/config.local.toml`）**
复制模板并填入本地中间件连接信息：
```toml
[mysqlConfig]
host = "127.0.0.1"
port = 3306
user = "root"
password = "your_password"
databaseName = "GopherAI"

# Redis / RabbitMQ / Email / JWT / RAG / 语音 等配置同样在此填写
```

**② 模型密钥配置（`.env.local` 或 `.env`）**
复制 `.env.example` 为 `.env.local` 并填入真实密钥：
```bash
cp .env.example .env.local
```
```bash
# .env.local —— 仅需填写 AI 模型相关环境变量
export OPENAI_API_KEY=sk-your-api-key-here
export OPENAI_MODEL_NAME=deepseek-chat
export OPENAI_BASE_URL=https://api.deepseek.com/v1
# VISION_ / TITLE_ / OLLAMA_ / RAG_ 等可选，详见 .env.example
```

> **无需手动 `source`**：进程启动时会自动读取 `.env.local` → `.env` 并注入环境变量（已存在的变量不覆盖），直接 `go run main.go` 即可。

### 3. 启动后端服务
```bash
go mod tidy
go run main.go
```
*(注：首次启动时，GORM `AutoMigrate` 会自动在 MySQL 中创建对应的数据表。)*

**启动期行为（fail-fast）：**
- 配置文件路径解析优先级：`GOPHERAI_CONFIG` 环境变量 > `config/config.local.toml` > `config/config.toml`。
- 若缺失核心模型配置（`OPENAI_API_KEY` / `OPENAI_MODEL_NAME` / `OPENAI_BASE_URL`），进程会在启动**第一秒**直接退出并打印：
  ```
  [config] init failed: model config missing required env: OPENAI_API_KEY, ...
  ```
  这是预期行为——缺配在启动期暴露，而非上线后运行时偶发崩溃。

### 4. 启动前端页面
```bash
cd vue-frontend
npm install
npm run serve
```
访问前端控制台输出的本地地址，即可体验完整的 GopherAI 应用！

---

## 📌 设计亮点解析

1. **为什么用 RabbitMQ？**：聊天场景下写操作极为频繁，直接落库会导致数据库连接池耗尽。引入 MQ 实现“写缓冲”，不仅提高了接口响应速度，还保证了数据库的平稳运行。
2. **AIHelperManager 的意义**：HTTP 是无状态的，但对话需要记忆。Manager 就像一个“全局内存收纳柜”，将每个用户的 `SessionID` 和包含聊天历史的 `AIHelper` 实例绑定。会话历史采用**惰性加载**（首次访问时从 DB 按 `sessionID` 拉取并重建上下文，`hydrated` 标记保证幂等），而非启动时全量预热——既避免大表扫描拖慢启动、又消除「历史强依赖内存」的死循环依赖。在此之上已落地 **LRU 容量治理**（容量上限 + 空闲 TTL 回收 + 淘汰前 flush 兜底），封住内存无限增长、消除 OOM 风险，为 Phase 3 多实例水平扩容打通主链路（详见 `ARCHITECTURE.md` §13.7）。
3. **配置是可治理的一等公民（统一收口 + fail-fast）**：模型密钥原本散落在 `model.go`、`summary.go`、`rag.go` 等 6+ 个文件中各自 `os.Getenv`，既无法集中校验、也容易漏配。现已全部收口到 `config.ModelConfig` 单一结构体，启动时统一读取、集中校验；基础设施走 TOML、模型密钥走环境变量，二者在 `config` 包内收敛为「环境变量 > TOML > 默认值」的单一加载策略。核心密钥缺失时启动即退出，杜绝「能启动但运行时才崩」的隐患。
4. **配置文件选用 TOML**：摒弃了易出错的 JSON 和 YAML，采用可读性极强、支持注释且类型安全的 TOML 格式，结合 Go 的反射机制 `toml.DecodeFile`，实现优雅的配置映射。
