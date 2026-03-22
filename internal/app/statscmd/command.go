package statscmd

import (
	"errors"
	"io"

	"github.com/spf13/cobra"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/app/graphhealth"
	"github.com/guillaume-galp/cge/internal/infra/kuzu"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

type StatsReader interface {
	Analyze(cmd *cobra.Command, workspace repo.Workspace) (graphhealth.Analysis, error)
}

type graphStatsReader struct {
	store *kuzu.Store
}

func (r graphStatsReader) Analyze(cmd *cobra.Command, workspace repo.Workspace) (graphhealth.Analysis, error) {
	if r.store == nil {
		r.store = kuzu.NewStore()
	}
	graph, err := r.store.ReadGraph(cmd.Context(), workspace)
	if err != nil {
		return graphhealth.Analysis{}, err
	}
	return graphhealth.AnalyzeGraph(graph)
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

			analysis, err := reader.Analyze(cmd, workspace)
			if err != nil {
				return handleStatsError(cmd.OutOrStdout(), output, err)
			}

			return cmdsupport.WriteSuccess(cmd.OutOrStdout(), output, "stats", resultEnvelope{
				Snapshot: snapshotCounts{
					Nodes:         analysis.Snapshot.Nodes,
					Relationships: analysis.Snapshot.Relationships,
				},
				Indicators: analysis.Indicators,
			})
		},
	}

	cmd.Flags().StringVar(&output, "output", "", "Write the structured JSON response to a file instead of stdout")

	return cmd
}

type resultEnvelope struct {
	Snapshot   snapshotCounts         `json:"snapshot"`
	Indicators graphhealth.Indicators `json:"indicators"`
}

type snapshotCounts struct {
	Nodes         int `json:"nodes"`
	Relationships int `json:"relationships"`
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
