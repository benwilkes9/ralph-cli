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

func TestRunPrompts_SelectDefaults(t *testing.T) {
	info := &ProjectInfo{
		ProjectName:    "myapp",
		Language:       LangPython,
		PackageManager: PmUV,
	}
	// Accessible mode: select option 1 for run command, select option 1 for goal.
	// Hidden input groups are skipped automatically.
	input := "1\n1\n"
	out := &bytes.Buffer{}

	err := RunPrompts(info, &PromptOptions{
		In:         &byteReader{strings.NewReader(input)},
		Out:        out,
		Accessible: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, "RunCmd", info.RunCmd, "uv run uvicorn myapp.main:app")
	assertEqual(t, "Goal", info.Goal, "Production-ready REST API")
}

func TestRunPrompts_CustomValues(t *testing.T) {
	info := &ProjectInfo{
		ProjectName:    "myapp",
		Language:       LangPython,
		PackageManager: PmUV,
	}
	// Select "Type something." (option 3) for run cmd, then type custom value.
	// Select "Type something." (option 3) for goal, then type custom value.
	input := "3\nmy custom cmd\n3\nmy custom goal\n"
	out := &bytes.Buffer{}

	err := RunPrompts(info, &PromptOptions{
		In:         &byteReader{strings.NewReader(input)},
		Out:        out,
		Accessible: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, "RunCmd", info.RunCmd, "my custom cmd")
	assertEqual(t, "Goal", info.Goal, "my custom goal")
}

func TestRunPrompts_OutputContainsPrompts(t *testing.T) {
	info := &ProjectInfo{
		ProjectName:    "myapp",
		Language:       LangPython,
		PackageManager: PmUV,
	}
	input := "1\n1\n"
	out := &bytes.Buffer{}

	if err := RunPrompts(info, &PromptOptions{
		In:         &byteReader{strings.NewReader(input)},
		Out:        out,
		Accessible: true,
	}); err != nil {
		t.Fatal(err)
	}

	output := out.String()
	if !strings.Contains(output, "How do you start") {
		t.Error("expected output to contain 'How do you start'")
	}
	if !strings.Contains(output, "ultimate goal") {
		t.Error("expected output to contain 'ultimate goal'")
	}
}

func TestRunPrompts_UnknownLanguageSelectsCustom(t *testing.T) {
	info := &ProjectInfo{
		ProjectName:    "myapp",
		Language:       LangUnknown,
		PackageManager: PmUnknown,
	}
	// Unknown PM: only "Type something." (option 1) for run → shows input.
	// Unknown lang: "Learning spike" (option 1), "Type something." (option 2) for goal.
	input := "1\nmy run cmd\n1\n"
	out := &bytes.Buffer{}

	err := RunPrompts(info, &PromptOptions{
		In:         &byteReader{strings.NewReader(input)},
		Out:        out,
		Accessible: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertEqual(t, "RunCmd", info.RunCmd, "my run cmd")
	assertEqual(t, "Goal", info.Goal, "Learning spike / reference")
}
