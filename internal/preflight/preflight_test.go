package preflight

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/benwilkes9/ralph-cli/internal/testutil"
)

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

func writeScaffold(t *testing.T, dir string) {
	t.Helper()
	ralphDir := filepath.Join(dir, ".ralph")
	require.NoError(t, os.MkdirAll(ralphDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("project: test"), 0o600))
}

func TestCheck_ConfigMissing(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)

	err := Check("main", "specs", ".ralph/plans/IMPLEMENTATION_PLAN_main.md")
	require.Error(t, err)
	assert.ErrorContains(t, err, `"ralph init"`)
}

func TestCheck_AutoCommitsUntracked(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)
	writeScaffold(t, clone)

	err := Check("main", "specs", ".ralph/plans/IMPLEMENTATION_PLAN_main.md")
	require.NoError(t, err)

	// Verify .ralph/ was committed.
	log := gitLog(t, clone)
	assert.Contains(t, log, "chore: scaffold ralph")
}

func TestCheck_AutoPushesBranch(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)
	writeScaffold(t, clone)

	// Create a local-only branch with .ralph/ committed.
	testutil.RunGit(t, clone, "add", ".ralph/")
	testutil.RunGit(t, clone, "commit", "-m", "add scaffold")
	testutil.RunGit(t, clone, "checkout", "-b", "feature-xyz")

	err := Check("feature-xyz", "specs", ".ralph/plans/IMPLEMENTATION_PLAN_feature-xyz.md")
	require.NoError(t, err)
}

func TestCheck_UnpushedChanges_NoAutoPush(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)
	writeScaffold(t, clone)

	// Commit and push scaffold.
	testutil.RunGit(t, clone, "add", ".ralph/")
	testutil.RunGit(t, clone, "commit", "-m", "add scaffold")
	testutil.RunGit(t, clone, "push", "origin", "main")

	// Make a local change without pushing.
	require.NoError(t, os.WriteFile(filepath.Join(clone, ".ralph", "config.yaml"), []byte("project: updated"), 0o600))
	testutil.RunGit(t, clone, "add", ".ralph/")
	testutil.RunGit(t, clone, "commit", "-m", "update scaffold")

	// Should succeed without pushing (bind mount reads host files directly).
	err := Check("main", "specs", ".ralph/plans/IMPLEMENTATION_PLAN_main.md")
	require.NoError(t, err)

	// Verify the unpushed changes are NOT auto-pushed.
	diff := gitDiff(t, clone, "origin/main", "--", ".ralph/")
	assert.NotEmpty(t, diff, "expected unpushed changes to remain — preflight should not auto-push")
}

func TestCheck_AutoCommitsSpecsDir(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)
	writeScaffold(t, clone)

	// Commit and push scaffold so it's tracked.
	testutil.RunGit(t, clone, "add", ".ralph/")
	testutil.RunGit(t, clone, "commit", "-m", "add scaffold")
	testutil.RunGit(t, clone, "push", "origin", "main")

	// Create untracked custom specs dir with .gitkeep.
	specsDir := filepath.Join(clone, "requirements", "v2")
	require.NoError(t, os.MkdirAll(specsDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(specsDir, ".gitkeep"), []byte(""), 0o600))

	err := Check("main", "requirements/v2", ".ralph/plans/IMPLEMENTATION_PLAN_main.md")
	require.NoError(t, err)

	log := gitLog(t, clone)
	assert.Contains(t, log, "chore: add requirements/v2 directory")
}

func TestCheck_AutoCommitsPlansDir(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)
	writeScaffold(t, clone)

	// Commit and push scaffold so it's tracked.
	testutil.RunGit(t, clone, "add", ".ralph/")
	testutil.RunGit(t, clone, "commit", "-m", "add scaffold")
	testutil.RunGit(t, clone, "push", "origin", "main")

	// Create untracked custom plans dir with .gitkeep.
	plansDir := filepath.Join(clone, "custom", "plans")
	require.NoError(t, os.MkdirAll(plansDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(plansDir, ".gitkeep"), []byte(""), 0o600))

	err := Check("main", "specs", "custom/plans/PLAN.md")
	require.NoError(t, err)

	log := gitLog(t, clone)
	assert.Contains(t, log, "chore: add custom/plans directory")
}

func TestCheck_CustomPlanDir_NoAutoPush(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)
	writeScaffold(t, clone)

	// Commit and push scaffold.
	testutil.RunGit(t, clone, "add", ".ralph/")
	testutil.RunGit(t, clone, "commit", "-m", "add scaffold")
	testutil.RunGit(t, clone, "push", "origin", "main")

	// Create and commit a plan file + .gitkeep in a custom dir outside .ralph/.
	plansDir := filepath.Join(clone, "plans")
	require.NoError(t, os.MkdirAll(plansDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(plansDir, ".gitkeep"), []byte(""), 0o600))
	planFile := filepath.Join(plansDir, "IMPLEMENTATION_PLAN_main.md")
	require.NoError(t, os.WriteFile(planFile, []byte("# Plan"), 0o600))
	testutil.RunGit(t, clone, "add", "plans/")
	testutil.RunGit(t, clone, "commit", "-m", "add plan")

	// Should succeed without pushing (bind mount reads host files directly).
	err := Check("main", "specs", "plans/IMPLEMENTATION_PLAN_main.md")
	require.NoError(t, err)

	// Verify the plan was NOT auto-pushed — diff should still exist.
	diff := gitDiff(t, clone, "origin/main", "--", "plans/")
	assert.NotEmpty(t, diff, "expected unpushed changes to remain — preflight should not auto-push")
}

func TestCheck_AllClean(t *testing.T) {
	_, clone := testutil.InitBareAndClone(t)
	testutil.Chdir(t, clone)
	writeScaffold(t, clone)

	// Everything committed and pushed — should be a no-op.
	testutil.RunGit(t, clone, "add", ".ralph/")
	testutil.RunGit(t, clone, "commit", "-m", "add scaffold")
	testutil.RunGit(t, clone, "push", "origin", "main")

	err := Check("main", "specs", ".ralph/plans/IMPLEMENTATION_PLAN_main.md")
	require.NoError(t, err)
}
