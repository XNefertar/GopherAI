package rabbitmq

import (
	"GopherAI/dao/message"
	"GopherAI/dao/outbox"
	"GopherAI/model"
	"context"
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
)

// MessageMQParam 通过 RabbitMQ 在生产者（outbox worker）和消费者之间传输的消息载荷。
// OutboxID 是幂等键：
//   - 消费者用它作为 message 表的唯一约束，保证"同一条 outbox 消息只会被写库一次"。
//   - 同时也用它回写 outbox 状态，确保 worker 不会无休止地重复投递。
type MessageMQParam struct {
	OutboxID  string `json:"outbox_id"`
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	UserName  string `json:"user_name"`
	IsUser    bool   `json:"is_user"`
}

func GenerateMessageMQParam(outboxID, sessionID, content, userName string, IsUser bool) []byte {
	param := MessageMQParam{
		OutboxID:  outboxID,
		SessionID: sessionID,
		Content:   content,
		UserName:  userName,
		IsUser:    IsUser,
	}
	data, _ := json.Marshal(param)
	return data
}

// MQMessage 消费者处理逻辑：
//  1. 基于 OutboxID 幂等写入 message 表；
//  2. 写入成功后回写 outbox 表状态为 Delivered；
//  3. 任一步失败均返回错误，worker 端的 visibilityTimeout + MarkFailed 会触发重试。
func MQMessage(msg *amqp.Delivery) error {
	var param MessageMQParam
	if err := json.Unmarshal(msg.Body, &param); err != nil {
		return err
	}

	newMsg := &model.Message{
		OutboxID:  param.OutboxID,
		SessionID: param.SessionID,
		Content:   param.Content,
		UserName:  param.UserName,
		IsUser:    param.IsUser,
	}

	if _, err := message.CreateMessageIdempotent(context.Background(), newMsg); err != nil {
		log.Printf("[MQ] idempotent insert message failed, outbox_id=%s err=%v", param.OutboxID, err)
		return err
	}

	if param.OutboxID != "" {
		if err := outbox.MarkDelivered(context.Background(), param.OutboxID); err != nil {
			log.Printf("[MQ] mark outbox delivered failed, outbox_id=%s err=%v", param.OutboxID, err)
			return err
		}
	}
	return nil
}
