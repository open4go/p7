package kf

import (
	"context"
	"fmt"
	"github.com/open4go/log"
	"github.com/segmentio/kafka-go"
	"sync"
	"time"
)

// WriterManager 管理多个 topic 对应的 Kafka Writer
type WriterManager struct {
	writers map[string]*kafka.Writer
	lock    sync.RWMutex
	brokers []string
}

// 全局实例
var (
	manager     *WriterManager
	managerOnce sync.Once
)

// InitWriterManager 初始化 Writer 管理器
func InitWriterManager(ctx context.Context, brokers []string) {
	managerOnce.Do(func() {
		manager = &WriterManager{
			writers: make(map[string]*kafka.Writer),
			brokers: brokers,
		}
		log.Log(ctx).WithField("brokers", brokers).
			Info("[Kafka] WriterManager initialized")
	})
}

// getWriter 获取或创建指定 topic 的 writer
func (m *WriterManager) getWriter(ctx context.Context, topic string) *kafka.Writer {
	m.lock.RLock()
	w, ok := m.writers[topic]
	m.lock.RUnlock()
	if ok {
		return w
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	// 双重检查
	if w, ok = m.writers[topic]; ok {
		return w
	}

	w = &kafka.Writer{
		Addr:         kafka.TCP(m.brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
		BatchTimeout: 10 * time.Millisecond,
	}
	m.writers[topic] = w

	log.Log(ctx).WithField("topic", topic).
		Info("[Kafka] Writer created for topic")

	return w
}

// SendString 向指定 topic 发送字符串消息
func SendString(ctx context.Context, topic, message string) error {
	if manager == nil {
		return fmt.Errorf("Kafka WriterManager not initialized")
	}

	w := manager.getWriter(ctx, topic)
	return w.WriteMessages(ctx, kafka.Message{
		Value: []byte(message),
	})
}

// SendBytes 向指定 topic 发送二进制消息
func SendBytes(ctx context.Context, topic string, data []byte) error {
	if manager == nil {
		return fmt.Errorf("Kafka WriterManager not initialized")
	}

	w := manager.getWriter(ctx, topic)
	return w.WriteMessages(ctx, kafka.Message{
		Value: data,
	})
}

// CloseAll 关闭所有 writer
func CloseAll() {
	if manager == nil {
		return
	}

	manager.lock.Lock()
	defer manager.lock.Unlock()

	for topic, w := range manager.writers {
		_ = w.Close()
		delete(manager.writers, topic)
	}
}
