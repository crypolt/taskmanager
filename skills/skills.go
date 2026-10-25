// Package skills carries the agent instruction files used by this project.
// Agent files are embedded, so a role prompt is a file on disk, not a string
// in code.
package skills

import (
	"embed"
	"sort"
	"strings"
)

//go:embed agents/*.md
var agentFS embed.FS

const dirAgents = "agents"

// Agent returns the prompt for a role. Roles: planner, analyze, design,
// implement, test, review, docs, generic, verify, aggregate.
func Agent(role string) (string, bool) {
	role = strings.TrimSpace(role)
	if role == "" {
		return "", false
	}
	data, err := agentFS.ReadFile(dirAgents + "/" + role + ".md")
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(data)), true
}

// Agents lists the available role prompts.
func Agents() []string {
	entries, err := agentFS.ReadDir(dirAgents)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		names = append(names, strings.TrimSuffix(entry.Name(), ".md"))
	}
	sort.Strings(names)
	return names
}
