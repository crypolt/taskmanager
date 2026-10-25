package internals

import (
	"context"
	"strings"
	"time"

	"taskmanager/internal/services/externals/llm"
	"taskmanager/models"
	"taskmanager/utils/helpers"
	"taskmanager/utils/i18n"
)

// fallbackTitle titles a subtask the planner left untitled: the localized
// stage label of its role plus the parent title. Roles with no stage label
// (generic) fall back to the generic label instead of a raw role code.
func fallbackTitle(ctx context.Context, role, title string) string {
	label := i18n.T(ctx, i18n.StageGeneric, nil)
	if key := stageKey(role); i18n.Has(i18n.FromContext(ctx), key) {
		label = i18n.T(ctx, key, nil)
	}
	return i18n.T(ctx, i18n.MsgTaskStageTitle, map[string]any{"stage": label, "title": title})
}

// stageKey maps a role to its stage label key; roles without one return ""
// and the caller uses the generic label.
func stageKey(role string) i18n.Key {
	switch role {
	case models.RoleAnalyze:
		return i18n.StageAnalyze
	case models.RoleDesign:
		return i18n.StageDesign
	case models.RoleImplement:
		return i18n.StageImplement
	case models.RoleTest:
		return i18n.StageTest
	case models.RoleReview:
		return i18n.StageReview
	case models.RoleDocs:
		return i18n.StageDocs
	case models.RolePlanner:
		return i18n.StagePlan
	}
	return ""
}

// plan decomposes a free-form task into subtasks chained by dependencies.
func (e *Engine) plan(ctx context.Context, task *models.Task) {
	chain, err := e.router.Chain(ctx, models.RolePlanner)
	if err != nil || len(chain) == 0 {
		e.fail(ctx, task, helpers.ErrNoModelForRole(models.RolePlanner))
		return
	}
	chain = ApplyHint(chain, task.ModelHint)
	route, ok := Pick(chain, 0)
	if !ok {
		e.fail(ctx, task, helpers.ErrEmptyChain)
		return
	}
	provider, err := e.reg.Get(route.Provider)
	if err != nil {
		e.fail(ctx, task, err)
		return
	}

	prompt := e.truncate(PlanPrompt(task.Title, task.Description, e.sharedContext(ctx, task)))
	attempt := e.startAttempt(ctx, task, route, models.AgentPlanner, prompt)
	callCtx, cancel := context.WithTimeout(ctx, e.cfg.ExecTimeout())
	defer cancel()
	response, err := provider.Chat(callCtx, llm.Request{
		Model:  route.Model,
		System: SystemPrompt(models.RolePlanner),
		Prompt: prompt,
	})
	e.finishAttempt(ctx, attempt, response, err)
	e.addUsage(ctx, task, response)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		e.fail(ctx, task, err)
		return
	}

	subtasks, parseErr := ParsePlan(response.Text)
	fallback := false
	if parseErr != nil {
		e.log.Warn("planner_parse_failed", "task", task.ID, "err", parseErr)
		fallback = true
		subtasks = []models.Subtask{{
			Title:       task.Title,
			Description: task.Description,
			Role:        models.RoleGeneric,
		}}
	}
	created, err := e.createSubtasks(ctx, task, subtasks)
	if err != nil {
		e.fail(ctx, task, err)
		return
	}
	e.remember(ctx, task, models.ContextKindPlan, i18n.T(ctx, i18n.MsgContextPlan, nil), renderPlanSummary(subtasks))

	task.Role = models.RoleAggregate
	if err := e.st.UpdateTask(ctx, task); err != nil {
		e.fail(ctx, task, err)
		return
	}
	message := i18n.T(ctx, i18n.MsgTaskPlanReady, map[string]any{"count": len(subtasks)})
	if fallback {
		message += i18n.T(ctx, i18n.MsgTaskPlanFallback, nil)
	}
	e.setStatus(ctx, task, models.StatusBlocked, message)
	// Children are published only after every subtask exists and the parent is
	// blocked: otherwise a fast first child can finish while the rest are still
	// being inserted, and the parent aggregates an incomplete set.
	for i := range created {
		e.publish(&created[i])
	}
	e.Wake()
}

