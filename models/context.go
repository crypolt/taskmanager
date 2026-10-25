package models

import "time"

// Shared-context entry kinds.
const (
	ContextKindPlan   = "plan"
	ContextKindResult = "result"
	ContextKindNote   = "note"
)

// ContextEntry is one shared-context record of a pipeline root. Every agent of
// the same root task sees these entries, so decisions and results are not lost
// between subtasks.
type ContextEntry struct {
	ID        string    `json:"id"`
	RootID    string    `json:"root_id"`
	TaskID    string    `json:"task_id"`
	Kind      string    `json:"kind"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Tokens    int       `json:"tokens"`
	CreatedAt time.Time `json:"created_at"`
}
