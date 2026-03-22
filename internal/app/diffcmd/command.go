package diffcmd

import (
	"errors"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/infra/kuzu"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

type Differ interface {
	Diff(cmd *cobra.Command, workspace repo.Workspace, from, to string) (kuzu.GraphDiff, error)
}

type graphDiffer struct {
	store *kuzu.Store
}

func (d graphDiffer) Diff(cmd *cobra.Command, workspace repo.Workspace, from, to string) (kuzu.GraphDiff, error) {
	if d.store == nil {
		d.store = kuzu.NewStore()
	}
	return d.store.Diff(cmd.Context(), workspace, from, to)
}

func NewCommand(startDir string, manager *repo.Manager) *cobra.Command {
	return newCommand(startDir, manager, graphDiffer{store: kuzu.NewStore()})
}

func newCommand(startDir string, manager *repo.Manager, differ Differ) *cobra.Command {
	if differ == nil {
		differ = graphDiffer{store: kuzu.NewStore()}
	}

	var from string
	var to string
	var output string

	cmd := &cobra.Command{
		Use:           "diff",
		Short:         "Diff graph revisions in the repo-local workspace",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, err := cmdsupport.RequireWorkspace(cmd, startDir, manager)
			if err != nil {
				return handleDiffError(cmd.OutOrStdout(), output, err)
			}

			missingFlags := []string{}
			if strings.TrimSpace(from) == "" {
				missingFlags = append(missingFlags, "--from")
			}
			if strings.TrimSpace(to) == "" {
				missingFlags = append(missingFlags, "--to")
			}
			if len(missingFlags) > 0 {
				return handleDiffError(cmd.OutOrStdout(), output, cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
					Category: "validation_error",
					Type:     "input_error",
					Code:     "missing_revision_anchors",
					Message:  "graph diff requires both revision anchors",
					Details: map[string]any{
						"missing_flags": missingFlags,
					},
				}, errors.New("graph diff requires both --from and --to")))
			}

			result, err := differ.Diff(cmd, workspace, strings.TrimSpace(from), strings.TrimSpace(to))
			if err != nil {
				return handleDiffError(cmd.OutOrStdout(), output, err)
			}

			return cmdsupport.WriteSuccess(cmd.OutOrStdout(), output, "diff", resultEnvelope{Diff: result})
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Older revision anchor to compare")
	cmd.Flags().StringVar(&to, "to", "", "Newer revision anchor to compare")
	cmd.Flags().StringVar(&output, "output", "", "Write the structured JSON response to a file instead of stdout")

	return cmd
}

type resultEnvelope struct {
	Diff kuzu.GraphDiff `json:"diff"`
}

func handleDiffError(w io.Writer, outputPath string, err error) error {
	if detail, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return cmdsupport.WriteFailure(w, outputPath, "diff", detail, err)
	}

	var diffErr *kuzu.DiffError
	if errors.As(err, &diffErr) {
		return cmdsupport.WriteFailure(w, outputPath, "diff", cmdsupport.ErrorDetail{
			Category: "operational_error",
			Type:     "diff_error",
			Code:     diffErr.Code,
			Message:  diffErr.Message,
			Details:  diffErr.Details,
		}, err)
	}

	return cmdsupport.WriteFailure(w, outputPath, "diff", cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "diff_error",
		Code:     "diff_failed",
		Message:  "graph diff failed",
		Details: map[string]any{
			"reason": err.Error(),
		},
	}, err)
}
