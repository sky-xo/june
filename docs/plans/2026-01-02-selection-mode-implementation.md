# Selection Mode Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add mouse-based text selection to the transcript panel, allowing users to select and copy text.

**Architecture:** Add `SelectionState` to track selection anchor/current positions in content coordinates. Intercept mouse drag events in Update(), apply inverted styling during View() rendering, and copy stripped plain text to system clipboard.

**Tech Stack:** Bubbletea (existing), lipgloss (existing), `github.com/acarl005/stripansi`, `golang.design/x/clipboard`

---

## Task 1: Add Dependencies

**Files:**
- Modify: `/Users/glowy/code/june/.worktrees/select-mode/go.mod`

**Step 1: Add clipboard and stripansi dependencies**

Run:
```bash
cd /Users/glowy/code/june/.worktrees/select-mode && go get golang.design/x/clipboard github.com/acarl005/stripansi
```

**Step 2: Verify dependencies installed**

Run:
```bash
cd /Users/glowy/code/june/.worktrees/select-mode && go mod tidy && cat go.mod | grep -E "(clipboard|stripansi)"
```

Expected: Both dependencies listed in go.mod

**Step 3: Commit**

```bash
cd /Users/glowy/code/june/.worktrees/select-mode && git add go.mod go.sum && git commit -m "deps: add clipboard and stripansi libraries"
```

---

## Task 2: Add SelectionState and Position Types

**Files:**
- Modify: `/Users/glowy/code/june/.worktrees/select-mode/internal/tui/model.go:48-62`

**Step 1: Write the test**

Create file `/Users/glowy/code/june/.worktrees/select-mode/internal/tui/selection_test.go`:

```go
package tui

import "testing"

func TestSelectionState_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		state    SelectionState
		expected bool
	}{
		{
			name:     "inactive selection is empty",
			state:    SelectionState{Active: false},
			expected: true,
		},
		{
			name:     "same anchor and current is empty",
			state:    SelectionState{Active: true, Anchor: Position{Row: 5, Col: 10}, Current: Position{Row: 5, Col: 10}},
			expected: true,
		},
		{
			name:     "different positions is not empty",
			state:    SelectionState{Active: true, Anchor: Position{Row: 5, Col: 10}, Current: Position{Row: 5, Col: 15}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.IsEmpty(); got != tt.expected {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSelectionState_Normalize(t *testing.T) {
	tests := []struct {
		name          string
		anchor        Position
		current       Position
		expectedStart Position
		expectedEnd   Position
	}{
		{
			name:          "anchor before current",
			anchor:        Position{Row: 2, Col: 5},
			current:       Position{Row: 4, Col: 10},
			expectedStart: Position{Row: 2, Col: 5},
			expectedEnd:   Position{Row: 4, Col: 10},
		},
		{
			name:          "anchor after current (different rows)",
			anchor:        Position{Row: 4, Col: 10},
			current:       Position{Row: 2, Col: 5},
			expectedStart: Position{Row: 2, Col: 5},
			expectedEnd:   Position{Row: 4, Col: 10},
		},
		{
			name:          "same row anchor after current",
			anchor:        Position{Row: 3, Col: 15},
			current:       Position{Row: 3, Col: 5},
			expectedStart: Position{Row: 3, Col: 5},
			expectedEnd:   Position{Row: 3, Col: 15},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := SelectionState{Active: true, Anchor: tt.anchor, Current: tt.current}
			start, end := s.Normalize()
			if start != tt.expectedStart || end != tt.expectedEnd {
				t.Errorf("Normalize() = (%v, %v), want (%v, %v)", start, end, tt.expectedStart, tt.expectedEnd)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestSelectionState" -v`

Expected: FAIL - types not defined

**Step 3: Add the types to model.go**

Add after the `focusedPanel` constants (around line 46) in `model.go`:

```go
// Position represents a location in content (row = line number, col = character offset)
type Position struct {
	Row int // Line number in content (0-indexed)
	Col int // Character position in line (0-indexed)
}

// SelectionState tracks text selection in the content panel
type SelectionState struct {
	Active   bool     // Whether selection mode is active
	Anchor   Position // Where the drag started
	Current  Position // Current drag position
	Dragging bool     // Whether mouse button is currently held down
}

// IsEmpty returns true if there's no actual selection (same start and end, or not active)
func (s SelectionState) IsEmpty() bool {
	if !s.Active {
		return true
	}
	return s.Anchor == s.Current
}

// Normalize returns start and end positions where start is always before end
func (s SelectionState) Normalize() (start, end Position) {
	if s.Anchor.Row < s.Current.Row || (s.Anchor.Row == s.Current.Row && s.Anchor.Col <= s.Current.Col) {
		return s.Anchor, s.Current
	}
	return s.Current, s.Anchor
}
```

