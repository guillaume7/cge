package workflowcmd

import (
	"context"
	"errors"
	"io"

	"github.com/spf13/cobra"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/app/contextprojector"
	"github.com/guillaume-galp/cge/internal/app/workflow"
	"github.com/guillaume-galp/cge/internal/infra/repo"
	"github.com/guillaume-galp/cge/internal/infra/textindex"
)

type Initializer interface {
	Init(ctx context.Context, startDir string) (workflow.InitResult, error)
}

type Starter interface {
	Start(ctx context.Context, startDir, task string, maxTokens int) (workflow.StartResult, error)
}

type AdvancedStarter interface {
	StartWithOptions(ctx context.Context, startDir, task string, maxTokens int, options workflow.StartOptions) (workflow.StartResult, error)
}

type Finisher interface {
	Finish(ctx context.Context, startDir, input string) (workflow.FinishResult, error)
}

type Benchmarker interface {
	SummarizeBenchmark(ctx context.Context, startDir, scenarioID string) (workflow.BenchmarkSummaryResult, error)
}

func NewCommand(startDir string, manager *repo.Manager) *cobra.Command {
	return newCommand(startDir, workflow.NewService(manager))
}

func newCommand(startDir string, initializer Initializer) *cobra.Command {
	if initializer == nil {
		initializer = workflow.NewService(nil)
	}
	starter, ok := initializer.(Starter)
	if !ok {
		starter = workflow.NewService(nil)
	}
	finisher, ok := initializer.(Finisher)
	if !ok {
		finisher = workflow.NewService(nil)
	}
	benchmarker, ok := initializer.(Benchmarker)
	if !ok {
		benchmarker = workflow.NewService(nil)
	}

	cmd := &cobra.Command{
		Use:           "workflow",
		Short:         "Manage delegated workflow bootstrap and orchestration state",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.AddCommand(newInitCommand(startDir, initializer))
	cmd.AddCommand(newStartCommand(startDir, starter))
	cmd.AddCommand(newFinishCommand(startDir, finisher))
	cmd.AddCommand(newBenchmarkCommand(startDir, benchmarker))
	return cmd
}

func newInitCommand(startDir string, initializer Initializer) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:           "init",
		Short:         "Initialize or refresh repo-local delegated workflow state",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := initializer.Init(cmd.Context(), startDir)
			if err != nil {
				return handleInitError(cmd.OutOrStdout(), output, err)
			}
			return cmdsupport.WriteSuccess(cmd.OutOrStdout(), output, "workflow.init", result)
		},
	}

	cmd.Flags().StringVar(&output, "output", "", "Write the structured JSON response to a file instead of stdout")
	return cmd
}

func handleInitError(w io.Writer, outputPath string, err error) error {
	if detail, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return cmdsupport.WriteFailure(w, outputPath, "workflow.init", detail, err)
	}

	detail := cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "workflow_error",
		Code:     "workflow_init_failed",
		Message:  "workflow bootstrap failed",
		Details: map[string]any{
			"reason": err.Error(),
		},
	}

	switch {
	case errors.Is(err, repo.ErrRepositoryRootNotFound):
		detail.Type = "workspace_error"
		detail.Code = "repository_root_not_found"
		detail.Message = "repository root could not be determined"
	case errors.Is(err, repo.ErrWorkspaceNotInitialized):
		detail.Type = "workspace_error"
		detail.Code = "workspace_not_initialized"
		detail.Message = "graph workspace has not been initialized"
		detail.Details["hint"] = `run "graph init" first`
	}

	return cmdsupport.WriteFailure(w, outputPath, "workflow.init", detail, err)
}

