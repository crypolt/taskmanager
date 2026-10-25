package handlers

import (
	"github.com/gofiber/fiber/v2"

	"taskmanager/models"
	"taskmanager/utils/helpers"
)

// PreviewPlan builds a plan from context without creating tasks.
func (h *Handler) PreviewPlan(c *fiber.Ctx) error {
	var in models.PlanInput
	if err := h.bind(c, &in); err != nil {
		return err
	}
	plan, err := h.Engine.PreviewPlan(c.UserContext(), in)
	if err != nil {
		return h.respondError(c, err)
	}
	return helpers.JSONOk(c, fiber.StatusOK, plan)
}

// CreatePlan turns an approved plan into a parent task with subtasks.
func (h *Handler) CreatePlan(c *fiber.Ctx) error {
	var in models.PlanInput
	if err := h.bind(c, &in); err != nil {
		return err
	}
	task, err := h.Engine.CreateFromPlan(c.UserContext(), in)
	if err != nil {
		return h.respondError(c, err)
	}
	return helpers.JSONOk(c, fiber.StatusCreated, task)
}