**Step 4: Add selection field to Model struct**

In the Model struct (around line 49), add:

```go
selection SelectionState // Text selection state
```

**Step 5: Run test to verify it passes**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestSelectionState" -v`

Expected: PASS

**Step 6: Run all tests**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./...`

Expected: All tests pass

**Step 7: Commit**

```bash
cd /Users/glowy/code/june/.worktrees/select-mode && git add -A && git commit -m "feat(selection): add SelectionState and Position types"
```

---

## Task 3: Add Content Line Tracking

The viewport content is a single formatted string. We need to track lines separately so we can map mouse positions to content positions.

**Files:**
- Modify: `/Users/glowy/code/june/.worktrees/select-mode/internal/tui/model.go`

**Step 1: Write the test**

Add to `selection_test.go`:

```go
func TestModel_ContentLines(t *testing.T) {
	m := NewModel("/test")
	m.width = 80
	m.height = 24

	// Set viewport content directly
	content := "Line one\nLine two\nLine three"
	m.viewport.SetContent(content)
	m.contentLines = strings.Split(content, "\n")

	if len(m.contentLines) != 3 {
		t.Errorf("Expected 3 content lines, got %d", len(m.contentLines))
	}
	if m.contentLines[0] != "Line one" {
		t.Errorf("Expected first line 'Line one', got %q", m.contentLines[0])
	}
}
```

Add `"strings"` to imports if not present.

**Step 2: Run test to verify it fails**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestModel_ContentLines" -v`

Expected: FAIL - contentLines field doesn't exist

**Step 3: Add contentLines field to Model**

In the Model struct, add:

```go
contentLines []string // Lines of content for selection mapping
```

**Step 4: Update updateViewport to populate contentLines**

Modify `updateViewport()` method:

```go
func (m *Model) updateViewport() {
	agent := m.SelectedAgent()
	if agent == nil {
		m.viewport.SetContent("")
		m.contentLines = nil
		return
	}
	entries := m.transcripts[agent.ID]
	content := formatTranscript(entries, m.viewport.Width)
	m.viewport.SetContent(content)
	m.contentLines = strings.Split(content, "\n")
}
```

**Step 5: Run test to verify it passes**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestModel_ContentLines" -v`

Expected: PASS

