package git

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeBranch(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"feature/auth-flow", "feature-auth-flow"},
		{"main", "main"},
		{"already-clean", "already-clean"},
		{"feat/nested/deep/branch", "feat-nested-deep-branch"},
		{"-leading-hyphen", "leading-hyphen"},
		{"trailing-hyphen-", "trailing-hyphen"},
		{"-both-sides-", "both-sides"},
		{"special!@#chars", "specialchars"},
		{"dots.are.ok", "dots.are.ok"},
		{"under_scores", "under_scores"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SanitizeBranch(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsProtectedBranch(t *testing.T) {
	defaultProtected := []string{"main", "master"}

	tests := []struct {
		branch    string
		protected []string
		want      bool
	}{
		{"main", defaultProtected, true},
		{"master", defaultProtected, true},
		{"Main", defaultProtected, true},
		{"MASTER", defaultProtected, true},
		{"develop", defaultProtected, false},
		{"feature/auth", defaultProtected, false},
		{"main-feature", defaultProtected, false},
		{"develop", []string{"develop", "staging"}, true},
		{"staging", []string{"develop", "staging"}, true},
		{"main", []string{"develop", "staging"}, false},
		{"main", nil, false},
		{"main", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			got := IsProtectedBranch(tt.branch, tt.protected)
			assert.Equal(t, tt.want, got)
		})
	}
}

// initRepo creates a git repo with an initial commit in a temp dir.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "--initial-branch=main"},
		{"config", "user.name", "test"},
		{"config", "user.email", "test@test.com"},
		{"config", "commit.gpgsign", "false"},
		{"commit", "--allow-empty", "-m", "init"},
	} {
		cmd := exec.CommandContext(context.Background(), "git", args...) //nolint:gosec // test helper
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v: %s", args, out)
	}
	return dir
}

func TestIsGitRepo(t *testing.T) {
	dir := initRepo(t)
	assert.True(t, IsGitRepo(dir))
	assert.False(t, IsGitRepo(t.TempDir()))
}

func TestHeadIn(t *testing.T) {
	dir := initRepo(t)
	head, err := HeadIn(dir)
	require.NoError(t, err)
	assert.Len(t, head, 40, "expected full SHA")
}

func TestBranchIn(t *testing.T) {
	dir := initRepo(t)
	branch, err := BranchIn(dir)
	require.NoError(t, err)
	assert.Equal(t, "main", branch)
}
