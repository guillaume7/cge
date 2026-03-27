package labcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/app/lab"
	"github.com/guillaume-galp/cge/internal/app/workflow"
	"github.com/guillaume-galp/cge/internal/infra/copilot"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func TestLabInitCommandBootstrapsWorkspaceAndReturnsMachineReadableSummary(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	cmd := newCommand(repoDir, lab.NewService(manager))
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeLabSuccessResponse(t, stdout.Bytes())
	if response.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want v1", response.SchemaVersion)
	}
	if response.Command != "lab.init" {
		t.Fatalf("command = %q, want lab.init", response.Command)
	}
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if !response.Result.Workspace.Initialized || response.Result.Workspace.AlreadyInitialized {
		t.Fatalf("workspace = %#v, want initialized", response.Result.Workspace)
	}
	if response.Result.Installed.Count != 6 {
		t.Fatalf("installed count = %d, want 6", response.Result.Installed.Count)
	}
	if _, err := os.Stat(filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, repo.LabSuiteManifestName)); err != nil {
		t.Fatalf("suite manifest stat: %v", err)
	}
}

func TestLabInitCommandReturnsStructuredErrorWhenRepositoryRootCannotBeDetermined(t *testing.T) {
	t.Parallel()

	startDir := t.TempDir()
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	cmd := newCommand(startDir, lab.NewService(manager))
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init"})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected ExecuteContext to return an error")
	}
	if !cmdsupport.IsSilentError(err) {
		t.Fatalf("error = %T, want silent error", err)
	}

	response := decodeLabErrorResponse(t, stdout.Bytes())
	if response.Command != "lab.init" {
		t.Fatalf("command = %q, want lab.init", response.Command)
	}
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Error == nil {
		t.Fatal("expected structured error payload")
	}
	if response.Error.Code != "repository_root_not_found" {
		t.Fatalf("error.code = %q, want repository_root_not_found", response.Error.Code)
	}
	if response.Error.Type != "workspace_error" {
		t.Fatalf("error.type = %q, want workspace_error", response.Error.Type)
	}
}

func TestHandleInitErrorPreservesCommandErrorDetails(t *testing.T) {
	t.Parallel()

	detail := cmdsupport.ErrorDetail{
		Category: "validation_error",
		Type:     "lab_error",
		Code:     "bad_lab_state",
		Message:  "lab state invalid",
	}
	buffer := &bytes.Buffer{}

	err := handleInitError(buffer, "", cmdsupport.NewCommandError(detail, errors.New("boom")))
	if err == nil || !cmdsupport.IsSilentError(err) {
		t.Fatalf("handleInitError error = %v, want silent error", err)
	}

	response := decodeLabErrorResponse(t, buffer.Bytes())
	if response.Error == nil || response.Error.Code != "bad_lab_state" {
		t.Fatalf("error = %#v, want preserved command error detail", response.Error)
	}
}

type labSuccessResponse struct {
	SchemaVersion string         `json:"schema_version"`
	Command       string         `json:"command"`
	Status        string         `json:"status"`
	Result        lab.InitResult `json:"result"`
}

type labRunSuccessResponse struct {
	SchemaVersion string        `json:"schema_version"`
	Command       string        `json:"command"`
	Status        string        `json:"status"`
	Result        lab.RunResult `json:"result"`
}

type labEvaluateSuccessResponse struct {
	SchemaVersion string             `json:"schema_version"`
	Command       string             `json:"command"`
	Status        string             `json:"status"`
	Result        lab.EvaluateResult `json:"result"`
}

type labEvaluatePresentSuccessResponse struct {
	SchemaVersion string                      `json:"schema_version"`
	Command       string                      `json:"command"`
	Status        string                      `json:"status"`
	Result        lab.PresentEvaluationResult `json:"result"`
}

type labReportSuccessResponse struct {
	SchemaVersion string           `json:"schema_version"`
	Command       string           `json:"command"`
	Status        string           `json:"status"`
	Result        lab.ReportResult `json:"result"`
}

type labErrorResponse struct {
	SchemaVersion string                  `json:"schema_version"`
	Command       string                  `json:"command"`
	Status        string                  `json:"status"`
	Error         *cmdsupport.ErrorDetail `json:"error"`
}

