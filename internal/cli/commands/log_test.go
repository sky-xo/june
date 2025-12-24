package commands

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"otto/internal/db"
	"otto/internal/repo"
)

func TestRunLog(t *testing.T) {
	conn, _ := db.Open(":memory:")
	defer conn.Close()

	// Create agent
	agent := repo.Agent{ID: "test-agent", Type: "claude", Task: "test", Status: "busy"}
	repo.CreateAgent(conn, agent)

	// Create some log entries
	repo.CreateLogEntry(conn, "test-agent", "out", "stdout", "line 1")
	repo.CreateLogEntry(conn, "test-agent", "out", "stdout", "line 2")
	repo.CreateLogEntry(conn, "test-agent", "out", "stderr", "error 1")

	var buf bytes.Buffer
	err := runLog(conn, "test-agent", 0, &buf)
	if err != nil {
		t.Fatalf("runLog failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "line 1") || !strings.Contains(output, "line 2") {
		t.Errorf("expected output to contain log entries, got: %s", output)
	}
}

func TestRunLogWithTail(t *testing.T) {
	conn, _ := db.Open(":memory:")
	defer conn.Close()

	agent := repo.Agent{ID: "test-agent", Type: "claude", Task: "test", Status: "busy"}
	repo.CreateAgent(conn, agent)

	// Create 10 entries
	for i := 0; i < 10; i++ {
		repo.CreateLogEntry(conn, "test-agent", "out", "stdout", fmt.Sprintf("line %d", i))
	}

	var buf bytes.Buffer
	err := runLog(conn, "test-agent", 3, &buf)
	if err != nil {
		t.Fatalf("runLog failed: %v", err)
	}

	output := buf.String()
	// Should only have last 3 lines
	if strings.Contains(output, "line 0") {
		t.Errorf("should not contain line 0 with --tail 3")
	}
	if !strings.Contains(output, "line 9") {
		t.Errorf("should contain line 9")
	}
}
