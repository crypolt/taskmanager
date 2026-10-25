package unit

import (
	"testing"

	"taskmanager/internal/services/internals"
	"taskmanager/models"
)

func parsePlan(raw string) ([]models.Subtask, error) { return internals.ParsePlan(raw) }

func TestParsePlanNormalizesRolesAndClamps(t *testing.T) {
	t.Parallel()

	raw := "```json\n" + `{"subtasks":[
		{"title":"  Разбор  ","description":"d","role":"analyze"},
		{"title":"Код","description":"d","role":"wizard"},
		{"title":"","description":"drop me","role":"test"},
		{"title":"Тесты","description":"d","role":"test"}
	]}` + "\n```"

	subtasks, err := internals.ParsePlan(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(subtasks) != 3 {
		t.Fatalf("want 3 subtasks, got %d", len(subtasks))
	}
	if subtasks[0].Title != "Разбор" {
		t.Fatalf("title not trimmed: %q", subtasks[0].Title)
	}
	if subtasks[1].Role != models.RoleGeneric {
		t.Fatalf("unknown role must fall back to generic, got %q", subtasks[1].Role)
	}
}

func TestParsePlanRejectsEmptyAndMissingJSON(t *testing.T) {
	t.Parallel()

	if _, err := internals.ParsePlan("no json here"); err == nil {
		t.Fatal("missing JSON must fail")
	}
	if _, err := internals.ParsePlan(`{"subtasks":[]}`); err == nil {
		t.Fatal("empty plan must fail")
	}
}

func TestParsePlanCapsSubtasks(t *testing.T) {
	t.Parallel()

	var raw string
	for i := 0; i < models.MaxPlanSubtasks+5; i++ {
		raw += `{"title":"t","description":"d","role":"generic"},`
	}
	raw = `{"subtasks":[` + raw[:len(raw)-1] + `]}`

	subtasks, err := internals.ParsePlan(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(subtasks) != models.MaxPlanSubtasks {
		t.Fatalf("want %d subtasks, got %d", models.MaxPlanSubtasks, len(subtasks))
	}
}

func TestParseVerdict(t *testing.T) {
	t.Parallel()

	verdict, err := internals.ParseVerdict(`{"pass":false,"score":40,"issues":["нет тестов"]}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if verdict.Pass || verdict.Score != 40 || len(verdict.Issues) != 1 {
		t.Fatalf("unexpected verdict: %+v", verdict)
	}

	broken, err := internals.ParseVerdict("the model rambled instead of answering")
	if err != nil {
		t.Fatalf("malformed verdict must not be a transport error: %v", err)
	}
	if broken.Pass {
		t.Fatal("malformed verdict must fail")
	}
	if len(broken.Issues) == 0 {
		t.Fatal("malformed verdict must carry an issue")
	}
}
