# Otto Cleanup Plan

**Status:** Ready for implementation

## Goal

Remove all vestigial Otto code. June is now a read-only Claude Code subagent viewer TUI.

## Changes

### 1. Remove unused packages

- Delete `internal/exec/` (process spawning - not imported anywhere)
- Delete `internal/skills/` (references non-existent commands)

### 2. Simplify CLI structure

- Remove `watch` subcommand - `june` should launch TUI directly
- Update `internal/cli/root.go` to run TUI as default action (not via subcommand)
- Delete `internal/cli/commands/watch.go` (merge logic into root.go)

### 3. Update string references

Files with "otto" references to fix:
- `.gitignore` - remove "otto" binary line
- `internal/scope/scope_test.go` - update test to use "june"
- `internal/claude/integration_test.go` - update path reference
- `internal/claude/projects_test.go` - update test case
- `docs/plans/2026-01-01-subagent-viewer-mvp.md` - update example

### 4. Rewrite README.md

Minimal README covering:
- What June is (read-only subagent viewer)
- How to install/build
- How to use (`june` launches TUI)
- Basic keyboard shortcuts
- Where it reads data from

### 5. Update CLAUDE.md

Remove references to:
- Database operations
- Spawning agents
- Old commands (spawn, prompt, dm, ask, complete, etc.)
- repo package

Update to reflect current structure.

### 6. Verify

- Run `make test` - all tests pass
- Run `make build` - builds successfully
- Run `./june` - TUI launches
