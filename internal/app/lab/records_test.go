package lab

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func TestServiceValidateRunRecordAcceptsCompleteRecord(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)

	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	suitePath := filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, repo.LabSuiteManifestName)
	writeJSONFixture(t, suitePath, SuiteManifest{
		SchemaVersion: SchemaVersion,
		SuiteID:       "delegated-workflow-evidence-v1",
		Tasks: []SuiteTask{
			{
				TaskID:                "task-001",
				Family:                "delegated-non-trivial-subtask",
				Description:           "implement retrieval ranking adjustment",
				AcceptanceCriteriaRef: "tasks/task-001/criteria.md",
			},
		},
	})

	record := RunRecord{
		SchemaVersion:       SchemaVersion,
		RunID:               "run-001",
		TaskID:              "task-001",
		ConditionID:         "with-graph",
		Model:               "claude-sonnet-4-20250514",
		SessionTopology:     "delegated-parallel",
		Seed:                int64Ptr(42),
		PromptVariant:       "default",
		StartedAt:           "2026-04-01T10:30:00Z",
		FinishedAt:          "2026-04-01T10:45:00Z",
		Telemetry:           validRunTelemetry(),
		KickoffInputsRef:    "artifacts/kickoff.json",
		SessionStructureRef: "artifacts/sessions/",
		WritebackOutputsRef: "artifacts/writeback.json",
		OutcomeArtifactsRef: "artifacts/output/",
	}

	if err := service.ValidateRunRecord(context.Background(), repoDir, record); err != nil {
		t.Fatalf("ValidateRunRecord returned error: %v", err)
	}
}

func TestServiceValidateRunRecordRejectsMissingRequiredTelemetryFields(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)

	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	suitePath := filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, repo.LabSuiteManifestName)
	writeJSONFixture(t, suitePath, SuiteManifest{
		SchemaVersion: SchemaVersion,
		SuiteID:       "delegated-workflow-evidence-v1",
		Tasks: []SuiteTask{
			{
				TaskID:                "task-001",
				Family:                "delegated-non-trivial-subtask",
				Description:           "implement retrieval ranking adjustment",
				AcceptanceCriteriaRef: "tasks/task-001/criteria.md",
			},
		},
	})

	record := RunRecord{
		SchemaVersion:   SchemaVersion,
		RunID:           "run-001",
		TaskID:          "task-001",
		ConditionID:     "with-graph",
		Model:           "claude-sonnet-4-20250514",
		SessionTopology: "delegated-parallel",
		Seed:            int64Ptr(42),
		PromptVariant:   "default",
		StartedAt:       "2026-04-01T10:30:00Z",
		FinishedAt:      "2026-04-01T10:45:00Z",
		Telemetry: &RunTelemetry{
			MeasurementStatus: "unavailable",
			Source:            "lab_run",
			InputTokens:       intPtr(8100),
			OutputTokens:      intPtr(6100),
			RetryCount:        intPtr(1),
			DelegatedSessions: intPtr(2),
			IncompleteReasons: []string{"token_measurement_incomplete"},
		},
		KickoffInputsRef:    "artifacts/kickoff.json",
		SessionStructureRef: "artifacts/sessions/",
		WritebackOutputsRef: "artifacts/writeback.json",
		OutcomeArtifactsRef: "artifacts/output/",
	}

	err := service.ValidateRunRecord(context.Background(), repoDir, record)
	if err == nil {
		t.Fatal("expected ValidateRunRecord to return an error")
	}

	detail, ok := cmdsupport.ErrorDetailFromError(err)
	if !ok {
		t.Fatalf("error = %T, want structured command error", err)
	}
	if detail.Category != "validation_error" {
		t.Fatalf("error.category = %q, want validation_error", detail.Category)
	}
	if detail.Type != "record_error" {
		t.Fatalf("error.type = %q, want record_error", detail.Type)
	}
	if detail.Code != "record_validation_failed" {
		t.Fatalf("error.code = %q, want record_validation_failed", detail.Code)
	}
	assertViolationPresent(t, detail, "telemetry.wall_clock_seconds", "field is required")
}

