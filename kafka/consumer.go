package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

// MessageHandler processes a single Kafka message.
type MessageHandler func(ctx context.Context, key string, value []byte) error

// Consumer reads messages from a Kafka topic and dispatches to a handler.
type Consumer struct {
	reader  *kafka.Reader
	handler MessageHandler
	logger  *slog.Logger
}

// NewConsumer creates a Kafka consumer for a specific topic.
func NewConsumer(cfg *Config, topic, groupID string, handler MessageHandler, logger *slog.Logger) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		Topic:    topic,
		GroupID:  groupID,
		Dialer:   cfg.NewDialer(),
		MinBytes: 1,
		MaxBytes: 10e6,
	})

	logger.Info("kafka consumer created",
		slog.String("topic", topic),
		slog.String("group_id", groupID),
	)

	return &Consumer{reader: r, handler: handler, logger: logger}
}

// Start begins consuming messages in a blocking loop. Cancel the context to stop.
func (c *Consumer) Start(ctx context.Context) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil // context cancelled, graceful shutdown
			}
			c.logger.Error("kafka fetch error", slog.String("error", err.Error()))
			continue
		}

		if err := c.handler(ctx, string(msg.Key), msg.Value); err != nil {
			c.logger.Error("kafka message handler error",
				slog.String("key", string(msg.Key)),
				slog.String("error", err.Error()),
			)
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error("kafka commit error", slog.String("error", err.Error()))
		}
	}
}

// Close closes the Kafka reader.
func (c *Consumer) Close() error {
	return c.reader.Close()
}

// ParseMessage is a helper to unmarshal a Kafka message value into a struct.
func ParseMessage[T any](value []byte) (*T, error) {
	var msg T
	if err := json.Unmarshal(value, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal kafka message: %w", err)
	}
	return &msg, nil
}
