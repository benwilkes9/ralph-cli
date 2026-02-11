package status

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/benmyles/ralph-cli/internal/state"
)

const samplePlan = `# Implementation Plan

## Overview
Build the thing.

### Task 1 -- Dependencies and project config
- [x] Initialize go module
- [x] Set up linting

### Task 2 -- Database layer
- [x] Create schema
- [x] Write migrations

### Task 3 -- Delete todo endpoint
- [ ] Add DELETE route
- [ ] Write handler

### Task 4 -- List filtering and search
- [ ] Add query params
`

func TestParsePlan(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "IMPLEMENTATION_PLAN.md")
	if err := os.WriteFile(path, []byte(samplePlan), 0o600); err != nil {
		t.Fatal(err)
	}

	tasks, err := ParsePlan(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(tasks) != 4 {
		t.Fatalf("expected 4 tasks, got %d", len(tasks))
	}

	want := []struct {
		title string
		done  bool
	}{
		{"Dependencies and project config", true},
		{"Database layer", true},
		{"Delete todo endpoint", false},
		{"List filtering and search", false},
	}
	for i, w := range want {
		if tasks[i].Title != w.title {
			t.Errorf("task %d: title=%q, want %q", i, tasks[i].Title, w.title)
		}
		if tasks[i].Done != w.done {
			t.Errorf("task %d: done=%v, want %v", i, tasks[i].Done, w.done)
		}
	}
}

func TestParsePlanMissingFile(t *testing.T) {
	tasks, err := ParsePlan("/nonexistent/path/plan.md")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if tasks != nil {
		t.Fatalf("expected nil tasks, got %v", tasks)
	}
}

func TestParseLogs(t *testing.T) {
	dir := t.TempDir()

	// Write two fixture JSONL files with result events.
	log1 := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}
{"type":"result","total_cost_usd":1.2345}
`
	log2 := `{"type":"result","total_cost_usd":0.5678}
`
	if err := os.WriteFile(filepath.Join(dir, "20260210-140000.jsonl"), []byte(log1), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "20260211-143000.jsonl"), []byte(log2), 0o600); err != nil {
		t.Fatal(err)
	}
	// Non-JSONL file should be ignored.
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignore me"), 0o600); err != nil {
		t.Fatal(err)
	}

	runs, err := ParseLogs(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}

	if runs[0].Time != time.Date(2026, 2, 10, 14, 0, 0, 0, time.UTC) {
		t.Errorf("run 0 time: got %v", runs[0].Time)
	}
	if runs[0].Cost != 1.2345 {
		t.Errorf("run 0 cost: got %f, want 1.2345", runs[0].Cost)
	}
	if runs[1].Cost != 0.5678 {
		t.Errorf("run 1 cost: got %f, want 0.5678", runs[1].Cost)
	}
}

func TestParseLogsMissingDir(t *testing.T) {
	runs, err := ParseLogs("/nonexistent/logs")
	if err != nil {
		t.Fatalf("expected no error for missing dir, got: %v", err)
	}
	if runs != nil {
		t.Fatalf("expected nil runs, got %v", runs)
	}
}

func TestRender(t *testing.T) {
	tasks := []Task{
		{Title: "Dependencies and project config", Done: true},
		{Title: "Database layer", Done: true},
		{Title: "Delete todo endpoint", Done: false},
		{Title: "List filtering and search", Done: false},
	}
	runs := []RunInfo{
		{Time: time.Date(2026, 2, 10, 14, 0, 0, 0, time.UTC), Cost: 1.2345},
		{Time: time.Date(2026, 2, 11, 14, 30, 0, 0, time.UTC), Cost: 2.2222},
	}
	lastRun := &state.RunRecord{
		Mode:       "build",
		StartedAt:  time.Date(2026, 2, 11, 14, 30, 0, 0, time.UTC),
		Iterations: 5,
	}

	var buf bytes.Buffer
	Render(&buf, "my-api", "feature/auth", tasks, runs, lastRun)
	out := buf.String()

	expects := []string{
		"Project: my-api",
		"Branch:  feature/auth",
		"Tasks:  2/4 complete (50%)",
		"Dependencies and project config",
		"Database layer",
		"Delete todo endpoint",
		"List filtering and search",
		"Last run:   2026-02-11 14:30 (build, 5 iterations)",
		"Total cost: $3.4567 across 2 iterations",
	}
	for _, s := range expects {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q\n\ngot:\n%s", s, out)
		}
	}
}

func TestRenderEmpty(t *testing.T) {
	var buf bytes.Buffer
	Render(&buf, "my-api", "main", nil, nil, nil)
	out := buf.String()

	if !strings.Contains(out, "Project: my-api") {
		t.Errorf("output missing project line: %s", out)
	}
	if strings.Contains(out, "Tasks:") {
		t.Errorf("should not show tasks section when no tasks: %s", out)
	}
	if strings.Contains(out, "Last run:") {
		t.Errorf("should not show runs section when no runs: %s", out)
	}
}
