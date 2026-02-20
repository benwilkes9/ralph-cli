package stream

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Event type constants.
const (
	eventAssistant = "assistant"
	eventUser      = "user"
	eventResult    = "result"
	contentToolUse = "tool_use"
)

// ANSI escape codes
const (
	Reset      = "\033[0m"
	Bold       = "\033[1m"
	Dim        = "\033[2m"
	White      = "\033[37m"
	Green      = "\033[32m"
	Red        = "\033[31m"
	Yellow     = "\033[33m"
	Cyan       = "\033[36m"
	Magenta    = "\033[35m"
	BoldCyan   = "\033[1;36m"
	BoldRed    = "\033[1;31m"
	BoldWhite  = "\033[1;37m"
	BoldGreen  = "\033[1;32m"
	BoldYellow = "\033[1;33m"
	BoldBlue   = "\033[1;34m"
)

// FormatTokens formats a token count for display (e.g. "45.3k", "1.5M").
// Uses floor (integer division), matching the bash implementation.
func FormatTokens(n int) string {
	switch {
	case n >= 1_000_000:
		whole := n / 100_000
		return fmt.Sprintf("%d.%dM", whole/10, whole%10)
	case n >= 1_000:
		whole := n / 100
		return fmt.Sprintf("%d.%dk", whole/10, whole%10)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// paramPriority is the extraction priority for tool_use input parameters.
var paramPriority = []string{
	"file_path", "description", "command", "pattern", "query", "url", "skill",
}

// Formatter writes formatted stream events to an io.Writer.
type Formatter struct {
	w io.Writer
}

// NewFormatter creates a Formatter that writes to w.
func NewFormatter(w io.Writer) *Formatter {
	return &Formatter{w: w}
}

// Format writes a human-readable representation of an event.
func (f *Formatter) Format(evt *Event) error {
	switch evt.Type {
	case eventAssistant:
		return f.formatAssistant(evt)
	case eventUser:
		return f.formatUser(evt)
	default:
		return nil
	}
}

func (f *Formatter) formatAssistant(evt *Event) error {
	if evt.Message == nil {
		return nil
	}
	for _, block := range evt.Message.Content {
		switch block.Type {
		case "text":
			if _, err := fmt.Fprintf(f.w, "%s%s%s%s\n", Bold, White, block.Text, Reset); err != nil {
				return fmt.Errorf("writing text block: %w", err)
			}
		case contentToolUse:
			if err := f.formatToolUse(&block); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *Formatter) formatToolUse(block *ContentBlock) error {
	if block.Name == "Task" {
		return f.formatTaskToolUse(block)
	}
	param := extractParam(block.Input)
	if _, err := fmt.Fprintf(f.w, "  %s· %s %s%s\n", Dim, block.Name, param, Reset); err != nil {
		return fmt.Errorf("writing tool use: %w", err)
	}
	return nil
}

func (f *Formatter) formatTaskToolUse(block *ContentBlock) error {
	var input map[string]json.RawMessage
	if err := json.Unmarshal(block.Input, &input); err != nil {
		return fmt.Errorf("parsing Task input: %w", err)
	}

	subagentType := jsonString(input["subagent_type"])
	if subagentType == "" {
		subagentType = jsonString(input["sub_agent_type"])
	}
	if subagentType == "" {
		subagentType = "agent"
	}

	description := jsonString(input["description"])
	if description == "" {
		description = "\u2014"
	}

	line := fmt.Sprintf("  %s▶ %s%s  %s%q%s", BoldCyan, subagentType, Reset, White, description, Reset)

	if model := jsonString(input["model"]); model != "" {
		line += fmt.Sprintf("  %smodel=%s%s", Dim, model, Reset)
	}
	if raw, ok := input["max_turns"]; ok {
		var n json.Number
		if json.Unmarshal(raw, &n) == nil {
			line += fmt.Sprintf("  %smax_turns=%s%s", Dim, n.String(), Reset)
		}
	}

	if _, err := fmt.Fprintln(f.w, line); err != nil {
		return fmt.Errorf("writing task tool use: %w", err)
	}
	return nil
}

func (f *Formatter) formatUser(evt *Event) error {
	tr := evt.ToolUseResult
	if tr == nil || tr.TotalTokens == 0 {
		return nil
	}

	if tr.Status == "completed" {
		duration := fmt.Sprintf("%.0f", float64(tr.TotalDurationMs)/1000)
		tools := fmt.Sprintf("%d", tr.TotalToolUseCount)
		tokens := FormatTokens(tr.TotalTokens)
		if _, err := fmt.Fprintf(f.w, "    %s✓ %s%s%ss, %s tool calls, %s tokens%s\n",
			Green, Reset, Dim, duration, tools, tokens, Reset); err != nil {
			return fmt.Errorf("writing tool result: %w", err)
		}
	} else {
		status := tr.Status
		if status == "" {
			status = "unknown"
		}
		if _, err := fmt.Fprintf(f.w, "    %s✗ %s%s\n", BoldRed, status, Reset); err != nil {
			return fmt.Errorf("writing tool error: %w", err)
		}
	}
	return nil
}

// extractParam extracts the most relevant parameter value from tool input JSON.
func extractParam(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var input map[string]json.RawMessage
	if err := json.Unmarshal(raw, &input); err != nil {
		return ""
	}

	for _, key := range paramPriority {
		if val, ok := input[key]; ok {
			s := jsonStringRaw(val)
			if s != "" {
				return truncate(s, 60)
			}
		}
	}

	// Fallback: sorted key names
	keys := make([]string, 0, len(input))
	for k := range input {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return truncate(strings.Join(keys, ", "), 60)
}

// jsonString unmarshals a JSON string value. Returns "" on any error.
func jsonString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}

// jsonStringRaw attempts to unmarshal as a string, falling back to the raw value.
func jsonStringRaw(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		// Not a string — use raw representation
		return strings.TrimSpace(string(raw))
	}
	return s
}

func truncate(s string, limit int) string {
	runes := []rune(s)
	if len(runes) > limit {
		return string(runes[:limit]) + "…"
	}
	return s
}
