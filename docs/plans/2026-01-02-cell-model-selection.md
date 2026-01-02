# Cell Model for Text Selection - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace ANSI string storage with a cell-based model so text selection works correctly without ANSI parsing bugs.

**Architecture:** Add ANSI-to-Cell parsing layer between formatTranscript() and contentLines storage. Selection modifies cell backgrounds directly. Render cells back to ANSI for viewport display.

**Tech Stack:** Go, bubbletea, lipgloss (for rendering)

---

## Progress (Updated 2026-01-02)

| Task | Status | Commit |
|------|--------|--------|
| Task 1: Cell Types and Basic Methods | ✅ Complete | `feat(cell): add Cell and StyledLine types` |
| Task 2: ANSI Parser - Basic Text | ✅ Complete | `775cedf feat(cell): add ParseStyledLine for plain text` |
| Task 3: ANSI Parser - SGR Sequences | ⏳ Pending | - |
| Task 4: StyledLine Renderer | ⏳ Pending | - |
| Task 5: WithSelection Method | ✅ Complete | `4146f35 feat(cell): add WithSelection for highlight application` |
| Task 6: Integration - Update Model | ⏳ Pending | - |
| Task 7: Cleanup old ANSI code | ⏳ Pending | - |

**Files created:**
- `internal/tui/cell.go` - Cell types, ParseStyledLine, WithSelection
- `internal/tui/cell_test.go` - Tests for all above

**To resume:** Continue from Task 3. Tasks 3 and 4 can run in parallel, then Task 6, then Task 7.

---

## Task 1: Cell Types and Basic Methods

**Files:**
- Create: `internal/tui/cell.go`
- Create: `internal/tui/cell_test.go`

**Step 1: Write test for basic Cell and StyledLine types**

```go
// internal/tui/cell_test.go
package tui

import "testing"

func TestStyledLineString(t *testing.T) {
	line := StyledLine{
		{Char: 'h', Style: CellStyle{}},
		{Char: 'i', Style: CellStyle{}},
	}
	if got := line.String(); got != "hi" {
		t.Errorf("String() = %q, want %q", got, "hi")
	}
}

func TestStyledLineLen(t *testing.T) {
	line := StyledLine{
		{Char: 'a', Style: CellStyle{}},
		{Char: 'b', Style: CellStyle{}},
		{Char: 'c', Style: CellStyle{}},
	}
	if got := len(line); got != 3 {
		t.Errorf("len() = %d, want 3", got)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tui -run TestStyledLine -v`
Expected: FAIL - types not defined

**Step 3: Write the cell types**

```go
// internal/tui/cell.go
package tui

// ColorType indicates how to interpret a Color value
type ColorType uint8

const (
	ColorNone     ColorType = iota // default/no color set
	ColorBasic                     // 0-15 standard + bright colors
	Color256                       // 0-255 palette
	ColorTrueColor                 // 24-bit RGB
)

// Color represents a terminal color
type Color struct {
	Type  ColorType
	Value uint32 // Basic: 0-15, Color256: 0-255, TrueColor: 0xRRGGBB
}

// CellStyle holds styling attributes for a cell
type CellStyle struct {
	FG     Color
	BG     Color
	Bold   bool
	Italic bool
}

// Cell represents a single character with its style
type Cell struct {
	Char  rune
	Style CellStyle
}

// StyledLine is a sequence of styled cells
type StyledLine []Cell

// String returns the plain text content without styling
func (sl StyledLine) String() string {
	runes := make([]rune, len(sl))
	for i, cell := range sl {
		runes[i] = cell.Char
	}
	return string(runes)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tui -run TestStyledLine -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/cell.go internal/tui/cell_test.go
git commit -m "feat(cell): add Cell and StyledLine types"
```

---

## Task 2: ANSI Parser - Basic Text

**Files:**
- Modify: `internal/tui/cell.go`
- Modify: `internal/tui/cell_test.go`

**Step 1: Write test for parsing plain text**

