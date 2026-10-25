package internals

import (
	"context"
	"fmt"
	"strings"
	"time"

	"taskmanager/internal/services/externals/llm"
	"taskmanager/models"
	"taskmanager/utils/helpers"
	"taskmanager/utils/tokenizer"
)

const (
	// maxContextEntryTokens caps one shared-context record.
	maxContextEntryTokens = 900
	// contextShare is the part of the prompt budget reserved for shared context.
	contextShare = 3
)

// rootID returns the pipeline root a task belongs to.
func (e *Engine) rootID(task *models.Task) string {
	if task.ParentID != nil && *task.ParentID != "" {
		return *task.ParentID
	}
	return task.ID
}

// sharedContext renders the accumulated context of a pipeline root. When the
// budget is tight the newest entries win, so recent decisions are never lost.
func (e *Engine) sharedContext(ctx context.Context, task *models.Task) string {
	entries, err := e.st.ListContext(ctx, e.rootID(task), 200)
	if err != nil || len(entries) == 0 {
		return ""
	}
	budget := e.contextBudget()
	chosen := make([]models.ContextEntry, 0, len(entries))
	used := 0
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if entry.TaskID == task.ID {
			continue
		}
		cost := tokenizer.Estimate(entry.Title) + entry.Tokens + 4
		if len(chosen) > 0 && used+cost > budget {
			break
		}
		chosen = append([]models.ContextEntry{entry}, chosen...)
		used += cost
	}
	if len(chosen) == 0 {
		return ""
	}
	var builder strings.Builder
	for _, entry := range chosen {
		fmt.Fprintf(&builder, "\n### %s [%s]\n%s\n", entry.Title, entry.Kind, entry.Body)
	}
	return strings.TrimSpace(builder.String())
}

func (e *Engine) contextBudget() int {
	budget := e.cfg.LLM.MaxPromptTokens
	if budget <= 0 {
		budget = 8000
	}
	return budget / contextShare
}

// remember appends one shared-context record for the pipeline root.
func (e *Engine) remember(ctx context.Context, task *models.Task, kind, title, body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	body = tokenizer.Truncate(body, maxContextEntryTokens)
	entry := &models.ContextEntry{
		ID:        helpers.NewID("ctx"),
		RootID:    e.rootID(task),
		TaskID:    task.ID,
		Kind:      kind,
		Title:     title,
		Body:      body,
		Tokens:    tokenizer.Estimate(body),
		CreatedAt: time.Now().UTC(),
	}
	if err := e.st.AddContext(ctx, entry); err != nil {
		e.log.Warn("save_context", "task", task.ID, "err", err)
	}
}

// renderPlanSummary lists plan steps for the shared context.
func renderPlanSummary(subtasks []models.Subtask) string {
	var builder strings.Builder
	for i, subtask := range subtasks {
		fmt.Fprintf(&builder, "%d. [%s] %s", i+1, subtask.Role, subtask.Title)
		if len(subtask.Scope) > 0 {
			fmt.Fprintf(&builder, " (scope: %s)", strings.Join(subtask.Scope, ", "))
		}
		builder.WriteString("\n")
	}
	return builder.String()
}

// estimateUsage fills missing token counters from the local tokenizer, so the
// budget works even when a provider reports nothing.
func estimateUsage(response llm.Response, system, prompt string) llm.Response {
	if response.InputTokens == 0 {
		response.InputTokens = tokenizer.Estimate(system) + tokenizer.Estimate(prompt)
	}
	if response.OutputTokens == 0 {
		response.OutputTokens = tokenizer.Estimate(response.Text)
	}
	return response
}
