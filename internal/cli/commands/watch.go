package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"june/internal/claude"
	"june/internal/scope"
	"june/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewWatchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "watch",
		Short: "Watch subagent activity",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunWatch()
		},
	}
}

// RunWatch starts the TUI.
func RunWatch() error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("watch requires a terminal")
	}

	// Get current git project root
	repoRoot := scope.RepoRoot()
	if repoRoot == "" {
		return fmt.Errorf("not in a git repository")
	}

	// Find Claude project directory
	absPath, err := filepath.Abs(repoRoot)
	if err != nil {
		return err
	}
	projectDir := claude.ProjectDir(absPath)

	// Check if directory exists
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return fmt.Errorf("no Claude Code sessions found for this project\n\nExpected: %s", projectDir)
	}

	// Run TUI
	model := tui.NewModel(projectDir)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()
	return err
}
