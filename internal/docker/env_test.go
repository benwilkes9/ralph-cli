package docker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEnvFile(t *testing.T) {
	t.Run("basic key-value pairs", func(t *testing.T) {
		f := writeTemp(t, "FOO=bar\nBAZ=qux\n")
		env, err := LoadEnvFile(f)
		require.NoError(t, err)
		assert.Equal(t, "bar", env["FOO"])
		assert.Equal(t, "qux", env["BAZ"])
	})

	t.Run("comments and blank lines", func(t *testing.T) {
		f := writeTemp(t, "# comment\n\nKEY=val\n  # indented comment\n")
		env, err := LoadEnvFile(f)
		require.NoError(t, err)
		assert.Len(t, env, 1)
		assert.Equal(t, "val", env["KEY"])
	})

	t.Run("double-quoted value", func(t *testing.T) {
		f := writeTemp(t, `KEY="hello world"`)
		env, err := LoadEnvFile(f)
		require.NoError(t, err)
		assert.Equal(t, "hello world", env["KEY"])
	})

	t.Run("single-quoted value", func(t *testing.T) {
		f := writeTemp(t, `KEY='hello world'`)
		env, err := LoadEnvFile(f)
		require.NoError(t, err)
		assert.Equal(t, "hello world", env["KEY"])
	})

	t.Run("equals in value", func(t *testing.T) {
		f := writeTemp(t, "KEY=a=b=c\n")
		env, err := LoadEnvFile(f)
		require.NoError(t, err)
		assert.Equal(t, "a=b=c", env["KEY"])
	})

	t.Run("missing file returns empty map", func(t *testing.T) {
		env, err := LoadEnvFile(filepath.Join(t.TempDir(), "nope"))
		require.NoError(t, err)
		assert.Empty(t, env)
	})
}

func TestValidateEnv(t *testing.T) {
	t.Run("all present in map", func(t *testing.T) {
		env := map[string]string{"A": "1", "B": "2"}
		require.NoError(t, ValidateEnv(env, []string{"A", "B"}))
	})

	t.Run("missing key errors", func(t *testing.T) {
		env := map[string]string{"A": "1"}
		err := ValidateEnv(env, []string{"A", "MISSING"})
		require.Error(t, err)
	})

	t.Run("falls back to os.Getenv", func(t *testing.T) {
		t.Setenv("TEST_RALPH_ENVVAR", "present")
		env := map[string]string{}
		require.NoError(t, ValidateEnv(env, []string{"TEST_RALPH_ENVVAR"}))
	})
}

func TestAllowedEnvVars(t *testing.T) {
	t.Run("allowed keys pass", func(t *testing.T) {
		allowed := []string{"ANTHROPIC_API_KEY", "GITHUB_PAT"}
		for _, k := range allowed {
			assert.True(t, allowedEnvVars[k], "expected %q to be allowed", k)
		}
	})

	t.Run("dangerous keys rejected", func(t *testing.T) {
		dangerous := []string{"PATH", "LD_PRELOAD", "http_proxy", "HOME"}
		for _, k := range dangerous {
			assert.False(t, allowedEnvVars[k], "expected %q to be disallowed", k)
		}
	})
}

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	return p
}
