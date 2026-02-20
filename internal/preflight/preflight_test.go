package preflight

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initBareAndClone creates a bare "remote" repo and clones it into a working
// directory. Returns the clone path. All git ops use the clone as cwd.
func initBareAndClone(t *testing.T) string {
	t.Helper()

	bare := t.TempDir()
	runGit(t, bare, "init", "--bare")

	clone := t.TempDir()
	runGitNoDir(t, "clone", bare, clone)

	// Configure user so commits from the git package work too.
	runGit(t, clone, "config", "user.name", "test")
	runGit(t, clone, "config", "user.email", "test@test.com")

	// Create an initial commit so HEAD and branch exist.
	runGit(t, clone, "commit", "--allow-empty", "-m", "init")
	runGit(t, clone, "push", "origin", "main")

	return clone
}

func runGit(t *testing.T, dir string, args ...string) {
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

func runGitNoDir(t *testing.T, args ...string) {
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

func gitLog(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", "log", "--oneline") //nolint:gosec // test helper
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git log failed: %s", err)
	}
	return string(out)
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck // best-effort restore
}

func writeScaffold(t *testing.T, dir string) {
	t.Helper()
	ralphDir := filepath.Join(dir, ".ralph")
	if err := os.MkdirAll(ralphDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("project: test"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCheck_ConfigMissing(t *testing.T) {
	clone := initBareAndClone(t)
	chdir(t, clone)

	err := Check("main")
	if err == nil {
		t.Fatal("expected error when config is missing")
	}
	if !strings.Contains(err.Error(), `"ralph init"`) {
		t.Errorf("expected actionable message mentioning ralph init, got: %s", err)
	}
}

func TestCheck_AutoCommitsUntracked(t *testing.T) {
	clone := initBareAndClone(t)
	chdir(t, clone)
	writeScaffold(t, clone)

	err := Check("main")
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}

	// Verify .ralph/ was committed.
	log := gitLog(t, clone)
	if !strings.Contains(log, "chore: scaffold ralph") {
		t.Errorf("expected auto-commit in log, got:\n%s", log)
	}
}

func TestCheck_AutoPushesBranch(t *testing.T) {
	clone := initBareAndClone(t)
	chdir(t, clone)
	writeScaffold(t, clone)

	// Create a local-only branch with .ralph/ committed.
	runGit(t, clone, "add", ".ralph/")
	runGit(t, clone, "commit", "-m", "add scaffold")
	runGit(t, clone, "checkout", "-b", "feature-xyz")

	err := Check("feature-xyz")
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}
}

func TestCheck_AutoPushesUnpushedChanges(t *testing.T) {
	clone := initBareAndClone(t)
	chdir(t, clone)
	writeScaffold(t, clone)

	// Commit and push scaffold.
	runGit(t, clone, "add", ".ralph/")
	runGit(t, clone, "commit", "-m", "add scaffold")
	runGit(t, clone, "push", "origin", "main")

	// Make a local change without pushing.
	if err := os.WriteFile(filepath.Join(clone, ".ralph", "config.yaml"), []byte("project: updated"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, clone, "add", ".ralph/")
	runGit(t, clone, "commit", "-m", "update scaffold")

	err := Check("main")
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}
}

func TestCheck_AllClean(t *testing.T) {
	clone := initBareAndClone(t)
	chdir(t, clone)
	writeScaffold(t, clone)

	// Everything committed and pushed — should be a no-op.
	runGit(t, clone, "add", ".ralph/")
	runGit(t, clone, "commit", "-m", "add scaffold")
	runGit(t, clone, "push", "origin", "main")

	err := Check("main")
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}
}
