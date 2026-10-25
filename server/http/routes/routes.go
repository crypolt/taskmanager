package routes

import (
	"github.com/gofiber/fiber/v2"

	"taskmanager/server/http/handlers"
	"taskmanager/utils/helpers"
)

// Register wires every HTTP endpoint. The API lives under /api/v1; static
// assets and the SPA fallback are registered by the server after this.
func Register(app *fiber.App, h *handlers.Handler) {
	api := app.Group("/api/v1")

	api.Get("/health", h.Health)
	api.Get("/stats", h.Stats)
	api.Get("/pipelines", h.Pipelines)

	api.Get("/tasks", h.ListTasks)
	api.Post("/tasks", h.CreateTask)
	api.Get("/tasks/:id", h.GetTask)
	api.Post("/tasks/:id/cancel", h.CancelTask)
	api.Post("/tasks/:id/retry", h.RetryTask)
	api.Post("/tasks/:id/answer", h.AnswerTask)
	api.Post("/tasks/:id/move", h.MoveTask)

	api.Post("/plans/preview", h.PreviewPlan)
	api.Post("/plans", h.CreatePlan)

	api.Get("/models", h.ListModels)
	api.Get("/models/usage", h.ModelUsage)
	api.Get("/models/usage/daily", h.ModelUsageDaily)
	api.Get("/models/health", h.ModelHealth)
	api.Post("/models", h.SetModelRoute)
	api.Post("/models/entries", h.DeleteModelEntry)
	api.Post("/models/move", h.MoveModelRoute)
	api.Get("/mcp", h.ListMCP)
	api.Post("/mcp", h.SaveMCP)
	api.Put("/mcp/:name", h.UpdateMCP)
	api.Delete("/mcp/:name", h.DeleteMCP)

	api.Get("/events", h.Events)

	app.All("/api/*", func(c *fiber.Ctx) error {
		return helpers.RespondError(c, helpers.ErrEndpointNotFound)
	})
}
