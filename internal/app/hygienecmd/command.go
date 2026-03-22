package hygienecmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/app/graphhealth"
	graphpayload "github.com/guillaume-galp/cge/internal/domain/payload"
	"github.com/guillaume-galp/cge/internal/infra/kuzu"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

type Engine interface {
	Analyze(cmd *cobra.Command, workspace repo.Workspace) (graphhealth.Analysis, error)
	Apply(cmd *cobra.Command, workspace repo.Workspace, plan graphhealth.HygienePlan) (ApplyOutcome, error)
}

type ApplyOutcome struct {
	BeforeRevision    kuzu.CurrentRevisionState  `json:"before_revision,omitempty"`
	SyncSummary       kuzu.GraphSyncSummary      `json:"sync_summary"`
	Applied           graphhealth.AppliedSummary `json:"applied"`
	SelectedActionIDs []string                   `json:"selected_action_ids"`
}

type graphEngine struct {
	store *kuzu.Store
}

func (e graphEngine) Analyze(cmd *cobra.Command, workspace repo.Workspace) (graphhealth.Analysis, error) {
	if e.store == nil {
		e.store = kuzu.NewStore()
	}
	graph, err := e.store.ReadGraph(cmd.Context(), workspace)
	if err != nil {
		return graphhealth.Analysis{}, err
	}
	return graphhealth.AnalyzeGraph(graph)
}

func (e graphEngine) Apply(cmd *cobra.Command, workspace repo.Workspace, plan graphhealth.HygienePlan) (ApplyOutcome, error) {
	if e.store == nil {
		e.store = kuzu.NewStore()
	}

	graph, err := e.store.ReadGraph(cmd.Context(), workspace)
	if err != nil {
		return ApplyOutcome{}, err
	}

	beforeRevision, err := e.store.CurrentRevision(cmd.Context(), workspace)
	if err != nil {
		return ApplyOutcome{}, err
	}

	applyResult, err := graphhealth.ApplyPlan(graph, plan)
	if err != nil {
		return ApplyOutcome{}, err
	}
	if len(applyResult.AppliedActionIDs) == 0 {
		return ApplyOutcome{}, cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "input_error",
			Code:     "no_selected_hygiene_actions",
			Message:  "hygiene apply requires at least one selected action",
			Details: map[string]any{
				"field": "selected_action_ids",
			},
		}, errors.New("hygiene apply requires at least one selected action"))
	}

	metadata := graphpayload.Metadata{
		AgentID:   "graph-hygiene",
		SessionID: fmt.Sprintf("hygiene-%d", time.Now().UTC().UnixNano()),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Revision: graphpayload.RevisionMetadata{
			Reason: fmt.Sprintf("Apply hygiene plan (%d action(s))", len(applyResult.AppliedActionIDs)),
			Properties: map[string]any{
				"hygiene": map[string]any{
					"selected_action_ids": applyResult.AppliedActionIDs,
					"before_snapshot":     applyResult.BeforeAnchor,
					"after_snapshot":      applyResult.AfterAnchor,
					"applied_summary":     applyResult.AppliedSummary,
				},
			},
		},
	}

	syncSummary, err := e.store.ReplaceGraph(cmd.Context(), workspace, metadata, applyResult.TargetGraph)
	if err != nil {
		return ApplyOutcome{}, err
	}

	return ApplyOutcome{
		BeforeRevision:    beforeRevision,
		SyncSummary:       syncSummary,
		Applied:           applyResult.AppliedSummary,
		SelectedActionIDs: applyResult.AppliedActionIDs,
	}, nil
}

func NewCommand(startDir string, manager *repo.Manager) *cobra.Command {
	return newCommand(startDir, manager, graphEngine{store: kuzu.NewStore()})
}

func newCommand(startDir string, manager *repo.Manager, engine Engine) *cobra.Command {
	if engine == nil {
		engine = graphEngine{store: kuzu.NewStore()}
	}

	var apply bool
	var file string
	var output string

	cmd := &cobra.Command{
		Use:           "hygiene",
		Short:         "Suggest or apply graph hygiene actions for the repo-local workspace",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, err := cmdsupport.RequireWorkspace(cmd, startDir, manager)
			if err != nil {
				return handleHygieneError(cmd.OutOrStdout(), output, err)
			}

			if !apply {
				analysis, err := engine.Analyze(cmd, workspace)
				if err != nil {
					return handleHygieneError(cmd.OutOrStdout(), output, err)
				}
				return cmdsupport.WriteSuccess(cmd.OutOrStdout(), output, "hygiene", suggestResultEnvelope{
					Mode: "suggest",
					Plan: analysis.Plan,
				})
			}

			planInput, err := resolveApplyInput(file)
			if err != nil {
				return handleHygieneError(cmd.OutOrStdout(), output, err)
			}
			plan, err := parsePlan(planInput)
			if err != nil {
				return handleHygieneError(cmd.OutOrStdout(), output, err)
			}
			if len(plan.SelectedActionIDs) == 0 {
				return handleHygieneError(cmd.OutOrStdout(), output, cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
					Category: "validation_error",
					Type:     "input_error",
					Code:     "no_selected_hygiene_actions",
					Message:  "hygiene apply requires at least one selected action",
					Details: map[string]any{
						"field": "selected_action_ids",
					},
				}, errors.New("hygiene apply requires at least one selected action")))
			}

			outcome, err := engine.Apply(cmd, workspace, plan)
			if err != nil {
				return handleHygieneError(cmd.OutOrStdout(), output, classifyApplyError(err))
			}

			return cmdsupport.WriteSuccess(cmd.OutOrStdout(), output, "hygiene", applyResultEnvelope{
				Mode:              "apply",
				Applied:           outcome.Applied,
				SelectedActionIDs: outcome.SelectedActionIDs,
				BeforeRevision:    outcome.BeforeRevision,
				Revision:          outcome.SyncSummary.Revision,
				SyncSummary:       outcome.SyncSummary,
			})
		},
	}

	cmd.Flags().BoolVar(&apply, "apply", false, "Apply a selected hygiene plan instead of suggesting actions")
	cmd.Flags().StringVar(&file, "file", "", "File containing a hygiene plan for --apply")
	cmd.Flags().StringVar(&output, "output", "", "Write the structured JSON response to a file instead of stdout")

	return cmd
}

