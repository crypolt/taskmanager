package internals

import (
	"context"
	"strings"
	"time"

	"taskmanager/internal/services/externals/llm"
	"taskmanager/logger"
	"taskmanager/models"
	"taskmanager/utils/helpers"
	"taskmanager/utils/i18n"
	"taskmanager/utils/tokenizer"
)

// escalate moves the task to the next model in the chain, or asks for a human
// when the attempt budget is exhausted.
func (e *Engine) escalate(ctx context.Context, task *models.Task, cause error) {
	next := task.AttemptCount + 1
	task.Error = cause.Error()
	task.AttemptCount = next
	if err := e.st.UpdateTask(ctx, task); err != nil {
		e.log.Error("save_task", "task", task.ID, "err", err)
	}
	if next >= task.MaxAttempts {
		e.setStatus(ctx, task, models.StatusNeedsHuman,
			i18n.T(ctx, i18n.MsgTaskNeedsHuman, map[string]any{"cause": cause.Error(), "attempts": next}))
		e.completeParent(ctx, *task)
		return
	}
	e.setStatus(ctx, task, models.StatusRetrying,
		i18n.T(ctx, i18n.MsgTaskRetryFailed, map[string]any{"attempt": next, "total": task.MaxAttempts, "reason": cause.Error()}))
	e.Wake()
}

func (e *Engine) fail(ctx context.Context, task *models.Task, cause error) {
	task.Error = cause.Error()
	if err := e.st.UpdateTask(ctx, task); err != nil {
		e.log.Error("save_task", "task", task.ID, "err", err)
	}
	e.setStatus(ctx, task, models.StatusFailed, cause.Error())
	e.completeParent(ctx, *task)
}

func (e *Engine) completeParent(ctx context.Context, child models.Task) {
	if !child.HasParent() {
		return
	}
	parent, err := e.st.GetTask(ctx, *child.ParentID)
	if err != nil {
		return
	}
	children, err := e.st.Children(ctx, parent.ID)
	if err != nil || len(children) == 0 {
		return
	}
	e.maybeFinishParent(ctx, &parent, children)
}

func (e *Engine) maybeFinishParent(ctx context.Context, parent *models.Task, children []models.Task) {
	blockedReason := ""
	for _, child := range children {
		switch child.Status {
		case models.StatusDone:
		case models.StatusFailed, models.StatusCancelled:
			blockedReason = i18n.T(ctx, i18n.MsgTaskStageFailed, map[string]any{"stage": child.Title, "status": statusLabel(ctx, child.Status)})
		case models.StatusNeedsHuman:
			blockedReason = i18n.T(ctx, i18n.MsgTaskStageNeedsHuman, map[string]any{"stage": child.Title})
		default:
			return
		}
		if blockedReason != "" {
			break
		}
	}
	if blockedReason != "" {
		if parent.Status != models.StatusNeedsHuman {
			parent.Error = blockedReason
			if err := e.st.UpdateTask(ctx, parent); err != nil {
				e.log.Error("save_task", "task", parent.ID, "err", err)
			}
			e.setStatus(ctx, parent, models.StatusNeedsHuman, blockedReason)
		}
		return
	}
	if parent.Status != models.StatusBlocked {
		return
	}
	parent.Status = models.StatusPending
	_ = e.st.SetStatus(ctx, parent.ID, models.StatusPending)
	if err := e.st.UpdateTask(ctx, parent); err != nil {
		e.log.Error("save_task", "task", parent.ID, "err", err)
	}
	e.event(ctx, parent.ID, models.EventTypeStagesDone, models.StatusBlocked, models.StatusPending,
		i18n.T(ctx, i18n.MsgTaskStagesDone, nil))
	e.publish(parent)
	e.Wake()
}

func (e *Engine) depsContext(ctx context.Context, task *models.Task) string {
	var builder strings.Builder
	for _, id := range task.DependsOn {
		dep, err := e.st.GetTask(ctx, id)
		if err != nil {
			continue
		}
		builder.WriteString("\n### " + dep.Title + "\n")
		if strings.TrimSpace(dep.Result) != "" {
			builder.WriteString(e.truncate(dep.Result))
		}
	}
	return strings.TrimSpace(builder.String())
}

