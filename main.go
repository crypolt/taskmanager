package main

import (
	"time"

	"go.uber.org/fx"

	"taskmanager/config"
	"taskmanager/internal/repository"
	"taskmanager/internal/services/externals/llm"
	"taskmanager/internal/services/externals/mcpconfig"
	"taskmanager/internal/services/internals"
	"taskmanager/logger"
	"taskmanager/server"
	httpserver "taskmanager/server/http"
	"taskmanager/server/http/handlers"
	"taskmanager/utils/i18n"
)

func main() {
	fx.New(
		fx.Provide(
			config.New,
			logger.InitLogger,
			fx.Annotate(newTaskRepository, fx.As(new(internals.Store))),
			newLLMRegistry,
			newMCPClient,
			internals.NewBus,
			internals.NewEngine,
			handlers.New,
			httpserver.NewServer,
		),
		fx.Invoke(initI18n),
		fx.Invoke(server.RegisterEngine),
		fx.Invoke(func(*httpserver.Server) {}),
	).Run()
}

func initI18n(cfg *config.Config) error {
	return i18n.Init(cfg.Language)
}

func newTaskRepository(cfg *config.Config) (*repository.Store, error) {
	return repository.Open(cfg.DB.Path)
}

// runnerConfig builds the task-runner opencode config: bounded steps and no
// MCP servers unless OPENCODE_MCP is enabled.
func runnerConfig(cfg *config.Config) string {
	if cfg.LLM.OpenCodeMCP {
		return ""
	}
	var names []string
	if items, err := newMCPClient(cfg).List(); err == nil {
		for _, server := range items.Servers {
			names = append(names, server.Name)
		}
	}
	path, err := llm.RunnerConfig(cfg.LLM.OpenCodeSteps, names)
	if err != nil {
		return ""
	}
	return path
}

func newMCPClient(cfg *config.Config) *mcpconfig.Client {
	return mcpconfig.New(cfg.MCP.ConfigPath)
}

func newLLMRegistry(cfg *config.Config) *llm.Registry {
	registry := llm.NewRegistry()
	registry.Register(&llm.OpenCode{
		Bin:     cfg.LLM.OpenCodeBin,
		Timeout: cfg.ExecTimeout(),
		Auto:    cfg.LLM.OpenCodeAuto,
		Agent:   llm.RunnerAgent,
		Pure:    cfg.LLM.OpenCodePure,
		Config:  runnerConfig(cfg),
		Idle:    time.Duration(cfg.LLM.OpenCodeIdleSec) * time.Second,
	})
	registry.Register(&llm.OpenAICompatible{
		ProviderName: "openai",
		BaseURL:      cfg.LLM.OpenAIBaseURL,
		APIKey:       cfg.LLM.OpenAIKey,
	})
	registry.Register(&llm.OpenAICompatible{
		ProviderName: "ollama",
		BaseURL:      cfg.LLM.OllamaBaseURL + "/v1",
	})
	registry.Register(&llm.Mock{})
	return registry
}
