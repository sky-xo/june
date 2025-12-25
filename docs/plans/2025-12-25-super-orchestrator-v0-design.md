# Super Orchestrator V0 Design

**Status:** Approved
**Created:** 2025-12-25

## Summary

This document captures the refined architecture decisions from the 2025-12-25 brainstorming session. It resolves the "split control planes" issue identified in architectural reviews and defines a simpler V0 approach.

## Core Problems Being Solved

1. **Claude skips rules/steps** - Despite strict instructions, Claude rationalizes skipping skills
2. **Codex forgets to poll** - When orchestrating, Codex must manually check for agent responses
3. **Compaction destroys context** - Skills, todos, and progress lost after compaction
4. **Need reliable multi-agent orchestration** - Coordinate Claude and Codex agents

## V0 Architecture Decision

### The Key Insight

The earlier designs had **two control planes** in tension:
- Super Orchestrator: `@otto` as central agent with event bus
- Flow Engine: Go harness controlling the flow

**V0 Resolution:** Use Codex as orchestrator (follows instructions), daemon handles wake-ups and context injection. No hard gates in V0 - just soft reminders.

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│ otto (TUI + Daemon)                                             │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ Event Bus                                                   ││
│  │  - Detects @mentions in messages                            ││
│  │  - Detects agent completion/failure                         ││
│  │  - Detects compaction events                                ││
│  └─────────────────────────────────────────────────────────────┘│
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ Wake-up Handler                                             ││
│  │  - Builds context injection (state + new messages)          ││
│  │  - Calls `otto prompt @otto "[context]"`                    ││
│  │  - Re-injects full skills after compaction                  ││
│  └─────────────────────────────────────────────────────────────┘│
│                              │                                  │
│                              ▼                                  │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ Orchestrator Agent (Codex, detached mode)                   ││
│  │  - Runs the skill flow (brainstorm → plan → impl → ...)     ││
│  │  - Spawns subagents via `otto spawn`                        ││
│  │  - Communicates via `otto say`                              ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

## Key Decisions

| Topic | Decision |
|-------|----------|
| **Primary interface** | TUI (`otto`) - will rename `otto watch` to just `otto` |
| **V0 orchestrator model** | Codex (reliable, follows instructions) |
| **Enforcement approach** | Soft reminders (Option C) - inject context, no hard gates |
| **Compaction handling** | Detect `context_compacted` event, re-inject full skills |
| **Agent wake-ups** | Daemon auto-wakes on @mentions and completions |
| **Headless mode** | `otto --headless` for future API/integrations (not V0 priority) |
| **Database** | Single global DB at `~/.otto/otto.db` (not per-project) |

## Global Database

Otto uses a **single global database** rather than one per project/branch:

```
~/.otto/otto.db    # All projects, all branches, all agents
```

**Why global?**
- **Cross-project coordination** - Frontend repo can message backend repo orchestrator
- **Single daemon** - One process watches one DB, routes all events
- **Unified TUI** - See all projects/branches in sidebar without multi-DB complexity
- **Simpler backup** - One file to back up

**Implications:**
- All tables have `project` and `branch` columns
- Queries must always filter by project/branch (indexes make this fast)
- Addressing supports cross-project: `@project:branch:agent`

## Addressing

Agents are addressed with hierarchical mentions using `:` as separator (not `/`, which would be ambiguous with branch names like `feature/login`).

| Format | Meaning | Example |
|--------|---------|---------|
| `@agent` | Same project, same branch | `@impl-1` |
| `@branch:agent` | Same project, different branch | `@feature/login:impl-1` |
| `@project:branch:agent` | Different project | `@backend-api:main:otto` |
| `@otto` | Current branch's orchestrator | Always resolves locally |
| `@human` | Human operator | TUI notification |

**Resolution:** When an agent says `@impl-1`, the daemon knows the sender's project/branch and resolves the full address. Cross-project mentions require explicit `@project:branch:agent` format.

**Why `:`?** Branch names can contain `/` (e.g., `feature/login`), so using `/` as separator would be ambiguous. The `:` separator is unambiguous and requires no escaping.

## Modes of Operation

