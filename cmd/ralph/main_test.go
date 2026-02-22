package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: No t.Parallel() in this file — os.Chdir is process-global.

// --- fakes ---

// fakeOrchestrator records BuildAndRun calls without touching Docker.
type fakeOrchestrator struct {
	calls []fakeCall
	err   error
}

type fakeCall struct {
	mode, branch, planFile, specsDir string
	maxIter                          int
}

func (f *fakeOrchestrator) BuildAndRun(mode string, maxIter int, branch, planFile, specsDir string) error {
	f.calls = append(f.calls, fakeCall{mode, branch, planFile, specsDir, maxIter})
	return f.err
}

// byteReader delivers one byte at a time so huh's per-field scanner reads
// prompts correctly in accessible mode (mirrors prompt_test.go).
type byteReader struct{ r io.Reader }

func (br *byteReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return br.r.Read(p[:1]) //nolint:wrapcheck // test helper
}

// --- helpers ---

// initSimpleRepo creates a git repo with an empty commit on feature-test.
func initSimpleRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGitCmd(t, dir, "init", "--initial-branch=main")
	runGitCmd(t, dir, "config", "user.name", "test")
	runGitCmd(t, dir, "config", "user.email", "test@test.com")
	runGitCmd(t, dir, "config", "commit.gpgsign", "false")
	runGitCmd(t, dir, "commit", "--allow-empty", "-m", "init")
	runGitCmd(t, dir, "checkout", "-b", "feature-test")
	return dir
}

// initRepoWithConfig creates initSimpleRepo plus a minimal .ralph/config.yaml.
func initRepoWithConfig(t *testing.T) string {
	t.Helper()
	dir := initSimpleRepo(t)
	ralph := filepath.Join(dir, ".ralph")
	require.NoError(t, os.MkdirAll(ralph, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(ralph, "config.yaml"), []byte("project: test\n"), 0o600))
	return dir
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...) //nolint:gosec // args are test-controlled
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %s\n%s", strings.Join(args, " "), err, out)
	}
}

func chdirTest(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck // best-effort restore
}

// --- validateRelativePath ---

func TestValidateRelativePath(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		// valid
		{"specs/feature", false},
		{"custom-specs", false},
		{"a/b/c", false},
		{".", false},

		// absolute paths
		{"/etc/passwd", true},
		{"/absolute/path", true},

		// traversal
		{"..", true},
		{"../outside", true},
		{"a/../../outside", true},
		{"specs/../../../etc", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			err := validateRelativePath("specs", tt.path)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "--specs")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// --- resolveRunParams ---

func TestResolveRunParams_DefaultPaths(t *testing.T) {
	dir := initRepoWithConfig(t)
	chdirTest(t, dir)

	cmd := planCmd(&fakeOrchestrator{})

	p, err := resolveRunParams(cmd)
	require.NoError(t, err)

	assert.Equal(t, "feature-test", p.branch)
	assert.Equal(t, "specs/feature-test", p.specsDir)
	assert.Contains(t, p.planFile, "feature-test")
}

func TestResolveRunParams_CustomSpecs(t *testing.T) {
	dir := initRepoWithConfig(t)
	chdirTest(t, dir)

	cmd := planCmd(&fakeOrchestrator{})
	require.NoError(t, cmd.Flags().Set("specs", "custom/path"))

	p, err := resolveRunParams(cmd)
	require.NoError(t, err)

	assert.Equal(t, "custom/path", p.specsDir)
}

func TestResolveRunParams_ProtectedBranch(t *testing.T) {
	// Stay on main (protected by default).
	dir := t.TempDir()
	runGitCmd(t, dir, "init", "--initial-branch=main")
	runGitCmd(t, dir, "config", "user.name", "test")
	runGitCmd(t, dir, "config", "user.email", "test@test.com")
	runGitCmd(t, dir, "config", "commit.gpgsign", "false")
	runGitCmd(t, dir, "commit", "--allow-empty", "-m", "init")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ralph"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".ralph", "config.yaml"), []byte("project: test\n"), 0o600))
	chdirTest(t, dir)

	cmd := planCmd(&fakeOrchestrator{})
	_, err := resolveRunParams(cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "feature branch")
}

// --- initCmd ---

func TestInitCmd_NotInGitRepo(t *testing.T) {
	chdirTest(t, t.TempDir())

	cmd := initCmd()
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "finding repo root")
}

