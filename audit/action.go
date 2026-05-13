package audit

// Action identifies the audited operation. Values follow SCREAMING_SNAKE_CASE
// and are defined per-service (e.g. `const ActionAgentProvision audit.Action = "AGENT_PROVISION"`)
// so the catalogue stays adjacent to the code that emits it.
type Action string

// ActorType is universal across services.
type ActorType string

const (
	ActorTypeUser   ActorType = "USER"
	ActorTypeSystem ActorType = "SYSTEM"
	ActorTypeWorker ActorType = "WORKER"
)

// Result is universal across services.
type Result string

const (
	ResultSuccess Result = "SUCCESS"
	ResultFailure Result = "FAILURE"
)
