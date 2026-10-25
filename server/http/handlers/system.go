package handlers

import (
	"github.com/gofiber/fiber/v2"

	"taskmanager/utils/helpers"
)

func (h *Handler) Health(c *fiber.Ctx) error {
	return helpers.JSONOk(c, fiber.StatusOK, fiber.Map{"status": "ok"})
}

func (h *Handler) Stats(c *fiber.Ctx) error {
	stats, err := h.Engine.Stats(c.UserContext())
	if err != nil {
		return h.respondError(c, err)
	}
	return helpers.JSONOk(c, fiber.StatusOK, stats)
}

func (h *Handler) Pipelines(c *fiber.Ctx) error {
	pipelines := h.Engine.Pipelines(c.UserContext())
	return helpers.JSONOk(c, fiber.StatusOK, fiber.Map{"items": pipelines, "total": len(pipelines)})
}
