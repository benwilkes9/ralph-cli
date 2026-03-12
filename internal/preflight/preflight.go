package preflight

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/benwilkes9/ralph-cli/internal/git"
	"github.com/benwilkes9/ralph-cli/internal/scaffold"
)

// CheckAdditionalDirs validates that each additional directory exists, is a git
// repo, and is on the expected branch. If a repo's branch is not pushed to the
// remote, it will be pushed automatically.
func CheckAdditionalDirs(branch string, dirs []string) error {
	for _, dir := range dirs {
		base := filepath.Base(dir)

		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return fmt.Errorf("preflight: additional directory %q does not exist", dir)
		} else if err != nil {
			return fmt.Errorf("preflight: checking additional directory %q: %w", dir, err)
		}

		if !git.IsGitRepo(dir) {
			return fmt.Errorf("preflight: %q is not a git repository", dir)
		}

		dirBranch, err := git.BranchIn(dir)
		if err != nil {
			return fmt.Errorf("preflight: getting branch for %q: %w", base, err)
		}
		if dirBranch != branch {
			return fmt.Errorf("preflight: repo %q is on branch %q, expected %q", base, dirBranch, branch)
		}

		exists, err := git.BranchExistsOnRemoteIn(dir, branch)
		if err != nil {
			return fmt.Errorf("preflight: checking remote branch for %q: %w", base, err)
		}
		if !exists {
			fmt.Printf("Pushing branch %q to origin for %s...\n", branch, base)
			if err := git.PushSetUpstreamIn(dir, branch); err != nil {
				return fmt.Errorf("preflight: git push -u origin %s in %q: %w", branch, base, err)
			}
		}
	}
	return nil
}

// Check runs pre-flight validation before launching Docker. It verifies that
// .ralph/ scaffold files exist on disk, auto-commits them if needed, ensures
// the specs and plans directories are tracked, and pushes the branch to the remote.
func Check(branch, specsDir, planFile string) error {
	repoRoot, err := git.RepoRoot()
	if err != nil {
		return fmt.Errorf("preflight: finding repo root: %w", err)
	}

	configPath := filepath.Join(repoRoot, ".ralph", "config.yaml")

	// 1. Config must exist locally.
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf(`".ralph/config.yaml" not found, run "ralph init" first`)
	} else if err != nil {
		return fmt.Errorf("preflight: checking config: %w", err)
	}

	// 2. Stage all scaffold-related paths and commit in a single step.
	//    This covers untracked files, modified files (e.g. .gitignore after
	//    append), specs dir, and plans dir.

	// .ralph/ directory — only add if not yet tracked.
	ralphTracked, err := git.IsTracked(".ralph/config.yaml")
	if err != nil {
		return fmt.Errorf("preflight: checking git tracking: %w", err)
	}
	if !ralphTracked {
		if err := git.Add(".ralph/"); err != nil {
			return fmt.Errorf("preflight: git add .ralph/: %w", err)
		}
	}

	// Root-level files (AGENTS.md, .env.example, .gitignore).
	// Always add if they exist — handles both untracked and modified.
	// git add on a clean tracked file is a no-op.
	for _, f := range scaffold.RootFiles() {
		if _, statErr := os.Stat(filepath.Join(repoRoot, f)); statErr != nil {
			continue
		}
		if err := git.Add(f); err != nil {
			return fmt.Errorf("preflight: git add %s: %w", f, err)
		}
	}

	// Specs and plans directories — add if untracked.
	for _, dir := range []string{specsDir, filepath.Dir(planFile)} {
		dirPath := filepath.Join(repoRoot, dir)
		if _, statErr := os.Stat(dirPath); os.IsNotExist(statErr) {
			continue
		}
		dirTracked, trackErr := git.IsTracked(filepath.Join(dir, ".gitkeep"))
		if trackErr != nil {
			return fmt.Errorf("preflight: checking git tracking for %s: %w", dir, trackErr)
		}
		if !dirTracked {
			if err := git.Add(dir + "/"); err != nil {
				return fmt.Errorf("preflight: git add %s/: %w", dir, err)
			}
		}
	}

	// Commit only if there are actually staged changes.
	hasChanges, err := git.HasStagedChanges()
	if err != nil {
		return fmt.Errorf("preflight: checking staged changes: %w", err)
	}
	if hasChanges {
		fmt.Println("Committing scaffold files...")
		if err := git.Commit("chore: scaffold ralph"); err != nil {
			return fmt.Errorf("preflight: %w", err)
		}
	}

	// 3. Push branch to remote if it doesn't exist there yet.
	exists, err := git.BranchExistsOnRemote(branch)
	if err != nil {
		return fmt.Errorf("preflight: checking remote branch: %w", err)
	}
	if !exists {
		fmt.Printf("Pushing branch %q to origin...\n", branch)
		if err := git.PushSetUpstream(branch); err != nil {
			return fmt.Errorf("preflight: git push -u origin %s: %w", branch, err)
		}
	}

	return nil
}
