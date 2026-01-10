# June Plugin Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Transform June into a Claude Code plugin with task-aware workflow skills.

**Architecture:** Add plugin structure (.claude-plugin/), create june-skills/ for custom skills, add Makefile targets to vendor superpowers and overlay customizations. Modify skills to use `june task` commands instead of TodoWrite.

**Tech Stack:** Go (CLI enhancement), Makefile (build process), Markdown (skills)

---

### Task 1: Add --note flag to task create

**Files:**
- Modify: `internal/cli/task.go:31-48`
- Modify: `internal/cli/task.go:95-160` (runTaskCreate function)
- Test: `internal/cli/task_test.go`

**Step 1: Write the failing test**

Add to `internal/cli/task_test.go`:

```go
func TestTaskCreateWithNote(t *testing.T) {
	// Setup temp DB
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create .june directory
	juneDir := filepath.Join(tmpDir, ".june")
	if err := os.MkdirAll(juneDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Need git repo for scope
	repoDir := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}
	origWd, _ := os.Getwd()
	os.Chdir(repoDir)
	defer os.Chdir(origWd)

	exec.Command("git", "init").Run()
	exec.Command("git", "checkout", "-b", "main").Run()

	cmd := newTaskCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"create", "Test task", "--note", "This is a note", "--json"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Parse JSON output to get task ID
	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify note was set by listing the task
	out.Reset()
	cmd = newTaskCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"list", result.ID, "--json"})
	cmd.Execute()

	// Check output contains the note
	if !bytes.Contains(out.Bytes(), []byte("This is a note")) {
		t.Errorf("Expected note in output, got: %s", out.String())
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run TestTaskCreateWithNote -v`
Expected: FAIL (--note flag not recognized)

**Step 3: Add --note flag to newTaskCreateCmd**

In `internal/cli/task.go`, modify `newTaskCreateCmd()`:

```go
func newTaskCreateCmd() *cobra.Command {
	var parentID string
	var note string
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "create <title> [titles...]",
		Short: "Create one or more tasks",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskCreate(cmd, args, parentID, note, outputJSON)
		},
	}

	cmd.Flags().StringVar(&parentID, "parent", "", "Parent task ID for creating child tasks")
	cmd.Flags().StringVar(&note, "note", "", "Set note on created task(s)")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output in JSON format")

	return cmd
}
```

**Step 4: Update runTaskCreate signature and implementation**

Modify `runTaskCreate` to accept and use the note:

```go
func runTaskCreate(cmd *cobra.Command, args []string, parentID, note string, outputJSON bool) error {
	// ... existing setup code ...

	for _, title := range args {
		id, err := generateUniqueTaskID(database)
		if err != nil {
			return fmt.Errorf("generate task ID: %w", err)
		}

		var notePtr *string
		if note != "" {
			notePtr = &note
		}

		task := db.Task{
			ID:        id,
			ParentID:  parentPtr,
			Title:     title,
			Status:    "open",
			Notes:     notePtr,
			RepoPath:  repoPath,
			Branch:    branch,
			CreatedAt: now,
			UpdatedAt: now,
		}
		// ... rest of function ...
	}
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./internal/cli -run TestTaskCreateWithNote -v`
Expected: PASS

**Step 6: Run all tests**

Run: `make test`
Expected: All tests pass

**Step 7: Commit**

```bash
git add internal/cli/task.go internal/cli/task_test.go
git commit -m "feat(cli): add --note flag to task create"
```

---

### Task 2: Create plugin infrastructure

**Files:**
- Create: `.claude-plugin/plugin.json`
- Modify: `.gitignore`
- Modify: `Makefile`

**Step 1: Create plugin.json**

Create `.claude-plugin/plugin.json`:

```json
{
  "name": "june",
  "description": "Task-aware workflow skills with persistent state",
  "version": "1.0.0"
}
```

**Step 2: Update .gitignore**

Add to `.gitignore`:

```
.skill-cache/
```

**Step 3: Add build-skills target to Makefile**

Add to `Makefile`:

```makefile
# Superpowers vendoring
SUPERPOWERS_VERSION := v4.0.3
SUPERPOWERS_REPO := https://github.com/obra/superpowers

.PHONY: build-skills
build-skills:
	@# Fetch superpowers if not cached
	@[ -d .skill-cache/superpowers ] || git clone $(SUPERPOWERS_REPO) .skill-cache/superpowers
	@cd .skill-cache/superpowers && git fetch && git checkout $(SUPERPOWERS_VERSION)
	@# Clean and copy superpowers skills
	rm -rf skills/
	cp -r .skill-cache/superpowers/skills skills/
	@# Overlay June's custom skills (override)
	cp -r june-skills/* skills/
	@echo "Skills assembled: superpowers $(SUPERPOWERS_VERSION) + june overrides"

.PHONY: update-superpowers
update-superpowers:
	cd .skill-cache/superpowers && git fetch origin main && git log --oneline HEAD..origin/main
	@echo "Review changes above, then update SUPERPOWERS_VERSION and run 'make build-skills'"
```

