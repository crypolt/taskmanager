package internals

import (
	"context"
	"strings"

	"taskmanager/internal/services/externals/llm"
	"taskmanager/models"
	"taskmanager/utils/i18n"
)

// aggregate merges stage results into the parent task outcome.
func (e *Engine) aggregate(ctx context.Context, task *models.Task, children []models.Task) {
	stages := e.renderStageResults(ctx, children)
	route, provider, err := e.pickRoute(ctx, models.RoleAggregate, task.ModelHint, 0)
	if err == nil {
		system := SystemPrompt(models.RoleAggregate)
		contextText := ""
		if shared := e.sharedContext(ctx, task); shared != "" {
			contextText = "\n\n## Shared context\n" + shared
		}
		prompt := e.truncate("Assemble the final result from the pipeline stages. Cover: what was done, key artifacts, final outcome, risks.\n\n## Task\n" +
			task.Title + "\n" + task.Description + contextText + "\n\n## Stage results" + stages)
		attempt := e.startAttempt(ctx, task, route, models.AgentAggregator, prompt)
		callCtx, cancel := context.WithTimeout(ctx, e.cfg.ExecTimeout())
		response, callErr := provider.Chat(callCtx, llm.Request{
			Model:  route.Model,
			System: system,
			Prompt: prompt,
		})
		cancel()
		response = estimateUsage(response, system, prompt)
		e.finishAttempt(ctx, attempt, response, callErr)
		e.addUsage(ctx, task, response)
		if callErr == nil {
			task.Result = response.Text
			task.Error = ""
			if err := e.st.UpdateTask(ctx, task); err != nil {
				e.log.Error("save_result", "task", task.ID, "err", err)
			}
			e.setStatus(ctx, task, models.StatusDone, i18n.T(ctx, i18n.MsgTaskAggregated, nil))
			e.event(ctx, task.ID, models.EventTypeCompleted, "", models.StatusDone,
				i18n.T(ctx, i18n.MsgTaskAggregatedBy, map[string]any{"model": route.Model}))
			return
		}
		e.log.Warn("aggregator_failed", "task", task.ID, "err", callErr)
	}

	task.Result = strings.TrimSpace(stages)
	task.Error = i18n.T(ctx, i18n.MsgTaskAggregatedNoLLM, nil)
	if err := e.st.UpdateTask(ctx, task); err != nil {
		e.log.Error("save_result", "task", task.ID, "err", err)
	}
	e.setStatus(ctx, task, models.StatusDone, i18n.T(ctx, i18n.MsgTaskAggregatedGlued, nil))
	e.event(ctx, task.ID, models.EventTypeCompleted, "", models.StatusDone, task.Error)
}

func (e *Engine) renderStageResults(ctx context.Context, children []models.Task) string {
	var builder strings.Builder
	for _, child := range children {
		builder.WriteString("\n\n### ")
		builder.WriteString(child.Title)
		builder.WriteString(" (")
		builder.WriteString(child.Status)
		builder.WriteString(")\n")
		switch {
		case strings.TrimSpace(child.Result) != "":
			builder.WriteString(e.truncate(child.Result))
		case child.Error != "":
			builder.WriteString(i18n.T(ctx, i18n.MsgTaskStageError, map[string]any{"reason": child.Error}))
		}
	}
	return builder.String()
}
