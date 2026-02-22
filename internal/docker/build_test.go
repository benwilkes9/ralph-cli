package docker

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildWithRunner_Defaults(t *testing.T) {
	r := &fakeRunner{}
	require.NoError(t, buildWithRunner(r, "", "", ""))

	require.Len(t, r.calls, 1)
	call := r.calls[0]
	assert.Equal(t, "docker", call[0])
	assert.Contains(t, call, DefaultTag)
	assert.Contains(t, call, DefaultDockerfile)
	assert.Contains(t, call, DefaultContext)
}

func TestBuildWithRunner_Args(t *testing.T) {
	r := &fakeRunner{}
	require.NoError(t, buildWithRunner(r, "my.Dockerfile", "mytag", "./ctx"))

	require.Len(t, r.calls, 1)
	assert.Equal(t, []string{"docker", "build", "-t", "mytag", "-f", "my.Dockerfile", "./ctx"}, r.calls[0])
}

func TestBuildWithRunner_WrapsError(t *testing.T) {
	r := &fakeRunner{errFor: map[string]error{"docker": errors.New("build failed")}}

	err := buildWithRunner(r, "", "", "")
	require.Error(t, err)
	assert.ErrorContains(t, err, "docker build:")
}
