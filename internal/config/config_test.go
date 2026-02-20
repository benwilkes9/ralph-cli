package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_Valid(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
project: test
phases:
  plan:
    max_iterations: 5
  build:
    max_iterations: 10
`)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Project != "test" {
		t.Errorf("Project = %q, want %q", cfg.Project, "test")
	}
	if cfg.Phases.Plan.MaxIterations != 5 {
		t.Errorf("Plan.MaxIterations = %d, want 5", cfg.Phases.Plan.MaxIterations)
	}
}

func TestLoad_DefaultValues(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "project: test\n")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Agent != "claude" {
		t.Errorf("Agent = %q, want %q", cfg.Agent, "claude")
	}
	if cfg.Phases.Plan.MaxIterations != 5 {
		t.Errorf("Plan.MaxIterations = %d, want default 5", cfg.Phases.Plan.MaxIterations)
	}
	if cfg.Phases.Build.MaxIterations != 20 {
		t.Errorf("Build.MaxIterations = %d, want default 20", cfg.Phases.Build.MaxIterations)
	}
}

func TestLoad_NegativeIterations(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
phases:
  plan:
    max_iterations: -1
`)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for negative max_iterations")
	}
	if !strings.Contains(err.Error(), "non-negative") {
		t.Errorf("error = %q, want mention of non-negative", err.Error())
	}
}

func TestLoad_ExcessiveIterations(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
phases:
  build:
    max_iterations: 999
`)

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for excessive max_iterations")
	}
	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("error = %q, want mention of exceeds maximum", err.Error())
	}
}

func TestLoad_FileTooLarge(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, ".ralph")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Fatal(err)
	}
	// Write a file larger than 64 KiB
	large := strings.Repeat("x", maxConfigSize+1)
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(large), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for oversized config")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("error = %q, want mention of too large", err.Error())
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load(t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing config")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, ":\n  bad:\nyaml: [")

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestPlanPathForBranch_DefaultDir(t *testing.T) {
	cfg := &Config{}
	cfg.applyDefaults()

	got := cfg.PlanPathForBranch("feature/auth-flow")
	want := ".ralph/plans/IMPLEMENTATION_PLAN_feature-auth-flow.md"
	if got != want {
		t.Errorf("PlanPathForBranch = %q, want %q", got, want)
	}
}

func TestPlanPathForBranch_CustomFile(t *testing.T) {
	cfg := &Config{
		Phases: Phases{
			Plan: PhaseConfig{Output: "my-plan.md"},
		},
	}

	got := cfg.PlanPathForBranch("feat/login")
	want := "my-plan_feat-login.md"
	if got != want {
		t.Errorf("PlanPathForBranch = %q, want %q", got, want)
	}
}

func TestPlanPathForBranch_CustomDir(t *testing.T) {
	cfg := &Config{
		Phases: Phases{
			Plan: PhaseConfig{Output: "plans/"},
		},
	}

	got := cfg.PlanPathForBranch("fix/bug-123")
	want := "plans/IMPLEMENTATION_PLAN_fix-bug-123.md"
	if got != want {
		t.Errorf("PlanPathForBranch = %q, want %q", got, want)
	}
}

func writeConfig(t *testing.T, dir, content string) {
	t.Helper()
	configDir := filepath.Join(dir, ".ralph")
	if err := os.MkdirAll(configDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
