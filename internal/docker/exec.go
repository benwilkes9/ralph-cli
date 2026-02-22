package docker

import (
	"context"
	"os"
	"os/exec"
)

// CommandRunner abstracts subprocess invocation for testing.
type CommandRunner interface {
	Run(name string, args ...string) error
}

type defaultRunner struct{}

func (defaultRunner) Run(name string, args ...string) error {
	cmd := exec.CommandContext(context.Background(), name, args...) //nolint:gosec // args are validated by callers
	cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run() //nolint:wrapcheck // callers wrap with context
}
