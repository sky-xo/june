// internal/claude/projects_test.go
package claude

import (
	"strings"
	"testing"
)

func TestPathToProjectDir(t *testing.T) {
	tests := []struct {
		absPath  string
		expected string
	}{
		{"/Users/glowy/code/june", "-Users-glowy-code-june"},
		{"/home/user/project", "-home-user-project"},
	}

	for _, tt := range tests {
		got := PathToProjectDir(tt.absPath)
		if got != tt.expected {
			t.Errorf("PathToProjectDir(%q) = %q, want %q", tt.absPath, got, tt.expected)
		}
	}
}

func TestClaudeProjectsDir(t *testing.T) {
	dir := ClaudeProjectsDir()
	if dir == "" {
		t.Error("ClaudeProjectsDir() returned empty string")
	}
	// Should end with .claude/projects
	if !strings.HasSuffix(dir, ".claude/projects") {
		t.Errorf("ClaudeProjectsDir() = %q, should end with .claude/projects", dir)
	}
}
