package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	assert.Equal(t, "test", cfg.Project)
	assert.Equal(t, 5, cfg.Phases.Plan.MaxIterations)
}

func TestLoad_DefaultValues(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "project: test\n")

	cfg, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, "claude", cfg.Agent)
	assert.Equal(t, 5, cfg.Phases.Plan.MaxIterations)
	assert.Equal(t, 20, cfg.Phases.Build.MaxIterations)
	assert.Equal(t, []string{"main", "master"}, cfg.ProtectedBranches)
}

func TestLoad_NegativeIterations(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
phases:
  plan:
    max_iterations: -1
`)

	_, err := Load(dir)
	require.Error(t, err)
	assert.ErrorContains(t, err, "non-negative")
}

func TestLoad_ExcessiveIterations(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
phases:
  build:
    max_iterations: 999
`)

	_, err := Load(dir)
	require.Error(t, err)
	assert.ErrorContains(t, err, "exceeds maximum")
}

func TestLoad_FileTooLarge(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, ".ralph")
	require.NoError(t, os.MkdirAll(configDir, 0o750))
	large := strings.Repeat("x", maxConfigSize+1)
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(large), 0o600))

	_, err := Load(dir)
	require.Error(t, err)
	assert.ErrorContains(t, err, "too large")
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load(t.TempDir())
	require.Error(t, err)
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, ":\n  bad:\nyaml: [")

	_, err := Load(dir)
	require.Error(t, err)
}

func TestLoad_NetworkExtraAllowedDomains(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
project: test
network:
  extra_allowed_domains:
    - pypi.org
    - files.pythonhosted.org
`)

	cfg, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"pypi.org", "files.pythonhosted.org"}, cfg.Network.ExtraAllowedDomains)
}

func TestLoad_DockerDepsDir(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
project: test
docker:
  deps_dir: node_modules
`)

	cfg, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, "node_modules", cfg.Docker.DepsDir)
}

func TestLoad_DepsDirTraversal(t *testing.T) {
	tests := []struct {
		depsDir string
		wantErr bool
	}{
		{"node_modules", false},
		{".venv", false},
		{"target", false},
		{"", false},
		{"../../etc", true},
		{"/etc", true},
		{"../outside", true},
		{".", true},
		{"foo/../../bar", true},
	}
	for _, tt := range tests {
		t.Run(tt.depsDir, func(t *testing.T) {
			dir := t.TempDir()
			yaml := "project: test\n"
			if tt.depsDir != "" {
				yaml += "docker:\n  deps_dir: " + tt.depsDir + "\n"
			}
			writeConfig(t, dir, yaml)

			_, err := Load(dir)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, "docker.deps_dir must be a relative path")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoad_BackwardCompat_NoNetworkOrDocker(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
project: test
phases:
  plan:
    max_iterations: 3
`)

	cfg, err := Load(dir)
	require.NoError(t, err)
	assert.Empty(t, cfg.Network.ExtraAllowedDomains)
	assert.Empty(t, cfg.Docker.DepsDir)
}

func TestPlanPathForBranch_DefaultDir(t *testing.T) {
	cfg := &Config{}
	cfg.applyDefaults()

	got := cfg.PlanPathForBranch("feature-auth-flow")
	assert.Equal(t, ".ralph/plans/IMPLEMENTATION_PLAN_feature-auth-flow.md", got)
}

func TestPlanPathForBranch_CustomFile(t *testing.T) {
	cfg := &Config{
		Phases: Phases{
			Plan: PhaseConfig{Output: "my-plan.md"},
		},
	}

	got := cfg.PlanPathForBranch("feat-login")
	assert.Equal(t, "my-plan_feat-login.md", got)
}

func TestPlanPathForBranch_CustomDir(t *testing.T) {
	cfg := &Config{
		Phases: Phases{
			Plan: PhaseConfig{Output: "plans/"},
		},
	}

	got := cfg.PlanPathForBranch("fix-bug-123")
	assert.Equal(t, "plans/IMPLEMENTATION_PLAN_fix-bug-123.md", got)
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
