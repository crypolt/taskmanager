package internals

import (
	"context"
	"strings"

	"taskmanager/models"
)

// Router selects a model for a role. The chain is ordered from cheap to
// strong; a failed attempt escalates to the next entry.
type Router struct {
	store Store
}

func NewRouter(store Store) *Router { return &Router{store: store} }

func (r *Router) Chain(ctx context.Context, role string) ([]models.Route, error) {
	chain, err := r.store.RouteChain(ctx, role)
	if err != nil {
		return nil, err
	}
	if len(chain) == 0 {
		return r.store.RouteChain(ctx, models.RoleGeneric)
	}
	return chain, nil
}

// Pick returns the chain entry for an attempt, capped at the last entry.
func Pick(chain []models.Route, attempt int) (models.Route, bool) {
	if len(chain) == 0 {
		return models.Route{}, false
	}
	index := attempt
	if index >= len(chain) {
		index = len(chain) - 1
	}
	if index < 0 {
		index = 0
	}
	return chain[index], true
}

// ApplyHint puts an explicit provider/model hint in front of the chain,
// keeping the rest as fallback.
func ApplyHint(chain []models.Route, hint string) []models.Route {
	hint = strings.TrimSpace(hint)
	if hint == "" {
		return chain
	}
	provider, model := "opencode", hint
	if index := strings.Index(hint, ":"); index > 0 {
		provider, model = hint[:index], hint[index+1:]
	}
	head := models.Route{Role: "hint", Position: 0, Provider: provider, Model: model}
	return append([]models.Route{head}, chain...)
}
