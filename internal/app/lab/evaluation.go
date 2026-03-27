package lab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

type EvaluationArtifact struct {
	SchemaVersion string             `json:"schema_version"`
	RunID         string             `json:"run_id"`
	Records       []EvaluationRecord `json:"records"`
}

type EvaluateResult struct {
	RunID           string           `json:"run_id"`
	ArtifactPath    string           `json:"artifact_path"`
	EvaluationCount int              `json:"evaluation_count"`
	Latest          EvaluationRecord `json:"latest"`
}

type PresentEvaluationRequest struct {
	RunID string `json:"run_id"`
	Blind bool   `json:"blind"`
}

type PresentEvaluationResult struct {
	RunID         string                    `json:"run_id"`
	Blind         bool                      `json:"blind"`
	RunRecordPath string                    `json:"run_record_path"`
	Task          EvaluationTaskInput       `json:"task"`
	Condition     *EvaluationConditionInput `json:"condition,omitempty"`
	Artifacts     EvaluationArtifactInput   `json:"artifacts"`
	HiddenFields  []string                  `json:"hidden_fields,omitempty"`
}

type EvaluationTaskInput struct {
	TaskID                string `json:"task_id"`
	Description           string `json:"description"`
	AcceptanceCriteriaRef string `json:"acceptance_criteria_ref"`
}

type EvaluationConditionInput struct {
	ConditionID  string `json:"condition_id"`
	WorkflowMode string `json:"workflow_mode"`
}

type EvaluationArtifactInput struct {
	WritebackOutputs any `json:"writeback_outputs,omitempty"`
	OutcomeSummary   any `json:"outcome_summary,omitempty"`
}

func (s *Service) Evaluate(ctx context.Context, startDir string, record EvaluationRecord) (EvaluateResult, error) {
	if s == nil || s.manager == nil {
		return EvaluateResult{}, errors.New("lab service is not configured")
	}
	if s.now == nil {
		s.now = func() time.Time { return time.Now().UTC() }
	}

	record.SchemaVersion = strings.TrimSpace(record.SchemaVersion)
	record.RunID = strings.TrimSpace(record.RunID)
	record.Evaluator = strings.TrimSpace(record.Evaluator)
	record.Notes = strings.TrimSpace(record.Notes)
	if record.SchemaVersion == "" {
		record.SchemaVersion = SchemaVersion
	}
	if strings.TrimSpace(record.EvaluatedAt) == "" {
		record.EvaluatedAt = s.now().UTC().Format(time.RFC3339)
	}

	workspace, err := s.manager.OpenWorkspace(ctx, startDir)
	if err != nil {
		return EvaluateResult{}, err
	}

	evaluationsPath := filepath.Join(workspace.WorkspacePath, repo.LabDirName, "evaluations")
	runsPath := filepath.Join(workspace.WorkspacePath, repo.LabDirName, "runs")
	if err := validateEvaluationRecord(evaluationsPath, runsPath, record); err != nil {
		return EvaluateResult{}, err
	}

	artifactPath := filepath.Join(evaluationsPath, record.RunID+".json")
	artifact, err := loadEvaluationArtifact(artifactPath, record.RunID)
	if err != nil {
		return EvaluateResult{}, err
	}
	artifact.Records = append(artifact.Records, record)
	if err := writeLedgerJSON(artifactPath, artifact); err != nil {
		return EvaluateResult{}, err
	}

	return EvaluateResult{
		RunID:           record.RunID,
		ArtifactPath:    relativePath(workspace.RepoRoot, artifactPath),
		EvaluationCount: len(artifact.Records),
		Latest:          record,
	}, nil
}

