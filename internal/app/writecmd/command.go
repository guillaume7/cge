package writecmd

import (
	"errors"
	"io"

	"github.com/spf13/cobra"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	graphpayload "github.com/guillaume-galp/cge/internal/domain/payload"
	"github.com/guillaume-galp/cge/internal/infra/kuzu"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

type Persister interface {
	Write(cmd *cobra.Command, workspace repo.Workspace, envelope graphpayload.Envelope) (kuzu.WriteSummary, error)
}

type graphPersister struct {
	store *kuzu.Store
}

func (p graphPersister) Write(cmd *cobra.Command, workspace repo.Workspace, envelope graphpayload.Envelope) (kuzu.WriteSummary, error) {
	if p.store == nil {
		p.store = kuzu.NewStore()
	}
	return p.store.Write(cmd.Context(), workspace, envelope)
}

func NewCommand(startDir string, manager *repo.Manager) *cobra.Command {
	return newCommand(startDir, manager, graphPersister{store: kuzu.NewStore()})
}

func newCommand(startDir string, manager *repo.Manager, persister Persister) *cobra.Command {
	if persister == nil {
		persister = graphPersister{store: kuzu.NewStore()}
	}

	var inlinePayload string
	var file string
	var output string

	cmd := &cobra.Command{
		Use:           "write",
		Short:         "Write graph payloads into the repo-local workspace",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, err := cmdsupport.RequireWorkspace(cmd, startDir, manager)
			if err != nil {
				return handleWriteError(cmd.OutOrStdout(), output, err)
			}

			input, err := cmdsupport.ResolveTextInput(cmd.InOrStdin(), inlinePayload, file, "payload", "--payload")
			if err != nil {
				return handleWriteError(cmd.OutOrStdout(), output, err)
			}

			envelope, err := graphpayload.ParseAndValidate(input.Value)
			if err != nil {
				return handleWriteError(cmd.OutOrStdout(), output, err)
			}

			summary, err := persister.Write(cmd, workspace, envelope)
			if err != nil {
				return handleWriteError(cmd.OutOrStdout(), output, err)
			}

			response := resultEnvelope{
				Summary: summary,
			}

			return cmdsupport.WriteSuccess(cmd.OutOrStdout(), output, "write", response)
		},
	}

	cmd.Flags().StringVar(&inlinePayload, "payload", "", "Inline graph payload to write")
	cmd.Flags().StringVar(&file, "file", "", "File containing the graph payload")
	cmd.Flags().StringVar(&output, "output", "", "Write the structured JSON response to a file instead of stdout")

	return cmd
}

type resultEnvelope struct {
	Summary kuzu.WriteSummary `json:"summary"`
}

func handleWriteError(w io.Writer, outputPath string, err error) error {
	if detail, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return cmdsupport.WriteFailure(w, outputPath, "write", detail, err)
	}

	var validationErr *graphpayload.ValidationError
	if errors.As(err, &validationErr) {
		return cmdsupport.WriteFailure(w, outputPath, "write", cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "validation_error",
			Code:     validationErr.Code,
			Message:  validationErr.Message,
			Details:  validationErr.Details,
		}, err)
	}

	var persistenceErr *kuzu.PersistenceError
	if errors.As(err, &persistenceErr) {
		return cmdsupport.WriteFailure(w, outputPath, "write", cmdsupport.ErrorDetail{
			Category: "operational_error",
			Type:     "persistence_error",
			Code:     persistenceErr.Code,
			Message:  persistenceErr.Message,
			Details:  persistenceErr.Details,
		}, err)
	}

	return cmdsupport.WriteFailure(w, outputPath, "write", cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "write_error",
		Code:     "write_failed",
		Message:  "graph write failed",
		Details: map[string]any{
			"reason": err.Error(),
		},
	}, err)
}
