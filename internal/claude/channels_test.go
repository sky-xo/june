// internal/claude/channels_test.go
package claude

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sky-xo/june/internal/agent"
	"github.com/sky-xo/june/internal/db"
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
		{
			// Hyphenated branch names should be preserved
			baseDir:    "-Users-test-code-june",
			projectDir: "-Users-test-code-june--worktrees-select-mode",
			repoName:   "june",
			want:       "june:select-mode",
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

	channels, err := ScanChannels(claudeProjects, "/Users/test/code/myproject", "myproject", nil)
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

	// Check agents are present - now using unified agent type
	if len(channels[0].Agents) != 1 || channels[0].Agents[0].ID != "def456" {
		t.Errorf("unexpected agents in feature1 channel: %v", channels[0].Agents)
	}
	if len(channels[1].Agents) != 1 || channels[1].Agents[0].ID != "abc123" {
		t.Errorf("unexpected agents in main channel: %v", channels[1].Agents)
	}

	// Verify unified agent fields
	if channels[0].Agents[0].Source != agent.SourceClaude {
		t.Errorf("expected source to be claude, got %s", channels[0].Agents[0].Source)
	}
	if channels[0].Agents[0].Branch != "feature1" {
		t.Errorf("expected branch to be feature1, got %s", channels[0].Agents[0].Branch)
	}
}

