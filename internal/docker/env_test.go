package docker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFile(t *testing.T) {
	t.Run("basic key-value pairs", func(t *testing.T) {
		f := writeTemp(t, "FOO=bar\nBAZ=qux\n")
		env, err := LoadEnvFile(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertEnv(t, env, "FOO", "bar")
		assertEnv(t, env, "BAZ", "qux")
	})

	t.Run("comments and blank lines", func(t *testing.T) {
		f := writeTemp(t, "# comment\n\nKEY=val\n  # indented comment\n")
		env, err := LoadEnvFile(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(env) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(env))
		}
		assertEnv(t, env, "KEY", "val")
	})

	t.Run("double-quoted value", func(t *testing.T) {
		f := writeTemp(t, `KEY="hello world"`)
		env, err := LoadEnvFile(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertEnv(t, env, "KEY", "hello world")
	})

	t.Run("single-quoted value", func(t *testing.T) {
		f := writeTemp(t, `KEY='hello world'`)
		env, err := LoadEnvFile(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertEnv(t, env, "KEY", "hello world")
	})

	t.Run("equals in value", func(t *testing.T) {
		f := writeTemp(t, "KEY=a=b=c\n")
		env, err := LoadEnvFile(f)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertEnv(t, env, "KEY", "a=b=c")
	})

	t.Run("missing file returns empty map", func(t *testing.T) {
		env, err := LoadEnvFile(filepath.Join(t.TempDir(), "nope"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(env) != 0 {
			t.Fatalf("expected empty map, got %v", env)
		}
	})
}

func TestValidateEnv(t *testing.T) {
	t.Run("all present in map", func(t *testing.T) {
		env := map[string]string{"A": "1", "B": "2"}
		if err := ValidateEnv(env, []string{"A", "B"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("missing key errors", func(t *testing.T) {
		env := map[string]string{"A": "1"}
		err := ValidateEnv(env, []string{"A", "MISSING"})
		if err == nil {
			t.Fatal("expected error for missing key")
		}
	})

	t.Run("falls back to os.Getenv", func(t *testing.T) {
		t.Setenv("TEST_RALPH_ENVVAR", "present")
		env := map[string]string{}
		if err := ValidateEnv(env, []string{"TEST_RALPH_ENVVAR"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestAllowedEnvVars(t *testing.T) {
	t.Run("allowed keys pass", func(t *testing.T) {
		env := map[string]string{
			"ANTHROPIC_API_KEY": "sk-test",
			"GITHUB_PAT":        "ghp_test",
		}
		for k := range env {
			if !allowedEnvVars[k] {
				t.Errorf("expected %q to be allowed", k)
			}
		}
	})

	t.Run("dangerous keys rejected", func(t *testing.T) {
		dangerous := []string{"PATH", "LD_PRELOAD", "http_proxy", "HOME"}
		for _, k := range dangerous {
			if allowedEnvVars[k] {
				t.Errorf("expected %q to be disallowed", k)
			}
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

func assertEnv(t *testing.T, env map[string]string, key, want string) {
	t.Helper()
	got, ok := env[key]
	if !ok {
		t.Fatalf("key %q not found in env", key)
	}
	if got != want {
		t.Errorf("env[%q] = %q, want %q", key, got, want)
	}
}
