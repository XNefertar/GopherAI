package session

import (
	"GopherAI/common/aihelper"
	"GopherAI/common/code"
	"GopherAI/common/logger"
	messageDao "GopherAI/dao/message"
	"GopherAI/dao/session"
	sessionDao "GopherAI/dao/session"
	"GopherAI/model"
	"context"
	"log"
	"net/http"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func GetUserSessionsByUserName(userName string) ([]model.SessionInfo, error) {
	//获取用户的所有会话ID

	manager := aihelper.GetGlobalManager()
	Sessions := manager.GetUserSessions(userName)

	var SessionInfos []model.SessionInfo

	for _, session := range Sessions {
		SessionInfos = append(SessionInfos, model.SessionInfo{
			SessionID: session,
			Title:     session, // 暂时用sessionID作为标题，后续重构需要的时候可以更改
		})
	}

	return SessionInfos, nil
}

func CreateSessionAndSendMessage(ctx context.Context, userName, kbID, userQuestion, modelType string) (string, string, code.Code) {
	//1：创建一个新的会话
	newSession := &model.Session{
		ID:         uuid.New().String(),
		UserName:   userName,
		Title:      userQuestion, // 可以根据需求设置标题，这边暂时用用户第一次的问题作为标题
		ActiveKBID: kbID,
	}
	createdSession, err := session.CreateSession(ctx, newSession)
	if err != nil {
		log.Println("CreateSessionAndSendMessage CreateSession error:", err)
		return "", "", code.CodeServerBusy
	}

	//2：获取AIHelper并通过其管理消息
	manager := aihelper.GetGlobalManager()

	// modelType=auto 时走混合路由器：根据问题特征自动选择具体模型，实现成本优化。
	if modelType == aihelper.ModelTypeAuto {
		helper, _, err := manager.GetOrCreateAIHelperWithAutoRoute(ctx, userName, createdSession.ID, userQuestion, false)
		if err != nil {
			log.Println("CreateSessionAndSendMessage auto route error:", err)
			return "", "", code.AIModelFail
		}
		aiResponse, err_ := helper.GenerateResponse(ctx, userName, userQuestion)
		if err_ != nil {
			log.Println("CreateSessionAndSendMessage GenerateResponse error:", err_)
			return "", "", code.AIModelFail
		}
		return createdSession.ID, aiResponse.Content, code.CodeSuccess
	}

	opts, err := aihelper.BuildSessionCreateOptions(modelType, userName, createdSession.ActiveKBID)
	if err != nil {
		log.Println("CreateSessionAndSendMessage BuildSessionCreateOptions error:", err)
		return "", "", code.AIModelFail
	}
	helper, err := manager.GetOrCreateAIHelper(ctx, userName, createdSession.ID, opts)
	if err != nil {
		log.Println("CreateSessionAndSendMessage GetOrCreateAIHelper error:", err)
		return "", "", code.AIModelFail
	}

	//3：生成AI回复
	aiResponse, err_ := helper.GenerateResponse(ctx, userName, userQuestion)
	if err_ != nil {
		log.Println("CreateSessionAndSendMessage GenerateResponse error:", err_)
		return "", "", code.AIModelFail
	}

	return createdSession.ID, aiResponse.Content, code.CodeSuccess
}

func CreateStreamSessionOnly(ctx context.Context, userName, kbID, userQuestion string) (string, code.Code) {
	newSession := &model.Session{
		ID:         uuid.New().String(),
		UserName:   userName,
		Title:      userQuestion,
		ActiveKBID: kbID,
	}
	createdSession, err := session.CreateSession(ctx, newSession)
	if err != nil {
		log.Println("CreateStreamSessionOnly CreateSession error:", err)
		return "", code.CodeServerBusy
	}
	return createdSession.ID, code.CodeSuccess
}

func StreamMessageToExistingSession(ctx context.Context, userName string, sessionID string, userQuestion string, modelType string, writer http.ResponseWriter) code.Code {
	// 确保 writer 支持 Flush
	flusher, ok := writer.(http.Flusher)
	if !ok {
		log.Println("StreamMessageToExistingSession: streaming unsupported")
		return code.CodeServerBusy
	}

	manager := aihelper.GetGlobalManager()
	sessionObj, getSessionErr := session.GetSessionByID(ctx, sessionID)
	if getSessionErr != nil {
		log.Println("StreamMessageToExistingSession GetSessionByID error: ", getSessionErr)
		return code.CodeServerBusy
	}

	var helper *aihelper.AIHelper
	if modelType == aihelper.ModelTypeAuto {
		// modelType=auto 时走混合路由器
		var err error
		helper, _, err = manager.GetOrCreateAIHelperWithAutoRoute(ctx, userName, sessionID, userQuestion, true)
		if err != nil {
			log.Println("StreamMessageToExistingSession auto route error:", err)
			return code.AIModelFail
		}
	} else {
		opts, err := aihelper.BuildSessionCreateOptions(modelType, userName, sessionObj.ActiveKBID)
		if err != nil {
			log.Println("StreamMessageToExistingSession BuildSessionCreateOptions error:", err)
			return code.AIModelFail
		}
		helper, err = manager.GetOrCreateAIHelper(ctx, userName, sessionID, opts)
		if err != nil {
			log.Println("StreamMessageToExistingSession GetOrCreateAIHelper error:", err)
			return code.AIModelFail
		}
	}

	cb := func(msg string) {
		// 直接发送数据，不转义
		// SSE 格式：data: <content>\n\n
		log.Printf("[SSE] Sending chunk: %s (len=%d)\n", msg, len(msg))
		_, err := writer.Write([]byte("data: " + msg + "\n\n"))
		if err != nil {
			log.Println("[SSE] Write error:", err)
			return
		}
		flusher.Flush() //  每次必须 flush
		log.Println("[SSE] Flushed")
	}

	_, err_ := helper.StreamResponse(ctx, userName, cb, userQuestion)
	if err_ != nil {
		log.Println("StreamMessageToExistingSession StreamResponse error:", err_)
		return code.AIModelFail
	}

	_, err := writer.Write([]byte("data: [DONE]\n\n"))
	if err != nil {
		log.Println("StreamMessageToExistingSession write DONE error:", err)
		return code.AIModelFail
	}
	flusher.Flush()

	return code.CodeSuccess
}

func ChatSend(ctx context.Context, userName string, sessionID string, userQuestion string, modelType string) (string, code.Code) {
	//1：获取AIHelper
	manager := aihelper.GetGlobalManager()
	sessionObj, getSessionErr := session.GetSessionByID(ctx, sessionID)
	if getSessionErr != nil {
		log.Println("StreamMessageToExistingSession GetSessionByID error: ", getSessionErr)
		return "", code.CodeServerBusy
	}

	var helper *aihelper.AIHelper
	if modelType == aihelper.ModelTypeAuto {
		// modelType=auto 时走混合路由器
		var err error
		helper, _, err = manager.GetOrCreateAIHelperWithAutoRoute(ctx, userName, sessionID, userQuestion, false)
		if err != nil {
			log.Println("ChatSend auto route error:", err)
			return "", code.AIModelFail
		}
	} else {
		opts, err := aihelper.BuildSessionCreateOptions(modelType, userName, sessionObj.ActiveKBID)
		if err != nil {
			log.Println("ChatSend BuildSessionCreateOptions error:", err)
			return "", code.AIModelFail
		}
		helper, err = manager.GetOrCreateAIHelper(ctx, userName, sessionID, opts)
		if err != nil {
			log.Println("ChatSend GetOrCreateAIHelper error:", err)
			return "", code.AIModelFail
		}
	}

	//2：生成AI回复
	aiResponse, err_ := helper.GenerateResponse(ctx, userName, userQuestion)
	if err_ != nil {
		log.Println("ChatSend GenerateResponse error:", err_)
		return "", code.AIModelFail
	}

	return aiResponse.Content, code.CodeSuccess
}

func GetChatHistory(userName string, sessionID string) ([]model.History, code.Code) {
	// 获取AIHelper中的消息历史
	manager := aihelper.GetGlobalManager()
	helper, exists := manager.GetAIHelper(userName, sessionID)
	if !exists {
		return nil, code.CodeServerBusy
	}

	messages := helper.GetMessages()
	history := make([]model.History, 0, len(messages))

	// 转换消息为历史格式（根据消息顺序或内容判断用户/AI消息）
	for _, msg := range messages {
		isUser := msg.IsUser
		history = append(history, model.History{
			IsUser:  isUser,
			Content: msg.Content,
		})
	}

	return history, code.CodeSuccess
}

func ChatStreamSend(ctx context.Context, userName string, sessionID string, userQuestion string, modelType string, writer http.ResponseWriter) code.Code {
	return StreamMessageToExistingSession(ctx, userName, sessionID, userQuestion, modelType, writer)
}

func DeleteSession(ctx context.Context, userName, sessionID string) code.Code {
	log := logger.With("userName", userName, "sessionID", sessionID)

	if err := sessionDao.SoftDeleteSession(ctx, sessionID, userName); err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Warn("session not found or not owned by user")
			return code.CodeSessionNotExist
		}
		log.Error("SoftDeleteSession failed", "error", err)
		return code.CodeServerBusy
	}

	if deleted, delErr := messageDao.HardDeleteMessageBySessionID(ctx, sessionID); delErr != nil {
		log.Warn("delete messages failed", "error", delErr)
	} else {
		log.Info("session deleted", "deletedMessages", deleted)
	}

	manager := aihelper.GetGlobalManager()
	manager.RemoveAIHelper(userName, sessionID)

	log.Info("session deleted")
	return code.CodeSuccess
}
