package main

import (
	"GopherAI/common/aihelper"
	"GopherAI/common/logger"
	"GopherAI/common/mysql"
	"GopherAI/common/rabbitmq"
	"GopherAI/common/redis"
	"GopherAI/config"
	"GopherAI/dao/message"
	"GopherAI/router"
	"context"
	"fmt"
)

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
	msgs, err := message.GetAllMessages(context.Background())
	if err != nil {
		return err
	}
	// 遍历数据库消息
	for i := range msgs {
		m := &msgs[i]
		l := logger.With("userName", m.UserName, "sessionID", m.SessionID)
		//默认openai模型
		modelType := "1"
		opts, err := aihelper.BuildSessionCreateOptions(modelType, m.UserName, "")
		if err != nil {
			l.Error("failed to build options for readDataFromDB", "error", err)
			continue
		}

		// 创建对应的 AIHelper
		helper, err := manager.GetOrCreateAIHelper(context.Background(), m.UserName, m.SessionID, opts)
		if err != nil {
			l.Error("failed to create helper for readDataFromDB", "error", err)
			continue
		}
		// 添加消息到内存中(不开启存储功能)
		helper.AddMessage(m.Content, m.UserName, m.IsUser, false)
	}

	logger.Info("AIHelperManager init success")
	return nil
}

func main() {
	conf := config.GetConfig()
	host := conf.MainConfig.Host
	port := conf.MainConfig.Port
	//初始化mysql
	if err := mysql.InitMysql(); err != nil {
		logger.Error("InitMysql failed", "error", err)
		return
	}
	//初始化AIHelperManager
	readDataFromDB()

	//初始化redis
	redis.Init()
	logger.Info("redis init success")
	//初始化rabbitmq
	rabbitmq.InitRabbitMQ()
	logger.Info("rabbitmq init success")

	err := StartServer(host, port) // 启动 HTTP 服务
	if err != nil {
		panic(err)
	}
}
