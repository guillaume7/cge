package lab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/app/workflow"
	"github.com/guillaume-galp/cge/internal/infra/copilot"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

const (
	RunStatusCompleted      = "completed"
	defaultPromptVariant    = "default"
	defaultKickoffMaxTokens = 1200
)

type WorkflowRunner interface {
	Start(ctx context.Context, startDir, task string, maxTokens int) (workflow.StartResult, error)
	Finish(ctx context.Context, startDir, input string) (workflow.FinishResult, error)
}

type RunRequest struct {
	TaskID                  string   `json:"task_id,omitempty"`
	TaskIDs                 []string `json:"task_ids,omitempty"`
	ConditionID             string   `json:"condition_id,omitempty"`
	ConditionIDs            []string `json:"condition_ids,omitempty"`
	Model                   string   `json:"model"`
	SessionTopology         string   `json:"session_topology"`
	Seed                    int64    `json:"seed"`
	Randomize               *bool    `json:"randomize,omitempty"`
	OutcomePayload          string   `json:"outcome_payload,omitempty"`
	CopilotSessionID        string   `json:"copilot_session_id,omitempty"`
	CopilotSessionStateRoot string   `json:"copilot_session_state_root,omitempty"`
}

type RunResult struct {
	RunID      string          `json:"run_id,omitempty"`
	Status     string          `json:"status"`
	Parameters RunParameters   `json:"parameters"`
	Execution  RunExecution    `json:"execution"`
	Batch      *BatchExecution `json:"batch,omitempty"`
}

type RunParameters struct {
	TaskID          string   `json:"task_id,omitempty"`
	TaskIDs         []string `json:"task_ids,omitempty"`
	ConditionID     string   `json:"condition_id,omitempty"`
	ConditionIDs    []string `json:"condition_ids,omitempty"`
	Model           string   `json:"model"`
	SessionTopology string   `json:"session_topology"`
	Seed            int64    `json:"seed"`
	Randomize       *bool    `json:"randomize,omitempty"`
}

type RunExecution struct {
	WorkflowMode string `json:"workflow_mode"`
	GraphBacked  bool   `json:"graph_backed"`
	KickoffUsed  bool   `json:"kickoff_used"`
	HandoffUsed  bool   `json:"handoff_used"`
	RecordPath   string `json:"record_path,omitempty"`
}

type BatchExecution struct {
	BatchID    string         `json:"batch_id"`
	Randomized bool           `json:"randomized"`
	PlanPath   string         `json:"plan_path"`
	RunCount   int            `json:"run_count"`
	Runs       []BatchRunItem `json:"runs"`
}

type BatchRunItem struct {
	Order       int    `json:"order"`
	TaskID      string `json:"task_id"`
	ConditionID string `json:"condition_id"`
	RunID       string `json:"run_id,omitempty"`
}

type BatchPlanArtifact struct {
	SchemaVersion   string           `json:"schema_version"`
	BatchID         string           `json:"batch_id"`
	PlannedAt       string           `json:"planned_at"`
	Model           string           `json:"model"`
	SessionTopology string           `json:"session_topology"`
	Seed            int64            `json:"seed"`
	Randomized      bool             `json:"randomized"`
	TaskIDs         []string         `json:"task_ids"`
	ConditionIDs    []string         `json:"condition_ids"`
	Entries         []BatchPlanEntry `json:"entries"`
}

type BatchPlanEntry struct {
	Order       int    `json:"order"`
	TaskID      string `json:"task_id"`
	ConditionID string `json:"condition_id"`
}

type runArtifacts struct {
	KickoffInputs    any
	SessionStructure any
	WritebackOutputs any
	OutcomeSummary   any
}

type resolvedRunRequest struct {
	taskIDs      []string
	conditionIDs []string
	tasks        map[string]SuiteTask
	conditions   map[string]Condition
	randomize    bool
	isBatch      bool
}

func (s *Service) WorkflowRunnerForTest(runner WorkflowRunner) {
	if s == nil {
		return
	}
	s.workflowRunner = runner
}

