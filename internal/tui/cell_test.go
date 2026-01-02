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
