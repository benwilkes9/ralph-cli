package docker

import (
	"fmt"
	"strings"

	"github.com/benwilkes9/ralph-cli/internal/git"
)

// RepoSlug extracts "owner/repo" from a git remote URL.
// Supports HTTPS (https://github.com/o/r.git) and SSH (git@github.com:o/r.git).
func RepoSlug(remoteURL string) (string, error) {
	url := strings.TrimSpace(remoteURL)
	url = strings.TrimSuffix(url, "/")
	url = strings.TrimSuffix(url, ".git")

	var path string
	switch {
	case strings.HasPrefix(url, "git@"):
		// git@github.com:owner/repo
		_, after, ok := strings.Cut(url, ":")
		if !ok {
			return "", fmt.Errorf("invalid SSH remote URL: %s", remoteURL)
		}
		path = after
	case strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://"):
		// https://github.com/owner/repo
		parts := strings.SplitN(url, "//", 2)
		if len(parts) < 2 {
			return "", fmt.Errorf("invalid HTTPS remote URL: %s", remoteURL)
		}
		// Strip host: github.com/owner/repo → owner/repo
		_, after, ok := strings.Cut(parts[1], "/")
		if !ok {
			return "", fmt.Errorf("invalid HTTPS remote URL: %s", remoteURL)
		}
		path = after
	default:
		return "", fmt.Errorf("unsupported remote URL format: %s", remoteURL)
	}

	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("cannot extract owner/repo from: %s", remoteURL)
	}

	return parts[0] + "/" + parts[1], nil
}

// DetectRepo returns the "owner/repo" slug for the origin remote.
func DetectRepo() (string, error) {
	url, err := git.RemoteURL("origin")
	if err != nil {
		return "", fmt.Errorf("getting origin remote URL: %w", err)
	}
	return RepoSlug(url)
}
