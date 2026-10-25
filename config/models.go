package config

import (
	"fmt"
	"strings"
	"time"
)

type Config struct {
	ServiceName string `kenv:"SERVICE_NAME|task-manager"`
	Language    string `kenv:"LANGUAGE|ru"`
	Server      struct {
		Address   string `kenv:"SERVER_ADDRESS|0.0.0.0:8080"`
		TraceMode bool   `kenv:"TRACE_MODE|true"`
		LogLevel  string `kenv:"LOG_LEVEL|debug"`
	}
	DB struct {
		Path string `kenv:"DB_PATH|taskmanager.db"`
	}
	MCP struct {
		ConfigPath string `kenv:"OPENCODE_CONFIG_PATH|"`
	}
	LLM struct {
		Workers         int    `kenv:"WORKERS|3"`
		MaxPromptChars  int    `kenv:"MAX_PROMPT_CHARS|40000"`
		MaxPromptTokens int    `kenv:"MAX_PROMPT_TOKENS|8000"`
		ExecTimeoutSec  int    `kenv:"EXEC_TIMEOUT_SEC|600"`
		OpenCodeBin     string `kenv:"OPENCODE_BIN|opencode"`
		OpenCodeAuto    bool   `kenv:"OPENCODE_AUTO_APPROVE|true"`
		OpenCodePure    bool   `kenv:"OPENCODE_PURE|true"`
		OpenCodeMCP     bool   `kenv:"OPENCODE_MCP|false"`
		OpenCodeSteps   int    `kenv:"OPENCODE_STEPS|40"`
		OpenCodeIdleSec int    `kenv:"OPENCODE_IDLE_SEC|120"`
		OpenAIBaseURL   string `kenv:"OPENAI_BASE_URL|https://api.openai.com/v1"`
		OpenAIKey       string `kenv:"OPENAI_API_KEY|"`
		OllamaBaseURL   string `kenv:"OLLAMA_BASE_URL|http://localhost:11434"`
	}
}

func (c *Config) Address() string { return c.Server.Address }

func (c *Config) ExecTimeout() time.Duration {
	if c.LLM.ExecTimeoutSec <= 0 {
		return 15 * time.Minute
	}
	return time.Duration(c.LLM.ExecTimeoutSec) * time.Second
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.Server.Address) == "" {
		return fmt.Errorf("SERVER_ADDRESS is required")
	}
	if c.LLM.Workers <= 0 {
		return fmt.Errorf("WORKERS must be greater than zero")
	}
	if c.DB.Path == "" {
		return fmt.Errorf("DB_PATH is required")
	}
	return nil
}
