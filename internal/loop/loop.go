package loop

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/benwilkes9/ralph-cli/internal/git"
	logfile "github.com/benwilkes9/ralph-cli/internal/log"
	"github.com/benwilkes9/ralph-cli/internal/state"
	"github.com/benwilkes9/ralph-cli/internal/stream"
	"github.com/benwilkes9/ralph-cli/internal/summary"
)

// Mode represents the loop mode (plan or build).
type Mode string

// Loop modes.
const (
	ModePlan  Mode = "plan"
	ModeBuild Mode = "build"
)

// Options configures a loop run.
type Options struct {
	Mode          Mode
	PromptFile    string
	MaxIterations int
	FreshContext  bool
	LogsDir       string
	Branch        string
	StateFile     string
	PlanFile      string
	SpecsDir      string
}

// Run executes the main iteration loop.
func Run(ctx context.Context, opts *Options, w io.Writer) error {
	RenderHeader(w, opts)

	// Seed stale detector with initial HEAD.
	head, err := git.Head()
	if err != nil {
		return fmt.Errorf("getting initial HEAD: %w", err)
	}
	stale := NewStaleDetector(DefaultMaxStale)
	stale.Check(head) // seed

	cumStats := &stream.CumulativeStats{}
	startTime := time.Now()

	var (
		cancelled    bool
		staleAborted bool
		logPaths     []string
	)
	for i := 1; ; i++ {
		if opts.MaxIterations > 0 && i > opts.MaxIterations {
			RenderMaxIterations(w, opts.MaxIterations)
			break
		}

		if ctx.Err() != nil {
			cancelled = true
			break
		}

		headBefore, err := git.Head()
		if err != nil {
			return fmt.Errorf("getting HEAD before iteration: %w", err)
		}

		RenderBanner(w, opts.Mode, i)

		logW, err := logfile.New(opts.LogsDir)
		if err != nil {
			return fmt.Errorf("creating log writer: %w", err)
		}

		iterStats, runErr := runClaude(ctx, opts, logW, w)
		logW.Close() //nolint:errcheck // best-effort log close
		logPaths = append(logPaths, logW.Path())

		if runErr != nil {
			return fmt.Errorf("running claude: %w", runErr)
		}

		if iterStats != nil {
			cumStats.Update(iterStats)
			RenderIterationSummary(w, iterStats, logW.Path())
		}

		// Push with fallback to --set-upstream.
		if pushErr := git.Push(opts.Branch); pushErr != nil {
			RenderPushFallback(w)
			if upErr := git.PushSetUpstream(opts.Branch); upErr != nil {
				fmt.Fprintf(w, "%sPush failed: %s%s\n", stream.Dim, upErr, stream.Reset) //nolint:errcheck // display-only
			}
		}

		// Check for stale iterations.
		headAfter, err := git.Head()
		if err != nil {
			return fmt.Errorf("getting HEAD after iteration: %w", err)
		}

		if headBefore == headAfter {
			abort, count := stale.Check(headAfter)
			RenderStaleWarning(w, count, stale.MaxStale())
			if abort {
				RenderStaleAbort(w, stale.MaxStale())
				staleAborted = true
				break
			}
		} else {
			stale.Check(headAfter) // reset
		}
	}

	summary.PrintBox(cumStats, time.Since(startTime))
	saveState(opts, cumStats, startTime, logPaths, cancelled, staleAborted)

	if staleAborted {
		return nil
	}
	if cancelled {
		return ctx.Err() //nolint:wrapcheck // propagate context cancellation directly
	}
	return nil
}

// saveState persists a RunRecord to state.json. Best-effort — errors are silently ignored.
func saveState(opts *Options, cumStats *stream.CumulativeStats, startTime time.Time, logPaths []string, cancelled, staleAborted bool) {
	if opts.StateFile == "" {
		return
	}

	runStatus := state.StatusCompleted
	switch {
	case staleAborted:
		runStatus = state.StatusStaleAbort
	case cancelled:
		runStatus = state.StatusCancelled
	case opts.MaxIterations > 0 && cumStats.Iterations >= opts.MaxIterations:
		runStatus = state.StatusMaxIterations
	}

	record := state.RunRecord{
		Mode:           string(opts.Mode),
		StartedAt:      startTime,
		FinishedAt:     time.Now(),
		Iterations:     cumStats.Iterations,
		TotalCost:      cumStats.TotalCost,
		PeakContext:    cumStats.PeakContext,
		SubagentTokens: cumStats.SubagentTokens,
		Status:         runStatus,
		LogFiles:       logPaths,
	}

	st, _ := state.Load(opts.StateFile) //nolint:errcheck // best-effort
	if st == nil {
		st = &state.State{}
	}
	st.Runs = append(st.Runs, record)
	_ = state.Save(opts.StateFile, st) //nolint:errcheck // best-effort
}

// claudeArgs builds the argument list for the claude CLI invocation.
func claudeArgs() []string {
	return []string{
		"-p",
		"--dangerously-skip-permissions",
		"--output-format=stream-json",
		"--model", "opus",
		"--verbose",
	}
}

// runClaude invokes the claude CLI, tees output to the log writer, and returns iteration stats.
func runClaude(ctx context.Context, opts *Options, logW *logfile.Writer, displayW io.Writer) (*stream.IterationStats, error) {
	args := claudeArgs()

	cmd := exec.CommandContext(ctx, "claude", args...) //nolint:gosec // args are static

	promptContent, err := os.ReadFile(opts.PromptFile)
	if err != nil {
		return nil, fmt.Errorf("reading prompt file: %w", err)
	}

	// Prepend dynamic context so Claude knows the branch-specific paths.
	var header bytes.Buffer
	fmt.Fprintf(&header, "PLAN_FILE: %s\n", opts.PlanFile)
	fmt.Fprintf(&header, "SPECS_DIR: %s\n", opts.SpecsDir)
	fmt.Fprintf(&header, "BRANCH: %s\n", opts.Branch)
	header.WriteString("---\n")

	combined := append(header.Bytes(), promptContent...)
	cmd.Stdin = bytes.NewReader(combined)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting claude: %w", err)
	}

	tee := io.TeeReader(stdout, logW)
	stats, processErr := stream.Process(tee, displayW)

	waitErr := cmd.Wait()

	// On context cancellation, return whatever stats we collected.
	if ctx.Err() != nil {
		return stats, ctx.Err() //nolint:wrapcheck // caller handles context error
	}

	if processErr != nil {
		return stats, fmt.Errorf("processing stream: %w", processErr)
	}
	if waitErr != nil {
		return stats, fmt.Errorf("claude exited: %w", waitErr)
	}

	return stats, nil
}