**Step 6: Run all tests**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./...`

Expected: All tests pass

**Step 7: Commit**

```bash
cd /Users/glowy/code/june/.worktrees/select-mode && git add -A && git commit -m "feat(selection): track content lines for position mapping"
```

---

## Task 4: Add Screen-to-Content Coordinate Mapping

**Files:**
- Modify: `/Users/glowy/code/june/.worktrees/select-mode/internal/tui/model.go`

**Step 1: Write the test**

Add to `selection_test.go`:

```go
func TestModel_ScreenToContentPosition(t *testing.T) {
	m := NewModel("/test")
	m.width = 80
	m.height = 24
	m.focusedPanel = panelRight

	// Set up viewport dimensions (simulate layout)
	m.viewport.Width = 50
	m.viewport.Height = 10
	m.viewport.YOffset = 0

	// Set content
	content := "Short\nA longer line here\nThird"
	m.viewport.SetContent(content)
	m.contentLines = strings.Split(content, "\n")

	tests := []struct {
		name        string
		screenX     int
		screenY     int
		expectedRow int
		expectedCol int
	}{
		{
			name:        "first character of first line",
			screenX:     sidebarWidth + 1, // After sidebar + left border
			screenY:     1,                 // After top border
			expectedRow: 0,
			expectedCol: 0,
		},
		{
			name:        "middle of second line",
			screenX:     sidebarWidth + 6, // 5 chars into content
			screenY:     2,                 // Second content line
			expectedRow: 1,
			expectedCol: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := m.screenToContentPosition(tt.screenX, tt.screenY)
			if pos.Row != tt.expectedRow || pos.Col != tt.expectedCol {
				t.Errorf("screenToContentPosition(%d, %d) = {Row:%d, Col:%d}, want {Row:%d, Col:%d}",
					tt.screenX, tt.screenY, pos.Row, pos.Col, tt.expectedRow, tt.expectedCol)
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestModel_ScreenToContentPosition" -v`

Expected: FAIL - method not defined

**Step 3: Implement screenToContentPosition**

Add to `model.go`:

```go
// screenToContentPosition converts screen coordinates to content position
// accounting for panel borders, sidebar width, and viewport scroll offset.
func (m *Model) screenToContentPosition(screenX, screenY int) Position {
	leftWidth, _, _, _ := m.layout()

	// Convert screen X to content column
	// Subtract: sidebar width + right panel left border (1)
	col := screenX - leftWidth - 1
	if col < 0 {
		col = 0
	}

	// Convert screen Y to content row
	// Subtract: top border (1), then add viewport scroll offset
	row := screenY - 1 + m.viewport.YOffset
	if row < 0 {
		row = 0
	}

	// Clamp to valid content range
	if len(m.contentLines) > 0 {
		if row >= len(m.contentLines) {
			row = len(m.contentLines) - 1
		}
		// Clamp column to line length (use visual width for ANSI content)
		lineLen := lipgloss.Width(m.contentLines[row])
		if col > lineLen {
			col = lineLen
		}
	}

	return Position{Row: row, Col: col}
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestModel_ScreenToContentPosition" -v`

Expected: PASS

**Step 5: Run all tests**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./...`

Expected: All tests pass

**Step 6: Commit**

```bash
cd /Users/glowy/code/june/.worktrees/select-mode && git add -A && git commit -m "feat(selection): add screen to content coordinate mapping"
```

---

## Task 5: Handle Mouse Drag Events for Selection

**Files:**
- Modify: `/Users/glowy/code/june/.worktrees/select-mode/internal/tui/model.go` (Update method)

**Step 1: Write the test**

Add to `selection_test.go`:

```go
func TestUpdate_MouseDragStartsSelection(t *testing.T) {
	m := NewModel("/test")
	m.width = 80
	m.height = 24
	m.focusedPanel = panelRight
	m.viewport.Width = 50
	m.viewport.Height = 10

	content := "Line one\nLine two\nLine three"
	m.viewport.SetContent(content)
	m.contentLines = strings.Split(content, "\n")

	// Simulate mouse press in content area
	pressMsg := tea.MouseMsg{
		X:      sidebarWidth + 5,
		Y:      2,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	}

	newModel, _ := m.Update(pressMsg)
	updated := newModel.(Model)

	if !updated.selection.Active {
		t.Error("Expected selection to be active after mouse press")
	}
	if !updated.selection.Dragging {
		t.Error("Expected dragging to be true after mouse press")
	}
}

func TestUpdate_MouseReleaseStopsDragging(t *testing.T) {
	m := NewModel("/test")
	m.width = 80
	m.height = 24
	m.focusedPanel = panelRight
	m.viewport.Width = 50
	m.viewport.Height = 10
	m.selection = SelectionState{
		Active:   true,
		Dragging: true,
		Anchor:   Position{Row: 1, Col: 5},
		Current:  Position{Row: 1, Col: 10},
	}

	content := "Line one\nLine two\nLine three"
	m.viewport.SetContent(content)
	m.contentLines = strings.Split(content, "\n")

	// Simulate mouse release
	releaseMsg := tea.MouseMsg{
		X:      sidebarWidth + 15,
		Y:      2,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	}

	newModel, _ := m.Update(releaseMsg)
	updated := newModel.(Model)

	if !updated.selection.Active {
		t.Error("Selection should remain active after release")
	}
	if updated.selection.Dragging {
		t.Error("Dragging should be false after release")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestUpdate_Mouse.*Selection" -v`

Expected: FAIL - selection not being set

**Step 3: Add mouse drag handling to Update method**

In the `Update` method, within the `case tea.MouseMsg:` block, after the existing click handling (around line 330), add:

```go
// Handle mouse events for text selection in right panel
if !inLeftPanel {
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft {
			// Start new selection
			pos := m.screenToContentPosition(msg.X, msg.Y)
			m.selection = SelectionState{
				Active:   true,
				Dragging: true,
				Anchor:   pos,
				Current:  pos,
			}
			return m, nil
		}
	case tea.MouseActionMotion:
		if m.selection.Dragging {
			// Update selection end point
			m.selection.Current = m.screenToContentPosition(msg.X, msg.Y)
			return m, nil
		}
	case tea.MouseActionRelease:
		if msg.Button == tea.MouseButtonLeft && m.selection.Dragging {
			// Finish dragging, keep selection active
			m.selection.Current = m.screenToContentPosition(msg.X, msg.Y)
			m.selection.Dragging = false

			// If empty selection (click without drag), exit selection mode
			if m.selection.IsEmpty() {
				m.selection.Active = false
			}
			return m, nil
		}
	}
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestUpdate_Mouse.*Selection" -v`

Expected: PASS

**Step 5: Run all tests**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./...`

Expected: All tests pass

**Step 6: Commit**

```bash
cd /Users/glowy/code/june/.worktrees/select-mode && git add -A && git commit -m "feat(selection): handle mouse drag events for selection"
```

---

## Task 6: Handle Keyboard Events in Selection Mode

**Files:**
- Modify: `/Users/glowy/code/june/.worktrees/select-mode/internal/tui/model.go` (Update method)

**Step 1: Write the test**

Add to `selection_test.go`:

```go
func TestUpdate_EscapeCancelsSelection(t *testing.T) {
	m := NewModel("/test")
	m.selection = SelectionState{
		Active: true,
		Anchor: Position{Row: 1, Col: 5},
		Current: Position{Row: 2, Col: 10},
	}

	msg := tea.KeyMsg{Type: tea.KeyEscape}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.selection.Active {
		t.Error("Expected selection to be inactive after Escape")
	}
}

func TestUpdate_CKeyInSelectionModeCopies(t *testing.T) {
	m := NewModel("/test")
	m.width = 80
	m.height = 24
	m.selection = SelectionState{
		Active:  true,
		Anchor:  Position{Row: 0, Col: 0},
		Current: Position{Row: 0, Col: 5},
	}
	m.contentLines = []string{"Hello World", "Line two"}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	// Selection should be cleared after copy
	if updated.selection.Active {
		t.Error("Expected selection to be inactive after copy")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestUpdate_.*Selection" -v`

Expected: FAIL - keys not handled in selection mode

**Step 3: Add keyboard handling for selection mode**

In the `Update` method, at the start of `case tea.KeyMsg:` (before existing key handling), add:

```go
	case tea.KeyMsg:
		// Handle selection mode keys first
		if m.selection.Active {
			switch msg.String() {
			case "esc":
				m.selection = SelectionState{}
				return m, nil
			case "c":
				m.copySelection()
				m.selection = SelectionState{}
				return m, nil
			}
			// In selection mode, block other keys except quit
			if msg.String() != "q" && msg.String() != "ctrl+c" {
				return m, nil
			}
		}
		// ... rest of existing key handling
```

**Step 4: Add copySelection stub method**

Add to `model.go`:

```go
// copySelection copies the selected text to the system clipboard
func (m *Model) copySelection() {
	// TODO: Implement in next task
}
```

**Step 5: Run test to verify it passes**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestUpdate_.*Selection" -v`

Expected: PASS

**Step 6: Run all tests**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./...`

Expected: All tests pass

**Step 7: Commit**

```bash
cd /Users/glowy/code/june/.worktrees/select-mode && git add -A && git commit -m "feat(selection): handle Escape and C keys in selection mode"
```

---

## Task 7: Implement Text Extraction and Clipboard Copy

**Files:**
- Modify: `/Users/glowy/code/june/.worktrees/select-mode/internal/tui/model.go`

**Step 1: Write the test**

Add to `selection_test.go`:

```go
func TestModel_GetSelectedText(t *testing.T) {
	m := NewModel("/test")
	m.contentLines = []string{
		"First line of text",
		"Second line here",
		"Third line content",
	}

	tests := []struct {
		name     string
		anchor   Position
		current  Position
		expected string
	}{
		{
			name:     "single line partial",
			anchor:   Position{Row: 0, Col: 6},
			current:  Position{Row: 0, Col: 10},
			expected: "line",
		},
		{
			name:     "multiple lines",
			anchor:   Position{Row: 0, Col: 6},
			current:  Position{Row: 1, Col: 6},
			expected: "line of text\nSecond",
		},
		{
			name:     "reversed selection",
			anchor:   Position{Row: 1, Col: 6},
			current:  Position{Row: 0, Col: 6},
			expected: "line of text\nSecond",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.selection = SelectionState{
				Active:  true,
				Anchor:  tt.anchor,
				Current: tt.current,
			}
			got := m.getSelectedText()
			if got != tt.expected {
				t.Errorf("getSelectedText() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestModel_GetSelectedText_StripsANSI(t *testing.T) {
	m := NewModel("/test")
	// Content with ANSI escape codes
	m.contentLines = []string{
		"\x1b[32mGreen text\x1b[0m normal",
	}
	m.selection = SelectionState{
		Active:  true,
		Anchor:  Position{Row: 0, Col: 0},
		Current: Position{Row: 0, Col: 16}, // "Green text normal" without codes
	}

	got := m.getSelectedText()

	// Should not contain ANSI codes
	if strings.Contains(got, "\x1b[") {
		t.Errorf("getSelectedText() should strip ANSI codes, got: %q", got)
	}
	// Should contain the text
	if !strings.Contains(got, "Green text") {
		t.Errorf("getSelectedText() should contain text, got: %q", got)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestModel_GetSelectedText" -v`

Expected: FAIL - method not defined

**Step 3: Implement getSelectedText**

Add to `model.go` (add import for `github.com/acarl005/stripansi`):

```go
import (
	// ... existing imports
	"github.com/acarl005/stripansi"
)

// getSelectedText extracts the selected text from content, stripping ANSI codes
func (m *Model) getSelectedText() string {
	if !m.selection.Active || m.selection.IsEmpty() {
		return ""
	}

	start, end := m.selection.Normalize()

	// Clamp to valid range
	if len(m.contentLines) == 0 {
		return ""
	}
	if start.Row >= len(m.contentLines) {
		return ""
	}
	if end.Row >= len(m.contentLines) {
		end.Row = len(m.contentLines) - 1
		end.Col = len(stripansi.Strip(m.contentLines[end.Row]))
	}

	var result strings.Builder

	for row := start.Row; row <= end.Row; row++ {
		line := stripansi.Strip(m.contentLines[row])

		startCol := 0
		endCol := len(line)

		if row == start.Row {
			startCol = start.Col
			if startCol > len(line) {
				startCol = len(line)
			}
		}
		if row == end.Row {
			endCol = end.Col
			if endCol > len(line) {
				endCol = len(line)
			}
		}

		if startCol < endCol {
			result.WriteString(line[startCol:endCol])
		}

		if row < end.Row {
			result.WriteString("\n")
		}
	}

	return result.String()
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestModel_GetSelectedText" -v`

Expected: PASS

**Step 5: Update copySelection to use clipboard**

Update the `copySelection` method (add import for `golang.design/x/clipboard`):

```go
import (
	// ... existing imports
	"golang.design/x/clipboard"
)

// copySelection copies the selected text to the system clipboard
func (m *Model) copySelection() {
	text := m.getSelectedText()
	if text == "" {
		return
	}

	// Initialize clipboard (safe to call multiple times)
	if err := clipboard.Init(); err != nil {
		return // Silently fail if clipboard unavailable
	}

	clipboard.Write(clipboard.FmtText, []byte(text))
}
```

**Step 6: Run all tests**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./...`

Expected: All tests pass

**Step 7: Commit**

```bash
cd /Users/glowy/code/june/.worktrees/select-mode && git add -A && git commit -m "feat(selection): implement text extraction and clipboard copy"
```

---

## Task 8: Render Selection with Inverted Colors

**Files:**
- Modify: `/Users/glowy/code/june/.worktrees/select-mode/internal/tui/model.go`

**Step 1: Write the test**

Add to `selection_test.go`:

```go
func TestModel_ApplySelectionHighlight(t *testing.T) {
	m := NewModel("/test")
	m.contentLines = []string{"Hello World"}
	m.selection = SelectionState{
		Active:  true,
		Anchor:  Position{Row: 0, Col: 0},
		Current: Position{Row: 0, Col: 5},
	}

	highlighted := m.applySelectionHighlight()

	// Should have exactly one line
	if len(highlighted) != 1 {
		t.Fatalf("Expected 1 line, got %d", len(highlighted))
	}

	// The highlighted version should be different from original
	if highlighted[0] == m.contentLines[0] {
		t.Error("Expected highlighting to modify the line")
	}

	// Should contain the word "Hello" somewhere (might be styled)
	stripped := stripANSI(highlighted[0])
	if !strings.Contains(stripped, "Hello") {
		t.Errorf("Expected 'Hello' in output, got: %s", stripped)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestModel_ApplySelectionHighlight" -v`

Expected: FAIL - method not defined

**Step 3: Implement applySelectionHighlight**

Add to `model.go`:

```go
// applySelectionHighlight returns content lines with selection highlighted using inverted colors
func (m *Model) applySelectionHighlight() []string {
	if !m.selection.Active || m.selection.IsEmpty() || len(m.contentLines) == 0 {
		return m.contentLines
	}

	start, end := m.selection.Normalize()
	result := make([]string, len(m.contentLines))
	copy(result, m.contentLines)

	// Clamp to valid range
	if start.Row >= len(m.contentLines) {
		return result
	}
	if end.Row >= len(m.contentLines) {
		end.Row = len(m.contentLines) - 1
		end.Col = lipgloss.Width(m.contentLines[end.Row])
	}

	invertStyle := lipgloss.NewStyle().Reverse(true)

	for row := start.Row; row <= end.Row; row++ {
		line := m.contentLines[row]
		strippedLine := stripansi.Strip(line)
		lineLen := len(strippedLine)

		startCol := 0
		endCol := lineLen

		if row == start.Row {
			startCol = start.Col
			if startCol > lineLen {
				startCol = lineLen
			}
		}
		if row == end.Row {
			endCol = end.Col
			if endCol > lineLen {
				endCol = lineLen
			}
		}

		if startCol >= endCol {
			continue
		}

		// Build the line with inverted selection
		// Note: This is simplified - for lines with ANSI codes, we use stripped text
		// A more sophisticated approach would parse and preserve ANSI codes
		before := strippedLine[:startCol]
		selected := strippedLine[startCol:endCol]
		after := strippedLine[endCol:]

		result[row] = before + invertStyle.Render(selected) + after
	}

	return result
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestModel_ApplySelectionHighlight" -v`

Expected: PASS

**Step 5: Integrate with View**

Modify the `View()` method to use highlighted content when selection is active. Find where the viewport is rendered (around line 483) and update:

```go
// Right panel: transcript
var rightContent string
if m.selection.Active && !m.selection.IsEmpty() {
	// Apply selection highlighting
	highlightedLines := m.applySelectionHighlight()
	// Only show visible portion based on viewport offset
	visibleStart := m.viewport.YOffset
	visibleEnd := visibleStart + m.viewport.Height
	if visibleEnd > len(highlightedLines) {
		visibleEnd = len(highlightedLines)
	}
	if visibleStart < len(highlightedLines) {
		rightContent = strings.Join(highlightedLines[visibleStart:visibleEnd], "\n")
	}
} else {
	rightContent = m.viewport.View()
}

var rightTitle string
// ... rest of right title logic ...

rightPanel := renderPanelWithTitle(rightTitle, rightContent, rightWidth, panelHeight, rightBorderColor)
```

**Step 6: Run all tests**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./...`

Expected: All tests pass

**Step 7: Commit**

```bash
cd /Users/glowy/code/june/.worktrees/select-mode && git add -A && git commit -m "feat(selection): render selection with inverted colors"
```

---

## Task 9: Add Header Indicator for Selection Mode

**Files:**
- Modify: `/Users/glowy/code/june/.worktrees/select-mode/internal/tui/model.go`

**Step 1: Write the test**

Add to `selection_test.go`:

```go
func TestView_ShowsSelectionIndicator(t *testing.T) {
	m := NewModel("/test")
	m.width = 80
	m.height = 24
	m.agents = []claude.Agent{{ID: "test123", FilePath: "/tmp/test.jsonl"}}
	m.selectedIdx = 0
	m.selection = SelectionState{
		Active:  true,
		Anchor:  Position{Row: 0, Col: 0},
		Current: Position{Row: 0, Col: 5},
	}
	m.contentLines = []string{"Hello World"}

	view := m.View()

	// Should contain the selection indicator
	if !strings.Contains(view, "SELECTING") {
		t.Errorf("Expected 'SELECTING' in view when selection active, got: %s", view)
	}
	if !strings.Contains(view, "C: copy") {
		t.Errorf("Expected 'C: copy' hint in view, got: %s", view)
	}
}

func TestView_NoSelectionIndicatorWhenInactive(t *testing.T) {
	m := NewModel("/test")
	m.width = 80
	m.height = 24
	m.agents = []claude.Agent{{ID: "test123", FilePath: "/tmp/test.jsonl"}}
	m.selectedIdx = 0
	m.selection = SelectionState{Active: false}
	m.contentLines = []string{"Hello World"}

	view := m.View()

	if strings.Contains(view, "SELECTING") {
		t.Errorf("Should not show 'SELECTING' when selection inactive")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestView_.*SelectionIndicator" -v`

Expected: FAIL - indicator not shown

**Step 3: Add selection indicator style**

Add to the style definitions at the top of `model.go` (around line 27):

```go
selectionIndicatorStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#32CD32")). // Limegreen
	Foreground(lipgloss.Color("#000000")). // Black text
	Bold(true).
	Padding(0, 1)
```

**Step 4: Update View to show indicator**

In the `View()` method, modify the right panel title logic (around line 476):

```go
// Right panel: transcript
var rightTitle string
if agent := m.SelectedAgent(); agent != nil {
	if agent.Description != "" {
		rightTitle = fmt.Sprintf("%s (%s) | %s", agent.Description, agent.ID, formatTimestamp(agent.LastMod))
	} else {
		rightTitle = fmt.Sprintf("%s | %s", agent.ID, formatTimestamp(agent.LastMod))
	}
}

// Add selection indicator to title
if m.selection.Active && !m.selection.IsEmpty() {
	indicator := selectionIndicatorStyle.Render("SELECTING · C: copy · Esc: cancel")
	if rightTitle != "" {
		rightTitle = rightTitle + " " + indicator
	} else {
		rightTitle = indicator
	}
}
```

**Step 5: Run test to verify it passes**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestView_.*SelectionIndicator" -v`

Expected: PASS

**Step 6: Run all tests**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./...`

Expected: All tests pass

**Step 7: Commit**

```bash
cd /Users/glowy/code/june/.worktrees/select-mode && git add -A && git commit -m "feat(selection): add limegreen header indicator in selection mode"
```

---

## Task 10: Implement Auto-scroll During Drag

**Files:**
- Modify: `/Users/glowy/code/june/.worktrees/select-mode/internal/tui/model.go`
- Modify: `/Users/glowy/code/june/.worktrees/select-mode/internal/tui/commands.go`

**Step 1: Write the test**

Add to `selection_test.go`:

```go
func TestUpdate_DragNearTopEdgeScrollsUp(t *testing.T) {
	m := NewModel("/test")
	m.width = 80
	m.height = 24
	m.focusedPanel = panelRight
	m.viewport.Width = 50
	m.viewport.Height = 10
	m.viewport.YOffset = 5 // Scrolled down

	// Create content with enough lines
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = fmt.Sprintf("Line %d content", i)
	}
	m.contentLines = lines
	m.viewport.SetContent(strings.Join(lines, "\n"))

	m.selection = SelectionState{
		Active:   true,
		Dragging: true,
		Anchor:   Position{Row: 7, Col: 0},
		Current:  Position{Row: 7, Col: 5},
	}

	// Simulate drag near top edge (Y=1 is the border, Y=2 is first content line)
	msg := tea.MouseMsg{
		X:      sidebarWidth + 5,
		Y:      2, // Near top edge
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionMotion,
	}

	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	// Should have scrolled up
	if updated.viewport.YOffset >= 5 {
		t.Errorf("Expected viewport to scroll up from offset 5, got %d", updated.viewport.YOffset)
	}
}
```

Add `"fmt"` to imports in test file.

**Step 2: Run test to verify it fails**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestUpdate_DragNearTopEdge" -v`

Expected: FAIL - no auto-scroll happening

**Step 3: Add auto-scroll logic to mouse motion handling**

Update the mouse motion handling in `Update()`:

```go
case tea.MouseActionMotion:
	if m.selection.Dragging {
		// Update selection end point
		m.selection.Current = m.screenToContentPosition(msg.X, msg.Y)

		// Auto-scroll if near edges
		_, _, _, contentHeight := m.layout()
		edgeThreshold := 2

		// Y position relative to content area (subtract top border)
		relativeY := msg.Y - 1

		if relativeY <= edgeThreshold && m.viewport.YOffset > 0 {
			// Near top edge - scroll up
			m.viewport.LineUp(1)
			// Update selection to follow scroll
			m.selection.Current = m.screenToContentPosition(msg.X, msg.Y)
		} else if relativeY >= contentHeight-edgeThreshold {
			// Near bottom edge - scroll down
			m.viewport.LineDown(1)
			// Update selection to follow scroll
			m.selection.Current = m.screenToContentPosition(msg.X, msg.Y)
		}

		return m, nil
	}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestUpdate_DragNearTopEdge" -v`

Expected: PASS

**Step 5: Run all tests**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./...`

Expected: All tests pass

**Step 6: Commit**

```bash
cd /Users/glowy/code/june/.worktrees/select-mode && git add -A && git commit -m "feat(selection): auto-scroll viewport during drag near edges"
```

---

## Task 11: Handle Click Outside Content Area

**Files:**
- Modify: `/Users/glowy/code/june/.worktrees/select-mode/internal/tui/model.go`

**Step 1: Write the test**

Add to `selection_test.go`:

```go
func TestUpdate_ClickOutsideContentExitsSelection(t *testing.T) {
	m := NewModel("/test")
	m.width = 80
	m.height = 24
	m.selection = SelectionState{
		Active:  true,
		Anchor:  Position{Row: 1, Col: 5},
		Current: Position{Row: 2, Col: 10},
	}

	// Click in left panel (sidebar)
	msg := tea.MouseMsg{
		X:      5, // In sidebar
		Y:      5,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
	}

	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.selection.Active {
		t.Error("Expected selection to be cancelled when clicking outside content area")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestUpdate_ClickOutsideContent" -v`

Expected: FAIL - selection remains active

**Step 3: Add click-outside handling**

In the `Update()` method, in the `case tea.MouseMsg:` block, add at the beginning (before the `inLeftPanel` check):

```go
// If selection is active and user clicks in left panel, cancel selection
if m.selection.Active && inLeftPanel && msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionRelease {
	m.selection = SelectionState{}
	// Don't return here - let the click also select an agent if applicable
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestUpdate_ClickOutsideContent" -v`

Expected: PASS

**Step 5: Run all tests**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./...`

Expected: All tests pass

**Step 6: Commit**

```bash
cd /Users/glowy/code/june/.worktrees/select-mode && git add -A && git commit -m "feat(selection): cancel selection when clicking outside content"
```

---

## Task 12: Preserve Selection While Scrolling

**Files:**
- Modify: `/Users/glowy/code/june/.worktrees/select-mode/internal/tui/model.go`

**Step 1: Write the test**

Add to `selection_test.go`:

```go
func TestUpdate_ScrollPreservesSelection(t *testing.T) {
	m := NewModel("/test")
	m.width = 80
	m.height = 24
	m.focusedPanel = panelRight
	m.viewport.Width = 50
	m.viewport.Height = 10

	// Create content with enough lines to scroll
	lines := make([]string, 30)
	for i := range lines {
		lines[i] = fmt.Sprintf("Line %d content here", i)
	}
	m.contentLines = lines
	m.viewport.SetContent(strings.Join(lines, "\n"))

	// Set up a selection
	originalSelection := SelectionState{
		Active:  true,
		Anchor:  Position{Row: 5, Col: 3},
		Current: Position{Row: 7, Col: 10},
	}
	m.selection = originalSelection

	// Scroll down with wheel
	msg := tea.MouseMsg{
		X:      sidebarWidth + 10,
		Y:      5,
		Button: tea.MouseButtonWheelDown,
	}

	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	// Selection should be preserved
	if !updated.selection.Active {
		t.Error("Selection should remain active after scrolling")
	}
	if updated.selection.Anchor != originalSelection.Anchor {
		t.Errorf("Selection anchor changed: got %v, want %v", updated.selection.Anchor, originalSelection.Anchor)
	}
	if updated.selection.Current != originalSelection.Current {
		t.Errorf("Selection current changed: got %v, want %v", updated.selection.Current, originalSelection.Current)
	}
}
```

**Step 2: Run test to verify it passes**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./internal/tui -run "TestUpdate_ScrollPreservesSelection" -v`

Expected: PASS (selection uses content positions, not screen positions, so scrolling naturally preserves it)

**Step 3: Commit (if test passes without changes)**

If the test passes without modification (because the design inherently supports this), commit with a note:

```bash
cd /Users/glowy/code/june/.worktrees/select-mode && git add -A && git commit -m "test(selection): verify selection persists during scroll"
```

---

## Task 13: Update CLAUDE.md with Selection Mode Documentation

**Files:**
- Modify: `/Users/glowy/code/june/.worktrees/select-mode/CLAUDE.md`

**Step 1: Update keyboard shortcuts section**

Add to the Keyboard Shortcuts section:

```markdown
## Keyboard Shortcuts

- `j`/`k` - Navigate agent list
- `u`/`d` - Page up/down in transcript
- `Tab` - Switch panel focus
- `q` - Quit

### Selection Mode (mouse-initiated)
- Click+drag in transcript - Start text selection
- `C` - Copy selection to clipboard and exit
- `Esc` - Exit selection mode without copying
```

**Step 2: Commit**

```bash
cd /Users/glowy/code/june/.worktrees/select-mode && git add -A && git commit -m "docs: add selection mode keyboard shortcuts"
```

---

## Task 14: Final Integration Test and Build

**Step 1: Run all tests**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && go test ./... -v`

Expected: All tests pass

**Step 2: Build the binary**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && make build`

Expected: Build succeeds

**Step 3: Manual smoke test**

Run: `cd /Users/glowy/code/june/.worktrees/select-mode && ./june`

Test manually:
1. Click and drag to select text in transcript
2. Verify inverted highlight appears
3. Verify header shows "SELECTING · C: copy · Esc: cancel"
4. Press C - verify text copied to clipboard
5. Click and drag again, press Esc - verify selection cancelled
6. Verify scroll wheel works during selection

**Step 4: Final commit if any fixes needed**

```bash
cd /Users/glowy/code/june/.worktrees/select-mode && git add -A && git commit -m "fix: integration fixes from manual testing"
```
