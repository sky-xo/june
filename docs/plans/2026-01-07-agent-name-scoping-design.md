# Agent Naming: Error on Collision with Output

## Problem

Agent names are globally unique, but the current behavior is suboptimal:
1. On collision: unclear error message
2. On success: silent (no confirmation of what was created)

## Design

### Keep Global Uniqueness

Names remain globally unique. No scoping to repo/branch - it adds complexity without enough benefit. The TUI already shows branch context visually.

### Error on Collision

When a name already exists, error with a helpful suggestion:

```
Error: agent "bugfix" already exists (spawned 2 hours ago)
Hint: use --name bugfix-2 or another unique name
```

This forces conscious choice of name, preventing accidental reference to old agents.

### Output Name on Success

Print the assigned name to stdout:

```bash
$ june spawn codex "fix auth" --name fix-auth
fix-auth
```

Confirms what was created. Useful for scripting and for Claude to capture the name.

### Exact Match for peek/logs

`june peek bugfix` finds only "bugfix", not "bugfix-2". Simple and predictable.

## Summary

| Aspect | Current | New |
|--------|---------|-----|
| Uniqueness scope | Global | Global (no change) |
| On collision | Unclear error | Helpful error with suggestion |
| Success output | Silent | Print assigned name |
| peek/logs lookup | Exact match | Exact match (no change) |

## Implementation

1. Update `spawn.go` collision error to include timestamp and suggestion
2. Add `fmt.Println(name)` on successful spawn
3. No database schema changes needed
