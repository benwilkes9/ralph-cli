package stream

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

// Event represents a single JSONL event from Claude's stream-json output.
type Event struct {
	Type          string         `json:"type"`
	Message       *Message       `json:"message,omitempty"`
	ToolUseResult *ToolUseResult `json:"tool_use_result,omitempty"`
	TotalCostUSD  float64        `json:"total_cost_usd,omitempty"`
}

// Message represents a Claude message with role, content, and usage.
type Message struct {
	Model   string         `json:"model,omitempty"`
	Role    string         `json:"role,omitempty"`
	Content []ContentBlock `json:"content,omitempty"`
	Usage   *Usage         `json:"usage,omitempty"`
}

// ContentBlock represents a content element within a Claude response.
type ContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// ToolUseResult contains the result of a tool invocation.
type ToolUseResult struct {
	// Regular tool fields
	Stdout string `json:"stdout,omitempty"`
	// Subagent fields (discriminator: TotalTokens > 0)
	Status            string `json:"status,omitempty"`
	TotalTokens       int    `json:"totalTokens,omitempty"`
	TotalDurationMs   int    `json:"totalDurationMs,omitempty"`
	TotalToolUseCount int    `json:"totalToolUseCount,omitempty"`
}

// Usage tracks token consumption for a single Claude response.
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// Parser reads JSONL lines and emits Events.
type Parser struct {
	scanner *bufio.Scanner
}

// NewParser creates a Parser that reads from r.
func NewParser(r io.Reader) *Parser {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB line buffer
	return &Parser{scanner: s}
}

// Next reads the next event. Returns io.EOF when done.
func (p *Parser) Next() (*Event, error) {
	for p.scanner.Scan() {
		line := p.scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var evt Event
		if err := json.Unmarshal(line, &evt); err != nil {
			// Skip malformed lines
			continue
		}
		return &evt, nil
	}

	if err := p.scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning stream: %w", err)
	}
	return nil, io.EOF
}
