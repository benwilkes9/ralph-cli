package stream

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatTokensFloor(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0k"},
		{1099, "1.0k"},
		{1100, "1.1k"},
		{45399, "45.3k"}, // floor: 45399/100=453 → 45.3k not 45.4k
		{45400, "45.4k"},
		{999999, "999.9k"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
		{1550000, "1.5M"},
		{1559999, "1.5M"},
	}

	for _, tt := range tests {
		got := FormatTokens(tt.input)
		assert.Equal(t, tt.want, got, "FormatTokens(%d)", tt.input)
	}
}

func TestFormatTextBlock(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)

	evt := &Event{
		Type: "assistant",
		Message: &Message{
			Role: "assistant",
			Content: []ContentBlock{
				{Type: "text", Text: "Hello world"},
			},
		},
	}

	require.NoError(t, f.Format(evt))

	got := buf.String()
	assert.Contains(t, got, "Hello world")
	assert.Contains(t, got, Bold)
	assert.Contains(t, got, White)
}

func TestFormatRegularToolUse(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)

	evt := &Event{
		Type: "assistant",
		Message: &Message{
			Role: "assistant",
			Content: []ContentBlock{
				{
					Type:  "tool_use",
					Name:  "Read",
					Input: json.RawMessage(`{"file_path":"/workspace/repo/main.go"}`),
				},
			},
		},
	}

	require.NoError(t, f.Format(evt))

	got := buf.String()
	assert.Contains(t, got, "· Read")
	assert.Contains(t, got, "/workspace/repo/main.go")
	assert.Contains(t, got, Dim)
}

func TestFormatTaskToolUse(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)

	evt := &Event{
		Type: "assistant",
		Message: &Message{
			Role: "assistant",
			Content: []ContentBlock{
				{
					Type:  "tool_use",
					Name:  "Task",
					Input: json.RawMessage(`{"description":"Run all tests","prompt":"run pytest","subagent_type":"Bash"}`),
				},
			},
		},
	}

	require.NoError(t, f.Format(evt))

	got := buf.String()
	assert.Contains(t, got, "▶ Bash")
	assert.Contains(t, got, `"Run all tests"`)
	assert.Contains(t, got, BoldCyan)
}

func TestFormatTaskWithModelHint(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)

	evt := &Event{
		Type: "assistant",
		Message: &Message{
			Role: "assistant",
			Content: []ContentBlock{
				{
					Type:  "tool_use",
					Name:  "Task",
					Input: json.RawMessage(`{"description":"Analyze code","prompt":"analyze","subagent_type":"general-purpose","model":"opus","max_turns":5}`),
				},
			},
		},
	}

	require.NoError(t, f.Format(evt))

	got := buf.String()
	assert.Contains(t, got, "model=opus")
	assert.Contains(t, got, "max_turns=5")
}

func TestFormatSubagentCompleted(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)

	evt := &Event{
		Type: "user",
		ToolUseResult: &ToolUseResult{
			Status:            "completed",
			TotalTokens:       8769,
			TotalDurationMs:   18515,
			TotalToolUseCount: 3,
		},
	}

	require.NoError(t, f.Format(evt))

	got := buf.String()
	assert.Contains(t, got, "✓")
	assert.Contains(t, got, Green)
	assert.Contains(t, got, "19s")
	assert.Contains(t, got, "3 tool calls")
	assert.Contains(t, got, "8.7k tokens")
}

func TestFormatSubagentFailed(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)

	evt := &Event{
		Type: "user",
		ToolUseResult: &ToolUseResult{
			Status:      "error",
			TotalTokens: 5000,
		},
	}

	require.NoError(t, f.Format(evt))

	got := buf.String()
	assert.Contains(t, got, "✗")
	assert.Contains(t, got, "error")
	assert.Contains(t, got, BoldRed)
}

func TestFormatIgnoresSystemAndResult(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)

	for _, evtType := range []string{"system", "result"} {
		buf.Reset()
		evt := &Event{Type: evtType}
		require.NoError(t, f.Format(evt))
		assert.Equal(t, 0, buf.Len(), "expected no output for %q event", evtType)
	}
}

func TestFormatUserWithoutTotalTokens(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)

	evt := &Event{
		Type: "user",
		ToolUseResult: &ToolUseResult{
			Stdout: "some output",
		},
	}

	require.NoError(t, f.Format(evt))
	assert.Equal(t, 0, buf.Len())
}

func TestExtractParamPriority(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"file_path first",
			`{"file_path":"/foo/bar","command":"ls"}`,
			"/foo/bar",
		},
		{
			"description over command",
			`{"description":"list files","command":"ls -la"}`,
			"list files",
		},
		{
			"command when no higher priority",
			`{"command":"git status","timeout":5000}`,
			"git status",
		},
		{
			"pattern extraction",
			`{"pattern":"*.go","path":"/src"}`,
			"*.go",
		},
		{
			"fallback to sorted keys",
			`{"alpha":1,"beta":2}`,
			"alpha, beta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractParam(json.RawMessage(tt.input))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractParamTruncation(t *testing.T) {
	longPath := "/workspace/repo/src/very/deeply/nested/directory/structure/that/goes/on/and/on/file.go"
	input := `{"file_path":"` + longPath + `"}`
	got := extractParam(json.RawMessage(input))

	assert.LessOrEqual(t, len([]rune(got)), 61, "expected truncation at 60 chars + ellipsis") // 60 + ellipsis
	assert.True(t, strings.HasSuffix(got, "…"), "expected ellipsis at end, got %q", got)
}

func TestFormatTaskFallbackSubagentType(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)

	evt := &Event{
		Type: "assistant",
		Message: &Message{
			Role: "assistant",
			Content: []ContentBlock{
				{
					Type:  "tool_use",
					Name:  "Task",
					Input: json.RawMessage(`{"description":"do stuff","prompt":"..."}`),
				},
			},
		},
	}

	require.NoError(t, f.Format(evt))
	assert.Contains(t, buf.String(), "▶ agent")
}
