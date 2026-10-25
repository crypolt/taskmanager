package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"taskmanager/internal/repository"
	"taskmanager/models"
	"taskmanager/utils/helpers"
)

func (h *Handler) ListModels(c *fiber.Ctx) error {
	modelsList, err := h.Engine.Models(c.UserContext())
	if err != nil {
		return h.respondError(c, err)
	}
	return helpers.JSONOk(c, fiber.StatusOK, modelsList)
}

func (h *Handler) SetModelRoute(c *fiber.Ctx) error {
	var in models.SetRouteInput
	if err := h.bind(c, &in); err != nil {
		return err
	}
	if in.Provider == "" {
		in.Provider = "opencode"
	}
	if err := h.Engine.SetRoute(c.UserContext(), in); err != nil {
		return h.respondError(c, err)
	}
	return helpers.JSONOk(c, fiber.StatusOK, fiber.Map{"ok": true})
}

func (h *Handler) ModelUsage(c *fiber.Ctx) error {
	usage, err := h.Engine.ModelUsage(c.UserContext())
	if err != nil {
		return h.respondError(c, err)
	}
	return helpers.JSONOk(c, fiber.StatusOK, usage)
}

func (h *Handler) ModelUsageDaily(c *fiber.Ctx) error {
	usage, err := h.Engine.DailyModelUsage(c.UserContext())
	if err != nil {
		return h.respondError(c, err)
	}
	return helpers.JSONOk(c, fiber.StatusOK, usage)
}

func (h *Handler) ModelHealth(c *fiber.Ctx) error {
	health, err := h.Engine.ModelHealth(c.UserContext())
	if err != nil {
		return h.respondError(c, err)
	}
	return helpers.JSONOk(c, fiber.StatusOK, health)
}

func (h *Handler) DeleteModelEntry(c *fiber.Ctx) error {
	var in models.DeleteRouteInput
	if err := h.bind(c, &in); err != nil {
		return err
	}
	if in.Role == "" || in.Position < 0 {
		return helpers.RespondError(c, helpers.ErrInvalidRoute)
	}
	if err := h.Engine.DeleteRoute(c.UserContext(), in.Role, in.Position); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helpers.RespondError(c, helpers.ErrRouteNotFound)
		}
		return h.respondError(c, err)
	}
	return helpers.JSONOk(c, fiber.StatusOK, fiber.Map{"ok": true})
}

func (h *Handler) MoveModelRoute(c *fiber.Ctx) error {
	var in models.MoveRouteInput
	if err := h.bind(c, &in); err != nil {
		return err
	}
	if in.Role == "" || in.From < 0 || in.To < 0 || in.From == in.To {
		return helpers.RespondError(c, helpers.ErrInvalidRoute)
	}
	if err := h.Engine.MoveRoute(c.UserContext(), in); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helpers.RespondError(c, helpers.ErrRouteNotFound)
		}
		return h.respondError(c, err)
	}
	return helpers.JSONOk(c, fiber.StatusOK, fiber.Map{"ok": true})
}
