package mq

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/viper"
)

func TestSendRequiresURL(t *testing.T) {
	viper.Reset()
	viper.Set("amqp.url", "")
	err := Send(context.Background(), "q", "hi")
	if !errors.Is(err, ErrAMQPNotConfigured) {
		t.Fatalf("want ErrAMQPNotConfigured, got %v", err)
	}
}

func TestReceiveRequiresURL(t *testing.T) {
	viper.Reset()
	viper.Set("amqp.url", "")
	err := Receive(context.Background(), "q", nil)
	if !errors.Is(err, ErrAMQPNotConfigured) {
		t.Fatalf("want ErrAMQPNotConfigured, got %v", err)
	}
}