**Step 4: Verify plugin.json is valid JSON**

Run: `cat .claude-plugin/plugin.json | jq .`
Expected: Valid JSON output

**Step 5: Commit**

```bash
git add .claude-plugin/plugin.json .gitignore Makefile
git commit -m "feat: add plugin infrastructure for Claude Code"
```

---

### Task 3: Create june-skills directory with custom skills

**Files:**
- Create: `june-skills/writing-plans/SKILL.md`
- Create: `june-skills/subagent-driven-development/SKILL.md`
- Create: `june-skills/fresheyes/SKILL.md`
- Create: `june-skills/fresheyes/fresheyes-full.md`
- Create: `june-skills/fresheyes/fresheyes-quick.md`
- Create: `june-skills/design-review/SKILL.md`
- Create: `june-skills/review-pr-comments/SKILL.md`
- Create: `june-skills/tool-scout/SKILL.md`
- Create: `june-skills/webresearcher/SKILL.md`
- Create: `june-skills/executing-plans/SKILL.md`

**Step 1: Create june-skills directory structure**

```bash
mkdir -p june-skills/{writing-plans,subagent-driven-development,fresheyes,design-review,review-pr-comments,tool-scout,webresearcher,executing-plans}
```

**Step 2: Copy existing custom skills**

Copy from user's `~/.claude/skills/`:

```bash
cp ~/.claude/skills/writing-plans/SKILL.md june-skills/writing-plans/
cp ~/.claude/skills/subagent-driven-development/SKILL.md june-skills/subagent-driven-development/
cp ~/.claude/skills/design-review/SKILL.md june-skills/design-review/
cp ~/.claude/skills/review-pr-comments/SKILL.md june-skills/review-pr-comments/
cp ~/.claude/skills/tool-scout/SKILL.md june-skills/tool-scout/
cp ~/.claude/skills/webresearcher/SKILL.md june-skills/webresearcher/
```

**Step 3: Copy fresheyes skill (from separate repo)**

```bash
cp /Users/glowy/code/fresheyes/skills/fresheyes/SKILL.md june-skills/fresheyes/
cp /Users/glowy/code/fresheyes/skills/fresheyes/fresheyes-full.md june-skills/fresheyes/
cp /Users/glowy/code/fresheyes/skills/fresheyes/fresheyes-quick.md june-skills/fresheyes/
```

**Step 4: Copy executing-plans from superpowers as base**

```bash
cp ~/.claude/plugins/cache/superpowers-marketplace/superpowers/4.0.3/skills/executing-plans/SKILL.md june-skills/executing-plans/
```

**Step 5: Verify all skills copied**

Run: `find june-skills -name "*.md" | wc -l`
Expected: At least 10 files

**Step 6: Commit**

```bash
git add june-skills/
git commit -m "feat: add june-skills directory with custom skills"
```

---

### Task 4: Modify writing-plans to use june tasks

**Files:**
- Modify: `june-skills/writing-plans/SKILL.md`

**Step 1: Read current writing-plans skill**

Review the skill to understand current structure.

**Step 2: Add task creation section**

Add after "Save plans to" section, before "Execution Handoff":

```markdown
## Task Persistence

After saving the plan, create June tasks for tracking:

```bash
# Create parent task for the plan
june task create "<Plan Title>" --json
# Returns: {"id": "t-xxxx"}

# Create child tasks for each numbered task in the plan
june task create "Task 1: <title>" --parent t-xxxx
june task create "Task 2: <title>" --parent t-xxxx
# ... for each task
```

Output to user:

```
Plan saved to docs/plans/<filename>.md
Tasks created: t-xxxx (N children)

Run `june task list t-xxxx` to see task breakdown.
```
```

**Step 3: Update skill header comment**

Update the comment at top to indicate June integration:

```markdown
# Based on: superpowers v4.0.3
# Customization: Fresheyes review for plans 200+ lines, June task persistence
```

**Step 4: Commit**

```bash
git add june-skills/writing-plans/SKILL.md
git commit -m "feat(skills): add june task creation to writing-plans"
```

---

### Task 5: Modify executing-plans to use june tasks

**Files:**
- Modify: `june-skills/executing-plans/SKILL.md`

**Step 1: Read current executing-plans skill**

Review the skill structure.

**Step 2: Replace TodoWrite references with june task commands**

In Step 1, change:

```markdown
4. If no concerns: Create TodoWrite and proceed
```

to:

```markdown
4. If no concerns: Read tasks with `june task list <parent-id> --json` and proceed
```

**Step 3: Update Step 2 for task status**

Change task status updates:

