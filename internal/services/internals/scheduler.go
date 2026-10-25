package internals

import (
	"context"
	"time"

	"taskmanager/models"
	"taskmanager/utils/i18n"
)

// dispatchLoop claims ready tasks and hands them to workers.
func (e *Engine) dispatchLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.wake:
		case <-ticker.C:
		}
		e.reconcile(ctx)
		e.dispatchReady(ctx)
	}
}

func (e *Engine) dispatchReady(ctx context.Context) {
	tasks, err := e.st.ReadyTasks(ctx)
	if err != nil {
		e.log.Error("list_ready_tasks", "err", err)
		return
	}
	active := e.activeScopes(ctx)
	for _, task := range tasks {
		ready, blockedBy := e.depsState(ctx, task)
		if blockedBy != "" {
			e.failOnDependency(ctx, &task, blockedBy)
			continue
		}
		if !ready {
			continue
		}
		if len(task.Scope) > 0 && models.ScopesConflictAny(task.Scope, active) {
			continue
		}
		claimed, err := e.st.ClaimTask(ctx, task.ID)
		if err != nil || !claimed {
			continue
		}
		task.Status = models.StatusRunning
		if len(task.Scope) > 0 {
			active = append(active, task.Scope)
		}
		e.publish(&task)
		select {
		case e.jobs <- task.ID:
		case <-ctx.Done():
			return
		}
	}
}

// reconcile finishes parents whose stages are all settled.
func (e *Engine) reconcile(ctx context.Context) {
	parents, err := e.st.ListTasks(ctx, models.StatusBlocked, 200)
	if err != nil {
		return
	}
	for _, parent := range parents {
		children, err := e.st.Children(ctx, parent.ID)
		if err != nil || len(children) == 0 {
			continue
		}
		item := parent
		e.maybeFinishParent(ctx, &item, children)
	}
}

func (e *Engine) failOnDependency(ctx context.Context, task *models.Task, reason string) {
	task.Status = models.StatusFailed
	task.Error = reason
	_ = e.st.UpdateTask(ctx, task)
	_ = e.st.SetStatus(ctx, task.ID, models.StatusFailed)
	e.event(ctx, task.ID, models.EventTypeDependencyFailed, "", models.StatusFailed, reason)
	e.publish(task)
	e.completeParent(ctx, *task)
}

func (e *Engine) depsState(ctx context.Context, task models.Task) (bool, string) {
	for _, id := range task.DependsOn {
		dep, err := e.st.GetTask(ctx, id)
		if err != nil {
			return false, i18n.T(ctx, i18n.MsgTaskDependencyGone, map[string]any{"dependency": id})
		}
		switch dep.Status {
		case models.StatusDone:
		case models.StatusFailed, models.StatusCancelled, models.StatusNeedsHuman:
			return false, i18n.T(ctx, i18n.MsgTaskStageFailed, map[string]any{"stage": dep.Title, "status": statusLabel(ctx, dep.Status)})
		default:
			return false, ""
		}
	}
	return true, ""
}

func (e *Engine) worker(ctx context.Context, n int) {
	for {
		select {
		case <-ctx.Done():
			return
		case id, ok := <-e.jobs:
			if !ok {
				return
			}
			e.runSafely(ctx, id)
		}
	}
}

func (e *Engine) runSafely(ctx context.Context, id string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			e.log.Error("worker_panic", "task", id, "panic", recovered)
			_ = e.st.SetStatus(context.Background(), id, models.StatusNeedsHuman)
		}
	}()
	e.process(ctx, id)
}

func (e *Engine) process(ctx context.Context, id string) {
	runCtx, cancel := context.WithCancel(ctx)
	e.runs.Store(id, cancel)
	defer func() {
		e.runs.Delete(id)
		cancel()
	}()
	ctx = runCtx
	task, err := e.st.GetTask(ctx, id)
	if err != nil {
		e.log.Error("load_task", "id", id, "err", err)
		return
	}
	if task.Status == models.StatusCancelled {
		return
	}
	if task.Status != models.StatusRunning {
		if claimed, _ := e.st.ClaimTask(ctx, id); !claimed {
			return
		}
	}
	children, _ := e.st.Children(ctx, id)

	switch {
	case task.Role == models.RolePlanner && len(children) == 0:
		e.plan(ctx, &task)
	case len(children) > 0:
		done := 0
		for _, child := range children {
			if child.Status == models.StatusDone {
				done++
			}
		}
		if done < len(children) {
			e.setStatus(ctx, &task, models.StatusBlocked,
				i18n.T(ctx, i18n.MsgTaskStagesPending, map[string]any{"done": done, "total": len(children)}))
			return
		}
		e.aggregate(ctx, &task, children)
	default:
		e.execute(ctx, &task)
	}
}

// activeScopes returns the scopes of tasks that already own files, so two
// parallel agents never edit the same paths.
func (e *Engine) activeScopes(ctx context.Context) [][]string {
	var out [][]string
	for _, status := range []string{models.StatusRunning, models.StatusVerifying, models.StatusRetrying} {
		tasks, err := e.st.ListTasks(ctx, status, 500)
		if err != nil {
			continue
		}
		for _, task := range tasks {
			if len(task.Scope) > 0 {
				out = append(out, task.Scope)
			}
		}
	}
	return out
}
