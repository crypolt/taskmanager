package models

import "time"

// Attempt is one model call: executor, verifier, planner or aggregator.
type Attempt struct {
	ID               string    `json:"id"`
	TaskID           string    `json:"task_id"`
	AttemptNo        int       `json:"attempt_no"`
	Role             string    `json:"role"`
	Agent            string    `json:"agent"`
	Provider         string    `json:"provider"`
	Model            string    `json:"model"`
	Status           string    `json:"status"`
	Verdict          string    `json:"verdict,omitempty"`
	Score            int       `json:"score"`
	InputTokens      int       `json:"input_tokens"`
	OutputTokens     int       `json:"output_tokens"`
	CacheReadTokens  int       `json:"cache_read_tokens"`
	CacheWriteTokens int       `json:"cache_write_tokens"`
	CostUSD          float64   `json:"cost_usd"`
	LatencyMS        int64     `json:"latency_ms"`
	Prompt           string    `json:"prompt,omitempty"`
	Output           string    `json:"output,omitempty"`
	Issues           string    `json:"issues,omitempty"`
	Error            string    `json:"error,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}
