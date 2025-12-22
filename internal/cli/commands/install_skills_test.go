package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunInstallSkillsCopiesAndOverwritesOttoOnly(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	source := filepath.Join(t.TempDir(), "skills")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("mkdir source: %v", err)
	}

	ottoSkill := filepath.Join(source, "otto-orchestrate")
	userSkill := filepath.Join(source, "user-skill")
	if err := os.MkdirAll(ottoSkill, 0o755); err != nil {
		t.Fatalf("mkdir otto: %v", err)
	}
	if err := os.MkdirAll(userSkill, 0o755); err != nil {
		t.Fatalf("mkdir user: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ottoSkill, "SKILL.md"), []byte("otto new"), 0o644); err != nil {
		t.Fatalf("write otto: %v", err)
	}
	if err := os.WriteFile(filepath.Join(userSkill, "SKILL.md"), []byte("user new"), 0o644); err != nil {
		t.Fatalf("write user: %v", err)
	}

	dest := filepath.Join(tempHome, ".claude", "skills")
	if err := os.MkdirAll(filepath.Join(dest, "otto-orchestrate"), 0o755); err != nil {
		t.Fatalf("mkdir dest otto: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dest, "user-skill"), 0o755); err != nil {
		t.Fatalf("mkdir dest user: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "otto-orchestrate", "SKILL.md"), []byte("otto old"), 0o644); err != nil {
		t.Fatalf("write dest otto: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dest, "user-skill", "SKILL.md"), []byte("user old"), 0o644); err != nil {
		t.Fatalf("write dest user: %v", err)
	}

	installed, err := runInstallSkills(source, dest)
	if err != nil {
		t.Fatalf("runInstallSkills: %v", err)
	}

	if len(installed) != 1 || installed[0] != "otto-orchestrate" {
		t.Fatalf("expected only otto-orchestrate installed, got %v", installed)
	}

	ottoBytes, _ := os.ReadFile(filepath.Join(dest, "otto-orchestrate", "SKILL.md"))
	userBytes, _ := os.ReadFile(filepath.Join(dest, "user-skill", "SKILL.md"))
	if string(ottoBytes) != "otto new" {
		t.Fatalf("expected otto skill overwritten, got %q", string(ottoBytes))
	}
	if string(userBytes) != "user old" {
		t.Fatalf("expected user skill preserved, got %q", string(userBytes))
	}
}
