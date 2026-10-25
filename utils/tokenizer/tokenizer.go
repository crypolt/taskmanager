// Package tokenizer estimates and fits text to a token budget without a
// model-specific BPE table. Estimates are intentionally conservative:
// overestimating saves money, underestimating breaks limits.
package tokenizer

import (
	"strings"
	"unicode"
)

// Estimate approximates the number of BPE tokens in text.
func Estimate(text string) int {
	var acc float64
	words := 0
	inWord := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			if inWord {
				words++
				inWord = false
			}
			continue
		}
		inWord = true
		acc += weight(r)
	}
	if inWord {
		words++
	}
	acc += float64(words) * 0.15
	n := int(acc + 0.5)
	if n == 0 && strings.TrimSpace(text) != "" {
		n = 1
	}
	return n
}

// EstimateParts sums the estimate of every part.
func EstimateParts(parts ...string) int {
	total := 0
	for _, part := range parts {
		total += Estimate(part)
	}
	return total
}

// Truncate cuts text to maxTokens at the nearest word or punctuation boundary
// and marks the cut with an ellipsis.
func Truncate(text string, maxTokens int) string {
	if maxTokens <= 0 {
		return ""
	}
	if Estimate(text) <= maxTokens {
		return text
	}
	limit := float64(maxTokens) - 1
	var acc float64
	cut := 0
	inWord := false
	for i, r := range text {
		if unicode.IsSpace(r) {
			if inWord {
				acc += 0.15
				inWord = false
			}
			cut = i
			continue
		}
		inWord = true
		acc += weight(r)
		if acc > limit {
			break
		}
		if unicode.IsPunct(r) {
			cut = i + len(string(r))
		}
	}
	if cut <= 0 {
		cut = len(text) / 2
	}
	out := strings.TrimSpace(text[:cut])
	if out == "" {
		return ""
	}
	return out + "…"
}

// Fits reports whether text is within the budget.
func Fits(text string, maxTokens int) bool {
	return maxTokens <= 0 || Estimate(text) <= maxTokens
}

func weight(r rune) float64 {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		return 0.25
	case unicode.Is(unicode.Cyrillic, r):
		return 0.45
	case unicode.Is(unicode.Han, r), unicode.Is(unicode.Hiragana, r), unicode.Is(unicode.Katakana, r):
		return 1.5
	case unicode.IsDigit(r):
		return 0.33
	case unicode.IsPunct(r), unicode.IsSymbol(r):
		return 0.33
	default:
		return 0.5
	}
}
