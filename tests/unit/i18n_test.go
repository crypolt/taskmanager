package unit

import (
	"context"
	"testing"

	"taskmanager/utils/helpers"
	"taskmanager/utils/i18n"
)

func langCtx(lang i18n.Lang) context.Context {
	return i18n.WithLang(context.Background(), lang)
}

func TestMessagesFollowLanguage(t *testing.T) {
	if err := i18n.Init("en"); err != nil {
		t.Fatalf("init: %v", err)
	}
	en := langCtx(i18n.EN)
	ru := langCtx(i18n.RU)

	if got := i18n.T(en, i18n.MsgTaskRetried, nil); got != "re-run requested by the user" {
		t.Fatalf("en message: %q", got)
	}
	if got := i18n.T(en, i18n.MsgTaskRetryFailed, map[string]any{"attempt": 2, "total": 3, "reason": "timeout"}); got != "attempt 2/3 failed: timeout" {
		t.Fatalf("en formatted message: %q", got)
	}
	if got := i18n.T(ru, i18n.MsgTaskRetried, nil); got != "повторный запуск по запросу пользователя" {
		t.Fatalf("ru message: %q", got)
	}
	if got := i18n.Translate(i18n.EN, i18n.MsgTaskAccepted, nil); got != "task accepted" {
		t.Fatalf("explicit language: %q", got)
	}
	if i18n.FromContext(context.Background()) != i18n.EN {
		t.Fatalf("context without a language must use the default")
	}
}

func TestDefaultLanguageHasBaseLocale(t *testing.T) {
	if err := i18n.Init("xx"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if i18n.Default() != i18n.RU {
		t.Fatalf("unsupported default must fall back to ru, got %q", i18n.Default())
	}
	if len(i18n.Languages()) < 2 {
		t.Fatalf("expected at least two languages, got %v", i18n.Languages())
	}
	if meta, ok := i18n.Meta(i18n.EN); !ok || meta.Code != "en" || meta.DateLocale == "" || meta.Dir != "ltr" {
		t.Fatalf("meta: %+v ok=%v", meta, ok)
	}
}

func TestMatchFallsBackToBaseTag(t *testing.T) {
	if lang, ok := i18n.Match("en-US"); !ok || lang != i18n.EN {
		t.Fatalf("en-US must resolve to en, got %q %v", lang, ok)
	}
	if _, ok := i18n.Match("klingon"); ok {
		t.Fatal("unknown tag must not match")
	}
	if i18n.Resolve("klingon", "en-GB") != i18n.EN {
		t.Fatal("resolve must skip unknown tags")
	}
}

func TestParseAcceptLanguageHonoursQuality(t *testing.T) {
	got := i18n.ParseAcceptLanguage("en;q=0.4, ru;q=0.9, de;q=0")
	if len(got) != 2 || got[0] != i18n.RU || got[1] != i18n.EN {
		t.Fatalf("accept-language order: %v", got)
	}
	if parsed := i18n.ParseAcceptLanguage(""); len(parsed) != 0 {
		t.Fatalf("empty header must yield no candidates, got %v", parsed)
	}
}

func TestUnknownKeyIsVisibleAndPlaceholdersSurvive(t *testing.T) {
	if err := i18n.Init("ru"); err != nil {
		t.Fatalf("init: %v", err)
	}
	ctx := langCtx(i18n.RU)

	if got := i18n.T(ctx, i18n.Key("nope"), nil); got != "nope" {
		t.Fatalf("unknown key must render as the key, got %q", got)
	}
	if i18n.Has(i18n.RU, i18n.Key("nope")) {
		t.Fatal("unknown key must not be reported as present")
	}
	if got := i18n.T(ctx, i18n.MsgTaskStageFailed, map[string]any{"stage": "review"}); got != "этап «review» завершился со статусом {status}" {
		t.Fatalf("missing arg must stay visible, got %q", got)
	}
}

func TestPluralFallsBackToWholeMessage(t *testing.T) {
	ctx := langCtx(i18n.RU)
	if got := i18n.Tn(ctx, i18n.MsgTaskPlanReady, 3, nil); got != "план готов: 3 подзадач" {
		t.Fatalf("plural fallback: %q", got)
	}
}

func TestAppErrorCarriesCodeStatusAndMessage(t *testing.T) {
	if err := i18n.Init("ru"); err != nil {
		t.Fatalf("init: %v", err)
	}

	err := helpers.ErrUnknownPipeline("nope")
	if err.Code != helpers.CodeBadRequest {
		t.Fatalf("code: %q", err.Code)
	}
	if err.HTTPStatus() != 400 {
		t.Fatalf("status: %d", err.HTTPStatus())
	}
	if got := err.Error(); got != "неизвестный конвейер nope" {
		t.Fatalf("message: %q", got)
	}

	notFound := helpers.ErrTaskNotFound
	if helpers.CodeOf(notFound) != helpers.CodeNotFound || helpers.StatusOf(notFound) != 404 {
		t.Fatalf("not found mapping: %s %d", helpers.CodeOf(notFound), helpers.StatusOf(notFound))
	}

	if _, ok := helpers.As(helpers.ErrEmptyChain); !ok {
		t.Fatal("catalogue error must be an AppError")
	}
	if helpers.CodeOf(nil) != helpers.CodeInternal {
		t.Fatal("foreign error must map to internal_error")
	}
}
