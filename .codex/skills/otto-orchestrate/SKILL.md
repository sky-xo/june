---
name: otto-orchestrate
description: Use when spawning or coordinating otto subagents to delegate work, isolate context, or run parallel tasks.
---

# Otto Orchestration

## Overview

Otto spawns and monitors subagents. The core principle is strict task isolation: give each agent only the context it needs and nothing more.

## When to Use

- You need parallel workers for independent tasks.
- You want to isolate context (e.g., reviews vs implementation).
- You need detached execution and later polling.

Do not use for quick inline questions or tightly coupled edits.

## Scope Control (Critical)

**Do NOT attach full plan files.** Paste only the Task N text into the prompt.

**Always include explicit guardrails:**
- “Do not read or act on other tasks.”
- “Stop after Task N and report.”

**Attach only required files.** If a file is not directly needed, omit it.

If the agent asks for more context, answer directly and re-dispatch with the minimal additions.

## Spawn an Agent

```bash
otto spawn codex "your task description" --detach
# Returns: agent-id
```

Options:
- `--name <name>` - Custom ID (e.g., `--name reviewer`)
- `--files <paths>` - Attach relevant files (keep minimal)
- `--context <text>` - Extra context (short, task-scoped)

## Example (Task-Scoped Prompt)

```bash
cat <<'EOF' | otto spawn codex --name task-1 --detach --files "path/to/file.go" ""
You are implementing Task 1: Add archived_at column.

Task text (ONLY Task 1):
[paste Task 1 text here]

Rules:
- Do not read or act on other tasks.
- Stop after Task 1 and report.
EOF
```

## Check Status

```bash
otto status
```

Shows all agents: `busy`, `complete`, `failed`, or `waiting`.

## Read Output

```bash
otto peek <agent-id>    # New output since last peek (advances cursor)
otto log <agent-id>     # Full history
otto log <agent-id> --tail 20   # Last 20 entries
```

Use `peek` for polling. Use `log` to review history.

## Send Follow-up

```bash
otto prompt <agent-id> "your message"
```

Use when an agent finishes and you need more work, or to answer a `waiting` agent.

## Typical Flow

```bash
# 1. Spawn
otto spawn codex "implement feature X" --name feature-x --detach

# 2. Poll until done
otto status                    # Check if still busy
otto peek feature-x            # Read new output

# 3. Follow up if needed
otto prompt feature-x "also add tests"
```

## Rationalizations Observed (Baseline)

| Observed behavior | Reality |
| --- | --- |
| “Updated retention cleanup to use archived_at…” | Out of scope when asked to only commit Task 1 changes. |

## Red Flags - Stop and Narrow Scope

- You are attaching the full plan file.
- The prompt mentions multiple tasks at once.
- The agent starts “helpfully” doing adjacent tasks.

## Quick Reference

| Action | Command |
| --- | --- |
| Spawn agent | `otto spawn codex "task" --detach` |
| Status | `otto status` |
| Read new output | `otto peek <agent-id>` |
| Full log | `otto log <agent-id>` |
| Follow-up | `otto prompt <agent-id> "message"` |

## Common Mistakes

- **Attaching the full plan file:** leads to task drift and scope creep.
- **Vague prompts:** “take a look around” invites extra work.
- **Missing stop rule:** agents continue beyond the assigned task.
