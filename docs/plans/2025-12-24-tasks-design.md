# Tasks Design

**Status:** Draft
**Created:** 2025-12-24
**Related:** [Super Orchestrator Design](./2025-12-24-super-orchestrator-design.md), [Skill Injection Design](./2025-12-24-skill-injection-design.md)

## Overview

Design for tracking execution state in Otto. Separates project backlog (TODO.md) from execution state (tasks table).

## TODO.md vs Tasks Table

Two distinct concepts:

| Concept | TODO.md | tasks table |
|---------|---------|-------------|
| **Purpose** | What work exists (backlog) | What's happening now (execution) |
| **Maintained by** | Human | Orchestrator/agents |
| **Storage** | File, version-controlled | SQLite, ephemeral |
| **Lifespan** | Long-lived, per-branch | Per-execution |
| **Example** | "Add authentication feature" | "Task 2: Add login endpoint" |

**The flow:**
1. Human picks item from TODO.md: "Add authentication"
2. Orchestrator brainstorms → creates design doc
3. Orchestrator writes plan → creates task breakdown in `tasks` table
4. Orchestrator executes → updates task status via agent assignments
5. Orchestrator finishes → human updates TODO.md (or orchestrator can)

TODO.md is the durable record. Tasks table is ephemeral execution state.

## Tasks Schema

```sql
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY,
    parent_id INTEGER REFERENCES tasks(id),  -- NULL = top-level
    plan_file TEXT,              -- path to plan file (e.g., docs/plans/2025-12-24-auth.md)
    project TEXT,
    branch TEXT,
    content TEXT NOT NULL,
    assigned_agent TEXT,         -- FK to agents (NULL = unassigned)
    completed_at DATETIME,       -- NULL = not completed
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tasks_project_branch ON tasks(project, branch);
CREATE INDEX idx_tasks_parent ON tasks(parent_id);
CREATE INDEX idx_tasks_agent ON tasks(assigned_agent);
```

## Task State is Derived

No `status` field needed. State is derived from data:

| assigned_agent | completed_at | → Derived State |
|----------------|--------------|-----------------|
| NULL | NULL | pending |
| set | NULL | active (check agent for busy/blocked/failed) |
| set | set | completed |

### Query Examples

**Get all tasks for a project/branch:**
```sql
SELECT * FROM tasks
WHERE project = ? AND branch = ?
ORDER BY created_at;
```

**Get active tasks with agent status:**
```sql
SELECT t.*, a.status as agent_status
FROM tasks t
JOIN agents a ON t.assigned_agent = a.id
WHERE t.completed_at IS NULL
  AND t.assigned_agent IS NOT NULL;
```

**Get pending tasks (not assigned, not completed):**
```sql
SELECT * FROM tasks
WHERE assigned_agent IS NULL
  AND completed_at IS NULL;
```

**Get task tree for a plan:**
```sql
WITH RECURSIVE task_tree AS (
    SELECT *, 0 as depth FROM tasks WHERE plan_file = ? AND parent_id IS NULL
    UNION ALL
    SELECT t.*, tt.depth + 1
    FROM tasks t
    JOIN task_tree tt ON t.parent_id = tt.id
)
SELECT * FROM task_tree ORDER BY depth, created_at;
```

## Hierarchical Tasks

Tasks form a tree structure that encodes the skill flow:

```
[ ] Implement authentication (plan: docs/plans/2025-12-24-auth.md)
    [ ] Brainstorm design
    [ ] Set up worktree
    [ ] Write implementation plan
    [ ] Execute plan
        [x] Task 1: Add user model (@backend, completed)
        [ ] Task 2: Add login endpoint (@backend, active)
            [x] Implementation
            [ ] Spec review          <-- current position
            [ ] Code quality review
        [ ] Task 3: Add JWT middleware (pending)
    [ ] Finish branch
```

### Finding Current Position

"Where am I in the workflow?" = Find the first incomplete leaf task:

```sql
WITH RECURSIVE task_tree AS (
    SELECT *, 0 as depth FROM tasks WHERE plan_file = ? AND parent_id IS NULL
    UNION ALL
    SELECT t.*, tt.depth + 1
    FROM tasks t
    JOIN task_tree tt ON t.parent_id = tt.id
)
SELECT * FROM task_tree
WHERE completed_at IS NULL
  AND id NOT IN (SELECT parent_id FROM tasks WHERE parent_id IS NOT NULL)
ORDER BY depth, created_at
LIMIT 1;
```

This returns the first incomplete task that has no children (leaf node).

## Task Operations

### Creating Tasks from Plan

When orchestrator writes a plan, it creates the task hierarchy:

```go
func CreateTasksFromPlan(db *sql.DB, planFile string, project, branch string) error {
    // Parse plan file
    // Create top-level task for the feature
    // Create child tasks for each step in the plan
}
```

### Assigning Task to Agent

```go
func AssignTask(db *sql.DB, taskID int, agentID string) error {
    _, err := db.Exec(`
        UPDATE tasks
        SET assigned_agent = ?, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `, agentID, taskID)
    return err
}
```

### Completing a Task

```go
func CompleteTask(db *sql.DB, taskID int) error {
    _, err := db.Exec(`
        UPDATE tasks
        SET completed_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `, taskID)
    return err
}
```

### Adding Subtasks

Agent breaks down their assigned task into subtasks:

```go
func AddSubtask(db *sql.DB, parentID int, content string) (int, error) {
    result, err := db.Exec(`
        INSERT INTO tasks (parent_id, content, project, branch, plan_file)
        SELECT ?, ?, project, branch, plan_file FROM tasks WHERE id = ?
    `, parentID, content, parentID)
    // return new task ID
}
```

## Integration with Agents

Task state is linked to agent state:

- When agent is assigned a task → task becomes "active"
- When agent status is "blocked" → task is effectively blocked
- When agent status is "failed" → task may need reassignment
- When agent completes → task should be marked completed

The orchestrator watches for agent status changes and updates task state accordingly.

## Rendering Tasks for Injection

When waking up an orchestrator, render tasks for context injection:

```go
func RenderTasksForInjection(db *sql.DB, planFile string) string {
    // Get task tree
    // Format as markdown checkbox list
    // Mark current position
    // Include agent status for active tasks
}
```

Output:
```markdown
## Current Tasks
- [x] Task 1: Add user model (@backend, completed)
- [ ] Task 2: Add login endpoint (@backend, in spec review)  <-- current
- [ ] Task 3: Add JWT middleware (pending)
```

## Open Questions

1. **Task cleanup** - When is the tasks table cleaned up? After branch merge? Manual archive?

2. **Task history** - Do we need to keep completed tasks for audit/debugging, or can they be purged?

3. **Cross-agent tasks** - Can a task be reassigned to a different agent mid-execution?
