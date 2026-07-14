package rabbitmq

import (
	"GopherAI/config"
	"fmt"
	"log"

	"github.com/streadway/amqp"
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

// Publish 发送消息
func (r *RabbitMQ) Publish(message []byte) error {
	// 创建队列（不存在时）
	// 使用默认交换机的情况下，queue即为key
	_, err := r.channel.QueueDeclare(r.Key, false, false, false, false, nil)
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

// Consume 消费者（可被 Cancel 取消）。
// handle: 消息的消费业务函数，用于消费消息；返回 nil 时自动 Ack，非 nil 时暂不重投（避免重复落库）。
// consumerTag: 消费者标识，用于 Shutdown 时精确取消该消费者。
func (r *RabbitMQ) Consume(consumerTag string, handle func(msg *amqp.Delivery) error) {
	// 创建队列
	q, err := r.channel.QueueDeclare(r.Key, false, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	// 接收消息：autoAck=false，由业务成功后再显式 Ack，避免消费中进程退出导致消息丢失
	msgs, err := r.channel.Consume(q.Name, consumerTag, false, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	// 处理消息
	for msg := range msgs {
		if err := handle(&msg); err != nil {
			// 业务失败：仅打印，不 Ack 也不 Nack（保留在队列，待后续可靠性改造接入 DLX）
			fmt.Println(err.Error())
			continue
		}
		_ = msg.Ack(false)
	}
}
