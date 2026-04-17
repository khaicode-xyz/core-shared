package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer wraps kafka-go writer for publishing messages.
type Producer struct {
	writer *kafka.Writer
	logger *slog.Logger
}

// NewProducer creates a Kafka producer for a specific topic.
func NewProducer(cfg *Config, topic string, logger *slog.Logger) *Producer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
		Transport:    cfg.Transport(),
	}

	logger.Info("kafka producer created",
		slog.String("topic", topic),
	)

	return &Producer{writer: w, logger: logger}
}

// Publish sends a message to Kafka with the given key and value.
func (p *Producer) Publish(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal kafka message: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(key),
		Value: data,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("publish to kafka: %w", err)
	}

	p.logger.Debug("kafka message published",
		slog.String("key", key),
		slog.Int("size", len(data)),
	)
	return nil
}

// Close closes the Kafka writer.
func (p *Producer) Close() error {
	return p.writer.Close()
}
