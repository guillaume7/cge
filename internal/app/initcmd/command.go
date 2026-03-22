package initcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func NewCommand(startDir string, manager *repo.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a repo-local graph workspace",
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := manager.InitWorkspace(cmd.Context(), startDir)
			if err != nil {
				return err
			}

			if result.AlreadyExists {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "graph workspace already exists at %s\n", result.WorkspacePath)
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "initialized graph workspace at %s\n", result.WorkspacePath)
			return err
		},
	}

	return cmd
}
