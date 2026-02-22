package docker

import (
	"fmt"
)

// Default values for docker build.
const (
	DefaultTag        = "ralph-loop"
	DefaultDockerfile = ".ralph/docker/Dockerfile"
	DefaultContext    = "."
)

// Build runs docker build with BuildKit enabled.
func Build(dockerfile, tag, contextDir string) error {
	return buildWithRunner(defaultRunner{}, dockerfile, tag, contextDir)
}

func buildWithRunner(runner CommandRunner, dockerfile, tag, contextDir string) error {
	if dockerfile == "" {
		dockerfile = DefaultDockerfile
	}
	if tag == "" {
		tag = DefaultTag
	}
	if contextDir == "" {
		contextDir = DefaultContext
	}

	if err := runner.Run("docker", "build", "-t", tag, "-f", dockerfile, contextDir); err != nil {
		return fmt.Errorf("docker build: %w", err)
	}
	return nil
}
