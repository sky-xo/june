// internal/claude/transcript.go
package claude

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
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

// ToolSummary returns a one-line summary of the tool use, e.g. "Bash: go test ./..."
func (e Entry) ToolSummary() string {
	if blocks, ok := e.Message.Content.([]interface{}); ok {
		for _, block := range blocks {
			if m, ok := block.(map[string]interface{}); ok {
				if m["type"] != "tool_use" {
					continue
				}
				name, _ := m["name"].(string)
				input, _ := m["input"].(map[string]interface{})
				if name == "" {
					continue
				}

				detail := extractToolDetail(name, input)
				if detail != "" {
					return name + ": " + detail
				}
				return name
			}
		}
	}
	return ""
}

// ToolInput returns the tool input map for tool_use entries.
func (e Entry) ToolInput() map[string]interface{} {
	if blocks, ok := e.Message.Content.([]interface{}); ok {
		for _, block := range blocks {
			if m, ok := block.(map[string]interface{}); ok {
				if m["type"] == "tool_use" {
					if input, ok := m["input"].(map[string]interface{}); ok {
						return input
					}
				}
			}
		}
	}
	return nil
}

// extractToolDetail extracts the relevant detail from tool input based on tool type.
func extractToolDetail(name string, input map[string]interface{}) string {
	if input == nil {
		return ""
	}

	switch name {
	case "Bash":
		// Prefer description if available, otherwise use command
		if desc, ok := input["description"].(string); ok && desc != "" {
			return desc
		}
		if cmd, ok := input["command"].(string); ok {
			// Take first line only (commands can have heredocs)
			if idx := strings.Index(cmd, "\n"); idx != -1 {
				return cmd[:idx] + "..."
			}
			return cmd
		}
	case "Read":
		if fp, ok := input["file_path"].(string); ok {
			return shortenPath(fp)
		}
	case "Edit", "Write":
		if fp, ok := input["file_path"].(string); ok {
			return shortenPath(fp)
		}
	case "Grep":
		pattern, _ := input["pattern"].(string)
		path, _ := input["path"].(string)
		if pattern != "" {
			if path != "" {
				return "\"" + pattern + "\" in " + shortenPath(path)
			}
			return "\"" + pattern + "\""
		}
	case "Glob":
		if pattern, ok := input["pattern"].(string); ok {
			return pattern
		}
	case "Task":
		if desc, ok := input["description"].(string); ok {
			return desc
		}
	case "WebFetch", "WebSearch":
		if url, ok := input["url"].(string); ok {
			return url
		}
		if query, ok := input["query"].(string); ok {
			return query
		}
	}
	return ""
}

// shortenPath removes common home directory prefix and returns a shorter path.
func shortenPath(path string) string {
	// Remove /Users/xxx/ or /home/xxx/ prefix
	if idx := strings.Index(path, "/code/"); idx != -1 {
		return path[idx+6:] // Skip "/code/"
	}
	// Fallback: just return last 2-3 components
	parts := strings.Split(path, "/")
	if len(parts) > 3 {
		return strings.Join(parts[len(parts)-3:], "/")
	}
	return path
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
