package commands

import (
	"fmt"

	"github.com/spf13/cobra"
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
	fmt.Println("TODO: Implement subagent viewer TUI")
	return nil
}
