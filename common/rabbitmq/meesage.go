package rabbitmq

import (
	"GopherAI/dao/message"
	"GopherAI/model"
	"context"
	"encoding/json"

	"github.com/streadway/amqp"
)

type MessageMQParam struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	UserName  string `json:"user_name"`
	IsUser    bool   `json:"is_user"`
}

func GenerateMessageMQParam(sessionID string, content string, userName string, IsUser bool) []byte {
	param := MessageMQParam{
		SessionID: sessionID,
		Content:   content,
		UserName:  userName,
		IsUser:    IsUser,
	}
	data, _ := json.Marshal(param)
	return data
}

// OnMessagePersisted 是消息落库成功后的回灌钩子，由 aihelper 在初始化时注册。
// 用于让会话淘汰前的 Flush 精确判断哪些消息已落库，从而避免与 MQ 消费者重复写。
// 声明在 rabbitmq 包内，避免 aihelper → rabbitmq → aihelper 的循环依赖。
var OnMessagePersisted func(userName, sessionID string)

func MQMessage(msg *amqp.Delivery) error {
	var param MessageMQParam
	err := json.Unmarshal(msg.Body, &param)
	if err != nil {
		return err
	}
	newMsg := &model.Message{
		SessionID: param.SessionID,
		Content:   param.Content,
		UserName:  param.UserName,
		IsUser:    param.IsUser,
	}
	//消费者异步插入到数据库中
	if _, err := message.CreateMessage(context.Background(), newMsg); err != nil {
		return err
	}
	// 落库成功后回灌：标记该会话对应消息已持久化
	if OnMessagePersisted != nil {
		OnMessagePersisted(param.UserName, param.SessionID)
	}
	return nil
}
