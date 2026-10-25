package middlewares

import (
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"

	"taskmanager/logger"
	"taskmanager/utils/helpers"
	"taskmanager/utils/i18n"
)

// Panic recovers handler panics and answers with a localized JSON error.
func Panic(base *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if recovered := recover(); recovered != nil {
				ctx := c.UserContext()
				logger.FromContext(ctx, base).Error("panic",
					slog.Any("error", fmt.Sprint(recovered)),
					slog.String("stack", string(debug.Stack())),
					slog.String("path", c.Path()),
				)
				_ = helpers.RespondError(c, helpers.New(helpers.CodeInternal, i18n.ErrInternal, nil))
			}
		}()
		return c.Next()
	}
}
