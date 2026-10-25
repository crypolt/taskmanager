package internals

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"taskmanager/config"
	"taskmanager/internal/services/externals/llm"
	"taskmanager/models"
	"taskmanager/utils/helpers"
	"taskmanager/utils/i18n"
)

// Engine is the task pipeline orchestrator: queue, workers, routing,
// planner/executor/verifier and retry/escalation.
type Engine struct {
	st     Store
	reg    *llm.Registry
	router *Router
	bus    *Bus
	log    *slog.Logger
	cfg    *config.Config

	jobs chan string
	wake chan struct{}
	wg   sync.WaitGroup

	// runs maps a task id to the cancel func of its in-flight model call, so
	// cancelling a task also kills the child CLI instead of letting it burn
	// tokens to completion.
	runs sync.Map
}

func NewEngine(st Store, reg *llm.Registry, b *Bus, log *slog.Logger, cfg *config.Config) *Engine {
	return &Engine{
		st:     st,
		reg:    reg,
		router: NewRouter(st),
		bus:    b,
		log:    log,
		cfg:    cfg,
		jobs:   make(chan string, max(1, cfg.LLM.Workers)),
		wake:   make(chan struct{}, 1),
	}
}

func (e *Engine) Start(ctx context.Context) error {
	if err := e.st.Recover(ctx); err != nil {
		return err
	}
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.dispatchLoop(ctx)
	}()
	for i := 0; i < max(1, e.cfg.LLM.Workers); i++ {
		e.wg.Add(1)
		go func(n int) {
			defer e.wg.Done()
			e.worker(ctx, n)
		}(i)
	}
	e.log.Info("engine_started", "workers", e.cfg.LLM.Workers)
	return nil
}

func (e *Engine) Stop() { e.wg.Wait() }

// Wake nudges the dispatcher without waiting for the polling tick.
func (e *Engine) Wake() {
	select {
	case e.wake <- struct{}{}:
	default:
	}
}

// --- read API used by handlers --------------------------------------------

func (e *Engine) ListTasks(ctx context.Context, status string) ([]models.Task, error) {
	tasks, err := e.st.ListTasks(ctx, status, 500)
	if err != nil {
		return nil, err
	}
	if tasks == nil {
		return []models.Task{}, nil
	}
	return tasks, nil
}

func (e *Engine) Detail(ctx context.Context, id string) (models.TaskDetail, error) {
	task, err := e.st.GetTask(ctx, id)
	if err != nil {
		return models.TaskDetail{}, err
	}
	children, _ := e.st.Children(ctx, id)
	attempts, _ := e.st.ListAttempts(ctx, id)
	events, _ := e.st.ListEvents(ctx, id, 200)
	entries, _ := e.st.ListContext(ctx, e.rootID(&task), 200)
	return models.TaskDetail{
		Task:     task,
		Children: nonNilTasks(children),
		Attempts: nonNilAttempts(attempts),
		Events:   nonNilEvents(events),
		Context:  nonNilContext(entries),
	}, nil
}

func (e *Engine) Stats(ctx context.Context) (models.Stats, error) { return e.st.Stats(ctx) }

// ModelUsage returns per-model token accounting for the models page.
func (e *Engine) ModelUsage(ctx context.Context) (models.ModelUsageResponse, error) {
	return e.st.ModelUsage(ctx)
}

func (e *Engine) Models(ctx context.Context) (models.ModelsResponse, error) {
	routes, err := e.st.Routes(ctx)
	if err != nil {
		return models.ModelsResponse{}, err
	}
	grouped := map[string][]models.Route{}
	for _, route := range routes {
		grouped[route.Role] = append(grouped[route.Role], route)
	}
	return models.ModelsResponse{Routes: grouped, Providers: e.reg.Names()}, nil
}

func (e *Engine) SetRoute(ctx context.Context, in models.SetRouteInput) error {
	in.Role = strings.TrimSpace(in.Role)
	in.Model = strings.TrimSpace(in.Model)
	in.Provider = strings.TrimSpace(in.Provider)
	if in.Provider == "" {
		in.Provider = "opencode"
	}
	if in.Role == "" || in.Model == "" {
		return helpers.ErrRoleModelNeeded
	}
	if in.Position < 0 || in.PriceIn < 0 || in.PriceOut < 0 {
		return helpers.ErrInvalidRoute
	}
	return e.st.SetRoute(ctx, in.Role, in.Position, in.Provider, in.Model, in.PriceIn, in.PriceOut, in.DailyTokens)
}

