// internal/claude/channels.go
package claude

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sky-xo/june/internal/agent"
	"github.com/sky-xo/june/internal/db"
)

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

	// Find the last "-worktrees-" marker and take everything after it.
	// This preserves hyphenated branch names like "select-mode".
	const worktreeMarker = "-worktrees-"
	lastIdx := strings.LastIndex(suffix, worktreeMarker)
	if lastIdx != -1 {
		branchName := suffix[lastIdx+len(worktreeMarker):]
		if branchName != "" {
			return repoName + ":" + branchName
		}
	}

	// Fallback: try removing leading dashes and "worktrees" prefix
	suffix = strings.TrimLeft(suffix, "-")
	suffix = strings.TrimPrefix(suffix, "worktrees-")
	suffix = strings.TrimPrefix(suffix, "worktrees")
	suffix = strings.TrimLeft(suffix, "-")

	if suffix == "" {
		return repoName + ":unknown"
	}
	return repoName + ":" + suffix
}

// ScanChannels scans Claude project directories and merges Codex agents.
// It returns unified channels containing both Claude and Codex agents.
func ScanChannels(claudeProjectsDir, basePath, repoName string, codexDB *db.DB) ([]agent.Channel, error) {
	relatedDirs := FindRelatedProjectDirs(claudeProjectsDir, basePath)

	// Map branch -> agents
	channelMap := make(map[string][]agent.Agent)

	baseDir := strings.ReplaceAll(basePath, "/", "-")

	// 1. Scan Claude agents
	for _, dir := range relatedDirs {
		dirName := filepath.Base(dir)
		channelName := ExtractChannelName(baseDir, dirName, repoName)

		claudeAgents, err := ScanAgents(dir)
		if err != nil {
			continue
		}

		// Extract branch from channel name (e.g., "june:main" -> "main")
		branch := strings.TrimPrefix(channelName, repoName+":")

		for _, ca := range claudeAgents {
			channelMap[channelName] = append(channelMap[channelName], ca.ToUnified(basePath, branch))
		}
	}

	// 2. Load Codex agents for this repo
	if codexDB != nil {
		codexAgents, err := codexDB.ListAgentsByRepo(basePath)
		if err == nil {
			for _, ca := range codexAgents {
				channelName := repoName + ":" + ca.Branch
				if ca.Branch == "" {
					channelName = repoName + ":main"
				}
				channelMap[channelName] = append(channelMap[channelName], ca.ToUnified())
			}
		}
	}

	// 3. Build and sort channels
	var channels []agent.Channel
	for name, agents := range channelMap {
		// Sort agents within channel by LastActivity (most recent first)
		sort.Slice(agents, func(i, j int) bool {
			return agents[i].LastActivity.After(agents[j].LastActivity)
		})
		channels = append(channels, agent.Channel{Name: name, Agents: agents})
	}

	// Sort: recent activity first, then alphabetically
	sort.Slice(channels, func(i, j int) bool {
		iRecent := channels[i].HasRecentActivity()
		jRecent := channels[j].HasRecentActivity()
		if iRecent != jRecent {
			return iRecent
		}
		return channels[i].Name < channels[j].Name
	})

	return channels, nil
}
