package codex

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// TranscriptEntry represents a parsed entry from a Codex session file
type TranscriptEntry struct {
	Type    string
	Content string
}

// ReadTranscript reads a Codex session file from the given line offset
// Returns entries and the new line count
func ReadTranscript(path string, fromLine int) ([]TranscriptEntry, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fromLine, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Set larger buffer for long lines
	buf := make([]byte, 0, 256*1024)
	scanner.Buffer(buf, 1024*1024)

	var entries []TranscriptEntry
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		if lineNum <= fromLine {
			continue
		}

		entry := parseEntry(scanner.Bytes())
		if entry.Content != "" {
			entries = append(entries, entry)
		}
	}

	return entries, lineNum, scanner.Err()
}

func parseEntry(data []byte) TranscriptEntry {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return TranscriptEntry{}
	}

	entryType, _ := raw["type"].(string)

	switch entryType {
	case "response_item":
		// Extract message content
		if payload, ok := raw["payload"].(map[string]interface{}); ok {
			if content, ok := payload["content"].(string); ok {
				return TranscriptEntry{Type: "message", Content: content}
			}
		}
	case "agent_reasoning":
		if content, ok := raw["content"].(string); ok {
			return TranscriptEntry{Type: "reasoning", Content: content}
		}
	case "function_call":
		if name, ok := raw["name"].(string); ok {
			return TranscriptEntry{Type: "tool", Content: fmt.Sprintf("[tool: %s]", name)}
		}
	case "function_call_output":
		if output, ok := raw["output"].(string); ok {
			// Truncate long outputs
			if len(output) > 200 {
				output = output[:200] + "..."
			}
			return TranscriptEntry{Type: "tool_output", Content: output}
		}
	}

	return TranscriptEntry{}
}

// FormatEntries formats transcript entries for display
func FormatEntries(entries []TranscriptEntry) string {
	var sb strings.Builder
	for _, e := range entries {
		switch e.Type {
		case "message":
			sb.WriteString(e.Content)
			sb.WriteString("\n\n")
		case "reasoning":
			sb.WriteString("[thinking] ")
			sb.WriteString(e.Content)
			sb.WriteString("\n\n")
		case "tool":
			sb.WriteString(e.Content)
			sb.WriteString("\n")
		case "tool_output":
			sb.WriteString("  -> ")
			sb.WriteString(e.Content)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
