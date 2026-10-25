package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"taskmanager/internal/services/externals/mcpconfig"
	"taskmanager/models"
	"taskmanager/utils/helpers"
)

func (h *Handler) ListMCP(c *fiber.Ctx) error {
	cfg, err := h.MCP.List()
	if err != nil {
		return helpers.RespondError(c, helpers.ErrMCPConfig(err))
	}
	return helpers.JSONOk(c, fiber.StatusOK, cfg)
}

func (h *Handler) SaveMCP(c *fiber.Ctx) error {
	var in models.MCPInput
	if err := h.bind(c, &in); err != nil {
		return err
	}
	return h.upsertMCP(c, in)
}

func (h *Handler) UpdateMCP(c *fiber.Ctx) error {
	var in models.MCPInput
	if err := h.bind(c, &in); err != nil {
		return err
	}
	in.Name = c.Params("name")
	return h.upsertMCP(c, in)
}

func (h *Handler) upsertMCP(c *fiber.Ctx, in models.MCPInput) error {
	if err := h.MCP.Upsert(in); err != nil {
		if errors.Is(err, mcpconfig.ErrNameRequired) {
			return helpers.RespondError(c, helpers.ErrMCPNameRequired)
		}
		return helpers.RespondError(c, helpers.ErrMCPConfig(err))
	}
	cfg, err := h.MCP.List()
	if err != nil {
		return helpers.RespondError(c, helpers.ErrMCPConfig(err))
	}
	return helpers.JSONOk(c, fiber.StatusOK, cfg)
}

func (h *Handler) DeleteMCP(c *fiber.Ctx) error {
	if err := h.MCP.Delete(c.Params("name")); err != nil {
		if errors.Is(err, mcpconfig.ErrNotFound) {
			return helpers.RespondError(c, helpers.ErrMCPNotFound)
		}
		return helpers.RespondError(c, helpers.ErrMCPConfig(err))
	}
	cfg, err := h.MCP.List()
	if err != nil {
		return helpers.RespondError(c, helpers.ErrMCPConfig(err))
	}
	return helpers.JSONOk(c, fiber.StatusOK, cfg)
}
