package session

import (
	"GopherAI/common/mysql"
	"GopherAI/model"
	"context"
)

func GetSessionsByUserName(ctx context.Context, UserName int64) ([]model.Session, error) {
	var sessions []model.Session
	err := mysql.DB.WithContext(ctx).Where("user_name = ?", UserName).Find(&sessions).Error
	return sessions, err
}

func CreateSession(ctx context.Context, session *model.Session) (*model.Session, error) {
	err := mysql.DB.WithContext(ctx).Create(session).Error
	return session, err
}

func GetSessionByID(ctx context.Context, sessionID string) (*model.Session, error) {
	var session model.Session
	err := mysql.DB.WithContext(ctx).Where("id = ?", sessionID).First(&session).Error
	return &session, err
}
