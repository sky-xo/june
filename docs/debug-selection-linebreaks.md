# Selection Highlighting Line Break Bug - Debug Notes

## RESOLVED ✓

**Root Cause:** Bubbletea's viewport component silently strips TAB characters from content. When TABs appeared between ANSI escape sequences and text content (e.g., `\x1b[38;5;251m\t\t\tmsg`), stripping the TABs corrupted the output into `\x1b[38;5;251mmsg` - making the 'm' appear as a character after the escape sequence terminator instead of the intended content.

**Fix:** Convert TAB characters to spaces in `StyledLine.Render()` (cell.go) before content is passed to the viewport.

---

## Original Problem Description

When text selection is active on syntax-highlighted code in the transcript view, lines are breaking in the middle unexpectedly. The background highlighting also extends incorrectly.

**Visual symptoms from screenshots:**
- Line `if i < len(visibleLines)-1 {` breaks after the `-`, with `1 {` appearing on a separate line
- Gray background highlighting (color 238) extends across these broken line segments
- Only happens with syntax-highlighted code (glamour/chroma output)
- Does NOT happen when selection is inactive

## CRITICAL DISCOVERY: Selection Offset Bug

**The line breaks are causing a coordinate mismatch between clicks and selections!**

Testing results:
- `for scanner.Scan() {` (no indentation) → selects correctly
- `    line := scanner.Text()` (4 spaces indent) → **off by 2 characters**
- Selecting "co" in `continue` selects "ue" instead (6 char offset on deeply indented line)

**Pattern: The offset correlates with indentation level, specifically the glamour margin.**

## Architecture Overview

The selection highlighting system uses a cell-based model:

1. **Cell Model** (`internal/tui/cell.go`):
   - `Cell` - single character with `CellStyle` (FG, BG, Bold, Italic)
   - `StyledLine` - slice of Cells
   - `ParseStyledLine(string)` - parses ANSI string into cells
   - `StyledLine.Render()` - converts cells back to ANSI string
   - `StyledLine.WithSelection(start, end, bgColor)` - applies highlight BG

2. **Rendering Pipeline**:
   ```
   formatTranscript()
     → glamour renders markdown with syntax highlighting
     → strings.Split(content, "\n")

   updateViewport()
     → ParseStyledLine() each line into cells
     → Store as m.contentLines []StyledLine

   renderViewportContent()
     → If selection active: applySelectionHighlight()
       → WithSelection() adds BG color 238 to selected cells
     → Render() each StyledLine back to ANSI
     → strings.Join(rendered, "\n")
     → viewport.SetContent()

   View()
     → viewport.View() returns visible lines
     → renderPanelWithTitle() pads/truncates and adds borders
   ```

3. **Width Handling**:
   - `viewport.Width` set to `rightWidth - 2` (panel width minus borders)
   - `renderPanelWithTitle` uses `lipgloss.Width()` for visual width
   - `truncateToWidth()` uses `ansi.Truncate()` for ANSI-safe truncation

## What We've Verified Works Correctly

### Cell Model (all tests pass)
- `ParseStyledLine` correctly creates one cell per visible character
- Escape sequences are skipped, not stored as cells
- `len(cells)` == visual character count
- `StyledLine.String()` returns plain text
- `StyledLine.Render()` produces valid ANSI with correct visual width

### Width Calculations
```go
// Test results:
original := "\x1b[38;5;81mif\x1b[0m \x1b[38;5;141mi\x1b[0m"  // glamour output
lipgloss.Width(original) = 4  // correct

rendered := "\x1b[38;5;81;48;5;238mif\x1b[0;48;5;238m..."  // with selection
lipgloss.Width(rendered) = 4  // still correct

// Even complex glamour output:
glamourLine (1006 bytes) → lipgloss.Width = 32
renderedWithSelection (318 bytes) → lipgloss.Width = 32
cellCount = 32  // all match!
```

### ANSI Truncation
- `ansi.Truncate()` correctly handles escape sequences
- Does not cut through escape sequences
- Returns string with correct visual width

## What We've Tried

### Fix 1: ANSI-aware truncation
Changed `truncateToWidth()` from binary-search-on-runes to using `ansi.Truncate()`:
```go
// OLD (broken - could slice through escape sequences):
runes := []rune(s)
result := string(runes[:mid])

// NEW:
result := ansi.Truncate(s, maxWidth, "")
```
**Result:** Still broken

### Fix 2: Defensive resets after truncation
```go
result := ansi.Truncate(s, maxWidth, "")
result = result + "\x1b[0m"  // ensure styles reset
```
**Result:** Still broken

### Fix 3: Defensive resets before padding
```go
if visualWidth < contentWidth {
    line = line + "\x1b[0m" + strings.Repeat(" ", contentWidth-visualWidth)
}
```
**Result:** Still broken

### Fix 4: Reset even when exact width
```go
} else {
    line = line + "\x1b[0m"  // prevent bleeding into borders
}
```
**Result:** Still broken

