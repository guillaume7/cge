package lab

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func TestServiceEvaluatePersistsSeparateArtifactAndPreservesRunLedger(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeSuiteWithSingleTask(t, repoDir)
	service.NowForTest(func() time.Time {
		return time.Date(2026, 4, 1, 10, 30, 0, 0, time.UTC)
	})
	service.WorkflowRunnerForTest(&stubWorkflowRunner{})

	runResult, err := service.Run(context.Background(), repoDir, RunRequest{
		TaskID:          "task-001",
		ConditionID:     "with-graph",
		Model:           "claude-sonnet",
		SessionTopology: "delegated-parallel",
		Seed:            42,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	runRecordPath := filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "runs", runResult.RunID, "run.json")
	before := readFile(t, runRecordPath)

	service.NowForTest(func() time.Time {
		return time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC)
	})

	result, err := service.Evaluate(context.Background(), repoDir, EvaluationRecord{
		RunID:     runResult.RunID,
		Evaluator: "human:alice",
		Scores: &EvaluationScores{
			Success:                boolPtr(true),
			QualityScore:           float64Ptr(0.9),
			ResumabilityScore:      float64Ptr(0.8),
			HumanInterventionCount: intPtr(1),
		},
		Notes: "strong implementation",
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}

	if got, want := result.ArtifactPath, ".graph/lab/evaluations/"+runResult.RunID+".json"; got != want {
		t.Fatalf("artifact_path = %q, want %q", got, want)
	}
	if result.EvaluationCount != 1 {
		t.Fatalf("evaluation_count = %d, want 1", result.EvaluationCount)
	}
	if result.Latest.EvaluatedAt != "2026-04-01T11:00:00Z" {
		t.Fatalf("evaluated_at = %q, want 2026-04-01T11:00:00Z", result.Latest.EvaluatedAt)
	}

	artifact := readEvaluationArtifact(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "evaluations", runResult.RunID+".json"))
	if len(artifact.Records) != 1 {
		t.Fatalf("evaluation records = %#v, want one record", artifact.Records)
	}
	if artifact.Records[0].Evaluator != "human:alice" {
		t.Fatalf("evaluator = %q, want human:alice", artifact.Records[0].Evaluator)
	}

	after := readFile(t, runRecordPath)
	if string(after) != string(before) {
		t.Fatalf("run record changed after evaluation\nbefore=%s\nafter=%s", before, after)
	}
}

