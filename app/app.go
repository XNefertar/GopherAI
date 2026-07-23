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
	"GopherAI/router"
)

// shutdownTimeout 是优雅停机各阶段的总超时上限。
// 超过该时间后，即使仍有 in-flight 请求/消息也会强制退出，避免进程 hang 死。
const shutdownTimeout = 30 * time.Second

// App 负责应用的依赖装配与生命周期管理。
//
// 设计动机：
//   - 原 main.go 把「初始化顺序」「HTTP 启动」「信号处理」全部手写揉在一起，
//     既无法优雅停机，也存在 readDataFromDB 全量预热导致的启动慢/内存膨胀。
//   - App 把「装配（Init）→ 运行（Run）→ 优雅停机（Shutdown）」三段式生命周期
//     收敛为单一入口，关闭顺序与依赖顺序严格逆向对应，体现应用骨架（Bootstrap）的分层设计。
//   - 会话历史不再启动时全量预热，改为首次访问时按需惰性加载（见 AIHelper.Hydrate），
//     启动复杂度从 O(全表消息) 降为 O(1)。
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

	// 1.5 会话缓存容量治理接线：
	//   - 把 MQ 落库成功的回灌通道接到 manager，让淘汰前的 Flush 精确判断已落库消息，避免重复写；
	//   - 启动后台空闲回收 sweeper（按 idleTimeout 淘汰长时间无访问的会话）。
	rabbitmq.OnMessagePersisted = func(userName, sessionID string) {
		aihelper.GetGlobalManager().MarkPersisted(userName, sessionID)
	}
	aihelper.GetGlobalManager().Start()

	// 1.6 主动初始化全局混合路由器（LLM 语义分类器），
	//     提前创建分类器 LLM 客户端并在启动日志中暴露创建结果。
	aihelper.InitGlobalRouter(context.Background())

	// 2. 启动 HTTP 服务（在 goroutine 中运行，便于主协程阻塞等待信号）
	//    注：会话历史不再全量预热，改为首次访问时按需惰性加载（见 AIHelper.Hydrate）
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

	// ②.5 落库兜底：先停 sweeper，再把内存中尚未被 MQ 消费者写回 DB 的消息直接 Flush，
	//      避免停机丢上下文（已脱离 map 的会话由 sweeper 自行 Flush，不会重复写）。
	aihelper.GetGlobalManager().Stop()
	aihelper.GetGlobalManager().FlushAll(ctx)

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
