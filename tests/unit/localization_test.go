package unit

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v2"

	"taskmanager/server/http/handlers"
	"taskmanager/server/http/middlewares"
	"taskmanager/server/http/routes"
	"taskmanager/utils/i18n"
)

// representativeKeys covers every backend namespace owned by the catalogue
// (error, task, pipeline, stage, context, static) plus the enum-driven
// task.state.* set. A missing key in any shipped locale fails the test.
var representativeKeys = []i18n.Key{
	i18n.ErrTaskNotFound,
	i18n.ErrEndpointNotFound,
	i18n.MsgTaskRetried,
	i18n.MsgPlanCreated,
	i18n.TaskStateRunning,
	i18n.PipelineDevName,
	i18n.StageAnalyze,
	i18n.MsgContextHumanNote,
	i18n.StaticUINotBuiltTitle,
}

func TestTranslatorResolvesEveryShippedLocale(t *testing.T) {
	if err := i18n.Init("ru"); err != nil {
		t.Fatalf("init: %v", err)
	}
	languages := i18n.Languages()
	if len(languages) < 2 {
		t.Fatalf("expected at least two shipped locales, got %v", languages)
	}

	for _, lang := range languages {
		lang := lang
		t.Run(string(lang), func(t *testing.T) {
			if !i18n.Supported(lang) {
				t.Fatalf("Languages() returned unsupported %q", lang)
			}
			for _, key := range representativeKeys {
				got := i18n.Translate(lang, key, nil)
				if strings.TrimSpace(got) == "" {
					t.Errorf("%s: key %q rendered an empty string", lang, key)
				}
				if got == string(key) {
					t.Errorf("%s: key %q is missing from the catalogue (rendered raw)", lang, key)
				}
			}
		})
	}
}

func TestShippedLocalesRenderDifferentText(t *testing.T) {
	if err := i18n.Init("ru"); err != nil {
		t.Fatalf("init: %v", err)
	}
	// A copy-paste that leaves the base text in another locale must fail: for
	// these keys the shipped translations are genuinely different sentences.
	keys := []i18n.Key{i18n.ErrTaskNotFound, i18n.MsgTaskRetried, i18n.StageAnalyze, i18n.StaticUINotBuiltTitle}
	for _, key := range keys {
		ru := i18n.Translate(i18n.RU, key, nil)
		en := i18n.Translate(i18n.EN, key, nil)
		if ru == en {
			t.Errorf("key %q renders identically in ru and en (%q) — translation is missing", key, ru)
		}
	}
}

