package lab

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

type RunRecord struct {
	SchemaVersion             string        `json:"schema_version"`
	RunID                     string        `json:"run_id"`
	TaskID                    string        `json:"task_id"`
	ConditionID               string        `json:"condition_id"`
	Model                     string        `json:"model"`
	SessionTopology           string        `json:"session_topology"`
	Seed                      *int64        `json:"seed"`
	PromptVariant             string        `json:"prompt_variant"`
	StartedAt                 string        `json:"started_at"`
	FinishedAt                string        `json:"finished_at"`
	Telemetry                 *RunTelemetry `json:"telemetry"`
	KickoffInputsRef          string        `json:"kickoff_inputs_ref"`
	WorkflowStartResponseRef  string        `json:"workflow_start_response_ref,omitempty"`
	BaselinePromptMetadataRef string        `json:"baseline_prompt_metadata_ref,omitempty"`
	SessionStructureRef       string        `json:"session_structure_ref"`
	WritebackOutputsRef       string        `json:"writeback_outputs_ref"`
	OutcomeArtifactsRef       string        `json:"outcome_artifacts_ref"`
}

type RunTelemetry struct {
	MeasurementStatus string   `json:"measurement_status"`
	Source            string   `json:"source"`
	Provider          string   `json:"provider,omitempty"`
	TotalTokens       *int     `json:"total_tokens,omitempty"`
	InputTokens       *int     `json:"input_tokens,omitempty"`
	OutputTokens      *int     `json:"output_tokens,omitempty"`
	WallClockSeconds  *int     `json:"wall_clock_seconds"`
	RetryCount        *int     `json:"retry_count"`
	DelegatedSessions *int     `json:"delegated_sessions"`
	IncompleteReasons []string `json:"incomplete_reasons,omitempty"`
}

type EvaluationRecord struct {
	SchemaVersion string            `json:"schema_version"`
	RunID         string            `json:"run_id"`
	Evaluator     string            `json:"evaluator"`
	EvaluatedAt   string            `json:"evaluated_at"`
	Scores        *EvaluationScores `json:"scores"`
	Notes         string            `json:"notes,omitempty"`
}

type EvaluationScores struct {
	Success                *bool    `json:"success"`
	QualityScore           *float64 `json:"quality_score"`
	ResumabilityScore      *float64 `json:"resumability_score"`
	HumanInterventionCount *int     `json:"human_intervention_count"`
}

func (s *Service) ValidateRunRecord(ctx context.Context, startDir string, record RunRecord) error {
	if s == nil || s.manager == nil {
		return errors.New("lab service is not configured")
	}

	workspace, err := s.manager.OpenWorkspace(ctx, startDir)
	if err != nil {
		return err
	}

	suitePath := filepath.Join(workspace.WorkspacePath, repo.LabDirName, repo.LabSuiteManifestName)
	suite, err := loadSuiteManifest(suitePath)
	if err != nil {
		return err
	}

	conditionsPath := filepath.Join(workspace.WorkspacePath, repo.LabDirName, repo.LabConditionsManifestName)
	conditions, err := loadConditionsManifest(conditionsPath)
	if err != nil {
		return err
	}

	return validateRunRecord(filepath.Join(workspace.WorkspacePath, repo.LabDirName, "runs"), suite, conditions, record)
}

func (s *Service) ValidateEvaluationRecord(ctx context.Context, startDir string, record EvaluationRecord) error {
	if s == nil || s.manager == nil {
		return errors.New("lab service is not configured")
	}

	workspace, err := s.manager.OpenWorkspace(ctx, startDir)
	if err != nil {
		return err
	}

	return validateEvaluationRecord(filepath.Join(workspace.WorkspacePath, repo.LabDirName, "evaluations"), filepath.Join(workspace.WorkspacePath, repo.LabDirName, "runs"), record)
}

