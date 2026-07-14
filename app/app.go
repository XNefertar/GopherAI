package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"GopherAI/common/aihelper"
	"GopherAI/common/mysql"
	"GopherAI/common/rabbitmq"
	"GopherAI/common/redis"
	appconfig "GopherAI/config"
	"GopherAI/dao/message"
	"GopherAI/router"
)

// shutdownTimeout 是优雅停机各阶段的总超时上限。
// 超过该时间后，即使仍有 in-flight 请求/消息也会强制退出，避免进程 hang 死。
const shutdownTimeout = 30 * time.Second

// App 负责应用的依赖装配与生命周期管理。
//
// 设计动机：
//   - 原 main.go 把「初始化顺序」「HTTP 启动」「信号处理」全部手写揉在一起，
//     既无法优雅停机，也存在 readDataFromDB 早于 InitRabbitMQ 的启动竞态。
//   - App 把「装配（Init）→ 运行（Run）→ 优雅停机（Shutdown）」三段式生命周期
//     收敛为单一入口，关闭顺序与依赖顺序严格逆向对应，体现应用骨架（Bootstrap）的分层设计。
type App struct {
	server *http.Server
}

// New 构造应用实例。
func New() *App {
	return &App{}
}

// Run 装配全部依赖、启动服务，并阻塞直到收到终止信号后优雅退出。
func (a *App) Run() error {
	conf := appconfig.GetConfig()
	host := conf.MainConfig.Host
	port := conf.MainConfig.Port

	// 1. 基础设施依赖装配（顺序即依赖顺序）
	if err := mysql.InitMysql(); err != nil {
		return fmt.Errorf("init mysql: %w", err)
	}
	redis.Init()
	rabbitmq.InitRabbitMQ()

	// 2. 内存会话预热（必须在 InitRabbitMQ 之后：AIHelper 的 saveFunc 依赖 MQ 发布者）
	if err := readDataFromDB(); err != nil {
		// 预热失败不阻塞启动，仅告警（历史消息缺失可后续按需加载）
		log.Printf("[app] warn: preload sessions from db failed: %v", err)
	}

	// 3. 启动 HTTP 服务（在 goroutine 中运行，便于主协程阻塞等待信号）
	r := router.InitRouter()
	a.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, port),
		Handler: r,
	}
	go func() {
		log.Printf("[app] http server listening on %s", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[app] http server fatal: %v", err)
		}
	}()

	// 4. 阻塞等待终止信号
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	log.Println("[app] received shutdown signal, draining...")

	// 5. 优雅停机
	return a.Shutdown()
}

// Shutdown 按依赖逆序释放资源：先停止接入新请求，再排空 MQ 消费，最后回收连接。
func (a *App) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// ① 停止接入：拒绝新请求，等待 in-flight 请求完成
	if a.server != nil {
		log.Println("[app] shutting down http server (waiting for in-flight requests)...")
		if err := a.server.Shutdown(ctx); err != nil {
			log.Printf("[app] http shutdown error: %v", err)
		}
	}

	// ② 排空消费：停止投递新消息，等待 in-flight 消息落库完成
	log.Println("[app] shutting down rabbitmq consumer...")
	rabbitmq.ShutdownRabbitMQ(ctx)

	// ③ 释放连接：Redis → MySQL
	log.Println("[app] closing redis...")
	if err := redis.Close(); err != nil {
		log.Printf("[app] redis close error: %v", err)
	}
	log.Println("[app] closing mysql...")
	if err := mysql.Close(); err != nil {
		log.Printf("[app] mysql close error: %v", err)
	}

	log.Println("[app] shutdown complete")
	return nil
}

// readDataFromDB 从数据库加载历史消息并预热 AIHelperManager。
// 原实现位于 main 包，现收口到 App 生命周期内，确保其在 InitRabbitMQ 之后执行。
func readDataFromDB() error {
	manager := aihelper.GetGlobalManager()
	msgs, err := message.GetAllMessages(context.Background())
	if err != nil {
		return err
	}
	for i := range msgs {
		m := &msgs[i]
		modelType := "1" // 默认 openai 模型
		opts, err := aihelper.BuildSessionCreateOptions(modelType, m.UserName, "")
		if err != nil {
			log.Printf("[readDataFromDB] failed to build options for user=%s session=%s: %v", m.UserName, m.SessionID, err)
			continue
		}
		helper, err := manager.GetOrCreateAIHelper(context.Background(), m.UserName, m.SessionID, opts)
		if err != nil {
			log.Printf("[readDataFromDB] failed to create helper for user=%s session=%s: %v", m.UserName, m.SessionID, err)
			continue
		}
		// 添加消息到内存中（不开启存储功能）
		helper.AddMessage(m.Content, m.UserName, m.IsUser, false)
	}
	log.Println("AIHelperManager init success")
	return nil
}
