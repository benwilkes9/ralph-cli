package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the ralph project configuration loaded from .ralph/config.yaml.
type Config struct {
	Project string `yaml:"project"`
	Agent   string `yaml:"agent"`

	Backpressure Backpressure `yaml:"backpressure"`
	Phases       Phases       `yaml:"phases"`
}

// Backpressure defines the commands used to validate code quality between iterations.
type Backpressure struct {
	Test      string `yaml:"test"`
	Typecheck string `yaml:"typecheck"`
	Lint      string `yaml:"lint"`
}

// Phases groups the plan and build phase configurations.
type Phases struct {
	Plan  PhaseConfig `yaml:"plan"`
	Build PhaseConfig `yaml:"build"`
}

// PhaseConfig holds settings for a single loop phase (plan or build).
type PhaseConfig struct {
	Prompt        string `yaml:"prompt"`
	Output        string `yaml:"output,omitempty"`
	MaxIterations int    `yaml:"max_iterations"`
	FreshContext  bool   `yaml:"fresh_context,omitempty"`
}

// Load reads .ralph/config.yaml from the given repo root.
func Load(repoRoot string) (*Config, error) {
	path := filepath.Join(repoRoot, ".ralph", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	cfg.applyDefaults()
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Agent == "" {
		c.Agent = "claude"
	}
	if c.Phases.Plan.Prompt == "" {
		c.Phases.Plan.Prompt = ".ralph/prompts/plan.md"
	}
	if c.Phases.Plan.Output == "" {
		c.Phases.Plan.Output = ".ralph/IMPLEMENTATION_PLAN.md"
	}
	if c.Phases.Plan.MaxIterations == 0 {
		c.Phases.Plan.MaxIterations = 5
	}
	if c.Phases.Build.Prompt == "" {
		c.Phases.Build.Prompt = ".ralph/prompts/build.md"
	}
	if c.Phases.Build.MaxIterations == 0 {
		c.Phases.Build.MaxIterations = 20
	}
}
