package summary

import (
	"fmt"
	"io"
	"time"

	"github.com/benwilkes9/ralph-cli/internal/stream"
)

const contextLimit = 200_000

// PrintBox renders the final job summary box to w.
//
//nolint:errcheck // display-only writes; io.Writer errors are non-actionable here
func PrintBox(w io.Writer, stats *stream.CumulativeStats, wallTime time.Duration) {
	pct := float64(stats.PeakContext) / float64(contextLimit) * 100
	peakCtx := fmt.Sprintf("%s / %s (%.0f%%)",
		stream.FormatTokens(stats.PeakContext),
		stream.FormatTokens(contextLimit),
		pct)

	fmt.Fprintln(w, "┌──────────────────────────────────────────┐")
	fmt.Fprintln(w, "│             JOB SUMMARY                  │")
	fmt.Fprintln(w, "├──────────────────────────────────────────┤")
	fmt.Fprintf(w, "│  Iterations       %-23d│\n", stats.Iterations)
	fmt.Fprintf(w, "│  Wall time        %-23s│\n", formatDuration(wallTime))
	fmt.Fprintf(w, "│  Peak context     %-23s│\n", peakCtx)
	fmt.Fprintf(w, "│  Subagent tokens  %-23s│\n", stream.FormatTokens(stats.SubagentTokens))
	fmt.Fprintf(w, "│  Total cost       $%-22.4f│\n", stats.TotalCost)
	fmt.Fprintln(w, "└──────────────────────────────────────────┘")
}

func formatDuration(d time.Duration) string {
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", m, s)
}
