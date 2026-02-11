package state

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoadMissingFile(t *testing.T) {
	s, err := Load("/nonexistent/state.json")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(s.Runs) != 0 {
		t.Fatalf("expected empty runs, got %d", len(s.Runs))
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	original := &State{
		Runs: []RunRecord{
			{
				Mode:           "build",
				StartedAt:      time.Date(2026, 2, 11, 14, 0, 0, 0, time.UTC),
				FinishedAt:     time.Date(2026, 2, 11, 14, 30, 0, 0, time.UTC),
				Iterations:     5,
				TotalCost:      1.2345,
				PeakContext:    150000,
				SubagentTokens: 5000,
				Status:         StatusCompleted,
				LogFiles:       []string{"logs/20260211-140000.jsonl"},
			},
		},
	}

	if err := Save(path, original); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(loaded.Runs))
	}

	r := loaded.Runs[0]
	if r.Mode != "build" {
		t.Errorf("mode: got %q, want %q", r.Mode, "build")
	}
	if r.Iterations != 5 {
		t.Errorf("iterations: got %d, want 5", r.Iterations)
	}
	if r.Status != StatusCompleted {
		t.Errorf("status: got %q, want %q", r.Status, StatusCompleted)
	}
	if r.TotalCost != 1.2345 {
		t.Errorf("cost: got %f, want 1.2345", r.TotalCost)
	}
}

func TestLastRun(t *testing.T) {
	s := &State{}
	if s.LastRun() != nil {
		t.Fatal("expected nil LastRun for empty state")
	}

	s.Runs = []RunRecord{
		{Mode: "plan", Iterations: 3},
		{Mode: "build", Iterations: 5},
	}
	last := s.LastRun()
	if last == nil {
		t.Fatal("expected non-nil LastRun")
	}
	if last.Mode != "build" {
		t.Errorf("last run mode: got %q, want %q", last.Mode, "build")
	}
	if last.Iterations != 5 {
		t.Errorf("last run iterations: got %d, want 5", last.Iterations)
	}
}
