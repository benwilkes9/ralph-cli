package stream

import (
	"bytes"
	"strings"
	"testing"
)

func TestProcessFullIteration(t *testing.T) {
	f := openFixture(t, "testdata/full_iteration.jsonl")

	var buf bytes.Buffer
	stats, err := Process(f, &buf)
	if err != nil {
		t.Fatal(err)
	}

	if stats.PeakContext <= 0 {
		t.Errorf("expected positive peak context, got %d", stats.PeakContext)
	}
	if stats.Cost <= 0 {
		t.Errorf("expected positive cost, got %f", stats.Cost)
	}
	if stats.ToolCalls <= 0 {
		t.Errorf("expected positive tool calls, got %d", stats.ToolCalls)
	}

	output := buf.String()
	if output == "" {
		t.Error("expected non-empty formatted output")
	}
}

func TestProcessWithSubagents(t *testing.T) {
	f := openFixture(t, "testdata/with_subagents.jsonl")

	var buf bytes.Buffer
	stats, err := Process(f, &buf)
	if err != nil {
		t.Fatal(err)
	}

	if stats.SubagentTokens <= 0 {
		t.Errorf("expected positive subagent tokens, got %d", stats.SubagentTokens)
	}

	output := buf.String()
	if !strings.Contains(output, "▶") {
		t.Error("expected Task tool formatting with ▶ in output")
	}
	if !strings.Contains(output, "✓") {
		t.Error("expected subagent completion checkmark in output")
	}
}

func TestProcessEmpty(t *testing.T) {
	r := strings.NewReader("")
	var buf bytes.Buffer

	stats, err := Process(r, &buf)
	if err != nil {
		t.Fatal(err)
	}

	if stats.PeakContext != 0 {
		t.Errorf("expected zero peak context, got %d", stats.PeakContext)
	}
	if stats.Cost != 0 {
		t.Errorf("expected zero cost, got %f", stats.Cost)
	}
	if stats.SubagentTokens != 0 {
		t.Errorf("expected zero subagent tokens, got %d", stats.SubagentTokens)
	}
	if stats.ToolCalls != 0 {
		t.Errorf("expected zero tool calls, got %d", stats.ToolCalls)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output, got %q", buf.String())
	}
}

func TestProcessMalformed(t *testing.T) {
	f := openFixture(t, "testdata/malformed.jsonl")

	var buf bytes.Buffer
	stats, err := Process(f, &buf)
	if err != nil {
		t.Fatal(err)
	}

	if stats.PeakContext == 0 {
		t.Error("expected non-zero peak context from valid assistant event in malformed fixture")
	}
}

func TestProcessStatsAccumulation(t *testing.T) {
	f := openFixture(t, "testdata/full_iteration.jsonl")

	var buf bytes.Buffer
	stats, err := Process(f, &buf)
	if err != nil {
		t.Fatal(err)
	}

	cum := &CumulativeStats{}
	cum.Update(stats)
	if cum.Iterations != 1 {
		t.Errorf("expected 1 iteration, got %d", cum.Iterations)
	}
	if cum.PeakContext != stats.PeakContext {
		t.Errorf("expected peak context %d, got %d", stats.PeakContext, cum.PeakContext)
	}
	if cum.TotalCost != stats.Cost {
		t.Errorf("expected cost %f, got %f", stats.Cost, cum.TotalCost)
	}
}