func decodeLabSuccessResponse(t *testing.T, payload []byte) labSuccessResponse {
	t.Helper()

	var response labSuccessResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal success response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeLabErrorResponse(t *testing.T, payload []byte) labErrorResponse {
	t.Helper()

	var response labErrorResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal error response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeLabRunSuccessResponse(t *testing.T, payload []byte) labRunSuccessResponse {
	t.Helper()

	var response labRunSuccessResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal run success response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeLabEvaluateSuccessResponse(t *testing.T, payload []byte) labEvaluateSuccessResponse {
	t.Helper()

	var response labEvaluateSuccessResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal evaluate success response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeLabEvaluatePresentSuccessResponse(t *testing.T, payload []byte) labEvaluatePresentSuccessResponse {
	t.Helper()

	var response labEvaluatePresentSuccessResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal evaluate present success response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeLabReportSuccessResponse(t *testing.T, payload []byte) labReportSuccessResponse {
	t.Helper()

	var response labReportSuccessResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal report success response: %v\npayload: %s", err, payload)
	}
	return response
}

func initGitRepository(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	runGitCommand(t, repoDir, "init")
	runGitCommand(t, repoDir, "config", "user.email", "test@example.com")
	runGitCommand(t, repoDir, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# repo\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile README.md: %v", err)
	}
	return repoDir
}

func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}

func TestLabRunCommandReturnsMachineReadableSummary(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	service := lab.NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeLabSuiteFixture(t, repoDir)
	service.NowForTest(func() time.Time {
		return time.Date(2026, 4, 1, 10, 30, 0, 0, time.UTC)
	})
	service.WorkflowRunnerForTest(&labCommandStubWorkflowRunner{})

	cmd := newCommand(repoDir, service)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"run",
		"--task", "task-001",
		"--condition", "with-graph",
		"--model", "claude-sonnet",
		"--topology", "delegated-parallel",
		"--seed", "42",
	})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeLabRunSuccessResponse(t, stdout.Bytes())
	if response.Command != "lab.run" {
		t.Fatalf("command = %q, want lab.run", response.Command)
	}
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if response.Result.RunID == "" {
		t.Fatal("expected non-empty run_id")
	}
	if response.Result.Parameters.TaskID != "task-001" || response.Result.Parameters.ConditionID != "with-graph" {
		t.Fatalf("parameters = %#v, want declared task and condition", response.Result.Parameters)
	}
	if !response.Result.Execution.KickoffUsed || !response.Result.Execution.HandoffUsed {
		t.Fatalf("execution = %#v, want graph-backed workflow primitive usage", response.Result.Execution)
	}
}

func TestLabRunCommandSupportsCopilotSessionUsageFlags(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	service := lab.NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeLabSuiteFixture(t, repoDir)
	service.NowForTest(func() time.Time {
		return time.Date(2026, 4, 1, 10, 30, 0, 0, time.UTC)
	})
	service.WorkflowRunnerForTest(&labCommandStubWorkflowRunner{})
	service.CopilotUsageCollectorForTest(labCommandStubCopilotUsageCollector{
		usage: copilot.SessionUsage{
			SessionID:    "copilot-session",
			Model:        "gpt-5.4",
			InputTokens:  900,
			OutputTokens: 100,
			TotalTokens:  1000,
		},
	})

	cmd := newCommand(repoDir, service)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"run",
		"--task", "task-001",
		"--condition", "with-graph",
		"--model", "gpt-5.4",
		"--topology", "delegated-parallel",
		"--seed", "42",
		"--copilot-session-id", "copilot-session",
		"--outcome-payload", `{"schema_version":"v1","metadata":{"agent_id":"developer","session_id":"copilot-session","timestamp":"2026-04-01T10:30:00Z"},"task":"implement retrieval ranking adjustment","summary":"done","decisions":[],"changed_artifacts":[],"follow_up":[]}`,
	})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeLabRunSuccessResponse(t, stdout.Bytes())
	if response.Result.RunID == "" {
		t.Fatal("expected non-empty run_id")
	}
}

