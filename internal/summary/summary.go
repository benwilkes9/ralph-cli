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

	fmt.Println("┌──────────────────────────────────────┐")
	fmt.Println("│         JOB SUMMARY                  │")
	fmt.Println("├──────────────────────────────────────┤")
	fmt.Printf("│  Iterations       %-19d│\n", stats.Iterations)
	fmt.Printf("│  Wall time        %-19s│\n", formatDuration(wallTime))
	fmt.Printf("│  Peak context     %s / %s (%0.f%%)%s│\n",
		stream.FormatTokens(stats.PeakContext),
		stream.FormatTokens(contextLimit),
		pct,
		padding(stats.PeakContext, pct))
	fmt.Printf("│  Subagent tokens  %-19s│\n", stream.FormatTokens(stats.SubagentTokens))
	fmt.Printf("│  Total cost       $%-18.4f│\n", stats.TotalCost)
	fmt.Println("└──────────────────────────────────────┘")
}

func formatDuration(d time.Duration) string {
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", m, s)
}

func padding(peak int, pct float64) string {
	// Approximate padding to align the box border
	label := fmt.Sprintf("%s / %s (%.0f%%)", stream.FormatTokens(peak), stream.FormatTokens(contextLimit), pct)
	need := 19 - len(label)
	if need <= 0 {
		return ""
	}
	p := make([]byte, need)
	for i := range p {
		p[i] = ' '
	}
	return string(p)
}
