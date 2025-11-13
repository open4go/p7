package kf

import (
	"context"
	"errors"
	"sync"

	"github.com/open4go/log"
	"github.com/segmentio/kafka-go"
)

var (
	readers   = make(map[string]*kafka.Reader)
	readersMu sync.Mutex
)

// InitReader 初始化指定 topic 的 Reader
func InitReader(ctx context.Context, brokers []string, topic, groupID string) error {
	readersMu.Lock()
	defer readersMu.Unlock()

	if _, exists := readers[topic]; exists {
		log.Log(ctx).WithField("topic", topic).Info("[Kafka] Reader already initialized")
		return nil
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: groupID,
	})

	readers[topic] = r
	log.Log(ctx).WithField("topic", topic).WithField("groupID", groupID).
		Info("[Kafka] Reader initialized for topic")

	return nil
}

// GetReader 获取指定 topic 的 Reader
func GetReader(topic string) (*kafka.Reader, error) {
	readersMu.Lock()
	defer readersMu.Unlock()

	r, exists := readers[topic]
	if !exists {
		return nil, ErrReaderNotInit
	}
	return r, nil
}

// CloseReader 关闭指定 topic 的 Reader
func CloseReader(topic string) error {
	readersMu.Lock()
	defer readersMu.Unlock()

	if r, ok := readers[topic]; ok {
		delete(readers, topic)
		return r.Close()
	}
	return nil
}

// CloseAllReaders 关闭所有 Reader
func CloseAllReaders() {
	readersMu.Lock()
	defer readersMu.Unlock()

	for topic, r := range readers {
		_ = r.Close()
		delete(readers, topic)
	}
}

// ConsumeLoop 启动一个 Topic 的消费循环
func ConsumeLoop(ctx context.Context, topic string, handler func([]byte)) error {
	r, err := GetReader(topic)
	if err != nil {
		return err
	}

	go func() {
		for {
			msg, err := r.ReadMessage(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					log.Log(ctx).WithField("topic", topic).Info("[Kafka] ConsumeLoop stopped")
					return
				}
				log.Log(ctx).WithField("topic", topic).Error(err)
				continue
			}
			handler(msg.Value)
		}
	}()

	return nil
}

var ErrReaderNotInit = errors.New("kafka reader not initialized")