### Fix 5: Incremental ANSI rendering (reduced output by 41%)
Changed `styleToANSI` from reset-and-restyle to incremental updates:
```go
// OLD: Always reset then restyle
if from != (CellStyle{}) && from != to {
    parts = append(parts, "0")  // Reset first
}

// NEW: Only emit what changed
if to.FG != from.FG {
    if to.FG.Type == ColorNone {
        parts = append(parts, "39") // default FG
    } else {
        parts = append(parts, colorToSGR(to.FG, false)...)
    }
}
// Similar for BG, bold, italic
```
**Result:** Output reduced from 292 to 171 bytes, but still broken

## Glamour Output Analysis

Glamour produces EXTREMELY verbose ANSI output:
```
Line 1 (padding): 1218 bytes for 80 spaces
  - Each space: \x1b[38;5;252m \x1b[0m (12 bytes per space!)

Line 2 (code): 1006 bytes for 32 visible chars
  - Each token wrapped: \x1b[38;5;39mif\x1b[0m\x1b[38;5;251m \x1b[0m...
```

After our parse+render:
- Without selection: ~201 bytes (grouped styles)
- With selection: ~318 bytes (grouped styles + BG)

Both have correct `lipgloss.Width` = 32.

## CURRENT LEADING HYPOTHESIS: BG-only at line start

Glamour adds a **2-space margin** to code blocks. These spaces are UNSTYLED in the original output:
```
  \x1b[38;5;252m    \x1b[0mline := scanner.Text()
^^-- plain spaces, no ANSI
```

When we apply selection, these become BG-only (no FG):
```
\x1b[48;5;238m  \x1b[38;5;252m    line...
```

**The theory:** Starting a line with `\x1b[48;5;238m` (BG-only, no FG) may cause terminal rendering issues.

**Evidence:**
- Lines without indentation work correctly
- Offset correlates with unstyled leading whitespace
- The 2-char offset matches glamour's 2-space margin

**Potential fix to try:** Always emit explicit FG at line start, even if default:
```go
// Instead of: \x1b[48;5;238m
// Emit:       \x1b[39;48;5;238m   (explicit default FG + BG)
```

## Unit Tests All Pass

Extensive testing showed:
- `lipgloss.Width()` returns identical values for selected/unselected
- `ansi.StringWidth()` returns identical values
- Cell counts match visual character counts
- The full pipeline simulation shows correct widths at every stage
- viewport.View() returns same number of lines

**The bug does NOT manifest in unit tests** - it only appears in actual terminal rendering.

## Hypotheses Ruled Out

1. ~~Width calculation errors~~ - All width tests pass
2. ~~ANSI sequence corruption~~ - Sequences are valid
3. ~~Cell count mismatches~~ - Cell counts are correct
4. ~~Join/Split cycle issues~~ - Tests show no corruption

## Hypotheses Still Possible

1. **BG-only sequences at line start** (LEADING THEORY)
   - Terminal may handle `\x1b[48;5;238m` differently at start of line
   - May need explicit FG even when "default"

2. **Terminal-specific rendering bug**
   - Only tested in iTerm2
   - Should test in other terminals

3. **Glamour margin handling**
   - The 2-space unstyled margin may need special handling
   - Could try preserving original glamour output for non-selected portions

## Key Files

- `internal/tui/cell.go` - Cell model, parsing, rendering (`styleToANSI`, `fullStyleToANSI`, `Render`)
- `internal/tui/cell_test.go` - Tests for cell model
- `internal/tui/model.go` - Main TUI model, `renderViewportContent`, `renderPanelWithTitle`, `screenToContentPosition`
- `internal/tui/selection.go` - Selection state and helpers

## Code Change Made (still in codebase)

The `styleToANSI` function in `cell.go` was changed to use incremental updates instead of reset-and-restyle. This reduced output by 41% but didn't fix the bug. The change is good regardless and should be kept.

## Reproduction

1. Run `./june` in a directory with Claude Code transcripts
2. Select a subagent that has syntax-highlighted code in its transcript
3. Click and drag to select text that includes code
4. Observe lines breaking in the middle
5. Try selecting specific characters on indented lines - note the offset!

## Suggested Next Steps (Priority Order)

### 1. Test BG-only hypothesis (MOST LIKELY FIX)
Modify `Render()` in `cell.go` to always emit explicit default FG (39) when starting a line with BG-only:
```go
// At the start of Render(), or when lastStyle is default and first cell has only BG:
// Instead of just \x1b[48;5;238m
// Emit \x1b[39;48;5;238m (explicit default FG + BG)
```

### 2. Create minimal standalone test
Write a Go program that:
- Prints our exact ANSI output directly to terminal (no bubbletea)
- Compare visual rendering of selected vs unselected
- This isolates whether bug is in our ANSI or in bubbletea/viewport

### 3. Test in different terminals
- Terminal.app
- kitty
- Alacritty
- VS Code integrated terminal

### 4. Hex dump comparison
Add debug mode that writes rendered output to files, then:
```bash
hexdump -C nosel.txt > nosel.hex
hexdump -C withsel.txt > withsel.hex
diff nosel.hex withsel.hex
```

### 5. Alternative approach: Don't re-render
Instead of parsing glamour output to cells and re-rendering:
- Keep original glamour strings
- For selection, inject BG codes at specific byte positions
- More complex but preserves original terminal-tested output
