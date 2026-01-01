package cli

import (
	"os"

	"june/internal/cli/commands"

	"github.com/spf13/cobra"
)

func Execute() {
	rootCmd := &cobra.Command{
		Use:   "june",
		Short: "Subagent viewer for Claude Code",
		RunE: func(cmd *cobra.Command, args []string) error {
			return commands.RunWatch()
		},
	}

	rootCmd.AddCommand(commands.NewWatchCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
