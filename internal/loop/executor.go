package loop

import (
	"context"
	"io"

	"github.com/benwilkes9/ralph-cli/internal/git"
	"github.com/benwilkes9/ralph-cli/internal/stream"
	"github.com/benwilkes9/ralph-cli/internal/ui"
)

// GitClient abstracts git operations used by the loop.
type GitClient interface {
	Head() (string, error)
	Push(branch string) error
	PushSetUpstream(branch string) error
	HeadIn(dir string) (string, error)
	PushIn(dir, branch string) error
	PushSetUpstreamIn(dir, branch string) error
}

// ClaudeRunner abstracts the claude CLI subprocess.
type ClaudeRunner interface {
	Run(ctx context.Context, opts *Options, logW, displayW io.Writer) (*stream.IterationStats, error)
}

type realGitClient struct{}

func (r *realGitClient) Head() (string, error) {
	return git.Head() //nolint:wrapcheck // thin adapter
}

func (r *realGitClient) Push(branch string) error {
	return git.Push(branch) //nolint:wrapcheck // thin adapter
}

func (r *realGitClient) PushSetUpstream(branch string) error {
	return git.PushSetUpstream(branch) //nolint:wrapcheck // thin adapter
}

func (r *realGitClient) HeadIn(dir string) (string, error) {
	return git.HeadIn(dir) //nolint:wrapcheck // thin adapter
}

func (r *realGitClient) PushIn(dir, branch string) error {
	return git.PushIn(dir, branch) //nolint:wrapcheck // thin adapter
}

func (r *realGitClient) PushSetUpstreamIn(dir, branch string) error {
	return git.PushSetUpstreamIn(dir, branch) //nolint:wrapcheck // thin adapter
}

type realClaudeRunner struct {
	theme *ui.Theme
}

func (r *realClaudeRunner) Run(ctx context.Context, opts *Options, logW, displayW io.Writer) (*stream.IterationStats, error) {
	return runClaude(ctx, opts, logW, displayW, r.theme)
}
