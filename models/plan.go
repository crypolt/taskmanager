package models

// Subtask is one unit of a planner-produced plan.
type Subtask struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Role        string   `json:"role"`
	Scope       []string `json:"scope,omitempty"`
}

// Verdict is the verifier's decision about an executor result.
type Verdict struct {
	Pass   bool     `json:"pass"`
	Score  int      `json:"score"`
	Issues []string `json:"issues"`
}

// PlanEnvelope is the JSON shape returned by the planner.
type PlanEnvelope struct {
	Subtasks []Subtask `json:"subtasks"`
}

// NormalizeVerdict clamps the verdict into its valid ranges.
func NormalizeVerdict(v Verdict) Verdict {
	if v.Score < 0 {
		v.Score = 0
	}
	if v.Score > 100 {
		v.Score = 100
	}
	if v.Issues == nil {
		v.Issues = []string{}
	}
	return v
}
