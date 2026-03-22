package querycmd

import (
	"errors"
	"io"

	"github.com/spf13/cobra"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/app/retrieval"
	"github.com/guillaume-galp/cge/internal/infra/repo"
	"github.com/guillaume-galp/cge/internal/infra/textindex"
)

type Querier interface {
	Query(cmd *cobra.Command, workspace repo.Workspace, task string) (retrieval.ResultSet, error)
}

type graphQuerier struct {
	engine *retrieval.Engine
}

func (q graphQuerier) Query(cmd *cobra.Command, workspace repo.Workspace, task string) (retrieval.ResultSet, error) {
	if q.engine == nil {
		q.engine = retrieval.NewEngine(nil, nil)
	}
	return q.engine.Query(cmd.Context(), workspace, task)
}

func NewCommand(startDir string, manager *repo.Manager) *cobra.Command {
	return newCommand(startDir, manager, graphQuerier{engine: retrieval.NewEngine(nil, nil)})
}

func newCommand(startDir string, manager *repo.Manager, querier Querier) *cobra.Command {
	if querier == nil {
		querier = graphQuerier{engine: retrieval.NewEngine(nil, nil)}
	}

	var task string
	var file string
	var output string

	cmd := &cobra.Command{
		Use:           "query",
		Short:         "Query the repo-local graph workspace",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, err := cmdsupport.RequireWorkspace(cmd, startDir, manager)
			if err != nil {
				return handleQueryError(cmd.OutOrStdout(), output, err)
			}

			input, err := cmdsupport.ResolveTextInput(cmd.InOrStdin(), task, file, "task", "--task")
			if err != nil {
				return handleQueryError(cmd.OutOrStdout(), output, err)
			}

			result, err := querier.Query(cmd, workspace, input.Value)
			if err != nil {
				return handleQueryError(cmd.OutOrStdout(), output, err)
			}

			response := resultEnvelope{
				Query: queryEnvelope{
					Task:   input.Value,
					Source: input.Source,
				},
				Index: indexEnvelope{
					Status: result.IndexStatus,
				},
				Results: result.Results,
			}

			return cmdsupport.WriteSuccess(cmd.OutOrStdout(), output, "query", response)
		},
	}

	cmd.Flags().StringVar(&task, "task", "", "Task text to run against the graph")
	cmd.Flags().StringVar(&file, "file", "", "File containing the task text")
	cmd.Flags().StringVar(&output, "output", "", "Write the structured JSON response to a file instead of stdout")

	return cmd
}

type resultEnvelope struct {
	Query   queryEnvelope      `json:"query"`
	Index   indexEnvelope      `json:"index"`
	Results []retrieval.Result `json:"results"`
}

type queryEnvelope struct {
	Task   string `json:"task"`
	Source string `json:"source"`
}

type indexEnvelope struct {
	Status string `json:"status"`
}

func handleQueryError(w io.Writer, outputPath string, err error) error {
	if detail, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return cmdsupport.WriteFailure(w, outputPath, "query", detail, err)
	}

	var indexErr *textindex.Error
	if errors.As(err, &indexErr) {
		return cmdsupport.WriteFailure(w, outputPath, "query", cmdsupport.ErrorDetail{
			Category: "operational_error",
			Type:     "index_error",
			Code:     indexErr.Code,
			Message:  indexErr.Message,
			Details:  indexErr.Details,
		}, err)
	}

	return cmdsupport.WriteFailure(w, outputPath, "query", cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "query_error",
		Code:     "query_failed",
		Message:  "graph query failed",
		Details: map[string]any{
			"reason": err.Error(),
		},
	}, err)
}