type suggestResultEnvelope struct {
	Mode string                  `json:"mode"`
	Plan graphhealth.HygienePlan `json:"plan"`
}

type applyResultEnvelope struct {
	Mode              string                     `json:"mode"`
	Applied           graphhealth.AppliedSummary `json:"applied"`
	SelectedActionIDs []string                   `json:"selected_action_ids"`
	BeforeRevision    kuzu.CurrentRevisionState  `json:"before_revision,omitempty"`
	Revision          kuzu.RevisionWriteSummary  `json:"revision"`
	SyncSummary       kuzu.GraphSyncSummary      `json:"sync_summary"`
}

func resolveApplyInput(file string) (string, error) {
	file = strings.TrimSpace(file)
	if file == "" {
		return "", cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "input_error",
			Code:     "missing_hygiene_plan",
			Message:  "hygiene apply requires an explicit plan file",
			Details: map[string]any{
				"accepted_sources": []string{"--file"},
			},
		}, errors.New("hygiene apply requires --file"))
	}
	payload, err := os.ReadFile(file)
	if err != nil {
		return "", cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "operational_error",
			Type:     "input_error",
			Code:     "input_file_read_failed",
			Message:  "hygiene plan file could not be read",
			Details: map[string]any{
				"path":   file,
				"reason": err.Error(),
			},
		}, err)
	}
	value := strings.TrimSpace(string(payload))
	if value == "" {
		return "", cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "input_error",
			Code:     "empty_input_file",
			Message:  "hygiene plan file is empty",
			Details: map[string]any{
				"path": file,
			},
		}, errors.New("hygiene plan file is empty"))
	}
	return value, nil
}

func parsePlan(input string) (graphhealth.HygienePlan, error) {
	var plan graphhealth.HygienePlan
	if err := json.Unmarshal([]byte(input), &plan); err == nil && strings.TrimSpace(plan.SnapshotAnchor) != "" {
		return plan, nil
	}

	var envelope struct {
		Result struct {
			Plan graphhealth.HygienePlan `json:"plan"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(input), &envelope); err == nil && strings.TrimSpace(envelope.Result.Plan.SnapshotAnchor) != "" {
		return envelope.Result.Plan, nil
	}

	return graphhealth.HygienePlan{}, cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
		Category: "validation_error",
		Type:     "input_error",
		Code:     "invalid_hygiene_plan",
		Message:  "hygiene plan input is not a valid plan payload",
		Details: map[string]any{
			"expected_shapes": []string{"hygiene plan JSON", "graph hygiene response envelope"},
		},
	}, errors.New("hygiene plan input is not a valid plan payload"))
}

func classifyApplyError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return err
	}

	message := err.Error()
	detail := cmdsupport.ErrorDetail{
		Category: "validation_error",
		Type:     "input_error",
		Code:     "unsafe_hygiene_plan",
		Message:  "hygiene plan is unsafe or invalid",
		Details: map[string]any{
			"reason": message,
		},
	}
	switch {
	case strings.Contains(message, "snapshot anchor does not match"):
		detail.Code = "stale_hygiene_plan"
		detail.Message = "hygiene plan snapshot does not match the current graph state"
	case strings.Contains(message, "unsupported hygiene action type"):
		detail.Code = "unsupported_hygiene_action"
		detail.Message = "hygiene plan contains an unsupported action"
	case strings.Contains(message, "selected hygiene action"):
		detail.Code = "undefined_hygiene_action"
		detail.Message = "hygiene plan selects an undefined action"
	}
	return cmdsupport.NewCommandError(detail, err)
}

func handleHygieneError(w io.Writer, outputPath string, err error) error {
	if detail, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return cmdsupport.WriteFailure(w, outputPath, "hygiene", detail, err)
	}

	var persistenceErr *kuzu.PersistenceError
	if errors.As(err, &persistenceErr) {
		return cmdsupport.WriteFailure(w, outputPath, "hygiene", cmdsupport.ErrorDetail{
			Category: "operational_error",
			Type:     "persistence_error",
			Code:     persistenceErr.Code,
			Message:  persistenceErr.Message,
			Details:  persistenceErr.Details,
		}, err)
	}

	return cmdsupport.WriteFailure(w, outputPath, "hygiene", cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "hygiene_error",
		Code:     "hygiene_failed",
		Message:  "graph hygiene failed",
		Details: map[string]any{
			"reason": err.Error(),
		},
	}, err)
}

func AnalyzeCurrentGraph(ctx context.Context, workspace repo.Workspace, store *kuzu.Store) (graphhealth.Analysis, error) {
	if store == nil {
		store = kuzu.NewStore()
	}
	graph, err := store.ReadGraph(ctx, workspace)
	if err != nil {
		return graphhealth.Analysis{}, err
	}
	return graphhealth.AnalyzeGraph(graph)
}
