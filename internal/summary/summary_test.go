package summary

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/benwilkes9/ralph-cli/internal/stream"
)

func printBox(stats *stream.CumulativeStats, wallTime time.Duration) string {
	var buf bytes.Buffer
	PrintBox(&buf, stats, wallTime)
	return buf.String()
}

func TestPrintBox_Header(t *testing.T) {
	out := printBox(&stream.CumulativeStats{}, 0)

	assert.Contains(t, out, "JOB SUMMARY")
	assert.Contains(t, out, "┌")
	assert.Contains(t, out, "┐")
	assert.Contains(t, out, "│")
	assert.Contains(t, out, "└")
	assert.Contains(t, out, "┘")
}

func TestPrintBox_Iterations(t *testing.T) {
	stats := &stream.CumulativeStats{Iterations: 7}
	out := printBox(stats, 0)

	assert.Contains(t, out, "7")
}

func TestPrintBox_WallTime(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0m 0s"},
		{time.Minute, "1m 0s"},
		{90 * time.Second, "1m 30s"},
		{62*time.Minute + 3*time.Second, "62m 3s"},
	}

	for _, tt := range tests {
		out := printBox(&stream.CumulativeStats{}, tt.d)
		assert.Contains(t, out, tt.want, "duration %v", tt.d)
	}
}

func TestPrintBox_PeakContextPercentage(t *testing.T) {
	// 100k tokens = 50% of 200k limit
	stats := &stream.CumulativeStats{PeakContext: 100_000}
	out := printBox(stats, 0)

	assert.Contains(t, out, "50%")
}

func TestPrintBox_TotalCost(t *testing.T) {
	stats := &stream.CumulativeStats{TotalCost: 1.2345}
	out := printBox(stats, 0)

	assert.Contains(t, out, "$1.2345")
}

func TestPrintBox_ZeroStats(t *testing.T) {
	// Must not panic on zero-value stats.
	assert.NotPanics(t, func() {
		printBox(&stream.CumulativeStats{}, 0)
	})
}
