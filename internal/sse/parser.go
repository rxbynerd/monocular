package sse

import (
	"encoding/json"
	"strings"
)

// GlobalEvent is the envelope from /global/event.
type GlobalEvent struct {
	Directory string       `json:"directory"` // may be empty for route-generated infra events
	Payload   EventPayload `json:"payload"`
}

// EventPayload contains the event type and its properties.
type EventPayload struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

// SSEParser processes raw SSE data lines and assembles complete events.
type SSEParser struct {
	buf []byte
}

// NewSSEParser creates a new SSE line parser.
func NewSSEParser() *SSEParser {
	return &SSEParser{}
}

// FeedLine processes one line from the SSE stream.
// Returns (event, nil) when a complete event is assembled.
// Returns (nil, nil) when still buffering.
// Returns (nil, err) for malformed JSON.
func (p *SSEParser) FeedLine(line string) (*GlobalEvent, error) {
	// SSE comment lines start with ':'
	if strings.HasPrefix(line, ":") {
		return nil, nil
	}

	// Ignore id: and retry: fields (server doesn't use these meaningfully)
	if strings.HasPrefix(line, "id:") || strings.HasPrefix(line, "retry:") {
		return nil, nil
	}

	// Data lines: strip "data: " or "data:" prefix and buffer
	if strings.HasPrefix(line, "data:") {
		data := strings.TrimPrefix(line, "data:")
		data = strings.TrimPrefix(data, " ")
		if len(p.buf) > 0 {
			p.buf = append(p.buf, '\n')
		}
		p.buf = append(p.buf, []byte(data)...)
		return nil, nil
	}

	// Empty line terminates the event block
	if line == "" && len(p.buf) > 0 {
		var event GlobalEvent
		err := json.Unmarshal(p.buf, &event)
		p.buf = p.buf[:0]
		if err != nil {
			return nil, err
		}
		return &event, nil
	}

	return nil, nil
}

// Reset clears any buffered data. Useful after connection errors.
func (p *SSEParser) Reset() {
	p.buf = p.buf[:0]
}
