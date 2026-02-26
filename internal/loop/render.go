package loop

import (
	"fmt"
	"io"
	"strings"

	"github.com/benwilkes9/ralph-cli/internal/stream"
	"github.com/benwilkes9/ralph-cli/internal/ui"
)

const contextLimit = 200_000

// RenderHeader prints the ASCII banner and configuration bar at the start of a loop run.
//
//nolint:errcheck // display-only writes to terminal
func RenderHeader(w io.Writer, opts *Options, theme *ui.Theme) {
	fmt.Fprintln(w, theme.Banner())
	fmt.Fprintln(w)

	bar := theme.Separator.Render("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	modeStyle := theme.ModeStyle(string(opts.Mode))

	fmt.Fprintln(w, bar)
	fmt.Fprintf(w, "  %s     %s\n", theme.Muted.Render("Mode"), modeStyle.Render(string(opts.Mode)))
	fmt.Fprintf(w, "  %s   %s\n", theme.Muted.Render("Prompt"), opts.PromptFile)
	fmt.Fprintf(w, "  %s   %s\n", theme.Muted.Render("Branch"), theme.Info.Render(opts.Branch))
	if opts.MaxIterations > 0 {
		fmt.Fprintf(w, "  %s      %s\n", theme.Muted.Render("Max"), fmt.Sprintf("%d iterations", opts.MaxIterations))
	}
	fmt.Fprintln(w, bar)
}

// RenderBanner prints the iteration box (e.g. ╔══╗ BUILD #1 ╚══╝).
//
//nolint:errcheck // display-only writes to terminal
func RenderBanner(w io.Writer, mode Mode, iteration int, theme *ui.Theme) {
	style := theme.IterationStyle(string(mode))
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
	fmt.Fprintf(w, "  %s\n", style.Render("╔══════════════════════════════════════╗"))
	fmt.Fprintf(w, "  %s  %s  %s%s%s\n",
		style.Render("║"), style.Render(label), fmt.Sprintf("#%d", iteration), pad, style.Render("║"))
	fmt.Fprintf(w, "  %s\n", style.Render("╚══════════════════════════════════════╝"))
	fmt.Fprintln(w)
}

// RenderIterationSummary prints the per-iteration context/cost line and log path.
//
//nolint:errcheck // display-only writes to terminal
func RenderIterationSummary(w io.Writer, stats *stream.IterationStats, logPath string, theme *ui.Theme) {
	pct := stats.PeakContext * 100 / contextLimit

	fmt.Fprintf(w, "\n  %s %s / %s context (%d%%)",
		theme.Muted.Render("────"),
		stream.FormatTokens(stats.PeakContext),
		stream.FormatTokens(contextLimit),
		pct)
	if stats.Cost > 0 {
		fmt.Fprintf(w, "  %s", theme.Cost.Render(fmt.Sprintf("$%.4f", stats.Cost)))
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s\n", theme.Muted.Render("raw log: "+logPath))
}

// RenderStaleWarning prints a warning when no new commits were detected.
//
//nolint:errcheck // display-only writes to terminal
func RenderStaleWarning(w io.Writer, count, threshold int, theme *ui.Theme) {
	fmt.Fprintf(w, "%s %s\n",
		theme.Warning.Render("No new commits this iteration"),
		theme.Muted.Render(fmt.Sprintf("(stale: %d/%d)", count, threshold)))
}

// RenderStaleAbort prints the abort message when the stale threshold is reached.
//
//nolint:errcheck // display-only writes to terminal
func RenderStaleAbort(w io.Writer, threshold int, theme *ui.Theme) {
	fmt.Fprintf(w, "%s %d consecutive iterations with no commits. Stopping.\n",
		theme.Error.Render("Stale loop detected:"), threshold)
}

// RenderMaxIterations prints the max iterations reached message.
//
//nolint:errcheck // display-only writes to terminal
func RenderMaxIterations(w io.Writer, threshold int, theme *ui.Theme) {
	fmt.Fprintln(w, theme.Warning.Render(fmt.Sprintf("Reached max iterations: %d", threshold)))
}

// RenderPushFallback prints a message when falling back to push -u.
//
//nolint:errcheck // display-only writes to terminal
func RenderPushFallback(w io.Writer, theme *ui.Theme) {
	fmt.Fprintln(w, theme.Warning.Render("Failed to push. Creating remote branch..."))
}
