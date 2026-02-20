package loop

import (
	"bytes"
	"strings"
	"testing"

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
		if !strings.Contains(out, want) {
			t.Errorf("RenderHeader missing %q in:\n%s", want, out)
		}
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
	out := buf.String()

	if strings.Contains(out, "iterations") {
		t.Errorf("RenderHeader should not show iterations line when MaxIterations=0, got:\n%s", out)
	}
}

func TestRenderHeaderPlanColor(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{Mode: ModePlan, PromptFile: "p.md", Branch: "main"}
	RenderHeader(&buf, &opts)

	if !strings.Contains(buf.String(), stream.BoldCyan) {
		t.Error("RenderHeader for plan mode should use BoldCyan")
	}
}

func TestRenderHeaderBuildColor(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{Mode: ModeBuild, PromptFile: "b.md", Branch: "main"}
	RenderHeader(&buf, &opts)

	if !strings.Contains(buf.String(), stream.BoldGreen) {
		t.Error("RenderHeader for build mode should use BoldGreen")
	}
}

func TestRenderBanner(t *testing.T) {
	var buf bytes.Buffer
	RenderBanner(&buf, ModeBuild, 1)
	out := buf.String()

	for _, want := range []string{"BUILD", "#1", "╔", "╚"} {
		if !strings.Contains(out, want) {
			t.Errorf("RenderBanner missing %q in:\n%s", want, out)
		}
	}
}

func TestRenderBannerPlan(t *testing.T) {
	var buf bytes.Buffer
	RenderBanner(&buf, ModePlan, 3)
	out := buf.String()

	if !strings.Contains(out, "PLAN") {
		t.Error("RenderBanner for plan mode should contain PLAN")
	}
	if !strings.Contains(out, stream.BoldCyan) {
		t.Error("RenderBanner for plan mode should use BoldCyan")
	}
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
		if !strings.Contains(out, want) {
			t.Errorf("RenderIterationSummary missing %q in:\n%s", want, out)
		}
	}
}

func TestRenderIterationSummaryZeroCost(t *testing.T) {
	var buf bytes.Buffer
	stats := &stream.IterationStats{
		PeakContext: 10_000,
		Cost:        0,
	}
	RenderIterationSummary(&buf, stats, "logs/test.jsonl")
	out := buf.String()

	if strings.Contains(out, "$") {
		t.Errorf("RenderIterationSummary should not show $ when cost is 0, got:\n%s", out)
	}
}

func TestRenderStaleWarning(t *testing.T) {
	var buf bytes.Buffer
	RenderStaleWarning(&buf, 1, 2)
	out := buf.String()

	if !strings.Contains(out, "No new commits") {
		t.Error("RenderStaleWarning should contain 'No new commits'")
	}
	if !strings.Contains(out, "1/2") {
		t.Error("RenderStaleWarning should contain count/max")
	}
	if !strings.Contains(out, stream.BoldYellow) {
		t.Error("RenderStaleWarning should use BoldYellow")
	}
}

func TestRenderStaleAbort(t *testing.T) {
	var buf bytes.Buffer
	RenderStaleAbort(&buf, 2)
	out := buf.String()

	if !strings.Contains(out, "Stale loop detected") {
		t.Error("RenderStaleAbort should contain 'Stale loop detected'")
	}
	if !strings.Contains(out, stream.BoldRed) {
		t.Error("RenderStaleAbort should use BoldRed")
	}
}

func TestRenderMaxIterations(t *testing.T) {
	var buf bytes.Buffer
	RenderMaxIterations(&buf, 5)
	out := buf.String()

	if !strings.Contains(out, "Reached max iterations: 5") {
		t.Errorf("RenderMaxIterations should contain message, got:\n%s", out)
	}
	if !strings.Contains(out, stream.BoldYellow) {
		t.Error("RenderMaxIterations should use BoldYellow")
	}
}

func TestModeColor(t *testing.T) {
	if got := modeColor(ModePlan); got != stream.BoldCyan {
		t.Errorf("modeColor(plan) = %q, want BoldCyan", got)
	}
	if got := modeColor(ModeBuild); got != stream.BoldGreen {
		t.Errorf("modeColor(build) = %q, want BoldGreen", got)
	}
}

func TestStaleDetectorIntegration(t *testing.T) {
	d := NewStaleDetector(2)

	// Seed
	abort, count := d.Check("aaa")
	if abort || count != 0 {
		t.Fatal("seed should not be stale")
	}

	// Change
	abort, count = d.Check("bbb")
	if abort || count != 0 {
		t.Fatal("change should reset stale")
	}

	// Stale 1
	abort, count = d.Check("bbb")
	if abort {
		t.Fatal("should not abort at stale count 1")
	}
	if count != 1 {
		t.Fatalf("expected stale count 1, got %d", count)
	}

	// Stale 2 → abort
	abort, count = d.Check("bbb")
	if !abort {
		t.Fatal("should abort at stale count 2")
	}
	if count != 2 {
		t.Fatalf("expected stale count 2, got %d", count)
	}
}

func TestStaleDetectorReset(t *testing.T) {
	d := NewStaleDetector(2)

	d.Check("aaa") // seed
	d.Check("aaa") // stale 1

	// HEAD changes → count resets
	abort, count := d.Check("bbb")
	if abort || count != 0 {
		t.Fatalf("expected reset after HEAD change, got abort=%v count=%d", abort, count)
	}

	// Stale again from 0
	abort, count = d.Check("bbb")
	if abort {
		t.Fatal("should not abort at stale count 1 after reset")
	}
	if count != 1 {
		t.Fatalf("expected stale count 1, got %d", count)
	}
}