func TestLabRunCommandReturnsStructuredValidationErrorForUnknownIdentifiers(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	service := lab.NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeLabSuiteFixture(t, repoDir)

	cmd := newCommand(repoDir, service)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"run",
		"--task", "task-missing",
		"--condition", "condition-missing",
		"--model", "claude-sonnet",
		"--topology", "delegated-parallel",
		"--seed", "42",
	})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected ExecuteContext to return an error")
	}
	if !cmdsupport.IsSilentError(err) {
		t.Fatalf("error = %T, want silent error", err)
	}

	response := decodeLabErrorResponse(t, stdout.Bytes())
	if response.Command != "lab.run" {
		t.Fatalf("command = %q, want lab.run", response.Command)
	}
	if response.Error == nil {
		t.Fatal("expected structured error payload")
	}
	if response.Error.Code != "lab_run_validation_failed" {
		t.Fatalf("error.code = %q, want lab_run_validation_failed", response.Error.Code)
	}
	assertLabRunViolationPresent(t, *response.Error, "task_id", "task_id must reference an existing suite task")
	assertLabRunViolationPresent(t, *response.Error, "condition_id", "condition_id must reference an existing condition")
}

func TestLabRunCommandSupportsBatchPlanningFlagsAndSequentialOrdering(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	service := lab.NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeLabBatchSuiteFixture(t, repoDir)
	service.NowForTest(func() time.Time {
		return time.Date(2026, 4, 1, 10, 30, 0, 0, time.UTC)
	})
	service.WorkflowRunnerForTest(&labCommandStubWorkflowRunner{})

	cmd := newCommand(repoDir, service)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"run",
		"--task", "task-001",
		"--task", "task-002",
		"--condition", "with-graph",
		"--condition", "without-graph",
		"--model", "claude-sonnet",
		"--topology", "delegated-parallel",
		"--seed", "42",
		"--no-randomize",
	})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeLabRunSuccessResponse(t, stdout.Bytes())
	if response.Result.Batch == nil {
		t.Fatal("expected batch execution summary")
	}
	if response.Result.Batch.Randomized {
		t.Fatalf("batch randomized = %v, want false", response.Result.Batch.Randomized)
	}
	if response.Result.Parameters.Randomize == nil || *response.Result.Parameters.Randomize {
		t.Fatalf("parameters.randomize = %#v, want false", response.Result.Parameters.Randomize)
	}

	got := make([]string, 0, len(response.Result.Batch.Runs))
	for _, item := range response.Result.Batch.Runs {
		got = append(got, item.TaskID+"/"+item.ConditionID)
	}
	want := []string{
		"task-001/with-graph",
		"task-001/without-graph",
		"task-002/with-graph",
		"task-002/without-graph",
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("batch order = %#v, want %#v", got, want)
		}
	}
}

func TestLabEvaluateCommandReturnsMachineReadableSummaryAndStoresSeparateArtifact(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	service := lab.NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeLabSuiteFixture(t, repoDir)
	service.NowForTest(func() time.Time {
		return time.Date(2026, 4, 1, 10, 30, 0, 0, time.UTC)
	})
	service.WorkflowRunnerForTest(&labCommandStubWorkflowRunner{})

	runResult, err := service.Run(context.Background(), repoDir, lab.RunRequest{
		TaskID:          "task-001",
		ConditionID:     "with-graph",
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

	cmd := newCommand(repoDir, service)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"evaluate",
		"--run", runResult.RunID,
		"--evaluator", "human:alice",
		"--success=true",
		"--quality", "0.9",
		"--resumability", "0.8",
		"--human-interventions", "1",
		"--notes", "strong implementation",
	})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeLabEvaluateSuccessResponse(t, stdout.Bytes())
	if response.Command != "lab.evaluate" {
		t.Fatalf("command = %q, want lab.evaluate", response.Command)
	}
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if got, want := response.Result.ArtifactPath, ".graph/lab/evaluations/"+runResult.RunID+".json"; got != want {
		t.Fatalf("artifact_path = %q, want %q", got, want)
	}
	if response.Result.EvaluationCount != 1 {
		t.Fatalf("evaluation_count = %d, want 1", response.Result.EvaluationCount)
	}
}

