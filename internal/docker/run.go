package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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
}

// Run executes docker run with the given options, attaching stdin/stdout/stderr.
func Run(opts *RunOptions) error {
	args := []string{
		"run", "--rm", "-it",
		"-e", "ANTHROPIC_API_KEY",
		"-e", "GITHUB_PAT",
		"-e", "REPO=" + opts.Repo,
		"-e", "BRANCH=" + opts.Branch,
		"-v", opts.LogsDir + ":/app/logs",
		opts.ImageTag,
		opts.Mode,
		strconv.Itoa(opts.MaxIter),
	}

	cmd := exec.CommandContext(context.Background(), "docker", args...) //nolint:gosec // args are constructed from validated options
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker run: %w", err)
	}
	return nil
}
