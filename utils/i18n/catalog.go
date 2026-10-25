package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// Locale files are embedded at build time. Adding a language means dropping a
// <code>.json file into locales/ — no Go code, config or migration changes.
//
//go:embed locales/*.json
var localeFiles embed.FS

// LocaleMeta is the "_meta" block of a language file. It drives the language
// switcher, Intl formatting and the document direction. Exactly one file of
// the catalogue sets Base: it is the fallback for partially translated
// languages and the source the missing-key diagnostics render against.
type LocaleMeta struct {
	Code       string `json:"code"`
	Label      string `json:"label"`
	DateLocale string `json:"dateLocale"`
	Dir        string `json:"dir"`
	Order      int    `json:"order"`
	Base       bool   `json:"base"`
}

type catalog struct {
	baseLang    Lang
	defaultLang Lang
	strict      bool
	langs       []Lang
	meta        map[Lang]LocaleMeta
	byTag       map[string]Lang
	messages    map[Lang]map[Key]string
}

var (
	active   atomic.Pointer[catalog]
	loadOnce sync.Once
	loadErr  error
	warned   sync.Map
)

var placeholderPattern = regexp.MustCompile(`\{([A-Za-z0-9_]+)\}`)

// catalogOrLoad returns the active catalogue. The i18n package is usable
// before Init (tests, package-level errors): the embedded catalogue is loaded
// lazily with the base language.
func catalogOrLoad() *catalog {
	if c := active.Load(); c != nil {
		return c
	}
	loadOnce.Do(func() {
		c, err := loadCatalog("")
		if err != nil {
			loadErr = err
			return
		}
		active.Store(c)
	})
	if c := active.Load(); c != nil {
		return c
	}
	// A broken embedded catalogue is a build-time defect, never a runtime
	// condition: fail loudly instead of serving raw keys.
	panic(fmt.Sprintf("i18n: cannot load locale catalogue: %v", loadErr))
}

func loadCatalog(defaultLang Lang) (*catalog, error) {
	entries, err := fs.ReadDir(localeFiles, "locales")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("no locale files embedded")
	}

	c := &catalog{
		strict:   os.Getenv("I18N_STRICT") == "1",
		meta:     make(map[Lang]LocaleMeta, len(names)),
		byTag:    make(map[string]Lang, len(names)),
		messages: make(map[Lang]map[Key]string, len(names)),
	}
	for _, name := range names {
		data, err := localeFiles.ReadFile(path.Join("locales", name))
		if err != nil {
			return nil, err
		}
		meta, messages, err := parseLocale(name, data)
		if err != nil {
			return nil, err
		}
		lang := Lang(meta.Code)
		c.meta[lang] = meta
		c.messages[lang] = messages
		c.byTag[normalizeTag(meta.Code)] = lang
		if meta.Base {
			if c.baseLang != "" {
				return nil, fmt.Errorf("locale %s: _meta.base conflicts with %s, only one locale may be the base", name, c.baseLang)
			}
			c.baseLang = lang
		}
	}
	if c.baseLang == "" {
		return nil, fmt.Errorf(`exactly one locale file must declare "base": true in _meta`)
	}
	c.defaultLang = c.resolveDefault(defaultLang)
	c.langs = make([]Lang, 0, len(c.meta))
	for lang := range c.meta {
		c.langs = append(c.langs, lang)
	}
	sort.Slice(c.langs, func(i, j int) bool {
		left, right := c.meta[c.langs[i]], c.meta[c.langs[j]]
		if left.Order != right.Order {
			return left.Order < right.Order
		}
		return left.Code < right.Code
	})
	return c, nil
}

func (c *catalog) resolveDefault(lang Lang) Lang {
	normalized := normalizeTag(string(lang))
	if match, ok := c.byTag[normalized]; ok {
		return match
	}
	if match, ok := c.byTag[baseTag(normalized)]; ok {
		return match
	}
	return c.baseLang
}

