package message

import (
	"GopherAI/common/logger"
	"GopherAI/common/mysql"
	"GopherAI/model"
	"context"
)

func GetMessagesBySessionID(ctx context.Context, sessionID string) ([]model.Message, error) {
	var msgs []model.Message
	err := mysql.DB.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at asc").Find(&msgs).Error
	return msgs, err
}

func GetMessagesBySessionIDs(ctx context.Context, sessionIDs []string) ([]model.Message, error) {
	var msgs []model.Message
	if len(sessionIDs) == 0 {
		return msgs, nil
	}
	err := mysql.DB.WithContext(ctx).Where("session_id IN ?", sessionIDs).Order("created_at asc").Find(&msgs).Error
	return msgs, err
}

func CreateMessage(ctx context.Context, message *model.Message) (*model.Message, error) {
	err := mysql.DB.WithContext(ctx).Create(message).Error
	return message, err
}

func GetAllMessages(ctx context.Context) ([]model.Message, error) {
	var msgs []model.Message
	err := mysql.DB.WithContext(ctx).Order("created_at asc").Find(&msgs).Error
	return msgs, err
}

func HardDeleteMessageBySessionID(ctx context.Context, sessionID string) (int64, error) {
	result := mysql.DB.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Delete(&model.Message{})
	if result.Error != nil {
		logger.With("sessionID", sessionID).Error("HardDeleteMessagesBySessionID failed", "error", result.Error)
	}
	return result.RowsAffected, result.Error
}
