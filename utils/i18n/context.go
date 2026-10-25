package i18n

import (
	"context"
)

type langKey struct{}

// WithLang stores the resolved request language in the context.
func WithLang(ctx context.Context, lang Lang) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if lang == "" {
		return ctx
	}
	return context.WithValue(ctx, langKey{}, lang)
}

// FromContext returns the request language, or the default language when the
// context carries none (background jobs, tests, startup).
func FromContext(ctx context.Context) Lang {
	if ctx != nil {
		if lang, ok := ctx.Value(langKey{}).(Lang); ok && lang != "" {
			return lang
		}
	}
	return Default()
}
