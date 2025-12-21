package tui

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestNewModel(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Create model
	m := NewModel(db)

	// Verify initial state
	if m.db == nil {
		t.Error("expected db to be set")
	}
	if len(m.messages) != 0 {
		t.Error("expected empty messages list")
	}
	if len(m.agents) != 0 {
		t.Error("expected empty agents list")
	}
	if m.scrollOffset != 0 {
		t.Error("expected scrollOffset to be 0")
	}
}

func TestModelView(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Create model with size
	m := NewModel(db)
	m.width = 80
	m.height = 24

	// Should render without panic
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}
