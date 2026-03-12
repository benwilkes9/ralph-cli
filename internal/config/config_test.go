package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const minimalConfig = "project: test\n"

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
	writeConfig(t, dir, minimalConfig)

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
			yaml := minimalConfig
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

func TestSpecsDirForBranch_Default(t *testing.T) {
	cfg := &Config{}
	assert.Equal(t, "specs/my-feature", cfg.SpecsDirForBranch("my-feature"))
}

func TestSpecsDirForBranch_Custom(t *testing.T) {
	cfg := &Config{SpecsDir: "docs/requirements"}
	assert.Equal(t, "docs/requirements/my-feature", cfg.SpecsDirForBranch("my-feature"))
}

func TestSpecsDirForBranch_Exact(t *testing.T) {
	cfg := &Config{SpecsDir: "my/exact/path", SpecsDirExact: true}
	assert.Equal(t, "my/exact/path", cfg.SpecsDirForBranch("my-feature"))
}

func TestLoad_SpecsDir(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
project: test
specs_dir: my/custom/dir
`)

	cfg, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, "my/custom/dir", cfg.SpecsDir)
	assert.False(t, cfg.SpecsDirExact)
}

func TestLoad_SpecsDirExact(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
project: test
specs_dir: my/exact/path
specs_dir_exact: true
`)

	cfg, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, "my/exact/path", cfg.SpecsDir)
	assert.True(t, cfg.SpecsDirExact)
}

func TestLoad_SpecsDirTraversal(t *testing.T) {
	tests := []struct {
		specsDir string
		wantErr  bool
	}{
		{"specs", false},
		{"docs/requirements", false},
		{"my/custom/path", false},
		{"", false},
		{"../../etc", true},
		{"/etc", true},
		{"../outside", true},
		{".", true},
		{"foo/../../bar", true},
	}
	for _, tt := range tests {
		t.Run(tt.specsDir, func(t *testing.T) {
			dir := t.TempDir()
			yaml := minimalConfig
			if tt.specsDir != "" {
				yaml += "specs_dir: " + tt.specsDir + "\n"
			}
			writeConfig(t, dir, yaml)

			_, err := Load(dir)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorContains(t, err, "specs_dir must be a relative path")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoad_AdditionalDirs(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `
project: test
additional_directories:
  - /home/user/repo-a
  - /home/user/repo-b
`)

	cfg, err := Load(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"/home/user/repo-a", "/home/user/repo-b"}, cfg.AdditionalDirs)
}

func TestLoad_AdditionalDirsValidation(t *testing.T) {
	tests := []struct {
		name    string
		dirs    string
		wantErr string
	}{
		{
			name:    "relative path rejected",
			dirs:    "  - relative/path",
			wantErr: "path must be absolute",
		},
		{
			name:    "duplicate path rejected",
			dirs:    "  - /home/user/repo-a\n  - /home/user/repo-a",
			wantErr: "duplicate path",
		},
		{
			name:    "basename collision rejected",
			dirs:    "  - /home/user/one/lib\n  - /home/user/two/lib",
			wantErr: "duplicate basename",
		},
		{
			name:    "path traversal via .. basename",
			dirs:    "  - /foo/bar/..",
			wantErr: "basename",
		},
		{
			name:    "path traversal via . basename",
			dirs:    "  - /foo/.",
			wantErr: "basename",
		},
		{
			name:    "reserved basename repo",
			dirs:    "  - /home/user/repo",
			wantErr: "reserved",
		},
		{
			name:    "comma in path rejected",
			dirs:    "  - /home/user/dir,name",
			wantErr: "commas",
		},
		{
			name: "empty list valid",
			dirs: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			yaml := minimalConfig
			if tt.dirs != "" {
				yaml += "additional_directories:\n" + tt.dirs + "\n"
			}
			writeConfig(t, dir, yaml)

			_, err := Load(dir)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
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
