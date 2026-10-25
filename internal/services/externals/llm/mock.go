package llm

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Mock is a deterministic provider used by tests and by `--provider mock`
// runs when no model backend is available.
type Mock struct{}

func (m *Mock) Name() string { return "mock" }

func (m *Mock) Chat(ctx context.Context, req Request) (Response, error) {
	select {
	case <-ctx.Done():
		return Response{}, ctx.Err()
	case <-time.After(10 * time.Millisecond):
	}
	lower := strings.ToLower(req.Prompt)
	var text string
	switch {
	case strings.Contains(lower, "json") && strings.Contains(lower, "verdict"):
		text = `{"pass": true, "score": 90, "issues": []}`
	case strings.Contains(lower, "json") && strings.Contains(lower, "subtasks"):
		text = `{"subtasks":[{"title":"Разбор требований","description":"Уточнить входы и выходы","role":"analyze"},{"title":"Реализация","description":"Выполнить работу","role":"implement"}]}`
	default:
		text = fmt.Sprintf("MOCK(%s): %s", req.Model, summarize(req.Prompt))
	}
	return Response{
		Text:         text,
		Provider:     m.Name(),
		Model:        req.Model,
		InputTokens:  EstimateTokens(req.Prompt),
		OutputTokens: EstimateTokens(text),
		CostUSD:      0,
		Latency:      time.Since(time.Now()),
	}, nil
}

func summarize(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}
