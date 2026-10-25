package server

import (
	"context"

	"go.uber.org/fx"

	"taskmanager/internal/services/internals"
)

// RegisterEngine ties the task engine to the application lifecycle.
func RegisterEngine(lc fx.Lifecycle, engine *internals.Engine) {
	var cancel context.CancelFunc
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			ctx, stop := context.WithCancel(context.Background())
			cancel = stop
			return engine.Start(ctx)
		},
		OnStop: func(context.Context) error {
			if cancel != nil {
				cancel()
			}
			engine.Stop()
			return nil
		},
	})
}
