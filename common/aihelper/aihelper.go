package aihelper

import (
	"GopherAI/common/rabbitmq"
	"GopherAI/dao/message"
	"GopherAI/model"
	"GopherAI/utils"
	"context"
	"sync"
)

// AIHelper AI助手结构体，包含消息历史和AI模型
type AIHelper struct {
	model    AIModel
	messages []*model.Message
	mu       sync.RWMutex
	//一个会话绑定一个AIHelper
	SessionID string
	saveFunc  func(*model.Message) (*model.Message, error)
	hydrated  bool // 标记历史是否已从 DB 惰性加载，避免重复加载
}

// NewAIHelper 创建新的AIHelper实例
func NewAIHelper(aiModel AIModel, SessionID string) *AIHelper {
	return &AIHelper{
		model:    aiModel,
		messages: make([]*model.Message, 0),
		//异步推送到消息队列中
		saveFunc: func(msg *model.Message) (*model.Message, error) {
			data := rabbitmq.GenerateMessageMQParam(msg.SessionID, msg.Content, msg.UserName, msg.IsUser)
			err := rabbitmq.RMQMessage.Publish(data)
			return msg, err
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

// Hydrate 惰性从 DB 加载会话历史到内存。
//
// 设计要点：
//   - 仅放入内存、不回写 MQ（直接 append 而非 AddMessage），避免历史消息被重复发布到消息队列落库。
//   - 使用 hydrated 标记 + double-check，保证并发下只加载一次。
//   - 本方法在 AIHelperManager 的锁之外调用，因此只操作本 helper 自身的 mu，不持有 manager 锁，
//     避免 DB 查询阻塞其他会话的并发创建。
func (a *AIHelper) Hydrate(ctx context.Context) error {
	a.mu.Lock()
	if a.hydrated {
		a.mu.Unlock()
		return nil
	}
	a.mu.Unlock()

	msgs, err := message.GetMessagesBySessionID(ctx, a.SessionID)
	if err != nil {
		return err
	}

	a.mu.Lock()
	if a.hydrated { // double-check，防止并发重复加载
		a.mu.Unlock()
		return nil
	}
	for i := range msgs {
		m := &msgs[i]
		a.messages = append(a.messages, m)
	}
	a.hydrated = true
	a.mu.Unlock()
	return nil
}
