package gemini

import (
	"errors"
	"os"
	"path/filepath"
)

// ErrSessionNotFound is returned when a session file cannot be found
var ErrSessionNotFound = errors.New("session file not found")

// FindSessionFile finds a Gemini session file by session ID.
// Looks in ~/.june/gemini/sessions/{session_id}.jsonl
func FindSessionFile(sessionID string) (string, error) {
	sessionsDir, err := SessionsDir()
	if err != nil {
		return "", err
	}

	sessionFile := filepath.Join(sessionsDir, sessionID+".jsonl")
	if _, err := os.Stat(sessionFile); err != nil {
		if os.IsNotExist(err) {
			return "", ErrSessionNotFound
		}
		return "", err
	}

	return sessionFile, nil
}

// SessionFilePath returns the path where a session file should be written.
func SessionFilePath(sessionID string) (string, error) {
	sessionsDir, err := SessionsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(sessionsDir, sessionID+".jsonl"), nil
}