func TestServicePresentEvaluationInputBlindsConditionMetadata(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeSuiteWithSingleTask(t, repoDir)
	service.NowForTest(func() time.Time {
		return time.Date(2026, 4, 1, 10, 30, 0, 0, time.UTC)
	})
	service.WorkflowRunnerForTest(&stubWorkflowRunner{})

	runResult, err := service.Run(context.Background(), repoDir, RunRequest{
		TaskID:          "task-001",
		ConditionID:     "with-graph",
		Model:           "claude-sonnet",
		SessionTopology: "delegated-parallel",
		Seed:            42,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	result, err := service.PresentEvaluationInput(context.Background(), repoDir, PresentEvaluationRequest{
		RunID: runResult.RunID,
		Blind: true,
	})
	if err != nil {
		t.Fatalf("PresentEvaluationInput returned error: %v", err)
	}

	if !result.Blind {
		t.Fatal("blind = false, want true")
	}
	if result.Condition != nil {
		t.Fatalf("condition = %#v, want nil for blinded presentation", result.Condition)
	}
	if result.Task.Description != "implement retrieval ranking adjustment" {
		t.Fatalf("task.description = %q, want task description", result.Task.Description)
	}
	if got, want := result.RunRecordPath, ".graph/lab/runs/"+runResult.RunID+"/run.json"; got != want {
		t.Fatalf("run_record_path = %q, want %q", got, want)
	}

	writebackPayload := marshalAnyForTest(t, result.Artifacts.WritebackOutputs)
	outcomePayload := marshalAnyForTest(t, result.Artifacts.OutcomeSummary)
	for _, payload := range []string{writebackPayload, outcomePayload} {
		if containsAny(payload, `"condition_id"`, `"workflow_mode"`, `"graph_backed"`, `"with-graph"`, `"without-graph"`) {
			t.Fatalf("blinded payload leaked condition metadata: %s", payload)
		}
	}
	if !containsAny(writebackPayload, `"summary"`) {
		t.Fatalf("writeback payload = %s, want summary content", writebackPayload)
	}
	if !containsAny(outcomePayload, `"status":"completed"`) {
		t.Fatalf("outcome payload = %s, want status", outcomePayload)
	}
}

func TestServiceEvaluateRejectsMissingRunReference(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	_, err := service.Evaluate(context.Background(), repoDir, EvaluationRecord{
		RunID:     "run-999",
		Evaluator: "human:alice",
		Scores: &EvaluationScores{
			Success:                boolPtr(false),
			QualityScore:           float64Ptr(0.1),
			ResumabilityScore:      float64Ptr(0.2),
			HumanInterventionCount: intPtr(3),
		},
	})
	if err == nil {
		t.Fatal("expected Evaluate to return an error")
	}

	detail, ok := cmdsupport.ErrorDetailFromError(err)
	if !ok {
		t.Fatalf("error = %T, want structured command error", err)
	}
	assertViolationPresent(t, detail, "run_id", "run_id must reference an existing run record")
}

func TestServiceEvaluateSupportsRepeatedAndMultiEvaluatorScoring(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeSuiteWithSingleTask(t, repoDir)
	service.NowForTest(func() time.Time {
		return time.Date(2026, 4, 1, 10, 30, 0, 0, time.UTC)
	})
	service.WorkflowRunnerForTest(&stubWorkflowRunner{})

	runResult, err := service.Run(context.Background(), repoDir, RunRequest{
		TaskID:          "task-001",
		ConditionID:     "without-graph",
		Model:           "claude-sonnet",
		SessionTopology: "delegated-parallel",
		Seed:            42,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	service.NowForTest(func() time.Time {
		return time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC)
	})
	first, err := service.Evaluate(context.Background(), repoDir, EvaluationRecord{
		RunID:     runResult.RunID,
		Evaluator: "human:alice",
		Scores: &EvaluationScores{
			Success:                boolPtr(true),
			QualityScore:           float64Ptr(0.75),
			ResumabilityScore:      float64Ptr(0.7),
			HumanInterventionCount: intPtr(1),
		},
		Notes: "first pass",
	})
	if err != nil {
		t.Fatalf("first Evaluate returned error: %v", err)
	}

	service.NowForTest(func() time.Time {
		return time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	})
	second, err := service.Evaluate(context.Background(), repoDir, EvaluationRecord{
		RunID:     runResult.RunID,
		Evaluator: "automated:rubric-v2",
		Scores: &EvaluationScores{
			Success:                boolPtr(true),
			QualityScore:           float64Ptr(0.85),
			ResumabilityScore:      float64Ptr(0.9),
			HumanInterventionCount: intPtr(0),
		},
		Notes: "rescored with improved rubric",
	})
	if err != nil {
		t.Fatalf("second Evaluate returned error: %v", err)
	}

	if first.EvaluationCount != 1 {
		t.Fatalf("first evaluation_count = %d, want 1", first.EvaluationCount)
	}
	if second.EvaluationCount != 2 {
		t.Fatalf("second evaluation_count = %d, want 2", second.EvaluationCount)
	}

	artifact := readEvaluationArtifact(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "evaluations", runResult.RunID+".json"))
	if len(artifact.Records) != 2 {
		t.Fatalf("records = %#v, want two stored evaluations", artifact.Records)
	}
	if artifact.Records[0].Evaluator != "human:alice" || artifact.Records[1].Evaluator != "automated:rubric-v2" {
		t.Fatalf("evaluators = %#v, want preserved submission order", []string{artifact.Records[0].Evaluator, artifact.Records[1].Evaluator})
	}
	for _, record := range artifact.Records {
		if record.RunID != runResult.RunID {
			t.Fatalf("record.run_id = %q, want %q", record.RunID, runResult.RunID)
		}
	}
}

func readEvaluationArtifact(t *testing.T, path string) EvaluationArtifact {
	t.Helper()

	var artifact EvaluationArtifact
	if err := json.Unmarshal(readFile(t, path), &artifact); err != nil {
		t.Fatalf("json.Unmarshal evaluation artifact: %v", err)
	}
	return artifact
}

func marshalAnyForTest(t *testing.T, value any) string {
	t.Helper()

	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return string(payload)
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
