package lab

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/app/workflow"
	"github.com/guillaume-galp/cge/internal/infra/copilot"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func TestServiceRunExecutesGraphBackedConditionWithWorkflowPrimitives(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeSuiteWithSingleTask(t, repoDir)

	now := time.Date(2026, 4, 1, 10, 30, 0, 0, time.UTC)
	runner := &stubWorkflowRunner{
		startResult: workflow.StartResult{
			Kickoff: workflow.KickoffEnvelope{
				Task: workflow.KickoffTaskDetails{
					Family:     workflow.KickoffFamilyVerificationAudit,
					Subprofile: workflow.KickoffVerificationProfileWorkflow,
				},
				Advisory: workflow.KickoffAdvisoryState{
					EffectiveMode:   workflow.KickoffModeMinimal,
					ConfidenceScore: 0.72,
				},
			},
		},
	}
	service.NowForTest(func() time.Time { return now })
	service.WorkflowRunnerForTest(runner)

	result, err := service.Run(context.Background(), repoDir, RunRequest{
		TaskID:          "task-001",
		ConditionID:     "with-graph",
		Model:           "claude-sonnet",
		SessionTopology: "delegated-parallel",
		Seed:            42,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if result.RunID != "run-20260401t103000z" {
		t.Fatalf("run_id = %q, want run-20260401t103000z", result.RunID)
	}
	if result.Status != RunStatusCompleted {
		t.Fatalf("status = %q, want %q", result.Status, RunStatusCompleted)
	}
	if !result.Execution.GraphBacked || !result.Execution.KickoffUsed || !result.Execution.HandoffUsed {
		t.Fatalf("execution = %#v, want graph-backed kickoff and handoff", result.Execution)
	}
	if result.Execution.WorkflowMode != WorkflowModeGraphBacked {
		t.Fatalf("workflow_mode = %q, want %q", result.Execution.WorkflowMode, WorkflowModeGraphBacked)
	}
	if len(runner.startCalls) != 1 || runner.startCalls[0] != "implement retrieval ranking adjustment" {
		t.Fatalf("start calls = %#v, want one task description", runner.startCalls)
	}
	if len(runner.finishPayloads) != 1 {
		t.Fatalf("finish payload count = %d, want 1", len(runner.finishPayloads))
	}

	record := readRunRecord(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "runs", result.RunID, "run.json"))
	if record.ConditionID != "with-graph" {
		t.Fatalf("record condition_id = %q, want with-graph", record.ConditionID)
	}
	if record.Telemetry == nil || record.Telemetry.DelegatedSessions == nil || *record.Telemetry.DelegatedSessions != 2 {
		t.Fatalf("record telemetry = %#v, want delegated_sessions=2", record.Telemetry)
	}
	if got := record.Telemetry.MeasurementStatus; got != "unavailable" {
		t.Fatalf("measurement_status = %q, want unavailable when no outcome payload is provided", got)
	}
	if record.Telemetry.TotalTokens != nil {
		t.Fatalf("total_tokens = %#v, want nil without authoritative execution usage", record.Telemetry.TotalTokens)
	}
	if got := record.WorkflowStartResponseRef; got != "artifacts/workflow-start-response.json" {
		t.Fatalf("workflow_start_response_ref = %q, want workflow start response ref", got)
	}
	assertExists(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "runs", result.RunID, record.WorkflowStartResponseRef))
}

func TestServiceRunExecutesBaselineConditionWithoutWorkflowPrimitives(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeSuiteWithSingleTask(t, repoDir)

	runner := &stubWorkflowRunner{}
	service.NowForTest(func() time.Time {
		return time.Date(2026, 4, 1, 10, 30, 0, 0, time.UTC)
	})
	service.WorkflowRunnerForTest(runner)

	result, err := service.Run(context.Background(), repoDir, RunRequest{
		TaskID:          "task-001",
		ConditionID:     "without-graph",
		Model:           "claude-sonnet",
		SessionTopology: "delegated-parallel",
		Seed:            42,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if result.Execution.GraphBacked || result.Execution.KickoffUsed || result.Execution.HandoffUsed {
		t.Fatalf("execution = %#v, want baseline without workflow primitives", result.Execution)
	}
	if result.Execution.WorkflowMode != WorkflowModeBaseline {
		t.Fatalf("workflow_mode = %q, want %q", result.Execution.WorkflowMode, WorkflowModeBaseline)
	}
	if len(runner.startCalls) != 0 || len(runner.finishPayloads) != 0 {
		t.Fatalf("runner calls = start:%#v finish:%#v, want none", runner.startCalls, runner.finishPayloads)
	}

	record := readRunRecord(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "runs", result.RunID, "run.json"))
	if record.Telemetry == nil || record.Telemetry.DelegatedSessions == nil || *record.Telemetry.DelegatedSessions != 1 {
		t.Fatalf("record telemetry = %#v, want delegated_sessions=1", record.Telemetry)
	}
	if got := record.Telemetry.MeasurementStatus; got != "unavailable" {
		t.Fatalf("measurement_status = %q, want unavailable for baseline runs without an outcome payload", got)
	}
	if got := record.BaselinePromptMetadataRef; got != "artifacts/baseline-prompt-metadata.json" {
		t.Fatalf("baseline_prompt_metadata_ref = %q, want baseline prompt metadata ref", got)
	}
	assertExists(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "runs", result.RunID, record.BaselinePromptMetadataRef))
}