func validateRunRecord(recordsPath string, suite SuiteManifest, conditions ConditionsManifest, record RunRecord) error {
	var violations []map[string]any

	validateSchemaVersion(record.SchemaVersion, &violations)
	validateIdentifier("run_id", record.RunID, &violations)

	if record.TaskID == "" {
		violations = append(violations, violation("task_id", "field is required"))
	} else if _, ok := suite.TaskByID(record.TaskID); !ok {
		violations = append(violations, violationWithValue("task_id", "task_id must reference an existing suite task", record.TaskID))
	}

	if record.ConditionID == "" {
		violations = append(violations, violation("condition_id", "field is required"))
	} else if _, ok := conditions.ConditionByID(record.ConditionID); !ok {
		violations = append(violations, violationWithValue("condition_id", "condition_id must reference an existing condition", record.ConditionID))
	}

	if record.Model == "" {
		violations = append(violations, violation("model", "field is required"))
	}
	if record.SessionTopology == "" {
		violations = append(violations, violation("session_topology", "field is required"))
	}
	if record.Seed == nil {
		violations = append(violations, violation("seed", "field is required"))
	}
	if record.PromptVariant == "" {
		violations = append(violations, violation("prompt_variant", "field is required"))
	}

	startedAt, startedAtOK := validateRFC3339Timestamp("started_at", record.StartedAt, &violations)
	finishedAt, finishedAtOK := validateRFC3339Timestamp("finished_at", record.FinishedAt, &violations)
	if startedAtOK && finishedAtOK && finishedAt.Before(startedAt) {
		violations = append(violations, violation("finished_at", "finished_at must not be earlier than started_at"))
	}

	if record.Telemetry == nil {
		violations = append(violations, violation("telemetry", "field is required"))
	} else {
		validateTelemetryStatus(record.Telemetry.MeasurementStatus, &violations)
		if strings.TrimSpace(record.Telemetry.Source) == "" {
			violations = append(violations, violation("telemetry.source", "field is required"))
		}
		validateOptionalNonNegativeInt("telemetry.total_tokens", record.Telemetry.TotalTokens, &violations)
		if record.Telemetry.TotalTokens != nil && *record.Telemetry.TotalTokens <= 0 {
			violations = append(violations, violation("telemetry.total_tokens", "must be greater than zero"))
		}
		validateOptionalNonNegativeInt("telemetry.input_tokens", record.Telemetry.InputTokens, &violations)
		validateOptionalNonNegativeInt("telemetry.output_tokens", record.Telemetry.OutputTokens, &violations)
		validateRequiredNonNegativeInt("telemetry.wall_clock_seconds", record.Telemetry.WallClockSeconds, &violations)
		if record.Telemetry.WallClockSeconds != nil && *record.Telemetry.WallClockSeconds <= 0 {
			violations = append(violations, violation("telemetry.wall_clock_seconds", "must be greater than zero"))
		}
		validateRequiredNonNegativeInt("telemetry.retry_count", record.Telemetry.RetryCount, &violations)
		validateRequiredNonNegativeInt("telemetry.delegated_sessions", record.Telemetry.DelegatedSessions, &violations)
		switch record.Telemetry.MeasurementStatus {
		case "complete":
			validateRequiredNonNegativeInt("telemetry.total_tokens", record.Telemetry.TotalTokens, &violations)
			validateRequiredNonNegativeInt("telemetry.input_tokens", record.Telemetry.InputTokens, &violations)
			validateRequiredNonNegativeInt("telemetry.output_tokens", record.Telemetry.OutputTokens, &violations)
		case "partial", "unavailable":
			if len(record.Telemetry.IncompleteReasons) == 0 {
				violations = append(violations, violation("telemetry.incomplete_reasons", "field is required when token telemetry is not complete"))
			}
		}
	}

	validateArtifactReference("kickoff_inputs_ref", record.KickoffInputsRef, &violations)
	if record.WorkflowStartResponseRef != "" {
		validateArtifactReference("workflow_start_response_ref", record.WorkflowStartResponseRef, &violations)
	}
	if record.BaselinePromptMetadataRef != "" {
		validateArtifactReference("baseline_prompt_metadata_ref", record.BaselinePromptMetadataRef, &violations)
	}
	validateArtifactReference("session_structure_ref", record.SessionStructureRef, &violations)
	validateArtifactReference("writeback_outputs_ref", record.WritebackOutputsRef, &violations)
	validateArtifactReference("outcome_artifacts_ref", record.OutcomeArtifactsRef, &violations)

	if len(violations) > 0 {
		return recordValidationError("run", recordsPath, violations)
	}
	return nil
}

func validateEvaluationRecord(recordsPath, runsPath string, record EvaluationRecord) error {
	var violations []map[string]any

	validateSchemaVersion(record.SchemaVersion, &violations)
	validateIdentifier("run_id", record.RunID, &violations)
	if record.RunID != "" && isSafeIdentifier(record.RunID) {
		if ok, checkedPaths := runRecordExists(runsPath, record.RunID); !ok {
			entry := violationWithValue("run_id", "run_id must reference an existing run record", record.RunID)
			entry["checked_paths"] = checkedPaths
			violations = append(violations, entry)
		}
	}
	if record.Evaluator == "" {
		violations = append(violations, violation("evaluator", "field is required"))
	}
	validateRFC3339Timestamp("evaluated_at", record.EvaluatedAt, &violations)

	if record.Scores == nil {
		violations = append(violations, violation("scores", "field is required"))
	} else {
		if record.Scores.Success == nil {
			violations = append(violations, violation("scores.success", "field is required"))
		}
		validateScore("scores.quality_score", record.Scores.QualityScore, &violations)
		validateScore("scores.resumability_score", record.Scores.ResumabilityScore, &violations)
		validateRequiredNonNegativeInt("scores.human_intervention_count", record.Scores.HumanInterventionCount, &violations)
	}

	if len(violations) > 0 {
		return recordValidationError("evaluation", recordsPath, violations)
	}
	return nil
}

