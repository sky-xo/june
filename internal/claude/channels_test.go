// internal/claude/channels_test.go
package claude

import (
	"os"
	"path/filepath"
	"testing"
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
