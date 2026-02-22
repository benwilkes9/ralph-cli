package docker

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/benwilkes9/ralph-cli/internal/git"
	"github.com/benwilkes9/ralph-cli/internal/preflight"
)

var requiredEnvVars = []string{"ANTHROPIC_API_KEY", "GITHUB_PAT"}

// allowedEnvVars is the set of env var names that may be loaded from .env.
// This prevents a compromised .env file from injecting vars like PATH or
// LD_PRELOAD into the process environment.
var allowedEnvVars = map[string]bool{
	"ANTHROPIC_API_KEY": true,
	"GITHUB_PAT":        true,
}

// BuildAndRun orchestrates the full Docker workflow: detect repo, load env,
// validate, build image, run container, and sync changes back.
func BuildAndRun(mode string, maxIterations int, branch, planFile, specsDir string) error {
	repo, err := DetectRepo()
	if err != nil {
		return fmt.Errorf("detecting repo: %w", err)
	}
	fmt.Printf("Repo: %s  Branch: %s\n", repo, branch)

	env, err := LoadEnvFile(".env")
	if err != nil {
		return fmt.Errorf("loading .env: %w", err)
	}

	for k, v := range env {
		if !allowedEnvVars[k] {
			return fmt.Errorf("disallowed env var in .env: %s (allowed: ANTHROPIC_API_KEY, GITHUB_PAT)", k)
		}
		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("setting env %s: %w", k, err)
		}
	}

	if err := ValidateEnv(env, requiredEnvVars); err != nil {
		return err
	}

	if err := preflight.Check(branch, specsDir, planFile); err != nil {
		return fmt.Errorf("preflight: %w", err)
	}

	if err := Build(DefaultDockerfile, DefaultTag, DefaultContext); err != nil {
		return err
	}

	logsDir, err := filepath.Abs(".ralph/logs")
	if err != nil {
		return fmt.Errorf("resolving logs dir: %w", err)
	}
	if err := os.MkdirAll(logsDir, 0o750); err != nil {
		return fmt.Errorf("creating logs dir: %w", err)
	}

	runOpts := &RunOptions{
		ImageTag: DefaultTag,
		Mode:     mode,
		MaxIter:  maxIterations,
		Branch:   branch,
		Repo:     repo,
		LogsDir:  logsDir,
		PlanFile: planFile,
		SpecsDir: specsDir,
	}
	if err := Run(runOpts); err != nil {
		return err
	}

	fmt.Println("Syncing changes from container...")
	if err := git.PullRebase(branch); err != nil {
		return fmt.Errorf("git pull --rebase: %w", err)
	}
	fmt.Println("Sync complete.")

	return nil
}
