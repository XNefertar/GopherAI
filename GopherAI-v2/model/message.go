package model

import (
	"time"
)

type Message struct {
	ID        uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	SessionID string `gorm:"index;not null;type:varchar(36)" json:"session_id"`
	UserName  string `gorm:"type:varchar(20)" json:"username"`
	Content   string `gorm:"type:text" json:"content"`
	IsUser    bool   `gorm:"not null;" json:"is_user"`
	// OutboxID 业务幂等键，来源于 MessageOutbox.OutboxID。
	// 消费者通过它保证同一条消息只会被落库一次。
	OutboxID  string    `gorm:"uniqueIndex;type:varchar(64)" json:"outbox_id"`
	CreatedAt time.Time `json:"created_at"`
}

type History struct {
	IsUser  bool   `json:"is_user"`
	Content string `json:"content"`
}
