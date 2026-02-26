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
	"github.com/benwilkes9/ralph-cli/internal/ui"
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

var taskHeadingRe = regexp.MustCompile(`^###\s+Task\s+[\d.]+\s*[-–—:]+\s*(.+)`)

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

// Render writes a formatted status summary to w with themed styling.
//
//nolint:errcheck // display output, best-effort writes
func Render(w io.Writer, project, branch string, tasks []Task, runs []RunInfo, lastRun *state.RunRecord, theme *ui.Theme) {
	fmt.Fprintln(w, theme.Banner())
	fmt.Fprintln(w)

	// Project/branch header box
	header := fmt.Sprintf("%s  ·  %s", project, branch)
	fmt.Fprintln(w, theme.StatusBox.Render(header))

	if len(tasks) > 0 {
		done := 0
		for _, t := range tasks {
			if t.Done {
				done++
			}
		}
		pct := done * 100 / len(tasks)
		fmt.Fprintf(w, "\n Tasks  %d/%d complete (%d%%)\n", done, len(tasks), pct)

		// Progress bar
		barWidth := 40
		filled := barWidth * pct / 100
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
		fmt.Fprintf(w, " %s  %d%%\n\n", theme.Success.Render(bar), pct)

		for _, t := range tasks {
			if t.Done {
				fmt.Fprintf(w, "  %s %s\n", theme.Success.Render("✓"), t.Title)
			} else {
				fmt.Fprintf(w, "  %s %s\n", theme.Muted.Render("·"), t.Title)
			}
		}
	}

	// Run info box
	var infoLines []string
	if lastRun != nil {
		infoLines = append(infoLines,
			fmt.Sprintf("Last run   %s (%s, %d iterations)",
				lastRun.StartedAt.Format("2006-01-02 15:04"), lastRun.Mode, lastRun.Iterations))
	} else if len(runs) > 0 {
		last := runs[len(runs)-1]
		infoLines = append(infoLines,
			fmt.Sprintf("Last run   %s", last.Time.Format("2006-01-02 15:04")))
	}

	if len(runs) > 0 {
		var totalCost float64
		for _, r := range runs {
			totalCost += r.Cost
		}
		infoLines = append(infoLines,
			fmt.Sprintf("Total cost %s across %d iterations",
				theme.Cost.Render(fmt.Sprintf("$%.4f", totalCost)), len(runs)))
	}

	if len(infoLines) > 0 {
		fmt.Fprintln(w)
		content := strings.Join(infoLines, "\n")
		fmt.Fprintln(w, theme.SummaryBox.Render(content))
	}
}