func TestServiceValidateRunRecordRejectsDanglingTaskAndConditionReferences(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)

	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	record := RunRecord{
		SchemaVersion:       SchemaVersion,
		RunID:               "run-001",
		TaskID:              "task-missing",
		ConditionID:         "condition-missing",
		Model:               "claude-sonnet-4-20250514",
		SessionTopology:     "delegated-parallel",
		Seed:                int64Ptr(42),
		PromptVariant:       "default",
		StartedAt:           "2026-04-01T10:30:00Z",
		FinishedAt:          "2026-04-01T10:45:00Z",
		Telemetry:           validRunTelemetry(),
		KickoffInputsRef:    "artifacts/kickoff.json",
		SessionStructureRef: "artifacts/sessions/",
		WritebackOutputsRef: "artifacts/writeback.json",
		OutcomeArtifactsRef: "artifacts/output/",
	}

	err := service.ValidateRunRecord(context.Background(), repoDir, record)
	if err == nil {
		t.Fatal("expected ValidateRunRecord to return an error")
	}

	detail, ok := cmdsupport.ErrorDetailFromError(err)
	if !ok {
		t.Fatalf("error = %T, want structured command error", err)
	}
	assertViolationPresent(t, detail, "task_id", "task_id must reference an existing suite task")
	assertViolationPresent(t, detail, "condition_id", "condition_id must reference an existing condition")
}

func TestServiceValidateEvaluationRecordAcceptsCompleteRecord(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)

	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	runRecordPath := filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "runs", "run-001.json")
	writeFixture(t, runRecordPath, "{\"run_id\":\"run-001\"}\n")

	record := EvaluationRecord{
		SchemaVersion: SchemaVersion,
		RunID:         "run-001",
		Evaluator:     "automated:rubric-v1",
		EvaluatedAt:   "2026-04-01T11:00:00Z",
		Scores: &EvaluationScores{
			Success:                boolPtr(true),
			QualityScore:           float64Ptr(0.85),
			ResumabilityScore:      float64Ptr(0.90),
			HumanInterventionCount: intPtr(0),
		},
		Notes: "acceptance criteria fully met",
	}

	if err := service.ValidateEvaluationRecord(context.Background(), repoDir, record); err != nil {
		t.Fatalf("ValidateEvaluationRecord returned error: %v", err)
	}
}

func TestServiceValidateEvaluationRecordAcceptsNestedRunLedgerPath(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)

	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	runRecordPath := filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "runs", "run-001", "run.json")
	writeFixture(t, runRecordPath, "{\"run_id\":\"run-001\"}\n")

	record := EvaluationRecord{
		SchemaVersion: SchemaVersion,
		RunID:         "run-001",
		Evaluator:     "human:alice",
		EvaluatedAt:   "2026-04-01T11:00:00Z",
		Scores: &EvaluationScores{
			Success:                boolPtr(true),
			QualityScore:           float64Ptr(0.85),
			ResumabilityScore:      float64Ptr(0.90),
			HumanInterventionCount: intPtr(0),
		},
	}

	if err := service.ValidateEvaluationRecord(context.Background(), repoDir, record); err != nil {
		t.Fatalf("ValidateEvaluationRecord returned error: %v", err)
	}
}

func TestServiceValidateEvaluationRecordRejectsDanglingRunReference(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)

	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	record := EvaluationRecord{
		SchemaVersion: SchemaVersion,
		RunID:         "run-missing",
		Evaluator:     "automated:rubric-v1",
		EvaluatedAt:   "2026-04-01T11:00:00Z",
		Scores: &EvaluationScores{
			Success:                boolPtr(false),
			QualityScore:           float64Ptr(0.20),
			ResumabilityScore:      float64Ptr(0.10),
			HumanInterventionCount: intPtr(2),
		},
	}

	err := service.ValidateEvaluationRecord(context.Background(), repoDir, record)
	if err == nil {
		t.Fatal("expected ValidateEvaluationRecord to return an error")
	}

	detail, ok := cmdsupport.ErrorDetailFromError(err)
	if !ok {
		t.Fatalf("error = %T, want structured command error", err)
	}
	if detail.Category != "validation_error" {
		t.Fatalf("error.category = %q, want validation_error", detail.Category)
	}
	if detail.Type != "record_error" {
		t.Fatalf("error.type = %q, want record_error", detail.Type)
	}
	if detail.Code != "record_validation_failed" {
		t.Fatalf("error.code = %q, want record_validation_failed", detail.Code)
	}
	assertViolationPresent(t, detail, "run_id", "run_id must reference an existing run record")
}

func validRunTelemetry() *RunTelemetry {
	return &RunTelemetry{
		MeasurementStatus: "complete",
		Source:            "workflow_finish_payload",
		Provider:          "copilot-cli",
		TotalTokens:       intPtr(14200),
		InputTokens:       intPtr(8100),
		OutputTokens:      intPtr(6100),
		WallClockSeconds:  intPtr(900),
		RetryCount:        intPtr(1),
		DelegatedSessions: intPtr(2),
	}
}

func intPtr(value int) *int {
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
