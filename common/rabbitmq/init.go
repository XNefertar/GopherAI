package rabbitmq

import (
	"context"
	"log"
	"sync"
)

const messageConsumerTag = "gopher-ai-message-consumer"

var (
	RMQMessage *RabbitMQ

	// consumerWg 用于在 Shutdown 时等待消费 goroutine 完全退出，
	// 确保 in-flight 消息处理完毕后再关闭连接。
	consumerWg sync.WaitGroup
)

func InitRabbitMQ() {
	// 创建MQ并启动消费者
	// 无论调用多少次 NewWorkRabbitMQ，只会创建一次连接
	// 不同队列共用一个连接，可以保持不同队列消费消息的顺序

	RMQMessage = NewWorkRabbitMQ("Message")
	consumerWg.Add(1)
	go func() {
		defer consumerWg.Done()
		RMQMessage.Consume(messageConsumerTag, MQMessage)
	}()
}

// ShutdownRabbitMQ 优雅停止消费者：先取消投递，待 goroutine 排空后关闭连接。
// 传入的 ctx 控制整体超时，避免关闭过程 hang 死。
func ShutdownRabbitMQ(ctx context.Context) {
	if RMQMessage == nil {
		return
	}

	// 1. 取消消费者：channel 停止向消费 goroutine 投递，for-range 自然退出
	if RMQMessage.channel != nil {
		if err := RMQMessage.channel.Cancel(messageConsumerTag, false); err != nil {
			log.Printf("[rabbitmq] cancel consumer error: %v", err)
		}
	}

	// 2. 等待消费 goroutine 完成 in-flight 消息处理
	done := make(chan struct{})
	go func() {
		consumerWg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		log.Printf("[rabbitmq] shutdown timeout waiting for consumer, forcing close")
	}

	// 3. 关闭连接
	RMQMessage.Destroy()
}
