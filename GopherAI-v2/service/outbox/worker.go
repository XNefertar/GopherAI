package outbox

import (
	"GopherAI/common/rabbitmq"
	"GopherAI/dao/outbox"
	"context"
	"log"
	"time"
)

// Worker 负责将 outbox 表中的待投递消息可靠地发送到 RabbitMQ。
//
// 工作流程：
//  1. 周期性从 outbox 表批量拾取 Pending / 过期 Sending 的记录；
//  2. 对每条记录发送到 MQ；
//  3. 成功发送后等待消费者回写 Delivered，失败则通过 MarkFailed 做退避重试。
//
// 典型的故障恢复场景：
//   - 应用崩溃：进程重启后，worker 会重新扫到 Pending 记录继续投递；
//   - MQ broker 重启：只要 outbox 记录未 Delivered，worker 会持续重发；
//   - 消费者异常：visibilityTimeout 过期后记录会被重新拾取，配合 OutboxID 幂等避免重复落库。
type Worker struct {
	BatchSize         int
	Interval          time.Duration
	VisibilityTimeout time.Duration
	MaxRetries        int
	Backoff           time.Duration
	stopCh            chan struct{}
}

// NewDefaultWorker 提供一组对中小规模流量安全的默认参数。
// 大流量场景应通过配置覆盖，并考虑多实例部署。
func NewDefaultWorker() *Worker {
	return &Worker{
		BatchSize:         50,
		Interval:          500 * time.Millisecond,
		VisibilityTimeout: 30 * time.Second,
		MaxRetries:        5,
		Backoff:           5 * time.Second,
		stopCh:            make(chan struct{}),
	}
}

// Start 在独立 goroutine 中启动扫描循环。
func (w *Worker) Start() {
	go w.loop()
	log.Println("[Outbox] worker started")
}

// Stop 发出停止信号，等到下一轮循环退出。
func (w *Worker) Stop() {
	close(w.stopCh)
}

func (w *Worker) loop() {
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			log.Println("[Outbox] worker stopped")
			return
		case <-ticker.C:
			w.tick()
		}
	}
}

func (w *Worker) tick() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	records, err := outbox.PickPending(ctx, w.BatchSize, w.VisibilityTimeout)
	if err != nil {
		log.Printf("[Outbox] pick pending error: %v", err)
		return
	}
	if len(records) == 0 {
		return
	}

	for i := range records {
		r := &records[i]
		body := rabbitmq.GenerateMessageMQParam(r.OutboxID, r.SessionID, r.Content, r.UserName, r.IsUser)

		if rabbitmq.RMQMessage == nil {
			// MQ 尚未初始化完毕，保留 Pending 状态等待下一轮
			_ = outbox.MarkFailed(ctx, r.ID, "rabbitmq not ready", w.MaxRetries, w.Backoff)
			continue
		}

		if err := rabbitmq.RMQMessage.Publish(body); err != nil {
			log.Printf("[Outbox] publish failed, outbox_id=%s err=%v", r.OutboxID, err)
			_ = outbox.MarkFailed(ctx, r.ID, err.Error(), w.MaxRetries, w.Backoff)
			continue
		}
		// 投递成功后仍保持 Sending 状态，等待消费者回写 Delivered；
		// 若消费者在 visibilityTimeout 内未完成消费，记录会被重新拾取并再次投递。
	}
}
