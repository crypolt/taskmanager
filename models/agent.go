package models

// Agent kinds recorded on an attempt.
const (
	AgentExecutor   = "executor"
	AgentVerifier   = "verifier"
	AgentAggregator = "aggregator"
	AgentPlanner    = "planner"
)

// Attempt outcomes.
const (
	AttemptStatusRunning = "running"
	AttemptStatusOK      = "ok"
	AttemptStatusError   = "error"
)
