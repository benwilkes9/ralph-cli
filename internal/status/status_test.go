package status

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/benwilkes9/ralph-cli/internal/state"
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
	require.NoError(t, os.WriteFile(path, []byte(samplePlan), 0o600))

	tasks, err := ParsePlan(path)
	require.NoError(t, err)
	require.Len(t, tasks, 4)

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
		assert.Equal(t, w.title, tasks[i].Title, "task %d title", i)
		assert.Equal(t, w.done, tasks[i].Done, "task %d done", i)
	}
}

const samplePlanNewFormat = `# Implementation Plan

## Overview
Build the thing.

### Task 1.1: Dependencies and project config
- [x] **Status:** Complete
- **Description:** Initialize go module and set up linting.

### Task 1.2: Database layer
- [x] **Status:** Complete
- **Description:** Create schema and write migrations.

### Task 2.1: Delete todo endpoint
- [ ] **Status:** Incomplete
- **Description:** Add DELETE route and write handler.

### Task 2.2: List filtering and search
- [ ] **Status:** Incomplete
- **Description:** Add query params.
`

func TestParsePlan_NewFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "IMPLEMENTATION_PLAN.md")
	require.NoError(t, os.WriteFile(path, []byte(samplePlanNewFormat), 0o600))

	tasks, err := ParsePlan(path)
	require.NoError(t, err)
	require.Len(t, tasks, 4)

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
		assert.Equal(t, w.title, tasks[i].Title, "task %d title", i)
		assert.Equal(t, w.done, tasks[i].Done, "task %d done", i)
	}
}

func TestParsePlanMissingFile(t *testing.T) {
	tasks, err := ParsePlan("/nonexistent/path/plan.md")
	require.NoError(t, err)
	assert.Nil(t, tasks)
}

func TestParseLogs(t *testing.T) {
	dir := t.TempDir()

	log1 := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}
{"type":"result","total_cost_usd":1.2345}
`
	log2 := `{"type":"result","total_cost_usd":0.5678}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "20260210-140000.jsonl"), []byte(log1), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "20260211-143000.jsonl"), []byte(log2), 0o600))
	// Non-JSONL file should be ignored.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignore me"), 0o600))

	runs, err := ParseLogs(dir)
	require.NoError(t, err)
	require.Len(t, runs, 2)

	assert.Equal(t, time.Date(2026, 2, 10, 14, 0, 0, 0, time.UTC), runs[0].Time)
	assert.Equal(t, 1.2345, runs[0].Cost)
	assert.Equal(t, 0.5678, runs[1].Cost)
}

func TestParseLogsMissingDir(t *testing.T) {
	runs, err := ParseLogs("/nonexistent/logs")
	require.NoError(t, err)
	assert.Nil(t, runs)
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

	for _, want := range []string{
		"Project: my-api",
		"Branch:  feature/auth",
		"Tasks:  2/4 complete (50%)",
		"Dependencies and project config",
		"Database layer",
		"Delete todo endpoint",
		"List filtering and search",
		"Last run:   2026-02-11 14:30 (build, 5 iterations)",
		"Total cost: $3.4567 across 2 iterations",
	} {
		assert.Contains(t, out, want)
	}
}

func TestRenderEmpty(t *testing.T) {
	var buf bytes.Buffer
	Render(&buf, "my-api", "main", nil, nil, nil)
	out := buf.String()

	assert.Contains(t, out, "Project: my-api")
	assert.NotContains(t, out, "Tasks:")
	assert.NotContains(t, out, "Last run:")
}