func validateSchemaVersion(version string, violations *[]map[string]any) {
	if version == "" {
		*violations = append(*violations, violation("schema_version", "field is required"))
		return
	}
	if version != SchemaVersion {
		*violations = append(*violations, violationWithAllowed("schema_version", "unsupported schema version", version, []string{SchemaVersion}))
	}
}

func validateRFC3339Timestamp(field, value string, violations *[]map[string]any) (time.Time, bool) {
	if value == "" {
		*violations = append(*violations, violation(field, "field is required"))
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		entry := violationWithValue(field, "must be a valid RFC3339 timestamp", value)
		entry["reason"] = err.Error()
		*violations = append(*violations, entry)
		return time.Time{}, false
	}
	return parsed, true
}

func validateRequiredNonNegativeInt(field string, value *int, violations *[]map[string]any) {
	if value == nil {
		*violations = append(*violations, violation(field, "field is required"))
		return
	}
	if *value < 0 {
		*violations = append(*violations, violationWithValue(field, "must be greater than or equal to zero", *value))
	}
}

func validateOptionalNonNegativeInt(field string, value *int, violations *[]map[string]any) {
	if value == nil {
		return
	}
	if *value < 0 {
		*violations = append(*violations, violationWithValue(field, "must be greater than or equal to zero", *value))
	}
}

func validateTelemetryStatus(status string, violations *[]map[string]any) {
	if strings.TrimSpace(status) == "" {
		*violations = append(*violations, violation("telemetry.measurement_status", "field is required"))
		return
	}
	switch status {
	case "complete", "partial", "unavailable":
		return
	default:
		*violations = append(*violations, violationWithAllowed("telemetry.measurement_status", "measurement_status must be recognized", status, []string{"complete", "partial", "unavailable"}))
	}
}

func validateScore(field string, value *float64, violations *[]map[string]any) {
	if value == nil {
		*violations = append(*violations, violation(field, "field is required"))
		return
	}
	if *value < 0 || *value > 1 {
		*violations = append(*violations, violationWithValue(field, "must be between 0 and 1", *value))
	}
}

func validateIdentifier(field, value string, violations *[]map[string]any) {
	if value == "" {
		*violations = append(*violations, violation(field, "field is required"))
		return
	}
	if !isSafeIdentifier(value) {
		*violations = append(*violations, violationWithValue(field, "must be a simple identifier without path separators", value))
	}
}

func isSafeIdentifier(value string) bool {
	if value == "" || value == "." || value == ".." {
		return false
	}
	if strings.Contains(value, "/") || strings.Contains(value, "\\") {
		return false
	}
	return filepath.Base(value) == value
}

func validateArtifactReference(field, value string, violations *[]map[string]any) {
	if value == "" {
		*violations = append(*violations, violation(field, "field is required"))
		return
	}
	if filepath.IsAbs(value) {
		*violations = append(*violations, violationWithValue(field, "must be a relative path", value))
		return
	}

	clean := filepath.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		*violations = append(*violations, violationWithValue(field, "must stay within the run artifact directory", value))
	}
}

func runRecordExists(runsPath, runID string) (bool, []string) {
	candidates := runRecordCandidatePaths(runsPath, runID)
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return true, candidates
		}
	}
	return false, candidates
}

func runRecordCandidatePaths(runsPath, runID string) []string {
	return []string{
		filepath.Join(runsPath, runID, "run.json"),
		filepath.Join(runsPath, runID+".json"),
		filepath.Join(runsPath, runID, "record.json"),
	}
}

func recordValidationError(kind, path string, violations []map[string]any) error {
	return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
		Category: "validation_error",
		Type:     "record_error",
		Code:     "record_validation_failed",
		Message:  fmt.Sprintf("%s record validation failed", kind),
		Details: map[string]any{
			"record_kind": kind,
			"path":        path,
			"violations":  violations,
		},
	}, fmt.Errorf("%s record validation failed", kind))
}

func violationWithValue(field, message string, value any) map[string]any {
	entry := violation(field, message)
	entry["value"] = value
	return entry
}
