package unit

import (
	"strings"
	"testing"

	"taskmanager/models"
	"taskmanager/skills"
)

func TestAgentPromptsPresentForEveryRole(t *testing.T) {
	t.Parallel()

	roles := []string{
		models.RolePlanner, models.RoleAnalyze, models.RoleDesign, models.RoleImplement,
		models.RoleTest, models.RoleReview, models.RoleDocs, models.RoleGeneric,
		models.RoleVerify, models.RoleAggregate,
	}
	for _, role := range roles {
		role := role
		t.Run(role, func(t *testing.T) {
			t.Parallel()
			prompt, ok := skills.Agent(role)
			if !ok {
				t.Fatalf("no prompt for role %q", role)
			}
			if len(prompt) < 200 {
				t.Fatalf("prompt for %q looks truncated: %d bytes", role, len(prompt))
			}
		})
	}
}

func TestPlannerAndVerifierStateMachineContracts(t *testing.T) {
	t.Parallel()

	planner, _ := skills.Agent(models.RolePlanner)
	if !strings.Contains(planner, `"subtasks"`) {
		t.Fatal("planner prompt must show the subtasks JSON contract")
	}
	verifier, _ := skills.Agent(models.RoleVerify)
	for _, want := range []string{`"pass"`, `"score"`, `"issues"`} {
		if !strings.Contains(verifier, want) {
			t.Fatalf("verifier prompt must contain %s", want)
		}
	}
}

func TestUnknownRoleFallsBackToGeneric(t *testing.T) {
	t.Parallel()

	if _, ok := skills.Agent("no-such-role"); ok {
		t.Fatal("unknown role must not resolve")
	}
	if _, ok := skills.Agent(models.RoleGeneric); !ok {
		t.Fatal("generic prompt must exist")
	}
}
