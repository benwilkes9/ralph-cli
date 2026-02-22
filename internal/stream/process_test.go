package stream

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessFullIteration(t *testing.T) {
	f := openFixture(t, "testdata/full_iteration.jsonl")

	var buf bytes.Buffer
	stats, err := Process(f, &buf)
	require.NoError(t, err)

	assert.Greater(t, stats.PeakContext, 0)
	assert.Greater(t, stats.Cost, 0.0)
	assert.Greater(t, stats.ToolCalls, 0)
	assert.NotEmpty(t, buf.String())
}

func TestProcessWithSubagents(t *testing.T) {
	f := openFixture(t, "testdata/with_subagents.jsonl")

	var buf bytes.Buffer
	stats, err := Process(f, &buf)
	require.NoError(t, err)

	assert.Greater(t, stats.SubagentTokens, 0)
	output := buf.String()
	assert.Contains(t, output, "▶")
	assert.Contains(t, output, "✓")
}

func TestProcessEmpty(t *testing.T) {
	r := strings.NewReader("")
	var buf bytes.Buffer

	stats, err := Process(r, &buf)
	require.NoError(t, err)

	assert.Equal(t, 0, stats.PeakContext)
	assert.Equal(t, 0.0, stats.Cost)
	assert.Equal(t, 0, stats.SubagentTokens)
	assert.Equal(t, 0, stats.ToolCalls)
	assert.Equal(t, 0, buf.Len())
}

func TestProcessMalformed(t *testing.T) {
	f := openFixture(t, "testdata/malformed.jsonl")

	var buf bytes.Buffer
	stats, err := Process(f, &buf)
	require.NoError(t, err)

	assert.Greater(t, stats.PeakContext, 0, "expected non-zero peak context from valid assistant event in malformed fixture")
}

func TestProcessStatsAccumulation(t *testing.T) {
	f := openFixture(t, "testdata/full_iteration.jsonl")

	var buf bytes.Buffer
	stats, err := Process(f, &buf)
	require.NoError(t, err)

	cum := &CumulativeStats{}
	cum.Update(stats)
	assert.Equal(t, 1, cum.Iterations)
	assert.Equal(t, stats.PeakContext, cum.PeakContext)
	assert.Equal(t, stats.Cost, cum.TotalCost)
}
