package gemini

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindSessionFile(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Create sessions directory and a test file
	sessionsDir := filepath.Join(tmpHome, ".june", "gemini", "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionID := "8b6238bf-8332-4fc7-ba9a-2f3323119bb2"
	sessionFile := filepath.Join(sessionsDir, sessionID+".jsonl")
	if err := os.WriteFile(sessionFile, []byte(`{"type":"init"}`), 0644); err != nil {
		t.Fatal(err)
	}

	found, err := FindSessionFile(sessionID)
	if err != nil {
		t.Fatalf("FindSessionFile failed: %v", err)
	}

	if found != sessionFile {
		t.Errorf("found = %q, want %q", found, sessionFile)
	}
}

func TestFindSessionFileNotFound(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Create empty sessions directory
	sessionsDir := filepath.Join(tmpHome, ".june", "gemini", "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	_, err := FindSessionFile("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("err = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionFilePath(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	sessionID := "test-session-123"
	path, err := SessionFilePath(sessionID)
	if err != nil {
		t.Fatalf("SessionFilePath failed: %v", err)
	}

	expected := filepath.Join(tmpHome, ".june", "gemini", "sessions", sessionID+".jsonl")
	if path != expected {
		t.Errorf("path = %q, want %q", path, expected)
	}
}

func TestSessionsDir(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	dir, err := SessionsDir()
	if err != nil {
		t.Fatalf("SessionsDir failed: %v", err)
	}

	expected := filepath.Join(tmpHome, ".june", "gemini", "sessions")
	if dir != expected {
		t.Errorf("dir = %q, want %q", dir, expected)
	}
}
