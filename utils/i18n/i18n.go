// Package i18n keeps every user-facing string of the backend in JSON locale
// files under locales/. Messages are addressed by a flat dot-path key and
// rendered in the language carried by the request context.
//
// Adding a language requires no code changes: drop a <code>.json file into
// locales/ with the full backend key set, a _meta block and the "base" flag
// left unset (exactly one catalogue file declares "base": true). This package
// owns the backend namespaces only: error, task, pipeline, stage, context
// and static — the SPA renders its own catalogue from web/src/locales.
package i18n

import "context"

// Init loads the embedded catalogue and sets the default language. It must be
// called once during startup; a broken catalogue is a startup error.
func Init(defaultLang string) error {
	c, err := loadCatalog(Lang(normalizeTag(defaultLang)))
	if err != nil {
		return err
	}
	active.Store(c)
	return nil
}

// Default returns the language used when a request carries none.
func Default() Lang {
	return catalogOrLoad().defaultLang
}

// Base returns the language whose locale file carries "base": true in _meta.
// It is the fallback for partially translated languages and the reference
// catalogue full translations are validated against.
func Base() Lang {
	return catalogOrLoad().baseLang
}

// Languages lists the shipped languages ordered by _meta.order.
func Languages() []Lang {
	c := catalogOrLoad()
	out := make([]Lang, len(c.langs))
	copy(out, c.langs)
	return out
}

// Meta returns the _meta block of a language (base tag accepted).
func Meta(lang Lang) (LocaleMeta, bool) {
	c := catalogOrLoad()
	if meta, ok := c.meta[lang]; ok {
		return meta, true
	}
	if matched, ok := Match(string(lang)); ok {
		return c.meta[matched], true
	}
	return LocaleMeta{}, false
}

// Supported reports whether a language is present in the catalogue.
func Supported(lang Lang) bool {
	_, ok := Meta(lang)
	return ok
}

// T renders a key in the language stored in ctx.
func T(ctx context.Context, key Key, args map[string]any) string {
	return catalogOrLoad().translate(FromContext(ctx), key, args)
}

// Tn renders a plural node, selecting the CLDR category for n and exposing it
// to the message as {count}.
func Tn(ctx context.Context, key Key, n int, args map[string]any) string {
	return catalogOrLoad().translatePlural(FromContext(ctx), key, n, args)
}

// Translate renders a key in an explicit language, ignoring the context.
func Translate(lang Lang, key Key, args map[string]any) string {
	if matched, ok := Match(string(lang)); ok {
		lang = matched
	}
	return catalogOrLoad().translate(lang, key, args)
}

// Has reports whether a key exists for a language (following the fallback
// chain, so it also answers for a partially translated locale).
func Has(lang Lang, key Key) bool {
	if matched, ok := Match(string(lang)); ok {
		lang = matched
	}
	_, ok := catalogOrLoad().lookup(lang, key)
	return ok
}
