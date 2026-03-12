package loop

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	logfile "github.com/benwilkes9/ralph-cli/internal/log"
	"github.com/benwilkes9/ralph-cli/internal/state"
	"github.com/benwilkes9/ralph-cli/internal/stream"
	"github.com/benwilkes9/ralph-cli/internal/summary"
	"github.com/benwilkes9/ralph-cli/internal/ui"
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
	Mode           Mode
	PromptFile     string
	MaxIterations  int
	FreshContext   bool
	LogsDir        string
	Branch         string
	StateFile      string
	PlanFile       string
	SpecsDir       string
	AdditionalDirs []string // container paths to additional repos
}

// Run executes the main iteration loop.
func Run(ctx context.Context, opts *Options, w io.Writer, theme *ui.Theme) error {
	return run(ctx, opts, w, theme, &realGitClient{}, &realClaudeRunner{theme: theme})
}

func run(ctx context.Context, opts *Options, w io.Writer, theme *ui.Theme, gitCl GitClient, claudeCl ClaudeRunner) error {
	RenderHeader(w, opts, theme)

	// Seed stale detector with initial composite HEAD.
	initHead, err := compositeHead(gitCl, opts.AdditionalDirs)
	if err != nil {
		return fmt.Errorf("getting initial HEAD: %w", err)
	}
	stale := NewStaleDetector(DefaultMaxStale)
	stale.Check(initHead) // seed

	cumStats := &stream.CumulativeStats{}
	startTime := time.Now()

	var (
		cancelled    bool
		staleAborted bool
		logPaths     []string
	)
	for i := 1; ; i++ {
		if opts.MaxIterations > 0 && i > opts.MaxIterations {
			RenderMaxIterations(w, opts.MaxIterations, theme)
			break
		}

		if ctx.Err() != nil {
			cancelled = true
			break
		}

		headBefore, err := compositeHead(gitCl, opts.AdditionalDirs)
		if err != nil {
			return fmt.Errorf("getting HEAD before iteration: %w", err)
		}

		RenderBanner(w, opts.Mode, i, theme)

		logW, err := logfile.New(opts.LogsDir)
		if err != nil {
			return fmt.Errorf("creating log writer: %w", err)
		}

		iterStats, runErr := claudeCl.Run(ctx, opts, logW, w)
		logW.Close() //nolint:errcheck // best-effort log close
		logPaths = append(logPaths, logW.Path())

		if runErr != nil {
			return fmt.Errorf("running claude: %w", runErr)
		}

		if iterStats != nil {
			cumStats.Update(iterStats)
			RenderIterationSummary(w, iterStats, logW.Path(), theme)
		}

		// Check for stale iterations.
		headAfter, err := compositeHead(gitCl, opts.AdditionalDirs)
		if err != nil {
			return fmt.Errorf("getting HEAD after iteration: %w", err)
		}

		if headBefore == headAfter {
			abort, count := stale.Check(headAfter)
			RenderStaleWarning(w, count, stale.MaxStale(), theme)
			if abort {
				RenderStaleAbort(w, stale.MaxStale(), theme)
				staleAborted = true
				break
			}
		} else {
			stale.Check(headAfter) // reset

			// Push primary repo with fallback to --set-upstream.
			if pushErr := gitCl.Push(opts.Branch); pushErr != nil {
				RenderPushFallback(w, theme)
				if upErr := gitCl.PushSetUpstream(opts.Branch); upErr != nil {
					fmt.Fprintf(w, "%s\n", theme.Muted.Render(fmt.Sprintf("Push failed: %s", upErr))) //nolint:errcheck // display-only
				}
			}

			// Push additional repos that changed.
			pushAdditionalDirs(gitCl, opts, headBefore, headAfter, w, theme)
		}
	}

	summary.PrintBox(w, cumStats, time.Since(startTime), theme)
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

// compositeHead concatenates the HEAD from the primary repo and all additional
// dirs into a single string for stale detection. Any repo changing resets stale.
func compositeHead(gitCl GitClient, additionalDirs []string) (string, error) {
	head, err := gitCl.Head()
	if err != nil {
		return "", fmt.Errorf("primary HEAD: %w", err)
	}
	for _, dir := range additionalDirs {
		h, err := gitCl.HeadIn(dir)
		if err != nil {
			return "", fmt.Errorf("HeadIn(%s): %w", dir, err)
		}
		head += ":" + h
	}
	return head, nil
}

// pushAdditionalDirs pushes any additional repos whose HEAD changed during the iteration.
func pushAdditionalDirs(gitCl GitClient, opts *Options, _, _ string, w io.Writer, theme *ui.Theme) {
	for _, dir := range opts.AdditionalDirs {
		if pushErr := gitCl.PushIn(dir, opts.Branch); pushErr != nil {
			fmt.Fprintf(w, "%s\n", theme.Muted.Render(fmt.Sprintf("Push failed for %s, trying --set-upstream...", dir))) //nolint:errcheck // display-only
			if upErr := gitCl.PushSetUpstreamIn(dir, opts.Branch); upErr != nil {
				fmt.Fprintf(w, "%s\n", theme.Muted.Render(fmt.Sprintf("Push failed for %s: %s", dir, upErr))) //nolint:errcheck // display-only
			}
		}
	}
}

// claudeArgs builds the argument list for the claude CLI invocation.
func claudeArgs(additionalDirs []string) []string {
	args := make([]string, 0, 7+2*len(additionalDirs))
	args = append(args,
		"-p",
		"--dangerously-skip-permissions",
		"--output-format=stream-json",
		"--model", "opus",
		"--verbose",
	)
	for _, dir := range additionalDirs {
		args = append(args, "--add-dir", dir)
	}
	return args
}

// runClaude invokes the claude CLI, tees output to the log writer, and returns iteration stats.
func runClaude(ctx context.Context, opts *Options, logW, displayW io.Writer, theme *ui.Theme) (*stream.IterationStats, error) {
	args := claudeArgs(opts.AdditionalDirs)

	cmd := exec.CommandContext(ctx, "claude", args...) //nolint:gosec // args are static

	cmd.Stderr = os.Stderr

	promptContent, err := os.ReadFile(opts.PromptFile)
	if err != nil {
		return nil, fmt.Errorf("reading prompt file: %w", err)
	}

	// Prepend dynamic context so Claude knows the branch-specific paths.
	var header bytes.Buffer
	fmt.Fprintf(&header, "PLAN_FILE: %s\n", opts.PlanFile)
	fmt.Fprintf(&header, "SPECS_DIR: %s\n", opts.SpecsDir)
	fmt.Fprintf(&header, "BRANCH: %s\n", opts.Branch)
	if len(opts.AdditionalDirs) > 0 {
		fmt.Fprintf(&header, "ADDITIONAL_REPOS: %s\n", strings.Join(opts.AdditionalDirs, ", "))
	}
	header.WriteString("---\n")

	combined := bytes.Join([][]byte{header.Bytes(), promptContent}, nil)
	cmd.Stdin = bytes.NewReader(combined)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting claude: %w", err)
	}

	tee := io.TeeReader(stdout, logW)
	stats, processErr := stream.Process(tee, displayW, theme)

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