func (s *Service) NowForTest(now func() time.Time) {
	if s == nil {
		return
	}
	s.now = now
}

func (s *Service) CopilotUsageCollectorForTest(collector CopilotSessionUsageCollector) {
	if s == nil {
		return
	}
	s.copilotUsageCollector = collector
}

func (s *Service) Run(ctx context.Context, startDir string, request RunRequest) (RunResult, error) {
	if s == nil || s.manager == nil {
		return RunResult{}, errors.New("lab service is not configured")
	}
	request.TaskID = strings.TrimSpace(request.TaskID)
	request.ConditionID = strings.TrimSpace(request.ConditionID)
	request.Model = strings.TrimSpace(request.Model)
	request.SessionTopology = strings.TrimSpace(request.SessionTopology)
	request.TaskIDs = normalizeIdentifiers(request.TaskIDs)
	request.ConditionIDs = normalizeIdentifiers(request.ConditionIDs)
	request.OutcomePayload = strings.TrimSpace(request.OutcomePayload)
	request.CopilotSessionID = strings.TrimSpace(request.CopilotSessionID)
	request.CopilotSessionStateRoot = strings.TrimSpace(request.CopilotSessionStateRoot)
	if s.workflowRunner == nil {
		s.workflowRunner = workflow.NewService(s.manager)
	}
	if s.now == nil {
		s.now = func() time.Time { return time.Now().UTC() }
	}

	workspace, err := s.manager.OpenWorkspace(ctx, startDir)
	if err != nil {
		return RunResult{}, err
	}

	suitePath := filepath.Join(workspace.WorkspacePath, repo.LabDirName, repo.LabSuiteManifestName)
	suite, err := loadSuiteManifest(suitePath)
	if err != nil {
		return RunResult{}, err
	}

	conditionsPath := filepath.Join(workspace.WorkspacePath, repo.LabDirName, repo.LabConditionsManifestName)
	conditions, err := loadConditionsManifest(conditionsPath)
	if err != nil {
		return RunResult{}, err
	}

	resolved, err := resolveRunRequest(request, suitePath, conditionsPath, suite, conditions)
	if err != nil {
		return RunResult{}, err
	}

	if !resolved.isBatch {
		task := resolved.tasks[resolved.taskIDs[0]]
		condition := resolved.conditions[resolved.conditionIDs[0]]
		request.TaskID = resolved.taskIDs[0]
		request.TaskIDs = nil
		request.ConditionID = resolved.conditionIDs[0]
		request.ConditionIDs = nil
		return s.runSingle(ctx, startDir, workspace, request, task, condition)
	}

	return s.runBatch(ctx, startDir, workspace, request, resolved)
}

func (s *Service) runBatch(ctx context.Context, startDir string, workspace repo.Workspace, request RunRequest, resolved resolvedRunRequest) (RunResult, error) {
	plannedAt := s.now().UTC()
	batchID := nextBatchID(filepath.Join(workspace.WorkspacePath, repo.LabDirName, "runs", "batches"), plannedAt)
	plan := buildBatchPlan(batchID, plannedAt, request, resolved)
	planPath, err := persistBatchPlan(workspace, plan)
	if err != nil {
		return RunResult{}, err
	}

	batchRuns := make([]BatchRunItem, 0, len(plan.Entries))
	for _, entry := range plan.Entries {
		task := resolved.tasks[entry.TaskID]
		condition := resolved.conditions[entry.ConditionID]
		singleResult, err := s.runSingle(ctx, startDir, workspace, RunRequest{
			TaskID:          entry.TaskID,
			ConditionID:     entry.ConditionID,
			Model:           request.Model,
			SessionTopology: request.SessionTopology,
			Seed:            request.Seed,
		}, task, condition)
		if err != nil {
			return RunResult{}, err
		}
		batchRuns = append(batchRuns, BatchRunItem{
			Order:       entry.Order,
			TaskID:      entry.TaskID,
			ConditionID: entry.ConditionID,
			RunID:       singleResult.RunID,
		})
	}

	return RunResult{
		Status: RunStatusCompleted,
		Parameters: RunParameters{
			TaskIDs:         append([]string(nil), resolved.taskIDs...),
			ConditionIDs:    append([]string(nil), resolved.conditionIDs...),
			Model:           request.Model,
			SessionTopology: request.SessionTopology,
			Seed:            request.Seed,
			Randomize:       boolPointer(resolved.randomize),
		},
		Batch: &BatchExecution{
			BatchID:    batchID,
			Randomized: resolved.randomize,
			PlanPath:   relativePath(workspace.RepoRoot, planPath),
			RunCount:   len(batchRuns),
			Runs:       batchRuns,
		},
	}, nil
}

