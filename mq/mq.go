package mq

import (
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/viper"
	"log"
	"time"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

// Send 将数据丢到队列中
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
// "amqp://guest:guest@localhost:5672/"
func Send(ctx context.Context, queueName string, msg string) error {
	amqpUrl := viper.GetString("amqp.url")
	conn, err := amqp.Dial(amqpUrl)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer func(conn *amqp.Connection) {
		err := conn.Close()
		if err != nil {
			failOnError(err, "Failed to declare a queue")
		}
	}(conn)

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer func(ch *amqp.Channel) {
		err := ch.Close()
		if err != nil {
			failOnError(err, "Failed to declare a queue")
		}
	}(ch)

	q, err := ch.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	failOnError(err, "Failed to declare a queue")

	err = ch.PublishWithContext(ctx,
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(msg),
		})
	failOnError(err, "Failed to publish a message")
	log.Printf(" [x] Sent %s\n", msg)
	return nil
}

// MessageHandler 定义一个处理函数类型
type MessageHandler func(string)

func Receive(ctx context.Context, queueName string, handler MessageHandler) {
	amqpUrl := viper.GetString("amqp.url")
	conn, err := amqp.Dial(amqpUrl)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer func(conn *amqp.Connection) {
		err := conn.Close()
		if err != nil {
			failOnError(err, "Failed to close connection")
		}
	}(conn)

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer func(ch *amqp.Channel) {
		err := ch.Close()
		if err != nil {
			failOnError(err, "Failed to close channel")
		}
	}(ch)

	q, err := ch.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	failOnError(err, "Failed to declare a queue")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	var forever chan struct{}

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			// 调用处理函数
			handler(string(d.Body))
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func Process(msg string) {
	// 在这里处理接收到的消息
	log.Printf("Processing message: %s", msg)
}
