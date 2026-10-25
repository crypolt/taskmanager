package handlers

import (
	"github.com/gofiber/fiber/v2"

	"taskmanager/models"
	"taskmanager/utils/helpers"
)

func (h *Handler) ListTasks(c *fiber.Ctx) error {
	tasks, err := h.Engine.ListTasks(c.UserContext(), c.Query("status"))
	if err != nil {
		return h.respondError(c, err)
	}
	return helpers.JSONOk(c, fiber.StatusOK, fiber.Map{"items": tasks, "total": len(tasks)})
}

func (h *Handler) GetTask(c *fiber.Ctx) error {
	detail, err := h.Engine.Detail(c.UserContext(), c.Params("id"))
	if err != nil {
		return h.respondError(c, err)
	}
	return helpers.JSONOk(c, fiber.StatusOK, detail)
}

func (h *Handler) CreateTask(c *fiber.Ctx) error {
	var in models.CreateTaskInput
	if err := h.bind(c, &in); err != nil {
		return err
	}
	task, err := h.Engine.Create(c.UserContext(), in)
	if err != nil {
		return h.respondError(c, err)
	}
	return helpers.JSONOk(c, fiber.StatusCreated, task)
}

func (h *Handler) CancelTask(c *fiber.Ctx) error {
	if err := h.Engine.Cancel(c.UserContext(), c.Params("id")); err != nil {
		return h.respondError(c, err)
	}
	return helpers.JSONOk(c, fiber.StatusOK, fiber.Map{"ok": true})
}

func (h *Handler) RetryTask(c *fiber.Ctx) error {
	if err := h.Engine.Retry(c.UserContext(), c.Params("id")); err != nil {
		return h.respondError(c, err)
	}
	return helpers.JSONOk(c, fiber.StatusOK, fiber.Map{"ok": true})
}

func (h *Handler) MoveTask(c *fiber.Ctx) error {
	var in models.MoveTaskInput
	if err := h.bind(c, &in); err != nil {
		return err
	}
	if err := h.Engine.Move(c.UserContext(), c.Params("id"), in.Status); err != nil {
		return h.respondError(c, err)
	}
	return helpers.JSONOk(c, fiber.StatusOK, fiber.Map{"ok": true})
}

func (h *Handler) AnswerTask(c *fiber.Ctx) error {
	var in models.AnswerInput
	if err := h.bind(c, &in); err != nil {
		return err
	}
	if err := h.Engine.Answer(c.UserContext(), c.Params("id"), in.Note); err != nil {
		return h.respondError(c, err)
	}
	return helpers.JSONOk(c, fiber.StatusOK, fiber.Map{"ok": true})
}
