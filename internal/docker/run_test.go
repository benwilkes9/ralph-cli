package docker

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseRunOpts() *RunOptions {
	return &RunOptions{
		ImageTag: "ralph-loop",
		Mode:     "build",
		MaxIter:  5,
		Branch:   "main",
		Repo:     "owner/repo",
		LogsDir:  "/tmp/logs",
		PlanFile: ".ralph/plans/PLAN.md",
		SpecsDir: "specs",
	}
}

func TestRunWithRunner_SecurityOptions(t *testing.T) {
	r := &fakeRunner{}
	require.NoError(t, runWithRunner(r, baseRunOpts()))

	require.Len(t, r.calls, 1)
	call := r.calls[0]
	assert.Equal(t, "docker", call[0])
	assert.Contains(t, call, "--rm")
	assert.Contains(t, call, "-it")
	assert.Contains(t, call, "--security-opt")
	assert.Contains(t, call, "no-new-privileges")
}

func TestRunWithRunner_EnvVars(t *testing.T) {
	r := &fakeRunner{}
	require.NoError(t, runWithRunner(r, baseRunOpts()))

	call := r.calls[0]
	assert.Contains(t, call, "ANTHROPIC_API_KEY")
	assert.Contains(t, call, "GITHUB_PAT")
	assert.Contains(t, call, "REPO=owner/repo")
	assert.Contains(t, call, "BRANCH=main")
	assert.Contains(t, call, "PLAN_FILE=.ralph/plans/PLAN.md")
	assert.Contains(t, call, "SPECS_DIR=specs")
}

func TestRunWithRunner_VolumeMount(t *testing.T) {
	r := &fakeRunner{}
	require.NoError(t, runWithRunner(r, baseRunOpts()))

	assert.Contains(t, r.calls[0], "/tmp/logs:/app/logs")
}

func TestRunWithRunner_PositionalArgs(t *testing.T) {
	r := &fakeRunner{}
	require.NoError(t, runWithRunner(r, baseRunOpts()))

	call := r.calls[0]
	dashIdx := -1
	for i, a := range call {
		if a == "--" {
			dashIdx = i
			break
		}
	}
	require.NotEqual(t, -1, dashIdx, "expected '--' separator in docker run args")

	trailing := call[dashIdx+1:]
	require.Len(t, trailing, 2)
	assert.Equal(t, "build", trailing[0])
	assert.Equal(t, "5", trailing[1])
}

func TestRunWithRunner_ImageTag(t *testing.T) {
	r := &fakeRunner{}
	opts := baseRunOpts()
	opts.ImageTag = "custom-image:v2"
	require.NoError(t, runWithRunner(r, opts))

	assert.Contains(t, r.calls[0], "custom-image:v2")
}

func TestRunWithRunner_WrapsError(t *testing.T) {
	r := &fakeRunner{errFor: map[string]error{"docker": errors.New("run failed")}}

	err := runWithRunner(r, baseRunOpts())
	require.Error(t, err)
	assert.ErrorContains(t, err, "docker run:")
}
