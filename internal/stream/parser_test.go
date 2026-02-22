package stream

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openFixture(t *testing.T, name string) *os.File {
	t.Helper()
	f, err := os.Open(name)
	require.NoError(t, err)
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
		require.NoError(t, err)
		events = append(events, evt)
	}

	assert.Len(t, events, 23)
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
		require.NoError(t, err)
		typeCounts[evt.Type]++
	}

	assert.GreaterOrEqual(t, typeCounts["system"], 1)
	assert.GreaterOrEqual(t, typeCounts[eventAssistant], 1)
	assert.GreaterOrEqual(t, typeCounts[eventUser], 1)
	assert.Equal(t, 1, typeCounts[eventResult])
}

func TestParseAssistantMessage(t *testing.T) {
	f := openFixture(t, "testdata/full_iteration.jsonl")

	p := NewParser(f)
	for {
		evt, err := p.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		if evt.Type != eventAssistant {
			continue
		}

		require.NotNil(t, evt.Message)
		assert.Equal(t, eventAssistant, evt.Message.Role)
		assert.NotEmpty(t, evt.Message.Model)
		assert.NotEmpty(t, evt.Message.Content)
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
		require.NoError(t, err)
		if evt.Type != eventAssistant || evt.Message == nil || evt.Message.Usage == nil {
			continue
		}

		u := evt.Message.Usage
		total := u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
		if total == 0 {
			continue
		}
		foundUsage = true
		assert.True(t, u.CacheCreationInputTokens > 0 || u.CacheReadInputTokens > 0 || u.InputTokens > 0,
			"expected at least one positive token field")
		break
	}
	assert.True(t, foundUsage, "no assistant event with usage found")
}

func TestParseResultEvent(t *testing.T) {
	f := openFixture(t, "testdata/full_iteration.jsonl")

	p := NewParser(f)
	for {
		evt, err := p.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		if evt.Type != eventResult {
			continue
		}

		assert.Greater(t, evt.TotalCostUSD, 0.0)
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
		require.NoError(t, err)
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
		require.NoError(t, err)
		if evt.Type != eventUser || evt.ToolUseResult == nil || evt.ToolUseResult.TotalTokens == 0 {
			continue
		}
		foundSubagent = true
		tr := evt.ToolUseResult
		assert.Greater(t, tr.TotalTokens, 0)
		assert.Greater(t, tr.TotalDurationMs, 0)
	}
	assert.True(t, foundSubagent, "no subagent result found in with_subagents.jsonl")
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
		require.NoError(t, err)
		events = append(events, evt)
	}

	// malformed.jsonl has: bad json, valid line, empty line, truncated json, valid user event
	assert.Len(t, events, 2)
}

func TestParseToolUseContent(t *testing.T) {
	f := openFixture(t, "testdata/full_iteration.jsonl")

	p := NewParser(f)
	for {
		evt, err := p.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		if evt.Type != eventAssistant || evt.Message == nil {
			continue
		}
		for _, block := range evt.Message.Content {
			if block.Type == contentToolUse {
				assert.NotEmpty(t, block.Name, "tool_use block has empty name")
				assert.NotEmpty(t, block.Input, "tool_use block has empty input")
				return
			}
		}
	}
}
