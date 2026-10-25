package unit

import (
	"testing"

	"taskmanager/models"
)

func TestNormalizeScope(t *testing.T) {
	t.Parallel()

	got := models.NormalizeScope([]string{" ./internal/foo/ ", "internal/foo", "", ".", "web/src"})
	if len(got) != 2 || got[0] != "internal/foo" || got[1] != "web/src" {
		t.Fatalf("unexpected scope: %#v", got)
	}

	many := make([]string, 0, models.MaxScopeEntries+5)
	for i := 0; i < models.MaxScopeEntries+5; i++ {
		many = append(many, string(rune('a'+i)))
	}
	if len(models.NormalizeScope(many)) != models.MaxScopeEntries {
		t.Fatal("scope must be capped")
	}
}

func TestScopesConflict(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		a, b []string
		want bool
	}{
		{"same file", []string{"web/src/app.js"}, []string{"web/src/app.js"}, true},
		{"parent dir", []string{"internal/services"}, []string{"internal/services/internals/engine.go"}, true},
		{"sibling dirs", []string{"internal/repository"}, []string{"internal/services"}, false},
		{"prefix trap", []string{"internal/foo"}, []string{"internal/foobar"}, false},
		{"empty scope", nil, []string{"web"}, false},
	}
	for _, tc := range cases {
		if got := models.ScopesConflict(tc.a, tc.b); got != tc.want {
			t.Fatalf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}

	if !models.ScopesConflictAny([]string{"web/src"}, [][]string{{"internal"}, {"web/src/ui"}}) {
		t.Fatal("conflict with any active scope must be detected")
	}
}

func TestParsePlanKeepsScope(t *testing.T) {
	t.Parallel()

	raw := `{"subtasks":[{"title":"API","description":"d","role":"implement","scope":[" ./internal/api/ ","internal/api"]}]}`
	subtasks, err := parsePlan(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(subtasks[0].Scope) != 1 || subtasks[0].Scope[0] != "internal/api" {
		t.Fatalf("scope not normalized: %#v", subtasks[0].Scope)
	}
}
