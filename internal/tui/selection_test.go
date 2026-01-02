package tui

import (
	"strings"
	"testing"

	"june/internal/claude"
)

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

func TestModel_ContentLines(t *testing.T) {
	m := NewModel("/test")
	m.width = 80
	m.height = 24

	// Set up agents and transcripts like the real application
	m.agents = []claude.Agent{
		{ID: "test-agent", FilePath: "/test/agent.jsonl"},
	}
	m.selectedIdx = 0

	// Add transcript entries for the selected agent
	m.transcripts["test-agent"] = []claude.Entry{
		{
			Type:    "user",
			Message: claude.Message{Content: "Hello world"},
		},
	}

	// Call updateViewport to populate contentLines
	m.updateViewport()

	// Verify contentLines was populated
	if len(m.contentLines) == 0 {
		t.Errorf("Expected contentLines to be populated, got empty slice")
	}

	// Verify the content contains the user message
	content := strings.Join(m.contentLines, "\n")
	if !strings.Contains(content, "Hello world") {
		t.Errorf("Expected contentLines to contain 'Hello world', got: %s", content)
	}
}

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
			screenY:     1,                // After top border
			expectedRow: 0,
			expectedCol: 0,
		},
		{
			name:        "middle of second line",
			screenX:     sidebarWidth + 6, // 5 chars into content
			screenY:     2,                // Second content line
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
