// internal/claude/transcript.go
package claude

import (
	"bufio"
	"encoding/json"
	"os"
	"time"
)

// Entry represents a single line in an agent transcript.
type Entry struct {
	Type      string    `json:"type"` // "user" or "assistant"
	Message   Message   `json:"message"`
	AgentID   string    `json:"agentId"`
	Timestamp time.Time `json:"timestamp"`
}

// Message represents the message content.
type Message struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"` // string or []ContentBlock
	StopReason *string     `json:"stop_reason"`
}

// ContentBlock represents a content block in assistant messages.
type ContentBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Name  string `json:"name,omitempty"`  // for tool_use
	Input any    `json:"input,omitempty"` // for tool_use
}

// TextContent extracts the text content from an entry.
func (e Entry) TextContent() string {
	switch c := e.Message.Content.(type) {
	case string:
		return c
	case []interface{}:
		for _, block := range c {
			if m, ok := block.(map[string]interface{}); ok {
				if m["type"] == "text" {
					if text, ok := m["text"].(string); ok {
						return text
					}
				}
			}
		}
	}
	return ""
}

// ToolName returns the tool name if this is a tool_use entry.
func (e Entry) ToolName() string {
	if blocks, ok := e.Message.Content.([]interface{}); ok {
		for _, block := range blocks {
			if m, ok := block.(map[string]interface{}); ok {
				if m["type"] == "tool_use" {
					if name, ok := m["name"].(string); ok {
						return name
					}
				}
			}
		}
	}
	return ""
}

// IsToolResult returns true if this is a tool_result entry.
func (e Entry) IsToolResult() bool {
	if blocks, ok := e.Message.Content.([]interface{}); ok {
		for _, block := range blocks {
			if m, ok := block.(map[string]interface{}); ok {
				if m["type"] == "tool_result" {
					return true
				}
			}
		}
	}
	return false
}

// ParseTranscript reads a JSONL file and returns all entries.
func ParseTranscript(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)

	// Increase buffer size for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 1MB max line

	for scanner.Scan() {
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue // Skip malformed lines
		}
		entries = append(entries, e)
	}

	return entries, scanner.Err()
}
