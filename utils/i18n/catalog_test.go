package i18n

import "testing"

func TestLocaleFilesHaveTheSameKeys(t *testing.T) {
	c, err := loadCatalog(RU)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	base, ok := c.messages[RU]
	if !ok {
		t.Fatal("base locale is missing")
	}
	for _, lang := range c.langs {
		if lang == RU {
			continue
		}
		messages := c.messages[lang]
		for key := range base {
			if _, ok := messages[key]; !ok {
				t.Errorf("locale %s is missing key %q", lang, key)
			}
		}
		for key := range messages {
			if _, ok := base[key]; !ok {
				t.Errorf("locale %s has an extra key %q", lang, key)
			}
		}
	}
}

func TestEveryCatalogKeyIsTranslated(t *testing.T) {
	c, err := loadCatalog(RU)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for lang, messages := range c.messages {
		for key, template := range messages {
			if template == "" {
				t.Errorf("locale %s: key %q has an empty text", lang, key)
			}
		}
	}
}

func TestPluralCategory(t *testing.T) {
	cases := []struct {
		lang Lang
		n    int
		want string
	}{
		{RU, 1, "one"},
		{RU, 2, "few"},
		{RU, 5, "many"},
		{RU, 11, "many"},
		{RU, 21, "one"},
		{EN, 1, "one"},
		{EN, 2, "other"},
		{Lang("pl"), 3, "few"},
		{Lang("cs"), 4, "few"},
	}
	for _, tc := range cases {
		if got := pluralCategory(tc.lang, tc.n); got != tc.want {
			t.Errorf("pluralCategory(%s, %d) = %q, want %q", tc.lang, tc.n, got, tc.want)
		}
	}
}

func TestExactlyOneBaseLocale(t *testing.T) {
	c, err := loadCatalog("")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c.baseLang != RU {
		t.Fatalf("base locale must be ru, got %q", c.baseLang)
	}
	if !c.meta[c.baseLang].Base {
		t.Fatal("base locale _meta must carry base:true")
	}
	if meta, ok := c.meta[EN]; ok && meta.Base {
		t.Fatal("en must not claim the base flag")
	}
	if Base() != RU {
		t.Fatalf("Base() = %q, want %q", Base(), RU)
	}
}

func TestEveryTemplateMatchesItsPlaceholders(t *testing.T) {
	c, err := loadCatalog("")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	base := c.messages[c.baseLang]
	for lang, messages := range c.messages {
		if lang == c.baseLang {
			continue
		}
		for key, template := range messages {
			baseTemplate, ok := base[key]
			if !ok {
				t.Errorf("locale %s has an unknown key %q", lang, key)
				continue
			}
			if placeholders(baseTemplate) == nil && placeholders(template) != nil ||
				placeholders(baseTemplate) != nil && placeholders(template) == nil {
				t.Errorf("locale %s: placeholder set differs for %q (%q vs %q)",
					lang, key, baseTemplate, template)
			}
		}
	}
}

// TestMissingKeyFallsBackThroughTheDefaultLocale pins the per-key resolution
// chain on a catalogue that is deliberately incomplete. The shipped files are
// key-parity enforced (TestLocaleFilesHaveTheSameKeys), so "a key translated in
// the base but missing from the requested language" is only observable on a
// synthetic catalogue. If chain() lost the default/base links, a partially
// translated language would render raw keys instead of falling back.
func TestMissingKeyFallsBackThroughTheDefaultLocale(t *testing.T) {
	t.Parallel()

	const (
		baseOnly    Key = "task.baseOnly"    // translated only in the base locale
		bothLocales Key = "task.bothLocales" // translated in the base and in the requested locale
		nowhere     Key = "task.nowhere"     // absent from every locale
	)

	newCatalog := func(defaultLang Lang) *catalog {
		return &catalog{
			baseLang:    RU,
			defaultLang: defaultLang,
			meta:        map[Lang]LocaleMeta{},
			byTag:       map[string]Lang{},
			messages: map[Lang]map[Key]string{
				RU: {baseOnly: "база", bothLocales: "база"},
				EN: {bothLocales: "requested"},
			},
		}
	}

	cases := []struct {
		name        string
		defaultLang Lang
		lang        Lang
		key         Key
		want        string
	}{
		{
			name:        "key missing from the requested locale renders the base text",
			defaultLang: RU,
			lang:        EN,
			key:         baseOnly,
			want:        "база",
		},
		{
			name:        "a key present in the requested locale wins over the base",
			defaultLang: RU,
			lang:        EN,
			key:         bothLocales,
			want:        "requested",
		},
		{
			name:        "regional tag resolves through its base language",
			defaultLang: RU,
			lang:        Lang("en-GB"),
			key:         bothLocales,
			want:        "requested",
		},
		{
			name:        "unsupported language resolves through the default locale",
			defaultLang: EN,
			lang:        Lang("de"),
			key:         bothLocales,
			want:        "requested",
		},
		{
			name:        "the catalogue base is the last link when the default lacks the key",
			defaultLang: EN,
			lang:        Lang("de"),
			key:         baseOnly,
			want:        "база",
		},
		{
			name:        "a key absent from every locale stays visible as its key",
			defaultLang: RU,
			lang:        EN,
			key:         nowhere,
			want:        string(nowhere),
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := newCatalog(tc.defaultLang)
			if got := c.translate(tc.lang, tc.key, nil); got != tc.want {
				t.Fatalf("translate(%q, %q) = %q, want %q", tc.lang, tc.key, got, tc.want)
			}
		})
	}
}

func placeholders(template string) map[string]bool {
	set := map[string]bool{}
	for _, match := range placeholderPattern.FindAllStringSubmatch(template, -1) {
		set[match[1]] = true
	}
	if len(set) == 0 {
		return nil
	}
	return set
}