func TestChannel_HasRecentActivity(t *testing.T) {
	now := time.Now()

	recentAgent := agent.Agent{ID: "recent", LastActivity: now.Add(-1 * time.Hour)}
	oldAgent := agent.Agent{ID: "old", LastActivity: now.Add(-24 * time.Hour)}
	activeAgent := agent.Agent{ID: "active", LastActivity: now.Add(-5 * time.Second)}

	tests := []struct {
		name     string
		agents   []agent.Agent
		expected bool
	}{
		{"has active agent", []agent.Agent{activeAgent, oldAgent}, true},
		{"has recent agent", []agent.Agent{recentAgent, oldAgent}, true},
		{"only old agents", []agent.Agent{oldAgent}, false},
		{"empty", []agent.Agent{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := agent.Channel{Agents: tt.agents}
			if got := ch.HasRecentActivity(); got != tt.expected {
				t.Errorf("HasRecentActivity() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestScanChannels_SortsByActivityThenAlphabetical(t *testing.T) {
	tmpDir := t.TempDir()
	claudeProjects := filepath.Join(tmpDir, ".claude", "projects")

	now := time.Now()
	// Use different times within the "recent" window to ensure we're testing
	// alphabetical sort, not just time-based sort. Zebra is MORE recent than beta,
	// but beta should come first alphabetically.
	zebraTime := now.Add(-30 * time.Minute) // more recent
	betaTime := now.Add(-1 * time.Hour)     // less recent, but both are "recent"
	alphaTime := now.Add(-48 * time.Hour)   // older
	gammaTime := now.Add(-24 * time.Hour)   // gamma is more recent than alpha, but both are "old"

	// Create channels: zebra (recent), alpha (old), beta (recent), gamma (old)
	dirs := []struct {
		name    string
		modTime time.Time
	}{
		{"-Users-test-code-proj--worktrees-zebra", zebraTime},
		{"-Users-test-code-proj--worktrees-alpha", alphaTime},
		{"-Users-test-code-proj--worktrees-beta", betaTime},
		{"-Users-test-code-proj--worktrees-gamma", gammaTime},
	}

	for _, d := range dirs {
		dir := filepath.Join(claudeProjects, d.name)
		os.MkdirAll(dir, 0755)
		agentFile := filepath.Join(dir, "agent-test.jsonl")
		os.WriteFile(agentFile, []byte(`{"type":"user","message":{"role":"user","content":"Test"}}`+"\n"), 0644)
		os.Chtimes(agentFile, d.modTime, d.modTime)
	}

	channels, err := ScanChannels(claudeProjects, "/Users/test/code/proj", "proj", nil)
	if err != nil {
		t.Fatalf("ScanChannels failed: %v", err)
	}

	// Expected order: recent channels alphabetically (beta, zebra), then old alphabetically (alpha, gamma)
	expectedOrder := []string{"proj:beta", "proj:zebra", "proj:alpha", "proj:gamma"}

	if len(channels) != 4 {
		t.Fatalf("expected 4 channels, got %d", len(channels))
	}

	for i, ch := range channels {
		if ch.Name != expectedOrder[i] {
			t.Errorf("channel[%d] = %s, want %s", i, ch.Name, expectedOrder[i])
		}
	}
}

func TestScanChannels_Integration(t *testing.T) {
	// Create a structure mimicking real June worktrees
	tmpDir := t.TempDir()
	claudeProjects := filepath.Join(tmpDir, ".claude", "projects")

	// Mimic: june main + 2 worktrees (including hyphenated name)
	dirs := []struct {
		name   string
		agents []string
	}{
		{"-Users-test-code-june", []string{"agent-main1.jsonl", "agent-main2.jsonl"}},
		{"-Users-test-code-june--worktrees-channels", []string{"agent-ch1.jsonl"}},
		{"-Users-test-code-june--worktrees-select-mode", []string{"agent-sel1.jsonl", "agent-sel2.jsonl"}},
	}

	for _, d := range dirs {
		dir := filepath.Join(claudeProjects, d.name)
		os.MkdirAll(dir, 0755)
		for _, a := range d.agents {
			os.WriteFile(filepath.Join(dir, a), []byte(`{"type":"user","message":{"role":"user","content":"Test task"}}`+"\n"), 0644)
		}
	}

	channels, err := ScanChannels(claudeProjects, "/Users/test/code/june", "june", nil)
	if err != nil {
		t.Fatalf("ScanChannels failed: %v", err)
	}

	// Verify all channels found
	if len(channels) != 3 {
		t.Fatalf("expected 3 channels, got %d", len(channels))
	}

	// Verify channel names contain expected patterns
	names := make(map[string]bool)
	for _, ch := range channels {
		names[ch.Name] = true
	}
	if !names["june:main"] {
		t.Error("missing june:main channel")
	}
	if !names["june:channels"] {
		t.Error("missing june:channels channel")
	}
	if !names["june:select-mode"] {
		t.Error("missing june:select-mode channel")
	}
}

func TestScanChannels_MergesCodexAgents(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	claudeProjects := filepath.Join(tmpDir, ".claude", "projects")

	// Create one Claude project dir with an agent
	mainDir := filepath.Join(claudeProjects, "-Users-test-code-myproject")
	os.MkdirAll(mainDir, 0755)
	os.WriteFile(filepath.Join(mainDir, "agent-claude1.jsonl"), []byte(`{"type":"user","message":{"role":"user","content":"Claude task"}}`+"\n"), 0644)

	// Create a real DB with Codex agents
	dbPath := filepath.Join(tmpDir, "test.db")
	testDB, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer testDB.Close()

	// Add a Codex agent on the same branch
	err = testDB.CreateAgent(db.Agent{
		Name:        "codex-agent",
		ULID:        "codex123",
		SessionFile: "/tmp/session.jsonl",
		RepoPath:    "/Users/test/code/myproject",
		Branch:      "main",
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	// Add a Codex agent on a different branch
	err = testDB.CreateAgent(db.Agent{
		Name:        "codex-feature",
		ULID:        "codex456",
		SessionFile: "/tmp/session2.jsonl",
		RepoPath:    "/Users/test/code/myproject",
		Branch:      "feature",
	})
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	channels, err := ScanChannels(claudeProjects, "/Users/test/code/myproject", "myproject", testDB)
	if err != nil {
		t.Fatalf("ScanChannels failed: %v", err)
	}

	// Should have 2 channels: main (with Claude + Codex) and feature (Codex only)
	if len(channels) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(channels))
	}

	// Find channels by name
	var mainChannel, featureChannel *agent.Channel
	for i := range channels {
		if channels[i].Name == "myproject:main" {
			mainChannel = &channels[i]
		}
		if channels[i].Name == "myproject:feature" {
			featureChannel = &channels[i]
		}
	}

	if mainChannel == nil {
		t.Fatal("missing myproject:main channel")
	}
	if featureChannel == nil {
		t.Fatal("missing myproject:feature channel")
	}

	// Main channel should have 2 agents (1 Claude + 1 Codex)
	if len(mainChannel.Agents) != 2 {
		t.Errorf("expected 2 agents in main channel, got %d", len(mainChannel.Agents))
	}

	// Feature channel should have 1 agent (Codex only)
	if len(featureChannel.Agents) != 1 {
		t.Errorf("expected 1 agent in feature channel, got %d", len(featureChannel.Agents))
	}

	// Verify sources are correct
	var hasClaude, hasCodex bool
	for _, a := range mainChannel.Agents {
		if a.Source == agent.SourceClaude {
			hasClaude = true
		}
		if a.Source == agent.SourceCodex {
			hasCodex = true
		}
	}
	if !hasClaude {
		t.Error("main channel missing Claude agent")
	}
	if !hasCodex {
		t.Error("main channel missing Codex agent")
	}

	// Feature channel should only have Codex
	if featureChannel.Agents[0].Source != agent.SourceCodex {
		t.Errorf("expected feature agent to be codex, got %s", featureChannel.Agents[0].Source)
	}
}
