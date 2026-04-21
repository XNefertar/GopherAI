package rabbitmq

import (
	"GopherAI/config"
	"fmt"
	"log"
	"strconv"

	"github.com/streadway/amqp"
)

const (
	retryCountHeader  = "x-retry-count"
	defaultMaxRetries = 3
	defaultRetryTTLMS = 5000
)

// 全局connection对象
// 所有RabbitMQ都会复用该对象
var conn *amqp.Connection

// 初始化connection
func initConn() {
	c := config.GetConfig()
	mqUrl := fmt.Sprintf(
		"amqp://%s:%s@%s:%d/%s",
		c.RabbitmqUsername, c.RabbitmqPassword, c.RabbitmqHost, c.RabbitmqPort, c.RabbitmqVhost,
	)
	log.Println("mqUrl is  " + mqUrl)
	var err error
	conn, err = amqp.Dial(mqUrl)
	if err != nil {
		log.Fatalf("RabbitMQ connection failed: %v", err) // 输出错误并退出程序
	}
}

// RabbitMQ RabbitMQ结构体
type RabbitMQ struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	Exchange string
	Key      string
}

// NewRabbitMQ 创建RabbitMQ对象
func NewRabbitMQ(exchange string, key string) *RabbitMQ {
	return &RabbitMQ{Exchange: exchange, Key: key}
}

// Destroy 断开 channel 和 connection
func (r *RabbitMQ) Destroy() {
	_ = r.channel.Close()
	_ = r.conn.Close()
}

// NewWorkRabbitMQ 创建Work模式的RabbitMQ实例
func NewWorkRabbitMQ(queue string) *RabbitMQ {
	// new rabbitmq
	rabbitmq := NewRabbitMQ("", queue)

	// get connection
	if conn == nil {
		initConn()
	}
	rabbitmq.conn = conn

	// get channel
	var err error
	rabbitmq.channel, err = rabbitmq.conn.Channel()
	if err != nil {
		panic(err.Error())
	}

	return rabbitmq
}

func (r *RabbitMQ) retryQueueName() string {
	return r.Key + ".retry"
}

func (r *RabbitMQ) deadQueueName() string {
	return r.Key + ".dead"
}

func (r *RabbitMQ) declareMainQueue() (amqp.Queue, error) {
	return r.channel.QueueDeclare(r.Key, false, false, false, false, nil)
}

func (r *RabbitMQ) declareRetryQueue() error {
	_, err := r.channel.QueueDeclare(
		r.retryQueueName(),
		false,
		false,
		false,
		false,
		amqp.Table{
			"x-message-ttl":             int32(defaultRetryTTLMS),
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": r.Key,
		},
	)
	return err
}

func (r *RabbitMQ) declareDeadQueue() error {
	_, err := r.channel.QueueDeclare(r.deadQueueName(), false, false, false, false, nil)
	return err
}

func (r *RabbitMQ) declareConsumerTopology() (amqp.Queue, error) {
	q, err := r.declareMainQueue()
	if err != nil {
		return q, err
	}
	if err := r.declareRetryQueue(); err != nil {
		return q, err
	}
	if err := r.declareDeadQueue(); err != nil {
		return q, err
	}
	return q, nil
}

func cloneHeaders(headers amqp.Table) amqp.Table {
	if headers == nil {
		return amqp.Table{}
	}
	cloned := make(amqp.Table, len(headers))
	for k, v := range headers {
		cloned[k] = v
	}
	return cloned
}

func parseRetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}
	raw, ok := headers[retryCountHeader]
	if !ok {
		return 0
	}

	switch v := raw.(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case string:
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return 0
}

func (r *RabbitMQ) publishWithHeaders(queueName string, delivery amqp.Delivery, headers amqp.Table) error {
	contentType := delivery.ContentType
	if contentType == "" {
		contentType = "text/plain"
	}

	return r.channel.Publish("", queueName, false, false, amqp.Publishing{
		ContentType:  contentType,
		Body:         delivery.Body,
		Headers:      headers,
		DeliveryMode: amqp.Persistent,
	})
}

func (r *RabbitMQ) handleConsumeFailure(msg *amqp.Delivery, handleErr error) error {
	retryCount := parseRetryCount(msg.Headers)
	headers := cloneHeaders(msg.Headers)

	if retryCount >= defaultMaxRetries {
		headers["x-final-error"] = handleErr.Error()
		if err := r.publishWithHeaders(r.deadQueueName(), *msg, headers); err != nil {
			return err
		}
		log.Printf("message moved to dead queue after %d retries: queue=%s err=%v", retryCount, r.deadQueueName(), handleErr)
		return nil
	}

	headers[retryCountHeader] = retryCount + 1
	if err := r.publishWithHeaders(r.retryQueueName(), *msg, headers); err != nil {
		return err
	}

	log.Printf("message scheduled for retry #%d: queue=%s err=%v", retryCount+1, r.retryQueueName(), handleErr)
	return nil
}

// Publish 发送消息
func (r *RabbitMQ) Publish(message []byte) error {
	// 创建队列（不存在时）
	// 使用默认交换机的情况下，queue即为key
	_, err := r.declareMainQueue()
	if err != nil {
		return err
	}

	// 调用 channel 发送消息到队列
	return r.channel.Publish(r.Exchange, r.Key, false, false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        message,
		},
	)
}

// Consume 消费者
// handle: 消息的消费业务函数，用于消费消息
func (r *RabbitMQ) Consume(handle func(msg *amqp.Delivery) error) {
	// 创建主队列、重试队列和死信队列
	q, err := r.declareConsumerTopology()
	if err != nil {
		panic(err)
	}

	// 接收消息
	msgs, err := r.channel.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	// 处理消息
	for msg := range msgs {
		if err := handle(&msg); err != nil {
			log.Printf("consumer handle failed: %v", err)
			if retryErr := r.handleConsumeFailure(&msg, err); retryErr != nil {
				log.Printf("consumer handoff failed, nack with requeue: %v", retryErr)
				if nackErr := msg.Nack(false, true); nackErr != nil {
					log.Printf("consumer nack failed: %v", nackErr)
				}
				continue
			}
			if ackErr := msg.Ack(false); ackErr != nil {
				log.Printf("consumer ack after failure handling failed: %v", ackErr)
			}
			continue
		}

		if ackErr := msg.Ack(false); ackErr != nil {
			log.Printf("consumer ack failed: %v", ackErr)
		}
	}
}
