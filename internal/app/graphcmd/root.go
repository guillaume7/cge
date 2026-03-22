package graphcmd

import (
	"context"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/guillaume-galp/cge/internal/app/contextcmd"
	"github.com/guillaume-galp/cge/internal/app/diffcmd"
	"github.com/guillaume-galp/cge/internal/app/explaincmd"
	"github.com/guillaume-galp/cge/internal/app/hygienecmd"
	"github.com/guillaume-galp/cge/internal/app/initcmd"
	"github.com/guillaume-galp/cge/internal/app/querycmd"
	"github.com/guillaume-galp/cge/internal/app/statscmd"
	"github.com/guillaume-galp/cge/internal/app/writecmd"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func NewRootCommand(startDir string, manager *repo.Manager) *cobra.Command {
	if manager == nil {
		manager = repo.NewManager(repo.NewGitRepositoryLocator())
	}

	cmd := &cobra.Command{
		Use:           "graph",
		Short:         "Manage the repository-scoped cognitive graph workspace",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.AddCommand(
		initcmd.NewCommand(startDir, manager),
		writecmd.NewCommand(startDir, manager),
		querycmd.NewCommand(startDir, manager),
		contextcmd.NewCommand(startDir, manager),
		explaincmd.NewCommand(startDir, manager),
		diffcmd.NewCommand(startDir, manager),
		statscmd.NewCommand(startDir, manager),
		hygienecmd.NewCommand(startDir, manager),
	)
	return cmd
}

func Execute(ctx context.Context, args []string, startDir string, stdin io.Reader, stdout, stderr io.Writer) error {
	cmd := NewRootCommand(startDir, nil)
	cmd.SetArgs(args)
	if stdin == nil {
		stdin = os.Stdin
	}
	cmd.SetIn(stdin)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	return cmd.ExecuteContext(ctx)
}
