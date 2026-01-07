# Agent Name Scoping & Auto-increment

## Problem

Agent names are currently globally unique across all projects/branches. This causes unnecessary collisions when reusing common names like "bugfix" across different branches or over time on the same branch.

## Design

### Scoping

Names become unique per `(repo_path, branch)` instead of globally.

**Database change:**
- Change primary key from `name` to composite `(repo_path, branch, name)`
- `GetAgent(name)` becomes `GetAgent(repoPath, branch, name)`
- Same for collision checks in `spawn.go`

**Effect:** "bugfix" on `main` and "bugfix" on `feature-x` can coexist. Different branches = different namespaces.

### Auto-increment

When a name collision occurs within the same branch, automatically suffix with `-2`, `-3`, etc.

**Logic in `spawn.go`:**
1. Check if "bugfix" exists for (repo, branch)
2. If no → use "bugfix"
3. If yes → find highest existing suffix, use next
   - "bugfix" exists → try "bugfix-2"
   - "bugfix-2" exists → try "bugfix-3"

**Database:** Add `GetAgentsByPrefix(repoPath, branch, namePrefix)` to find all matching agents for suffix calculation.

### Output

On success, print just the assigned name to stdout:

```bash
$ june spawn codex "fix auth" --name bugfix
bugfix

$ june spawn codex "another fix" --name bugfix
bugfix-2
```

### CLI Changes

`peek` and `logs` commands scope lookups by current repo/branch. `june peek bugfix` finds "bugfix" for the current branch, not globally.

**Edge case:** Outside a git repo, repo_path and branch are empty strings. Names share the `("", "", name)` namespace - auto-increment still works.

## Summary

| Aspect | Current | New |
|--------|---------|-----|
| Uniqueness scope | Global | Per (repo, branch) |
| On collision | Error | Auto-increment (-2, -3) |
| Success output | Silent | Print assigned name |
| `--name` flag | Required | Required (no change) |
| `peek`/`logs` lookup | Global | Scoped to current repo/branch |
