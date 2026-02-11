package stream

import (
	"errors"
	"io"
	"os"
	"testing"
)

func openFixture(t *testing.T, name string) *os.File {
	t.Helper()
	f, err := os.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	})
	return f
}

func TestParseFullIteration(t *testing.T) {
	f := openFixture(t, "testdata/full_iteration.jsonl")

	p := NewParser(f)
	var events []*Event
	for {
		evt, err := p.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, evt)
	}

	if len(events) != 23 {
		t.Fatalf("expected 23 events, got %d", len(events))
	}
}

func TestParseEventTypes(t *testing.T) {
	f := openFixture(t, "testdata/full_iteration.jsonl")

	p := NewParser(f)
	typeCounts := map[string]int{}
	for {
		evt, err := p.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		typeCounts[evt.Type]++
	}

	if typeCounts["system"] < 1 {
		t.Error("expected at least 1 system event")
	}
	if typeCounts[eventAssistant] < 1 {
		t.Error("expected at least 1 assistant event")
	}
	if typeCounts[eventUser] < 1 {
		t.Error("expected at least 1 user event")
	}
	if typeCounts[eventResult] != 1 {
		t.Errorf("expected exactly 1 result event, got %d", typeCounts[eventResult])
	}
}

func TestParseAssistantMessage(t *testing.T) {
	f := openFixture(t, "testdata/full_iteration.jsonl")

	p := NewParser(f)
	for {
		evt, err := p.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if evt.Type != eventAssistant {
			continue
		}

		if evt.Message == nil {
			t.Fatal("assistant event has nil Message")
		}
		if evt.Message.Role != eventAssistant {
			t.Errorf("expected role=assistant, got %q", evt.Message.Role)
		}
		if evt.Message.Model == "" {
			t.Error("expected non-empty model")
		}
		if len(evt.Message.Content) == 0 {
			t.Error("expected at least one content block")
		}
		return // only check first assistant event
	}
}

func TestParseUsageFields(t *testing.T) {
	f := openFixture(t, "testdata/full_iteration.jsonl")

	p := NewParser(f)
	var foundUsage bool
	for {
		evt, err := p.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if evt.Type != eventAssistant || evt.Message == nil || evt.Message.Usage == nil {
			continue
		}

		u := evt.Message.Usage
		total := u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
		if total == 0 {
			continue
		}
		foundUsage = true

		if u.CacheCreationInputTokens == 0 && u.CacheReadInputTokens == 0 && u.InputTokens == 0 {
			t.Error("expected at least one positive token field")
		}
		break
	}
	if !foundUsage {
		t.Error("no assistant event with usage found")
	}
}

func TestParseResultEvent(t *testing.T) {
	f := openFixture(t, "testdata/full_iteration.jsonl")

	p := NewParser(f)
	for {
		evt, err := p.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if evt.Type != eventResult {
			continue
		}

		if evt.TotalCostUSD <= 0 {
			t.Errorf("expected positive total_cost_usd, got %f", evt.TotalCostUSD)
		}
		return
	}
	t.Error("no result event found")
}

func TestParseToolUseResult(t *testing.T) {
	f := openFixture(t, "testdata/full_iteration.jsonl")

	p := NewParser(f)
	for {
		evt, err := p.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if evt.Type != eventUser || evt.ToolUseResult == nil {
			continue
		}
		return
	}
	t.Error("no user event with tool_use_result found")
}

func TestParseSubagentResult(t *testing.T) {
	f := openFixture(t, "testdata/with_subagents.jsonl")

	p := NewParser(f)
	var foundSubagent bool
	for {
		evt, err := p.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if evt.Type != eventUser || evt.ToolUseResult == nil || evt.ToolUseResult.TotalTokens == 0 {
			continue
		}
		foundSubagent = true
		tr := evt.ToolUseResult
		if tr.TotalTokens <= 0 {
			t.Error("expected positive totalTokens")
		}
		if tr.TotalDurationMs <= 0 {
			t.Error("expected positive totalDurationMs")
		}
	}
	if !foundSubagent {
		t.Error("no subagent result found in with_subagents.jsonl")
	}
}

func TestParseMalformedLines(t *testing.T) {
	f := openFixture(t, "testdata/malformed.jsonl")

	p := NewParser(f)
	var events []*Event
	for {
		evt, err := p.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, evt)
	}

	// malformed.jsonl has: bad json, valid line, empty line, truncated json, valid user event
	if len(events) != 2 {
		t.Errorf("expected 2 valid events from malformed.jsonl, got %d", len(events))
	}
}

func TestParseToolUseContent(t *testing.T) {
	f := openFixture(t, "testdata/full_iteration.jsonl")

	p := NewParser(f)
	for {
		evt, err := p.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if evt.Type != eventAssistant || evt.Message == nil {
			continue
		}
		for _, block := range evt.Message.Content {
			if block.Type == contentToolUse {
				if block.Name == "" {
					t.Error("tool_use block has empty name")
				}
				if len(block.Input) == 0 {
					t.Error("tool_use block has empty input")
				}
				return
			}
		}
	}
}
