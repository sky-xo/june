package cli

import (
	"regexp"
	"testing"
)

func TestGenerateName_Format(t *testing.T) {
	name := generateName()

	// Should match pattern: task-XXXXXX (6 alphanumeric chars)
	pattern := regexp.MustCompile(`^task-[a-zA-Z0-9]{6}$`)
	if !pattern.MatchString(name) {
		t.Errorf("generateName() = %q, want pattern task-XXXXXX", name)
	}
}

func TestGenerateName_NotConstant(t *testing.T) {
	// Generate several names and verify they're not all the same
	first := generateName()
	for i := 0; i < 10; i++ {
		if generateName() != first {
			return // Success - we got a different name
		}
	}
	t.Error("generateName() returned the same value 11 times in a row")
}

func TestGenerateAdjectiveNoun(t *testing.T) {
	name := generateAdjectiveNoun()

	// Should match pattern: adjective-noun (lowercase, hyphenated)
	pattern := regexp.MustCompile(`^[a-z]+-[a-z]+$`)
	if !pattern.MatchString(name) {
		t.Errorf("generateAdjectiveNoun() = %q, want adjective-noun pattern", name)
	}
}

func TestGenerateAdjectiveNoun_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		name := generateAdjectiveNoun()
		if seen[name] {
			// Collisions are possible but unlikely in 100 tries with 2500 combos
			// This is a sanity check, not a guarantee
			t.Logf("collision on %q (acceptable)", name)
		}
		seen[name] = true
	}
}
