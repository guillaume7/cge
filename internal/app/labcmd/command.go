package labcmd

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/app/lab"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

type Initializer interface {
	Init(ctx context.Context, startDir string) (lab.InitResult, error)
}

type Runner interface {
	Run(ctx context.Context, startDir string, request lab.RunRequest) (lab.RunResult, error)
}

type Evaluator interface {
	Evaluate(ctx context.Context, startDir string, record lab.EvaluationRecord) (lab.EvaluateResult, error)
}

type EvaluationPresenter interface {
	PresentEvaluationInput(ctx context.Context, startDir string, request lab.PresentEvaluationRequest) (lab.PresentEvaluationResult, error)
}

type Reporter interface {
	Report(ctx context.Context, startDir string, request lab.ReportRequest) (lab.ReportResult, error)
}

func NewCommand(startDir string, manager *repo.Manager) *cobra.Command {
	return newCommand(startDir, lab.NewService(manager))
}

func newCommand(startDir string, initializer Initializer) *cobra.Command {
	if initializer == nil {
		initializer = lab.NewService(nil)
	}
	runner, ok := initializer.(Runner)
	if !ok {
		runner = lab.NewService(nil)
	}
	evaluator, ok := initializer.(Evaluator)
	if !ok {
		evaluator = lab.NewService(nil)
	}
	presenter, ok := initializer.(EvaluationPresenter)
	if !ok {
		presenter = lab.NewService(nil)
	}
	reporter, ok := initializer.(Reporter)
	if !ok {
		reporter = lab.NewService(nil)
	}

	cmd := &cobra.Command{
		Use:           "lab",
		Short:         "Manage repo-local experiment lab scaffolding",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.AddCommand(newInitCommand(startDir, initializer))
	cmd.AddCommand(newRunCommand(startDir, runner))
	cmd.AddCommand(newEvaluateCommand(startDir, evaluator, presenter))
	cmd.AddCommand(newReportCommand(startDir, reporter))
	return cmd
}

func newInitCommand(startDir string, initializer Initializer) *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:           "init",
		Short:         "Initialize or refresh repo-local experiment lab scaffolding",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := initializer.Init(cmd.Context(), startDir)
			if err != nil {
				return handleInitError(cmd.OutOrStdout(), output, err)
			}
			return cmdsupport.WriteSuccess(cmd.OutOrStdout(), output, "lab.init", result)
		},
	}

	cmd.Flags().StringVar(&output, "output", "", "Write the structured JSON response to a file instead of stdout")
	return cmd
}

func handleInitError(w io.Writer, outputPath string, err error) error {
	if detail, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return cmdsupport.WriteFailure(w, outputPath, "lab.init", detail, err)
	}

	detail := cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "lab_error",
		Code:     "lab_init_failed",
		Message:  "lab bootstrap failed",
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

	return cmdsupport.WriteFailure(w, outputPath, "lab.init", detail, err)
}

