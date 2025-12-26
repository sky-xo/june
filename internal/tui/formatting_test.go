package tui

import (
	"database/sql"
	"testing"

	"otto/internal/repo"
)

func TestFormatLogEntryReasoning(t *testing.T) {
	entry := repo.LogEntry{EventType: "reasoning", Content: sql.NullString{String: "I will do X", Valid: true}}
	out := FormatLogEntry(entry)
	if out != "[reasoning] I will do X" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestFormatLogEntryCommandExecutionBoth(t *testing.T) {
	entry := repo.LogEntry{
		EventType: "command_execution",
		Command:   sql.NullString{String: "git status", Valid: true},
		Content:   sql.NullString{String: "On branch main", Valid: true},
	}
	out := FormatLogEntry(entry)
	expected := "git status\nOn branch main"
	if out != expected {
		t.Fatalf("unexpected output: %q, expected: %q", out, expected)
	}
}

func TestFormatLogEntryCommandExecutionCommandOnly(t *testing.T) {
	entry := repo.LogEntry{
		EventType: "command_execution",
		Command:   sql.NullString{String: "ls -la", Valid: true},
		Content:   sql.NullString{}, // empty/invalid
	}
	out := FormatLogEntry(entry)
	if out != "ls -la" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestFormatLogEntryCommandExecutionContentOnly(t *testing.T) {
	entry := repo.LogEntry{
		EventType: "command_execution",
		Command:   sql.NullString{}, // empty/invalid
		Content:   sql.NullString{String: "Command output here", Valid: true},
	}
	out := FormatLogEntry(entry)
	if out != "Command output here" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestFormatLogEntryAgentMessage(t *testing.T) {
	entry := repo.LogEntry{
		EventType: "agent_message",
		Content:   sql.NullString{String: "Task completed successfully", Valid: true},
	}
	out := FormatLogEntry(entry)
	if out != "Task completed successfully" {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestFormatLogEntryUnknownType(t *testing.T) {
	entry := repo.LogEntry{
		EventType: "unknown_event",
		Content:   sql.NullString{String: "Some content", Valid: true},
	}
	out := FormatLogEntry(entry)
	if out != "Some content" {
		t.Fatalf("unexpected output: %q", out)
	}
}