func TestInitCmd_CreatesScaffold(t *testing.T) {
	dir := t.TempDir()
	runGitCmd(t, dir, "init", "--initial-branch=main")
	runGitCmd(t, dir, "config", "user.name", "test")
	runGitCmd(t, dir, "config", "user.email", "test@test.com")
	chdirTest(t, dir)

	cmd := initCmd()
	cmd.SetIn(&byteReader{strings.NewReader("1\n1\n")})
	cmd.SetOut(&bytes.Buffer{})

	require.NoError(t, cmd.Execute())
	assert.DirExists(t, filepath.Join(dir, ".ralph"))
}

// --- planCmd ---

func TestPlanCmd_CreatesDirectoriesAndCallsOrchestrator(t *testing.T) {
	dir := initRepoWithConfig(t)
	chdirTest(t, dir)

	fake := &fakeOrchestrator{}
	cmd := planCmd(fake)

	require.NoError(t, cmd.Execute())

	assert.FileExists(t, filepath.Join(dir, "specs", "feature-test", ".gitkeep"))
	assert.FileExists(t, filepath.Join(dir, ".ralph", "plans", ".gitkeep"))

	require.Len(t, fake.calls, 1)
	assert.Equal(t, "plan", fake.calls[0].mode)
	assert.Equal(t, "feature-test", fake.calls[0].branch)
}

func TestPlanCmd_ProtectedBranch(t *testing.T) {
	dir := t.TempDir()
	runGitCmd(t, dir, "init", "--initial-branch=main")
	runGitCmd(t, dir, "config", "user.name", "test")
	runGitCmd(t, dir, "config", "user.email", "test@test.com")
	runGitCmd(t, dir, "config", "commit.gpgsign", "false")
	runGitCmd(t, dir, "commit", "--allow-empty", "-m", "init")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".ralph"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".ralph", "config.yaml"), []byte("project: test\n"), 0o600))
	chdirTest(t, dir)

	fake := &fakeOrchestrator{}
	cmd := planCmd(fake)

	require.Error(t, cmd.Execute())
	assert.Empty(t, fake.calls)
}

// --- applyCmd ---

func TestApplyCmd_MissingPlanFile(t *testing.T) {
	dir := initRepoWithConfig(t)
	chdirTest(t, dir)

	fake := &fakeOrchestrator{}
	cmd := applyCmd(fake)

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Empty(t, fake.calls)
}

func TestApplyCmd_CallsOrchestratorWithBuildMode(t *testing.T) {
	dir := initRepoWithConfig(t)
	chdirTest(t, dir)

	planPath := filepath.Join(dir, ".ralph", "plans", "IMPLEMENTATION_PLAN_feature-test.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(planPath), 0o750))
	require.NoError(t, os.WriteFile(planPath, []byte("# Plan\n"), 0o600))

	fake := &fakeOrchestrator{}
	cmd := applyCmd(fake)

	require.NoError(t, cmd.Execute())

	require.Len(t, fake.calls, 1)
	assert.Equal(t, "build", fake.calls[0].mode)
	assert.Equal(t, "feature-test", fake.calls[0].branch)
}

// --- statusCmd ---

func TestStatusCmd_RendersOutput(t *testing.T) {
	dir := initRepoWithConfig(t)
	chdirTest(t, dir)

	cmd := statusCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "test")         // project name from config
	assert.Contains(t, out.String(), "feature-test") // branch name
}

// --- loopCmd ---

func TestLoopCmd_InvalidMode(t *testing.T) {
	cmd := loopCmd()
	cmd.SetArgs([]string{"foo"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown mode")
}
