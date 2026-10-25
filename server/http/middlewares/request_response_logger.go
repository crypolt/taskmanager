package middlewares

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"taskmanager/logger"
)

// RequestResponseLogger logs one line per request after it is handled.
func RequestResponseLogger(base *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		log := logger.FromContext(c.UserContext(), base)
		level := slog.LevelInfo
		if quietPath(c.Path()) {
			level = slog.LevelDebug
		}
		log.Log(c.UserContext(), level, "request",
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", c.Response().StatusCode()),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			slog.String("ip", c.IP()),
		)
		return err
	}
}

// quietPath keeps health checks and static assets out of the info log.
func quietPath(path string) bool {
	return path == "/health" || strings.HasPrefix(path, "/assets/")
}
