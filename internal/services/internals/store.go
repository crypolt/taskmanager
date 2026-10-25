package internals

import (
	"context"

	"taskmanager/models"
)

// Store is the persistence port used by the engine. It is declared on the
// consumer side (internals), implemented by internal/repository.
type Store interface {
	CreateTask(ctx context.Context, t *models.Task) error
	GetTask(ctx context.Context, id string) (models.Task, error)
	ListTasks(ctx context.Context, status string, limit int) ([]models.Task, error)
	Children(ctx context.Context, parentID string) ([]models.Task, error)
	UpdateTask(ctx context.Context, t *models.Task) error
	SetStatus(ctx context.Context, id, status string) error
	ClaimTask(ctx context.Context, id string) (bool, error)
	ReadyTasks(ctx context.Context) ([]models.Task, error)
	Recover(ctx context.Context) error
	AddAttempt(ctx context.Context, a *models.Attempt) error
	UpdateAttempt(ctx context.Context, a *models.Attempt) error
	ListAttempts(ctx context.Context, taskID string) ([]models.Attempt, error)
	AddEvent(ctx context.Context, e *models.Event) error
	ListEvents(ctx context.Context, taskID string, limit int) ([]models.Event, error)
	AddContext(ctx context.Context, entry *models.ContextEntry) error
	ListContext(ctx context.Context, rootID string, limit int) ([]models.ContextEntry, error)
	Routes(ctx context.Context) ([]models.Route, error)
	RouteChain(ctx context.Context, role string) ([]models.Route, error)
	SetRoute(ctx context.Context, role string, position int, provider, model string, priceIn, priceOut float64, dailyTokens int) error
	SetRouteLimit(ctx context.Context, role string, position int, dailyTokens int) error
	DeleteRoute(ctx context.Context, role string, position int) error
	MoveRoute(ctx context.Context, role string, from int, to int) error
	DailyModelUsage(ctx context.Context) (models.ModelUsageResponse, error)
	ModelHealth(ctx context.Context) (map[string]models.ModelHealth, error)
	Stats(ctx context.Context) (models.Stats, error)
	ModelUsage(ctx context.Context) (models.ModelUsageResponse, error)
}
