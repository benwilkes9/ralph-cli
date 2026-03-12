package loop

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/benwilkes9/ralph-cli/internal/state"
	"github.com/benwilkes9/ralph-cli/internal/stream"
	"github.com/benwilkes9/ralph-cli/internal/ui"
)

// --- fakes ---

type fakeGit struct {
	heads          []string
	headIdx        int
	pushErr        error
	upstreamCalled bool
	// Additional dirs support.
	additionalHeads map[string][]string // dir → sequence of HEADs
	additionalIdx   map[string]int
	pushedDirs      []string
}

func (f *fakeGit) Head() (string, error) {
	if len(f.heads) == 0 {
		return "0000000000000000000000000000000000000000", nil
	}
	sha := f.heads[f.headIdx%len(f.heads)]
	f.headIdx++
	return sha, nil
}

func (f *fakeGit) Push(_ string) error { return f.pushErr }

func (f *fakeGit) PushSetUpstream(_ string) error {
	f.upstreamCalled = true
	return nil
}

func (f *fakeGit) HeadIn(dir string) (string, error) {
	heads, ok := f.additionalHeads[dir]
	if !ok || len(heads) == 0 {
		return "0000000000000000000000000000000000000000", nil
	}
	if f.additionalIdx == nil {
		f.additionalIdx = make(map[string]int)
	}
	idx := f.additionalIdx[dir] % len(heads)
	f.additionalIdx[dir]++
	return heads[idx], nil
}

func (f *fakeGit) PushIn(dir, _ string) error {
	f.pushedDirs = append(f.pushedDirs, dir)
	return f.pushErr
}

func (f *fakeGit) PushSetUpstreamIn(dir, _ string) error {
	f.pushedDirs = append(f.pushedDirs, dir)
	f.upstreamCalled = true
	return nil
}

type fakeClaude struct {
	stats  *stream.IterationStats
	err    error
	called int
}

func (f *fakeClaude) Run(_ context.Context, _ *Options, logW, _ io.Writer) (*stream.IterationStats, error) {
	f.called++
	fmt.Fprintln(logW, `{}`) //nolint:errcheck // test helper write
	return f.stats, f.err
}

// --- helpers ---

func baseOpts(t *testing.T) *Options {
	t.Helper()
	return &Options{
		Mode:          ModeBuild,
		PromptFile:    filepath.Join(t.TempDir(), "prompt.md"),
		MaxIterations: 1,
		LogsDir:       t.TempDir(),
		StateFile:     filepath.Join(t.TempDir(), "state.json"),
		Branch:        "main",
	}
}

func iterStats() *stream.IterationStats {
	return &stream.IterationStats{PeakContext: 1000, Cost: 0.01}
}

var runTheme = ui.DefaultTheme()

// --- tests ---

func TestRun_MaxIterationsExits(t *testing.T) {
	opts := baseOpts(t)
	opts.MaxIterations = 1

	g := &fakeGit{heads: []string{"sha-a", "sha-b"}}
	c := &fakeClaude{stats: iterStats()}

	var buf bytes.Buffer
	err := run(context.Background(), opts, &buf, runTheme, g, c)
	require.NoError(t, err)

	assert.Equal(t, 1, c.called)

	st, loadErr := state.Load(opts.StateFile)
	require.NoError(t, loadErr)
	require.Len(t, st.Runs, 1)
	assert.Equal(t, 1, st.Runs[0].Iterations)
	assert.Equal(t, state.StatusMaxIterations, st.Runs[0].Status)
}

func TestRun_CancellationBeforeLoop(t *testing.T) {
	opts := baseOpts(t)
	opts.MaxIterations = 10

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled

	g := &fakeGit{heads: []string{"sha-a"}}
	c := &fakeClaude{stats: iterStats()}

	var buf bytes.Buffer
	err := run(ctx, opts, &buf, runTheme, g, c)
	assert.True(t, errors.Is(err, context.Canceled))
	assert.Equal(t, 0, c.called, "claude should not be invoked when context is already cancelled")

	st, loadErr := state.Load(opts.StateFile)
	require.NoError(t, loadErr)
	require.Len(t, st.Runs, 1)
	assert.Equal(t, state.StatusCancelled, st.Runs[0].Status)
}

func TestRun_StaleAbort(t *testing.T) {
	opts := baseOpts(t)
	opts.MaxIterations = 10

	// All Head() calls return the same SHA → stale after DefaultMaxStale iterations.
	g := &fakeGit{heads: []string{"same-sha"}}
	c := &fakeClaude{stats: iterStats()}

	var buf bytes.Buffer
	err := run(context.Background(), opts, &buf, runTheme, g, c)
	require.NoError(t, err)

	assert.Equal(t, DefaultMaxStale, c.called)

	st, loadErr := state.Load(opts.StateFile)
	require.NoError(t, loadErr)
	require.Len(t, st.Runs, 1)
	assert.Equal(t, state.StatusStaleAbort, st.Runs[0].Status)
}

