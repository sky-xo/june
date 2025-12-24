# Codex Orchestrator Instructions

Status: Ready for review

These instructions are for Codex agents acting as orchestrators. Unlike Claude Code, Codex can't handle background jobs well, so it uses `--detach` and polling patterns.

---

## Otto Multi-Agent Orchestration

Otto is a CLI for spawning and coordinating AI agents.

## Spawning Subagents

Use `--detach` to spawn agents without blocking:

```bash
otto spawn codex "your task description" --detach
```

This prints the agent ID and returns immediately. The agent runs in the background.

Optional flags:
- `--name <name>` - Custom agent name
- `--context <text>` - Additional context
- `--files <paths>` - Relevant files

## Checking Agent Status

```bash
otto status
```

Shows all agents with their status (busy, complete, failed, waiting).

## Reading Agent Output

**Peek** - Show unread logs and advance cursor (for polling):
```bash
otto peek <agent-id>
```

**Log** - Show full log history (doesn't advance cursor):
```bash
otto log <agent-id>
otto log <agent-id> --tail 20  # Last 20 entries
```

## Message Channel

All agents share a message stream. Check for messages:
```bash
otto messages
```

## Workflow Example

```bash
# Spawn a detached agent
otto spawn codex "implement feature X" --detach
# Returns: agent-id-123

# Check status
otto status

# Poll for new output
otto peek agent-id-123

# Or view full history
otto log agent-id-123 --tail 50
```

## Note on Detached Agents

Detached agents run in the background with full log capture. You can:
- Check stdout/stderr via `otto peek <agent-id>` or `otto log <agent-id>`
- View messages via `otto messages`
- Resume Codex agents with `otto prompt <agent-id> "follow-up message"`

Status automatically updates to `complete` or `failed` when the agent finishes.

---

## Future: Auto-Injection

Idea: Otto could auto-inject agent-specific instructions when spawning:
- Detect agent type (codex vs claude)
- Inject appropriate orchestration patterns
- Could use a `--instructions` flag or config file
