package i18n_test

import (
	"strings"
	"testing"

	"taskmanager/models"
	"taskmanager/utils/i18n"
)

// TestStatusesHaveBackendKeys guards the enum completeness invariant from the
// localization design: every task status the API can emit has a
// task.state.<code> key in every shipped language.
func TestStatusesHaveBackendKeys(t *testing.T) {
	if err := i18n.Init("ru"); err != nil {
		t.Fatalf("init: %v", err)
	}
	statuses := models.AllStatuses()
	if len(statuses) == 0 {
		t.Fatal("AllStatuses must enumerate the status enum")
	}
	for _, status := range statuses {
		key := models.StatusKey(status)
		if key == "" {
			t.Fatalf("status %q has no task.state key", status)
		}
		for _, lang := range i18n.Languages() {
			if !i18n.Has(lang, key) {
				t.Errorf("locale %s is missing status key %q", lang, key)
			}
		}
	}
	if i18n.Has(i18n.RU, i18n.Key("task.state.missing")) {
		t.Fatal("an invented status must not appear in the catalogue")
	}
}

// TestAPIErrorTemplatesCarryNoCausePlaceholder checks the error contract: API
// error templates keep the technical cause outside the sentence (Detail), so
// none of them may demand {reason}/{cause} to render a complete sentence.
func TestAPIErrorTemplatesCarryNoCausePlaceholder(t *testing.T) {
	if err := i18n.Init("ru"); err != nil {
		t.Fatalf("init: %v", err)
	}
	keys := []i18n.Key{
		i18n.ErrVerification, i18n.ErrPlanFailed, i18n.ErrMCPConfig,
	}
	for _, lang := range i18n.Languages() {
		for _, key := range keys {
			rendered := i18n.Translate(lang, key, nil)
			if rendered == string(key) {
				t.Fatalf("template %q in %q is missing from the catalogue", key, lang)
			}
			for _, banned := range []string{"{reason}", "{cause}"} {
				if strings.Contains(rendered, banned) {
					t.Errorf("template %q in %q still carries %q: %q", key, lang, banned, rendered)
				}
			}
		}
	}
}
