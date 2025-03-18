package p7

import (
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/viper"
	"log"
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
