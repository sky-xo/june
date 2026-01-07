# Remove Dead MCP Config Code

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove dead code related to MCP server configuration via `~/.june/config.yaml` since `GEMINI_CONFIG_DIR` is not supported by Gemini CLI.

**Architecture:** The June config system was built assuming Gemini CLI would respect `GEMINI_CONFIG_DIR` to load `settings.json` from a custom location. This env var is not implemented in Gemini CLI - they use XDG standards instead. The entire MCP config chain is dead code that should be removed.

**Tech Stack:** Go, YAML dependency to be removed

---

## Summary of Dead Code

| Location | What to Remove |
|----------|----------------|
| `internal/config/` | Entire package (config.go, config_test.go, paths.go, paths_test.go) |
| `internal/gemini/home.go` | `MCPServerConfig` struct, `WriteSettings()` function |
| `internal/gemini/home_test.go` | `TestWriteSettings`, `TestWriteSettingsEmpty` tests |
| `internal/cli/spawn.go` | MCP config block (lines 222-276), config import |
| `go.mod` / `go.sum` | `gopkg.in/yaml.v3` dependency |

---

### Task 1: Remove MCP Config Code from spawn.go

**Files:**
- Modify: `internal/cli/spawn.go`

**Step 1: Remove config import**

Delete from imports:

```go
// DELETE:
	"june/internal/config"
```

**Step 2: Remove LoadConfig call and comment**

Delete lines 222-230 (the config loading block and its comment):

```go
// DELETE THIS ENTIRE BLOCK:
	// Load June config for MCP servers
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
```

**Step 3: Fix geminiHome to discard return value**

Change line ~248 from:

```go
	geminiHome, err := gemini.EnsureGeminiHome()
```

To:

```go
	_, err = gemini.EnsureGeminiHome()
```

Note: We still call `EnsureGeminiHome()` because it creates the sessions directory needed for Gemini transcripts.

**Step 4: Remove the MCP config block**

Delete the entire `if len(cfg.MCPServers) > 0 { ... }` block (approximately lines 260-276 after previous deletions):

```go
// DELETE THIS ENTIRE BLOCK:
	// Only set GEMINI_CONFIG_DIR if we have MCP servers to configure
	// Otherwise, let Gemini use its normal ~/.gemini/ config (preserves user's existing MCP servers)
	if len(cfg.MCPServers) > 0 {
		// Write settings.json with MCP servers from config
		mcpServers := make(map[string]gemini.MCPServerConfig)
		for name, server := range cfg.MCPServers {
			mcpServers[name] = gemini.MCPServerConfig{
				Command: server.Command,
				Args:    server.Args,
				Env:     server.Env,
			}
		}
		if err := gemini.WriteSettings(geminiHome, mcpServers); err != nil {
			return fmt.Errorf("failed to write gemini settings: %w", err)
		}
		geminiCmd.Env = append(os.Environ(), fmt.Sprintf("GEMINI_CONFIG_DIR=%s", geminiHome))
	}
```

**Step 5: Run tests to verify nothing breaks**

Run: `make test`
Expected: All tests pass

**Step 6: Commit**

```bash
git add internal/cli/spawn.go
git commit -m "$(cat <<'EOF'
refactor: remove dead MCP config code from spawn

GEMINI_CONFIG_DIR is not supported by Gemini CLI - they use XDG
standards instead. Remove the config loading and settings.json
writing that was never functional.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Remove WriteSettings from gemini/home.go and Tests

**Files:**
- Modify: `internal/gemini/home.go`
- Modify: `internal/gemini/home_test.go`

**Step 1: Remove MCPServerConfig struct and WriteSettings function from home.go**

Delete lines 75-99:

```go
// DELETE THIS:
// MCPServerConfig represents an MCP server for Gemini's settings.json
type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// WriteSettings writes a settings.json file with MCP server configuration.
// Overwrites any existing settings.json.
func WriteSettings(geminiHome string, mcpServers map[string]MCPServerConfig) error {
	if mcpServers == nil {
		mcpServers = make(map[string]MCPServerConfig)
	}

	settings := map[string]interface{}{
		"mcpServers": mcpServers,
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(geminiHome, "settings.json"), data, 0600)
}
```

**Step 2: Remove encoding/json import from home.go**

```go
// DELETE from imports:
	"encoding/json"
```

**Step 3: Delete TestWriteSettings and TestWriteSettingsEmpty from home_test.go**

Delete the entire `TestWriteSettings` function (lines 153-201) and `TestWriteSettingsEmpty` function (lines 203-226).

**Step 4: Remove encoding/json import from home_test.go**

```go
// DELETE from imports:
	"encoding/json"
```

**Step 5: Run tests**

Run: `make test`
Expected: All tests pass

**Step 6: Commit**

```bash
git add internal/gemini/home.go internal/gemini/home_test.go
git commit -m "$(cat <<'EOF'
refactor: remove WriteSettings from gemini home

No longer needed since GEMINI_CONFIG_DIR doesn't work.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Delete Entire config Package

**Files:**
- Delete: `internal/config/config.go`
- Delete: `internal/config/config_test.go`
- Delete: `internal/config/paths.go`
- Delete: `internal/config/paths_test.go`
- Delete: `internal/config/` directory

**Step 1: Delete all files in internal/config/**

```bash
rm -rf internal/config/
```

Note: `DataDir()` in paths.go was only used by `config.go` (which we deleted). The entire package is now dead code.

**Step 2: Run tests**

Run: `make test`
Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/config/
git commit -m "$(cat <<'EOF'
refactor: remove internal/config package

The entire package only contained MCP config code which is dead
since GEMINI_CONFIG_DIR is not supported by Gemini CLI.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Remove yaml.v3 Dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

**Step 1: Run go mod tidy**

```bash
go mod tidy
```

This removes `gopkg.in/yaml.v3` since nothing uses it anymore.

**Step 2: Verify dependency removed**

```bash
grep yaml go.mod
```

Expected: No output (yaml not in go.mod)

**Step 3: Run tests**

Run: `make test`
Expected: All tests pass

**Step 4: Build the binary**

```bash
make build
```

Expected: Build succeeds

**Step 5: Commit**

```bash
git add go.mod go.sum
git commit -m "$(cat <<'EOF'
chore: remove unused yaml.v3 dependency

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: Final Verification

**Step 1: Run full test suite**

```bash
make test
```

Expected: All tests pass

**Step 2: Build and test spawn**

```bash
make build
./june spawn gemini "Say hello" --name test-verify
```

Expected: Agent spawns successfully (MCP servers configured directly in ~/.gemini/settings.json still work)

**Step 3: Clean up ~/.june/config.yaml (optional, user action)**

The `~/.june/config.yaml` file is now ignored. User can delete it or leave it.

---

## Notes

- `~/.june/gemini/` directory and auth file copying still works (for session isolation)
- MCP servers should be configured directly in `~/.gemini/settings.json` (Gemini's native config)
- All commits maintain passing tests - no broken intermediate states
