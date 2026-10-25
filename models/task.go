package models

import "time"

// Task is a node of the task pipeline: a root task, a pipeline stage or a
// planner-generated subtask.
type Task struct {
	ID           string     `json:"id"`
	ParentID     *string    `json:"parent_id,omitempty"`
	Pipeline     string     `json:"pipeline,omitempty"`
	Stage        string     `json:"stage,omitempty"`
	Role         string     `json:"role"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Status       string     `json:"status"`
	Priority     int        `json:"priority"`
	DependsOn    []string   `json:"depends_on"`
	Scope        []string   `json:"scope,omitempty"`
	ModelHint    string     `json:"model_hint,omitempty"`
	Result       string     `json:"result,omitempty"`
	Error        string     `json:"error,omitempty"`
	BudgetTokens int        `json:"budget_tokens"`
	SpentTokens  int        `json:"spent_tokens"`
	CostUSD      float64    `json:"cost_usd"`
	AttemptCount int        `json:"attempt_count"`
	MaxAttempts  int        `json:"max_attempts"`
	HumanNote    string     `json:"human_note,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
}

// HasParent reports whether the task belongs to a pipeline.
func (t Task) HasParent() bool { return t.ParentID != nil && *t.ParentID != "" }

// BudgetExhausted reports whether the task spent its whole token budget.
func (t Task) BudgetExhausted() bool {
	return t.BudgetTokens > 0 && t.SpentTokens >= t.BudgetTokens
}
