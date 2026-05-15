package main

import (
	"GopherAI/common/aihelper"
	"GopherAI/common/mysql"
	"GopherAI/common/rabbitmq"
	"GopherAI/common/redis"
	"GopherAI/config"
	"GopherAI/dao/message"
	"GopherAI/router"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
)

func StartServer(addr string, port int) error {
	r := router.InitRouter()
	//服务器静态资源路径映射关系，这里目前不需要
	// r.Static(config.GetConfig().HttpFilePath, config.GetConfig().MusicFilePath)
	return r.Run(fmt.Sprintf("%s:%d", addr, port))
}

// benchMode 返回当前压测模式标识，空字符串代表正常业务模式。
func benchMode() string {
	return os.Getenv("BENCH_MODE")
}

// isBenchMode 判断当前是否处于已识别的压测模式。
//   - session-list：用于会话列表内存压测，不依赖外部中间件
//   - http-baseline：用于纯 HTTP 基线压测，启动精简版 Gin
func isBenchMode() bool {
	switch benchMode() {
	case "session-list", "http-baseline":
		return true
	default:
		return false
	}
}

// seedBenchSessionsIfNeeded 仅在 session-list 模式下生效。
// 它会基于环境变量 BENCH_USER / BENCH_SESSIONS 在内存中预生成会话，
// 用于压测“按用户取会话列表”的纯内存路径。
func seedBenchSessionsIfNeeded() {
	if benchMode() != "session-list" {
		return
	}

	userName := os.Getenv("BENCH_USER")
	if userName == "" {
		userName = "bench-user"
	}

	countStr := os.Getenv("BENCH_SESSIONS")
	if countStr == "" {
		countStr = "100"
	}
	count, err := strconv.Atoi(countStr)
	if err != nil || count < 0 {
		log.Printf("invalid BENCH_SESSIONS=%q, fallback to 100", countStr)
		count = 100
	}

	manager := aihelper.GetGlobalManager()
	manager.SeedUserSessions(userName, count)
	log.Printf(
		"seeded benchmark sessions: user=%s requested=%d actual=%d",
		userName,
		count,
		manager.CountUserSessions(userName),
	)
}

// 从数据库加载消息并初始化 AIHelperManager
func readDataFromDB() error {
	manager := aihelper.GetGlobalManager()
	// 从数据库读取所有消息
	msgs, err := message.GetAllMessages(context.Background())
	if err != nil {
		return err
	}
	// 遍历数据库消息
	for i := range msgs {
		m := &msgs[i]
		//默认openai模型
		modelType := "1"
		opts, err := aihelper.BuildSessionCreateOptions(modelType, m.UserName)
		if err != nil {
			log.Printf("[readDataFromDB] failed to build options for user=%s session=%s: %v", m.UserName, m.SessionID, err)
			continue
		}

		// 创建对应的 AIHelper
		helper, err := manager.GetOrCreateAIHelper(context.Background(), m.UserName, m.SessionID, opts)
		if err != nil {
			log.Printf("[readDataFromDB] failed to create helper for user=%s session=%s: %v", m.UserName, m.SessionID, err)
			continue
		}
		log.Println("readDataFromDB init:  ", helper.SessionID)
		// 添加消息到内存中(不开启存储功能)
		helper.AddMessage(m.Content, m.UserName, m.IsUser, false)
	}

	log.Println("AIHelperManager init success ")
	return nil
}

func main() {
	conf := config.GetConfig()
	host := conf.MainConfig.Host
	port := conf.MainConfig.Port

	if isBenchMode() {
		// 压测模式只做必要的内存初始化，不依赖 MySQL/Redis/RabbitMQ，
		// 这样可以让 HTTP 基线和会话列表压测更纯粹、可重复。
		log.Printf("running in benchmark mode: %s", benchMode())
		seedBenchSessionsIfNeeded()
	} else {
		//初始化mysql
		if err := mysql.InitMysql(); err != nil {
			log.Println("InitMysql error , " + err.Error())
			return
		}
		//初始化AIHelperManager
		readDataFromDB()

		//初始化redis
		redis.Init()
		log.Println("redis init success  ")
		//初始化rabbitmq
		rabbitmq.InitRabbitMQ()
		log.Println("rabbitmq init success  ")
	}

	err := StartServer(host, port) // 启动 HTTP 服务
	if err != nil {
		panic(err)
	}
}
