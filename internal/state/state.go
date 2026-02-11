package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// DefaultPath is the default location for state.json relative to repo root.
const DefaultPath = ".ralph/state.json"

// RunStatus describes how a loop run ended.
type RunStatus string

// Run statuses.
const (
	StatusCompleted     RunStatus = "completed"
	StatusStaleAbort    RunStatus = "stale_abort"
	StatusCancelled     RunStatus = "cancelled"
	StatusMaxIterations RunStatus = "max_iterations"
)

// RunRecord captures metadata from a single loop run.
type RunRecord struct {
	Mode           string    `json:"mode"`
	StartedAt      time.Time `json:"started_at"`
	FinishedAt     time.Time `json:"finished_at"`
	Iterations     int       `json:"iterations"`
	TotalCost      float64   `json:"total_cost"`
	PeakContext    int       `json:"peak_context"`
	SubagentTokens int       `json:"subagent_tokens"`
	Status         RunStatus `json:"status"`
	LogFiles       []string  `json:"log_files"`
}

// State holds all recorded loop runs.
type State struct {
	Runs []RunRecord `json:"runs"`
}

// Load reads state from disk. Returns an empty State if the file does not exist.
func Load(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &State{}, nil
		}
		return nil, fmt.Errorf("reading state: %w", err)
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing state: %w", err)
	}
	return &s, nil
}

// Save writes state to disk atomically.
func Save(path string, s *State) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing state: %w", err)
	}
	return nil
}

// LastRun returns the most recent run record, or nil if there are no runs.
func (s *State) LastRun() *RunRecord {
	if len(s.Runs) == 0 {
		return nil
	}
	return &s.Runs[len(s.Runs)-1]
}