func (e *Engine) createSubtasks(ctx context.Context, parent *models.Task, subtasks []models.Subtask) ([]models.Task, error) {
	var previous string
	created := make([]models.Task, 0, len(subtasks))
	for _, subtask := range subtasks {
		child := &models.Task{
			ID:           helpers.NewID("tsk"),
			ParentID:     &parent.ID,
			Pipeline:     parent.Pipeline,
			Stage:        subtask.Role,
			Role:         subtask.Role,
			Title:        subtask.Title,
			Description:  subtask.Description + "\n\n## Task\n" + parent.Title + "\n" + parent.Description,
			Scope:        models.NormalizeScope(subtask.Scope),
			Status:       models.StatusPending,
			Priority:     parent.Priority,
			ModelHint:    parent.ModelHint,
			MaxAttempts:  parent.MaxAttempts,
			BudgetTokens: parent.BudgetTokens,
			DependsOn:    []string{},
			CreatedAt:    time.Now().UTC(),
		}
		if previous != "" {
			child.DependsOn = []string{previous}
		}
		if err := e.st.CreateTask(ctx, child); err != nil {
			return nil, err
		}
		created = append(created, *child)
		previous = child.ID
	}
	return created, nil
}

// PreviewPlan builds a subtask plan from free-form context. Nothing is
// persisted: the user reviews and edits the plan before creating tasks.
func (e *Engine) PreviewPlan(ctx context.Context, in models.PlanInput) (models.PlanPreview, error) {
	title := strings.TrimSpace(in.Title)
	contextText := strings.TrimSpace(in.Context)
	if contextText == "" {
		return models.PlanPreview{}, helpers.ErrContextRequired
	}
	if title == "" {
		title = firstLine(contextText, 80)
	}

	chain, err := e.router.Chain(ctx, models.RolePlanner)
	if err != nil || len(chain) == 0 {
		return models.PlanPreview{}, helpers.ErrNoModelForRole(models.RolePlanner)
	}
	chain = ApplyHint(chain, in.Model)
	route, ok := Pick(chain, 0)
	if !ok {
		return models.PlanPreview{}, helpers.ErrEmptyChain
	}
	provider, err := e.reg.Get(route.Provider)
	if err != nil {
		return models.PlanPreview{}, err
	}

	prompt := e.truncate(PlanPrompt(title, contextText, ""))
	callCtx, cancel := context.WithTimeout(ctx, e.cfg.ExecTimeout())
	defer cancel()
	response, err := provider.Chat(callCtx, llm.Request{
		Model:  route.Model,
		System: SystemPrompt(models.RolePlanner),
		Prompt: prompt,
	})
	if err != nil {
		return models.PlanPreview{}, helpers.ErrPlanGeneration(err)
	}

	subtasks, err := ParsePlan(response.Text)
	if err != nil {
		return models.PlanPreview{}, helpers.ErrPlanGeneration(err)
	}
	steps := make([]models.PlanStep, 0, len(subtasks))
	for i, subtask := range subtasks {
		step := models.PlanStep{
			Title:       subtask.Title,
			Description: subtask.Description,
			Role:        subtask.Role,
			Scope:       subtask.Scope,
			DependsOn:   []int{},
		}
		if i > 0 {
			step.DependsOn = []int{i - 1}
		}
		steps = append(steps, step)
	}
	return models.PlanPreview{
		Title:    title,
		Context:  contextText,
		Provider: route.Provider,
		Model:    route.Model,
		Subtasks: steps,
	}, nil
}

