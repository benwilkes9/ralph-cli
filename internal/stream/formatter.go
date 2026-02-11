package stream

import "fmt"

// ANSI escape codes
const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"
	White  = "\033[37m"
	Cyan   = "\033[36m"
	Green  = "\033[32m"
	Red    = "\033[31m"
	Yellow = "\033[33m"
)

// FormatTokens formats a token count for display (e.g. "45.3k", "1.5M").
func FormatTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}
