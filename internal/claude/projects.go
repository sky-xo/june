// internal/claude/projects.go
package claude

import (
	"os"
	"path/filepath"
	"strings"
)

// ClaudeProjectsDir returns ~/.claude/projects
func ClaudeProjectsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "projects")
}

// PathToProjectDir converts an absolute path to Claude's directory format.
// Example: /Users/glowy/code/otto -> -Users-glowy-code-otto
func PathToProjectDir(absPath string) string {
	return strings.ReplaceAll(absPath, "/", "-")
}

// ProjectDir returns the full path to a project's Claude directory.
func ProjectDir(absPath string) string {
	return filepath.Join(ClaudeProjectsDir(), PathToProjectDir(absPath))
}
