package session

import (
	"GopherAI/common/aihelper"
	"GopherAI/common/code"
	"GopherAI/common/logger"
	"GopherAI/common/titlesummary"
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

func GetUserSessionsByUserName(ctx context.Context, userName string) ([]model.SessionInfo, error) {
	// 从数据库读取会话以获取准确的标题
	sessions, err := sessionDao.GetSessionsByUserName(ctx, userName)
	if err != nil {
		log.Printf("GetUserSessionsByUserName db error: %v", err)
		// fallback: 从内存管理器获取 session ID
		manager := aihelper.GetGlobalManager()
		ids := manager.GetUserSessions(userName)
		infos := make([]model.SessionInfo, 0, len(ids))
		for _, sid := range ids {
			infos = append(infos, model.SessionInfo{
				SessionID: sid,
				Title:     sid, // fallback
			})
		}
		return infos, nil
	}

	infos := make([]model.SessionInfo, 0, len(sessions))
	for _, s := range sessions {
		title := s.Title
		if title == "" {
			title = s.ID
		}
		infos = append(infos, model.SessionInfo{
			SessionID: s.ID,
			Title:     title,
		})
	}
	return infos, nil
}

func CreateSessionAndSendMessage(ctx context.Context, userName, kbID, userQuestion, modelType string) (string, string, string, code.Code) {
	//1：创建一个新的会话
	newSession := &model.Session{
		ID:         uuid.New().String(),
		UserName:   userName,
		Title:      userQuestion, // 先用原问题占位，后续并发更新为 AI 生成的标题
		ActiveKBID: kbID,
	}
	createdSession, err := session.CreateSession(ctx, newSession)
	if err != nil {
		log.Println("CreateSessionAndSendMessage CreateSession error:", err)
		return "", "", "", code.CodeServerBusy
	}

	// 并发：GLM 生成标题 & AI 生成回复，取 max 耗时
	type titleResult struct{ title string }
	titleCh := make(chan titleResult, 1)
	go func() {
		t := generateSessionTitleSync(createdSession.ID, userQuestion)
		titleCh <- titleResult{title: t}
	}()

	//2：获取AIHelper并通过其管理消息
	manager := aihelper.GetGlobalManager()

	var aiResponse *model.Message
	var aiErr error

	// modelType=auto 时走混合路由器：根据问题特征自动选择具体模型，实现成本优化。
	if modelType == aihelper.ModelTypeAuto {
		helper, _, aerr := manager.GetOrCreateAIHelperWithAutoRoute(ctx, userName, createdSession.ID, userQuestion, false)
		if aerr != nil {
			log.Println("CreateSessionAndSendMessage auto route error:", aerr)
			return "", "", "", code.AIModelFail
		}
		aiResponse, aiErr = helper.GenerateResponse(ctx, userName, userQuestion)
	} else {
		opts, optsErr := aihelper.BuildSessionCreateOptions(modelType, userName, createdSession.ActiveKBID)
		if optsErr != nil {
			log.Println("CreateSessionAndSendMessage BuildSessionCreateOptions error:", optsErr)
			return "", "", "", code.AIModelFail
		}
		helper, oerr := manager.GetOrCreateAIHelper(ctx, userName, createdSession.ID, opts)
		if oerr != nil {
			log.Println("CreateSessionAndSendMessage GetOrCreateAIHelper error:", oerr)
			return "", "", "", code.AIModelFail
		}
		aiResponse, aiErr = helper.GenerateResponse(ctx, userName, userQuestion)
	}

	if aiErr != nil {
		log.Println("CreateSessionAndSendMessage GenerateResponse error:", aiErr)
		return "", "", "", code.AIModelFail
	}

	// 等待 GLM 标题生成结果（通常比 AI 回复快得多）
	title := <-titleCh
	if title.title != "" {
		_ = sessionDao.UpdateSessionTitle(context.Background(), createdSession.ID, title.title)
		return createdSession.ID, title.title, aiResponse.Content, code.CodeSuccess
	}

	return createdSession.ID, userQuestion, aiResponse.Content, code.CodeSuccess
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

	// 异步生成精简标题
	go generateSessionTitle(createdSession.ID, userQuestion)

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
	for i, msg := range messages {
		isUser := i%2 == 0
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

// generateSessionTitle 异步调用 GLM-4-Flash 生成精简标题并更新数据库（供流式接口使用）
func generateSessionTitle(sessionID, userQuestion string) {
	title := titlesummary.GenerateTitle(context.Background(), userQuestion)
	if title == "" {
		return
	}

	if err := sessionDao.UpdateSessionTitle(context.Background(), sessionID, title); err != nil {
		log.Printf("generateSessionTitle update failed: session=%s err=%v", sessionID, err)
	} else {
		log.Printf("generateSessionTitle success: session=%s title=%q", sessionID, title)
	}
}

// generateSessionTitleSync 同步调用 GLM-4-Flash 生成精简标题（仅生成，不写 DB）
func generateSessionTitleSync(sessionID, userQuestion string) string {
	title := titlesummary.GenerateTitle(context.Background(), userQuestion)
	if title == "" {
		return ""
	}
	log.Printf("generateSessionTitleSync: session=%s title=%q", sessionID, title)
	return title
}