func parseLocale(name string, data []byte) (LocaleMeta, map[Key]string, error) {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return LocaleMeta{}, nil, fmt.Errorf("locale %s: %w", name, err)
	}
	rawMeta, ok := root["_meta"].(map[string]any)
	if !ok {
		return LocaleMeta{}, nil, fmt.Errorf("locale %s: missing _meta block", name)
	}
	for _, field := range []string{"code", "label", "dateLocale", "dir", "order"} {
		if _, ok := rawMeta[field]; !ok {
			return LocaleMeta{}, nil, fmt.Errorf("locale %s: _meta.%s is required", name, field)
		}
	}
	encoded, err := json.Marshal(rawMeta)
	if err != nil {
		return LocaleMeta{}, nil, fmt.Errorf("locale %s: %w", name, err)
	}
	var meta LocaleMeta
	if err := json.Unmarshal(encoded, &meta); err != nil {
		return LocaleMeta{}, nil, fmt.Errorf("locale %s: %w", name, err)
	}
	if meta.Code == "" {
		return LocaleMeta{}, nil, fmt.Errorf("locale %s: _meta.code is empty", name)
	}
	if meta.Code+".json" != name {
		return LocaleMeta{}, nil, fmt.Errorf("locale %s: _meta.code %q does not match the file name", name, meta.Code)
	}
	if meta.Dir != "ltr" && meta.Dir != "rtl" {
		return LocaleMeta{}, nil, fmt.Errorf("locale %s: _meta.dir must be ltr or rtl", name)
	}
	delete(root, "_meta")

	messages := map[Key]string{}
	if err := flatten("", root, messages); err != nil {
		return LocaleMeta{}, nil, fmt.Errorf("locale %s: %w", name, err)
	}
	return meta, messages, nil
}

func flatten(prefix string, node map[string]any, out map[Key]string) error {
	for key, value := range node {
		if strings.Contains(key, ".") {
			return fmt.Errorf("key segment %q must not contain a dot", key)
		}
		full := key
		if prefix != "" {
			full = prefix + "." + key
		}
		switch typed := value.(type) {
		case string:
			out[Key(full)] = typed
		case map[string]any:
			if err := flatten(full, typed, out); err != nil {
				return err
			}
		default:
			return fmt.Errorf("key %q must be a string or an object", full)
		}
	}
	return nil
}

// chain returns the fallback order for a language:
// requested -> base tag -> default -> catalogue base.
func (c *catalog) chain(lang Lang) []Lang {
	out := make([]Lang, 0, 4)
	add := func(candidate Lang) {
		if candidate == "" {
			return
		}
		for _, existing := range out {
			if existing == candidate {
				return
			}
		}
		out = append(out, candidate)
	}
	add(lang)
	add(Lang(baseTag(normalizeTag(string(lang)))))
	add(c.defaultLang)
	add(c.baseLang)
	return out
}

func (c *catalog) lookup(lang Lang, key Key) (string, bool) {
	for _, candidate := range c.chain(lang) {
		messages, ok := c.messages[candidate]
		if !ok {
			continue
		}
		if template, ok := messages[key]; ok {
			return template, true
		}
	}
	return "", false
}

func (c *catalog) translate(lang Lang, key Key, args map[string]any) string {
	template, ok := c.lookup(lang, key)
	if !ok {
		// Calling t() on a plural node without a category: use "other" and
		// report it, instead of rendering the raw key.
		if other, found := c.lookup(lang, key+".other"); found {
			warnOnce(lang, key, "i18n_plural_without_category")
			template, ok = other, true
		}
	}
	if !ok {
		return c.missing(lang, key)
	}
	return render(template, args, lang, key)
}

func (c *catalog) translatePlural(lang Lang, key Key, n int, args map[string]any) string {
	merged := make(map[string]any, len(args)+1)
	for name, value := range args {
		merged[name] = value
	}
	merged["count"] = n
	category := Key(string(key) + "." + pluralCategory(lang, n))
	for _, candidate := range []Key{category, key + ".other"} {
		if template, ok := c.lookup(lang, candidate); ok {
			return render(template, merged, lang, candidate)
		}
	}
	if template, ok := c.lookup(lang, key); ok {
		return render(template, merged, lang, key)
	}
	return c.missing(lang, key)
}

func (c *catalog) missing(lang Lang, key Key) string {
	if c.strict {
		panic(fmt.Sprintf("i18n: missing key %q for language %q", key, lang))
	}
	warnOnce(lang, key, "i18n_missing_key")
	return string(key)
}

func render(template string, args map[string]any, lang Lang, key Key) string {
	if !strings.ContainsRune(template, '{') {
		return template
	}
	return placeholderPattern.ReplaceAllStringFunc(template, func(match string) string {
		name := match[1 : len(match)-1]
		value, ok := args[name]
		if !ok || value == nil {
			// An absent argument keeps the placeholder visible instead of
			// silently rendering an empty string.
			warnOnce(lang, key, "i18n_missing_arg", "arg", name)
			return match
		}
		return fmt.Sprint(value)
	})
}

func warnOnce(lang Lang, key Key, message string, attrs ...any) {
	token := string(lang) + "\x00" + string(key) + "\x00" + message + "\x00" + fmt.Sprint(attrs...)
	if _, loaded := warned.LoadOrStore(token, struct{}{}); loaded {
		return
	}
	fields := []any{"lang", string(lang), "key", string(key)}
	fields = append(fields, attrs...)
	slog.Warn(message, fields...)
}