```go
func TestParseStyledLinePlainText(t *testing.T) {
	line := ParseStyledLine("hello")
	if got := line.String(); got != "hello" {
		t.Errorf("String() = %q, want %q", got, "hello")
	}
	if len(line) != 5 {
		t.Errorf("len = %d, want 5", len(line))
	}
}

func TestParseStyledLineEmpty(t *testing.T) {
	line := ParseStyledLine("")
	if len(line) != 0 {
		t.Errorf("len = %d, want 0", len(line))
	}
}

func TestParseStyledLineUnicode(t *testing.T) {
	line := ParseStyledLine("hello 世界")
	if got := line.String(); got != "hello 世界" {
		t.Errorf("String() = %q, want %q", got, "hello 世界")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tui -run TestParseStyledLine -v`
Expected: FAIL - ParseStyledLine not defined

**Step 3: Write basic parser (no ANSI yet)**

```go
// ParseStyledLine parses a string with ANSI escape codes into a StyledLine
func ParseStyledLine(s string) StyledLine {
	var result StyledLine
	var currentStyle CellStyle

	runes := []rune(s)
	i := 0

	for i < len(runes) {
		r := runes[i]

		if r == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			// Start of CSI sequence - skip for now, implement in next task
			// Find the end of the sequence (letter a-zA-Z)
			j := i + 2
			for j < len(runes) && !isCSITerminator(runes[j]) {
				j++
			}
			if j < len(runes) {
				j++ // include terminator
			}
			i = j
			continue
		}

		result = append(result, Cell{Char: r, Style: currentStyle})
		i++
	}

	return result
}

// isCSITerminator returns true if r terminates a CSI sequence
func isCSITerminator(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tui -run TestParseStyledLine -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/cell.go internal/tui/cell_test.go
git commit -m "feat(cell): add ParseStyledLine for plain text"
```

---

## Task 3: ANSI Parser - SGR Sequences (Colors & Attributes)

**Files:**
- Modify: `internal/tui/cell.go`
- Modify: `internal/tui/cell_test.go`

**Step 1: Write tests for ANSI color parsing**

```go
func TestParseStyledLineBasicColors(t *testing.T) {
	// Red foreground: ESC[31m
	line := ParseStyledLine("\x1b[31mred\x1b[0m")
	if line[0].Style.FG.Type != ColorBasic || line[0].Style.FG.Value != 1 {
		t.Errorf("expected red FG, got %+v", line[0].Style.FG)
	}
	if got := line.String(); got != "red" {
		t.Errorf("String() = %q, want %q", got, "red")
	}
}

func TestParseStyledLine256Colors(t *testing.T) {
	// 256-color foreground: ESC[38;5;196m (bright red)
	line := ParseStyledLine("\x1b[38;5;196mtext\x1b[0m")
	if line[0].Style.FG.Type != Color256 || line[0].Style.FG.Value != 196 {
		t.Errorf("expected 256-color 196, got %+v", line[0].Style.FG)
	}
}

func TestParseStyledLineTrueColor(t *testing.T) {
	// Truecolor foreground: ESC[38;2;255;128;0m (orange)
	line := ParseStyledLine("\x1b[38;2;255;128;0mtext\x1b[0m")
	if line[0].Style.FG.Type != ColorTrueColor {
		t.Errorf("expected truecolor, got %v", line[0].Style.FG.Type)
	}
	expected := uint32(0xFF8000)
	if line[0].Style.FG.Value != expected {
		t.Errorf("expected 0x%06X, got 0x%06X", expected, line[0].Style.FG.Value)
	}
}

func TestParseStyledLineBold(t *testing.T) {
	line := ParseStyledLine("\x1b[1mbold\x1b[0m")
	if !line[0].Style.Bold {
		t.Error("expected bold")
	}
}

func TestParseStyledLineReset(t *testing.T) {
	line := ParseStyledLine("\x1b[31mred\x1b[0mnormal")
	// 'n' of 'normal' should have no color
	nIdx := 3 // r=0, e=1, d=2, n=3
	if line[nIdx].Style.FG.Type != ColorNone {
		t.Errorf("expected reset, got %+v", line[nIdx].Style.FG)
	}
}

func TestParseStyledLineBackground(t *testing.T) {
	// Background: ESC[48;5;238m
	line := ParseStyledLine("\x1b[48;5;238mtext\x1b[0m")
	if line[0].Style.BG.Type != Color256 || line[0].Style.BG.Value != 238 {
		t.Errorf("expected 256-color BG 238, got %+v", line[0].Style.BG)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tui -run TestParseStyledLine -v`
