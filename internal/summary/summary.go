package summary

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/benwilkes9/ralph-cli/internal/stream"
	"github.com/benwilkes9/ralph-cli/internal/ui"
)

const contextLimit = 200_000

// PrintBox renders the final job summary box to w using Lip Gloss styled borders.
//
//nolint:errcheck // display-only writes; io.Writer errors are non-actionable here
func PrintBox(w io.Writer, stats *stream.CumulativeStats, wallTime time.Duration, theme *ui.Theme) {
	pct := float64(stats.PeakContext) / float64(contextLimit) * 100
	peakCtx := fmt.Sprintf("%s / %s (%.0f%%)",
		stream.FormatTokens(stats.PeakContext),
		stream.FormatTokens(contextLimit),
		pct)

	rows := []string{
		"           JOB SUMMARY",
		strings.Repeat("─", 38),
		fmt.Sprintf("Iterations       %-21d", stats.Iterations),
		fmt.Sprintf("Wall time        %-21s", formatDuration(wallTime)),
		fmt.Sprintf("Peak context     %-21s", peakCtx),
		fmt.Sprintf("Subagent tokens  %-21s", stream.FormatTokens(stats.SubagentTokens)),
		fmt.Sprintf("Total cost       %s", theme.Cost.Render(fmt.Sprintf("$%.4f", stats.TotalCost))),
	}

	content := strings.Join(rows, "\n")
	fmt.Fprintln(w, theme.SummaryBox.Render(content))
}

func formatDuration(d time.Duration) string {
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", m, s)
}
