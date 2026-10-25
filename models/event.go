package models

import "time"

// Event types written to the audit log and pushed over SSE.
const (
	EventTypeCreated          = "created"
	EventTypeStatus           = "status"
	EventTypeExecuting        = "executing"
	EventTypeVerified         = "verified"
	EventTypeCompleted        = "completed"
	EventTypeStagesDone       = "stages_done"
	EventTypeDependencyFailed = "dependency_failed"
)

// Event is an audit record of one state transition or pipeline decision.
type Event struct {
	ID         int64     `json:"id"`
	TaskID     string    `json:"task_id"`
	Type       string    `json:"type"`
	FromStatus string    `json:"from_status,omitempty"`
	ToStatus   string    `json:"to_status,omitempty"`
	Message    string    `json:"message,omitempty"`
	Payload    string    `json:"payload,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// EventEnvelope is the transport frame for SSE: {"type":"task","data":{...}}.
type EventEnvelope struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}
