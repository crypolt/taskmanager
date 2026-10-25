package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// RunnerAgent is the agent name used for headless task runs.
const RunnerAgent = "task-runner"

// RunnerConfig writes a temporary opencode config for task runs: a dedicated
// agent with a bounded step count and without subagents, skills or MCP tools.
// A full MCP tool set costs ~70k prompt tokens on every step, so disabling the
// servers listed by the caller keeps runs fast and cheap.
func RunnerConfig(steps int, mcpNames []string) (string, error) {
	if steps <= 0 {
		steps = 12
	}
	agent := map[string]any{
		"description": "Headless task executor: repository tools only, bounded steps.",
		"mode":        "primary",
		"steps":       steps,
		"tools": map[string]any{
			"read": true, "glob": true, "grep": true, "bash": true,
			"edit": true, "write": true, "todowrite": true,
			"task": false, "skill": false, "question": false, "webfetch": false,
		},
	}
	doc := map[string]any{
		"$schema": "https://opencode.ai/config.json",
		"agent":   map[string]any{RunnerAgent: agent},
	}
	if len(mcpNames) > 0 {
		servers := make(map[string]any, len(mcpNames))
		for _, name := range mcpNames {
			servers[name] = map[string]any{"enabled": false}
		}
		doc["mcp"] = servers
	}
	body, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(os.TempDir(), fmt.Sprintf("taskmanager-opencode-%d.json", os.Getpid()))
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return "", err
	}
	return path, nil
}