func TestRun_AlternatingHeadsNoStale(t *testing.T) {
	opts := baseOpts(t)
	opts.MaxIterations = 3

	// Alternating HEADs means no stale detection triggers.
	g := &fakeGit{heads: []string{"sha-a", "sha-b"}}
	c := &fakeClaude{stats: iterStats()}

	var buf bytes.Buffer
	err := run(context.Background(), opts, &buf, runTheme, g, c)
	require.NoError(t, err)

	assert.Equal(t, 3, c.called)

	st, loadErr := state.Load(opts.StateFile)
	require.NoError(t, loadErr)
	require.Len(t, st.Runs, 1)
	assert.Equal(t, state.StatusMaxIterations, st.Runs[0].Status)
}

func TestRun_PushFallback(t *testing.T) {
	opts := baseOpts(t)
	opts.MaxIterations = 1

	g := &fakeGit{
		heads:   []string{"sha-a", "sha-b"},
		pushErr: errors.New("no upstream"),
	}
	c := &fakeClaude{stats: iterStats()}

	var buf bytes.Buffer
	err := run(context.Background(), opts, &buf, runTheme, g, c)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Failed to push")
	assert.Contains(t, out, "Creating remote branch")
	assert.True(t, g.upstreamCalled, "PushSetUpstream should be called on push failure")
}

func TestClaudeArgs(t *testing.T) {
	args := claudeArgs(nil)
	assert.Contains(t, args, "-p")
	assert.Contains(t, args, "--dangerously-skip-permissions")
	assert.Contains(t, args, "--output-format=stream-json")
	assert.Contains(t, args, "--model")
}

func TestClaudeArgs_WithAdditionalDirs(t *testing.T) {
	args := claudeArgs([]string{"/workspace/repo-a", "/workspace/repo-b"})
	assert.Contains(t, args, "--add-dir")

	// Verify both dirs appear after --add-dir flags.
	count := 0
	for i, a := range args {
		if a == "--add-dir" {
			count++
			require.Less(t, i+1, len(args))
		}
	}
	assert.Equal(t, 2, count)
}

func TestRun_AdditionalDir_ChangeResetsStale(t *testing.T) {
	opts := baseOpts(t)
	opts.MaxIterations = 3
	opts.AdditionalDirs = []string{"/workspace/repo-b"}

	// Primary repo HEAD never changes, but additional repo does.
	g := &fakeGit{
		heads: []string{"same-sha"},
		additionalHeads: map[string][]string{
			"/workspace/repo-b": {"sha-x", "sha-x", "sha-x", "sha-y", "sha-y", "sha-y", "sha-z", "sha-z"},
		},
	}
	c := &fakeClaude{stats: iterStats()}

	var buf bytes.Buffer
	err := run(context.Background(), opts, &buf, runTheme, g, c)
	require.NoError(t, err)

	// Should run all 3 iterations (not stale-abort) because the additional repo changes.
	assert.Equal(t, 3, c.called)
}

func TestRun_AdditionalDir_AllUnchangedIsStale(t *testing.T) {
	opts := baseOpts(t)
	opts.MaxIterations = 10
	opts.AdditionalDirs = []string{"/workspace/repo-b"}

	// Both repos never change → stale.
	g := &fakeGit{
		heads: []string{"same-sha"},
		additionalHeads: map[string][]string{
			"/workspace/repo-b": {"same-sha-b"},
		},
	}
	c := &fakeClaude{stats: iterStats()}

	var buf bytes.Buffer
	err := run(context.Background(), opts, &buf, runTheme, g, c)
	require.NoError(t, err)

	assert.Equal(t, DefaultMaxStale, c.called)
}

func TestRun_AdditionalDir_PushedOnChange(t *testing.T) {
	opts := baseOpts(t)
	opts.MaxIterations = 1
	opts.AdditionalDirs = []string{"/workspace/repo-b"}

	// Both repos change.
	g := &fakeGit{
		heads: []string{"sha-a", "sha-b"},
		additionalHeads: map[string][]string{
			"/workspace/repo-b": {"sha-x", "sha-x", "sha-y"},
		},
	}
	c := &fakeClaude{stats: iterStats()}

	var buf bytes.Buffer
	err := run(context.Background(), opts, &buf, runTheme, g, c)
	require.NoError(t, err)

	assert.Contains(t, g.pushedDirs, "/workspace/repo-b")
}

func TestSaveState_PersistsRecord(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	opts := &Options{
		Mode:      ModePlan,
		StateFile: stateFile,
	}
	cumStats := &stream.CumulativeStats{
		Iterations:  3,
		TotalCost:   1.23,
		PeakContext: 50000,
	}
	logPaths := []string{"logs/a.jsonl", "logs/b.jsonl"}

	saveState(opts, cumStats, time.Now(), logPaths, false, false)

	st, err := state.Load(stateFile)
	require.NoError(t, err)
	require.Len(t, st.Runs, 1)

	r := st.Runs[0]
	assert.Equal(t, "plan", r.Mode)
	assert.Equal(t, 3, r.Iterations)
	assert.Equal(t, 1.23, r.TotalCost)
	assert.Equal(t, state.StatusCompleted, r.Status)
	assert.Equal(t, logPaths, r.LogFiles)
}
