package contextcmd

import (
	"errors"
	"io"

	"github.com/spf13/cobra"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/app/contextprojector"
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
	return newCommand(startDir, manager, graphQuerier{engine: retrieval.NewEngine(nil, nil)}, contextprojector.NewProjector())
}

func newCommand(startDir string, manager *repo.Manager, querier Querier, projector contextprojector.Projector) *cobra.Command {
	if querier == nil {
		querier = graphQuerier{engine: retrieval.NewEngine(nil, nil)}
	}

	var task string
	var file string
	var maxTokens int
	var output string

	cmd := &cobra.Command{
		Use:           "context",
		Short:         "Project prompt-ready context from the repo-local graph",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, err := cmdsupport.RequireWorkspace(cmd, startDir, manager)
			if err != nil {
				return handleContextError(cmd.OutOrStdout(), output, err)
			}

			input, err := cmdsupport.ResolveTextInput(cmd.InOrStdin(), task, file, "task", "--task")
			if err != nil {
				return handleContextError(cmd.OutOrStdout(), output, err)
			}

			if err := contextprojector.ValidateMaxTokens(maxTokens); err != nil {
				return handleContextError(cmd.OutOrStdout(), output, err)
			}

			results, err := querier.Query(cmd, workspace, input.Value)
			if err != nil {
				return handleContextError(cmd.OutOrStdout(), output, err)
			}

			contextEnvelope, err := projector.Project(results, maxTokens)
			if err != nil {
				return handleContextError(cmd.OutOrStdout(), output, err)
			}

			response := resultEnvelope{
				Query: queryEnvelope{
					Task:   input.Value,
					Source: input.Source,
				},
				Index:   indexEnvelope{Status: results.IndexStatus},
				Context: contextEnvelope,
			}

			return cmdsupport.WriteSuccess(cmd.OutOrStdout(), output, "context", response)
		},
	}

	cmd.Flags().StringVar(&task, "task", "", "Task text to project into compact context")
	cmd.Flags().StringVar(&file, "file", "", "File containing the task text")
	cmd.Flags().IntVar(&maxTokens, "max-tokens", 1200, "Maximum approximate token budget for projected context")
	cmd.Flags().StringVar(&output, "output", "", "Write the structured JSON response to a file instead of stdout")

	return cmd
}

type resultEnvelope struct {
	Query   queryEnvelope             `json:"query"`
	Index   indexEnvelope             `json:"index"`
	Context contextprojector.Envelope `json:"context"`
}

type queryEnvelope struct {
	Task   string `json:"task"`
	Source string `json:"source"`
}

type indexEnvelope struct {
	Status string `json:"status"`
}

func handleContextError(w io.Writer, outputPath string, err error) error {
	if detail, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return cmdsupport.WriteFailure(w, outputPath, "context", detail, err)
	}

	var validationErr *contextprojector.ValidationError
	if errors.As(err, &validationErr) {
		return cmdsupport.WriteFailure(w, outputPath, "context", cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "validation_error",
			Code:     validationErr.Code,
			Message:  validationErr.Message,
			Details:  validationErr.Details,
		}, err)
	}

	var indexErr *textindex.Error
	if errors.As(err, &indexErr) {
		return cmdsupport.WriteFailure(w, outputPath, "context", cmdsupport.ErrorDetail{
			Category: "operational_error",
			Type:     "index_error",
			Code:     indexErr.Code,
			Message:  indexErr.Message,
			Details:  indexErr.Details,
		}, err)
	}

	return cmdsupport.WriteFailure(w, outputPath, "context", cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "context_error",
		Code:     "context_failed",
		Message:  "graph context projection failed",
		Details: map[string]any{
			"reason": err.Error(),
		},
	}, err)
}
