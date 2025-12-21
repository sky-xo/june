# Otto: Topics for Further Discussion

This file captures topics we haven't fully resolved yet, for continuation in future sessions.

## Open Topics

### 1. `otto handoff` Command

**Context:** When one agent finishes and needs to pass work to another (e.g., debug agent → fix agent).

**Current approach (works):**
1. Agent posts findings to #main channel
2. Orchestrator reads them
3. Orchestrator spawns next agent with `--context "findings..."`

**Proposed alternative:**
```bash
otto handoff --agent agent-debug --to codex "Fix the leak" --context "Root cause: ..."
```

**Tradeoff:** Convenience vs. orchestrator visibility. Handoff bypasses orchestrator, current approach keeps orchestrator in the loop.

**Decision needed:** Add this for v0, or defer?

---

### 2. Channels Within Orchestrators (v2?)

**Context:** What if you want multiple work streams in one branch?

**Current design:** One orchestrator per project/branch. Multiple work streams = multiple branches.

**Alternative:** Channels within an orchestrator (like Slack channels):
```
Orchestrator: my-app/main
├── #auth-feature (agents working on auth)
├── #bugfix-123 (agents fixing bug)
└── #main (general)
```

**Current thinking:** Skip for v0. Branch-per-task is simpler. Revisit if it feels too heavy in practice.

---

### 3. Message Filtering/Pagination

**Context:** After hours of work, message history could be huge.

**Proposed solution:**
```bash
otto messages              # unread only (default)
otto messages --all        # everything
otto messages --last 20    # recent
otto messages --from agent-abc  # from specific agent
```

**Status:** Not yet added to design doc. Probably add for v0.

---

### 4. Package Name

**Options:**
- `otto` - clean but might conflict
- `otto-agent` - descriptive
- `otto-cli` - explicit about what it is

**Decision needed:** Check npm for conflicts, pick one.

---

## Key Decisions Already Made

These are resolved - documenting for context continuity:

1. **Architecture:** CLI tool called via Bash, not MCP server. Simpler installation.

2. **Storage:** SQLite in `~/.otto/orchestrators/<project>/<branch>/otto.db`

3. **Messaging:** Group chat model with #main channel. @mentions for attention. DMs available but discouraged.

4. **Agent identification:** Explicit `--agent <id>` flag on all commands (env vars don't persist between tool calls).

5. **Worktrees:** `--worktree <name>` flag creates isolated workspace in `.worktrees/`. Follows superpowers conventions.

6. **Orchestrator scoping:** Auto-detect from project dir + git branch. Override with `--in <name>`.

7. **Ephemeral orchestrator model:** Conversations are disposable, state lives in otto.db and plan documents.

8. **Compatibility:** Works inside packnplay containers with no modifications. Complements superpowers skills.

9. **Session resume:** Both Claude Code (`--resume`) and Codex (`codex resume`) support session resume. Otto tracks session IDs.

---

## Superpowers Integration

Otto works with superpowers skills:

| Agent Role | Skills Used |
|------------|-------------|
| Orchestrator | brainstorming, writing-plans, dispatching-parallel-agents |
| Implementation agent | executing-plans, test-driven-development |
| Debug agent | systematic-debugging |
| Review agent | requesting-code-review |
| All agents | verification-before-completion |

The orchestrator uses `using-git-worktrees` when setting up agent workspaces.

---

## Files in This Project

- `docs/design.md` - Main design document (comprehensive)
- `docs/scenarios.md` - 5 usage scenarios testing the API
- `docs/discuss.md` - This file (open topics)

---

## Next Steps for Implementation

Once discussion is complete:

1. Initialize npm package
2. Implement Phase 1 MVP (see design.md)
3. Test against scenarios
4. Iterate based on real usage