func (e *Engine) SetRouteLimit(ctx context.Context, role string, position int, dailyTokens int) error {
	return e.st.SetRouteLimit(ctx, role, position, dailyTokens)
}

// DailyModelUsage returns per-model token accounting for the last 24 hours.
func (e *Engine) DailyModelUsage(ctx context.Context) (models.ModelUsageResponse, error) {
	return e.st.DailyModelUsage(ctx)
}

// ModelHealth returns the latest provider refusal per provider/model pair.
func (e *Engine) ModelHealth(ctx context.Context) (map[string]models.ModelHealth, error) {
	return e.st.ModelHealth(ctx)
}

func (e *Engine) DeleteRoute(ctx context.Context, role string, position int) error {
	role = strings.TrimSpace(role)
	if role == "" || position < 0 {
		return helpers.ErrInvalidRoute
	}
	return e.st.DeleteRoute(ctx, role, position)
}

func (e *Engine) MoveRoute(ctx context.Context, in models.MoveRouteInput) error {
	in.Role = strings.TrimSpace(in.Role)
	if in.Role == "" || in.From < 0 || in.To < 0 || in.From == in.To {
		return helpers.ErrInvalidRoute
	}
	return e.st.MoveRoute(ctx, in.Role, in.From, in.To)
}

func (e *Engine) Pipelines(ctx context.Context) []models.Pipeline {
	out := make([]models.Pipeline, 0, len(models.Pipelines))
	for _, pipeline := range models.Pipelines {
		out = append(out, localizePipeline(ctx, pipeline))
	}
	return out
}

// localizePipeline renders a template in the request language.
func localizePipeline(ctx context.Context, pipeline models.Pipeline) models.Pipeline {
	pipeline.Name = i18n.T(ctx, i18n.Key(pipeline.NameKey), nil)
	pipeline.About = i18n.T(ctx, i18n.Key(pipeline.AboutKey), nil)
	stages := make([]models.Stage, len(pipeline.Stages))
	for i, stage := range pipeline.Stages {
		stage.Title = i18n.T(ctx, i18n.Key(stage.TitleKey), nil)
		stages[i] = stage
	}
	pipeline.Stages = stages
	return pipeline
}

func (e *Engine) Subscribe() chan models.EventEnvelope { return e.bus.Subscribe() }

func (e *Engine) Unsubscribe(ch chan models.EventEnvelope) { e.bus.Unsubscribe(ch) }

// --- write API ------------------------------------------------------------

func (e *Engine) Create(ctx context.Context, in models.CreateTaskInput) (models.Task, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return models.Task{}, helpers.ErrTitleRequired
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

	switch in.Pipeline {
	case models.PipelineIDDev:
		return e.createDevPipeline(ctx, in)
	case models.PipelineIDFree:
		return e.createFreeTask(ctx, in)
	default:
		return e.createSingleTask(ctx, in)
	}
}

func (e *Engine) createDevPipeline(ctx context.Context, in models.CreateTaskInput) (models.Task, error) {
	pipeline := localizePipeline(ctx, models.PipelineDev)
	root := e.newTask(in, models.RoleAggregate)
	root.Status = models.StatusBlocked
	if err := e.st.CreateTask(ctx, root); err != nil {
		return models.Task{}, err
	}
	e.event(ctx, root.ID, models.EventTypeCreated, "", root.Status,
		i18n.T(ctx, i18n.MsgTaskAcceptedPipeline, map[string]any{"pipeline": pipeline.Name}))

	var previous string
	for _, stage := range pipeline.Stages {
		child := e.newTask(in, stage.Role)
		child.ParentID = &root.ID
		child.Stage = stage.Key
		child.Title = i18n.T(ctx, i18n.MsgTaskStageTitle, map[string]any{"stage": stage.Title, "title": in.Title})
		child.Description = stage.Brief + "\n\n## Task\n" + in.Title + "\n" + in.Description
		child.Status = models.StatusPending
		if previous != "" {
			child.DependsOn = []string{previous}
		}
		if err := e.st.CreateTask(ctx, child); err != nil {
			return models.Task{}, err
		}
		previous = child.ID
	}
	e.publish(root)
	return *root, nil
}

