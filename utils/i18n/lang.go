package i18n

import (
	"sort"
	"strconv"
	"strings"
)

// Lang is a BCP-47 language tag supported by the catalogue.
type Lang string

// Base languages shipped with the binary. A third language only needs a new
// file under locales/ — no Go code changes.
const (
	RU Lang = "ru"
	EN Lang = "en"
)

func normalizeTag(tag string) string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	return strings.ReplaceAll(tag, "_", "-")
}

func baseTag(tag string) string {
	if index := strings.IndexByte(tag, '-'); index >= 0 {
		return tag[:index]
	}
	return tag
}

// Match resolves a tag against the catalogue: an exact tag first, then its
// base tag (de-AT -> de). An unknown tag is not an error, it simply does not
// match, so callers can silently fall through to the next source.
func Match(tag string) (Lang, bool) {
	normalized := normalizeTag(tag)
	if normalized == "" || normalized == "*" {
		return "", false
	}
	c := catalogOrLoad()
	if lang, ok := c.byTag[normalized]; ok {
		return lang, true
	}
	if lang, ok := c.byTag[baseTag(normalized)]; ok {
		return lang, true
	}
	return "", false
}

// ParseAcceptLanguage parses an Accept-Language header, honouring q-values and
// preserving their order. Unknown entries and q=0 are dropped.
func ParseAcceptLanguage(header string) []Lang {
	type candidate struct {
		tag   string
		q     float64
		index int
	}
	candidates := make([]candidate, 0, 4)
	for index, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		tag, quality := part, 1.0
		if semi := strings.IndexByte(part, ';'); semi >= 0 {
			tag = strings.TrimSpace(part[:semi])
			for _, parameter := range strings.Split(part[semi+1:], ";") {
				parameter = strings.TrimSpace(parameter)
				if value, ok := strings.CutPrefix(parameter, "q="); ok {
					if parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
						quality = parsed
					}
				}
			}
		}
		if quality <= 0 {
			continue
		}
		candidates = append(candidates, candidate{tag: tag, q: quality, index: index})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].q != candidates[j].q {
			return candidates[i].q > candidates[j].q
		}
		return candidates[i].index < candidates[j].index
	})
	out := make([]Lang, 0, len(candidates))
	seen := map[Lang]bool{}
	for _, item := range candidates {
		lang, ok := Match(item.tag)
		if !ok || seen[lang] {
			continue
		}
		seen[lang] = true
		out = append(out, lang)
	}
	return out
}

// Resolve returns the first supported candidate, or the default language.
func Resolve(candidates ...string) Lang {
	for _, candidate := range candidates {
		if lang, ok := Match(candidate); ok {
			return lang
		}
	}
	return Default()
}

// pluralCategory implements the CLDR cardinal categories needed by the shipped
// languages. An unknown language falls back to one/other, and every catalogue
// is required to define "other", so a new language works without code changes.
func pluralCategory(lang Lang, n int) string {
	if n < 0 {
		n = -n
	}
	switch baseTag(normalizeTag(string(lang))) {
	case "ru", "uk", "be", "sr", "hr", "bs":
		mod10, mod100 := n%10, n%100
		switch {
		case mod10 == 1 && mod100 != 11:
			return "one"
		case mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14):
			return "few"
		default:
			return "many"
		}
	case "pl":
		mod10, mod100 := n%10, n%100
		switch {
		case n == 1:
			return "one"
		case mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14):
			return "few"
		default:
			return "many"
		}
	case "cs", "sk":
		switch {
		case n == 1:
			return "one"
		case n >= 2 && n <= 4:
			return "few"
		default:
			return "other"
		}
	default:
		if n == 1 {
			return "one"
		}
		return "other"
	}
}
