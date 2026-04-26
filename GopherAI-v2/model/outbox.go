package model

import "time"

// MessageOutbox 事务外盒表（Outbox Pattern）
//
// 用途：
//  1. 业务侧产生聊天消息时，先通过事务写入本表，而不是直接发 MQ；
//  2. 后台 worker 周期性扫描未投递的记录并发送到 RabbitMQ；
//  3. 消费者幂等写入 message 表后再回写 outbox 状态。
//
// 通过这种方式保证：即使应用或 MQ broker 崩溃重启，
// 只要 outbox 表里的记录还在，消息最终都能被可靠投递与消费。
type MessageOutbox struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	OutboxID  string    `gorm:"uniqueIndex;type:varchar(64);not null" json:"outbox_id"` // 业务幂等键
	SessionID string    `gorm:"index;type:varchar(36);not null" json:"session_id"`
	UserName  string    `gorm:"type:varchar(20);not null" json:"user_name"`
	Content   string    `gorm:"type:text" json:"content"`
	IsUser    bool      `gorm:"not null" json:"is_user"`
	Status    int       `gorm:"index;not null;default:0" json:"status"` // 0 Pending, 1 Sending, 2 Delivered, 3 Failed
	Retries   int       `gorm:"not null;default:0" json:"retries"`      // 重试次数
	LastError string    `gorm:"type:varchar(512)" json:"last_error"`    // 最近一次失败原因
	NextRunAt time.Time `gorm:"index" json:"next_run_at"`               // 下次可被 worker 拾取的时间
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

const (
	OutboxStatusPending   = 0
	OutboxStatusSending   = 1
	OutboxStatusDelivered = 2
	OutboxStatusFailed    = 3
)
