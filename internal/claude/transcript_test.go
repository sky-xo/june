// internal/claude/transcript_test.go
package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTranscript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent-test.jsonl")

	content := `{"type":"user","message":{"role":"user","content":"Hello"},"timestamp":"2025-01-01T12:00:00Z","agentId":"test"}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Hi there"}]},"timestamp":"2025-01-01T12:00:01Z","agentId":"test"}`

	os.WriteFile(path, []byte(content), 0644)

	entries, err := ParseTranscript(path)
	if err != nil {
		t.Fatalf("ParseTranscript: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	if entries[0].Type != "user" {
		t.Errorf("entries[0].Type = %q, want user", entries[0].Type)
	}
	if entries[1].Type != "assistant" {
		t.Errorf("entries[1].Type = %q, want assistant", entries[1].Type)
	}
}

func TestEntryContent(t *testing.T) {
	// User message with string content
	e1 := Entry{
		Type: "user",
		Message: Message{
			Role:    "user",
			Content: "Hello world",
		},
	}
	if got := e1.TextContent(); got != "Hello world" {
		t.Errorf("TextContent() = %q, want %q", got, "Hello world")
	}

	// Assistant message with content blocks
	e2 := Entry{
		Type: "assistant",
		Message: Message{
			Role: "assistant",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "Response here"},
			},
		},
	}
	if got := e2.TextContent(); got != "Response here" {
		t.Errorf("TextContent() = %q, want %q", got, "Response here")
	}
}
