package loop

import (
	"fmt"
	"io"
	"strings"

	"github.com/benmyles/ralph-cli/internal/stream"
)

const contextLimit = 200_000

// RenderHeader prints the configuration bar at the start of a loop run.
//
//nolint:errcheck // display-only writes to terminal
func RenderHeader(w io.Writer, opts *Options) {
	bar := stream.BoldBlue + "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" + stream.Reset

	color := modeColor(opts.Mode)

	fmt.Fprintln(w, bar)
	fmt.Fprintf(w, "  %sMode%s     %s%s%s\n", stream.Dim, stream.Reset, color, opts.Mode, stream.Reset)
	fmt.Fprintf(w, "  %sPrompt%s   %s%s%s\n", stream.Dim, stream.Reset, stream.White, opts.PromptFile, stream.Reset)
	fmt.Fprintf(w, "  %sBranch%s   %s%s%s\n", stream.Dim, stream.Reset, stream.BoldCyan, opts.Branch, stream.Reset)
	if opts.MaxIterations > 0 {
		fmt.Fprintf(w, "  %sMax%s      %s%d iterations%s\n", stream.Dim, stream.Reset, stream.White, opts.MaxIterations, stream.Reset)
	}
	fmt.Fprintln(w, bar)
}

// RenderBanner prints the iteration box (e.g. ╔══╗ BUILD #1 ╚══╝).
//
//nolint:errcheck // display-only writes to terminal
func RenderBanner(w io.Writer, mode Mode, iteration int) {
	color := modeColor(mode)
	label := "BUILD"
	if mode == ModePlan {
		label = "PLAN"
	}

	inner := fmt.Sprintf("  %s  #%d", label, iteration)
	padLen := 38 - len(inner)
	if padLen < 0 {
		padLen = 0
	}
	pad := strings.Repeat(" ", padLen)

	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s╔══════════════════════════════════════╗%s\n", color, stream.Reset)
	fmt.Fprintf(w, "  %s║%s  %s%s%s  %s#%d%s%s%s║%s\n",
		color, stream.Reset, color, label, stream.Reset, stream.BoldWhite, iteration, stream.Reset, pad, color, stream.Reset)
	fmt.Fprintf(w, "  %s╚══════════════════════════════════════╝%s\n", color, stream.Reset)
	fmt.Fprintln(w)
}

// RenderIterationSummary prints the per-iteration context/cost line and log path.
//
//nolint:errcheck // display-only writes to terminal
func RenderIterationSummary(w io.Writer, stats *stream.IterationStats, logPath string) {
	pct := stats.PeakContext * 100 / contextLimit

	fmt.Fprintf(w, "\n  %s────%s %s / %s context (%d%%)",
		stream.Dim, stream.Reset,
		stream.FormatTokens(stats.PeakContext),
		stream.FormatTokens(contextLimit),
		pct)
	if stats.Cost > 0 {
		fmt.Fprintf(w, "  %s$%.4f%s", stream.Magenta, stats.Cost, stream.Reset)
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %sraw log: %s%s\n", stream.Dim, logPath, stream.Reset)
}

// RenderStaleWarning prints a warning when no new commits were detected.
//
//nolint:errcheck // display-only writes to terminal
func RenderStaleWarning(w io.Writer, count, threshold int) {
	fmt.Fprintf(w, "%sNo new commits this iteration%s %s(stale: %d/%d)%s\n",
		stream.BoldYellow, stream.Reset, stream.Dim, count, threshold, stream.Reset)
}

// RenderStaleAbort prints the abort message when the stale threshold is reached.
//
//nolint:errcheck // display-only writes to terminal
func RenderStaleAbort(w io.Writer, threshold int) {
	fmt.Fprintf(w, "%sStale loop detected:%s %d consecutive iterations with no commits. Stopping.\n",
		stream.BoldRed, stream.Reset, threshold)
}

// RenderMaxIterations prints the max iterations reached message.
//
//nolint:errcheck // display-only writes to terminal
func RenderMaxIterations(w io.Writer, threshold int) {
	fmt.Fprintf(w, "%sReached max iterations: %d%s\n", stream.BoldYellow, threshold, stream.Reset)
}

// RenderPushFallback prints a message when falling back to push -u.
//
//nolint:errcheck // display-only writes to terminal
func RenderPushFallback(w io.Writer) {
	fmt.Fprintf(w, "%sFailed to push. Creating remote branch...%s\n", stream.Yellow, stream.Reset)
}

func modeColor(m Mode) string {
	if m == ModePlan {
		return stream.BoldCyan
	}
	return stream.BoldGreen
}
