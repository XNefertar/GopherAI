package session

import (
	"GopherAI/common/logger"
	"GopherAI/common/mysql"
	"GopherAI/model"
	"context"

	"gorm.io/gorm"
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

func SoftDeleteSession(ctx context.Context, sessionID, userName string) error {
	result := mysql.DB.WithContext(ctx).
		Where("id = ? AND user_name = ?", sessionID, userName).
		Delete(&model.Session{})
	if result.Error != nil {
		logger.With("userName", userName, "sessionID", sessionID).Error("SoftDeleteSession failed", "error", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
