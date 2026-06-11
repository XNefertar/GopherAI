package session

import (
	"GopherAI/common/aihelper"
	"GopherAI/common/code"
	"GopherAI/common/logger"
	"GopherAI/dao/session"
	"GopherAI/model"
	"context"
	"net/http"

	"github.com/google/uuid"
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

<<<<<<< Updated upstream
func CreateSessionAndSendMessage(ctx context.Context, userName, kbID, userQuestion, modelType string) (string, string, code.Code) {
	//1：创建一个新的会话
	newSession := &model.Session{
		ID:         uuid.New().String(),
		UserName:   userName,
		Title:      userQuestion, // 可以根据需求设置标题，这边暂时用用户第一次的问题作为标题
		ActiveKBID: kbID,
=======
<<<<<<< Updated upstream
func CreateSessionAndSendMessage(ctx context.Context, userName string, userQuestion string, modelType string) (string, string, code.Code) {
	//1：创建一个新的会话
	newSession := &model.Session{
		ID:       uuid.New().String(),
		UserName: userName,
		Title:    userQuestion, // 可以根据需求设置标题，这边暂时用用户第一次的问题作为标题
=======
func CreateSessionAndSendMessage(ctx context.Context, userName, kbID, userQuestion, modelType string) (string, string, code.Code) {
	sessionID := uuid.New().String()
	l := logger.With("userName", userName, "kbID", kbID, "sessionID", sessionID)
	//1：创建一个新的会话
	newSession := &model.Session{
		ID:         sessionID,
		UserName:   userName,
		Title:      userQuestion, // 可以根据需求设置标题，这边暂时用用户第一次的问题作为标题
		ActiveKBID: kbID,
>>>>>>> Stashed changes
>>>>>>> Stashed changes
	}
	createdSession, err := session.CreateSession(ctx, newSession)
	if err != nil {
		l.Error("CreateSession failed", "error", err)
		return "", "", code.CodeServerBusy
	}

	//2：获取AIHelper并通过其管理消息
	manager := aihelper.GetGlobalManager()

	// modelType=auto 时走混合路由器：根据问题特征自动选择具体模型，实现成本优化。
	if modelType == aihelper.ModelTypeAuto {
		helper, _, err := manager.GetOrCreateAIHelperWithAutoRoute(ctx, userName, createdSession.ID, userQuestion, false)
		if err != nil {
			l.Error("auto route failed", "error", err)
			return "", "", code.AIModelFail
		}
		aiResponse, err_ := helper.GenerateResponse(ctx, userName, userQuestion)
		if err_ != nil {
			l.Error("GenerateResponse failed", "error", err_)
			return "", "", code.AIModelFail
		}
		return createdSession.ID, aiResponse.Content, code.CodeSuccess
	}

	opts, err := aihelper.BuildSessionCreateOptions(modelType, userName, createdSession.ActiveKBID)
	if err != nil {
		l.Error("BuildSessionCreateOptions failed", "error", err)
		return "", "", code.AIModelFail
	}
	helper, err := manager.GetOrCreateAIHelper(ctx, userName, createdSession.ID, opts)
	if err != nil {
		l.Error("GetOrCreateAIHelper failed", "error", err)
		return "", "", code.AIModelFail
	}

	//3：生成AI回复
	aiResponse, err_ := helper.GenerateResponse(ctx, userName, userQuestion)
	if err_ != nil {
		l.Error("GenerateResponse failed", "error", err_)
		return "", "", code.AIModelFail
	}

	return createdSession.ID, aiResponse.Content, code.CodeSuccess
}

<<<<<<< Updated upstream
func CreateStreamSessionOnly(ctx context.Context, userName, kbID, userQuestion string) (string, code.Code) {
	newSession := &model.Session{
		ID:         uuid.New().String(),
		UserName:   userName,
		Title:      userQuestion,
		ActiveKBID: kbID,
=======
<<<<<<< Updated upstream
func CreateStreamSessionOnly(ctx context.Context, userName string, userQuestion string) (string, code.Code) {
	newSession := &model.Session{
		ID:       uuid.New().String(),
		UserName: userName,
		Title:    userQuestion,
=======
func CreateStreamSessionOnly(ctx context.Context, userName, kbID, userQuestion string) (string, code.Code) {
	sessionID := uuid.New().String()
	l := logger.With("userName", userName, "kbID", kbID, "sessionID", sessionID)
	newSession := &model.Session{
		ID:         sessionID,
		UserName:   userName,
		Title:      userQuestion,
		ActiveKBID: kbID,
>>>>>>> Stashed changes
>>>>>>> Stashed changes
	}
	createdSession, err := session.CreateSession(ctx, newSession)
	if err != nil {
		l.Error("CreateStreamSessionOnly failed", "error", err)
		return "", code.CodeServerBusy
	}
	return createdSession.ID, code.CodeSuccess
}

func StreamMessageToExistingSession(ctx context.Context, userName string, sessionID string, userQuestion string, modelType string, writer http.ResponseWriter) code.Code {
	l := logger.With("userName", userName, "sessionID", sessionID)
	// 确保 writer 支持 Flush
	flusher, ok := writer.(http.Flusher)
	if !ok {
		l.Error("streaming unsupported")
		return code.CodeServerBusy
	}

	manager := aihelper.GetGlobalManager()
<<<<<<< Updated upstream
	sessionObj, getSessionErr := session.GetSessionByID(ctx, sessionID)
	if getSessionErr != nil {
		log.Println("StreamMessageToExistingSession GetSessionByID error: ", getSessionErr)
		return code.CodeServerBusy
	}
=======
<<<<<<< Updated upstream
=======
	sessionObj, getSessionErr := session.GetSessionByID(ctx, sessionID)
	if getSessionErr != nil {
		l.Error("GetSessionByID failed", "error", getSessionErr)
		return code.CodeServerBusy
	}
	l = l.With("kbID", sessionObj.ActiveKBID)
>>>>>>> Stashed changes
>>>>>>> Stashed changes

	var helper *aihelper.AIHelper
	if modelType == aihelper.ModelTypeAuto {
		// modelType=auto 时走混合路由器
		var err error
		helper, _, err = manager.GetOrCreateAIHelperWithAutoRoute(ctx, userName, sessionID, userQuestion, true)
		if err != nil {
			l.Error("auto route failed", "error", err)
			return code.AIModelFail
		}
	} else {
		opts, err := aihelper.BuildSessionCreateOptions(modelType, userName, sessionObj.ActiveKBID)
		if err != nil {
			l.Error("BuildSessionCreateOptions failed", "error", err)
			return code.AIModelFail
		}
		helper, err = manager.GetOrCreateAIHelper(ctx, userName, sessionID, opts)
		if err != nil {
			l.Error("GetOrCreateAIHelper failed", "error", err)
			return code.AIModelFail
		}
	}

	cb := func(msg string) {
		// 直接发送数据，不转义
		// SSE 格式：data: <content>\n\n
		l.Debug("SSE chunk sent", "len", len(msg))
		_, err := writer.Write([]byte("data: " + msg + "\n\n"))
		if err != nil {
			l.Error("SSE write failed", "error", err)
			return
		}
		flusher.Flush()
	}

	_, err_ := helper.StreamResponse(ctx, userName, cb, userQuestion)
	if err_ != nil {
		l.Error("StreamResponse failed", "error", err_)
		return code.AIModelFail
	}

	_, err := writer.Write([]byte("data: [DONE]\n\n"))
	if err != nil {
		l.Error("write DONE failed", "error", err)
		return code.AIModelFail
	}
	flusher.Flush()

	return code.CodeSuccess
}

func ChatSend(ctx context.Context, userName string, sessionID string, userQuestion string, modelType string) (string, code.Code) {
	l := logger.With("userName", userName, "sessionID", sessionID)
	//1：获取AIHelper
	manager := aihelper.GetGlobalManager()
<<<<<<< Updated upstream
	sessionObj, getSessionErr := session.GetSessionByID(ctx, sessionID)
	if getSessionErr != nil {
		log.Println("StreamMessageToExistingSession GetSessionByID error: ", getSessionErr)
		return "", code.CodeServerBusy
	}
=======
<<<<<<< Updated upstream
=======
	sessionObj, getSessionErr := session.GetSessionByID(ctx, sessionID)
	if getSessionErr != nil {
		l.Error("GetSessionByID failed", "error", getSessionErr)
		return "", code.CodeServerBusy
	}
	l = l.With("kbID", sessionObj.ActiveKBID)
>>>>>>> Stashed changes
>>>>>>> Stashed changes

	var helper *aihelper.AIHelper
	if modelType == aihelper.ModelTypeAuto {
		// modelType=auto 时走混合路由器
		var err error
		helper, _, err = manager.GetOrCreateAIHelperWithAutoRoute(ctx, userName, sessionID, userQuestion, false)
		if err != nil {
			l.Error("auto route failed", "error", err)
			return "", code.AIModelFail
		}
	} else {
		opts, err := aihelper.BuildSessionCreateOptions(modelType, userName, sessionObj.ActiveKBID)
		if err != nil {
			l.Error("BuildSessionCreateOptions failed", "error", err)
			return "", code.AIModelFail
		}
		helper, err = manager.GetOrCreateAIHelper(ctx, userName, sessionID, opts)
		if err != nil {
			l.Error("GetOrCreateAIHelper failed", "error", err)
			return "", code.AIModelFail
		}
	}

	//2：生成AI回复
	aiResponse, err_ := helper.GenerateResponse(ctx, userName, userQuestion)
	if err_ != nil {
		l.Error("GenerateResponse failed", "error", err_)
		return "", code.AIModelFail
	}

	return aiResponse.Content, code.CodeSuccess
}

func GetChatHistory(userName string, sessionID string) ([]model.History, code.Code) {
	// 获取AIHelper中的消息历史
	manager := aihelper.GetGlobalManager()
	helper, exists := manager.GetAIHelper(userName, sessionID)
	if !exists {
		logger.With("userName", userName, "sessionID", sessionID).Warn("AIHelper not found for GetChatHistory")
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