// TestUnsupportedLanguageFallsBackToDefaultLocale proves the resolution chain:
// a language that is not in the catalogue (or only a region of one) resolves
// through the deployment default, which itself is the base locale.
func TestUnsupportedLanguageFallsBackToDefaultLocale(t *testing.T) {
	if err := i18n.Init("ru"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if i18n.Default() != i18n.RU {
		t.Fatalf("default: %q, want ru", i18n.Default())
	}
	base := i18n.Translate(i18n.RU, i18n.ErrTaskNotFound, nil)

	for _, unsupported := range []i18n.Lang{"de", "de-AT", "klingon", "zz-ZZ"} {
		got := i18n.Translate(unsupported, i18n.ErrTaskNotFound, nil)
		if got == string(i18n.ErrTaskNotFound) {
			t.Errorf("%q: unsupported language rendered the raw key", unsupported)
		}
		if got != base {
			t.Errorf("%q: unsupported language = %q, want the default-locale text %q", unsupported, got, base)
		}
		if !i18n.Has(unsupported, i18n.ErrTaskNotFound) {
			t.Errorf("%q: Has must resolve through the fallback chain", unsupported)
		}
	}

	// A key absent from every locale stays visible as the key itself: the
	// fallback chain never yields an empty message.
	if got := i18n.Translate(i18n.EN, i18n.Key("error.doesNotExist"), nil); got != "error.doesNotExist" {
		t.Errorf("unknown key must render as itself, got %q", got)
	}
}

// TestDefaultLanguageIsUsedWithoutContextData guards the entry point used by
// background work (queue consumers, startup): no request context must not mean
// no language.
func TestDefaultLanguageIsUsedWithoutContextData(t *testing.T) {
	if err := i18n.Init("ru"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if got, want := i18n.FromContext(context.Background()), i18n.RU; got != want {
		t.Fatalf("FromContext(background) = %q, want %q", got, want)
	}
	if got := i18n.T(context.Background(), i18n.MsgTaskRetried, nil); got != i18n.Translate(i18n.RU, i18n.MsgTaskRetried, nil) {
		t.Fatalf("T without a language must render the default locale, got %q", got)
	}
}

// TestConcurrentTranslationKeepsLanguagesIsolated exercises the shared,
// atomically swapped catalogue: parallel request handlers must never see the
// other request's language. Run with -race to make the isolation meaningful.
func TestConcurrentTranslationKeepsLanguagesIsolated(t *testing.T) {
	if err := i18n.Init("ru"); err != nil {
		t.Fatalf("init: %v", err)
	}
	wantEN := i18n.Translate(i18n.EN, i18n.MsgTaskRetried, nil)
	wantRU := i18n.Translate(i18n.RU, i18n.MsgTaskRetried, nil)

	var wg sync.WaitGroup
	for index := 0; index < 64; index++ {
		index := index
		wg.Add(1)
		go func() {
			defer wg.Done()
			lang, want := i18n.EN, wantEN
			if index%2 == 0 {
				lang, want = i18n.RU, wantRU
			}
			ctx := i18n.WithLang(context.Background(), lang)
			for repeat := 0; repeat < 50; repeat++ {
				if got := i18n.T(ctx, i18n.MsgTaskRetried, nil); got != want {
					t.Errorf("lang %s: got %q, want %q", lang, got, want)
					return
				}
				if got := i18n.Translate(lang, i18n.MsgTaskRetried, nil); got != want {
					t.Errorf("Translate(%s): got %q, want %q", lang, got, want)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestParseAcceptLanguageHandlesBoundaryInput(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   []i18n.Lang
	}{
		{name: "empty", header: ""},
		{name: "wildcard only", header: "*"},
		{name: "malformed q keeps the candidate", header: "en;q=abc", want: []i18n.Lang{i18n.EN}},
		{name: "zero quality drops the tag", header: "en;q=0", want: nil},
		{name: "underscore normalisation", header: "ru_RU", want: []i18n.Lang{i18n.RU}},
		{name: "region falls back to base", header: "en-GB;q=0.9", want: []i18n.Lang{i18n.EN}},
		{name: "unknown tags are skipped", header: "de, klingon", want: nil},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := i18n.ParseAcceptLanguage(tc.header)
			if len(got) != len(tc.want) {
				t.Fatalf("ParseAcceptLanguage(%q) = %v, want %v", tc.header, got, tc.want)
			}
			for index := range got {
				if got[index] != tc.want[index] {
					t.Fatalf("ParseAcceptLanguage(%q) = %v, want %v", tc.header, got, tc.want)
				}
			}
		})
	}
}

// newLocalizedAPI builds the production route table behind the real Locale
// middleware. The engine is intentionally nil: the covered endpoints fail
// before reaching it (body parsing and the API catch-all route).
func newLocalizedAPI(t *testing.T) *fiber.App {
	t.Helper()
	if err := i18n.Init("ru"); err != nil {
		t.Fatalf("init: %v", err)
	}
	app := fiber.New()
	app.Use(middlewares.Locale(i18n.RU))
	routes.Register(app, &handlers.Handler{Log: slog.New(slog.NewTextHandler(io.Discard, nil))})
	return app
}

// TestHandlerErrorBodyFollowsRequestedLanguage is the end-to-end language
// switch: the same real route, the same handler, different request language —
// the rendered human message changes while the machine-readable identity stays
// stable.
func TestHandlerErrorBodyFollowsRequestedLanguage(t *testing.T) {
	app := newLocalizedAPI(t)

	cases := []struct {
		name     string
		method   string
		target   string
		header   map[string]string
		body     string
		wantLang string
		wantCode string
		wantKey  string
		wantText string
		notText  string
	}{
		{
			name:     "default locale",
			method:   "POST",
			target:   "/api/v1/tasks",
			body:     "{}",
			wantLang: "ru",
			wantCode: "bad_request",
			wantKey:  "error.titleRequired",
			wantText: "укажите название задачи",
			notText:  "task title is required",
		},
		{
			name:     "query selects english",
			method:   "POST",
			target:   "/api/v1/tasks?lang=en",
			body:     "{}",
			wantLang: "en",
			wantCode: "bad_request",
			wantKey:  "error.titleRequired",
			wantText: "task title is required",
			notText:  "укажите название задачи",
		},
		{
			name:     "error arguments render in the request language",
			method:   "POST",
			target:   "/api/v1/tasks?lang=en",
			body:     `{"title":"x","pipeline":"nope"}`,
			wantLang: "en",
			wantCode: "bad_request",
			wantKey:  "error.unknownPipeline",
			wantText: "unknown pipeline nope",
			notText:  "неизвестный конвейер",
		},
		{
			name:     "header selects english",
			method:   "POST",
			target:   "/api/v1/tasks",
			header:   map[string]string{"X-Locale": "en"},
			body:     "{}",
			wantLang: "en",
			wantCode: "bad_request",
			wantKey:  "error.titleRequired",
			wantText: "task title is required",
			notText:  "укажите название задачи",
		},
		{
			name:     "unknown language falls back to the deployment default",
			method:   "GET",
			target:   "/api/v1/does-not-exist?lang=klingon",
			wantLang: "ru",
			wantCode: "not_found",
			wantKey:  "error.endpointNotFound",
			wantText: "эндпоинт не найден",
			notText:  "endpoint not found",
		},
		{
			name:     "catch-all route renders english",
			method:   "GET",
			target:   "/api/v1/does-not-exist?lang=en",
			wantLang: "en",
			wantCode: "not_found",
			wantKey:  "error.endpointNotFound",
			wantText: "endpoint not found",
			notText:  "эндпоинт не найден",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			for name, value := range tc.header {
				req.Header.Set(name, value)
			}
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer resp.Body.Close()

			if got := resp.Header.Get("Content-Language"); got != tc.wantLang {
				t.Fatalf("Content-Language = %q, want %q", got, tc.wantLang)
			}
			raw, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			payload := string(raw)
			for _, want := range []string{`"code":"` + tc.wantCode + `"`, `"key":"` + tc.wantKey + `"`, tc.wantText} {
				if !strings.Contains(payload, want) {
					t.Errorf("body %s must contain %s", payload, want)
				}
			}
			if tc.notText != "" && strings.Contains(payload, tc.notText) {
				t.Errorf("body %s must not contain the other locale's text %q", payload, tc.notText)
			}
		})
	}
}

func TestHandlerVaryHeaderCoversLanguageInputs(t *testing.T) {
	app := newLocalizedAPI(t)
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()
	if got := resp.Header.Get("Vary"); got != "Accept-Language, X-Locale" {
		t.Fatalf("Vary = %q, want %q", got, "Accept-Language, X-Locale")
	}
}