func (s *Service) runSingle(ctx context.Context, startDir string, workspace repo.Workspace, request RunRequest, task SuiteTask, condition Condition) (RunResult, error) {
	startedAt := s.now().UTC()
	runID := nextRunID(filepath.Join(workspace.WorkspacePath, repo.LabDirName, "runs"), startedAt)

	execution := RunExecution{
		WorkflowMode: condition.WorkflowMode,
		GraphBacked:  condition.WorkflowMode == WorkflowModeGraphBacked,
	}
	var executionTelemetry *workflow.ExecutionTelemetry
	payload := request.OutcomePayload
	if request.CopilotSessionID != "" {
		var err error
		payload, err = s.enrichOutcomePayloadWithCopilotUsage(ctx, request, payload)
		if err != nil {
			return RunResult{}, err
		}
	}

	if execution.GraphBacked {
		_, err := s.workflowRunner.Start(ctx, startDir, task.Description, defaultKickoffMaxTokens)
		if err != nil {
			return RunResult{}, err
		}
		execution.KickoffUsed = true

		if payload == "" {
			payload, err = buildWorkflowFinishPayload(runID, startedAt, task)
			if err != nil {
				return RunResult{}, err
			}
		}
		finishResult, err := s.workflowRunner.Finish(ctx, startDir, payload)
		if err != nil {
			return RunResult{}, err
		}
		executionTelemetry = finishResult.ExecutionTelemetry
		execution.HandoffUsed = true
	} else if payload != "" {
		telemetry, err := workflow.ExtractExecutionTelemetryFromFinishPayload(payload)
		if err != nil {
			return RunResult{}, err
		}
		executionTelemetry = telemetry
	}
	finishedAt := s.now().UTC()
	telemetry := buildRunTelemetry(execution, executionTelemetry, elapsedWallClockSeconds(startedAt, finishedAt), payload != "")

	record := buildRunRecord(runID, request, condition, startedAt, finishedAt, telemetry)
	runsPath := filepath.Join(workspace.WorkspacePath, repo.LabDirName, "runs")
	if err := validateRunRecord(runsPath, SuiteManifest{Tasks: []SuiteTask{task}}, ConditionsManifest{Conditions: []Condition{condition}}, record); err != nil {
		return RunResult{}, err
	}
	recordPath, err := persistRunLedger(workspace, record, buildRunArtifacts(task, request, condition, startedAt, finishedAt, execution))
	if err != nil {
		return RunResult{}, err
	}
	execution.RecordPath = relativePath(workspace.RepoRoot, recordPath)

	return RunResult{
		RunID:  runID,
		Status: RunStatusCompleted,
		Parameters: RunParameters{
			TaskID:          request.TaskID,
			ConditionID:     request.ConditionID,
			Model:           request.Model,
			SessionTopology: request.SessionTopology,
			Seed:            request.Seed,
		},
		Execution: execution,
	}, nil
}

