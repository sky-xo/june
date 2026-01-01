package tui

import (
	"testing"
	"time"
)

func TestFormatTimestamp(t *testing.T) {
	// Fix "now" to a known time for predictable tests
	// We'll test relative to actual time.Now() since the function uses it

	now := time.Now()

	tests := []struct {
		name     string
		input    time.Time
		contains string // We check contains since exact time formatting varies
	}{
		{
			name:     "today shows just time",
			input:    now.Add(-1 * time.Hour),
			contains: "PM", // or AM depending on time - just verify it's a time format
		},
		{
			name:     "yesterday shows Yesterday @",
			input:    now.AddDate(0, 0, -1),
			contains: "Yesterday @",
		},
		{
			name:     "3 days ago shows weekday",
			input:    now.AddDate(0, 0, -3),
			contains: "@", // Should have weekday @ time
		},
		{
			name:     "2 weeks ago shows date",
			input:    now.AddDate(0, 0, -14),
			contains: "@", // Should have "14 Jan @" or similar
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTimestamp(tt.input)
			if !contains(result, tt.contains) {
				t.Errorf("formatTimestamp(%v) = %q, expected to contain %q", tt.input, result, tt.contains)
			}
		})
	}
}

func TestFormatTimestamp_Today(t *testing.T) {
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)

	result := formatTimestamp(oneHourAgo)

	// Today's time should NOT contain "Yesterday" or "@" prefix
	if contains(result, "Yesterday") {
		t.Errorf("Today's timestamp should not contain 'Yesterday': %q", result)
	}
	// Should be just a time like "3:04:05 PM"
	if contains(result, "@") {
		t.Errorf("Today's timestamp should not contain '@': %q", result)
	}
}

func TestFormatTimestamp_Yesterday(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	result := formatTimestamp(yesterday)

	if !contains(result, "Yesterday @") {
		t.Errorf("Yesterday's timestamp should contain 'Yesterday @': %q", result)
	}
}

func TestFormatTimestamp_ThisWeek(t *testing.T) {
	now := time.Now()
	threeDaysAgo := now.AddDate(0, 0, -3)

	result := formatTimestamp(threeDaysAgo)

	// Should contain a weekday abbreviation
	weekdays := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	hasWeekday := false
	for _, day := range weekdays {
		if contains(result, day) {
			hasWeekday = true
			break
		}
	}
	if !hasWeekday {
		t.Errorf("This week's timestamp should contain weekday: %q", result)
	}
	if !contains(result, "@") {
		t.Errorf("This week's timestamp should contain '@': %q", result)
	}
}

func TestFormatTimestamp_Older(t *testing.T) {
	now := time.Now()
	twoWeeksAgo := now.AddDate(0, 0, -14)

	result := formatTimestamp(twoWeeksAgo)

	// Should contain a month abbreviation
	months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	hasMonth := false
	for _, month := range months {
		if contains(result, month) {
			hasMonth = true
			break
		}
	}
	if !hasMonth {
		t.Errorf("Older timestamp should contain month: %q", result)
	}
	if !contains(result, "@") {
		t.Errorf("Older timestamp should contain '@': %q", result)
	}
}

// helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchString(s, substr)))
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
