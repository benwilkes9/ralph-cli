package stream

import (
	"bufio"
	"encoding/json"
	"io"
)

// Event represents a single JSONL event from Claude's stream-json output.
type Event struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message,omitempty"`

	// Top-level fields that vary by event type
	Role         string         `json:"role,omitempty"`
	Content      []ContentBlock `json:"content,omitempty"`
	StopReason   string         `json:"stop_reason,omitempty"`
	Result       *ResultData    `json:"result,omitempty"`
	SubagentType string         `json:"subtype,omitempty"`

	// Usage info (present on assistant events)
	Usage *Usage `json:"usage,omitempty"`
}

// ContentBlock represents a content element within a Claude response.
type ContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// ResultData contains cost information from a completed Claude session.
type ResultData struct {
	TotalCostUSD float64 `json:"total_cost_usd"`
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
		return nil, err
	}
	return nil, io.EOF
}
