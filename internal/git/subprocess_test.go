package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/benwilkes9/ralph-cli/internal/testutil"
)

// NOTE: Do NOT add t.Parallel() to any test in this file.
// os.Chdir is process-global; parallel tests would race on the working directory.

// TestHead verifies that Head() returns a 40-character hex SHA and errors
// when called outside a git repo.
func TestHead(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)

	head, err := Head()
	require.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`^[0-9a-f]{40}$`), head)
}

func TestHead_NotARepo(t *testing.T) {
	testutil.Chdir(t, t.TempDir())

	_, err := Head()
	assert.Error(t, err)
}

// TestBranch verifies Branch() returns the current branch name.
func TestBranch(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)

	branch, err := Branch()
	require.NoError(t, err)
	assert.Equal(t, "main", branch)
}

func TestBranch_FeatureBranch(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)

	testutil.RunGit(t, clone, "checkout", "-b", "feature-test")

	branch, err := Branch()
	require.NoError(t, err)
	assert.Equal(t, "feature-test", branch)
}

// TestAddAndCommit verifies that Add + Commit creates a commit visible in git log.
func TestAddAndCommit(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)

	f := filepath.Join(clone, "hello.txt")
	require.NoError(t, os.WriteFile(f, []byte("hello"), 0o600))

	require.NoError(t, Add("hello.txt"))
	require.NoError(t, Commit("test: add hello"))

	cmd := exec.CommandContext(context.Background(), "git", "log", "--oneline") //nolint:gosec // test helper
	cmd.Dir = clone
	out, err := cmd.Output()
	require.NoError(t, err)
	assert.Contains(t, string(out), "test: add hello")
}

// TestAdd_NonExistentPath verifies that staging a missing file returns an error.
func TestAdd_NonExistentPath(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)

	err := Add("nonexistent.txt")
	assert.Error(t, err)
}

// TestPush verifies that Push sends a committed change to the bare remote.
func TestPush(t *testing.T) {
	bare, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)

	f := filepath.Join(clone, "pushed.txt")
	require.NoError(t, os.WriteFile(f, []byte("pushed"), 0o600))
	testutil.RunGit(t, clone, "add", "pushed.txt")
	testutil.RunGit(t, clone, "commit", "-m", "push test")

	require.NoError(t, Push("main"))

	// The commit should now be visible in the bare remote's log.
	cmd := exec.CommandContext(context.Background(), "git", "log", "--oneline") //nolint:gosec // test helper
	cmd.Dir = bare
	out, err := cmd.Output()
	require.NoError(t, err)
	assert.Contains(t, string(out), "push test")
}

// TestPushSetUpstream verifies that a local-only branch is created on the remote.
func TestPushSetUpstream(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)

	testutil.RunGit(t, clone, "checkout", "-b", "upstream-test")

	require.NoError(t, PushSetUpstream("upstream-test"))

	// The branch should now exist on origin as seen from the clone.
	exists, err := BranchExistsOnRemote("upstream-test")
	require.NoError(t, err)
	assert.True(t, exists)
}

// TestPullRebase verifies that a commit added to the bare remote is pulled into the clone.
func TestPullRebase(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)

	// Discover the remote URL of the current clone.
	remoteURL, err := RemoteURL("origin")
	require.NoError(t, err)

	// Push a new commit via a second clone of the same remote.
	clone2 := t.TempDir()
	testutil.RunGitNoDir(t, "clone", remoteURL, clone2)
	testutil.RunGit(t, clone2, "config", "user.name", "test")
	testutil.RunGit(t, clone2, "config", "user.email", "test@test.com")
	testutil.RunGit(t, clone2, "config", "commit.gpgsign", "false")

	f := filepath.Join(clone2, "remote.txt")
	require.NoError(t, os.WriteFile(f, []byte("remote"), 0o600))
	testutil.RunGit(t, clone2, "add", "remote.txt")
	testutil.RunGit(t, clone2, "commit", "-m", "remote commit")
	testutil.RunGit(t, clone2, "push", "origin", "main")

	require.NoError(t, PullRebase("main"))

	// Verify the pulled commit is visible in the first clone.
	cmd := exec.CommandContext(context.Background(), "git", "log", "--oneline") //nolint:gosec // test helper
	cmd.Dir = clone
	out, err := cmd.Output()
	require.NoError(t, err)
	assert.Contains(t, string(out), "remote commit")
}

// TestRemoteURL verifies RemoteURL returns the origin URL and errors for unknown remotes.
func TestRemoteURL(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)

	url, err := RemoteURL("origin")
	require.NoError(t, err)
	assert.NotEmpty(t, url)
}

func TestRemoteURL_Unknown(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)

	_, err := RemoteURL("nonexistent")
	assert.Error(t, err)
}

// TestRepoRoot verifies RepoRoot returns the clone root even from a subdirectory.
func TestRepoRoot(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	sub := filepath.Join(clone, "a", "b")
	require.NoError(t, os.MkdirAll(sub, 0o750))
	testutil.Chdir(t, sub)

	root, err := RepoRoot()
	require.NoError(t, err)

	// Resolve symlinks so temp dir path comparisons are reliable.
	wantEval, err := filepath.EvalSymlinks(clone)
	require.NoError(t, err)
	gotEval, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)
	assert.Equal(t, wantEval, gotEval)
}

// TestIsTracked verifies IsTracked behaviour for untracked, committed, and missing paths.
func TestIsTracked(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)

	// Untracked file.
	f := filepath.Join(clone, "untracked.txt")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0o600))

	tracked, err := IsTracked("untracked.txt")
	require.NoError(t, err)
	assert.False(t, tracked)

	// After staging + committing.
	testutil.RunGit(t, clone, "add", "untracked.txt")
	testutil.RunGit(t, clone, "commit", "-m", "track it")

	tracked, err = IsTracked("untracked.txt")
	require.NoError(t, err)
	assert.True(t, tracked)

	// Non-existent path.
	tracked, err = IsTracked("does-not-exist.txt")
	require.NoError(t, err)
	assert.False(t, tracked)
}

// TestBranchExistsOnRemote verifies BranchExistsOnRemote for existing and local-only branches.
func TestBranchExistsOnRemote(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)

	exists, err := BranchExistsOnRemote("main")
	require.NoError(t, err)
	assert.True(t, exists)

	testutil.RunGit(t, clone, "checkout", "-b", "local-only")

	exists, err = BranchExistsOnRemote("local-only")
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestDiffFromRemote verifies DiffFromRemote is empty when synced and non-empty after a local commit.
func TestDiffFromRemote(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)

	diff, err := DiffFromRemote("main", ".")
	require.NoError(t, err)
	assert.Empty(t, diff)

	// Add a local commit without pushing.
	f := filepath.Join(clone, "local.txt")
	require.NoError(t, os.WriteFile(f, []byte("local"), 0o600))
	testutil.RunGit(t, clone, "add", "local.txt")
	testutil.RunGit(t, clone, "commit", "-m", "local change")

	diff, err = DiffFromRemote("main", ".")
	require.NoError(t, err)
	assert.NotEmpty(t, diff)
}
