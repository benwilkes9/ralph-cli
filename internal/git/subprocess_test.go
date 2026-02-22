package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NOTE: Do NOT add t.Parallel() to any test in this file.
// os.Chdir is process-global; parallel tests would race on the working directory.

// initBareAndClone creates a bare remote and a clone, configures git identity,
// makes an initial commit, pushes to the remote, and returns (bare, clone) paths.
func initBareAndClone(t *testing.T) (bare, clone string) {
	t.Helper()

	bare = t.TempDir()
	runGitHelper(t, bare, "init", "--bare", "--initial-branch=main")

	clone = t.TempDir()
	runGitHelperNoDir(t, "clone", bare, clone)

	runGitHelper(t, clone, "config", "user.name", "test")
	runGitHelper(t, clone, "config", "user.email", "test@test.com")
	runGitHelper(t, clone, "config", "commit.gpgsign", "false")

	runGitHelper(t, clone, "commit", "--allow-empty", "-m", "init")
	runGitHelper(t, clone, "push", "origin", "main")

	return bare, clone
}

func runGitHelper(t *testing.T, dir string, args ...string) {
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

func runGitHelperNoDir(t *testing.T, args ...string) {
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

func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck // best-effort restore
}

// TestHead verifies that Head() returns a 40-character hex SHA and errors
// when called outside a git repo.
func TestHead(t *testing.T) {
	_, clone := initBareAndClone(t)
	chdir(t, clone)

	head, err := Head()
	require.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`^[0-9a-f]{40}$`), head)
}

func TestHead_NotARepo(t *testing.T) {
	chdir(t, t.TempDir())

	_, err := Head()
	assert.Error(t, err)
}

// TestBranch verifies Branch() returns the current branch name.
func TestBranch(t *testing.T) {
	_, clone := initBareAndClone(t)
	chdir(t, clone)

	branch, err := Branch()
	require.NoError(t, err)
	assert.Equal(t, "main", branch)
}

func TestBranch_FeatureBranch(t *testing.T) {
	_, clone := initBareAndClone(t)
	chdir(t, clone)

	runGitHelper(t, clone, "checkout", "-b", "feature-test")

	branch, err := Branch()
	require.NoError(t, err)
	assert.Equal(t, "feature-test", branch)
}

// TestAddAndCommit verifies that Add + Commit creates a commit visible in git log.
func TestAddAndCommit(t *testing.T) {
	_, clone := initBareAndClone(t)
	chdir(t, clone)

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
	_, clone := initBareAndClone(t)
	chdir(t, clone)

	err := Add("nonexistent.txt")
	assert.Error(t, err)
}

// TestPush verifies that Push sends a committed change to the bare remote.
func TestPush(t *testing.T) {
	bare, clone := initBareAndClone(t)
	chdir(t, clone)

	f := filepath.Join(clone, "pushed.txt")
	require.NoError(t, os.WriteFile(f, []byte("pushed"), 0o600))
	runGitHelper(t, clone, "add", "pushed.txt")
	runGitHelper(t, clone, "commit", "-m", "push test")

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
	_, clone := initBareAndClone(t)
	chdir(t, clone)

	runGitHelper(t, clone, "checkout", "-b", "upstream-test")

	require.NoError(t, PushSetUpstream("upstream-test"))

	// The branch should now exist on origin as seen from the clone.
	exists, err := BranchExistsOnRemote("upstream-test")
	require.NoError(t, err)
	assert.True(t, exists)
}

// TestPullRebase verifies that a commit added to the bare remote is pulled into the clone.
func TestPullRebase(t *testing.T) {
	_, clone := initBareAndClone(t)
	chdir(t, clone)

	// Discover the remote URL of the current clone.
	remoteURL, err := RemoteURL("origin")
	require.NoError(t, err)

	// Push a new commit via a second clone of the same remote.
	clone2 := t.TempDir()
	runGitHelperNoDir(t, "clone", remoteURL, clone2)
	runGitHelper(t, clone2, "config", "user.name", "test")
	runGitHelper(t, clone2, "config", "user.email", "test@test.com")

	f := filepath.Join(clone2, "remote.txt")
	require.NoError(t, os.WriteFile(f, []byte("remote"), 0o600))
	runGitHelper(t, clone2, "add", "remote.txt")
	runGitHelper(t, clone2, "commit", "-m", "remote commit")
	runGitHelper(t, clone2, "push", "origin", "main")

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
	_, clone := initBareAndClone(t)
	chdir(t, clone)

	url, err := RemoteURL("origin")
	require.NoError(t, err)
	assert.NotEmpty(t, url)
}

func TestRemoteURL_Unknown(t *testing.T) {
	_, clone := initBareAndClone(t)
	chdir(t, clone)

	_, err := RemoteURL("nonexistent")
	assert.Error(t, err)
}

// TestRepoRoot verifies RepoRoot returns the clone root even from a subdirectory.
func TestRepoRoot(t *testing.T) {
	_, clone := initBareAndClone(t)
	sub := filepath.Join(clone, "a", "b")
	require.NoError(t, os.MkdirAll(sub, 0o750))
	chdir(t, sub)

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
	_, clone := initBareAndClone(t)
	chdir(t, clone)

	// Untracked file.
	f := filepath.Join(clone, "untracked.txt")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0o600))

	tracked, err := IsTracked("untracked.txt")
	require.NoError(t, err)
	assert.False(t, tracked)

	// After staging + committing.
	runGitHelper(t, clone, "add", "untracked.txt")
	runGitHelper(t, clone, "commit", "-m", "track it")

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
	_, clone := initBareAndClone(t)
	chdir(t, clone)

	exists, err := BranchExistsOnRemote("main")
	require.NoError(t, err)
	assert.True(t, exists)

	runGitHelper(t, clone, "checkout", "-b", "local-only")

	exists, err = BranchExistsOnRemote("local-only")
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestDiffFromRemote verifies DiffFromRemote is empty when synced and non-empty after a local commit.
func TestDiffFromRemote(t *testing.T) {
	_, clone := initBareAndClone(t)
	chdir(t, clone)

	diff, err := DiffFromRemote("main", ".")
	require.NoError(t, err)
	assert.Empty(t, diff)

	// Add a local commit without pushing.
	f := filepath.Join(clone, "local.txt")
	require.NoError(t, os.WriteFile(f, []byte("local"), 0o600))
	runGitHelper(t, clone, "add", "local.txt")
	runGitHelper(t, clone, "commit", "-m", "local change")

	diff, err = DiffFromRemote("main", ".")
	require.NoError(t, err)
	assert.NotEmpty(t, diff)
}