func TestLabEvaluatePresentCommandReturnsBlindedPresentation(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	service := lab.NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeLabSuiteFixture(t, repoDir)
	service.NowForTest(func() time.Time {
		return time.Date(2026, 4, 1, 10, 30, 0, 0, time.UTC)
	})
	service.WorkflowRunnerForTest(&labCommandStubWorkflowRunner{})

	runResult, err := service.Run(context.Background(), repoDir, lab.RunRequest{
		TaskID:          "task-001",
		ConditionID:     "with-graph",
		Model:           "claude-sonnet",
		SessionTopology: "delegated-parallel",
		Seed:            42,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	cmd := newCommand(repoDir, service)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"evaluate",
		"present",
		"--run", runResult.RunID,
		"--blind",
	})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeLabEvaluatePresentSuccessResponse(t, stdout.Bytes())
	if response.Command != "lab.evaluate.present" {
		t.Fatalf("command = %q, want lab.evaluate.present", response.Command)
	}
	if !response.Result.Blind {
		t.Fatal("blind = false, want true")
	}
	if response.Result.Condition != nil {
		t.Fatalf("condition = %#v, want nil", response.Result.Condition)
	}

	payload, err := json.Marshal(response.Result.Artifacts)
	if err != nil {
		t.Fatalf("json.Marshal artifacts: %v", err)
	}
	if bytes.Contains(payload, []byte("condition_id")) || bytes.Contains(payload, []byte("workflow_mode")) || bytes.Contains(payload, []byte("graph_backed")) {
		t.Fatalf("artifacts leaked condition metadata: %s", payload)
	}
}

func TestLabReportCommandReturnsMachineReadableSummaryForSelectedRuns(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	service := lab.NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeLabReportSuiteFixture(t, repoDir)
	writeLabReportRunFixture(t, repoDir, "run-graph", "task-001", "with-graph", "model-a", "delegated-parallel", 1, 100, 10, true, 0.9, 0.8)
	writeLabReportRunFixture(t, repoDir, "run-base", "task-001", "without-graph", "model-a", "delegated-parallel", 1, 150, 20, true, 0.7, 0.6)
	service.NowForTest(func() time.Time {
		return time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	})

	cmd := newCommand(repoDir, service)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{
		"report",
		"--run", "run-graph",
		"--run", "run-base",
	})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeLabReportSuccessResponse(t, stdout.Bytes())
	if response.Command != "lab.report" {
		t.Fatalf("command = %q, want lab.report", response.Command)
	}
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if response.Result.Report.Selection.Mode != "selected_runs" {
		t.Fatalf("selection.mode = %q, want selected_runs", response.Result.Report.Selection.Mode)
	}
	if response.Result.Report.RunsIncluded != 2 {
		t.Fatalf("runs_included = %d, want 2", response.Result.Report.RunsIncluded)
	}
	if len(response.Result.Report.PairedComparisons) != 1 {
		t.Fatalf("paired_comparisons = %#v, want one focused comparison", response.Result.Report.PairedComparisons)
	}
	if got, want := response.Result.ArtifactPath, ".graph/lab/reports/report-20260401t120000z.json"; got != want {
		t.Fatalf("artifact_path = %q, want %q", got, want)
	}
}

func writeLabReportSuiteFixture(t *testing.T, repoDir string) {
	t.Helper()

	payload, err := json.MarshalIndent(lab.SuiteManifest{
		SchemaVersion: lab.SchemaVersion,
		SuiteID:       "delegated-workflow-evidence-v1",
		Tasks: []lab.SuiteTask{
			{
				TaskID:                "task-001",
				Family:                "delegated-non-trivial-subtask",
				Description:           "implement retrieval ranking adjustment",
				AcceptanceCriteriaRef: "tasks/task-001/criteria.md",
			},
		},
	}, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent suite fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, repo.LabSuiteManifestName), append(payload, '\n'), 0o644); err != nil {
		t.Fatalf("os.WriteFile suite fixture: %v", err)
	}
}