func resolveRunRequest(
	request RunRequest,
	suitePath, conditionsPath string,
	suite SuiteManifest,
	conditions ConditionsManifest,
) (resolvedRunRequest, error) {
	request.TaskID = strings.TrimSpace(request.TaskID)
	request.ConditionID = strings.TrimSpace(request.ConditionID)
	request.Model = strings.TrimSpace(request.Model)
	request.SessionTopology = strings.TrimSpace(request.SessionTopology)
	request.CopilotSessionID = strings.TrimSpace(request.CopilotSessionID)
	request.CopilotSessionStateRoot = strings.TrimSpace(request.CopilotSessionStateRoot)

	taskIDs := normalizeIdentifiers(request.TaskIDs)
	if len(taskIDs) == 0 && request.TaskID != "" {
		taskIDs = []string{request.TaskID}
	}
	conditionIDs := normalizeIdentifiers(request.ConditionIDs)
	if len(conditionIDs) == 0 && request.ConditionID != "" {
		conditionIDs = []string{request.ConditionID}
	}

	taskField := "task_id"
	if len(taskIDs) > 1 {
		taskField = "task_ids"
	}
	conditionField := "condition_id"
	if len(conditionIDs) > 1 {
		conditionField = "condition_ids"
	}

	var violations []map[string]any
	if len(taskIDs) == 0 {
		violations = append(violations, violation(taskField, "field is required"))
	}
	if len(conditionIDs) == 0 {
		violations = append(violations, violation(conditionField, "field is required"))
	}
	if request.Model == "" {
		violations = append(violations, violation("model", "field is required"))
	} else if strings.ContainsRune(request.Model, '\x00') {
		violations = append(violations, violationWithValue("model", "must not contain null bytes", request.Model))
	}
	if request.SessionTopology == "" {
		violations = append(violations, violation("session_topology", "field is required"))
	} else if strings.ContainsRune(request.SessionTopology, '\x00') {
		violations = append(violations, violationWithValue("session_topology", "must not contain null bytes", request.SessionTopology))
	}
	if strings.ContainsRune(request.OutcomePayload, '\x00') {
		violations = append(violations, violationWithValue("outcome_payload", "must not contain null bytes", request.OutcomePayload))
	}
	if strings.ContainsRune(request.CopilotSessionID, '\x00') {
		violations = append(violations, violationWithValue("copilot_session_id", "must not contain null bytes", request.CopilotSessionID))
	}
	if strings.ContainsRune(request.CopilotSessionStateRoot, '\x00') {
		violations = append(violations, violationWithValue("copilot_session_state_root", "must not contain null bytes", request.CopilotSessionStateRoot))
	}

	tasks := make(map[string]SuiteTask, len(taskIDs))
	for index, taskID := range taskIDs {
		field := taskField
		if len(taskIDs) > 1 {
			field = fmt.Sprintf("task_ids[%d]", index)
		}
		if !isSafeIdentifier(taskID) {
			violations = append(violations, violationWithValue(field, "must be a simple identifier without path separators", taskID))
			continue
		}
		task, ok := suite.TaskByID(taskID)
		if !ok {
			entry := violationWithValue(field, "task_id must reference an existing suite task", taskID)
			entry["manifest_path"] = filepath.ToSlash(suitePath)
			violations = append(violations, entry)
			continue
		}
		tasks[taskID] = task
	}

	resolvedConditions := make(map[string]Condition, len(conditionIDs))
	for index, conditionID := range conditionIDs {
		field := conditionField
		if len(conditionIDs) > 1 {
			field = fmt.Sprintf("condition_ids[%d]", index)
		}
		if !isSafeIdentifier(conditionID) {
			violations = append(violations, violationWithValue(field, "must be a simple identifier without path separators", conditionID))
			continue
		}
		condition, ok := conditions.ConditionByID(conditionID)
		if !ok {
			entry := violationWithValue(field, "condition_id must reference an existing condition", conditionID)
			entry["manifest_path"] = filepath.ToSlash(conditionsPath)
			violations = append(violations, entry)
			continue
		}
		resolvedConditions[conditionID] = condition
	}

	if len(violations) > 0 {
		return resolvedRunRequest{}, cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "validation_error",
			Code:     "lab_run_validation_failed",
			Message:  "lab run request validation failed",
			Details: map[string]any{
				"violations": violations,
			},
		}, fmt.Errorf("lab run request validation failed"))
	}

	randomize := true
	if request.Randomize != nil {
		randomize = *request.Randomize
	}
	if len(taskIDs)*len(conditionIDs) > 1 && request.OutcomePayload != "" {
		violations := []map[string]any{
			violation("outcome_payload", "outcome_payload is only supported for single-run execution"),
		}
		return resolvedRunRequest{}, cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "validation_error",
			Code:     "lab_run_validation_failed",
			Message:  "lab run request validation failed",
			Details: map[string]any{
				"violations": violations,
			},
		}, fmt.Errorf("lab run request validation failed"))
	}
	if request.CopilotSessionID != "" && request.OutcomePayload == "" {
		violations := []map[string]any{
			violation("copilot_session_id", "copilot_session_id requires outcome_payload for single-run execution"),
		}
		return resolvedRunRequest{}, cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "validation_error",
			Code:     "lab_run_validation_failed",
			Message:  "lab run request validation failed",
			Details: map[string]any{
				"violations": violations,
			},
		}, fmt.Errorf("lab run request validation failed"))
	}
	if len(taskIDs)*len(conditionIDs) > 1 && request.CopilotSessionID != "" {
		violations := []map[string]any{
			violation("copilot_session_id", "copilot_session_id is only supported for single-run execution"),
		}
		return resolvedRunRequest{}, cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "validation_error",
			Code:     "lab_run_validation_failed",
			Message:  "lab run request validation failed",
			Details: map[string]any{
				"violations": violations,
			},
		}, fmt.Errorf("lab run request validation failed"))
	}

	return resolvedRunRequest{
		taskIDs:      taskIDs,
		conditionIDs: conditionIDs,
		tasks:        tasks,
		conditions:   resolvedConditions,
		randomize:    randomize,
		isBatch:      len(taskIDs)*len(conditionIDs) > 1,
	}, nil
}