func newRunCommand(startDir string, runner Runner) *cobra.Command {
	if runner == nil {
		runner = lab.NewService(nil)
	}

	var output string
	request := lab.RunRequest{}
	var taskIDs []string
	var conditionIDs []string
	var noRandomize bool
	var outcomePayload string
	var outcomeFile string
	var copilotSessionID string
	var copilotSessionStateRoot string

	cmd := &cobra.Command{
		Use:           "run",
		Short:         "Execute a controlled experiment run for a declared task and condition",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			request.TaskID = ""
			request.TaskIDs = normalizeFlagValues(taskIDs)
			if len(request.TaskIDs) == 1 {
				request.TaskID = request.TaskIDs[0]
				request.TaskIDs = nil
			}

			request.ConditionID = ""
			request.ConditionIDs = normalizeFlagValues(conditionIDs)
			if len(request.ConditionIDs) == 1 {
				request.ConditionID = request.ConditionIDs[0]
				request.ConditionIDs = nil
			}

			request.Randomize = nil
			if noRandomize {
				randomize := false
				request.Randomize = &randomize
			}
			request.OutcomePayload = ""
			if outcomePayload != "" || outcomeFile != "" {
				input, err := cmdsupport.ResolveTextInput(cmd.InOrStdin(), outcomePayload, outcomeFile, "payload", "--outcome-payload")
				if err != nil {
					return handleRunError(cmd.OutOrStdout(), output, err)
				}
				request.OutcomePayload = input.Value
			}
			request.CopilotSessionID = strings.TrimSpace(copilotSessionID)
			request.CopilotSessionStateRoot = strings.TrimSpace(copilotSessionStateRoot)

			result, err := runner.Run(cmd.Context(), startDir, request)
			if err != nil {
				return handleRunError(cmd.OutOrStdout(), output, err)
			}
			return cmdsupport.WriteSuccess(cmd.OutOrStdout(), output, "lab.run", result)
		},
	}

	cmd.Flags().StringSliceVar(&taskIDs, "task", nil, "Suite task identifier to execute; repeat for batch runs")
	cmd.Flags().StringSliceVar(&conditionIDs, "condition", nil, "Condition identifier to apply; repeat for batch runs")
	cmd.Flags().StringVar(&request.Model, "model", "", "Declared model identifier for the run")
	cmd.Flags().StringVar(&request.SessionTopology, "topology", "", "Declared session topology for the run")
	cmd.Flags().Int64Var(&request.Seed, "seed", 0, "Randomization seed for the controlled run")
	cmd.Flags().BoolVar(&noRandomize, "no-randomize", false, "Preserve natural task × condition order for batch runs")
	cmd.Flags().StringVar(&outcomePayload, "outcome-payload", "", "Inline delegated outcome payload carrying execution telemetry")
	cmd.Flags().StringVar(&outcomeFile, "outcome-file", "", "File containing the delegated outcome payload for a single run")
	cmd.Flags().StringVar(&copilotSessionID, "copilot-session-id", "", "Collect authoritative token usage from the local Copilot session-state for this single run")
	cmd.Flags().StringVar(&copilotSessionStateRoot, "copilot-session-root", "", "Override the Copilot session-state root used with --copilot-session-id")
	cmd.Flags().StringVar(&output, "output", "", "Write the structured JSON response to a file instead of stdout")
	return cmd
}

func normalizeFlagValues(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func handleRunError(w io.Writer, outputPath string, err error) error {
	return handleLabCommandError(w, outputPath, "lab.run", "lab_run_failed", "lab run execution failed", err)
}

func newEvaluateCommand(startDir string, evaluator Evaluator, presenter EvaluationPresenter) *cobra.Command {
	if evaluator == nil {
		evaluator = lab.NewService(nil)
	}
	if presenter == nil {
		presenter = lab.NewService(nil)
	}

	var output string
	var runID string
	var evaluatorID string
	var success bool
	var quality float64
	var resumability float64
	var humanInterventions int
	var notes string
	var evaluatedAt string

	cmd := &cobra.Command{
		Use:           "evaluate",
		Short:         "Record evaluation scores separately from run execution",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := evaluator.Evaluate(cmd.Context(), startDir, lab.EvaluationRecord{
				SchemaVersion: lab.SchemaVersion,
				RunID:         runID,
				Evaluator:     evaluatorID,
				EvaluatedAt:   evaluatedAt,
				Scores: &lab.EvaluationScores{
					Success:                &success,
					QualityScore:           &quality,
					ResumabilityScore:      &resumability,
					HumanInterventionCount: &humanInterventions,
				},
				Notes: notes,
			})
			if err != nil {
				return handleEvaluateError(cmd.OutOrStdout(), output, err)
			}
			return cmdsupport.WriteSuccess(cmd.OutOrStdout(), output, "lab.evaluate", result)
		},
	}

	cmd.Flags().StringVar(&runID, "run", "", "Run identifier to evaluate")
	cmd.Flags().StringVar(&evaluatorID, "evaluator", "", "Evaluator identity, e.g. human:alice or automated:rubric-v1")
	cmd.Flags().BoolVar(&success, "success", false, "Whether the run satisfied the evaluation success criterion")
	cmd.Flags().Float64Var(&quality, "quality", 0, "Quality score between 0 and 1")
	cmd.Flags().Float64Var(&resumability, "resumability", 0, "Resumability score between 0 and 1")
	cmd.Flags().IntVar(&humanInterventions, "human-interventions", 0, "Human intervention count")
	cmd.Flags().StringVar(&notes, "notes", "", "Optional evaluation notes")
	cmd.Flags().StringVar(&evaluatedAt, "evaluated-at", "", "RFC3339 timestamp for the evaluation; defaults to current UTC time")
	cmd.Flags().StringVar(&output, "output", "", "Write the structured JSON response to a file instead of stdout")
	_ = cmd.MarkFlagRequired("run")
	_ = cmd.MarkFlagRequired("evaluator")
	_ = cmd.MarkFlagRequired("success")
	_ = cmd.MarkFlagRequired("quality")
	_ = cmd.MarkFlagRequired("resumability")
	_ = cmd.MarkFlagRequired("human-interventions")

	cmd.AddCommand(newEvaluatePresentCommand(startDir, presenter))
	return cmd
}

