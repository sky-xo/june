# Otto Hypotheses

This document captures the core bets Otto is making. These are testable claims, not finished conclusions. As we validate or invalidate each hypothesis, we'll update this doc.

## Context

Otto exists in a space where vanilla Claude Code + superpowers skills + hooks is already "pretty good." The question is: does building a dedicated orchestration layer provide enough value to justify the complexity?

## Hypothesis 1: Flows as First-Class Citizens

**Claim:** Developers work in predefined flows (brainstorm → plan → implement → review → ship). Making flows first-class—tracked, configurable, persistent—is more valuable than ad-hoc skill invocation.

**What Otto provides:**

- Flow state tracked in SQLite, survives session restarts
- Each stage can have its own model/skill configuration via YAML
- The system knows "you're in implementation" and can inject appropriate context
- Cross-session continuity without manual state management

**What vanilla Claude lacks:**

- No persistent flow state across sessions
- Skills invoked ad-hoc, not tied to workflow stages
- Agents can drift from the intended workflow

**How to test:**

- Run agents through multi-step workflows
- Observe: Do agents stay on track? Does flow state help prevent drift?
- Compare to ad-hoc skill invocation

**Status:** Untested

---

## Hypothesis 2: Unified Multi-Project Control Plane

**Claim:** One TUI with visibility into all projects/branches/agents reduces context-switching overhead compared to managing multiple terminal tabs.

**What Otto provides:**

- Single pane of glass: see all agents across all projects
- Visual indication of which agents need attention
- Click to switch context, prompt, move on
- Know immediately when any agent completes

**The pain point this solves:**

- "Which of my Claude terminals is done and waiting for me?"
- Context-switching between 5+ terminal tabs to check agent status
- Missing completion notifications

**How to test:**

- Work on 3+ features across 2+ projects simultaneously
- Compare: time spent context-switching, missed notifications, cognitive load

**Status:** Untested

---

## Hypothesis 3: Model-Aware Orchestration

**Claim:** Different models have different behavioral tendencies. A system that encodes this knowledge produces better outcomes than naive "just spawn an agent" approaches.

**Observed model tendencies:**

- **Codex:** Rigid rule-follower. Won't bend even when asked. Excellent for precise execution, struggles with ambiguity.
- **Claude:** More flexible, but can abandon rules unprompted. Tends to be agreeable/deferential in debates.

**What Otto can do:**

