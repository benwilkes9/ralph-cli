package summary

import (
	"fmt"
	"time"

	"github.com/benwilkes9/ralph-cli/internal/stream"
)

const contextLimit = 200_000

// PrintBox renders the final job summary box.
func PrintBox(stats *stream.CumulativeStats, wallTime time.Duration) {
	pct := float64(stats.PeakContext) / float64(contextLimit) * 100
	peakCtx := fmt.Sprintf("%s / %s (%.0f%%)",
		stream.FormatTokens(stats.PeakContext),
		stream.FormatTokens(contextLimit),
		pct)

	fmt.Println("┌──────────────────────────────────────────┐")
	fmt.Println("│             JOB SUMMARY                  │")
	fmt.Println("├──────────────────────────────────────────┤")
	fmt.Printf("│  Iterations       %-23d│\n", stats.Iterations)
	fmt.Printf("│  Wall time        %-23s│\n", formatDuration(wallTime))
	fmt.Printf("│  Peak context     %-23s│\n", peakCtx)
	fmt.Printf("│  Subagent tokens  %-23s│\n", stream.FormatTokens(stats.SubagentTokens))
	fmt.Printf("│  Total cost       $%-22.4f│\n", stats.TotalCost)
	fmt.Println("└──────────────────────────────────────────┘")
}

func formatDuration(d time.Duration) string {
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", m, s)
}
