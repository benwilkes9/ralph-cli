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
	runGit(t, bare, "init", "--bare", "--initial-branch=main")

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

func gitDiff(t *testing.T, dir string, args ...string) string {
	t.Helper()
	fullArgs := append([]string{"diff"}, args...)                        //nolint:gocritic // append to separate slice is intentional
	cmd := exec.CommandContext(context.Background(), "git", fullArgs...) //nolint:gosec // args are test-controlled
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git diff failed: %s", err)
	}
	return strings.TrimSpace(string(out))
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

	err := Check("main", "specs", ".ralph/plans/IMPLEMENTATION_PLAN_main.md")
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

	err := Check("main", "specs", ".ralph/plans/IMPLEMENTATION_PLAN_main.md")
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

	err := Check("feature-xyz", "specs", ".ralph/plans/IMPLEMENTATION_PLAN_feature-xyz.md")
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

	err := Check("main", "specs", ".ralph/plans/IMPLEMENTATION_PLAN_main.md")
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}
}

func TestCheck_AutoCommitsSpecsDir(t *testing.T) {
	clone := initBareAndClone(t)
	chdir(t, clone)
	writeScaffold(t, clone)

	// Commit and push scaffold so it's tracked.
	runGit(t, clone, "add", ".ralph/")
	runGit(t, clone, "commit", "-m", "add scaffold")
	runGit(t, clone, "push", "origin", "main")

	// Create untracked custom specs dir with .gitkeep.
	specsDir := filepath.Join(clone, "requirements", "v2")
	if err := os.MkdirAll(specsDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(specsDir, ".gitkeep"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	err := Check("main", "requirements/v2", ".ralph/plans/IMPLEMENTATION_PLAN_main.md")
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}

	log := gitLog(t, clone)
	if !strings.Contains(log, "chore: add requirements/v2 directory") {
		t.Errorf("expected auto-commit for custom specs dir in log, got:\n%s", log)
	}
}

func TestCheck_AutoCommitsPlansDir(t *testing.T) {
	clone := initBareAndClone(t)
	chdir(t, clone)
	writeScaffold(t, clone)

	// Commit and push scaffold so it's tracked.
	runGit(t, clone, "add", ".ralph/")
	runGit(t, clone, "commit", "-m", "add scaffold")
	runGit(t, clone, "push", "origin", "main")

	// Create untracked custom plans dir with .gitkeep.
	plansDir := filepath.Join(clone, "custom", "plans")
	if err := os.MkdirAll(plansDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plansDir, ".gitkeep"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}

	err := Check("main", "specs", "custom/plans/PLAN.md")
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}

	log := gitLog(t, clone)
	if !strings.Contains(log, "chore: add custom/plans directory") {
		t.Errorf("expected auto-commit for custom plans dir in log, got:\n%s", log)
	}
}

func TestCheck_AutoPushesCustomPlanDir(t *testing.T) {
	clone := initBareAndClone(t)
	chdir(t, clone)
	writeScaffold(t, clone)

	// Commit and push scaffold.
	runGit(t, clone, "add", ".ralph/")
	runGit(t, clone, "commit", "-m", "add scaffold")
	runGit(t, clone, "push", "origin", "main")

	// Create and commit a plan file + .gitkeep in a custom dir outside .ralph/.
	plansDir := filepath.Join(clone, "plans")
	if err := os.MkdirAll(plansDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plansDir, ".gitkeep"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	planFile := filepath.Join(plansDir, "IMPLEMENTATION_PLAN_main.md")
	if err := os.WriteFile(planFile, []byte("# Plan"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, clone, "add", "plans/")
	runGit(t, clone, "commit", "-m", "add plan")

	// Check should push the custom plans/ dir to origin.
	err := Check("main", "specs", "plans/IMPLEMENTATION_PLAN_main.md")
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}

	// Verify the plan was pushed — no diff between local and remote.
	diff := gitDiff(t, clone, "origin/main", "--", "plans/")
	if diff != "" {
		t.Errorf("expected custom plan dir to be pushed, but got diff:\n%s", diff)
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

	err := Check("main", "specs", ".ralph/plans/IMPLEMENTATION_PLAN_main.md")
	if err != nil {
		t.Fatalf("expected no error, got: %s", err)
	}
}
