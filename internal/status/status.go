package status

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/benwilkes9/ralph-cli/internal/state"
	"github.com/benwilkes9/ralph-cli/internal/stream"
)

// Task represents a single task parsed from the implementation plan.
type Task struct {
	Title string
	Done  bool
}

// RunInfo holds metadata from a single log file.
type RunInfo struct {
	Time time.Time
	Cost float64
}

var taskHeadingRe = regexp.MustCompile(`^###\s+Task\s+\d+\s*[-–—]+\s*(.+)`)

// ParsePlan reads an IMPLEMENTATION_PLAN.md and extracts tasks from it.
// Returns nil, nil if the file does not exist (plan not yet generated).
func ParsePlan(path string) ([]Task, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening plan: %w", err)
	}
	defer f.Close() //nolint:errcheck // read-only

	var tasks []Task
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		if m := taskHeadingRe.FindStringSubmatch(line); m != nil {
			tasks = append(tasks, Task{Title: strings.TrimSpace(m[1])})
			continue
		}

		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]") {
			if len(tasks) > 0 {
				tasks[len(tasks)-1].Done = true
			}
		} else if strings.HasPrefix(trimmed, "- [ ]") {
			if len(tasks) > 0 {
				tasks[len(tasks)-1].Done = false
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning plan: %w", err)
	}
	return tasks, nil
}

// ParseLogs scans a logs directory for .jsonl files and extracts run info.
// Returns nil, nil if the directory does not exist.
func ParseLogs(logsDir string) ([]RunInfo, error) {
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading logs dir: %w", err)
	}

	var runs []RunInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".jsonl")
		t, err := time.Parse("20060102-150405", name)
		if err != nil {
			continue // skip files that don't match the timestamp format
		}

		cost, err := extractCost(filepath.Join(logsDir, entry.Name()))
		if err != nil {
			continue
		}

		runs = append(runs, RunInfo{Time: t, Cost: cost})
	}

	sort.Slice(runs, func(i, j int) bool {
		return runs[i].Time.Before(runs[j].Time)
	})
	return runs, nil
}

func extractCost(path string) (float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("opening log %s: %w", filepath.Base(path), err)
	}
	defer f.Close() //nolint:errcheck // read-only

	p := stream.NewParser(f)
	for {
		evt, err := p.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("parsing log %s: %w", filepath.Base(path), err)
		}
		if evt.Type == "result" && evt.TotalCostUSD > 0 {
			return evt.TotalCostUSD, nil
		}
	}
	return 0, nil
}

// Render writes a formatted status summary to w.
//
//nolint:errcheck // display output, best-effort writes
func Render(w io.Writer, project, branch string, tasks []Task, runs []RunInfo, lastRun *state.RunRecord) {
	fmt.Fprintf(w, "Project: %s\n", project)
	fmt.Fprintf(w, "Branch:  %s\n", branch)

	if len(tasks) > 0 {
		done := 0
		for _, t := range tasks {
			if t.Done {
				done++
			}
		}
		pct := 0
		if len(tasks) > 0 {
			pct = done * 100 / len(tasks)
		}
		fmt.Fprintf(w, "\nTasks:  %d/%d complete (%d%%)\n", done, len(tasks), pct)
		for _, t := range tasks {
			if t.Done {
				fmt.Fprintf(w, "  %s\u2713%s %s\n", stream.Green, stream.Reset, t.Title)
			} else {
				fmt.Fprintf(w, "  %s\u00b7%s %s\n", stream.Dim, stream.Reset, t.Title)
			}
		}
	}

	if lastRun != nil {
		fmt.Fprintf(w, "\nLast run:   %s (%s, %d iterations)\n",
			lastRun.StartedAt.Format("2006-01-02 15:04"), lastRun.Mode, lastRun.Iterations)
	} else if len(runs) > 0 {
		last := runs[len(runs)-1]
		fmt.Fprintf(w, "\nLast run:   %s\n", last.Time.Format("2006-01-02 15:04"))
	}

	if len(runs) > 0 {
		var totalCost float64
		for _, r := range runs {
			totalCost += r.Cost
		}
		fmt.Fprintf(w, "Total cost: $%.4f across %d iterations\n", totalCost, len(runs))
	}
}
