package outbox

import (
	"GopherAI/common/mysql"
	"GopherAI/model"
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateOutbox 在业务侧把消息先写入 outbox 表，等待后台 worker 投递。
// 使用 OnConflict DoNothing 保证即使重复提交同一个 OutboxID 也不会产生重复记录。
func CreateOutbox(ctx context.Context, record *model.MessageOutbox) error {
	return mysql.DB.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(record).Error
}

// PickPending 以"批量抢占"方式取出一批待投递记录。
// 1. 只取 Pending 或到期的 Sending 记录（处理 worker 异常退出后遗留的中间态）；
// 2. 通过事务 + FOR UPDATE 保证单机或多实例下不会重复处理同一条记录；
// 3. 命中后立刻将其标记为 Sending 并推后 NextRunAt，作为软锁。
func PickPending(ctx context.Context, batch int, visibilityTimeout time.Duration) ([]model.MessageOutbox, error) {
	var picked []model.MessageOutbox
	now := time.Now()

	err := mysql.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status IN ? AND next_run_at <= ?",
				[]int{model.OutboxStatusPending, model.OutboxStatusSending},
				now,
			).
			Order("id ASC").
			Limit(batch).
			Find(&picked).Error; err != nil {
			return err
		}
		if len(picked) == 0 {
			return nil
		}

		ids := make([]uint, 0, len(picked))
		for _, r := range picked {
			ids = append(ids, r.ID)
		}

		// 提前占位：把这批记录置为 Sending 并延长下次可被拾取的时间
		return tx.Model(&model.MessageOutbox{}).
			Where("id IN ?", ids).
			Updates(map[string]interface{}{
				"status":      model.OutboxStatusSending,
				"next_run_at": now.Add(visibilityTimeout),
			}).Error
	})
	return picked, err
}

// MarkDelivered 消息已确认被消费者处理完成（写入 message 表），更新状态为 Delivered。
func MarkDelivered(ctx context.Context, outboxID string) error {
	return mysql.DB.WithContext(ctx).
		Model(&model.MessageOutbox{}).
		Where("outbox_id = ?", outboxID).
		Updates(map[string]interface{}{
			"status":     model.OutboxStatusDelivered,
			"last_error": "",
		}).Error
}

// MarkFailed 投递失败，重置为 Pending 并记录重试次数；
// 达到最大重试次数后置为 Failed，由人工介入或独立告警流程处理。
func MarkFailed(ctx context.Context, id uint, reason string, maxRetries int, backoff time.Duration) error {
	var record model.MessageOutbox
	if err := mysql.DB.WithContext(ctx).First(&record, id).Error; err != nil {
		return err
	}

	record.Retries++
	record.LastError = truncate(reason, 500)
	record.NextRunAt = time.Now().Add(backoff)
	if record.Retries >= maxRetries {
		record.Status = model.OutboxStatusFailed
	} else {
		record.Status = model.OutboxStatusPending
	}

	return mysql.DB.WithContext(ctx).
		Model(&model.MessageOutbox{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      record.Status,
			"retries":     record.Retries,
			"last_error":  record.LastError,
			"next_run_at": record.NextRunAt,
		}).Error
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
