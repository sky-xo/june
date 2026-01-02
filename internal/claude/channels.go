// internal/claude/channels.go
package claude

import (
	"os"
	"path/filepath"
	"strings"
)

// Channel represents a group of agents from a branch/worktree.
type Channel struct {
	Name   string  // Display name like "june:main" or "june:channels"
	Dir    string  // Full path to Claude project directory
	Agents []Agent // Agents in this channel
}

// FindRelatedProjectDirs finds all Claude project directories that share
// the same base project path (main repo + worktrees).
func FindRelatedProjectDirs(claudeProjectsDir, basePath string) []string {
	// Convert base path to Claude's dash format
	basePrefix := strings.ReplaceAll(basePath, "/", "-")

	entries, err := os.ReadDir(claudeProjectsDir)
	if err != nil {
		return nil
	}

	var related []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		// Match exact base or base with worktree suffix
		if name == basePrefix || strings.HasPrefix(name, basePrefix+"-") {
			related = append(related, filepath.Join(claudeProjectsDir, name))
		}
	}
	return related
}

// ExtractChannelName creates a display name like "june:main" or "june:feature".
// baseDir is the main repo's Claude dir name (e.g., "-Users-test-code-june")
// projectDir is the current dir name (e.g., "-Users-test-code-june--worktrees-channels")
// repoName is the repository name (e.g., "june")
func ExtractChannelName(baseDir, projectDir, repoName string) string {
	if projectDir == baseDir {
		return repoName + ":main"
	}

	// Extract worktree name from suffix
	suffix := strings.TrimPrefix(projectDir, baseDir)
	// Remove leading dashes and "worktrees" segments
	suffix = strings.TrimLeft(suffix, "-")

	// Handle nested worktrees like "--worktrees--worktrees-channels"
	// Split by "-" and find the last meaningful segment
	parts := strings.Split(suffix, "-")

	// Filter out "worktrees" and empty parts, keep last segment
	var lastPart string
	for _, p := range parts {
		if p != "" && p != "worktrees" {
			lastPart = p
		}
	}

	if lastPart == "" {
		return repoName + ":unknown"
	}
	return repoName + ":" + lastPart
}