func writeLabReportRunFixture(t *testing.T, repoDir, runID, taskID, conditionID, model, topology string, seed int64, tokens, wallClock int, success bool, quality, resumability float64) {
	t.Helper()

	record := lab.RunRecord{
		SchemaVersion:   lab.SchemaVersion,
		RunID:           runID,
		TaskID:          taskID,
		ConditionID:     conditionID,
		Model:           model,
		SessionTopology: topology,
		Seed:            &seed,
		PromptVariant:   "default",
		StartedAt:       "2026-04-01T10:30:00Z",
		FinishedAt:      "2026-04-01T10:45:00Z",
		Telemetry: &lab.RunTelemetry{
			MeasurementStatus: "complete",
			Source:            "workflow_finish_payload",
			Provider:          "copilot-cli",
			TotalTokens:       intPointerForCommandTest(tokens),
			InputTokens:       intPointerForCommandTest(tokens / 2),
			OutputTokens:      intPointerForCommandTest(tokens / 2),
			WallClockSeconds:  intPointerForCommandTest(wallClock),
			RetryCount:        intPointerForCommandTest(0),
			DelegatedSessions: intPointerForCommandTest(1),
		},
		KickoffInputsRef:    "artifacts/task-input.json",
		SessionStructureRef: "artifacts/sessions/",
		WritebackOutputsRef: "artifacts/task-output.json",
		OutcomeArtifactsRef: "artifacts/output/" + conditionID + "/",
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent run fixture: %v", err)
	}
	runPath := filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "runs", runID, "run.json")
	if err := os.MkdirAll(filepath.Dir(runPath), 0o755); err != nil {
		t.Fatalf("os.MkdirAll run dir: %v", err)
	}
	if err := os.WriteFile(runPath, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("os.WriteFile run fixture: %v", err)
	}

	evaluation := lab.EvaluationArtifact{
		SchemaVersion: lab.SchemaVersion,
		RunID:         runID,
		Records: []lab.EvaluationRecord{
			{
				SchemaVersion: lab.SchemaVersion,
				RunID:         runID,
				Evaluator:     "human:alice",
				EvaluatedAt:   "2026-04-01T11:00:00Z",
				Scores: &lab.EvaluationScores{
					Success:                boolPointerForCommandTest(success),
					QualityScore:           &quality,
					ResumabilityScore:      &resumability,
					HumanInterventionCount: intPointerForCommandTest(0),
				},
			},
		},
	}
	evaluationData, err := json.MarshalIndent(evaluation, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent evaluation fixture: %v", err)
	}
	evaluationPath := filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "evaluations", runID+".json")
	if err := os.MkdirAll(filepath.Dir(evaluationPath), 0o755); err != nil {
		t.Fatalf("os.MkdirAll evaluation dir: %v", err)
	}
	if err := os.WriteFile(evaluationPath, append(evaluationData, '\n'), 0o644); err != nil {
		t.Fatalf("os.WriteFile evaluation fixture: %v", err)
	}
}

func intPointerForCommandTest(value int) *int {
	return &value
}

func boolPointerForCommandTest(value bool) *bool {
	return &value
}

type labCommandStubWorkflowRunner struct{}

type labCommandStubCopilotUsageCollector struct {
	usage copilot.SessionUsage
	err   error
}

func (labCommandStubWorkflowRunner) Start(context.Context, string, string, int) (workflow.StartResult, error) {
	return workflow.StartResult{}, nil
}

func (labCommandStubWorkflowRunner) Finish(context.Context, string, string) (workflow.FinishResult, error) {
	return workflow.FinishResult{}, nil
}

func (s labCommandStubCopilotUsageCollector) CollectSessionUsage(context.Context, copilot.SessionUsageRequest) (copilot.SessionUsage, error) {
	return s.usage, s.err
}

func writeLabSuiteFixture(t *testing.T, repoDir string) {
	t.Helper()

	payload, err := json.MarshalIndent(lab.SuiteManifest{
		SchemaVersion: lab.SchemaVersion,
		SuiteID:       "delegated-workflow-evidence-v1",
		Tasks: []lab.SuiteTask{
			{
				TaskID:                "task-001",
				Family:                "delegated-non-trivial-subtask",
				Description:           "implement retrieval ranking adjustment",
				AcceptanceCriteriaRef: "tasks/task-001/criteria.md",
			},
		},
	}, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent suite fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, repo.LabSuiteManifestName), append(payload, '\n'), 0o644); err != nil {
		t.Fatalf("os.WriteFile suite fixture: %v", err)
	}
}

func writeLabBatchSuiteFixture(t *testing.T, repoDir string) {
	t.Helper()

	payload, err := json.MarshalIndent(lab.SuiteManifest{
		SchemaVersion: lab.SchemaVersion,
		SuiteID:       "delegated-workflow-evidence-v1",
		Tasks: []lab.SuiteTask{
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
	}, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent suite fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, repo.LabSuiteManifestName), append(payload, '\n'), 0o644); err != nil {
		t.Fatalf("os.WriteFile suite fixture: %v", err)
	}
}

func assertLabRunViolationPresent(t *testing.T, detail cmdsupport.ErrorDetail, field, message string) {
	t.Helper()

	raw, ok := detail.Details["violations"]
	if !ok {
		t.Fatalf("error.details = %#v, missing violations", detail.Details)
	}

	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("violations = %#v, want []any", raw)
	}

	for _, item := range items {
		violation, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if violation["field"] == field && violation["message"] == message {
			return
		}
	}

	t.Fatalf("violations = %#v, want field=%q message=%q", raw, field, message)
}
