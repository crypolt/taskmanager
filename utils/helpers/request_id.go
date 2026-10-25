package helpers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type ctxKey string

const requestIDKey ctxKey = "request_id"

// EnsureRequestID returns the context with a request_id: the existing one when
// present, or a freshly generated one.
func EnsureRequestID(ctx context.Context) (context.Context, string) {
	if id := GetRequestID(ctx); id != "" {
		return ctx, id
	}
	id := NewID("req")
	return context.WithValue(ctx, requestIDKey, id), id
}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// NewID returns prefix + 12 hex chars, the project-wide identifier format.
func NewID(prefix string) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return prefix + "_000000000000"
	}
	return prefix + "_" + hex.EncodeToString(b)
}
