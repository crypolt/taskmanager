package models

// CreateTaskInput is the POST /tasks request body.
type CreateTaskInput struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	Pipeline     string `json:"pipeline"`
	Priority     int    `json:"priority"`
	Model        string `json:"model"`
	MaxAttempts  int    `json:"max_attempts"`
	BudgetTokens int    `json:"budget_tokens"`
}

// AnswerInput carries the human clarification for a task in needs_human.
type MoveTaskInput struct {
	Status string `json:"status"`
}

type AnswerInput struct {
	Note string `json:"note"`
}

// PlanStep is one subtask of a user-editable plan. DependsOn holds indexes of
// earlier steps; an empty list means "the previous step".
type PlanStep struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Role        string   `json:"role"`
	Scope       []string `json:"scope"`
	DependsOn   []int    `json:"depends_on"`
}

// PlanInput is the POST /plans/preview and POST /plans request body.
type PlanInput struct {
	Title        string     `json:"title"`
	Context      string     `json:"context"`
	Pipeline     string     `json:"pipeline"`
	Priority     int        `json:"priority"`
	Model        string     `json:"model"`
	MaxAttempts  int        `json:"max_attempts"`
	BudgetTokens int        `json:"budget_tokens"`
	Subtasks     []PlanStep `json:"subtasks"`
}

// PlanPreview is a plan built from context but not yet turned into tasks.
type PlanPreview struct {
	Title    string     `json:"title"`
	Context  string     `json:"context"`
	Provider string     `json:"provider"`
	Model    string     `json:"model"`
	Subtasks []PlanStep `json:"subtasks"`
}
