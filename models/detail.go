package models

// TaskDetail is the aggregated read model of one task with its pipeline
// children, model attempts and audit events.
type TaskDetail struct {
	Task     Task           `json:"task"`
	Children []Task         `json:"children"`
	Attempts []Attempt      `json:"attempts"`
	Events   []Event        `json:"events"`
	Context  []ContextEntry `json:"context"`
}
