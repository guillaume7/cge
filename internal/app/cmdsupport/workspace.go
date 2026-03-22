package cmdsupport

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func RequireWorkspace(cmd *cobra.Command, startDir string, manager *repo.Manager) (repo.Workspace, error) {
	workspace, err := manager.OpenWorkspace(cmd.Context(), startDir)
	if err == nil {
		return workspace, nil
	}

	detail := ErrorDetail{
		Category: "operational_error",
		Type:     "workspace_error",
		Code:     "workspace_open_failed",
		Message:  "graph workspace could not be opened",
		Details: map[string]any{
			"reason": err.Error(),
		},
	}

	switch {
	case errors.Is(err, repo.ErrRepositoryRootNotFound):
		detail.Code = "repository_root_not_found"
		detail.Message = "repository root could not be determined"
	case errors.Is(err, repo.ErrWorkspaceNotInitialized):
		detail.Code = "workspace_not_initialized"
		detail.Message = "graph workspace has not been initialized"
		detail.Details["hint"] = `run "graph init" first`
	}

	return repo.Workspace{}, NewCommandError(detail, err)
}