| Mode | How it works |
|------|--------------|
| **TUI mode** (`otto`) | Primary interface. Daemon + TUI in one process. Shows channels, chat with orchestrator. |
| **CLI mode** (from orchestrator) | Orchestrator uses `otto spawn`, `otto say`, etc. These are the API. |
| **Headless** (`otto --headless`) | Future: daemon without TUI, for Slack/API integrations. |

## Context Injection

### On Normal Wake-up (no compaction)

Only inject NEW information the orchestrator hasn't seen:

```
[New messages since last wake]
@impl-1: implementation complete, tests passing
@reviewer: LGTM, approved

[State summary - always included, cheap]
Phase: implementation | Task 4/7: Add password validation
```

**Message count is naturally small.** Between wake-ups, there are typically only 2-5 messages:
- 1 completion/failure message (the trigger)
- Maybe 1-2 status updates via `otto say`
- Maybe a question if agent got stuck

No message limits needed for V0.

### After Compaction

Codex loses skills after compaction. Re-inject full context:

```
[Skills - re-injected after compaction]
You are the orchestrator using subagent-driven-development.
Follow this flow: brainstorming → planning → implementation → review → finishing
...

[State summary]
Phase: implementation | Task 4/7: Add password validation

[New messages]
@impl-1: implementation complete
```

### Detecting Compaction (Codex)

Parse the JSON output stream for `context_compacted` event:

```json
{"type":"context_compacted"}
```

When detected, set flag in DB. Next wake-up re-injects full skills.

## Codex JSON Output Format

From research of `github.com/openai/codex` source:

### Event Types

| Event | Description |
|-------|-------------|
| `thread.started` | Session start, includes `thread_id` |
| `turn.started` | Turn begins |
| `turn.completed` | Turn ends, includes token usage |
| `turn.failed` | Turn failed with error |
| `item.started/updated/completed` | Item lifecycle |
| `context_compacted` | **Compaction happened** |
| `warning` | General warning (e.g., post-compaction warning) |
| `error` | Fatal error |

### Item Types (in `item.*` events)

| Item Type | Fields | Display As |
|-----------|--------|------------|
| `reasoning` | `text` | Collapsed thinking (gray) |
| `agent_message` | `text` | Message bubble |
| `command_execution` | `command`, `aggregated_output`, `exit_code`, `status` | Code block |
| `file_change` | file details | Diff view |
| `mcp_tool_call` | tool call details | Tool badge |
| `todo_list` | plan items | Checklist |

### Implementation: Parse and Store

Currently we store raw JSON. Improve to:

1. Parse each JSON line as it arrives
2. Detect event types (especially `context_compacted`)
3. Extract content for nice display
4. Store structured data in DB

## TUI Design

### Main View (click project name)

```
┌──────────────┬──────────────────────────────────────────────┐
│ Projects     │ Activity Feed (agent-to-agent, status)       │
│              │ @impl-1 completed task 3                     │
│ otto/main    │ @reviewer: LGTM on the API changes           │
│   @impl-1 *  ├──────────────────────────────────────────────┤
│   @reviewer  │ Chat with Orchestrator                       │
│              │                                              │
│              │ You: build a login page                      │
│              │ @otto: Let's brainstorm. What auth...        │
│              │                                              │
│              │ > [type here]                                │
└──────────────┴──────────────────────────────────────────────┘
```

- **Top panel:** Activity feed (agent status, agent-to-agent messages)
- **Bottom panel (larger):** Chat with orchestrator
- **Sidebar:** Click project = orchestrator chat, click agent = transcript

### Agent Transcript View (click agent)

Shows parsed, nicely formatted output:
- Reasoning as collapsed gray text
- Commands as code blocks with output
- Messages as bubbles

## Event Routing

| Event | Routes to... |
|-------|--------------|
| `@mention` of specific agent | Daemon auto-wakes that agent |
| Agent `complete` | Daemon notifies orchestrator |
| Agent `failed` (crash/timeout) | Daemon notifies orchestrator with error details |
| Agent `ask` (blocked) | Orchestrator decides: answer, escalate, or spawn specialist |
| `@human` | Human via TUI notification |
| `@otto` | Orchestrator |

### Agent Failure Handling

When an agent crashes, times out, or exits with an error, the **daemon detects it** (not the orchestrator). The daemon then notifies the orchestrator with details:

