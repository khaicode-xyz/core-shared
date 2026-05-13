package audit

import "context"

// Emitter publishes audit events. Implementations must be safe for concurrent
// use. The contract is fire-and-forget from the caller's perspective: an
// Emit failure must never abort the business operation that produced the
// event — callers log the error and continue.
type Emitter interface {
	Emit(ctx context.Context, event Event) error
}

// NoopEmitter discards every event. Use in unit tests or when audit is
// intentionally disabled (e.g. before the audit topic is provisioned).
type NoopEmitter struct{}

func (NoopEmitter) Emit(context.Context, Event) error { return nil }
