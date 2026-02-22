package docker

import (
	"errors"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseRunOpts() *RunOptions {
	return &RunOptions{
		ImageTag:       "ralph-loop",
		Mode:           "build",
		MaxIter:        5,
		Branch:         "main",
		ProjectDir:     "/home/user/project",
		PlanFile:       ".ralph/plans/PLAN.md",
		SpecsDir:       "specs",
		AllowedDomains: DefaultAllowedDomains,
		ProjectName:    "myproject",
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

func TestRunWithRunner_CapAddNetAdmin(t *testing.T) {
	r := &fakeRunner{}
	require.NoError(t, runWithRunner(r, baseRunOpts()))

	call := r.calls[0]
	assert.Contains(t, call, "--cap-add")
	assert.Contains(t, call, "NET_ADMIN")
}

func TestRunWithRunner_EnvVars(t *testing.T) {
	r := &fakeRunner{}
	require.NoError(t, runWithRunner(r, baseRunOpts()))

	call := r.calls[0]
	assert.Contains(t, call, "ANTHROPIC_API_KEY")
	assert.Contains(t, call, "GITHUB_PAT")
	assert.Contains(t, call, "BRANCH=main")
	assert.Contains(t, call, "PLAN_FILE=.ralph/plans/PLAN.md")
	assert.Contains(t, call, "SPECS_DIR=specs")
}

func TestRunWithRunner_AllowedDomainsEnvVar(t *testing.T) {
	r := &fakeRunner{}
	require.NoError(t, runWithRunner(r, baseRunOpts()))

	call := r.calls[0]
	assert.Contains(t, call, "ALLOWED_DOMAINS=api.anthropic.com,github.com,api.github.com,registry.npmjs.org")
}

func TestRunWithRunner_NoRepoEnvVar(t *testing.T) {
	r := &fakeRunner{}
	require.NoError(t, runWithRunner(r, baseRunOpts()))

	call := r.calls[0]
	for _, arg := range call {
		assert.NotContains(t, arg, "REPO=")
	}
}

func TestRunWithRunner_BindMount(t *testing.T) {
	r := &fakeRunner{}
	require.NoError(t, runWithRunner(r, baseRunOpts()))

	call := r.calls[0]
	if runtime.GOOS == osDarwin {
		assert.Contains(t, call, "/home/user/project:/workspace/repo:delegated")
	} else {
		assert.Contains(t, call, "/home/user/project:/workspace/repo")
	}
}

func TestRunWithRunner_NoLogsVolume(t *testing.T) {
	r := &fakeRunner{}
	require.NoError(t, runWithRunner(r, baseRunOpts()))

	call := r.calls[0]
	for _, arg := range call {
		assert.NotContains(t, arg, "/app/logs")
	}
}

func TestRunWithRunner_DepsVolume_Present(t *testing.T) {
	r := &fakeRunner{}
	opts := baseRunOpts()
	opts.DepsDir = "node_modules"
	require.NoError(t, runWithRunner(r, opts))

	assert.Contains(t, r.calls[0], "ralph-deps-myproject:/workspace/repo/node_modules")
	assert.Contains(t, r.calls[0], "DEPS_DIR=node_modules")
}

func TestRunWithRunner_DepsVolume_Absent(t *testing.T) {
	r := &fakeRunner{}
	opts := baseRunOpts()
	opts.DepsDir = ""
	require.NoError(t, runWithRunner(r, opts))

	call := r.calls[0]
	for _, arg := range call {
		assert.NotContains(t, arg, "ralph-deps-")
		assert.NotContains(t, arg, "DEPS_DIR=")
	}
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