```
@otto: @impl-1 failed (exit code 1, error: process killed after 10min timeout)
```

The orchestrator then decides what to do:
- Retry with the same agent
- Spawn a new agent
- Escalate to human
- Mark the task as blocked and move on

This keeps reliable detection in the daemon (code) and smart decisions in the orchestrator (LLM).

## CLI Commands

### Core Commands

```bash
# Prompt orchestrator (default) or specific agent
otto prompt "build a login page"              # → orchestrator
otto prompt impl-1 "try a different approach"  # → specific agent

# Agent communication
otto say "status update"                       # Post to channel
otto say "@otto I need help"                   # Mention triggers wake-up
otto ask "what should I do next?"              # Block until answered
otto complete "finished the task"              # Mark agent complete

# Spawning
otto spawn codex "implement feature X"         # Spawn agent (orchestrator only)
otto spawn codex "task" --name impl-1          # With custom name

# Management
otto kill impl-1                               # Terminate agent
otto kill impl-1 -f                            # Force kill (SIGKILL)
otto status                                    # Show all agents
otto messages                                  # Show message stream
```

### Agent Lifecycle

```
spawn → busy → blocked (via ask) → busy (via answer) → complete
                  ↓
               failed (process dies)
                  ↓
              archived (via archive command)
```

## TODO.md vs Tasks

Two distinct concepts:

| Concept | TODO.md | tasks table |
|---------|---------|-------------|
| **Purpose** | What work exists (backlog) | What's happening now (execution) |
| **Maintained by** | Human | Orchestrator/agents |
| **Storage** | File, version-controlled | SQLite, ephemeral |
| **Lifespan** | Long-lived, per-branch | Per-execution |
| **Example** | "Add authentication feature" | "Task 2: Add login endpoint" |

**The flow:**
1. Human picks item from TODO.md: "Add authentication"
2. Orchestrator brainstorms → creates design doc
3. Orchestrator writes plan → creates task breakdown in `tasks` table
4. Orchestrator executes → updates task status via agent assignments
5. Orchestrator finishes → human updates TODO.md

TODO.md is the durable record. Tasks table is ephemeral execution state.

## V0 Implementation Scope

### What V0 Includes

1. **Tasks table** - Hierarchical tree of tasks (root = workflow, children = phases/items)
2. **Event detection** - Parse messages for @mentions, detect completions and failures
3. **Auto-wake** - On event, call `otto prompt "[context]"` (defaults to orchestrator)
4. **Failure detection** - Daemon monitors agent processes, notifies orchestrator on crash/timeout
5. **Context builder** - Assemble: state summary (from tasks DB) + new messages
6. **Compaction detection** - Parse `context_compacted` from JSON stream
7. **Skill re-injection** - On compaction, re-inject full orchestrator skills + state from DB
8. **Codex output parsing** - Store structured data, display nicely in TUI

### What V0 Does NOT Include

- Hard gates on flow transitions (add later if soft reminders fail)
- Claude as orchestrator (add later with gates)
- Multiple root tasks per branch (V0 uses single root task per branch)
- Artifact checks by daemon (orchestrator handles)
- External skill resolution (hardcode superpowers flow for V0)

## Data Model

### Core Concept: Tasks as a Tree

Everything is a **task**. A "workflow" is just a root-level task:

```
"Build auth system"           (parent_id: NULL)  ← root task = "workflow"
├── "brainstorming"           (parent_id: above)
├── "planning"                (parent_id: above)
├── "implementation"          (parent_id: above)
│   ├── "Add login form"      (parent_id: implementation)
│   └── "Add password reset"  (parent_id: implementation)
├── "review"                  (parent_id: above)
└── "finishing"               (parent_id: above)
```

This simplifies the model:
- **No separate workflows table** - a workflow is just a root task
- **Workflow state** survives agent crashes (lives in DB, not agent context)
- **After compaction**, we query tasks to reconstruct state

### Schema