- Tailor prompts per model (give Codex permission to deviate; give Claude stronger guardrails)
- Assign roles strategically (Codex for implementation/review, Claude for brainstorming)
- Design balanced interactions (two Codex agents debating won't just agree)

**Example configuration:**

```yaml
stages:
  implementation:
    model: codex
    prompt_modifier: "strict execution, minimal deviation"

  brainstorming:
    model: claude
    prompt_modifier: "explore freely, challenge assumptions"

  review:
    model: codex
    prompt_modifier: "be critical, don't accept sloppy work"

  debate:
    models: [codex, codex]
    prompt_modifier: "defend your position, don't concede easily"
```

**How to test:**

- Run agent debates with different model combinations
- Track: Do model-specific prompts improve output quality?

**Status:** Untested

---

## Hypothesis 4: Codex as Orchestrator

**Claim:** Codex's behavioral characteristics (strict rule-following, certain code implementation strengths) might make it an interesting orchestrator—worth experimenting to see what happens.

**Why this matters:**

- Claude Code has hooks, Task tool, can spawn subagents natively
- Codex has none of these—Otto provides them
- Codex's rigidity might actually be a _feature_ for orchestration (consistent, predictable coordination)

**Open questions:**

- Does Codex's rule-following make orchestration more reliable?
- Does its rigidity become a problem when workflows need adaptation?
- How does it compare to Claude for different types of coordination tasks?
- What unexpected behaviors emerge when Codex runs the show?

**This is purely exploratory:**
The interest is in understanding what happens when a more rigid, rule-following model takes the orchestrator role. It might be better for some workflows, worse for others, or reveal entirely unexpected patterns.

**How to test:**

- Run various workflows with Codex as orchestrator
- Observe: Where does rigidity help? Where does it hurt?
- Compare qualitatively to Claude orchestrator experiences

**Status:** Untested

---

## Hypothesis 5: Agent-to-Agent Communication

**Claim:** Direct agent-to-agent communication (peer-to-peer) may produce better outcomes than pure hub-and-spoke (all agents report to orchestrator) for certain tasks.

**Hub-and-spoke:**

- Orchestrator synthesizes all inputs
- Single point of control and context
- User preferences flow through orchestrator

**Peer-to-peer:**

- Agents can debate directly without bottleneck
- Richer technical back-and-forth
- Agents can challenge each other's assumptions

**Example use case:**
Two agents each create an implementation plan, then discuss directly to converge on the best approach.

**Risks:**

- Peer conversations can diverge or go in circles
- Agents might "agree to agree" on suboptimal solutions
- Less visibility for human

**How to test:**

- For a feature with multiple valid approaches, compare:
  - Option A: Two agents produce plans → Orchestrator picks/synthesizes
  - Option B: Two agents produce plans → Agents debate → Converge together
- Measure: quality of final plan, edge cases caught, time spent

**Status:** Untested (genuinely novel research question)

---

## Hypothesis 6: Resilient Subagent Execution

**Claim:** Otto's built-in task tracking and logging enable automatic recovery from subagent compaction—reducing a manual, multi-step recovery workflow to zero-config automatic continuation.

**The problem with vanilla Claude:**

- Subagents can compact mid-task, losing internal context
- Recovery requires manual intervention: `/export` → `/clear` → tell fresh Claude to read export "from the bottom up"
- Fresh Claude has to figure out where things left off by reading the exported conversation
- While you _could_ configure subagents to track progress (via beads, checkpoint files, etc.), it requires setup and discipline

**What Otto provides:**

- Task progress tracked automatically in SQLite—no setup required
- Agent activity logged persistently—survives compaction/crash
- Orchestrator can detect agent failure and auto-spawn replacement
- New agent receives: incomplete tasks + recent logs from dead agent
- Recovery is automatic, not a manual export/clear/re-read workflow

**The recovery mechanism:**

1. Agent works through tasks, marking complete as it goes (task list = checkpoint)
2. Otto logs agent activity to SQLite
3. Agent compacts/crashes
4. Orchestrator detects failure
5. Auto-spawns new agent with: remaining tasks + last N lines of logs
6. New agent checks codebase state, continues from where previous left off
7. Retry limits prevent infinite loops on persistently-failing tasks

**Key insight:**
This isn't "Otto can do something impossible." Vanilla Claude + careful configuration could achieve similar results. The value is: Otto makes it automatic and zero-config, whereas vanilla requires manual setup and manual intervention every time recovery is needed.

**How to test:**

- Intentionally trigger compaction during multi-task agent work
- Compare: vanilla recovery workflow vs Otto automatic recovery
- Measure: time to recovery, work lost, human intervention required

**Status:** Untested

---

## The Meta-Hypothesis

**Otto's overall bet:** An integrated, opinionated system for multi-agent development—with tracked flows, unified visibility, and model-aware orchestration—beats assembling parts (Claude Code + beads + superpowers) for developers doing complex, multi-project work.

**This bet wins if:**

- The conventions Otto encodes are actually good
- Multi-project/multi-agent workflows are common enough to justify the system
- The integration saves more time than vanilla composition costs

**This bet loses if:**

- Everyone's workflow is too different (conventions don't fit)
- Claude Code + plugins evolves faster than Otto can keep up
- The orchestration overhead exceeds the benefits

---

## Validation Plan

1. **Finish current TUI sprint** - Get basic visibility and control working
2. **Use Otto for a real external project feature** - Parallel Codex agents, review, ship
3. **Compare honestly** - Would vanilla Claude have been faster? What friction did Otto add or remove?
4. **Share with vibez community** - Get feedback from practitioners hitting the same problems
5. **Update this doc** - Mark hypotheses as validated, invalidated, or refined

---

## Log

| Date       | Update                                                                                                       |
| ---------- | ------------------------------------------------------------------------------------------------------------ |
| 2024-12-29 | Initial hypotheses documented based on project reflection                                                    |
| 2024-12-30 | Added Hypothesis 6: Resilient Subagent Execution (based on Vibes Coding Group feedback on compaction issues) |