Expected: FAIL - colors not being parsed

**Step 3: Implement SGR parsing**

```go
// ParseStyledLine parses a string with ANSI escape codes into a StyledLine
func ParseStyledLine(s string) StyledLine {
	var result StyledLine
	var currentStyle CellStyle

	runes := []rune(s)
	i := 0

	for i < len(runes) {
		r := runes[i]

		if r == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			// CSI sequence
			j := i + 2
			// Collect parameters until terminator
			for j < len(runes) && !isCSITerminator(runes[j]) {
				j++
			}
			if j < len(runes) {
				terminator := runes[j]
				params := string(runes[i+2 : j])
				if terminator == 'm' {
					// SGR sequence - parse and update style
					currentStyle = parseSGR(params, currentStyle)
				}
				j++ // skip terminator
			}
			i = j
			continue
		}

		result = append(result, Cell{Char: r, Style: currentStyle})
		i++
	}

	return result
}

// parseSGR parses SGR (Select Graphic Rendition) parameters and updates style
func parseSGR(params string, style CellStyle) CellStyle {
	if params == "" || params == "0" {
		// Reset
		return CellStyle{}
	}

	parts := splitSGRParams(params)
	i := 0

	for i < len(parts) {
		p := parts[i]

		switch {
		case p == 0:
			style = CellStyle{} // reset
		case p == 1:
			style.Bold = true
		case p == 3:
			style.Italic = true
		case p == 22:
			style.Bold = false
		case p == 23:
			style.Italic = false

		// Basic foreground colors (30-37)
		case p >= 30 && p <= 37:
			style.FG = Color{Type: ColorBasic, Value: uint32(p - 30)}

		// Basic background colors (40-47)
		case p >= 40 && p <= 47:
			style.BG = Color{Type: ColorBasic, Value: uint32(p - 40)}

		// Bright foreground colors (90-97)
		case p >= 90 && p <= 97:
			style.FG = Color{Type: ColorBasic, Value: uint32(p - 90 + 8)}

		// Bright background colors (100-107)
		case p >= 100 && p <= 107:
			style.BG = Color{Type: ColorBasic, Value: uint32(p - 100 + 8)}

		// Extended foreground color
		case p == 38:
			if i+1 < len(parts) {
				if parts[i+1] == 5 && i+2 < len(parts) {
					// 256-color: 38;5;n
					style.FG = Color{Type: Color256, Value: uint32(parts[i+2])}
					i += 2
				} else if parts[i+1] == 2 && i+4 < len(parts) {
					// Truecolor: 38;2;r;g;b
					r, g, b := uint32(parts[i+2]), uint32(parts[i+3]), uint32(parts[i+4])
					style.FG = Color{Type: ColorTrueColor, Value: (r << 16) | (g << 8) | b}
					i += 4
				}
			}

		// Extended background color
		case p == 48:
			if i+1 < len(parts) {
				if parts[i+1] == 5 && i+2 < len(parts) {
					// 256-color: 48;5;n
					style.BG = Color{Type: Color256, Value: uint32(parts[i+2])}
					i += 2
				} else if parts[i+1] == 2 && i+4 < len(parts) {
					// Truecolor: 48;2;r;g;b
					r, g, b := uint32(parts[i+2]), uint32(parts[i+3]), uint32(parts[i+4])
					style.BG = Color{Type: ColorTrueColor, Value: (r << 16) | (g << 8) | b}
					i += 4
				}
			}

		case p == 39:
			style.FG = Color{} // default FG
		case p == 49:
			style.BG = Color{} // default BG
		}

		i++
	}

	return style
}

// splitSGRParams splits "1;31;48;5;238" into []int{1, 31, 48, 5, 238}
func splitSGRParams(params string) []int {
	if params == "" {
		return []int{0}
	}

	var result []int
	var current int
	hasDigit := false

	for _, r := range params {
		if r >= '0' && r <= '9' {
			current = current*10 + int(r-'0')
			hasDigit = true
		} else if r == ';' {
			result = append(result, current)
			current = 0
			hasDigit = false
		}
	}

	if hasDigit {
		result = append(result, current)
	}

	return result
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tui -run TestParseStyledLine -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/cell.go internal/tui/cell_test.go
git commit -m "feat(cell): parse SGR sequences (colors, bold, italic)"
```

