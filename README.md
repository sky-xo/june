# Otto

A multi-agent orchestrator for Claude Code and Codex.

Otto lets you spawn multiple AI agents that work in parallel, communicate through a shared message stream, and coordinate handoffs—all from a single Claude Code session.

## Features

- **Spawn agents**: Launch Claude Code or Codex agents for parallel tasks
- **Shared messaging**: Agents communicate via @mentions in a shared channel
- **Persistent state**: SQLite-backed, survives session restarts
- **Real-time monitoring**: Watch message stream with TUI or simple polling
- **Orchestrator control**: Route questions, coordinate handoffs, escalate to human

## Installation

```bash
go install github.com/youruser/otto@latest
```

Or build from source:

```bash
git clone https://github.com/youruser/otto
cd otto
make build
```

## Quick Start

```bash
# Spawn an agent
otto spawn claude "implement user authentication"

# Check agent status
otto status

# View messages
otto messages

# Watch in real-time (TUI mode)
otto watch --ui
```

## Commands

| Command | Description |
|---------|-------------|
| `otto spawn <type> "<task>"` | Spawn a new claude/codex agent |
| `otto status` | List agents and their states |
| `otto messages` | View message stream |
| `otto watch [--ui]` | Monitor messages in real-time |
| `otto prompt <agent> "<msg>"` | Send prompt to an agent |
| `otto attach <agent>` | Get command to attach to agent session |
| `otto say "<msg>"` | Post message as orchestrator |
| `otto ask --id <agent> "<q>"` | Agent asks a question |
| `otto complete --id <agent> "<summary>"` | Agent marks task done |

## How It Works

```
You ←→ Claude Code (orchestrator)
         │
         │ calls otto CLI
         ▼
    ┌─────────────────────────────────────┐
    │  otto CLI                           │
    │  - spawn agents                     │
    │  - check status                     │
    │  - send/receive messages            │
    └─────────────────────────────────────┘
         │
         ├──────────────┬──────────────┐
         ▼              ▼              ▼
    Claude Code     Codex         Claude Code
    (design)      (implement)     (review)
```

State is stored in `~/.otto/orchestrators/<project>/<branch>/otto.db`.

## Development

```bash
make build    # Build binary
make test     # Run tests
make watch    # Build and run TUI
```

## License

MIT
