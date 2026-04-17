# GopherAI：企业级 Go 语言 AI 应用服务平台

GopherAI 是一个基于 Go 语言构建的、高并发、可扩展且真正具备落地商用能力的 AI 应用服务平台。

本项目不仅仅是一个调用大模型 API 的简单 Demo，而是深入工程化实践，涵盖了**异步消息处理**、**多会话上下文流式输出**、**多模型工厂模式接入**以及**多模态（图像识别、TTS、RAG）**等复杂的真实业务场景。旨在提供一套从后端服务到前端交互（Vue3）的全栈 AI 产品级解决方案。

---

## 🚀 核心特性

- **🧠 智能多会话管理与上下文记忆**
  自定义 `AIHelperManager` 调度器，支持单用户多会话隔离。系统启动时自动从 MySQL 加载历史聊天记录，重构会话内存上下文，实现无缝持久化记忆。
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
- **缓存**: Redis (验证码管理、防重放)
- **消息队列**: RabbitMQ (异步日志落库)
- **机器学习/推理**: ONNXRuntime-Go (图像识别本地推理)
- **其他组件**: TOML 配置解析、JWT 鉴权、SMTP 邮件发送

### 前端 (Frontend)
- **核心框架**: Vue 3 + Vue Router
- **构建工具**: Webpack
- **主要页面**: 登录/注册、AI 聊天 (SSE 接收)、多模态功能展示页

---

## � 项目结构概览

本项目分为 `v1` 和 `v2` 两个版本：
- **`GopherAI-v1`**：实现了基础的核心链路，包括完整的用户体系、AI 聊天流式输出、Redis+RabbitMQ 异步高并发处理、以及基于 ONNX 的图像识别。
- **`GopherAI-v2`**：在 v1 基础上进行了架构重构和能力扩充，新增了 RAG (知识检索)、TTS (语音合成)、以及 MCP 协议集成，代码组织更趋近企业级微服务标准。

以 `v2` 为例的核心目录说明：

```text
GopherAI/GopherAI-v2/
├── common/             # 公共基础设施层
│   ├── aihelper/       # 核心！AI 助手管理器、模型工厂与大模型调用逻辑
│   ├── image/          # ONNXRuntime 图像推理引擎
│   ├── mysql/          # GORM 数据库初始化与连接池配置
│   ├── rabbitmq/       # 消息队列初始化与发布/消费逻辑
│   ├── redis/          # Redis 缓存初始化与验证码校验
│   └── mcp/            # Model Context Protocol 客户端/服务端实现
├── config/             # TOML 配置文件及解析解析逻辑
├── controller/         # HTTP 请求入口 (参数绑定与响应返回)
├── service/            # 核心业务逻辑层 (协调 Controller 与 Dao)
├── dao/                # 数据访问层 (封装数据库 CRUD)
├── middleware/         # Gin 中间件 (JWT 鉴权等)
├── model/              # 数据库结构体定义 (GORM Tags)
├── router/             # API 路由注册中心
├── vue-frontend/       # Vue3 前端工程代码
└── main.go             # 项目启动入口 (初始化资源、加载历史会话、启动 Web 服务)
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

### 2. 配置文件
进入 `GopherAI-v2/config/` (或 v1 目录)，修改 `config.toml`，填入你本地的中间件连接信息以及相关的邮箱/AI 大模型配置：
```toml
[mysqlConfig]
host = "127.0.0.1"
port = 3306
user = "root"
password = "your_password"
databaseName = "GopherAI"

# ... 修改 Redis、RabbitMQ、Email 和 大模型相关的配置
```

### 3. 启动后端服务
```bash
cd GopherAI-v2
go mod tidy
go run main.go
```
*(注：首次启动时，GORM `AutoMigrate` 会自动在 MySQL 中创建对应的数据表。)*

### 4. 启动前端页面
```bash
cd GopherAI-v2/vue-frontend
npm install
npm run serve
```
访问前端控制台输出的本地地址，即可体验完整的 GopherAI 应用！

---

## 📌 设计亮点解析

1. **为什么用 RabbitMQ？**：聊天场景下写操作极为频繁，直接落库会导致数据库连接池耗尽。引入 MQ 实现“写缓冲”，不仅提高了接口响应速度，还保证了数据库的平稳运行。
2. **AIHelperManager 的意义**：HTTP 是无状态的，但对话需要记忆。Manager 就像一个“全局内存收纳柜”，将每个用户的 `SessionID` 和包含聊天历史的 `AIHelper` 实例绑定，并在服务重启时自动从 DB 预热，实现商用级别的持久化。
3. **配置文件选用 TOML**：摒弃了易出错的 JSON 和 YAML，采用可读性极强、支持注释且类型安全的 TOML 格式，结合 Go 的反射机制 `toml.DecodeFile`，实现优雅的配置映射。