func newStartCommand(startDir string, starter Starter) *cobra.Command {
	if starter == nil {
		starter = workflow.NewService(nil)
	}

	var task string
	var file string
	var output string
	var maxTokens int
	var kickoffMode string

	cmd := &cobra.Command{
		Use:           "start",
		Short:         "Inspect delegated-workflow readiness and recommend the next action",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			input, err := cmdsupport.ResolveTextInput(cmd.InOrStdin(), task, file, "task", "--task")
			if err != nil {
				return handleStartError(cmd.OutOrStdout(), output, err)
			}

			var result workflow.StartResult
			var startErr error
			if advanced, ok := starter.(AdvancedStarter); ok {
				result, startErr = advanced.StartWithOptions(cmd.Context(), startDir, input.Value, maxTokens, workflow.StartOptions{
					KickoffMode: kickoffMode,
				})
			} else {
				result, startErr = starter.Start(cmd.Context(), startDir, input.Value, maxTokens)
			}
			if startErr != nil {
				return handleStartError(cmd.OutOrStdout(), output, startErr)
			}

			return cmdsupport.WriteSuccess(cmd.OutOrStdout(), output, "workflow.start", startResultEnvelope{
				Task: taskEnvelope{
					Value:  input.Value,
					Source: input.Source,
				},
				Recommendation: result.Recommendation,
				Readiness:      result.Readiness,
				Kickoff:        result.Kickoff,
			})
		},
	}

	cmd.Flags().StringVar(&task, "task", "", "Task text for the delegated workflow kickoff")
	cmd.Flags().StringVar(&file, "file", "", "File containing the task text")
	cmd.Flags().IntVar(&maxTokens, "max-tokens", 1200, "Maximum approximate token budget for projected kickoff context")
	cmd.Flags().StringVar(&kickoffMode, "kickoff-mode", workflow.KickoffModeAuto, "Kickoff mode override: auto, minimal, or none")
	cmd.Flags().StringVar(&output, "output", "", "Write the structured JSON response to a file instead of stdout")
	return cmd
}

type startResultEnvelope struct {
	Task           taskEnvelope             `json:"task"`
	Recommendation string                   `json:"recommendation"`
	Readiness      workflow.ReadinessState  `json:"readiness"`
	Kickoff        workflow.KickoffEnvelope `json:"kickoff"`
}

type taskEnvelope struct {
	Value  string `json:"value"`
	Source string `json:"source"`
}

func handleStartError(w io.Writer, outputPath string, err error) error {
	if detail, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return cmdsupport.WriteFailure(w, outputPath, "workflow.start", detail, err)
	}

	var validationErr *contextprojector.ValidationError
	if errors.As(err, &validationErr) {
		return cmdsupport.WriteFailure(w, outputPath, "workflow.start", cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "validation_error",
			Code:     validationErr.Code,
			Message:  validationErr.Message,
			Details:  validationErr.Details,
		}, err)
	}

	var indexErr *textindex.Error
	if errors.As(err, &indexErr) {
		return cmdsupport.WriteFailure(w, outputPath, "workflow.start", cmdsupport.ErrorDetail{
			Category: "operational_error",
			Type:     "index_error",
			Code:     indexErr.Code,
			Message:  indexErr.Message,
			Details:  indexErr.Details,
		}, err)
	}

	detail := cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "workflow_error",
		Code:     "workflow_start_failed",
		Message:  "workflow readiness inspection failed",
		Details: map[string]any{
			"reason": err.Error(),
		},
	}

	switch {
	case errors.Is(err, repo.ErrRepositoryRootNotFound):
		detail.Type = "workspace_error"
		detail.Code = "repository_root_not_found"
		detail.Message = "repository root could not be determined"
	case errors.Is(err, repo.ErrWorkspaceNotInitialized):
		detail.Type = "workspace_error"
		detail.Code = "workspace_not_initialized"
		detail.Message = "graph workspace has not been initialized"
		detail.Details["hint"] = `run "graph init" first`
	}

	return cmdsupport.WriteFailure(w, outputPath, "workflow.start", detail, err)
}