func newEvaluatePresentCommand(startDir string, presenter EvaluationPresenter) *cobra.Command {
	if presenter == nil {
		presenter = lab.NewService(nil)
	}

	var output string
	request := lab.PresentEvaluationRequest{}

	cmd := &cobra.Command{
		Use:           "present",
		Short:         "Prepare evaluation input for a run, optionally blinded to condition",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := presenter.PresentEvaluationInput(cmd.Context(), startDir, request)
			if err != nil {
				return handleEvaluatePresentError(cmd.OutOrStdout(), output, err)
			}
			return cmdsupport.WriteSuccess(cmd.OutOrStdout(), output, "lab.evaluate.present", result)
		},
	}

	cmd.Flags().StringVar(&request.RunID, "run", "", "Run identifier to present for evaluation")
	cmd.Flags().BoolVar(&request.Blind, "blind", false, "Hide condition labels and workflow-mode cues in the presented input")
	cmd.Flags().StringVar(&output, "output", "", "Write the structured JSON response to a file instead of stdout")
	_ = cmd.MarkFlagRequired("run")
	return cmd
}

func handleEvaluateError(w io.Writer, outputPath string, err error) error {
	return handleLabCommandError(w, outputPath, "lab.evaluate", "lab_evaluate_failed", "lab evaluation failed", err)
}

func handleEvaluatePresentError(w io.Writer, outputPath string, err error) error {
	return handleLabCommandError(w, outputPath, "lab.evaluate.present", "lab_evaluation_present_failed", "lab evaluation input preparation failed", err)
}

func newReportCommand(startDir string, reporter Reporter) *cobra.Command {
	if reporter == nil {
		reporter = lab.NewService(nil)
	}

	var output string
	var runIDs []string

	cmd := &cobra.Command{
		Use:           "report",
		Short:         "Aggregate run and evaluation artifacts into a machine-readable lab report",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := reporter.Report(cmd.Context(), startDir, lab.ReportRequest{
				RunIDs: normalizeFlagValues(runIDs),
			})
			if err != nil {
				return handleReportError(cmd.OutOrStdout(), output, err)
			}
			return cmdsupport.WriteSuccess(cmd.OutOrStdout(), output, "lab.report", result)
		},
	}

	cmd.Flags().StringSliceVar(&runIDs, "run", nil, "Run identifier to include in the report; repeat to focus the report on a selected run set")
	cmd.Flags().StringVar(&output, "output", "", "Write the structured JSON response to a file instead of stdout")
	return cmd
}

func handleLabCommandError(w io.Writer, outputPath, command, code, message string, err error) error {
	if detail, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return cmdsupport.WriteFailure(w, outputPath, command, detail, err)
	}

	detail := cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "lab_error",
		Code:     code,
		Message:  message,
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

	return cmdsupport.WriteFailure(w, outputPath, command, detail, err)
}

func handleReportError(w io.Writer, outputPath string, err error) error {
	return handleLabCommandError(w, outputPath, "lab.report", "lab_report_failed", "lab report generation failed", err)
}
