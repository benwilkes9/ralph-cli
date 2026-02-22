// Package testutil provides shared git test helpers for use across packages.
// It is not a _test.go file so it can be imported by test files in other packages.
package testutil

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Chdir changes the working directory to dir for the duration of the test,
// restoring the original directory in t.Cleanup.
func Chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck // best-effort restore
}

// RunGit runs a git command in dir, failing the test on error.
func RunGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...) //nolint:gosec // args are test-controlled
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %s\n%s", strings.Join(args, " "), err, out)
	}
}

// RunGitNoDir runs a git command in the current working directory,
// failing the test on error.
func RunGitNoDir(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...) //nolint:gosec // args are test-controlled
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %s\n%s", strings.Join(args, " "), err, out)
	}
}

// InitBareAndClone creates a bare "remote" repo and clones it into a working
// directory. Returns (bare, clone) paths. The clone has an initial commit pushed
// to the remote and git identity configured.
func InitBareAndClone(t *testing.T) (bare, clone string) {
	t.Helper()

	bare = t.TempDir()
	RunGit(t, bare, "init", "--bare", "--initial-branch=main")

	clone = t.TempDir()
	RunGitNoDir(t, "clone", bare, clone)

	RunGit(t, clone, "config", "user.name", "test")
	RunGit(t, clone, "config", "user.email", "test@test.com")
	RunGit(t, clone, "config", "commit.gpgsign", "false")

	RunGit(t, clone, "commit", "--allow-empty", "-m", "init")
	RunGit(t, clone, "push", "origin", "main")

	return bare, clone
}