func (s *Service) PresentEvaluationInput(ctx context.Context, startDir string, request PresentEvaluationRequest) (PresentEvaluationResult, error) {
	if s == nil || s.manager == nil {
		return PresentEvaluationResult{}, errors.New("lab service is not configured")
	}

	request.RunID = strings.TrimSpace(request.RunID)
	if request.RunID == "" {
		return PresentEvaluationResult{}, cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "record_error",
			Code:     "record_validation_failed",
			Message:  "evaluation input request validation failed",
			Details: map[string]any{
				"record_kind": "evaluation_input",
				"violations": []map[string]any{
					violation("run_id", "field is required"),
				},
			},
		}, fmt.Errorf("evaluation input request validation failed"))
	}
	if !isSafeIdentifier(request.RunID) {
		return PresentEvaluationResult{}, cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "record_error",
			Code:     "record_validation_failed",
			Message:  "evaluation input request validation failed",
			Details: map[string]any{
				"record_kind": "evaluation_input",
				"violations": []map[string]any{
					violationWithValue("run_id", "must be a simple identifier without path separators", request.RunID),
				},
			},
		}, fmt.Errorf("evaluation input request validation failed"))
	}

	workspace, err := s.manager.OpenWorkspace(ctx, startDir)
	if err != nil {
		return PresentEvaluationResult{}, err
	}

	runsPath := filepath.Join(workspace.WorkspacePath, repo.LabDirName, "runs")
	runRecordPath, err := resolveRunRecordPath(runsPath, request.RunID)
	if err != nil {
		return PresentEvaluationResult{}, err
	}

	record, err := loadRunRecordFromPath(runRecordPath)
	if err != nil {
		return PresentEvaluationResult{}, err
	}

	suitePath := filepath.Join(workspace.WorkspacePath, repo.LabDirName, repo.LabSuiteManifestName)
	suite, err := loadSuiteManifest(suitePath)
	if err != nil {
		return PresentEvaluationResult{}, err
	}
	conditionsPath := filepath.Join(workspace.WorkspacePath, repo.LabDirName, repo.LabConditionsManifestName)
	conditions, err := loadConditionsManifest(conditionsPath)
	if err != nil {
		return PresentEvaluationResult{}, err
	}

	task, ok := suite.TaskByID(record.TaskID)
	if !ok {
		return PresentEvaluationResult{}, cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "record_error",
			Code:     "record_validation_failed",
			Message:  "evaluation input request validation failed",
			Details: map[string]any{
				"record_kind": "evaluation_input",
				"violations": []map[string]any{
					violationWithValue("task_id", "task_id must reference an existing suite task", record.TaskID),
				},
			},
		}, fmt.Errorf("evaluation input request validation failed"))
	}

	writebackOutputs, err := loadOptionalJSONArtifact(filepath.Join(filepath.Dir(runRecordPath), filepath.Clean(record.WritebackOutputsRef)))
	if err != nil {
		return PresentEvaluationResult{}, err
	}
	outcomeSummary, err := loadOptionalJSONArtifact(filepath.Join(filepath.Dir(runRecordPath), filepath.Clean(record.OutcomeArtifactsRef), "summary.json"))
	if err != nil {
		return PresentEvaluationResult{}, err
	}

	condition, _ := conditions.ConditionByID(record.ConditionID)
	result := PresentEvaluationResult{
		RunID:         record.RunID,
		Blind:         request.Blind,
		RunRecordPath: relativePath(workspace.RepoRoot, runRecordPath),
		Task: EvaluationTaskInput{
			TaskID:                task.TaskID,
			Description:           task.Description,
			AcceptanceCriteriaRef: task.AcceptanceCriteriaRef,
		},
		Artifacts: EvaluationArtifactInput{
			WritebackOutputs: writebackOutputs,
			OutcomeSummary:   outcomeSummary,
		},
	}

	if request.Blind {
		result.HiddenFields = []string{
			"condition_id",
			"workflow_mode",
			"graph_backed",
			"kickoff_used",
			"handoff_used",
			"delegated_sessions",
		}
		result.Artifacts.WritebackOutputs = sanitizeBlindedValue(result.Artifacts.WritebackOutputs)
		result.Artifacts.OutcomeSummary = sanitizeBlindedValue(result.Artifacts.OutcomeSummary)
		return result, nil
	}

	result.Condition = &EvaluationConditionInput{
		ConditionID:  record.ConditionID,
		WorkflowMode: condition.WorkflowMode,
	}
	return result, nil
}

