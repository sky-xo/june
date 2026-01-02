// internal/tui/cell.go
package tui

import (
	"strconv"
	"strings"
)

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

// ParseStyledLine parses a string with ANSI escape codes into a StyledLine
func ParseStyledLine(s string) StyledLine {
	var result StyledLine
	var currentStyle CellStyle

	runes := []rune(s)
	i := 0

	for i < len(runes) {
		r := runes[i]

		if r == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			// Start of CSI sequence
			// Find the end of the sequence (letter a-zA-Z)
			j := i + 2
			for j < len(runes) && !isCSITerminator(runes[j]) {
				j++
			}
			if j < len(runes) {
				terminator := runes[j]
				if terminator == 'm' {
					// SGR sequence - parse and apply style
					paramStr := string(runes[i+2 : j])
					currentStyle = applySGR(currentStyle, paramStr)
				}
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

// applySGR applies SGR (Select Graphic Rendition) parameters to a style
func applySGR(style CellStyle, paramStr string) CellStyle {
	params := splitSGRParams(paramStr)

	// Empty params or just "0" means reset
	if len(params) == 0 {
		return CellStyle{}
	}

	i := 0
	for i < len(params) {
		code := params[i]

		switch {
		case code == 0:
			// Reset all
			style = CellStyle{}
		case code == 1:
			// Bold
			style.Bold = true
		case code == 3:
			// Italic
			style.Italic = true
		case code == 22:
			// Bold off
			style.Bold = false
		case code == 23:
			// Italic off
			style.Italic = false
		case code >= 30 && code <= 37:
			// Basic foreground colors (30-37 -> 0-7)
			style.FG = Color{Type: ColorBasic, Value: uint32(code - 30)}
		case code >= 40 && code <= 47:
			// Basic background colors (40-47 -> 0-7)
			style.BG = Color{Type: ColorBasic, Value: uint32(code - 40)}
		case code >= 90 && code <= 97:
			// Bright foreground colors (90-97 -> 8-15)
			style.FG = Color{Type: ColorBasic, Value: uint32(code - 90 + 8)}
		case code >= 100 && code <= 107:
			// Bright background colors (100-107 -> 8-15)
			style.BG = Color{Type: ColorBasic, Value: uint32(code - 100 + 8)}
		case code == 38:
			// Extended foreground color
			if i+1 < len(params) {
				if params[i+1] == 5 && i+2 < len(params) {
					// 256-color: 38;5;n
					style.FG = Color{Type: Color256, Value: uint32(params[i+2])}
					i += 2
				} else if params[i+1] == 2 && i+4 < len(params) {
					// Truecolor: 38;2;r;g;b
					r := uint32(params[i+2])
					g := uint32(params[i+3])
					b := uint32(params[i+4])
					style.FG = Color{Type: ColorTrueColor, Value: (r << 16) | (g << 8) | b}
					i += 4
				}
			}
		case code == 48:
			// Extended background color
			if i+1 < len(params) {
				if params[i+1] == 5 && i+2 < len(params) {
					// 256-color: 48;5;n
					style.BG = Color{Type: Color256, Value: uint32(params[i+2])}
					i += 2
				} else if params[i+1] == 2 && i+4 < len(params) {
					// Truecolor: 48;2;r;g;b
					r := uint32(params[i+2])
					g := uint32(params[i+3])
					b := uint32(params[i+4])
					style.BG = Color{Type: ColorTrueColor, Value: (r << 16) | (g << 8) | b}
					i += 4
				}
			}
		case code == 39:
			// Default foreground
			style.FG = Color{}
		case code == 49:
			// Default background
			style.BG = Color{}
		}
		i++
	}

	return style
}

// splitSGRParams splits an SGR parameter string into integers
// e.g., "1;31;48;5;238" -> []int{1, 31, 48, 5, 238}
func splitSGRParams(s string) []int {
	if s == "" {
		return nil
	}

	var result []int
	var current int
	hasDigit := false

	for _, r := range s {
		if r >= '0' && r <= '9' {
			current = current*10 + int(r-'0')
			hasDigit = true
		} else if r == ';' {
			if hasDigit {
				result = append(result, current)
			} else {
				result = append(result, 0) // Empty param defaults to 0
			}
			current = 0
			hasDigit = false
		}
		// Ignore other characters
	}

	// Don't forget the last number
	if hasDigit {
		result = append(result, current)
	} else if len(result) > 0 {
		// Trailing semicolon
		result = append(result, 0)
	}

	return result
}

// isCSITerminator returns true if r terminates a CSI sequence
func isCSITerminator(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

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
		// Convert TABs to spaces - bubbletea viewport strips TABs which
		// corrupts ANSI sequences when TABs appear between escape codes and content
		if cell.Char == '\t' {
			buf.WriteRune(' ')
		} else {
			buf.WriteRune(cell.Char)
		}
	}

	// Reset at end if we had any styling
	if lastStyle != (CellStyle{}) {
		buf.WriteString("\x1b[0m")
	}

	return buf.String()
}

// styleToANSI generates ANSI codes to transition from old style to new style.
// Uses incremental updates - only emits codes for attributes that actually change.
// This produces shorter sequences and avoids potential terminal edge cases.
func styleToANSI(from, to CellStyle) string {
	// If target is default style, just reset
	if to == (CellStyle{}) {
		if from != (CellStyle{}) {
			return "\x1b[0m"
		}
		return ""
	}

	// If source is default, emit full target style
	if from == (CellStyle{}) {
		return fullStyleToANSI(to)
	}

	// Incremental update - only emit what changed
	var parts []string

	// Handle bold changes
	if to.Bold != from.Bold {
		if to.Bold {
			parts = append(parts, "1")
		} else {
			parts = append(parts, "22") // bold off
		}
	}

	// Handle italic changes
	if to.Italic != from.Italic {
		if to.Italic {
			parts = append(parts, "3")
		} else {
			parts = append(parts, "23") // italic off
		}
	}

	// Handle FG color changes
	if to.FG != from.FG {
		if to.FG.Type == ColorNone {
			parts = append(parts, "39") // default FG
		} else {
			parts = append(parts, colorToSGR(to.FG, false)...)
		}
	}

	// Handle BG color changes
	if to.BG != from.BG {
		if to.BG.Type == ColorNone {
			parts = append(parts, "49") // default BG
		} else {
			parts = append(parts, colorToSGR(to.BG, true)...)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return "\x1b[" + strings.Join(parts, ";") + "m"
}

// fullStyleToANSI emits a complete style specification (used when starting from default)
func fullStyleToANSI(style CellStyle) string {
	var parts []string

	if style.Bold {
		parts = append(parts, "1")
	}
	if style.Italic {
		parts = append(parts, "3")
	}
	if style.FG.Type != ColorNone {
		parts = append(parts, colorToSGR(style.FG, false)...)
	} else if style.BG.Type != ColorNone {
		// Always emit explicit default FG when we have BG but no FG.
		// Some terminals handle BG-only sequences (\x1b[48;5;Nm) poorly at line start,
		// causing rendering issues like unexpected line breaks.
		parts = append(parts, "39")
	}
	if style.BG.Type != ColorNone {
		parts = append(parts, colorToSGR(style.BG, true)...)
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
