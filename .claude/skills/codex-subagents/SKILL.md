---
name: codex-subagents
description: Use when user explicitly asks to spawn a Codex agent, e.g. "spin up a codex agent", "get codex to...", "use codex for...", "spawn codex"
---

# Codex Subagents

Spawn and monitor Codex agents using June.

## Commands

```bash
june spawn codex "task description" --name <name>   # Spawn agent
june peek <name>                                     # New output since last peek
june logs <name>                                     # Full transcript
```

## Usage

1. Spawn with a descriptive `--name`
2. Use `peek` to check for new output (advances cursor)
3. Use `logs` to review full transcript (doesn't advance cursor)

## Notes

- Names must be unique (state stored in `~/.june/june.db`)
- Agents inherit current git repo/branch context
