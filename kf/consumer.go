package kf

import (
	"context"
	"github.com/open4go/log"
	"github.com/segmentio/kafka-go"
	"sync"
)

var (
	reader     *kafka.Reader
	readerOnce sync.Once
)

// InitReader 初始化全局 Kafka Reader
func InitReader(ctx context.Context, brokers []string, topic, groupID string) {
	readerOnce.Do(func() {
		reader = kafka.NewReader(kafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		})
		log.Log(ctx).WithField("topic", topic).WithField("groupID", groupID).
			Info("[Kafka] Reader initialized for topic")
	})
}

// CloseReader 关闭 Reader
func CloseReader() error {
	if reader != nil {
		return reader.Close()
	}
	return nil
}

// ReadMessage 手动读取一条消息（同步阻塞）
func ReadMessage(ctx context.Context) ([]byte, error) {
	if reader == nil {
		return nil, ErrReaderNotInit
	}
	msg, err := reader.ReadMessage(ctx)
	if err != nil {
		return nil, err
	}
	log.Log(ctx).WithField("payload", msg.Value).Debug("[Kafka] Consumed message")
	return msg.Value, nil
}

// ConsumeLoop 启动一个后台协程持续消费消息
// handler 函数用于处理每条消息
func ConsumeLoop(ctx context.Context, handler func([]byte)) {
	if reader == nil {
		log.Log(ctx).
			Info("[Kafka] Reader not initialized, cannot start consumer loop")
		return
	}

	go func() {
		for {
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				log.Log(ctx).Error(err)
				continue
			}
			handler(msg.Value)
		}
	}()
}

var ErrReaderNotInit = &ReaderNotInitError{}

type ReaderNotInitError struct{}

func (e *ReaderNotInitError) Error() string {
	return "kafka reader not initialized"
}
