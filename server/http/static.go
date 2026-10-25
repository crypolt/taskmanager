package http

import (
	"fmt"
	stdhttp "net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"

	"taskmanager/assets"
	"taskmanager/utils/i18n"
)

// renderPlaceholder builds the page served when the frontend has not been
// built. All of its text comes from the locale catalogue in the request
// language.
func renderPlaceholder(c *fiber.Ctx) string {
	ctx := c.UserContext()
	lang := i18n.FromContext(ctx)
	dir := "ltr"
	if meta, ok := i18n.Meta(lang); ok && meta.Dir != "" {
		dir = meta.Dir
	}
	title := i18n.T(ctx, i18n.StaticUINotBuiltTitle, nil)
	body := i18n.T(ctx, i18n.StaticUINotBuiltBody, nil)
	api := i18n.T(ctx, i18n.StaticUINotBuiltAPI, map[string]any{"path": "/api/v1"})
	brand := i18n.T(ctx, i18n.StaticBrand, nil)
	var b strings.Builder
	fmt.Fprintf(&b, `<!doctype html>
<html lang="%s" dir="%s"><head><meta charset="utf-8"><title>%s</title>
<style>body{font-family:-apple-system,sans-serif;background:#0e1014;color:#e8ebf0;display:grid;place-items:center;height:100vh;margin:0}
code{background:#1b202a;padding:2px 6px;border-radius:6px}</style></head>
<body><div><h1>%s</h1><p>%s <code>cd web &amp;&amp; npm install &amp;&amp; npm run build</code></p>
<p>%s</p></div></body></html>`,
		lang, dir, escapeHTML(brand), escapeHTML(title), escapeHTML(body), escapeHTML(api))
	return b.String()
}

func escapeHTML(s string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return replacer.Replace(s)
}

// RegisterStatic serves the embedded Vue build with SPA history fallback.
func RegisterStatic(app *fiber.App) {
	dist := assets.Dist()
	if dist == nil {
		app.Get("/*", func(c *fiber.Ctx) error {
			return c.Type("html").SendString(renderPlaceholder(c))
		})
		return
	}
	app.Use("/", filesystem.New(filesystem.Config{
		Root:         stdhttp.FS(dist),
		Index:        "index.html",
		NotFoundFile: "index.html",
	}))
}
