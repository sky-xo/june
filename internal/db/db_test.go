package db

import (
	"path/filepath"
	"testing"
)

func TestEnsureSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "otto.db")

	conn, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer conn.Close()

	// Verify tables exist
	var name string
	if err := conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='agents'").Scan(&name); err != nil {
		t.Fatalf("agents table missing: %v", err)
	}
	if err := conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='messages'").Scan(&name); err != nil {
		t.Fatalf("messages table missing: %v", err)
	}

	// Verify indexes exist
	indexes := []string{"idx_messages_created", "idx_agents_status"}
	for _, idx := range indexes {
		if err := conn.QueryRow("SELECT name FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&name); err != nil {
			t.Fatalf("index %q missing: %v", idx, err)
		}
	}
}