// CreateFromPlan turns an approved plan into a parent task with ready-made
// subtasks chained by dependencies. The parent only aggregates the result, so
// the planner is not invoked again.
func (e *Engine) CreateFromPlan(ctx context.Context, in models.PlanInput) (models.Task, error) {
	title := strings.TrimSpace(in.Title)
	contextText := strings.TrimSpace(in.Context)
	if contextText == "" {
		return models.Task{}, helpers.ErrContextRequired
	}
	if len(in.Subtasks) == 0 {
		return models.Task{}, helpers.ErrPlanEmpty
	}
	if title == "" {
		title = firstLine(contextText, 80)
	}
	if in.Pipeline == "" {
		in.Pipeline = models.PipelineIDFree
	}
	if _, ok := models.FindPipeline(in.Pipeline); !ok {
		return models.Task{}, helpers.ErrUnknownPipeline(in.Pipeline)
	}
	if in.MaxAttempts <= 0 {
		in.MaxAttempts = 3
	}

	create := models.CreateTaskInput{
		Title:        title,
		Description:  contextText,
		Pipeline:     in.Pipeline,
		Priority:     in.Priority,
		Model:        in.Model,
		MaxAttempts:  in.MaxAttempts,
		BudgetTokens: in.BudgetTokens,
	}

	root := e.newTask(create, models.RoleAggregate)
	root.Status = models.StatusBlocked
	if err := e.st.CreateTask(ctx, root); err != nil {
		return models.Task{}, err
	}
	e.event(ctx, root.ID, models.EventTypeCreated, "", root.Status,
		i18n.T(ctx, i18n.MsgPlanCreated, map[string]any{"count": len(in.Subtasks)}))

	ids := make([]string, len(in.Subtasks))
	children := make([]*models.Task, len(in.Subtasks))
	for i, step := range in.Subtasks {
		role := strings.TrimSpace(step.Role)
		if !models.IsPlanRole(role) {
			role = models.RoleGeneric
		}
		child := e.newTask(create, role)
		child.ParentID = &root.ID
		child.Stage = role
		child.Scope = models.NormalizeScope(step.Scope)
		child.Title = strings.TrimSpace(step.Title)
		if child.Title == "" {
			child.Title = fallbackTitle(ctx, role, title)
		}
		child.Description = strings.TrimSpace(step.Description) + "\n\n## Task\n" + title + "\n" + contextText
		child.Status = models.StatusPending
		ids[i] = child.ID
		children[i] = child
	}

	for i, step := range in.Subtasks {
		dependsOn := make([]string, 0, len(step.DependsOn))
		for _, index := range step.DependsOn {
			if index >= 0 && index < i && index < len(ids) {
				dependsOn = append(dependsOn, ids[index])
			}
		}
		if len(dependsOn) == 0 && i > 0 {
			dependsOn = []string{ids[i-1]}
		}
		children[i].DependsOn = dependsOn
		if err := e.st.CreateTask(ctx, children[i]); err != nil {
			return models.Task{}, err
		}
	}

	summary := make([]models.Subtask, 0, len(in.Subtasks))
	for _, step := range in.Subtasks {
		summary = append(summary, models.Subtask{
			Title: strings.TrimSpace(step.Title),
			Role:  step.Role,
			Scope: models.NormalizeScope(step.Scope),
		})
	}
	e.remember(ctx, root, models.ContextKindPlan, i18n.T(ctx, i18n.MsgContextPlan, nil), renderPlanSummary(summary))

	for _, child := range children {
		e.publish(child)
	}
	e.publish(root)
	e.Wake()
	return *root, nil
}

// firstLine returns a short title derived from the first line of the context.
func firstLine(text string, limit int) string {
	line := strings.TrimSpace(strings.SplitN(text, "\n", 2)[0])
	runes := []rune(line)
	if len(runes) > limit {
		return strings.TrimSpace(string(runes[:limit])) + "…"
	}
	if line == "" {
		return strings.TrimSpace(text)
	}
	return line
}
