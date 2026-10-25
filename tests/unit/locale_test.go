package unit

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"taskmanager/server/http/middlewares"
	"taskmanager/utils/i18n"
)

func newLocaleApp(t *testing.T) *fiber.App {
	t.Helper()
	if err := i18n.Init("ru"); err != nil {
		t.Fatalf("init: %v", err)
	}
	app := fiber.New()
	app.Use(middlewares.Locale(i18n.RU))
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"lang":  string(i18n.FromContext(c.UserContext())),
			"retry": i18n.T(c.UserContext(), i18n.MsgTaskRetried, nil),
		})
	})
	return app
}

func TestLocaleMiddlewareResolutionOrder(t *testing.T) {
	app := newLocaleApp(t)

	cases := []struct {
		name   string
		query  string
		header map[string]string
		cookie string
		want   string
		text   string
	}{
		{name: "default", want: "ru", text: "повторный запуск по запросу пользователя"},
		{name: "query lang", query: "?lang=en", want: "en", text: "re-run requested by the user"},
		{name: "query locale", query: "?locale=en-GB", want: "en", text: "re-run requested by the user"},
		{name: "header", header: map[string]string{"X-Locale": "en"}, want: "en", text: "re-run requested by the user"},
		{name: "cookie", cookie: "lang=en", want: "en", text: "re-run requested by the user"},
		{name: "accept-language", header: map[string]string{"Accept-Language": "en-US,en;q=0.9"}, want: "en", text: "re-run requested by the user"},
		{name: "unknown is skipped", query: "?lang=klingon", want: "ru", text: "повторный запуск по запросу пользователя"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/"+tc.query, nil)
			for name, value := range tc.header {
				req.Header.Set(name, value)
			}
			if tc.cookie != "" {
				req.Header.Set("Cookie", tc.cookie)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			if got := resp.Header.Get("Content-Language"); got != tc.want {
				t.Fatalf("Content-Language = %q, want %q", got, tc.want)
			}
			if got := resp.Header.Get("Vary"); got != "Accept-Language, X-Locale" {
				t.Fatalf("Vary = %q", got)
			}
			raw, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			body := string(raw)
			if want := `"lang":"` + tc.want + `"`; !strings.Contains(body, want) {
				t.Fatalf("body %s must contain %s", body, want)
			}
			if !strings.Contains(body, tc.text) {
				t.Fatalf("body %s must contain %q", body, tc.text)
			}
		})
	}
}
