package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/khaicode-xyz/core-shared/kafka"
	"github.com/khaicode-xyz/core-shared/middleware"
	"go.opentelemetry.io/otel/trace"
)

// KafkaEmitter publishes audit events onto a Kafka topic. The topic is bound
// to the supplied producer (one producer == one topic) so this package adds
// no implicit topic name — services pass the env-configured value when
// constructing the producer.
type KafkaEmitter struct {
	producer *kafka.Producer
	service  string
}

// NewKafkaEmitter binds a KafkaEmitter to an already-built kafka.Producer. The
// service argument is the canonical service name stamped onto every event
// when the caller leaves Event.Service blank.
func NewKafkaEmitter(producer *kafka.Producer, service string) *KafkaEmitter {
	return &KafkaEmitter{producer: producer, service: service}
}

// Emit auto-fills correlation fields (event_id, occurred_at, request_id,
// trace_id, schema_version, service) from the context when the caller has
// left them blank, then publishes the event keyed by event_id so Kafka
// partitioning preserves per-event ordering.
func (e *KafkaEmitter) Emit(ctx context.Context, event Event) error {
	e.fill(ctx, &event)
	if err := e.producer.Publish(ctx, event.EventID, event); err != nil {
		return fmt.Errorf("publish audit event: %w", err)
	}
	return nil
}

func (e *KafkaEmitter) fill(ctx context.Context, event *Event) {
	if event.EventID == "" {
		if id, err := uuid.NewV7(); err == nil {
			event.EventID = id.String()
		} else {
			event.EventID = uuid.NewString()
		}
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	if event.Service == "" {
		event.Service = e.service
	}
	if event.SchemaVersion == 0 {
		event.SchemaVersion = SchemaVersion
	}
	if event.RequestID == "" {
		event.RequestID = middleware.GetRequestID(ctx)
	}
	if event.TraceID == "" {
		if span := trace.SpanContextFromContext(ctx); span.IsValid() {
			event.TraceID = span.TraceID().String()
		}
	}
}
