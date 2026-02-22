package scaffold

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultsInput selects option 1 for all three prompts (run cmd, goal, specs dir).
const defaultsInput = "1\n1\n1\n"

// byteReader wraps a reader to deliver one byte at a time, preventing
// bufio.Scanner from over-buffering when huh creates a new scanner per field.
type byteReader struct {
	r io.Reader
}

func (br *byteReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return br.r.Read(p[:1]) //nolint:wrapcheck // test helper
}

func TestRunPrompts_SelectDefaults(t *testing.T) {
	info := &ProjectInfo{
		ProjectName:    "myapp",
		Language:       LangPython,
		PackageManager: PmUV,
		SpecsDir:       "specs",
	}
	// Accessible mode: select option 1 for run command, option 1 for goal, option 1 for specs dir.
	// Hidden input groups are skipped automatically.
	input := defaultsInput
	out := &bytes.Buffer{}

	err := RunPrompts(info, &PromptOptions{
		In:         &byteReader{strings.NewReader(input)},
		Out:        out,
		Accessible: true,
	})
	require.NoError(t, err)

	assertEqual(t, "RunCmd", info.RunCmd, "uv run uvicorn myapp.main:app")
	assertEqual(t, "Goal", info.Goal, "Production-ready REST API")
	assertEqual(t, "SpecsDir", info.SpecsDir, "specs")
}

func TestRunPrompts_CustomValues(t *testing.T) {
	info := &ProjectInfo{
		ProjectName:    "myapp",
		Language:       LangPython,
		PackageManager: PmUV,
		SpecsDir:       "specs",
	}
	// Select "Type something." (option 3) for run cmd, then type custom value.
	// Select "Type something." (option 3) for goal, then type custom value.
	// Select "Type something." (option 3) for specs dir, then type custom value.
	input := "3\nmy custom cmd\n3\nmy custom goal\n3\nmy/specs\n"
	out := &bytes.Buffer{}

	err := RunPrompts(info, &PromptOptions{
		In:         &byteReader{strings.NewReader(input)},
		Out:        out,
		Accessible: true,
	})
	require.NoError(t, err)

	assertEqual(t, "RunCmd", info.RunCmd, "my custom cmd")
	assertEqual(t, "Goal", info.Goal, "my custom goal")
	assertEqual(t, "SpecsDir", info.SpecsDir, "my/specs")
}

func TestRunPrompts_OutputContainsPrompts(t *testing.T) {
	info := &ProjectInfo{
		ProjectName:    "myapp",
		Language:       LangPython,
		PackageManager: PmUV,
		SpecsDir:       "specs",
	}
	input := defaultsInput
	out := &bytes.Buffer{}

	err := RunPrompts(info, &PromptOptions{
		In:         &byteReader{strings.NewReader(input)},
		Out:        out,
		Accessible: true,
	})
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "How do you start")
	assert.Contains(t, output, "ultimate goal")
	assert.Contains(t, output, "specs live")
}

func TestRunPrompts_UnknownLanguageSelectsCustom(t *testing.T) {
	info := &ProjectInfo{
		ProjectName:    "myapp",
		Language:       LangUnknown,
		PackageManager: PmUnknown,
		SpecsDir:       "specs",
	}
	// Unknown PM: only "Type something." (option 1) for run → shows input.
	// Unknown lang: "Learning spike" (option 1), "Type something." (option 2) for goal.
	// Specs dir: "specs" (option 1).
	input := "1\nmy run cmd\n1\n1\n"
	out := &bytes.Buffer{}

	err := RunPrompts(info, &PromptOptions{
		In:         &byteReader{strings.NewReader(input)},
		Out:        out,
		Accessible: true,
	})
	require.NoError(t, err)

	assertEqual(t, "RunCmd", info.RunCmd, "my run cmd")
	assertEqual(t, "Goal", info.Goal, "Learning spike / reference")
	assertEqual(t, "SpecsDir", info.SpecsDir, "specs")
}

func TestValidateSpecsDir(t *testing.T) {
	tests := []struct {
		name    string
		dir     string
		wantErr bool
	}{
		{"valid simple", "specs", false},
		{"valid nested", "docs/specs", false},
		{"path traversal", "../../etc", true},
		{"leading dotdot", "../outside", true},
		{"absolute path", "/tmp/specs", true},
		{"dot only", ".", true},
		{"dotdot only", "..", true},
		{"embedded dotdot", "specs/../../etc", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSpecsDir(tt.dir)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRunPrompts_ValidatesSpecsDir(t *testing.T) {
	info := &ProjectInfo{
		ProjectName:    "myapp",
		Language:       LangPython,
		PackageManager: PmUV,
		SpecsDir:       "../../etc", // simulate malicious default
	}
	// Select option 1 for all prompts (uses defaults/first options).
	// Specs dir select: option 1 = "specs", which overwrites the bad default.
	input := defaultsInput
	out := &bytes.Buffer{}

	err := RunPrompts(info, &PromptOptions{
		In:         &byteReader{strings.NewReader(input)},
		Out:        out,
		Accessible: true,
	})
	// Selecting "specs" (valid) should pass validation.
	require.NoError(t, err)
	assertEqual(t, "SpecsDir", info.SpecsDir, "specs")
}
