package http

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/fx"

	"taskmanager/config"
	"taskmanager/server/http/handlers"
	"taskmanager/server/http/middlewares"
	"taskmanager/server/http/routes"
	"taskmanager/utils/helpers"
	"taskmanager/utils/i18n"
)

type Server struct {
	app *fiber.App
	cfg *config.Config
	log *slog.Logger
}

func NewServer(lc fx.Lifecycle, cfg *config.Config, log *slog.Logger, h *handlers.Handler) *Server {
	app := fiber.New(fiber.Config{
		AppName:               cfg.ServiceName,
		DisableStartupMessage: true,
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          0,
		IdleTimeout:           120 * time.Second,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			if appErr, ok := helpers.As(err); ok {
				return helpers.RespondError(c, appErr)
			}
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				switch fiberErr.Code {
				case fiber.StatusNotFound:
					return helpers.RespondError(c, helpers.ErrEndpointNotFound)
				case fiber.StatusMethodNotAllowed:
					return helpers.RespondError(c, helpers.ErrMethodNotAllowed)
				}
			}
			log.Error("http_error", slog.Any("err", err), slog.String("path", c.Path()))
			return helpers.RespondError(c, helpers.New(helpers.CodeInternal, i18n.ErrInternal, nil))
		},
	})

	app.Use(recover.New())
	app.Use(middlewares.RequestMeta(log))
	app.Use(middlewares.Locale(i18n.Lang(cfg.Language)))
	app.Use(middlewares.Panic(log))
	app.Use(middlewares.RequestResponseLogger(log))

	routes.Register(app, h)
	RegisterStatic(app)

	s := &Server{app: app, cfg: cfg, log: log}
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				log.Info("http_server_started", "address", cfg.Address())
				if err := app.Listen(cfg.Address()); err != nil {
					log.Error("http_server_stopped", "err", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return app.ShutdownWithContext(ctx)
		},
	})
	return s
}