func (e *Engine) createFreeTask(ctx context.Context, in models.CreateTaskInput) (models.Task, error) {
	root := e.newTask(in, models.RolePlanner)
	root.Status = models.StatusPending
	if err := e.st.CreateTask(ctx, root); err != nil {
		return models.Task{}, err
	}
	e.event(ctx, root.ID, models.EventTypeCreated, "", root.Status,
		i18n.T(ctx, i18n.MsgTaskAcceptedPlanner, nil))
	e.publish(root)
	e.Wake()
	return *root, nil
}

func (e *Engine) createSingleTask(ctx context.Context, in models.CreateTaskInput) (models.Task, error) {
	root := e.newTask(in, models.RoleGeneric)
	root.Pipeline = models.PipelineIDSingle
	root.Status = models.StatusPending
	if err := e.st.CreateTask(ctx, root); err != nil {
		return models.Task{}, err
	}
	e.event(ctx, root.ID, models.EventTypeCreated, "", root.Status, i18n.T(ctx, i18n.MsgTaskAccepted, nil))
	e.publish(root)
	e.Wake()
	return *root, nil
}

func (e *Engine) newTask(in models.CreateTaskInput, role string) *models.Task {
	return &models.Task{
		ID:           helpers.NewID("tsk"),
		Pipeline:     in.Pipeline,
		Role:         role,
		Title:        in.Title,
		Description:  in.Description,
		Priority:     in.Priority,
		ModelHint:    in.Model,
		MaxAttempts:  in.MaxAttempts,
		BudgetTokens: in.BudgetTokens,
		DependsOn:    []string{},
		CreatedAt:    time.Now().UTC(),
	}
}

func (e *Engine) Cancel(ctx context.Context, id string) error {
	task, err := e.st.GetTask(ctx, id)
	if err != nil {
		return err
	}
	if task.Status == models.StatusDone || task.Status == models.StatusCancelled {
		return nil
	}
	children, _ := e.st.Children(ctx, id)
	for _, child := range children {
		item := child
		if item.Status == models.StatusDone || item.Status == models.StatusCancelled {
			continue
		}
		e.stopRun(item.ID)
		e.setStatus(ctx, &item, models.StatusCancelled, i18n.T(ctx, i18n.MsgTaskCancelledParent, nil))
	}
	e.stopRun(id)
	e.setStatus(ctx, &task, models.StatusCancelled, i18n.T(ctx, i18n.MsgTaskCancelled, nil))
	return nil
}

// stopRun cancels the model call of a task, if one is in flight.
func (e *Engine) stopRun(id string) {
	if cancel, ok := e.runs.LoadAndDelete(id); ok {
		cancel.(context.CancelFunc)()
	}
}

func (e *Engine) Retry(ctx context.Context, id string) error {
	task, err := e.st.GetTask(ctx, id)
	if err != nil {
		return err
	}
	if children, err := e.st.Children(ctx, id); err == nil && len(children) > 0 {
		return e.resumeParent(ctx, &task)
	}
	task.AttemptCount = 0
	task.Error = ""
	if err := e.st.UpdateTask(ctx, &task); err != nil {
		return err
	}
	e.setStatus(ctx, &task, models.StatusPending, i18n.T(ctx, i18n.MsgTaskRetried, nil))
	e.Wake()
	return nil
}

