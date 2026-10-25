package internals

import "testing"

// The verifier often mentions paths like `{en,ru}.json` before its verdict.
// The first brace in such a reply is not JSON, so parsing must skip it.
const verdictWithBracesInProse = "Verified positively: `web/src/locales/{en,ru}.json` (183 keys, valid).\n\n" +
	`{"pass": false, "score": 65, "issues": ["npm run check:locales exits 1"]}`

func TestParseVerdictSkipsProseBraces(t *testing.T) {
	verdict, err := ParseVerdict(verdictWithBracesInProse)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict.Pass || verdict.Score != 65 || len(verdict.Issues) != 1 {
		t.Fatalf("unexpected verdict: %+v", verdict)
	}
}

func TestParseVerdictInFence(t *testing.T) {
	raw := "Итог:\n```json\n{\"pass\": true, \"score\": 90}\n```\n"
	verdict, err := ParseVerdict(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !verdict.Pass || verdict.Score != 90 {
		t.Fatalf("unexpected verdict: %+v", verdict)
	}
}

func TestParsePlanSkipsProseBraces(t *testing.T) {
	raw := "Scope: `{internal,web}`.\n{\"subtasks\": [{\"title\": \"Audit\", \"role\": \"analyze\"}]}"
	plan, err := ParsePlan(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan) != 1 || plan[0].Title != "Audit" {
		t.Fatalf("unexpected plan: %+v", plan)
	}
}

func TestParseVerdictWithoutJSON(t *testing.T) {
	verdict, err := ParseVerdict("looks fine to me")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verdict.Pass {
		t.Fatalf("expected rejection, got %+v", verdict)
	}
}
