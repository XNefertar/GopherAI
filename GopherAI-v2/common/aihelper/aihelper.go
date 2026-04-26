package aihelper

import (
	"GopherAI/dao/outbox"
	"GopherAI/model"
	"GopherAI/utils"
	"context"
	"log"
	"sync"
	"time"
)

// AIHelper AI助手结构体，包含消息历史和AI模型
type AIHelper struct {
	model    AIModel
	messages []*model.Message
	mu       sync.RWMutex
	//一个会话绑定一个AIHelper
	SessionID string
	saveFunc  func(*model.Message) (*model.Message, error)
}

// NewAIHelper 创建新的AIHelper实例
func NewAIHelper(aiModel AIModel, SessionID string) *AIHelper {
	return &AIHelper{
		model:    aiModel,
		messages: make([]*model.Message, 0),
		// 采用 Outbox Pattern：
		// 主链路不再直接把消息发送到 RabbitMQ，而是先把消息事务性地写入 outbox 表，
		// 由后台 worker 负责可靠投递到 MQ，消费者再幂等写入 message 表。
		// 即便应用或 MQ broker 崩溃，只要 outbox 记录存在，消息最终都能被重新投递。
		saveFunc: func(msg *model.Message) (*model.Message, error) {
			record := &model.MessageOutbox{
				OutboxID:  utils.GenerateUUID(),
				SessionID: msg.SessionID,
				UserName:  msg.UserName,
				Content:   msg.Content,
				IsUser:    msg.IsUser,
				Status:    model.OutboxStatusPending,
				NextRunAt: time.Now(),
			}
			if err := outbox.CreateOutbox(context.Background(), record); err != nil {
				log.Printf("[Outbox] enqueue failed, session=%s user=%s err=%v",
					msg.SessionID, msg.UserName, err)
				return msg, err
			}
			msg.OutboxID = record.OutboxID
			return msg, nil
		},
		SessionID: SessionID,
	}
}

// addMessage 添加消息到内存中并调用自定义存储函数
func (a *AIHelper) AddMessage(Content string, UserName string, IsUser bool, Save bool) {
	userMsg := model.Message{
		SessionID: a.SessionID,
		Content:   Content,
		UserName:  UserName,
		IsUser:    IsUser,
	}
	a.mu.Lock()
	a.messages = append(a.messages, &userMsg)
	a.mu.Unlock()
	if Save {
		a.saveFunc(&userMsg)
	}
}

// SaveMessage 保存消息到数据库（通过回调函数避免循环依赖）
// 通过传入func，自己调用外部的保存函数，即可支持同步异步等多种策略
func (a *AIHelper) SetSaveFunc(saveFunc func(*model.Message) (*model.Message, error)) {
	a.saveFunc = saveFunc
}

// GetMessages 获取所有消息历史
func (a *AIHelper) GetMessages() []*model.Message {
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := make([]*model.Message, len(a.messages))
	copy(out, a.messages)
	return out
}

// 同步生成
func (a *AIHelper) GenerateResponse(ctx context.Context, userName string, userQuestion string) (*model.Message, error) {

	//调用存储函数
	a.AddMessage(userQuestion, userName, true, true)

	a.mu.RLock()
	//将model.Message转化成schema.Message
	messages := utils.ConvertToSchemaMessages(a.messages)
	currentModel := a.model
	a.mu.RUnlock()

	//调用模型生成回复
	schemaMsg, err := currentModel.GenerateResponse(ctx, messages)
	if err != nil {
		return nil, err
	}

	//将schema.Message转化成model.Message
	modelMsg := utils.ConvertToModelMessage(a.SessionID, userName, schemaMsg)

	//调用存储函数
	a.AddMessage(modelMsg.Content, userName, false, true)

	return modelMsg, nil
}

// 流式生成
func (a *AIHelper) StreamResponse(ctx context.Context, userName string, cb StreamCallback, userQuestion string) (*model.Message, error) {

	//调用存储函数
	a.AddMessage(userQuestion, userName, true, true)

	a.mu.RLock()
	messages := utils.ConvertToSchemaMessages(a.messages)
	currentModel := a.model
	a.mu.RUnlock()

	content, err := currentModel.StreamResponse(ctx, messages, cb)
	if err != nil {
		return nil, err
	}
	//转化成model.Message
	modelMsg := &model.Message{
		SessionID: a.SessionID,
		UserName:  userName,
		Content:   content,
		IsUser:    false,
	}

	//调用存储函数
	a.AddMessage(modelMsg.Content, userName, false, true)

	return modelMsg, nil
}

// GetModelType 获取模型类型
func (a *AIHelper) GetModelType() string {
	return a.model.GetModelType()
}

func (a *AIHelper) SwitchModel(newModel AIModel) {
	a.mu.Lock()
	a.model = newModel
	a.mu.Unlock()
}
