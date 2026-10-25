// Package mcpconfig reads and writes the MCP section of the opencode config,
// so the UI can manage the same servers the agents actually use.
package mcpconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"taskmanager/models"
)

// Mask replaces secret environment values in API responses. A save request
// that still carries the mask keeps the stored value.
const Mask = "••••••"

// Client owns one opencode config file.
type Client struct {
	path string
}

// New resolves the config path: OPENCODE_CONFIG_PATH, then OPENCODE_CONFIG,
// then the user's global opencode config.
func New(path string) *Client {
	if strings.TrimSpace(path) == "" {
		path = os.Getenv("OPENCODE_CONFIG")
	}
	if strings.TrimSpace(path) == "" {
		base := os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			home, err := os.UserHomeDir()
			if err == nil {
				base = filepath.Join(home, ".config")
			}
		}
		path = filepath.Join(base, "opencode", "opencode.json")
	}
	return &Client{path: path}
}

// Path returns the config file the client works with.
func (c *Client) Path() string { return c.path }

// List returns every MCP server with secret environment values masked.
func (c *Client) List() (models.MCPConfig, error) {
	doc, err := c.load()
	if err != nil {
		return models.MCPConfig{}, err
	}
	servers, err := decodeServers(doc)
	if err != nil {
		return models.MCPConfig{}, err
	}
	out := models.MCPConfig{Path: c.path, Servers: []models.MCPServer{}}
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		server, err := decodeServer(name, servers[name])
		if err != nil {
			return models.MCPConfig{}, err
		}
		out.Servers = append(out.Servers, server)
	}
	return out, nil
}

// Upsert creates or updates one server and keeps the rest of the config intact.
func (c *Client) Upsert(in models.MCPInput) error {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return ErrNameRequired
	}
	doc, err := c.load()
	if err != nil {
		return err
	}
	servers, err := decodeServers(doc)
	if err != nil {
		return err
	}
	obj := map[string]json.RawMessage{}
	if raw, ok := servers[name]; ok {
		if err := json.Unmarshal(raw, &obj); err != nil {
			return fmt.Errorf("mcp %q: %w", name, err)
		}
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	serverType := strings.TrimSpace(in.Type)
	if serverType == "" {
		serverType = "local"
	}
	command := in.Command
	if command == nil {
		command = []string{}
	}
	env, err := mergeEnvironment(in.Environment, obj)
	if err != nil {
		return err
	}
	setRaw(obj, "type", serverType)
	setRaw(obj, "enabled", enabled)
	setRaw(obj, "command", command)
	setRaw(obj, "environment", env)
	if strings.TrimSpace(in.URL) != "" {
		setRaw(obj, "url", strings.TrimSpace(in.URL))
	} else {
		delete(obj, "url")
	}
	raw, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	servers[name] = raw
	return c.save(doc, servers)
}

// Delete removes one server from the config.
func (c *Client) Delete(name string) error {
	doc, err := c.load()
	if err != nil {
		return err
	}
	servers, err := decodeServers(doc)
	if err != nil {
		return err
	}
	if _, ok := servers[name]; !ok {
		return ErrNotFound
	}
	delete(servers, name)
	return c.save(doc, servers)
}

// ErrNameRequired reports a server without a name.
var ErrNameRequired = fmt.Errorf("mcp server name is required")

// ErrNotFound reports a server that is not in the config.
var ErrNotFound = fmt.Errorf("mcp server not found")

func (c *Client) load() (map[string]json.RawMessage, error) {
	data, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]json.RawMessage{}, nil
		}
		return nil, err
	}
	doc := map[string]json.RawMessage{}
	if len(strings.TrimSpace(string(data))) == 0 {
		return doc, nil
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", c.path, err)
	}
	return doc, nil
}

func (c *Client) save(doc map[string]json.RawMessage, servers map[string]json.RawMessage) error {
	raw, err := json.Marshal(servers)
	if err != nil {
		return err
	}
	doc["mcp"] = raw
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(c.path); err == nil {
		backup := c.path + ".bak-" + time.Now().Format("20060102-150405")
		if data, err := os.ReadFile(c.path); err == nil {
			_ = os.WriteFile(backup, data, 0o600)
		}
	}
	tmp, err := os.CreateTemp(filepath.Dir(c.path), ".opencode-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpName, c.path)
}

func decodeServers(doc map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	servers := map[string]json.RawMessage{}
	raw, ok := doc["mcp"]
	if !ok || len(raw) == 0 {
		return servers, nil
	}
	if err := json.Unmarshal(raw, &servers); err != nil {
		return nil, fmt.Errorf("parse mcp section: %w", err)
	}
	return servers, nil
}

func decodeServer(name string, raw json.RawMessage) (models.MCPServer, error) {
	server := models.MCPServer{Name: name, Type: "local", Enabled: true, Command: []string{}, Environment: map[string]string{}}
	obj := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return server, fmt.Errorf("mcp %q: %w", name, err)
	}
	if value, ok := obj["type"]; ok {
		_ = json.Unmarshal(value, &server.Type)
	}
	if value, ok := obj["enabled"]; ok {
		_ = json.Unmarshal(value, &server.Enabled)
	}
	if value, ok := obj["command"]; ok {
		_ = json.Unmarshal(value, &server.Command)
	}
	if value, ok := obj["url"]; ok {
		_ = json.Unmarshal(value, &server.URL)
	}
	if value, ok := obj["environment"]; ok {
		_ = json.Unmarshal(value, &server.Environment)
	}
	if server.Environment == nil {
		server.Environment = map[string]string{}
	}
	for key := range server.Environment {
		if isSecret(key) {
			server.Environment[key] = Mask
			server.Masked = append(server.Masked, key)
		}
	}
	sort.Strings(server.Masked)
	return server, nil
}

func mergeEnvironment(in map[string]string, obj map[string]json.RawMessage) (map[string]string, error) {
	stored := map[string]string{}
	if raw, ok := obj["environment"]; ok {
		_ = json.Unmarshal(raw, &stored)
	}
	out := map[string]string{}
	for key, value := range in {
		if strings.TrimSpace(key) == "" {
			continue
		}
		if value == Mask {
			if old, ok := stored[key]; ok {
				out[key] = old
				continue
			}
			continue
		}
		out[key] = value
	}
	return out, nil
}

func setRaw(obj map[string]json.RawMessage, key string, value any) {
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	obj[key] = raw
}

func isSecret(key string) bool {
	upper := strings.ToUpper(key)
	for _, part := range []string{"TOKEN", "SECRET", "PASSWORD", "PASS", "API_KEY", "PRIVATE", "CREDENTIAL", "AUTH"} {
		if strings.Contains(upper, part) {
			return true
		}
	}
	return false
}
