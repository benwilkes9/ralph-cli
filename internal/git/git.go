package git

import (
	"context"
	"os/exec"
	"strings"
)

// Head returns the current HEAD commit hash.
func Head() (string, error) {
	out, err := run("rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Branch returns the current branch name.
func Branch() (string, error) {
	out, err := run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Push pushes the given branch to origin.
func Push(branch string) error {
	_, err := run("push", "origin", branch)
	return err
}

// PullRebase performs a pull --rebase on the given branch.
func PullRebase(branch string) error {
	_, err := run("pull", "--rebase", "origin", branch)
	return err
}

// RepoRoot returns the top-level directory of the git repo.
func RepoRoot() (string, error) {
	out, err := run("rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func run(args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