func (s *Service) enrichOutcomePayloadWithCopilotUsage(ctx context.Context, request RunRequest, payload string) (string, error) {
	if strings.TrimSpace(request.CopilotSessionID) == "" {
		return payload, nil
	}
	if strings.TrimSpace(payload) == "" {
		return "", cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "validation_error",
			Code:     "lab_run_validation_failed",
			Message:  "lab run request validation failed",
			Details: map[string]any{
				"violations": []map[string]any{
					violation("copilot_session_id", "copilot_session_id requires outcome_payload for single-run execution"),
				},
			},
		}, fmt.Errorf("lab run request validation failed"))
	}
	if s == nil || s.copilotUsageCollector == nil {
		return "", cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "operational_error",
			Type:     "telemetry_error",
			Code:     "copilot_usage_collector_unavailable",
			Message:  "copilot session usage collector is not configured",
		}, errors.New("copilot session usage collector is not configured"))
	}

	usage, err := s.copilotUsageCollector.CollectSessionUsage(ctx, copilot.SessionUsageRequest{
		SessionID:        request.CopilotSessionID,
		Model:            request.Model,
		SessionStateRoot: request.CopilotSessionStateRoot,
	})
	if err != nil {
		return "", cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "operational_error",
			Type:     "telemetry_error",
			Code:     "copilot_usage_collection_failed",
			Message:  "copilot session usage collection failed",
			Details: map[string]any{
				"session_id": request.CopilotSessionID,
				"model":      request.Model,
				"reason":     err.Error(),
			},
		}, err)
	}

	return workflow.ApplyExecutionTelemetryToFinishPayload(payload, workflow.ExecutionTelemetry{
		MeasurementStatus: workflow.ExecutionTelemetryStatusComplete,
		Source:            copilot.SessionUsageSource,
		Provider:          copilot.SessionUsageProvider,
		InputTokens:       intPointer(usage.InputTokens),
		OutputTokens:      intPointer(usage.OutputTokens),
		TotalTokens:       intPointer(usage.TotalTokens),
	})
}

