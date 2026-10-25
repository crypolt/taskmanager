package internals

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// maxWorkspaceReport caps what we hand to the verifier: a diff stat plus the
// porcelain listing is enough to point it at the right files, and a truncated
// section is still useful while a huge one would eat the prompt budget.
const maxWorkspaceReport = 4000

// workspaceChanges summarises what changed in the working tree since the last
// commit. It is advisory: outside a git repository (or without git installed)
// the verifier simply gets no workspace section and falls back to the output.
func (e *Engine) workspaceChanges(ctx context.Context) string {
	status := runGit(ctx, "status", "--porcelain")
	stat := runGit(ctx, "diff", "--stat")
	parts := make([]string, 0, 2)
	if status != "" {
		parts = append(parts, "Changed files (git status --porcelain):\n"+status)
	}
	if stat != "" {
		parts = append(parts, "Diff stat:\n"+stat)
	}
	if len(parts) == 0 {
		return ""
	}
	out := strings.Join(parts, "\n\n")
	if len(out) > maxWorkspaceReport {
		out = out[:maxWorkspaceReport] + "\n… (truncated)"
	}
	return out
}

func runGit(ctx context.Context, args ...string) string {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// sameOutput reports whether two executor responses are effectively identical.
// Models that hit the same wall twice return the same text, and re-verifying it
// only burns tokens: the caller escalates instead.
func sameOutput(a, b string) bool {
	normalize := func(s string) string {
		return strings.Join(strings.Fields(strings.ToLower(s)), " ")
	}
	left, right := normalize(a), normalize(b)
	if left == "" || right == "" {
		return false
	}
	return left == right
}
