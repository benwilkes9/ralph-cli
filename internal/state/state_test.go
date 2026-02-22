package state

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMissingFile(t *testing.T) {
	s, err := Load("/nonexistent/state.json")
	require.NoError(t, err)
	assert.Empty(t, s.Runs)
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

	require.NoError(t, Save(path, original))

	loaded, err := Load(path)
	require.NoError(t, err)
	require.Len(t, loaded.Runs, 1)

	r := loaded.Runs[0]
	assert.Equal(t, "build", r.Mode)
	assert.Equal(t, 5, r.Iterations)
	assert.Equal(t, StatusCompleted, r.Status)
	assert.Equal(t, 1.2345, r.TotalCost)
}

func TestLastRun(t *testing.T) {
	s := &State{}
	assert.Nil(t, s.LastRun())

	s.Runs = []RunRecord{
		{Mode: "plan", Iterations: 3},
		{Mode: "build", Iterations: 5},
	}
	last := s.LastRun()
	require.NotNil(t, last)
	assert.Equal(t, "build", last.Mode)
	assert.Equal(t, 5, last.Iterations)
}
