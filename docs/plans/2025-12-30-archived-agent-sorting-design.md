# Archived Agent Sorting by Last Activity

**Status:** Ready for implementation
**Date:** 2025-12-30

## Problem

Archived agents in the TUI sidebar are currently sorted by `ArchivedAt` (when they were archived). Users expect them sorted by last activity (most recent message) so recently-active agents appear first.

## Solution

Query-time join: fetch last message timestamp per archived agent before sorting.

## Implementation Tasks

### Task 1: Add repo function to get last activity times

**File:** `internal/repo/messages.go`

Add function:
```go
func (r *Repo) GetAgentLastActivity(project, branch string, agentNames []string) (map[string]time.Time, error)
```

- Query: `SELECT from_agent, MAX(created_at) FROM logs WHERE project=? AND branch=? AND from_agent IN (?) GROUP BY from_agent`
- Returns map of agent name → last activity time
- Use `logs` table since it has all agent output (messages table is for inter-agent communication)

### Task 2: Update sortArchivedAgents to use last activity

**File:** `internal/tui/watch.go`

- Change `sortArchivedAgents(agents []repo.Agent)` signature to accept activity map
- Sort by activity time (descending), fall back to `ArchivedAt` if no activity found
- Update call site in `sidebarItems()` to pass activity data

### Task 3: Fetch activity data in sidebarItems

**File:** `internal/tui/watch.go`

- Before sorting archived agents, collect all archived agent names
- Call `GetAgentLastActivity()` once for all archived agents
- Pass result to `sortArchivedAgents()`

## Testing

- Unit test for `GetAgentLastActivity` with multiple agents
- Unit test for `sortArchivedAgents` with activity map
- Manual verification in TUI

## Files Changed

1. `internal/repo/messages.go` - new function
2. `internal/tui/watch.go` - updated sorting logic
3. `internal/repo/messages_test.go` - new test
4. `internal/tui/watch_test.go` - updated test (if sorting is tested)