---

## Task 4: StyledLine Renderer

**Files:**
- Modify: `internal/tui/cell.go`
- Modify: `internal/tui/cell_test.go`

**Step 1: Write test for rendering back to ANSI**

```go
func TestStyledLineRender(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"plain", "hello"},
		{"basic color", "\x1b[31mred\x1b[0m"},
		{"256 color", "\x1b[38;5;196mtext\x1b[0m"},
		{"bold", "\x1b[1mbold\x1b[0m normal"},
		{"background", "\x1b[48;5;238mhighlighted\x1b[0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line := ParseStyledLine(tt.input)
			rendered := line.Render()
			// Parse the rendered output and compare strings
			reparsed := ParseStyledLine(rendered)
			if line.String() != reparsed.String() {
				t.Errorf("text mismatch: %q vs %q", line.String(), reparsed.String())
			}
			// Verify styles match
			for i := range line {
				if i < len(reparsed) && line[i].Style != reparsed[i].Style {
					t.Errorf("style mismatch at %d: %+v vs %+v", i, line[i].Style, reparsed[i].Style)
				}
			}
		})
	}
}

func TestStyledLineRenderEmpty(t *testing.T) {
	line := StyledLine{}
	if got := line.Render(); got != "" {
		t.Errorf("Render() = %q, want empty", got)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tui -run TestStyledLineRender -v`
Expected: FAIL - Render not defined

**Step 3: Implement Render method**

```go
// Render converts a StyledLine back to an ANSI-escaped string
func (sl StyledLine) Render() string {
	if len(sl) == 0 {
		return ""
	}

	var buf strings.Builder
	var lastStyle CellStyle

	for _, cell := range sl {
		if cell.Style != lastStyle {
			// Emit style change
			buf.WriteString(styleToANSI(lastStyle, cell.Style))
			lastStyle = cell.Style
		}
		buf.WriteRune(cell.Char)
	}

	// Reset at end if we had any styling
	if lastStyle != (CellStyle{}) {
		buf.WriteString("\x1b[0m")
	}

	return buf.String()
}

// styleToANSI generates ANSI codes to transition from old style to new style
func styleToANSI(from, to CellStyle) string {
	// If target is default, just reset
	if to == (CellStyle{}) {
		if from != (CellStyle{}) {
			return "\x1b[0m"
		}
		return ""
	}

	var parts []string

	// If coming from styled, reset first for simplicity
	// (A more optimal impl would compute minimal diff)
	if from != (CellStyle{}) && from != to {
		parts = append(parts, "0")
	}

	if to.Bold {
		parts = append(parts, "1")
	}
	if to.Italic {
		parts = append(parts, "3")
	}

	if to.FG.Type != ColorNone {
		parts = append(parts, colorToSGR(to.FG, false)...)
	}
	if to.BG.Type != ColorNone {
		parts = append(parts, colorToSGR(to.BG, true)...)
	}

	if len(parts) == 0 {
		return ""
	}

	return "\x1b[" + strings.Join(parts, ";") + "m"
}

// colorToSGR converts a Color to SGR parameter strings
func colorToSGR(c Color, isBG bool) []string {
	switch c.Type {
	case ColorBasic:
		base := 30
		if isBG {
			base = 40
		}
		if c.Value >= 8 {
			base += 60 // bright colors
			return []string{strconv.Itoa(base + int(c.Value) - 8)}
		}
		return []string{strconv.Itoa(base + int(c.Value))}

	case Color256:
		if isBG {
			return []string{"48", "5", strconv.Itoa(int(c.Value))}
		}
		return []string{"38", "5", strconv.Itoa(int(c.Value))}

	case ColorTrueColor:
		r := (c.Value >> 16) & 0xFF
		g := (c.Value >> 8) & 0xFF
		b := c.Value & 0xFF
		if isBG {
			return []string{"48", "2", strconv.Itoa(int(r)), strconv.Itoa(int(g)), strconv.Itoa(int(b))}
		}
		return []string{"38", "2", strconv.Itoa(int(r)), strconv.Itoa(int(g)), strconv.Itoa(int(b))}
	}

	return nil
}
```

