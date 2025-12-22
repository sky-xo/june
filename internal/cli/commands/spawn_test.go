package commands

import (
	"database/sql"
	"strings"
	"testing"

	"otto/internal/repo"
)

func TestSpawnBuildsCommand(t *testing.T) {
	cmd := buildSpawnCommand("claude", "task", "sess-123")
	if got := cmd[0]; got != "claude" {
		t.Fatalf("expected claude, got %q", got)
	}
}

func TestSpawnBuildsClaudeCommand(t *testing.T) {
	cmd := buildSpawnCommand("claude", "test prompt", "session-123")
	expected := []string{"claude", "-p", "test prompt", "--session-id", "session-123"}

	if len(cmd) != len(expected) {
		t.Fatalf("expected %d args, got %d", len(expected), len(cmd))
	}

	for i, arg := range expected {
		if cmd[i] != arg {
			t.Fatalf("arg %d: expected %q, got %q", i, arg, cmd[i])
		}
	}
}

func TestSpawnBuildsCodexCommand(t *testing.T) {
	cmd := buildSpawnCommand("codex", "test prompt", "session-123")
	expected := []string{"codex", "exec", "test prompt"}

	if len(cmd) != len(expected) {
		t.Fatalf("expected %d args, got %d", len(expected), len(cmd))
	}

	for i, arg := range expected {
		if cmd[i] != arg {
			t.Fatalf("arg %d: expected %q, got %q", i, arg, cmd[i])
		}
	}
}

func TestGenerateAgentID(t *testing.T) {
	db := openTestDB(t)

	tests := []struct {
		name     string
		task     string
		expected string
	}{
		{"simple", "auth backend", "authbackend"},
		{"with dashes", "auth-backend-api", "authbackendapi"},
		{"long task", "this is a very long task name that exceeds sixteen chars", "thisisaverylongt"},
		{"special chars", "task#1: fix @bugs!", "task1fixbugs"},
		{"empty after filter", "!!!", "agent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateAgentID(db, tt.task)
			if result != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGenerateAgentIDUnique(t *testing.T) {
	db := openTestDB(t)

	// Create first agent
	_ = repo.CreateAgent(db, repo.Agent{
		ID:     "authbackend",
		Type:   "claude",
		Task:   "task",
		Status: "working",
		SessionID: sql.NullString{String: "session-1", Valid: true},
	})

	// Generate ID for same task should get -2 suffix
	result := generateAgentID(db, "auth backend")
	if result != "authbackend-2" {
		t.Fatalf("expected authbackend-2, got %q", result)
	}

	// Create second agent
	_ = repo.CreateAgent(db, repo.Agent{
		ID:     "authbackend-2",
		Type:   "claude",
		Task:   "task",
		Status: "working",
		SessionID: sql.NullString{String: "session-2", Valid: true},
	})

	// Generate ID for same task should get -3 suffix
	result = generateAgentID(db, "auth backend")
	if result != "authbackend-3" {
		t.Fatalf("expected authbackend-3, got %q", result)
	}
}

func TestBuildSpawnPrompt(t *testing.T) {
	prompt := buildSpawnPrompt("test-agent", "build auth", "", "")

	if !strings.Contains(prompt, "test-agent") {
		t.Fatal("prompt should contain agent ID")
	}
	if !strings.Contains(prompt, "build auth") {
		t.Fatal("prompt should contain task")
	}
	if !strings.Contains(prompt, "otto messages --id test-agent") {
		t.Fatal("prompt should contain communication template")
	}
}

func TestBuildSpawnPromptWithFilesAndContext(t *testing.T) {
	prompt := buildSpawnPrompt("test-agent", "task", "auth.go,user.go", "use JWT tokens")

	if !strings.Contains(prompt, "auth.go,user.go") {
		t.Fatal("prompt should contain files")
	}
	if !strings.Contains(prompt, "use JWT tokens") {
		t.Fatal("prompt should contain context")
	}
}
