// internal/claude/channels_test.go
package claude

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindRelatedProjectDirs(t *testing.T) {
	// Create temp Claude projects directory structure
	tmpDir := t.TempDir()
	claudeProjects := filepath.Join(tmpDir, ".claude", "projects")
	os.MkdirAll(claudeProjects, 0755)

	// Create project dirs
	dirs := []string{
		"-Users-test-code-myproject",
		"-Users-test-code-myproject--worktrees-feature1",
		"-Users-test-code-myproject--worktrees-feature2",
		"-Users-test-code-other", // unrelated
	}
	for _, d := range dirs {
		os.MkdirAll(filepath.Join(claudeProjects, d), 0755)
	}

	// Test finding related dirs
	basePath := "/Users/test/code/myproject"
	related := FindRelatedProjectDirs(claudeProjects, basePath)

	if len(related) != 3 {
		t.Errorf("expected 3 related dirs, got %d: %v", len(related), related)
	}
}

func TestExtractChannelName(t *testing.T) {
	tests := []struct {
		baseDir    string
		projectDir string
		repoName   string
		want       string
	}{
		{
			baseDir:    "-Users-test-code-myproject",
			projectDir: "-Users-test-code-myproject",
			repoName:   "myproject",
			want:       "myproject:main",
		},
		{
			baseDir:    "-Users-test-code-myproject",
			projectDir: "-Users-test-code-myproject--worktrees-feature1",
			repoName:   "myproject",
			want:       "myproject:feature1",
		},
		{
			baseDir:    "-Users-test-code-june",
			projectDir: "-Users-test-code-june--worktrees--worktrees-channels",
			repoName:   "june",
			want:       "june:channels",
		},
	}

	for _, tt := range tests {
		got := ExtractChannelName(tt.baseDir, tt.projectDir, tt.repoName)
		if got != tt.want {
			t.Errorf("ExtractChannelName(%q, %q, %q) = %q, want %q",
				tt.baseDir, tt.projectDir, tt.repoName, got, tt.want)
		}
	}
}

func TestScanChannels(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	claudeProjects := filepath.Join(tmpDir, ".claude", "projects")

	// Create two project dirs with agent files
	mainDir := filepath.Join(claudeProjects, "-Users-test-code-myproject")
	worktreeDir := filepath.Join(claudeProjects, "-Users-test-code-myproject--worktrees-feature1")
	os.MkdirAll(mainDir, 0755)
	os.MkdirAll(worktreeDir, 0755)

	// Create agent files
	os.WriteFile(filepath.Join(mainDir, "agent-abc123.jsonl"), []byte(`{"type":"user","message":{"role":"user","content":"Main task"}}`+"\n"), 0644)
	os.WriteFile(filepath.Join(worktreeDir, "agent-def456.jsonl"), []byte(`{"type":"user","message":{"role":"user","content":"Feature task"}}`+"\n"), 0644)

	// Touch the feature file to make it more recent
	futureTime := time.Now().Add(time.Hour)
	os.Chtimes(filepath.Join(worktreeDir, "agent-def456.jsonl"), futureTime, futureTime)

	channels, err := ScanChannels(claudeProjects, "/Users/test/code/myproject", "myproject")
	if err != nil {
		t.Fatalf("ScanChannels failed: %v", err)
	}

	if len(channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(channels))
	}

	// Check channel names (sorted by most recent agent)
	if channels[0].Name != "myproject:feature1" {
		t.Errorf("expected first channel to be myproject:feature1, got %s", channels[0].Name)
	}
	if channels[1].Name != "myproject:main" {
		t.Errorf("expected second channel to be myproject:main, got %s", channels[1].Name)
	}

	// Check agents are present
	if len(channels[0].Agents) != 1 || channels[0].Agents[0].ID != "def456" {
		t.Errorf("unexpected agents in feature1 channel: %v", channels[0].Agents)
	}
	if len(channels[1].Agents) != 1 || channels[1].Agents[0].ID != "abc123" {
		t.Errorf("unexpected agents in main channel: %v", channels[1].Agents)
	}
}
