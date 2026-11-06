package mq

import (
	"context"
	"errors"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/spf13/viper"
)

// ErrAMQPNotConfigured is returned when amqp.url is empty.
var ErrAMQPNotConfigured = errors.New("amqp.url not configured")

func amqpURL() (string, error) {
	url := viper.GetString("amqp.url")
	if url == "" {
		return "", ErrAMQPNotConfigured
	}
	return url, nil
}

// Send publishes msg to queueName. Connection / channel / publish failures
// return an error; they never panic (S-08 / order-task stability).
func Send(ctx context.Context, queueName string, msg string) error {
	amqpUrl, err := amqpURL()
	if err != nil {
		return err
	}
	conn, err := amqp.Dial(amqpUrl)
	if err != nil {
		return fmt.Errorf("Failed to connect to RabbitMQ: %w", err)
	}
	defer func() { _ = conn.Close() }()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("Failed to open a channel: %w", err)
	}
	defer func() { _ = ch.Close() }()

	q, err := ch.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("Failed to declare a queue: %w", err)
	}

	if err = ch.PublishWithContext(ctx,
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(msg),
		}); err != nil {
		return fmt.Errorf("Failed to publish a message: %w", err)
	}
	log.Printf(" [x] Sent %s\n", msg)
	return nil
}

// MessageHandler is the consumer callback.
type MessageHandler func(string)

// Receive consumes queueName forever. Dial / channel failures return an error
// instead of panicking so the caller (e.g. device-api) can stay up.
func Receive(ctx context.Context, queueName string, handler MessageHandler) error {
	amqpUrl, err := amqpURL()
	if err != nil {
		return err
	}
	conn, err := amqp.Dial(amqpUrl)
	if err != nil {
		return fmt.Errorf("Failed to connect to RabbitMQ: %w", err)
	}
	defer func() { _ = conn.Close() }()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("Failed to open a channel: %w", err)
	}
	defer func() { _ = ch.Close() }()

	q, err := ch.QueueDeclare(
		queueName,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("Failed to declare a queue: %w", err)
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("Failed to register a consumer: %w", err)
	}

	log.Printf(" [*] Waiting for messages on %s", queueName)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-msgs:
			if !ok {
				return errors.New("amqp consumer channel closed")
			}
			if handler != nil {
				handler(string(d.Body))
			}
		}
	}
}

// Process is a no-op sample handler kept for API compatibility.
func Process(msg string) {
	log.Printf("Processing message: %s", msg)
}
