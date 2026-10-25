package handlers

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"taskmanager/internal/repository"
	"taskmanager/internal/services/externals/mcpconfig"
	"taskmanager/internal/services/internals"
	"taskmanager/utils/helpers"
)

// Handler holds the application services used by HTTP endpoints.
type Handler struct {
	Engine *internals.Engine
	Log    *slog.Logger
	MCP    *mcpconfig.Client
}

func New(engine *internals.Engine, log *slog.Logger, mcp *mcpconfig.Client) *Handler {
	return &Handler{Engine: engine, Log: log, MCP: mcp}
}

func (h *Handler) bind(c *fiber.Ctx, dst any) error {
	if err := c.BodyParser(dst); err != nil {
		return helpers.RespondError(c, helpers.ErrInvalidJSON)
	}
	return nil
}

func (h *Handler) respondError(c *fiber.Ctx, err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return helpers.RespondError(c, helpers.ErrTaskNotFound)
	}
	if _, ok := helpers.As(err); !ok {
		h.Log.Error("handler_error", "err", err)
	}
	return helpers.RespondError(c, err)
}
