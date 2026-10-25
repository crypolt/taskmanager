package models

import "strings"

// MaxScopeEntries caps how many paths a task may own.
const MaxScopeEntries = 12

// NormalizeScope trims, deduplicates and caps a task scope.
func NormalizeScope(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, entry := range in {
		entry = strings.TrimSpace(entry)
		entry = strings.TrimPrefix(entry, "./")
		entry = strings.TrimSuffix(entry, "/")
		if entry == "" || entry == "." || seen[entry] {
			continue
		}
		seen[entry] = true
		out = append(out, entry)
		if len(out) == MaxScopeEntries {
			break
		}
	}
	return out
}

// ScopesConflict reports whether two tasks may write to the same paths. An
// empty scope means "no claim" and never conflicts.
func ScopesConflict(a, b []string) bool {
	for _, left := range a {
		for _, right := range b {
			if pathsOverlap(left, right) {
				return true
			}
		}
	}
	return false
}

// ScopesConflictAny reports whether scope intersects any of the others.
func ScopesConflictAny(scope []string, others [][]string) bool {
	for _, other := range others {
		if ScopesConflict(scope, other) {
			return true
		}
	}
	return false
}

func pathsOverlap(a, b string) bool {
	a = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(a), "./"), "/")
	b = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(b), "./"), "/")
	if a == "" || b == "" {
		return false
	}
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}
