package config

import (
	"fmt"
	"os"

	"github.com/kamalshkeir/kenv"
)

const envFile = ".env"

// New loads env (optionally from .env), fills Config and validates it.
// Any missing required variable is a startup error (fail fast).
func New() (*Config, error) {
	if _, err := os.Stat(envFile); err == nil {
		kenv.Load(envFile)
	}
	var cfg Config
	if err := kenv.Fill(&cfg); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}
	return &cfg, nil
}
