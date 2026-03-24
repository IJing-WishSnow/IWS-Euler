package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/segmentio/kafka-go"
)

const OrdersTopic = "orders"

var KafkaBroker = getEnv("KAFKA_BROKER", "localhost:9094")

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Producer Kafka 生产者
type Producer struct {
	writer *kafka.Writer
}

func NewProducer(topic string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(KafkaBroker),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

// Send 发送消息到 Kafka
func (p *Producer) Send(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: data,
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

