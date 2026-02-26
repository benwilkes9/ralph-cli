package docker

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/benwilkes9/ralph-cli/internal/config"
	"github.com/benwilkes9/ralph-cli/internal/preflight"
	"github.com/benwilkes9/ralph-cli/internal/ui"
)

// requiredEnvVars lists env vars required beyond auth credentials
// (auth is validated separately by ResolveAuth).
var requiredEnvVars = []string{"GITHUB_PAT"}

// allowedEnvVars is the set of env var names that may be loaded from .env.
// This prevents a compromised .env file from injecting vars like PATH or
// LD_PRELOAD into the process environment.
var allowedEnvVars = map[string]bool{
	"ANTHROPIC_API_KEY":       true,
	"CLAUDE_CODE_OAUTH_TOKEN": true,
	"GITHUB_PAT":              true,
}

// AuthMethod indicates how the container authenticates with Claude.
type AuthMethod int

const (
	// AuthAPIKey uses ANTHROPIC_API_KEY (usage-based billing).
	AuthAPIKey AuthMethod = iota
	// AuthOAuth uses CLAUDE_CODE_OAUTH_TOKEN (Claude Max subscription).
	AuthOAuth
)

// ResolveAuth determines the authentication method from the environment.
// API key takes precedence when both are set.
func ResolveAuth(env map[string]string) (AuthMethod, error) {
	hasAPIKey := env["ANTHROPIC_API_KEY"] != "" || os.Getenv("ANTHROPIC_API_KEY") != ""
	hasOAuth := env["CLAUDE_CODE_OAUTH_TOKEN"] != "" || os.Getenv("CLAUDE_CODE_OAUTH_TOKEN") != ""

	switch {
	case hasAPIKey:
		return AuthAPIKey, nil
	case hasOAuth:
		return AuthOAuth, nil
	default:
		return 0, fmt.Errorf("missing auth: set ANTHROPIC_API_KEY or CLAUDE_CODE_OAUTH_TOKEN")
	}
}

// BuildAndRun orchestrates the full Docker workflow: detect repo, load env,
// validate, build image, run container with bind mount.
func BuildAndRun(w io.Writer, theme *ui.Theme, mode string, maxIterations int, branch, planFile, specsDir string) error {
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
			return fmt.Errorf("disallowed env var in .env: %s (allowed: ANTHROPIC_API_KEY, CLAUDE_CODE_OAUTH_TOKEN, GITHUB_PAT)", k)
		}
		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("setting env %s: %w", k, err)
		}
	}

	auth, err := ResolveAuth(env)
	if err != nil {
		return err
	}

	if err := ValidateEnv(env, requiredEnvVars); err != nil {
		return err
	}

	if err := preflight.Check(branch, specsDir, planFile); err != nil {
		return err //nolint:wrapcheck // preflight errors already have context
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

	fmt.Fprintf(w, "%s %s  %s %s\n", //nolint:errcheck // display-only
		theme.Muted.Render("Repo:"), repo,
		theme.Muted.Render("Branch:"), theme.Info.Render(branch))
	fmt.Fprintf(w, "%s %s → /workspace/repo\n", theme.Muted.Render("Mount:"), repoRoot) //nolint:errcheck // display-only
	if cfg.Docker.DepsDir != "" {
		fmt.Fprintf(w, "%s ralph-deps-%s → %s\n", //nolint:errcheck // display-only
			theme.Muted.Render("Deps volume:"), cfg.Project, cfg.Docker.DepsDir)
	}
	fmt.Fprintf(w, "%s %s\n", //nolint:errcheck // display-only
		theme.Muted.Render("Network allowlist:"), strings.Join(allowedDomains, ", "))
	fmt.Fprintln(w, theme.Muted.Render("Workspace is shared — changes appear on the host in real time.")) //nolint:errcheck // display-only

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
		Auth:           auth,
	}

	return Run(runOpts)
}
