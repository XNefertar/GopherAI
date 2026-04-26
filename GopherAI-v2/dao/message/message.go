package message

import (
	"GopherAI/common/mysql"
	"GopherAI/model"
	"context"

	"gorm.io/gorm/clause"
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

// CreateMessageIdempotent 基于 OutboxID 做幂等写入。
// 如果同一个 OutboxID 已经存在对应消息，则不再重复插入，
// 从而保证消费者在出现"消息重投 / 崩溃重启 / Worker 重复拾取"等场景时的最终一致性。
func CreateMessageIdempotent(ctx context.Context, message *model.Message) (*model.Message, error) {
	err := mysql.DB.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "outbox_id"}},
			DoNothing: true,
		}).
		Create(message).Error
	return message, err
}

func GetAllMessages(ctx context.Context) ([]model.Message, error) {
	var msgs []model.Message
	err := mysql.DB.WithContext(ctx).Order("created_at asc").Find(&msgs).Error
	return msgs, err
}
