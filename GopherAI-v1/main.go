package main

import (
	"GopherAI/common/aihelper"
	"GopherAI/common/mysql"
	"GopherAI/common/rabbitmq"
	"GopherAI/common/redis"
	"GopherAI/config"
	"GopherAI/dao/message"
	"GopherAI/router"
	"fmt"
	"log"
	"os"
	"strconv"
)

func benchMode() string {
	return os.Getenv("BENCH_MODE")
}

func isBenchMode() bool {
	switch benchMode() {
	case "session-list", "http-baseline":
		return true
	default:
		return false
	}
}

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

func isHTTPBaselineMode() bool {
	return os.Getenv("BENCH_MODE") == "http-baseline"
}

func StartServer(addr string, port int) error {
	r := router.InitRouter()
	//服务器静态资源路径映射关系，这里目前不需要
	// r.Static(config.GetConfig().HttpFilePath, config.GetConfig().MusicFilePath)
	return r.Run(fmt.Sprintf("%s:%d", addr, port))
}

// 从数据库加载消息并初始化 AIHelperManager
func readDataFromDB() error {
	manager := aihelper.GetGlobalManager()
	// 从数据库读取所有消息
	msgs, err := message.GetAllMessages()
	if err != nil {
		return err
	}
	// 遍历数据库消息
	for i := range msgs {
		m := &msgs[i]
		//默认openai模型
		modelType := "1"
		config := make(map[string]interface{})

		// 创建对应的 AIHelper
		helper, err := manager.GetOrCreateAIHelper(m.UserName, m.SessionID, modelType, config)
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
