package scaffold

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunPrompts_AllDefaults(t *testing.T) {
	info := &ProjectInfo{
		RunCmd: "uv run uvicorn app:app",
	}
	input := "\n\n\n\n" // accept all defaults
	out := &bytes.Buffer{}

	err := RunPrompts(info, &PromptOptions{
		In:  strings.NewReader(input),
		Out: out,
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
	input := "go run ./cmd/server\nBuild a REST API\nn\nDATABASE_URL, REDIS_URL\n"
	out := &bytes.Buffer{}

	err := RunPrompts(info, &PromptOptions{
		In:  strings.NewReader(input),
		Out: out,
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
		In:  strings.NewReader(input),
		Out: out,
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
	input := "\n\ny\n\n"
	out := &bytes.Buffer{}

	if err := RunPrompts(info, &PromptOptions{
		In:  strings.NewReader(input),
		Out: out,
	}); err != nil {
		t.Fatal(err)
	}

	if !info.CreateSpecs {
		t.Error("expected CreateSpecs to be true with explicit 'y'")
	}
}
