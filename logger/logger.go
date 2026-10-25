package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"taskmanager/config"
	"taskmanager/utils/helpers"
)

// InitLogger is the single logger provider of the service (structured JSON on
// stdout). Attrs passed to slog are emitted as flat fields, so the output is
// greppable and every request-scoped line carries request_id.
func InitLogger(cfg *config.Config) (*slog.Logger, error) {
	level := parseLevel(cfg.Server.LogLevel)
	if cfg.Server.TraceMode {
		level = slog.LevelDebug
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler), nil
}

func parseLevel(value string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug
	case "warning", "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// FromContext returns the logger enriched with the request_id from ctx.
func FromContext(ctx context.Context, base *slog.Logger) *slog.Logger {
	if base == nil {
		base = slog.Default()
	}
	if id := helpers.GetRequestID(ctx); id != "" {
		return base.With(slog.String("request_id", id))
	}
	return base
}

// WithContext returns attrs with request_id prepended, ready for a log call:
//
//	logger.FromContext(ctx, log).Info("msg", logger.WithContext(ctx, slog.String("k", "v"))...)
func WithContext(ctx context.Context, attrs ...any) []any {
	if id := helpers.GetRequestID(ctx); id != "" {
		return append([]any{slog.String("request_id", id)}, attrs...)
	}
	return attrs
}
