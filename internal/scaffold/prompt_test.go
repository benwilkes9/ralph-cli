package scaffold

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// byteReader wraps a reader to deliver one byte at a time, preventing
// bufio.Scanner from over-buffering when huh creates a new scanner per field.
type byteReader struct {
	r io.Reader
}

func (br *byteReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return br.r.Read(p[:1])
}

func TestRunPrompts_AllDefaults(t *testing.T) {
	info := &ProjectInfo{
		RunCmd: "uv run uvicorn app:app",
	}
	// In accessible mode: 4 prompts, all accept defaults with empty lines.
	// Confirm default is true, empty line → yes.
	input := "\n\n\n\n"
	out := &bytes.Buffer{}

	err := RunPrompts(info, &PromptOptions{
		In:         &byteReader{strings.NewReader(input)},
		Out:        out,
		Accessible: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, "RunCmd", info.RunCmd, "uv run uvicorn app:app")
	assertEqual(t, "Goal", info.Goal, "")
	if !info.CreateSpecs {
		t.Error("expected CreateSpecs to default to true")
	}
	if len(info.EnvVars) != 0 {
		t.Errorf("expected no EnvVars, got %v", info.EnvVars)
	}
}

func TestRunPrompts_CustomValues(t *testing.T) {
	info := &ProjectInfo{
		RunCmd: "npm start",
	}
	// Input 1: override run command
	// Input 2: set goal
	// Confirm: "n" to decline specs
	// Input 3: env vars
	input := "go run ./cmd/server\nBuild a REST API\nn\nDATABASE_URL, REDIS_URL\n"
	out := &bytes.Buffer{}

	err := RunPrompts(info, &PromptOptions{
		In:         &byteReader{strings.NewReader(input)},
		Out:        out,
		Accessible: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, "RunCmd", info.RunCmd, "go run ./cmd/server")
	assertEqual(t, "Goal", info.Goal, "Build a REST API")
	if info.CreateSpecs {
		t.Error("expected CreateSpecs to be false")
	}
	if len(info.EnvVars) != 2 {
		t.Fatalf("expected 2 EnvVars, got %v", info.EnvVars)
	}
	assertEqual(t, "EnvVars[0]", info.EnvVars[0], "DATABASE_URL")
	assertEqual(t, "EnvVars[1]", info.EnvVars[1], "REDIS_URL")
}

func TestRunPrompts_OutputContainsPrompts(t *testing.T) {
	info := &ProjectInfo{}
	input := "\n\n\n\n"
	out := &bytes.Buffer{}

	if err := RunPrompts(info, &PromptOptions{
		In:         &byteReader{strings.NewReader(input)},
		Out:        out,
		Accessible: true,
	}); err != nil {
		t.Fatal(err)
	}

	output := out.String()
	if !strings.Contains(output, "Run command") {
		t.Error("expected output to contain 'Run command'")
	}
	if !strings.Contains(output, "Project goal") {
		t.Error("expected output to contain 'Project goal'")
	}
	if !strings.Contains(output, "specs") {
		t.Error("expected output to contain 'specs'")
	}
	if !strings.Contains(output, "env vars") {
		t.Error("expected output to contain 'env vars'")
	}
}

func TestRunPrompts_YesNoExplicitYes(t *testing.T) {
	info := &ProjectInfo{}
	// Empty for inputs, "y" for confirm, empty for env vars
	input := "\n\ny\n\n"
	out := &bytes.Buffer{}

	if err := RunPrompts(info, &PromptOptions{
		In:         &byteReader{strings.NewReader(input)},
		Out:        out,
		Accessible: true,
	}); err != nil {
		t.Fatal(err)
	}

	if !info.CreateSpecs {
		t.Error("expected CreateSpecs to be true with explicit 'y'")
	}
}
