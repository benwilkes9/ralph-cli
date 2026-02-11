package docker

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadEnvFile parses a KEY=VALUE env file. It skips blank lines and # comments,
// and strips surrounding quotes from values. Returns an empty map (not error) if
// the file does not exist.
func LoadEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("opening env file: %w", err)
	}
	defer f.Close() //nolint:errcheck // read-only

	env := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = stripQuotes(value)

		if key != "" {
			env[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading env file: %w", err)
	}
	return env, nil
}

// ValidateEnv checks that all required keys are present in the env map or in
// os.Getenv as a fallback. Returns an error listing any missing keys.
func ValidateEnv(env map[string]string, required []string) error {
	var missing []string
	for _, key := range required {
		if _, ok := env[key]; ok {
			continue
		}
		if os.Getenv(key) != "" {
			continue
		}
		missing = append(missing, key)
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}
	return nil
}

func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
