package internals

import (
	"context"
	"strings"

	"taskmanager/internal/services/externals/llm"
	"taskmanager/models"
	"taskmanager/utils/helpers"
	"taskmanager/utils/i18n"
)

// execute runs the executor model, then the independent verifier.
func (e *Engine) execute(ctx context.Context, task *models.Task) {
	if task.Status == models.StatusCancelled {
		return
	}
	if task.BudgetExhausted() {
		e.escalate(ctx, task, helpers.ErrBudgetExhausted(task.BudgetTokens))
		return
	}

	route, provider, err := e.pickRoute(ctx, task.Role, task.ModelHint, task.AttemptCount)
	if err != nil {
		e.fail(ctx, task, err)
		return
	}

	contextText := e.depsContext(ctx, task)
	if shared := e.sharedContext(ctx, task); shared != "" {
		if contextText != "" {
			contextText += "\n\n"
		}
		contextText += shared
	}
	if len(task.Scope) > 0 {
		scopeLine := "Your scope (files and directories you own): " + strings.Join(task.Scope, ", ") +
			". Do not modify anything outside it."
		contextText = scopeLine + "\n\n" + contextText
	}
	system := SystemPrompt(task.Role)
	prompt := e.truncate(ExecPrompt(task.Title, task.Description, contextText))
	if task.HumanNote != "" {
		prompt += "\n\n## Human clarification\n" + task.HumanNote
	}

	e.event(ctx, task.ID, models.EventTypeExecuting, models.StatusRunning, models.StatusRunning,
		i18n.T(ctx, i18n.MsgTaskExecuting, map[string]any{"model": route.Model, "provider": route.Provider}))

	attempt := e.startAttempt(ctx, task, route, models.AgentExecutor, prompt)
	callCtx, cancel := context.WithTimeout(ctx, e.cfg.ExecTimeout())
	response, err := provider.Chat(callCtx, llm.Request{
		Model:  route.Model,
		System: system,
		Prompt: prompt,
	})
	cancel()
	response = estimateUsage(response, system, prompt)
	// A cancelled run must still persist what it spent, and the cancelled
	// context would reject the write.
	finalCtx := ctx
	if ctx.Err() != nil {
		finalCtx = context.WithoutCancel(ctx)
	}
	e.finishAttempt(finalCtx, attempt, response, err)
	e.addUsage(finalCtx, task, response)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		e.escalate(ctx, task, err)
		return
	}
	e.setStatus(ctx, task, models.StatusVerifying, i18n.T(ctx, i18n.MsgTaskExecutionDone, nil))

	verdict, err := e.verify(ctx, task, response.Text)
	if err != nil {
		e.escalate(ctx, task, helpers.ErrVerification(err))
		return
	}
	if verdict.Pass {
		task.Result = response.Text
		task.Error = ""
		if err := e.st.UpdateTask(ctx, task); err != nil {
			e.log.Error("save_result", "task", task.ID, "err", err)
		}
		e.setStatus(ctx, task, models.StatusDone, i18n.T(ctx, i18n.MsgTaskVerdictPass, map[string]any{"score": verdict.Score}))
		e.event(ctx, task.ID, models.EventTypeVerified, "", models.StatusDone,
			i18n.T(ctx, i18n.MsgTaskVerified, map[string]any{"model": route.Model, "score": verdict.Score}))
		e.remember(ctx, task, models.ContextKindResult, task.Title, response.Text)
		e.completeParent(ctx, *task)
		return
	}

	issues := strings.Join(verdict.Issues, "; ")
	if issues == "" {
		issues = i18n.T(ctx, i18n.MsgTaskVerdictRejected, nil)
	}
	e.escalate(ctx, task, helpers.ErrReviewRejected(verdict.Score, issues))
}

func (e *Engine) verify(ctx context.Context, task *models.Task, output string) (models.Verdict, error) {
	chain, err := e.router.Chain(ctx, models.RoleVerify)
	if err != nil || len(chain) == 0 {
		return models.Verdict{Pass: true, Score: 100}, nil
	}
	route, ok := Pick(chain, task.AttemptCount)
	if !ok {
		return models.Verdict{Pass: true, Score: 100}, nil
	}
	provider, err := e.reg.Get(route.Provider)
	if err != nil {
		return models.Verdict{}, err
	}

	system := VerifySystem()
	prompt := e.truncate(VerifyPrompt(task.Title, task.Description, output, e.sharedContext(ctx, task)))
	attempt := e.startAttempt(ctx, task, route, models.AgentVerifier, prompt)
	callCtx, cancel := context.WithTimeout(ctx, e.cfg.ExecTimeout())
	defer cancel()
	response, err := provider.Chat(callCtx, llm.Request{
		Model:  route.Model,
		System: system,
		Prompt: prompt,
	})
	response = estimateUsage(response, system, prompt)
	e.finishAttempt(ctx, attempt, response, err)
	e.addUsage(ctx, task, response)
	if err != nil {
		return models.Verdict{}, err
	}
	return ParseVerdict(response.Text)
}

func (e *Engine) pickRoute(ctx context.Context, role, hint string, attempt int) (models.Route, llm.Provider, error) {
	chain, err := e.router.Chain(ctx, role)
	if err != nil {
		return models.Route{}, nil, err
	}
	if len(chain) == 0 {
		return models.Route{}, nil, helpers.ErrNoModelForRole(role)
	}
	chain = ApplyHint(chain, hint)
	route, ok := Pick(chain, attempt)
	if !ok {
		return models.Route{}, nil, helpers.ErrEmptyChain
	}
	provider, err := e.reg.Get(route.Provider)
	if err != nil {
		return models.Route{}, nil, err
	}
	return route, provider, nil
}
