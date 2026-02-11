package stream

import (
	"errors"
	"io"
)

// Process reads a JSONL stream, formats events to w, and returns iteration stats.
// The per-iteration summary line is NOT rendered here — that's the caller's job.
func Process(r io.Reader, w io.Writer) (*IterationStats, error) {
	parser := NewParser(r)
	formatter := NewFormatter(w)
	stats := &IterationStats{}

	for {
		evt, err := parser.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return stats, err
		}

		// Accumulate stats
		switch evt.Type {
		case eventAssistant:
			if evt.Message != nil {
				stats.ObserveAssistant(evt.Message.Usage)
				for _, block := range evt.Message.Content {
					if block.Type == contentToolUse {
						stats.ObserveToolUse()
					}
				}
			}
		case eventUser:
			if evt.ToolUseResult != nil && evt.ToolUseResult.TotalTokens > 0 {
				stats.ObserveSubagent(evt.ToolUseResult.TotalTokens)
			}
		case eventResult:
			stats.ObserveResult(evt.TotalCostUSD)
		}

		// Format for display
		if err := formatter.Format(evt); err != nil {
			return stats, err
		}
	}

	return stats, nil
}
