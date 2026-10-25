package unit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"taskmanager/internal/services/externals/mcpconfig"
	"taskmanager/models"
)

func TestMCPConfigMasksSecretsAndKeepsThemOnSave(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "opencode.json")
	seed := `{"$schema":"https://opencode.ai/config.json","mcp":{"demo":{"type":"local","enabled":true,"command":["/bin/echo"],"environment":{"GITLAB_TOKEN":"secret","PLAIN":"v"}}}}`
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	client := mcpconfig.New(path)
	cfg, err := client.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(cfg.Servers) != 1 || cfg.Servers[0].Environment["GITLAB_TOKEN"] != mcpconfig.Mask {
		t.Fatalf("secret not masked: %+v", cfg.Servers)
	}

	enabled := false
	if err := client.Upsert(models.MCPInput{
		Name:        "demo",
		Type:        "local",
		Enabled:     &enabled,
		Command:     []string{"/bin/echo", "hi"},
		Environment: map[string]string{"GITLAB_TOKEN": mcpconfig.Mask, "PLAIN": "v2"},
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var doc struct {
		Schema string `json:"$schema"`
		MCP    map[string]struct {
			Enabled     bool              `json:"enabled"`
			Environment map[string]string `json:"environment"`
		} `json:"mcp"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if doc.Schema == "" {
		t.Fatal("$schema must be preserved")
	}
	if doc.MCP["demo"].Enabled {
		t.Fatal("enabled flag not saved")
	}
	if doc.MCP["demo"].Environment["GITLAB_TOKEN"] != "secret" {
		t.Fatalf("masked secret must keep the stored value, got %q", doc.MCP["demo"].Environment["GITLAB_TOKEN"])
	}
	if doc.MCP["demo"].Environment["PLAIN"] != "v2" {
		t.Fatal("plain env not updated")
	}
}

func TestMCPConfigDeleteUnknown(t *testing.T) {
	t.Parallel()

	client := mcpconfig.New(filepath.Join(t.TempDir(), "opencode.json"))
	if err := client.Delete("nope"); err != mcpconfig.ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
