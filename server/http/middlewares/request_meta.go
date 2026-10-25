package middlewares

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"

	"taskmanager/logger"
	"taskmanager/utils/helpers"
)

// RequestIDHeader is the inbound/outbound header carrying the request id.
const RequestIDHeader = "X-Request-ID"

// RequestMeta issues (or reuses) a request id, stores it in the user context
// and puts a request-scoped logger into Locals.
func RequestMeta(base *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		rid := strings.TrimSpace(c.Get(RequestIDHeader))
		if rid == "" {
			rid = helpers.NewID("req")
		}
		ctx := helpers.WithRequestID(c.UserContext(), rid)
		c.SetUserContext(ctx)
		c.Locals("logger", logger.FromContext(ctx, base))
		c.Set(RequestIDHeader, rid)
		return c.Next()
	}
}
