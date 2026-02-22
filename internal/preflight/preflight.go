package preflight

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/benwilkes9/ralph-cli/internal/git"
)

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

	// 2. Auto-commit .ralph/ if not tracked.
	tracked, err := git.IsTracked(".ralph/config.yaml")
	if err != nil {
		return fmt.Errorf("preflight: checking git tracking: %w", err)
	}
	if !tracked {
		fmt.Println("Committing .ralph/ scaffold files...")
		if err := git.Add(".ralph/"); err != nil {
			return fmt.Errorf("preflight: git add .ralph/: %w", err)
		}
		if err := git.Commit("chore: scaffold ralph"); err != nil {
			return fmt.Errorf("preflight: git commit: %w", err)
		}
	}

	// 2b. Auto-commit specs dir and plans dir if present but untracked.
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
			fmt.Printf("Committing %s/ directory...\n", dir)
			if addErr := git.Add(dir + "/"); addErr != nil {
				return fmt.Errorf("preflight: git add %s/: %w", dir, addErr)
			}
			if commitErr := git.Commit(fmt.Sprintf("chore: add %s directory", dir)); commitErr != nil {
				return fmt.Errorf("preflight: git commit: %w", commitErr)
			}
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