func normalizeIdentifiers(values []string) []string {
	if len(values) == 0 {
		return nil
	}
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

func buildBatchPlan(batchID string, plannedAt time.Time, request RunRequest, resolved resolvedRunRequest) BatchPlanArtifact {
	entries := make([]BatchPlanEntry, 0, len(resolved.taskIDs)*len(resolved.conditionIDs))
	for _, taskID := range resolved.taskIDs {
		for _, conditionID := range resolved.conditionIDs {
			entries = append(entries, BatchPlanEntry{TaskID: taskID, ConditionID: conditionID})
		}
	}
	if resolved.randomize {
		rng := rand.New(rand.NewSource(request.Seed))
		rng.Shuffle(len(entries), func(i, j int) {
			entries[i], entries[j] = entries[j], entries[i]
		})
	}
	for index := range entries {
		entries[index].Order = index + 1
	}

	return BatchPlanArtifact{
		SchemaVersion:   SchemaVersion,
		BatchID:         batchID,
		PlannedAt:       plannedAt.UTC().Format(time.RFC3339),
		Model:           request.Model,
		SessionTopology: request.SessionTopology,
		Seed:            request.Seed,
		Randomized:      resolved.randomize,
		TaskIDs:         append([]string(nil), resolved.taskIDs...),
		ConditionIDs:    append([]string(nil), resolved.conditionIDs...),
		Entries:         entries,
	}
}

func persistBatchPlan(workspace repo.Workspace, plan BatchPlanArtifact) (string, error) {
	batchDir := filepath.Join(workspace.WorkspacePath, repo.LabDirName, "runs", "batches", plan.BatchID)
	planPath := filepath.Join(batchDir, "plan.json")
	if err := os.MkdirAll(batchDir, 0o755); err != nil {
		return "", fmt.Errorf("create batch plan directory %s: %w", batchDir, err)
	}
	if err := writeLedgerJSON(planPath, plan); err != nil {
		return "", err
	}
	return planPath, nil
}

func buildWorkflowFinishPayload(runID string, finishedAt time.Time, task SuiteTask) (string, error) {
	payload := struct {
		SchemaVersion string `json:"schema_version"`
		Metadata      struct {
			AgentID   string `json:"agent_id"`
			SessionID string `json:"session_id"`
			Timestamp string `json:"timestamp"`
		} `json:"metadata"`
		Task             string                           `json:"task"`
		Summary          string                           `json:"summary"`
		Decisions        []workflow.FinishDecision        `json:"decisions"`
		ChangedArtifacts []workflow.FinishChangedArtifact `json:"changed_artifacts"`
		FollowUp         []workflow.FinishFollowUp        `json:"follow_up"`
	}{
		SchemaVersion:    SchemaVersion,
		Task:             task.Description,
		Summary:          fmt.Sprintf("Completed controlled run for %s", task.TaskID),
		Decisions:        []workflow.FinishDecision{},
		ChangedArtifacts: []workflow.FinishChangedArtifact{},
		FollowUp:         []workflow.FinishFollowUp{},
	}
	payload.Metadata.AgentID = "lab-runner"
	payload.Metadata.SessionID = runID
	payload.Metadata.Timestamp = finishedAt.UTC().Format(time.RFC3339)

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode workflow finish payload: %w", err)
	}
	return string(data), nil
}

func buildRunRecord(
	runID string,
	request RunRequest,
	condition Condition,
	startedAt, finishedAt time.Time,
	telemetry *RunTelemetry,
) RunRecord {
	kickoffRef := "artifacts/task-input.json"
	writebackRef := "artifacts/task-output.json"
	if telemetry != nil && telemetry.DelegatedSessions != nil && *telemetry.DelegatedSessions == 2 {
		kickoffRef = "artifacts/kickoff.json"
		writebackRef = "artifacts/writeback.json"
	}
	seed := request.Seed

	return RunRecord{
		SchemaVersion:       SchemaVersion,
		RunID:               runID,
		TaskID:              request.TaskID,
		ConditionID:         request.ConditionID,
		Model:               request.Model,
		SessionTopology:     request.SessionTopology,
		Seed:                &seed,
		PromptVariant:       defaultPromptVariant,
		StartedAt:           startedAt.UTC().Format(time.RFC3339),
		FinishedAt:          finishedAt.UTC().Format(time.RFC3339),
		Telemetry:           telemetry,
		KickoffInputsRef:    kickoffRef,
		SessionStructureRef: "artifacts/sessions/",
		WritebackOutputsRef: writebackRef,
		OutcomeArtifactsRef: fmt.Sprintf("artifacts/output/%s/", condition.ConditionID),
	}
}

