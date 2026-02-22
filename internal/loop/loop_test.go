package loop

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/benwilkes9/ralph-cli/internal/stream"
)

func TestRenderHeader(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Mode:          ModeBuild,
		PromptFile:    ".ralph/prompts/build.md",
		Branch:        "feat/awesome",
		MaxIterations: 10,
	}
	RenderHeader(&buf, &opts)
	out := buf.String()

	for _, want := range []string{"━━━", "build", ".ralph/prompts/build.md", "feat/awesome", "10 iterations"} {
		assert.Contains(t, out, want)
	}
}

func TestRenderHeaderNoMax(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{
		Mode:          ModeBuild,
		PromptFile:    "build.md",
		Branch:        "main",
		MaxIterations: 0,
	}
	RenderHeader(&buf, &opts)
	assert.NotContains(t, buf.String(), "iterations")
}

func TestRenderHeaderPlanColor(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{Mode: ModePlan, PromptFile: "p.md", Branch: "main"}
	RenderHeader(&buf, &opts)
	assert.Contains(t, buf.String(), stream.BoldCyan)
}

func TestRenderHeaderBuildColor(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{Mode: ModeBuild, PromptFile: "b.md", Branch: "main"}
	RenderHeader(&buf, &opts)
	assert.Contains(t, buf.String(), stream.BoldGreen)
}

func TestRenderBanner(t *testing.T) {
	var buf bytes.Buffer
	RenderBanner(&buf, ModeBuild, 1)
	out := buf.String()

	for _, want := range []string{"BUILD", "#1", "╔", "╚"} {
		assert.Contains(t, out, want)
	}
}

func TestRenderBannerPlan(t *testing.T) {
	var buf bytes.Buffer
	RenderBanner(&buf, ModePlan, 3)
	out := buf.String()

	assert.Contains(t, out, "PLAN")
	assert.Contains(t, out, stream.BoldCyan)
}

func TestRenderIterationSummary(t *testing.T) {
	var buf bytes.Buffer
	stats := &stream.IterationStats{
		PeakContext: 85_200,
		Cost:        0.0234,
	}
	RenderIterationSummary(&buf, stats, "logs/20250101-120000.jsonl")
	out := buf.String()

	for _, want := range []string{"85.2k", "200.0k", "42%", "$0.0234", "logs/20250101-120000.jsonl"} {
		assert.Contains(t, out, want)
	}
}

func TestRenderIterationSummaryZeroCost(t *testing.T) {
	var buf bytes.Buffer
	stats := &stream.IterationStats{
		PeakContext: 10_000,
		Cost:        0,
	}
	RenderIterationSummary(&buf, stats, "logs/test.jsonl")
	assert.NotContains(t, buf.String(), "$")
}

func TestRenderStaleWarning(t *testing.T) {
	var buf bytes.Buffer
	RenderStaleWarning(&buf, 1, 2)
	out := buf.String()

	assert.Contains(t, out, "No new commits")
	assert.Contains(t, out, "1/2")
	assert.Contains(t, out, stream.BoldYellow)
}

func TestRenderStaleAbort(t *testing.T) {
	var buf bytes.Buffer
	RenderStaleAbort(&buf, 2)
	out := buf.String()

	assert.Contains(t, out, "Stale loop detected")
	assert.Contains(t, out, stream.BoldRed)
}

func TestRenderMaxIterations(t *testing.T) {
	var buf bytes.Buffer
	RenderMaxIterations(&buf, 5)
	out := buf.String()

	assert.Contains(t, out, "Reached max iterations: 5")
	assert.Contains(t, out, stream.BoldYellow)
}

func TestModeColor(t *testing.T) {
	assert.Equal(t, stream.BoldCyan, modeColor(ModePlan))
	assert.Equal(t, stream.BoldGreen, modeColor(ModeBuild))
}

func TestStaleDetectorIntegration(t *testing.T) {
	d := NewStaleDetector(2)

	// Seed: first call always returns not-stale
	abort, count := d.Check("aaa")
	assert.False(t, abort, "seed should not abort")
	assert.Equal(t, 0, count, "seed should have count 0")

	// HEAD changes: stale count resets
	abort, count = d.Check("bbb")
	assert.False(t, abort, "change should not abort")
	assert.Equal(t, 0, count, "change should reset count")

	// Same HEAD: stale count 1
	abort, count = d.Check("bbb")
	assert.False(t, abort, "should not abort at stale count 1")
	assert.Equal(t, 1, count)

	// Same HEAD again: stale count 2 → abort
	abort, count = d.Check("bbb")
	assert.True(t, abort, "should abort at stale count 2")
	assert.Equal(t, 2, count)
}

func TestStaleDetectorReset(t *testing.T) {
	d := NewStaleDetector(2)

	d.Check("aaa") // seed
	d.Check("aaa") // stale 1

	// HEAD changes → count resets
	abort, count := d.Check("bbb")
	assert.False(t, abort)
	assert.Equal(t, 0, count)

	// Stale again from 0
	abort, count = d.Check("bbb")
	assert.False(t, abort, "should not abort at stale count 1 after reset")
	assert.Equal(t, 1, count)
}
