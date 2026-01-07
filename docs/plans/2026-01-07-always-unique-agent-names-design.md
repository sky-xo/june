# Always-Unique Agent Names

## Problem

Current naming has friction:
1. User-provided names can collide, causing errors
2. Auto-generated names (`task-abc123`) aren't memorable
3. Callers can guess names, leading to accidental reuse

## Solution

Always append a 4-character suffix derived from the Codex ULID to every agent name.

### Naming Pattern

`<prefix>-<suffix>`

- **Suffix:** Last 4 characters of the Codex ULID (lowercase)
- **Prefix:** User-provided via `--name`, or auto-generated adjective-noun

### Examples

| User provides | Generated name |
|---------------|----------------|
| `--name refactor` | `refactor-9c4f` |
| `--name fix-auth` | `fix-auth-a3b2` |
| *(nothing)* | `swift-falcon-7d1e` |

### Collision Handling

1. Build name from prefix + last 4 chars of ULID
2. Check DB for collision (astronomically rare: ~1 in 1M per prefix)
3. If collision: replace suffix with 4 random hex bytes, retry up to 10x
4. Store final name

### Auto-Generated Prefixes

When no `--name` provided, generate adjective-noun combo:
- ~50 adjectives: `swift`, `quiet`, `bold`, `clever`, `bright`...
- ~50 nouns: `falcon`, `river`, `spark`, `stone`, `wave`...
- 2,500 combinations × 1M suffixes = effectively infinite

### Flow Change

```
Before:                          After:
1. Resolve name                  1. Get prefix (user or adjective-noun)
2. Spawn Codex                   2. Spawn Codex
3. Get ULID                      3. Get ULID
4. Store in DB                   4. Build name: prefix + ulid[-4:]
                                 5. Check collision, retry if needed
                                 6. Store in DB
```

## Guarantees

- Same prefix twice → different names (different ULIDs)
- No user-facing collision errors
- Caller must capture returned name — can't guess it
- Names look good in TUI: `swift-falcon-7d1e`, `refactor-9c4f`

## Files to Modify

1. `internal/cli/spawn.go` - New flow, name built after ULID received
2. `internal/cli/names.go` (new) - Word lists, `generateAdjectiveNoun()`, `buildAgentName()`
3. `internal/cli/spawn_test.go` - Update tests for new flow

## Non-Changes

- DB schema unchanged (still stores `name` and `ulid`)
- TUI unchanged (just displays names)
- `peek`, `logs` commands unchanged (lookup by name)
- Flag stays `--name` (not `--label`)
