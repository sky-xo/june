# June

A read-only subagent viewer for Claude Code.

When you use Claude Code's Task tool to spawn subagents, they run in the background and their activity can be hard to track. June provides a terminal UI to watch what your subagents are doing in real-time.

## What It Shows

- List of all subagents spawned in your current project
- Real-time transcript of each agent's conversation
- Activity indicator (active vs. completed based on file modification time)
- Tool calls, results, and assistant responses

## Installation

Install with Go:

```bash
go install github.com/sky-xo/june@latest
```

Or build from source:

```bash
git clone https://github.com/sky-xo/june
cd june
make build
```

## Usage

Run `june` from any git repository where you've used Claude Code:

```bash
june
```

The TUI will launch showing any subagents that have been spawned in that project.

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down in agent list |
| `u` / `d` | Page up/down in transcript |
| `Tab` | Switch focus between panels |
| `q` | Quit |

## How It Works

Claude Code stores subagent transcripts at:

```
~/.claude/projects/{project-path}/agent-{id}.jsonl
```

June watches these files and displays their contents in a friendly TUI. It's completely read-only and does not modify any files.

## Requirements

- Must be run from within a git repository
- Requires a terminal (no headless mode)
- Claude Code must have been used in the project (creates the session files)

## Development

```bash
make build    # Build the binary
make test     # Run tests
./june        # Launch TUI
```

## License

MIT
