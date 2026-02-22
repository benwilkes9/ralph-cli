package docker

import (
	"fmt"
	"strconv"
)

// RunOptions configures a docker run invocation.
type RunOptions struct {
	ImageTag string
	Mode     string // "plan" or "build"
	MaxIter  int
	Branch   string
	Repo     string
	LogsDir  string
	PlanFile string
	SpecsDir string
}

// Run executes docker run with the given options, attaching stdin/stdout/stderr.
func Run(opts *RunOptions) error {
	return runWithRunner(defaultRunner{}, opts)
}

func runWithRunner(runner CommandRunner, opts *RunOptions) error {
	args := []string{
		"run", "--rm", "-it",
		"--security-opt", "no-new-privileges",
		"-e", "ANTHROPIC_API_KEY",
		"-e", "GITHUB_PAT",
		"-e", "REPO=" + opts.Repo,
		"-e", "BRANCH=" + opts.Branch,
		"-e", "PLAN_FILE=" + opts.PlanFile,
		"-e", "SPECS_DIR=" + opts.SpecsDir,
		"-v", opts.LogsDir + ":/app/logs",
		opts.ImageTag,
		"--",
		opts.Mode,
		strconv.Itoa(opts.MaxIter),
	}

	if err := runner.Run("docker", args...); err != nil {
		return fmt.Errorf("docker run: %w", err)
	}
	return nil
}