// resumeParent re-queues the unfinished stages of a parent task and keeps the
// finished ones (and their results) intact. Used by retry and by a human
// answer: a partly completed pipeline continues instead of restarting.
func (e *Engine) resumeParent(ctx context.Context, parent *models.Task) error {
	children, err := e.st.Children(ctx, parent.ID)
	if err != nil {
		return err
	}
	for i := range children {
		child := &children[i]
		switch child.Status {
		case models.StatusDone, models.StatusRunning, models.StatusVerifying, models.StatusPlanning:
			continue
		}
		child.Status = models.StatusPending
		child.AttemptCount = 0
		child.Error = ""
		child.StartedAt = nil
		child.FinishedAt = nil
		if err := e.st.UpdateTask(ctx, child); err != nil {
			return err
		}
		e.event(ctx, child.ID, models.EventTypeStatus, "", models.StatusPending,
			i18n.T(ctx, i18n.MsgTaskResumed, nil))
		e.publish(child)
	}
	parent.AttemptCount = 0
	parent.Error = ""
	if err := e.st.UpdateTask(ctx, parent); err != nil {
		return err
	}
	e.setStatus(ctx, parent, models.StatusBlocked, i18n.T(ctx, i18n.MsgTaskResumed, nil))
	if fresh, err := e.st.Children(ctx, parent.ID); err == nil && len(fresh) > 0 {
		e.maybeFinishParent(ctx, parent, fresh)
	}
	e.Wake()
	return nil
}

// Move applies a manual status change from the board. Only statuses a human
// can meaningfully set are accepted: engine-owned states (planning, running,
// verifying, retrying) are refused, so a drag never fakes execution.
func (e *Engine) Move(ctx context.Context, id, status string) error {
	status = strings.TrimSpace(status)
	switch status {
	case models.StatusPending, models.StatusDone, models.StatusCancelled, models.StatusNeedsHuman:
	default:
		return helpers.ErrTaskMoveInvalid(status)
	}
	task, err := e.st.GetTask(ctx, id)
	if err != nil {
		return err
	}
	if task.Status == status {
		return nil
	}
	from := task.Status
	task.Status = status
	task.Error = ""
	switch status {
	case models.StatusPending:
		task.AttemptCount = 0
		task.StartedAt = nil
		task.FinishedAt = nil
	case models.StatusDone, models.StatusCancelled:
		now := time.Now().UTC()
		if task.StartedAt == nil {
			task.StartedAt = &now
		}
		task.FinishedAt = &now
	}
	if err := e.st.UpdateTask(ctx, &task); err != nil {
		return err
	}
	e.event(ctx, task.ID, models.EventTypeStatus, from, status,
		i18n.T(ctx, i18n.MsgTaskMoved, map[string]any{"status": statusLabel(ctx, status)}))
	e.publish(&task)
	if status == models.StatusPending {
		e.Wake()
	} else {
		e.completeParent(ctx, task)
	}
	return nil
}

func (e *Engine) Answer(ctx context.Context, id, note string) error {
	task, err := e.st.GetTask(ctx, id)
	if err != nil {
		return err
	}
	task.HumanNote = note
	e.remember(ctx, &task, models.ContextKindNote, i18n.T(ctx, i18n.MsgContextHumanNote, nil), note)
	if children, err := e.st.Children(ctx, id); err == nil && len(children) > 0 {
		return e.resumeParent(ctx, &task)
	}
	task.AttemptCount = 0
	task.Error = ""
	if err := e.st.UpdateTask(ctx, &task); err != nil {
		return err
	}
	e.setStatus(ctx, &task, models.StatusPending, i18n.T(ctx, i18n.MsgTaskAnswered, nil))
	e.Wake()
	return nil
}

// --- small helpers --------------------------------------------------------

// statusLabel renders a task status code in the request language. Unknown
// codes (statuses a human invented) stay raw instead of rendering a missing
// key.
func statusLabel(ctx context.Context, status string) string {
	if key := models.StatusKey(status); key != "" {
		return i18n.T(ctx, key, nil)
	}
	return status
}

func nonNilTasks(items []models.Task) []models.Task {
	if items == nil {
		return []models.Task{}
	}
	return items
}

func nonNilAttempts(items []models.Attempt) []models.Attempt {
	if items == nil {
		return []models.Attempt{}
	}
	return items
}

func nonNilEvents(items []models.Event) []models.Event {
	if items == nil {
		return []models.Event{}
	}
	return items
}

func nonNilContext(items []models.ContextEntry) []models.ContextEntry {
	if items == nil {
		return []models.ContextEntry{}
	}
	return items
}
