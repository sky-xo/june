package scope

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RepoRoot returns git repo root or empty string if not in a git repo.
// For worktrees, this returns the main repository root, not the worktree directory.
func RepoRoot() string {
	// Use --git-common-dir to get the main repo's .git directory
	// This works for both regular repos and worktrees
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	gitCommonDir := strings.TrimSpace(string(out))

	// For regular repos: --git-common-dir returns ".git" (relative)
	// For worktrees: --git-common-dir returns absolute path like "/path/to/repo/.git"
	// We need to handle both cases

	if !filepath.IsAbs(gitCommonDir) {
		// Regular repo case: gitCommonDir is ".git"
		// Get the current directory as the repo root
		cmd = exec.Command("git", "rev-parse", "--show-toplevel")
		out, err = cmd.Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}

	// Worktree case: gitCommonDir is absolute path to main repo's .git
	// The parent of .git is the main repo root
	return filepath.Dir(gitCommonDir)
}

// BranchName returns current branch or empty string if not in a git repo.
func BranchName() string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// CurrentScope returns the scope for the current directory.
// It uses git to determine repo root and branch name.
// Falls back to using the current directory basename if git is unavailable.
// Defaults to "main" as the branch name if the branch cannot be determined.
func CurrentScope() string {
	repoRoot := RepoRoot()
	if repoRoot == "" {
		// Fallback: use current directory
		cwd, err := os.Getwd()
		if err != nil {
			return "unknown"
		}
		repoRoot = cwd
	}

	branch := BranchName()
	if branch == "" {
		// Default to "main" when branch cannot be determined
		branch = "main"
	}

	return Scope(repoRoot, branch)
}