func newFinishCommand(startDir string, finisher Finisher) *cobra.Command {
	if finisher == nil {
		finisher = workflow.NewService(nil)
	}

	var inlinePayload string
	var file string
	var output string

	cmd := &cobra.Command{
		Use:           "finish",
		Short:         "Persist delegated-task outcomes and return a handoff envelope",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			input, err := cmdsupport.ResolveTextInput(cmd.InOrStdin(), inlinePayload, file, "payload", "--payload")
			if err != nil {
				return handleFinishError(cmd.OutOrStdout(), output, err)
			}

			result, err := finisher.Finish(cmd.Context(), startDir, input.Value)
			if err != nil {
				return handleFinishError(cmd.OutOrStdout(), output, err)
			}

			return cmdsupport.WriteSuccess(cmd.OutOrStdout(), output, "workflow.finish", result)
		},
	}

	cmd.Flags().StringVar(&inlinePayload, "payload", "", "Inline delegated-task outcome payload")
	cmd.Flags().StringVar(&file, "file", "", "File containing the delegated-task outcome payload")
	cmd.Flags().StringVar(&output, "output", "", "Write the structured JSON response to a file instead of stdout")
	return cmd
}

func handleFinishError(w io.Writer, outputPath string, err error) error {
	if detail, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return cmdsupport.WriteFailure(w, outputPath, "workflow.finish", detail, err)
	}

	detail := cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "workflow_error",
		Code:     "workflow_finish_failed",
		Message:  "workflow finish failed",
		Details: map[string]any{
			"reason": err.Error(),
		},
	}

	switch {
	case errors.Is(err, repo.ErrRepositoryRootNotFound):
		detail.Type = "workspace_error"
		detail.Code = "repository_root_not_found"
		detail.Message = "repository root could not be determined"
	case errors.Is(err, repo.ErrWorkspaceNotInitialized):
		detail.Type = "workspace_error"
		detail.Code = "workspace_not_initialized"
		detail.Message = "graph workspace has not been initialized"
		detail.Details["hint"] = `run "graph init" first`
	}

	return cmdsupport.WriteFailure(w, outputPath, "workflow.finish", detail, err)
}

func newBenchmarkCommand(startDir string, benchmarker Benchmarker) *cobra.Command {
	if benchmarker == nil {
		benchmarker = workflow.NewService(nil)
	}

	var scenarioID string
	var output string

	cmd := &cobra.Command{
		Use:           "benchmark",
		Short:         "Summarize delegated-workflow benchmark artifacts from local reports",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := benchmarker.SummarizeBenchmark(cmd.Context(), startDir, scenarioID)
			if err != nil {
				return handleBenchmarkError(cmd.OutOrStdout(), output, err)
			}
			return cmdsupport.WriteSuccess(cmd.OutOrStdout(), output, "workflow.benchmark", result)
		},
	}

	cmd.Flags().StringVar(&scenarioID, "scenario", "", "Summarize a single benchmark scenario by scenario_id")
	cmd.Flags().StringVar(&output, "output", "", "Write the structured JSON response to a file instead of stdout")
	return cmd
}

func handleBenchmarkError(w io.Writer, outputPath string, err error) error {
	if detail, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return cmdsupport.WriteFailure(w, outputPath, "workflow.benchmark", detail, err)
	}

	detail := cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "benchmark_error",
		Code:     "workflow_benchmark_failed",
		Message:  "workflow benchmark summary failed",
		Details: map[string]any{
			"reason": err.Error(),
		},
	}

	switch {
	case errors.Is(err, repo.ErrRepositoryRootNotFound):
		detail.Type = "workspace_error"
		detail.Code = "repository_root_not_found"
		detail.Message = "repository root could not be determined"
	case errors.Is(err, repo.ErrWorkspaceNotInitialized):
		detail.Type = "workspace_error"
		detail.Code = "workspace_not_initialized"
		detail.Message = "graph workspace has not been initialized"
		detail.Details["hint"] = `run "graph init" first`
	}

	return cmdsupport.WriteFailure(w, outputPath, "workflow.benchmark", detail, err)
}
