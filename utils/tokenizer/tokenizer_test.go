package tokenizer

import (
	"strings"
	"testing"
)

func TestEstimateGrowsWithText(t *testing.T) {
	short := Estimate("hello world")
	long := Estimate(strings.Repeat("hello world ", 100))
	if short <= 0 || long <= short {
		t.Fatalf("estimate must grow: short=%d long=%d", short, long)
	}
	if Estimate("") != 0 {
		t.Fatalf("empty text must be 0 tokens")
	}
}

func TestTruncateRespectsBudget(t *testing.T) {
	text := strings.Repeat("Implement the CSV export with headers, escaping and tests. ", 200)
	got := Truncate(text, 120)
	if Estimate(got) > 120 {
		t.Fatalf("truncated text is %d tokens, budget 120", Estimate(got))
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("truncated text must end with an ellipsis")
	}
	if !Fits(text, 100000) {
		t.Fatalf("large budget must fit")
	}
}

func TestTruncateKeepsShortText(t *testing.T) {
	if got := Truncate("short", 100); got != "short" {
		t.Fatalf("short text must be untouched, got %q", got)
	}
}
