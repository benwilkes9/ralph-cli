package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/benwilkes9/ralph-cli/internal/git"
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

// maxConfigSize is the maximum config file size we'll read (64 KiB).
const maxConfigSize = 64 * 1024

// Load reads .ralph/config.yaml from the given repo root.
func Load(repoRoot string) (*Config, error) {
	path := filepath.Join(repoRoot, ".ralph", "config.yaml")

	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	if info.Size() > maxConfigSize {
		return nil, fmt.Errorf("config file too large: %d bytes (max %d)", info.Size(), maxConfigSize)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	cfg.applyDefaults()
	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Phases.Plan.MaxIterations < 0 {
		return fmt.Errorf("phases.plan.max_iterations must be non-negative")
	}
	if c.Phases.Build.MaxIterations < 0 {
		return fmt.Errorf("phases.build.max_iterations must be non-negative")
	}
	if c.Phases.Plan.MaxIterations > 100 {
		return fmt.Errorf("phases.plan.max_iterations exceeds maximum (100)")
	}
	if c.Phases.Build.MaxIterations > 100 {
		return fmt.Errorf("phases.build.max_iterations exceeds maximum (100)")
	}
	return nil
}

func (c *Config) applyDefaults() {
	if c.Agent == "" {
		c.Agent = "claude"
	}
	if c.Phases.Plan.Prompt == "" {
		c.Phases.Plan.Prompt = ".ralph/prompts/plan.md"
	}
	if c.Phases.Plan.Output == "" {
		c.Phases.Plan.Output = ".ralph/plans/"
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

// PlanPathForBranch returns the branch-specific plan file path.
// If the configured output is a directory (ends with /), the plan file is
// placed inside it as IMPLEMENTATION_PLAN_{sanitized-branch}.md.
// If the user set a custom file path, the branch suffix is inserted before
// the extension.
func (c *Config) PlanPathForBranch(branch string) string {
	sanitized := git.SanitizeBranch(branch)
	output := c.Phases.Plan.Output

	if strings.HasSuffix(output, "/") {
		return output + "IMPLEMENTATION_PLAN_" + sanitized + ".md"
	}

	// Custom file path: insert branch before extension.
	ext := filepath.Ext(output)
	base := strings.TrimSuffix(output, ext)
	return base + "_" + sanitized + ext
}
