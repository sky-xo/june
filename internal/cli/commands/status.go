package commands

import (
	"database/sql"
	"fmt"

	"otto/internal/repo"

	"github.com/spf13/cobra"
)

func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "List agents and their statuses",
		RunE: func(cmd *cobra.Command, args []string) error {
			conn, err := openDB()
			if err != nil {
				return err
			}
			defer conn.Close()

			return runStatus(conn)
		},
	}
	return cmd
}

func runStatus(db *sql.DB) error {
	agents, err := repo.ListAgents(db)
	if err != nil {
		return err
	}

	for _, a := range agents {
		fmt.Printf("%s [%s]: %s - %s\n", a.ID, a.Type, a.Status, a.Task)
	}

	return nil
}
