package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/benwilkes9/ralph-cli/internal/config"
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
// validate, build image, run container with bind mount.
func BuildAndRun(mode string, maxIterations int, branch, planFile, specsDir string) error {
	repo, err := DetectRepo()
	if err != nil {
		return fmt.Errorf("detecting repo: %w", err)
	}

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

	repoRoot, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("resolving project dir: %w", err)
	}

	cfg, err := config.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	allowedDomains := AllowedDomains(cfg.Network.ExtraAllowedDomains)

	fmt.Printf("Repo: %s  Branch: %s\n", repo, branch)
	fmt.Printf("Mount: %s → /workspace/repo\n", repoRoot)
	if cfg.Docker.DepsDir != "" {
		fmt.Printf("Deps volume: ralph-deps-%s → %s\n", cfg.Project, cfg.Docker.DepsDir)
	}
	fmt.Printf("Network allowlist: %s\n", strings.Join(allowedDomains, ", "))
	fmt.Println("Workspace is shared — changes appear on the host in real time.")

	runOpts := &RunOptions{
		ImageTag:       DefaultTag,
		Mode:           mode,
		MaxIter:        maxIterations,
		Branch:         branch,
		ProjectDir:     repoRoot,
		PlanFile:       planFile,
		SpecsDir:       specsDir,
		AllowedDomains: allowedDomains,
		DepsDir:        cfg.Docker.DepsDir,
		ProjectName:    cfg.Project,
	}

	return Run(runOpts)
}