// startAttempt records the model call before it runs, so the UI shows a live
// attempt (and a real counter) while the provider is still thinking.
func (e *Engine) startAttempt(ctx context.Context, task *models.Task, route models.Route, agent, prompt string) *models.Attempt {
	attempt := &models.Attempt{
		ID:        helpers.NewID("att"),
		TaskID:    task.ID,
		AttemptNo: task.AttemptCount + 1,
		Role:      task.Role,
		Agent:     agent,
		Provider:  route.Provider,
		Model:     route.Model,
		Status:    models.AttemptStatusRunning,
		Prompt:    prompt,
		CreatedAt: time.Now().UTC(),
	}
	if err := e.st.AddAttempt(ctx, attempt); err != nil {
		logger.FromContext(ctx, e.log).Error("add_attempt", "task", task.ID, "err", err)
	}
	e.bus.Publish(models.EventEnvelope{Type: "attempt", Data: attempt})
	return attempt
}

// finishAttempt stores the outcome of a call started with startAttempt.
// A cancelled run still persists what it spent, so the write must not inherit
// the cancelled context.
func (e *Engine) finishAttempt(ctx context.Context, attempt *models.Attempt, response llm.Response, callErr error) {
	if ctx.Err() != nil {
		ctx = context.WithoutCancel(ctx)
	}
	attempt.InputTokens = response.InputTokens
	attempt.OutputTokens = response.OutputTokens
	attempt.CacheReadTokens = response.CacheReadTokens
	attempt.CacheWriteTokens = response.CacheWriteTokens
	attempt.CostUSD = response.CostUSD
	attempt.LatencyMS = time.Since(attempt.CreatedAt).Milliseconds()
	attempt.Output = response.Text
	attempt.Status = models.AttemptStatusOK
	if callErr != nil {
		attempt.Status = models.AttemptStatusError
		attempt.Error = callErr.Error()
	}
	if err := e.st.UpdateAttempt(ctx, attempt); err != nil {
		logger.FromContext(ctx, e.log).Error("update_attempt", "task", attempt.TaskID, "err", err)
	}
	log := logger.FromContext(ctx, e.log)
	attrs := []any{
		"task", attempt.TaskID,
		"role", attempt.Role,
		"provider", attempt.Provider,
		"model", attempt.Model,
		"status", attempt.Status,
		"input_tokens", attempt.InputTokens,
		"output_tokens", attempt.OutputTokens,
		"latency_ms", attempt.LatencyMS,
		"cost_usd", attempt.CostUSD,
	}
	if callErr != nil {
		log.Error("model_call", append(attrs, "err", callErr.Error())...)
	} else {
		log.Info("model_call", attrs...)
	}
	e.bus.Publish(models.EventEnvelope{Type: "attempt", Data: attempt})
}

func (e *Engine) addUsage(ctx context.Context, task *models.Task, response llm.Response) {
	if response.InputTokens == 0 && response.OutputTokens == 0 {
		return
	}
	if ctx.Err() != nil {
		ctx = context.WithoutCancel(ctx)
	}
	task.SpentTokens += response.InputTokens + response.OutputTokens
	task.CostUSD += response.CostUSD
	if err := e.st.UpdateTask(ctx, task); err != nil {
		e.log.Error("save_usage", "task", task.ID, "err", err)
	}
}

func (e *Engine) setStatus(ctx context.Context, task *models.Task, status, message string) {
	from := task.Status
	if err := e.st.SetStatus(ctx, task.ID, status); err != nil {
		logger.FromContext(ctx, e.log).Error("set_status", "task", task.ID, "err", err)
	}
	task.Status = status
	task.UpdatedAt = time.Now().UTC()
	if models.IsTerminalStatus(status) {
		finishedAt := task.UpdatedAt
		task.FinishedAt = &finishedAt
	}
	e.event(ctx, task.ID, models.EventTypeStatus, from, status, message)
	e.publish(task)
}

func (e *Engine) event(ctx context.Context, taskID, eventType, from, to, message string) {
	event := &models.Event{TaskID: taskID, Type: eventType, FromStatus: from, ToStatus: to, Message: message}
	if err := e.st.AddEvent(ctx, event); err == nil {
		e.bus.Publish(models.EventEnvelope{Type: "event", Data: event})
	}
}

func (e *Engine) publish(task *models.Task) {
	e.bus.Publish(models.EventEnvelope{Type: "task", Data: task})
}

func (e *Engine) truncate(text string) string {
	if limit := e.cfg.LLM.MaxPromptChars; limit > 0 && len(text) > limit {
		text = text[:limit] + "\n...[truncated]"
	}
	if maxTokens := e.cfg.LLM.MaxPromptTokens; maxTokens > 0 {
		text = tokenizer.Truncate(text, maxTokens)
	}
	return text
}