```sql
-- Agents: spawned Claude/Codex instances
-- Identity is (project, branch, name) - matches addressing format @project:branch:agent
CREATE TABLE agents (
    project TEXT NOT NULL,        -- 'backend-api', 'frontend-app'
    branch TEXT NOT NULL,         -- 'main', 'feature-login'
    name TEXT NOT NULL,           -- 'impl-1', 'otto', 'reviewer'
    type TEXT NOT NULL,           -- 'claude', 'codex'
    status TEXT DEFAULT 'idle',   -- idle, busy, blocked, complete, failed
    session_id TEXT,              -- Codex thread_id or Claude session
    compacted_at TIMESTAMP,       -- When last compaction detected
    last_seen_message_id TEXT,    -- For "inject only new messages" on wake-up
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (project, branch, name)
);
-- Note: Agent's current task comes from tasks.assigned_agent, not stored here

-- Messages: agent-to-agent and human-to-agent communication
CREATE TABLE messages (
    id TEXT PRIMARY KEY,          -- Message ID (UUID)
    project TEXT NOT NULL,
    branch TEXT NOT NULL,
    from_agent TEXT,              -- Agent name, or NULL for human messages
    content TEXT NOT NULL,
    mentions TEXT,                -- JSON array of resolved @mentions
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tasks: hierarchical tree (root tasks are "workflows")
CREATE TABLE tasks (
    project TEXT NOT NULL,
    branch TEXT NOT NULL,
    id TEXT NOT NULL,             -- Task ID within project/branch
    parent_id TEXT,               -- NULL for root tasks (workflows)
    name TEXT NOT NULL,           -- 'Build auth system', 'brainstorming', etc.
    sort_index INTEGER NOT NULL DEFAULT 0,  -- For deterministic ordering
    assigned_agent TEXT,          -- Agent name working on this (nullable)
    result TEXT,                  -- NULL = not done, 'completed', 'skipped'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (project, branch, id)
);

-- Note: logs table already exists (see db.go). V0 adds `type TEXT` column for event parsing.

-- Indexes for fast filtering
CREATE INDEX idx_agents_project_branch ON agents(project, branch);
CREATE INDEX idx_messages_project_branch ON messages(project, branch);
CREATE INDEX idx_tasks_project_branch ON tasks(project, branch);
CREATE INDEX idx_tasks_parent ON tasks(parent_id);
```

**Note:** No foreign keys between tables. Project/branch are plain text columns, not FKs to a projects table. This keeps things simple - if a project is renamed, a simple UPDATE fixes all references.

### Derived State

**Root tasks** = `SELECT * FROM tasks WHERE project = ? AND branch = ? AND parent_id IS NULL ORDER BY sort_index` (these are "workflows")

**Current phase** = first child of root task that's not completed (result IS NULL)

**Task status** = derived from `result` + `assigned_agent` + agent join:
```sql
SELECT t.*,
  CASE
    WHEN t.result = 'completed' THEN 'completed'
    WHEN t.result = 'skipped' THEN 'skipped'
    WHEN a.status = 'blocked' THEN 'blocked'
    WHEN a.status = 'failed' THEN 'failed'
    WHEN a.name IS NOT NULL AND a.status != 'idle' THEN 'active'
    ELSE 'pending'
  END as derived_status
FROM tasks t
LEFT JOIN agents a ON t.assigned_agent = a.name
  AND t.project = a.project AND t.branch = a.branch
```

All queries filter by project+branch. The indexes make this fast.

### V0 Pattern

V0 assumes one root task per project+branch:
- `GetOrCreateRootTask(project, branch, name)` returns the single root task
- Child tasks are phases, grandchildren are implementation items
- Later, we can support multiple root tasks per project+branch for parallel workstreams

## Open Questions for V1

1. **Claude orchestrator support** - Add hard gates when Claude is orchestrator?
2. **Multiple workflows per branch** - UI for managing parallel workstreams
3. **Artifact verification** - Should daemon verify artifacts before advancing?
4. **Cross-project coordination patterns** - How should orchestrators in different repos coordinate? (Addressing is solved, but patterns for handoff, status checking, failure propagation need design)
5. **Agent preferences config** - Which agent type for which task type (e.g., always use Claude for brainstorming)
6. **Harness-driven flow** - If soft reminders fail, add YAML-defined flow with hard gates
7. **Project metadata** - Should we add a projects table for settings, paths, and other metadata?