func loadEvaluationArtifact(path, runID string) (EvaluationArtifact, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return EvaluationArtifact{
				SchemaVersion: SchemaVersion,
				RunID:         runID,
				Records:       []EvaluationRecord{},
			}, nil
		}
		return EvaluationArtifact{}, fmt.Errorf("read evaluation artifact %s: %w", path, err)
	}

	var artifact EvaluationArtifact
	if err := json.Unmarshal(payload, &artifact); err != nil {
		return EvaluationArtifact{}, cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "record_error",
			Code:     "record_parse_failed",
			Message:  "evaluation artifact could not be parsed",
			Details: map[string]any{
				"path":   path,
				"reason": err.Error(),
			},
		}, fmt.Errorf("parse evaluation artifact %s: %w", path, err))
	}
	if artifact.SchemaVersion == "" {
		artifact.SchemaVersion = SchemaVersion
	}
	if artifact.RunID == "" {
		artifact.RunID = runID
	}
	if artifact.RunID != runID {
		return EvaluationArtifact{}, cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "record_error",
			Code:     "record_validation_failed",
			Message:  "evaluation artifact validation failed",
			Details: map[string]any{
				"record_kind": "evaluation_artifact",
				"path":        path,
				"violations": []map[string]any{
					violationWithValue("run_id", "run_id must match the target evaluation artifact", artifact.RunID),
				},
			},
		}, fmt.Errorf("evaluation artifact validation failed"))
	}
	if artifact.Records == nil {
		artifact.Records = []EvaluationRecord{}
	}
	return artifact, nil
}

func resolveRunRecordPath(runsPath, runID string) (string, error) {
	candidates := runRecordCandidatePaths(runsPath, runID)
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}

	return "", cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
		Category: "validation_error",
		Type:     "record_error",
		Code:     "record_validation_failed",
		Message:  "evaluation input request validation failed",
		Details: map[string]any{
			"record_kind": "evaluation_input",
			"violations": []map[string]any{
				func() map[string]any {
					entry := violationWithValue("run_id", "run_id must reference an existing run record", runID)
					entry["checked_paths"] = candidates
					return entry
				}(),
			},
		},
	}, fmt.Errorf("evaluation input request validation failed"))
}

func loadRunRecordFromPath(path string) (RunRecord, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return RunRecord{}, fmt.Errorf("read run record %s: %w", path, err)
	}

	var record RunRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		return RunRecord{}, cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "record_error",
			Code:     "record_parse_failed",
			Message:  "run record could not be parsed",
			Details: map[string]any{
				"path":   path,
				"reason": err.Error(),
			},
		}, fmt.Errorf("parse run record %s: %w", path, err))
	}
	return record, nil
}

func loadOptionalJSONArtifact(path string) (any, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read evaluation source artifact %s: %w", path, err)
	}

	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		return nil, cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "record_error",
			Code:     "record_parse_failed",
			Message:  "evaluation source artifact could not be parsed",
			Details: map[string]any{
				"path":   path,
				"reason": err.Error(),
			},
		}, fmt.Errorf("parse evaluation source artifact %s: %w", path, err))
	}
	return value, nil
}

func sanitizeBlindedValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		sanitized := make(map[string]any, len(typed))
		for key, entry := range typed {
			if isConditionRevealingKey(key) {
				continue
			}
			sanitized[key] = sanitizeBlindedValue(entry)
		}
		return sanitized
	case []any:
		items := make([]any, 0, len(typed))
		for _, entry := range typed {
			items = append(items, sanitizeBlindedValue(entry))
		}
		return items
	default:
		return value
	}
}

func isConditionRevealingKey(key string) bool {
	switch key {
	case "condition_id", "workflow_mode", "graph_backed", "kickoff_used", "handoff_used", "delegated_sessions":
		return true
	default:
		return false
	}
}
