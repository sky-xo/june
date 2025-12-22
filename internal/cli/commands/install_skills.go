package commands

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"otto/internal/scope"

	"github.com/spf13/cobra"
)

func NewInstallSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install-skills",
		Short: "Install bundled Otto skills into ~/.claude/skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			sourceRoot := scope.RepoRoot()
			if sourceRoot == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				sourceRoot = cwd
			}
			source := filepath.Join(sourceRoot, "skills")

			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			dest := filepath.Join(home, ".claude", "skills")

			installed, err := runInstallSkills(source, dest)
			if err != nil {
				return err
			}

			if len(installed) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No skills installed")
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Installed skills (%d): %s\n", len(installed), strings.Join(installed, ", "))
			return nil
		},
	}
	return cmd
}

func runInstallSkills(source, dest string) ([]string, error) {
	entries, err := os.ReadDir(source)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return nil, err
	}

	var installed []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		destPath := filepath.Join(dest, name)
		if _, err := os.Stat(destPath); err == nil {
			if !strings.HasPrefix(name, "otto-") {
				continue
			}
			if err := os.RemoveAll(destPath); err != nil {
				return nil, err
			}
		}

		sourcePath := filepath.Join(source, name)
		if err := copyDir(sourcePath, destPath); err != nil {
			return nil, err
		}
		installed = append(installed, name)
	}

	return installed, nil
}

func copyDir(source, dest string) error {
	return filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		outPath := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(outPath, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			return err
		}
		defer out.Close()

		if _, err := io.Copy(out, in); err != nil {
			return err
		}
		return nil
	})
}
