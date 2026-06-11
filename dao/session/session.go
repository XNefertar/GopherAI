package session

import (
	"GopherAI/common/logger"
	"GopherAI/common/mysql"
	"GopherAI/model"
	"context"
)

func GetSessionsByUserName(ctx context.Context, UserName int64) ([]model.Session, error) {
	var sessions []model.Session
	err := mysql.DB.WithContext(ctx).Where("user_name = ?", UserName).Find(&sessions).Error
	if err != nil {
		logger.Error("GetSessionsByUserName failed", "userName", UserName, "error", err)
	}
	return sessions, err
}

func CreateSession(ctx context.Context, session *model.Session) (*model.Session, error) {
	err := mysql.DB.WithContext(ctx).Create(session).Error
	if err != nil {
		logger.With("userName", session.UserName, "sessionID", session.ID, "kbID", session.ActiveKBID).Error("CreateSession failed", "error", err)
	}
	return session, err
}

func GetSessionByID(ctx context.Context, sessionID string) (*model.Session, error) {
	var session model.Session
	err := mysql.DB.WithContext(ctx).Where("id = ?", sessionID).First(&session).Error
	if err != nil {
		logger.With("sessionID", sessionID).Error("GetSessionByID failed", "error", err)
	}
	return &session, err
}