func TestServiceRunPersistsMeasuredTelemetryFromOutcomePayload(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeSuiteWithSingleTask(t, repoDir)

	runner := &stubWorkflowRunner{}
	service.NowForTest(func() time.Time {
		return time.Date(2026, 4, 1, 10, 30, 0, 0, time.UTC)
	})
	service.WorkflowRunnerForTest(runner)

	result, err := service.Run(context.Background(), repoDir, RunRequest{
		TaskID:          "task-001",
		ConditionID:     "with-graph",
		Model:           "claude-sonnet",
		SessionTopology: "delegated-parallel",
		Seed:            42,
		OutcomePayload: `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "finish-session",
    "timestamp": "2026-04-01T10:30:00Z",
    "revision": {
      "properties": {
        "execution_usage": {
          "measurement_status": "complete",
          "source": "workflow_finish_payload",
          "provider": "copilot-cli",
          "input_tokens": 800,
          "output_tokens": 200,
          "total_tokens": 1000
        }
      }
    }
  },
  "task": "implement retrieval ranking adjustment",
  "summary": "Measured execution completed.",
  "decisions": [],
  "changed_artifacts": [],
  "follow_up": []
}`,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	record := readRunRecord(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "runs", result.RunID, "run.json"))
	if got := record.Telemetry.MeasurementStatus; got != "complete" {
		t.Fatalf("measurement_status = %q, want complete", got)
	}
	if record.Telemetry.TotalTokens == nil || *record.Telemetry.TotalTokens != 1000 {
		t.Fatalf("total_tokens = %#v, want 1000", record.Telemetry.TotalTokens)
	}
	if got := record.Telemetry.Provider; got != "copilot-cli" {
		t.Fatalf("provider = %q, want copilot-cli", got)
	}
}

func TestServiceRunEnrichesOutcomePayloadFromCopilotSessionUsage(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeSuiteWithSingleTask(t, repoDir)

	runner := &stubWorkflowRunner{}
	service.NowForTest(func() time.Time {
		return time.Date(2026, 4, 1, 10, 30, 0, 0, time.UTC)
	})
	service.WorkflowRunnerForTest(runner)
	service.CopilotUsageCollectorForTest(stubCopilotUsageCollector{
		usage: copilot.SessionUsage{
			SessionID:    "copilot-session",
			Model:        "gpt-5.4",
			InputTokens:  810,
			OutputTokens: 190,
			TotalTokens:  1000,
		},
	})

	result, err := service.Run(context.Background(), repoDir, RunRequest{
		TaskID:           "task-001",
		ConditionID:      "with-graph",
		Model:            "gpt-5.4",
		SessionTopology:  "delegated-parallel",
		Seed:             42,
		CopilotSessionID: "copilot-session",
		OutcomePayload: `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "copilot-session",
    "timestamp": "2026-04-01T10:30:00Z"
  },
  "task": "implement retrieval ranking adjustment",
  "summary": "Live delegated execution completed.",
  "decisions": [],
  "changed_artifacts": [],
  "follow_up": []
}`,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	record := readRunRecord(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "runs", result.RunID, "run.json"))
	if got := record.Telemetry.MeasurementStatus; got != "complete" {
		t.Fatalf("measurement_status = %q, want complete", got)
	}
	if record.Telemetry.TotalTokens == nil || *record.Telemetry.TotalTokens != 1000 {
		t.Fatalf("total_tokens = %#v, want 1000", record.Telemetry.TotalTokens)
	}
	if got := record.Telemetry.Source; got != copilot.SessionUsageSource {
		t.Fatalf("source = %q, want %q", got, copilot.SessionUsageSource)
	}
	if len(runner.finishPayloads) != 1 {
		t.Fatalf("finish payload count = %d, want 1", len(runner.finishPayloads))
	}
	telemetry, err := workflow.ExtractExecutionTelemetryFromFinishPayload(runner.finishPayloads[0])
	if err != nil {
		t.Fatalf("ExtractExecutionTelemetryFromFinishPayload returned error: %v", err)
	}
	if telemetry == nil || telemetry.TotalTokens == nil || *telemetry.TotalTokens != 1000 {
		t.Fatalf("telemetry = %#v, want complete collected usage", telemetry)
	}
}

func TestServiceRunRejectsUnknownTaskOrConditionIdentifiers(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeSuiteWithSingleTask(t, repoDir)

	_, err := service.Run(context.Background(), repoDir, RunRequest{
		TaskID:          "task-missing",
		ConditionID:     "condition-missing",
		Model:           "claude-sonnet",
		SessionTopology: "delegated-parallel",
		Seed:            42,
	})
	if err == nil {
		t.Fatal("expected Run to return an error")
	}

	detail, ok := cmdsupport.ErrorDetailFromError(err)
	if !ok {
		t.Fatalf("error = %T, want structured command error", err)
	}
	if detail.Category != "validation_error" {
		t.Fatalf("error.category = %q, want validation_error", detail.Category)
	}
	if detail.Type != "validation_error" {
		t.Fatalf("error.type = %q, want validation_error", detail.Type)
	}
	if detail.Code != "lab_run_validation_failed" {
		t.Fatalf("error.code = %q, want lab_run_validation_failed", detail.Code)
	}
	assertViolationPresent(t, detail, "task_id", "task_id must reference an existing suite task")
	assertViolationPresent(t, detail, "condition_id", "condition_id must reference an existing condition")
}

func TestServiceRunRandomizesBatchOrderingByDefaultAndPersistsPlanBeforeExecution(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeSuiteWithTwoTasks(t, repoDir)

	now := time.Date(2026, 4, 1, 10, 30, 0, 0, time.UTC)
	expectedPlanPath := filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "runs", "batches", "batch-20260401t103000z", "plan.json")
	runner := &stubWorkflowRunner{
		onStart: func() {
			if _, err := os.Stat(expectedPlanPath); err != nil {
				t.Fatalf("expected plan artifact to exist before execution begins: %v", err)
			}
		},
	}
	service.NowForTest(func() time.Time { return now })
	service.WorkflowRunnerForTest(runner)

	result, err := service.Run(context.Background(), repoDir, RunRequest{
		TaskIDs:         []string{"task-001", "task-002"},
		ConditionIDs:    []string{"with-graph", "without-graph"},
		Model:           "claude-sonnet",
		SessionTopology: "delegated-parallel",
		Seed:            42,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if result.Batch == nil {
		t.Fatal("expected batch result")
	}
	if !result.Batch.Randomized {
		t.Fatalf("batch randomized = %v, want true", result.Batch.Randomized)
	}
	if got, want := result.Batch.PlanPath, ".graph/lab/runs/batches/batch-20260401t103000z/plan.json"; got != want {
		t.Fatalf("plan_path = %q, want %q", got, want)
	}

	gotOrder := batchAssignments(result.Batch.Runs)
	naturalOrder := []string{
		"task-001/with-graph",
		"task-001/without-graph",
		"task-002/with-graph",
		"task-002/without-graph",
	}
	if equalStringSlices(gotOrder, naturalOrder) {
		t.Fatalf("batch order = %#v, want shuffled order different from natural product order %#v", gotOrder, naturalOrder)
	}

	plan := readBatchPlan(t, expectedPlanPath)
	if !plan.Randomized {
		t.Fatalf("persisted plan randomized = %v, want true", plan.Randomized)
	}
	if got := batchPlanAssignments(plan.Entries); !equalStringSlices(got, gotOrder) {
		t.Fatalf("persisted plan order = %#v, want %#v", got, gotOrder)
	}
}

func TestServiceRunReproducesBatchOrderingWithSameSeed(t *testing.T) {
	t.Parallel()

	firstOrder := executeBatchRun(t, 42, nil)
	secondOrder := executeBatchRun(t, 42, nil)
	if !equalStringSlices(firstOrder, secondOrder) {
		t.Fatalf("first order = %#v, second order = %#v, want identical ordering", firstOrder, secondOrder)
	}
}

func TestServiceRunPreservesSequentialBatchOrderingWhenRandomizationIsDisabled(t *testing.T) {
	t.Parallel()

	randomize := false
	gotOrder := executeBatchRun(t, 42, &randomize)
	wantOrder := []string{
		"task-001/with-graph",
		"task-001/without-graph",
		"task-002/with-graph",
		"task-002/without-graph",
	}
	if !equalStringSlices(gotOrder, wantOrder) {
		t.Fatalf("batch order = %#v, want natural task × condition order %#v", gotOrder, wantOrder)
	}
}

type stubWorkflowRunner struct {
	startCalls     []string
	finishPayloads []string
	onStart        func()
	startResult    workflow.StartResult
}

type stubCopilotUsageCollector struct {
	usage copilot.SessionUsage
	err   error
}

func (s *stubWorkflowRunner) Start(_ context.Context, _ string, task string, _ int) (workflow.StartResult, error) {
	if s.onStart != nil {
		s.onStart()
	}
	s.startCalls = append(s.startCalls, task)
	return s.startResult, nil
}

func (s *stubWorkflowRunner) Finish(_ context.Context, _ string, input string) (workflow.FinishResult, error) {
	s.finishPayloads = append(s.finishPayloads, input)
	telemetry, err := workflow.ExtractExecutionTelemetryFromFinishPayload(input)
	if err != nil {
		return workflow.FinishResult{}, err
	}
	return workflow.FinishResult{ExecutionTelemetry: telemetry}, nil
}

func (s stubCopilotUsageCollector) CollectSessionUsage(context.Context, copilot.SessionUsageRequest) (copilot.SessionUsage, error) {
	return s.usage, s.err
}

func writeSuiteWithSingleTask(t *testing.T, repoDir string) {
	t.Helper()

	writeJSONFixture(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, repo.LabSuiteManifestName), SuiteManifest{
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
}

func writeSuiteWithTwoTasks(t *testing.T, repoDir string) {
	t.Helper()

	writeJSONFixture(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, repo.LabSuiteManifestName), SuiteManifest{
		SchemaVersion: SchemaVersion,
		SuiteID:       "delegated-workflow-evidence-v1",
		Tasks: []SuiteTask{
			{
				TaskID:                "task-001",
				Family:                "delegated-non-trivial-subtask",
				Description:           "implement retrieval ranking adjustment",
				AcceptanceCriteriaRef: "tasks/task-001/criteria.md",
			},
			{
				TaskID:                "task-002",
				Family:                "delegated-non-trivial-subtask",
				Description:           "tighten workflow handoff summary",
				AcceptanceCriteriaRef: "tasks/task-002/criteria.md",
			},
		},
	})
}

func executeBatchRun(t *testing.T, seed int64, randomize *bool) []string {
	t.Helper()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeSuiteWithTwoTasks(t, repoDir)
	service.NowForTest(func() time.Time {
		return time.Date(2026, 4, 1, 10, 30, 0, 0, time.UTC)
	})
	service.WorkflowRunnerForTest(&stubWorkflowRunner{})

	result, err := service.Run(context.Background(), repoDir, RunRequest{
		TaskIDs:         []string{"task-001", "task-002"},
		ConditionIDs:    []string{"with-graph", "without-graph"},
		Model:           "claude-sonnet",
		SessionTopology: "delegated-parallel",
		Seed:            seed,
		Randomize:       randomize,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Batch == nil {
		t.Fatal("expected batch result")
	}
	return batchAssignments(result.Batch.Runs)
}

func readBatchPlan(t *testing.T, path string) BatchPlanArtifact {
	t.Helper()

	var plan BatchPlanArtifact
	if err := json.Unmarshal(readFile(t, path), &plan); err != nil {
		t.Fatalf("json.Unmarshal batch plan: %v", err)
	}
	return plan
}

func batchAssignments(items []BatchRunItem) []string {
	assignments := make([]string, 0, len(items))
	for _, item := range items {
		assignments = append(assignments, item.TaskID+"/"+item.ConditionID)
	}
	return assignments
}

func batchPlanAssignments(items []BatchPlanEntry) []string {
	assignments := make([]string, 0, len(items))
	for _, item := range items {
		assignments = append(assignments, item.TaskID+"/"+item.ConditionID)
	}
	return assignments
}

func equalStringSlices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func readRunRecord(t *testing.T, path string) RunRecord {
	t.Helper()

	var record RunRecord
	if err := json.Unmarshal(readFile(t, path), &record); err != nil {
		t.Fatalf("json.Unmarshal run record: %v", err)
	}
	return record
}
