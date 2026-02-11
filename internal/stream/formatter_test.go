package stream

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
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
		if got != tt.want {
			t.Errorf("FormatTokens(%d) = %q, want %q", tt.input, got, tt.want)
		}
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

	if err := f.Format(evt); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	if !strings.Contains(got, "Hello world") {
		t.Errorf("expected output to contain 'Hello world', got %q", got)
	}
	if !strings.Contains(got, Bold) {
		t.Error("expected bold ANSI code in text output")
	}
	if !strings.Contains(got, White) {
		t.Error("expected white ANSI code in text output")
	}
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

	if err := f.Format(evt); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	if !strings.Contains(got, "· Read") {
		t.Errorf("expected '· Read' in output, got %q", got)
	}
	if !strings.Contains(got, "/workspace/repo/main.go") {
		t.Errorf("expected file_path in output, got %q", got)
	}
	if !strings.Contains(got, Dim) {
		t.Error("expected dim ANSI code in tool output")
	}
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

	if err := f.Format(evt); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	if !strings.Contains(got, "▶ Bash") {
		t.Errorf("expected '▶ Bash' in output, got %q", got)
	}
	if !strings.Contains(got, `"Run all tests"`) {
		t.Errorf("expected quoted description in output, got %q", got)
	}
	if !strings.Contains(got, BoldCyan) {
		t.Error("expected BoldCyan ANSI code in Task output")
	}
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

	if err := f.Format(evt); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	if !strings.Contains(got, "model=opus") {
		t.Errorf("expected 'model=opus' in output, got %q", got)
	}
	if !strings.Contains(got, "max_turns=5") {
		t.Errorf("expected 'max_turns=5' in output, got %q", got)
	}
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

	if err := f.Format(evt); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	if !strings.Contains(got, "✓") {
		t.Error("expected green checkmark in completed subagent output")
	}
	if !strings.Contains(got, Green) {
		t.Error("expected green ANSI code")
	}
	if !strings.Contains(got, "19s") {
		t.Errorf("expected '19s' duration, got %q", got)
	}
	if !strings.Contains(got, "3 tool calls") {
		t.Errorf("expected '3 tool calls', got %q", got)
	}
	if !strings.Contains(got, "8.7k tokens") {
		t.Errorf("expected '8.7k tokens', got %q", got)
	}
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

	if err := f.Format(evt); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	if !strings.Contains(got, "✗") {
		t.Error("expected red X in failed subagent output")
	}
	if !strings.Contains(got, "error") {
		t.Error("expected status text in failed subagent output")
	}
	if !strings.Contains(got, BoldRed) {
		t.Error("expected BoldRed ANSI code")
	}
}

func TestFormatIgnoresSystemAndResult(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(&buf)

	for _, evtType := range []string{"system", "result"} {
		buf.Reset()
		evt := &Event{Type: evtType}
		if err := f.Format(evt); err != nil {
			t.Fatal(err)
		}
		if buf.Len() != 0 {
			t.Errorf("expected no output for %q event, got %q", evtType, buf.String())
		}
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

	if err := f.Format(evt); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output for regular user event, got %q", buf.String())
	}
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
			if got != tt.want {
				t.Errorf("extractParam(%s) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractParamTruncation(t *testing.T) {
	longPath := "/workspace/repo/src/very/deeply/nested/directory/structure/that/goes/on/and/on/file.go"
	input := `{"file_path":"` + longPath + `"}`
	got := extractParam(json.RawMessage(input))

	if len([]rune(got)) > 61 { // 60 + ellipsis
		t.Errorf("expected truncation at 60 chars + ellipsis, got length %d: %q", len([]rune(got)), got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected ellipsis at end, got %q", got)
	}
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

	if err := f.Format(evt); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	if !strings.Contains(got, "▶ agent") {
		t.Errorf("expected fallback '▶ agent' when no subagent_type, got %q", got)
	}
}
