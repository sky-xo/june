package cli

import "testing"

func TestVersion(t *testing.T) {
	// Save original values to restore after tests
	origVersion := version
	origCommit := commit
	defer func() {
		version = origVersion
		commit = origCommit
	}()

	tests := []struct {
		name        string
		version     string
		commit      string
		wantContain string
		wantExact   string
	}{
		{
			name:      "tagged release version",
			version:   "v0.2.0",
			commit:    "abc1234",
			wantExact: "v0.2.0",
		},
		{
			name:      "git describe version",
			version:   "v0.2.0-41-g72507bd",
			commit:    "72507bd",
			wantExact: "v0.2.0-41-g72507bd",
		},
		{
			name:      "dirty working tree",
			version:   "v0.2.0-41-g72507bd-dirty",
			commit:    "72507bd",
			wantExact: "v0.2.0-41-g72507bd-dirty",
		},
		{
			name:      "dev version with commit",
			version:   "dev",
			commit:    "abc1234",
			wantExact: "dev (abc1234)",
		},
		{
			name:      "dev version with unknown commit",
			version:   "dev",
			commit:    "unknown",
			wantExact: "dev",
		},
		{
			name:      "dev version without commit",
			version:   "dev",
			commit:    "",
			wantExact: "dev",
		},
		{
			name:        "empty version falls back",
			version:     "",
			commit:      "",
			wantContain: "", // Will be "dev" or module version
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version = tt.version
			commit = tt.commit

			got := Version()

			if tt.wantExact != "" && got != tt.wantExact {
				t.Errorf("Version() = %q, want %q", got, tt.wantExact)
			}
			if tt.wantContain != "" && got != tt.wantContain {
				// For empty version case, we just check it returns something
				if got == "" {
					t.Errorf("Version() returned empty string")
				}
			}
		})
	}
}

func TestVersionWithEmptyFallsToDev(t *testing.T) {
	// Save original values
	origVersion := version
	origCommit := commit
	defer func() {
		version = origVersion
		commit = origCommit
	}()

	// When both are empty and no module info, should return "dev"
	version = ""
	commit = ""

	got := Version()
	// It will either return "dev" or a module version - both are valid
	if got == "" {
		t.Error("Version() should not return empty string")
	}
}
