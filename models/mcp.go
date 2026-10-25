package models

// MCPServer is one MCP server from the opencode config.
type MCPServer struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Enabled     bool              `json:"enabled"`
	Command     []string          `json:"command"`
	Environment map[string]string `json:"environment"`
	URL         string            `json:"url,omitempty"`
	// Masked lists environment keys whose values are hidden in API responses.
	Masked []string `json:"masked,omitempty"`
}

// MCPConfig is the MCP page read model: where the config lives and what it holds.
type MCPConfig struct {
	Path    string      `json:"path"`
	Servers []MCPServer `json:"servers"`
}

// MCPInput creates or replaces one MCP server.
type MCPInput struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	Enabled     *bool             `json:"enabled"`
	Command     []string          `json:"command"`
	Environment map[string]string `json:"environment"`
	URL         string            `json:"url"`
}
