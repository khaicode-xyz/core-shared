// Package audit defines the audit-event contract shared by every ECQ Core
// service. Services build an Event and hand it to an Emitter; the emitter
// publishes onto the audit pipeline (typically a dedicated Kafka topic that a
// sink worker drains into a long-lived `audit_events` collection).
//
// Audit events are NOT operational logs. They record who did what to which
// entity, with what result — the answer to "what changed and why" that
// stdout/SignOz logs are too noisy to provide. Treat them as immutable
// business facts.
package audit

import "time"

// SchemaVersion lets the sink worker route old payloads through migration
// shims instead of refusing them. Bump only when the on-the-wire shape changes
// in a non-additive way.
const SchemaVersion = 1

// Event is the on-the-wire shape published to the audit topic. Optional fields
// are tagged omitempty so the sink can rely on field presence to detect intent.
type Event struct {
	EventID       string         `json:"event_id" bson:"event_id"`
	OccurredAt    time.Time      `json:"occurred_at" bson:"occurred_at"`
	Service       string         `json:"service" bson:"service"`
	Actor         Actor          `json:"actor" bson:"actor"`
	Action        Action         `json:"action" bson:"action"`
	Entity        Entity         `json:"entity" bson:"entity"`
	Result        Result         `json:"result" bson:"result"`
	ErrorCode     string         `json:"error_code,omitempty" bson:"error_code,omitempty"`
	RequestID     string         `json:"request_id,omitempty" bson:"request_id,omitempty"`
	TraceID       string         `json:"trace_id,omitempty" bson:"trace_id,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty" bson:"metadata,omitempty"`
	SchemaVersion int            `json:"schema_version" bson:"schema_version"`
}

// Actor is WHO performed the action.
type Actor struct {
	Type      ActorType `json:"type" bson:"type"`
	ID        string    `json:"id,omitempty" bson:"id,omitempty"`
	Email     string    `json:"email,omitempty" bson:"email,omitempty"`
	IP        string    `json:"ip,omitempty" bson:"ip,omitempty"`
	UserAgent string    `json:"user_agent,omitempty" bson:"user_agent,omitempty"`
}

// Entity is WHAT the action targeted. Type is service-defined (e.g. AGENT,
// JOB, CONFIG) and stays a free-form string so each service can model its own
// domain without coordinating a global registry.
type Entity struct {
	Type string `json:"type" bson:"type"`
	ID   string `json:"id,omitempty" bson:"id,omitempty"`
	Name string `json:"name,omitempty" bson:"name,omitempty"`
}
