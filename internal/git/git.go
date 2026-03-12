package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
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

// HasStagedChanges returns true if there are changes in the staging area.
func HasStagedChanges() (bool, error) {
	cmd := exec.CommandContext(context.Background(), "git", "diff", "--cached", "--quiet") //nolint:gosec // hardcoded args
	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return true, nil // exit 1 = there are staged changes
		}
		return false, fmt.Errorf("git diff --cached: %w", err)
	}
	return false, nil // exit 0 = clean
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

// unsafeChars matches characters that are not alphanumeric, hyphens, underscores, or dots.
var unsafeChars = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

// SanitizeBranch converts a branch name into a safe filesystem-friendly string.
// Slashes are replaced with hyphens, unsafe characters are stripped, and
// leading/trailing hyphens are trimmed.
func SanitizeBranch(branch string) string {
	s := strings.ReplaceAll(branch, "/", "-")
	s = unsafeChars.ReplaceAllString(s, "")
	s = strings.Trim(s, "-")
	return s
}

// IsProtectedBranch returns true if branch matches any entry in the protected
// list (case-insensitive comparison).
func IsProtectedBranch(branch string, protected []string) bool {
	for _, p := range protected {
		if strings.EqualFold(branch, p) {
			return true
		}
	}
	return false
}

// IsGitRepo returns true if dir is a git repository.
func IsGitRepo(dir string) bool {
	_, err := runIn(dir, "rev-parse", "--git-dir")
	return err == nil
}

// HeadIn returns the HEAD commit hash for the repo at dir.
func HeadIn(dir string) (string, error) {
	out, err := runIn(dir, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// BranchIn returns the current branch name for the repo at dir.
func BranchIn(dir string) (string, error) {
	out, err := runIn(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// PushIn pushes the given branch to origin for the repo at dir.
func PushIn(dir, branch string) error {
	_, err := runIn(dir, "push", "origin", branch)
	return err
}

// PushSetUpstreamIn pushes and sets upstream for the repo at dir.
func PushSetUpstreamIn(dir, branch string) error {
	_, err := runIn(dir, "push", "-u", "origin", branch)
	return err
}

// BranchExistsOnRemoteIn returns true if the branch exists on origin for the repo at dir.
func BranchExistsOnRemoteIn(dir, branch string) (bool, error) {
	out, err := runIn(dir, "ls-remote", "--heads", "origin", "refs/heads/"+branch)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func runIn(dir string, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("git: no subcommand specified")
	}
	full := append([]string{"-C", dir}, args...)                     //nolint:gocritic // building new slice intentionally
	cmd := exec.CommandContext(context.Background(), "git", full...) //nolint:gosec // args are hardcoded by callers in this package
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("git %s: %w\n%s", args[0], err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return string(out), nil
}

func run(args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("git: no subcommand specified")
	}
	cmd := exec.CommandContext(context.Background(), "git", args...) //nolint:gosec // args are hardcoded by callers in this package
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("git %s: %w\n%s", args[0], err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return string(out), nil
}