func buildRunTelemetry(
	execution RunExecution,
	executionTelemetry *workflow.ExecutionTelemetry,
	wallClockSeconds int,
	outcomeProvided bool,
) *RunTelemetry {
	delegatedSessions := 1
	if execution.GraphBacked {
		delegatedSessions = 2
	}

	telemetry := &RunTelemetry{
		MeasurementStatus: "unavailable",
		Source:            "lab_run",
		WallClockSeconds:  intPointer(wallClockSeconds),
		RetryCount:        intPointer(0),
		DelegatedSessions: intPointer(delegatedSessions),
		IncompleteReasons: []string{"execution_usage_not_supplied"},
	}
	if !outcomeProvided {
		if execution.GraphBacked {
			telemetry.IncompleteReasons = []string{"synthetic_finish_payload_carries_no_execution_usage"}
		} else {
			telemetry.IncompleteReasons = []string{"baseline_run_did_not_receive_execution_outcome"}
		}
		return telemetry
	}
	if executionTelemetry == nil {
		telemetry.Source = workflow.ExecutionTelemetrySourceFinishInput
		telemetry.IncompleteReasons = []string{"outcome_payload_has_no_execution_usage"}
		return telemetry
	}

	telemetry.MeasurementStatus = executionTelemetry.MeasurementStatus
	telemetry.Source = executionTelemetry.Source
	telemetry.Provider = executionTelemetry.Provider
	telemetry.TotalTokens = executionTelemetry.TotalTokens
	telemetry.InputTokens = executionTelemetry.InputTokens
	telemetry.OutputTokens = executionTelemetry.OutputTokens
	telemetry.IncompleteReasons = append([]string(nil), executionTelemetry.IncompleteReasons...)
	if telemetry.Source == "" {
		telemetry.Source = workflow.ExecutionTelemetrySourceFinishInput
	}
	if telemetry.MeasurementStatus == "" {
		telemetry.MeasurementStatus = "unavailable"
	}
	if telemetry.MeasurementStatus != "complete" && len(telemetry.IncompleteReasons) == 0 {
		telemetry.IncompleteReasons = []string{"token_measurement_incomplete"}
	}
	return telemetry
}

func elapsedWallClockSeconds(startedAt, finishedAt time.Time) int {
	elapsed := int(finishedAt.Sub(startedAt).Seconds())
	if elapsed <= 0 {
		return 1
	}
	return elapsed
}

func buildRunArtifacts(
	task SuiteTask,
	request RunRequest,
	condition Condition,
	startedAt, finishedAt time.Time,
	execution RunExecution,
) runArtifacts {
	sessionStructure := map[string]any{
		"session_topology": execution.WorkflowMode,
		"graph_backed":     execution.GraphBacked,
		"kickoff_used":     execution.KickoffUsed,
		"handoff_used":     execution.HandoffUsed,
	}
	if execution.GraphBacked {
		sessionStructure["delegated_sessions"] = 2
	} else {
		sessionStructure["delegated_sessions"] = 1
	}

	return runArtifacts{
		KickoffInputs: map[string]any{
			"task_id":          task.TaskID,
			"task":             task.Description,
			"condition_id":     condition.ConditionID,
			"workflow_mode":    condition.WorkflowMode,
			"model":            request.Model,
			"session_topology": request.SessionTopology,
			"seed":             request.Seed,
		},
		SessionStructure: sessionStructure,
		WritebackOutputs: map[string]any{
			"summary":      fmt.Sprintf("Completed controlled run for %s", task.TaskID),
			"finished_at":  finishedAt.UTC().Format(time.RFC3339),
			"graph_backed": execution.GraphBacked,
		},
		OutcomeSummary: map[string]any{
			"task_id":          task.TaskID,
			"condition_id":     condition.ConditionID,
			"model":            request.Model,
			"session_topology": request.SessionTopology,
			"started_at":       startedAt.UTC().Format(time.RFC3339),
			"finished_at":      finishedAt.UTC().Format(time.RFC3339),
			"status":           RunStatusCompleted,
		},
	}
}