```markdown
### Step 2: Execute Batch
**Default: First 3 tasks**

For each task:
1. Mark as in_progress: `june task update <task-id> --status in_progress`
2. Follow each step exactly (plan has bite-sized steps)
3. Run verifications as specified
4. Mark as completed: `june task update <task-id> --status closed --note "Verified"`
```

**Step 4: Add resume section**

Add new section after "The Process":

```markdown
## Resuming After Compaction

If context was compacted, check task state:

```bash
june task list <parent-id>
```

Find the first task with status `open` or `in_progress` and resume from there.
```

**Step 5: Add header comment**

```markdown
# Based on: superpowers v4.0.3
# Customization: June task persistence instead of TodoWrite
```

**Step 6: Commit**

```bash
git add june-skills/executing-plans/SKILL.md
git commit -m "feat(skills): add june task updates to executing-plans"
```

---

### Task 6: Modify subagent-driven-development to use june tasks

**Files:**
- Modify: `june-skills/subagent-driven-development/SKILL.md`

**Step 1: Read current skill**

Review the skill structure.

**Step 2: Replace TodoWrite with june task commands**

Change step 1 from:

```markdown
1. **Read plan, extract tasks, create TodoWrite**
```

to:

```markdown
1. **Read plan, check existing tasks or create new ones**
   - If parent task ID provided: `june task list <parent-id> --json`
   - If no tasks exist: Create from plan using `june task create`
```

**Step 3: Update per-task flow**

```markdown
2. **Per task:**
   - `june task update <task-id> --status in_progress`
   - Dispatch **implementer** (haiku or opus based on complexity)
   - ... (reviews as before) ...
   - `june task update <task-id> --status closed --note "Implemented, tests pass"`
```

**Step 4: Update header comment**

```markdown
# Based on: superpowers v4.0.3
# Customization: Model selection per step, June task persistence
```

**Step 5: Commit**

```bash
git add june-skills/subagent-driven-development/SKILL.md
git commit -m "feat(skills): add june task updates to subagent-driven-development"
```

---

### Task 7: Build and verify skills assembly

**Files:**
- Creates: `skills/` (assembled from superpowers + june-skills)

**Step 1: Run build-skills**

```bash
make build-skills
```

Expected output:
```
Skills assembled: superpowers v4.0.3 + june overrides
```

**Step 2: Verify skill count**

```bash
ls skills/ | wc -l
```

Expected: 14 skills (all from superpowers)

**Step 3: Verify june overrides applied**

```bash
head -5 skills/writing-plans/SKILL.md
```

Expected: Should show June customization comment, not original superpowers.

**Step 4: Verify plugin structure**

```bash
ls -la .claude-plugin/
ls skills/
```

Expected: plugin.json exists, skills/ has all skills.

**Step 5: Commit assembled skills**

```bash
git add skills/
git commit -m "feat: assemble skills from superpowers + june overrides"
```

---

### Task 8: Test plugin with Claude Code

**Step 1: Test plugin loads**

```bash
claude --plugin-dir . --version
```

Expected: Claude Code starts without plugin errors.

**Step 2: Verify skills are available**

Start Claude Code with plugin:

```bash
claude --plugin-dir .
```

Then type: `/june:writing-plans`

Expected: Skill should be recognized and loaded.

**Step 3: Test task create with note**

```bash
./june task create "Test task" --note "Test note" --json
```

Expected: Task created with note visible in `june task list <id>`.

**Step 4: Document test results**

If any issues found, fix and re-test before proceeding.

**Step 5: Final commit (if any fixes)**

```bash
git add -A
git commit -m "fix: address plugin testing issues"
```

---

### Task 9: Update documentation

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`

**Step 1: Add plugin section to README**

Add after "Task Commands" section:

```markdown
## Claude Code Plugin

June is also a Claude Code plugin providing task-aware workflow skills.

### Installation

```bash
git clone https://github.com/sky-xo/june
cd june
claude --plugin-dir .
```

### Available Skills

All superpowers skills plus June-specific customizations:
- `june:writing-plans` - Creates persistent June tasks from plans
- `june:executing-plans` - Updates task status during execution
- `june:subagent-driven-development` - Model selection + task tracking
- `june:fresheyes` - Multi-agent code review

### Building Skills

To update vendored superpowers or rebuild skills:

```bash
make build-skills
```
```

**Step 2: Update CLAUDE.md**

Add note about plugin capability in the "What is June?" section.

**Step 3: Commit**

```bash
git add README.md CLAUDE.md
git commit -m "docs: add plugin documentation"
```

---

## Summary

| Task | Description |
|------|-------------|
| 1 | Add --note flag to task create |
| 2 | Create plugin infrastructure |
| 3 | Create june-skills directory with custom skills |
| 4 | Modify writing-plans to use june tasks |
| 5 | Modify executing-plans to use june tasks |
| 6 | Modify subagent-driven-development to use june tasks |
| 7 | Build and verify skills assembly |
| 8 | Test plugin with Claude Code |
| 9 | Update documentation |

**Estimated commits:** 9
