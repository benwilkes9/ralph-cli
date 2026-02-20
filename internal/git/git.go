package git

import (
	"context"
	"errors"
	"fmt"
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

// Add stages the given paths.
func Add(paths ...string) error {
	args := append([]string{"add"}, paths...)
	_, err := run(args...)
	return err
}

// Commit creates a commit with the given message.
func Commit(message string) error {
	_, err := run("commit", "-m", message)
	return err
}

// Push pushes the given branch to origin.
func Push(branch string) error {
	_, err := run("push", "origin", branch)
	return err
}

// PushSetUpstream pushes and sets the upstream tracking branch.
func PushSetUpstream(branch string) error {
	_, err := run("push", "-u", "origin", branch)
	return err
}

// PullRebase performs a pull --rebase on the given branch.
func PullRebase(branch string) error {
	_, err := run("pull", "--rebase", "origin", branch)
	return err
}

// RemoteURL returns the URL configured for the given remote.
func RemoteURL(name string) (string, error) {
	out, err := run("remote", "get-url", name)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// RepoRoot returns the top-level directory of the git repo.
func RepoRoot() (string, error) {
	out, err := run("rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// IsTracked returns true if the given path is tracked by git (committed).
func IsTracked(path string) (bool, error) {
	_, err := run("ls-files", "--error-unmatch", path)
	if err != nil {
		// Exit code 1 means not tracked — not an error for our purposes.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// BranchExistsOnRemote returns true if the branch exists on the origin remote.
func BranchExistsOnRemote(branch string) (bool, error) {
	out, err := run("ls-remote", "--heads", "origin", "refs/heads/"+branch)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// DiffFromRemote returns the diff output for the given path between HEAD and origin/branch.
// A non-empty result means there are unpushed changes at that path.
func DiffFromRemote(branch, path string) (string, error) {
	out, err := run("diff", "origin/"+branch, "--", path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func run(args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("git: no subcommand specified")
	}
	cmd := exec.CommandContext(context.Background(), "git", args...) //nolint:gosec // args are hardcoded by callers in this package
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return string(out), nil
}