Add import at top of cell.go:
```go
import (
	"strconv"
	"strings"
)
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tui -run TestStyledLineRender -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/cell.go internal/tui/cell_test.go
git commit -m "feat(cell): add StyledLine.Render() for ANSI output"
```

---

## Task 5: WithSelection Method

**Files:**
- Modify: `internal/tui/cell.go`
- Modify: `internal/tui/cell_test.go`

**Step 1: Write test for selection highlighting**

```go
func TestStyledLineWithSelection(t *testing.T) {
	line := ParseStyledLine("hello world")
	highlight := Color{Type: Color256, Value: 238}

	selected := line.WithSelection(0, 5, highlight) // "hello"

	// First 5 chars should have highlight BG
	for i := 0; i < 5; i++ {
		if selected[i].Style.BG != highlight {
			t.Errorf("char %d should be highlighted", i)
		}
	}
	// Rest should not
	for i := 5; i < len(selected); i++ {
		if selected[i].Style.BG == highlight {
			t.Errorf("char %d should not be highlighted", i)
		}
	}
}

func TestStyledLineWithSelectionPreservesExistingStyle(t *testing.T) {
	line := ParseStyledLine("\x1b[31mred text\x1b[0m")
	highlight := Color{Type: Color256, Value: 238}

	selected := line.WithSelection(0, 3, highlight) // "red"

	// Should have both red FG and highlight BG
	if selected[0].Style.FG.Type != ColorBasic || selected[0].Style.FG.Value != 1 {
		t.Error("should preserve red foreground")
	}
	if selected[0].Style.BG != highlight {
		t.Error("should have highlight background")
	}
}

func TestStyledLineWithSelectionPartial(t *testing.T) {
	line := ParseStyledLine("hello")
	highlight := Color{Type: Color256, Value: 238}

	// Select middle: "ell"
	selected := line.WithSelection(1, 4, highlight)

	if selected[0].Style.BG.Type != ColorNone {
		t.Error("'h' should not be highlighted")
	}
	if selected[1].Style.BG != highlight {
		t.Error("'e' should be highlighted")
	}
	if selected[4].Style.BG.Type != ColorNone {
		t.Error("'o' should not be highlighted")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/tui -run TestStyledLineWithSelection -v`
Expected: FAIL - WithSelection not defined

**Step 3: Implement WithSelection**

```go
// WithSelection returns a copy of the line with selection highlighting applied
// from startCol to endCol (exclusive). The highlight background is applied
// while preserving existing foreground colors and attributes.
func (sl StyledLine) WithSelection(startCol, endCol int, bg Color) StyledLine {
	if len(sl) == 0 {
		return sl
	}

	// Clamp to valid range
	if startCol < 0 {
		startCol = 0
	}
	if endCol > len(sl) {
		endCol = len(sl)
	}
	if startCol >= endCol {
		return sl
	}

	// Make a copy
	result := make(StyledLine, len(sl))
	copy(result, sl)

	// Apply highlight background to selected range
	for i := startCol; i < endCol; i++ {
		result[i].Style.BG = bg
	}

	return result
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/tui -run TestStyledLineWithSelection -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/tui/cell.go internal/tui/cell_test.go
git commit -m "feat(cell): add WithSelection for highlight application"
```

---

## Task 6: Integration - Update Model ContentLines Type

**Files:**
- Modify: `internal/tui/model.go`

**Step 1: Update contentLines type and related methods**

Change in Model struct:
```go
// Before:
contentLines []string

// After:
contentLines []StyledLine
```

**Step 2: Update updateViewport()**

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

	// Parse ANSI content into StyledLines
	lines := strings.Split(content, "\n")
	m.contentLines = make([]StyledLine, len(lines))
	for i, line := range lines {
		m.contentLines[i] = ParseStyledLine(line)
	}

	// Render for viewport (apply selection if active)
	m.renderViewportContent()
}

