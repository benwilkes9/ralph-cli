package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// Default values for docker build.
const (
	DefaultTag        = "ralph-loop"
	DefaultDockerfile = ".ralph/docker/Dockerfile"
	DefaultContext    = "."
)

// Build runs docker build with BuildKit enabled.
func Build(dockerfile, tag, contextDir string) error {
	if dockerfile == "" {
		dockerfile = DefaultDockerfile
	}
	if tag == "" {
		tag = DefaultTag
	}
	if contextDir == "" {
		contextDir = DefaultContext
	}

	cmd := exec.CommandContext(context.Background(), "docker", "build", "-t", tag, "-f", dockerfile, contextDir) //nolint:gosec // user-controlled paths
	cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build: %w", err)
	}
	return nil
}