func persistRunLedger(workspace repo.Workspace, record RunRecord, artifacts runArtifacts) (string, error) {
	runDir := filepath.Join(workspace.WorkspacePath, repo.LabDirName, "runs", record.RunID)
	recordPath := filepath.Join(runDir, "run.json")
	artifactsDir := filepath.Join(runDir, "artifacts")

	if _, err := os.Stat(runDir); err == nil {
		return "", duplicateRunIDError(workspace, record.RunID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect run ledger directory %s: %w", runDir, err)
	}

	if err := os.MkdirAll(filepath.Join(artifactsDir, "sessions"), 0o755); err != nil {
		return "", fmt.Errorf("create run ledger directory %s: %w", runDir, err)
	}
	if err := os.MkdirAll(filepath.Join(artifactsDir, "output", record.ConditionID), 0o755); err != nil {
		return "", fmt.Errorf("create outcome artifact directory for %s: %w", record.RunID, err)
	}

	if err := writeLedgerJSON(filepath.Join(runDir, record.KickoffInputsRef), artifacts.KickoffInputs); err != nil {
		return "", err
	}
	if err := writeLedgerJSON(filepath.Join(runDir, filepath.Clean(record.SessionStructureRef), "summary.json"), artifacts.SessionStructure); err != nil {
		return "", err
	}
	if err := writeLedgerJSON(filepath.Join(runDir, record.WritebackOutputsRef), artifacts.WritebackOutputs); err != nil {
		return "", err
	}
	if err := writeLedgerJSON(filepath.Join(runDir, filepath.Clean(record.OutcomeArtifactsRef), "summary.json"), artifacts.OutcomeSummary); err != nil {
		return "", err
	}
	if err := writeLedgerJSON(recordPath, record); err != nil {
		return "", err
	}

	return recordPath, nil
}

func writeLedgerJSON(path string, payload any) error {
	data, err := marshalJSON(payload)
	if err != nil {
		return fmt.Errorf("encode ledger artifact %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write ledger artifact %s: %w", path, err)
	}
	return nil
}

func duplicateRunIDError(workspace repo.Workspace, runID string) error {
	runDir := filepath.Join(workspace.WorkspacePath, repo.LabDirName, "runs", runID)
	return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "lab_error",
		Code:     "duplicate_run_id",
		Message:  "run ledger already contains this run_id",
		Details: map[string]any{
			"run_id":   runID,
			"run_path": relativePath(workspace.RepoRoot, runDir),
		},
	}, fmt.Errorf("run ledger already contains run_id %q", runID))
}

func nextRunID(runsPath string, now time.Time) string {
	base := "run-" + strings.ToLower(now.UTC().Format("20060102T150405Z"))
	candidate := base
	for index := 2; ; index++ {
		if _, err := os.Stat(filepath.Join(runsPath, candidate)); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%02d", base, index)
	}
}

func nextBatchID(batchesPath string, now time.Time) string {
	base := "batch-" + strings.ToLower(now.UTC().Format("20060102T150405Z"))
	candidate := base
	for index := 2; ; index++ {
		if _, err := os.Stat(filepath.Join(batchesPath, candidate)); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%02d", base, index)
	}
}

func intPointer(value int) *int {
	return &value
}

func boolPointer(value bool) *bool {
	return &value
}
