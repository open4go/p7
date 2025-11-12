package kf

import (
	"fmt"
	"github.com/open4go/log"
	"github.com/segmentio/kafka-go"
	"golang.org/x/net/context"
	"sync"
	"time"
)

// 全局 writer 单例
var (
	writer *kafka.Writer
	once   sync.Once
)

// InitWriter 初始化全局 Kafka Writer
func InitWriter(ctx context.Context, brokers []string, topic string) {
	once.Do(func() {
		writer = &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireAll,
			Async:        false,
			BatchTimeout: 10 * time.Millisecond,
		}
		log.Log(ctx).WithField("topic", topic).
			Info("[Kafka] Writer initialized for topic")
	})
}

// Close 关闭 writer
func Close() error {
	if writer != nil {
		return writer.Close()
	}
	return nil
}

// SendString 发送字符串消息
func SendString(ctx context.Context, message string) error {
	if writer == nil {
		return fmt.Errorf("kafka writer not initialized")
	}

	return writer.WriteMessages(ctx, kafka.Message{
		Value: []byte(message),
	})
}

// SendBytes 发送二进制消息
func SendBytes(ctx context.Context, data []byte) error {
	if writer == nil {
		return fmt.Errorf("kafka writer not initialized")
	}

	return writer.WriteMessages(ctx, kafka.Message{
		Value: data,
	})
}