// renderViewportContent renders contentLines to viewport, applying selection if active
func (m *Model) renderViewportContent() {
	lines := m.contentLines
	if m.selection.Active && !m.selection.IsEmpty() {
		lines = m.applySelectionHighlight()
	}

	rendered := make([]string, len(lines))
	for i, line := range lines {
		rendered[i] = line.Render()
	}
	m.viewport.SetContent(strings.Join(rendered, "\n"))
}
```

**Step 3: Update applySelectionHighlight()**

```go
// applySelectionHighlight returns content lines with selection highlighting applied
func (m *Model) applySelectionHighlight() []StyledLine {
	if !m.selection.Active || m.selection.IsEmpty() || len(m.contentLines) == 0 {
		return m.contentLines
	}

	start, end := m.selection.Normalize()
	result := make([]StyledLine, len(m.contentLines))
	copy(result, m.contentLines)

	// Clamp to valid range
	if start.Row >= len(m.contentLines) {
		return result
	}
	if end.Row >= len(m.contentLines) {
		end.Row = len(m.contentLines) - 1
		end.Col = len(m.contentLines[end.Row])
	}

	highlightBG := Color{Type: Color256, Value: 238}

	for row := start.Row; row <= end.Row; row++ {
		startCol := 0
		endCol := len(m.contentLines[row])

		if row == start.Row {
			startCol = start.Col
		}
		if row == end.Row {
			endCol = end.Col
		}

		if startCol < endCol {
			result[row] = m.contentLines[row].WithSelection(startCol, endCol, highlightBG)
		}
	}

	return result
}
```

**Step 4: Update getSelectedText()**

```go
// getSelectedText extracts the selected text from content
func (m *Model) getSelectedText() string {
	if !m.selection.Active || m.selection.IsEmpty() {
		return ""
	}

	start, end := m.selection.Normalize()

	if len(m.contentLines) == 0 || start.Row >= len(m.contentLines) {
		return ""
	}
	if end.Row >= len(m.contentLines) {
		end.Row = len(m.contentLines) - 1
		end.Col = len(m.contentLines[end.Row])
	}

	var result strings.Builder

	for row := start.Row; row <= end.Row; row++ {
		line := m.contentLines[row]
		lineLen := len(line)

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

		if startCol < endCol {
			// Extract plain text from cells
			for i := startCol; i < endCol; i++ {
				result.WriteRune(line[i].Char)
			}
		}

		if row < end.Row {
			result.WriteString("\n")
		}
	}

	return result.String()
}
```

**Step 5: Update restoreViewportContent() and updateSelectionHighlight()**

```go
// restoreViewportContent renders the viewport without selection highlighting
func (m *Model) restoreViewportContent() {
	m.renderViewportContent()
}

// updateSelectionHighlight re-renders viewport with current selection state
func (m *Model) updateSelectionHighlight() {
	m.renderViewportContent()
}
```

**Step 6: Update screenToContentPosition() - use StyledLine length**

```go
// In screenToContentPosition, change:
lineLen := lipgloss.Width(m.contentLines[row])

// To:
lineLen := len(m.contentLines[row])
```

**Step 7: Run tests**

Run: `make test`
Expected: PASS (may need minor fixes)

**Step 8: Commit**

```bash
git add internal/tui/model.go
git commit -m "refactor(model): use StyledLine for contentLines"
```

---

## Task 7: Cleanup - Remove Old ANSI Slicing Code

**Files:**
- Modify: `internal/tui/model.go`

**Step 1: Remove obsolete functions**

Delete these functions from model.go:
- `sliceByVisualWidth()`
- Any remaining ANSI manipulation code

**Step 2: Run tests**

Run: `make test`
Expected: PASS

**Step 3: Build and manual test**

Run: `make build && ./june`
Test: Select text in middle of syntax-highlighted code, verify no indentation issues

**Step 4: Commit**

```bash
git add internal/tui/model.go
git commit -m "refactor(model): remove obsolete ANSI slicing code"
```

---

## Summary

| Task | Description | Parallel? |
|------|-------------|-----------|
| 1 | Cell types + String() | - |
| 2 | Parser - plain text | After 1 |
| 3 | Parser - SGR sequences | After 2 |
| 4 | Render() method | After 3 |
| 5 | WithSelection() | After 1 (parallel with 2-4) |
| 6 | Model integration | After 4, 5 |
| 7 | Cleanup | After 6 |

**Parallelization opportunity:** Tasks 2-4 (parser chain) can run in parallel with Task 5 (WithSelection) since they only share the Cell types from Task 1.
