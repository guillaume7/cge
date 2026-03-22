package statscmd

import (
	"errors"
	"io"

	"github.com/spf13/cobra"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/infra/kuzu"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

type StatsReader interface {
	Stats(cmd *cobra.Command, workspace repo.Workspace) (kuzu.GraphStats, error)
}

type graphStatsReader struct {
	store *kuzu.Store
}

func (r graphStatsReader) Stats(cmd *cobra.Command, workspace repo.Workspace) (kuzu.GraphStats, error) {
	if r.store == nil {
		r.store = kuzu.NewStore()
	}
	return r.store.Stats(cmd.Context(), workspace)
}

func NewCommand(startDir string, manager *repo.Manager) *cobra.Command {
	return newCommand(startDir, manager, graphStatsReader{store: kuzu.NewStore()})
}

func newCommand(startDir string, manager *repo.Manager, reader StatsReader) *cobra.Command {
	if reader == nil {
		reader = graphStatsReader{store: kuzu.NewStore()}
	}

	var output string

	cmd := &cobra.Command{
		Use:           "stats",
		Short:         "Show graph statistics for the repo-local workspace",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, err := cmdsupport.RequireWorkspace(cmd, startDir, manager)
			if err != nil {
				return handleStatsError(cmd.OutOrStdout(), output, err)
			}

			stats, err := reader.Stats(cmd, workspace)
			if err != nil {
				return handleStatsError(cmd.OutOrStdout(), output, err)
			}

			return cmdsupport.WriteSuccess(cmd.OutOrStdout(), output, "stats", resultEnvelope{
				Snapshot: stats,
			})
		},
	}

	cmd.Flags().StringVar(&output, "output", "", "Write the structured JSON response to a file instead of stdout")

	return cmd
}

type resultEnvelope struct {
	Snapshot kuzu.GraphStats `json:"snapshot"`
}

func handleStatsError(w io.Writer, outputPath string, err error) error {
	if detail, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return cmdsupport.WriteFailure(w, outputPath, "stats", detail, err)
	}

	var persistenceErr *kuzu.PersistenceError
	if errors.As(err, &persistenceErr) {
		return cmdsupport.WriteFailure(w, outputPath, "stats", cmdsupport.ErrorDetail{
			Category: "operational_error",
			Type:     "persistence_error",
			Code:     persistenceErr.Code,
			Message:  persistenceErr.Message,
			Details:  persistenceErr.Details,
		}, err)
	}

	return cmdsupport.WriteFailure(w, outputPath, "stats", cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "stats_error",
		Code:     "stats_failed",
		Message:  "graph stats failed",
		Details: map[string]any{
			"reason": err.Error(),
		},
	}, err)
}
