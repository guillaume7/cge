package lab

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func TestServiceInitBootstrapsWorkspaceAndLabScaffolding(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)

	result, err := service.Init(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	if !result.Workspace.Initialized || result.Workspace.AlreadyInitialized {
		t.Fatalf("workspace = %#v, want initialized and not already initialized", result.Workspace)
	}
	if result.Lab.SchemaVersion != SchemaVersion {
		t.Fatalf("lab schema_version = %q, want %q", result.Lab.SchemaVersion, SchemaVersion)
	}
	if result.Installed.Count != 6 {
		t.Fatalf("installed count = %d, want 6", result.Installed.Count)
	}
	if result.Refreshed.Count != 0 {
		t.Fatalf("refreshed count = %d, want 0", result.Refreshed.Count)
	}
	if result.Preserved.Count != 0 {
		t.Fatalf("preserved count = %d, want 0", result.Preserved.Count)
	}

	assertExists(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.ConfigFileName))
	assertExists(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName))
	assertExists(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, repo.LabSuiteManifestName))
	assertExists(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, repo.LabConditionsManifestName))
	assertExists(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "runs"))
	assertExists(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "evaluations"))
	assertExists(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "reports"))

	suite := readSuiteManifest(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, repo.LabSuiteManifestName))
	if suite.SchemaVersion != SchemaVersion {
		t.Fatalf("suite schema_version = %q, want %q", suite.SchemaVersion, SchemaVersion)
	}
	if suite.SuiteID != "delegated-workflow-evidence-v1" {
		t.Fatalf("suite_id = %q, want delegated-workflow-evidence-v1", suite.SuiteID)
	}
	if len(suite.Tasks) != 0 {
		t.Fatalf("suite tasks = %#v, want empty scaffold", suite.Tasks)
	}

	conditions := readConditionsManifest(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, repo.LabConditionsManifestName))
	if len(conditions.Conditions) != 5 {
		t.Fatalf("conditions = %#v, want 5 default conditions (with-graph, without-graph, with-harness, without-harness, graph-only)", conditions.Conditions)
	}
	conditionIDs := make([]string, 0, len(conditions.Conditions))
	for _, c := range conditions.Conditions {
		conditionIDs = append(conditionIDs, c.ConditionID)
	}
	for _, wantID := range []string{"with-graph", "without-graph", "with-harness", "without-harness", "graph-only"} {
		found := false
		for _, id := range conditionIDs {
			if id == wantID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("default conditions missing %q; got %v", wantID, conditionIDs)
		}
	}
}

func TestServiceInitRefreshesScaffoldingAndPreservesExistingArtifacts(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)

	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("initial Init returned error: %v", err)
	}

	labDir := filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName)
	suitePath := filepath.Join(labDir, repo.LabSuiteManifestName)
	if err := os.WriteFile(suitePath, []byte("{\"schema_version\":\"stale\"}\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile suite: %v", err)
	}

	runPath := filepath.Join(labDir, "runs", "run-001.json")
	evaluationPath := filepath.Join(labDir, "evaluations", "run-001.json")
	reportPath := filepath.Join(labDir, "reports", "report-001.json")
	writeFixture(t, runPath, `{"run_id":"run-001"}`+"\n")
	writeFixture(t, evaluationPath, `{"run_id":"run-001","success":true}`+"\n")
	writeFixture(t, reportPath, `{"report_id":"report-001"}`+"\n")

	result, err := service.Init(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("refresh Init returned error: %v", err)
	}

	if result.Workspace.Initialized || !result.Workspace.AlreadyInitialized {
		t.Fatalf("workspace = %#v, want already initialized", result.Workspace)
	}
	if result.Installed.Count != 0 {
		t.Fatalf("installed count = %d, want 0", result.Installed.Count)
	}
	if result.Refreshed.Count != 1 {
		t.Fatalf("refreshed count = %d, want 1", result.Refreshed.Count)
	}
	if got := result.Refreshed.Items[0].Path; got != ".graph/lab/suite.json" {
		t.Fatalf("refreshed path = %q, want .graph/lab/suite.json", got)
	}
	if got := result.Refreshed.Items[0].Reason; got != "existing manifest was invalid or stale" {
		t.Fatalf("refreshed reason = %q, want invalid/stale manifest note", got)
	}
	if result.Preserved.Count != 3 {
		t.Fatalf("preserved count = %d, want 3", result.Preserved.Count)
	}

	preservedPaths := map[string]bool{}
	for _, item := range result.Preserved.Items {
		preservedPaths[item.Path] = true
		if item.Status != "preserved" {
			t.Fatalf("preserved item status = %q, want preserved", item.Status)
		}
	}
	for _, path := range []string{
		".graph/lab/runs/run-001.json",
		".graph/lab/evaluations/run-001.json",
		".graph/lab/reports/report-001.json",
	} {
		if !preservedPaths[path] {
			t.Fatalf("preserved paths = %#v, missing %q", preservedPaths, path)
		}
	}

	if got := string(readFile(t, runPath)); got != "{\"run_id\":\"run-001\"}\n" {
		t.Fatalf("run artifact changed = %q, want preserved content", got)
	}
	if got := string(readFile(t, evaluationPath)); got != "{\"run_id\":\"run-001\",\"success\":true}\n" {
		t.Fatalf("evaluation artifact changed = %q, want preserved content", got)
	}
	if got := string(readFile(t, reportPath)); got != "{\"report_id\":\"report-001\"}\n" {
		t.Fatalf("report artifact changed = %q, want preserved content", got)
	}

	suite := readSuiteManifest(t, suitePath)
	if suite.SchemaVersion != SchemaVersion {
		t.Fatalf("suite schema_version after refresh = %q, want %q", suite.SchemaVersion, SchemaVersion)
	}
}

func TestServiceInitPreservesValidCustomizedManifests(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)

	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("initial Init returned error: %v", err)
	}

	labDir := filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName)
	suitePath := filepath.Join(labDir, repo.LabSuiteManifestName)
	conditionsPath := filepath.Join(labDir, repo.LabConditionsManifestName)

	customSuite := SuiteManifest{
		SchemaVersion: SchemaVersion,
		SuiteID:       "repo-dogfooding-v1",
		Tasks: []SuiteTask{
			{
				TaskID:                "repo-task-001",
				Family:                "delegated-non-trivial-subtask",
				Description:           "verify repo-local delegated workflow kickoff and handoff",
				AcceptanceCriteriaRef: "docs/themes/TH3-graph-backed-delegated-workflow/epics/E3-benchmark-and-repo-dogfooding/stories/US3-repo-workflow-snippets-and-hook-verification.md",
			},
			{
				TaskID:                "repo-task-002",
				Family:                "delegated-non-trivial-subtask",
				Description:           "derive machine-readable lab reports with explicit limitations",
				AcceptanceCriteriaRef: "docs/themes/TH4-experimental-evidence-lab/epics/E3-evaluation-reporting-and-dogfooding/stories/US2-lab-report-command.md",
			},
		},
	}
	customConditions := ConditionsManifest{
		SchemaVersion: SchemaVersion,
		Conditions: []Condition{
			{
				ConditionID:  "with-graph",
				WorkflowMode: WorkflowModeGraphBacked,
				Description:  "graph-backed delegated workflow using graph workflow start and finish",
			},
			{
				ConditionID:  "without-graph",
				WorkflowMode: WorkflowModeBaseline,
				Description:  "baseline delegated workflow without graph workflow start/finish",
			},
		},
		BlockingFactors: []string{
			BlockingFactorTaskFamily,
			BlockingFactorModel,
			BlockingFactorSessionTopology,
		},
	}
	writeJSONFixture(t, suitePath, customSuite)
	writeJSONFixture(t, conditionsPath, customConditions)

	result, err := service.Init(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("refresh Init returned error: %v", err)
	}

	if result.Refreshed.Count != 0 {
		t.Fatalf("refreshed count = %d, want 0 for valid customized manifests", result.Refreshed.Count)
	}
	if result.Skipped.Count < 2 {
		t.Fatalf("skipped count = %d, want at least manifest entries", result.Skipped.Count)
	}

	skippedReasons := map[string]string{}
	for _, item := range result.Skipped.Items {
		skippedReasons[item.Path] = item.Reason
	}
	if got := skippedReasons[".graph/lab/suite.json"]; got != "existing valid customized manifest preserved" {
		t.Fatalf("suite skip reason = %q, want customized manifest preserved", got)
	}
	if got := skippedReasons[".graph/lab/conditions.json"]; got != "existing valid customized manifest preserved" {
		t.Fatalf("conditions skip reason = %q, want customized manifest preserved", got)
	}

	suite := readSuiteManifest(t, suitePath)
	if suite.SuiteID != customSuite.SuiteID || len(suite.Tasks) != len(customSuite.Tasks) {
		t.Fatalf("suite after refresh = %#v, want preserved custom suite %#v", suite, customSuite)
	}
	conditions := readConditionsManifest(t, conditionsPath)
	if conditions.Conditions[0].Description != customConditions.Conditions[0].Description {
		t.Fatalf("conditions after refresh = %#v, want preserved custom conditions %#v", conditions, customConditions)
	}
}

func TestServiceLoadSuiteManifestLoadsValidManifestAndSupportsLookupByTaskID(t *testing.T) {
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
			{
				TaskID:                "task-002",
				Family:                "delegated-non-trivial-subtask",
				Description:           "tighten workflow handoff summary",
				AcceptanceCriteriaRef: "tasks/task-002/criteria.md",
			},
		},
	})

	manifest, err := service.LoadSuiteManifest(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("LoadSuiteManifest returned error: %v", err)
	}

	task, ok := manifest.TaskByID("task-001")
	if !ok {
		t.Fatalf("TaskByID(task-001) = not found, want found")
	}
	if task.Family != "delegated-non-trivial-subtask" {
		t.Fatalf("task family = %q, want delegated-non-trivial-subtask", task.Family)
	}
	if task.AcceptanceCriteriaRef != "tasks/task-001/criteria.md" {
		t.Fatalf("acceptance_criteria_ref = %q, want tasks/task-001/criteria.md", task.AcceptanceCriteriaRef)
	}
	if _, ok := manifest.TaskByID("missing"); ok {
		t.Fatal("TaskByID(missing) = found, want not found")
	}
}

func TestServiceLoadConditionsManifestLoadsValidManifestAndSupportsLookupByConditionID(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)

	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	conditionsPath := filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, repo.LabConditionsManifestName)
	writeJSONFixture(t, conditionsPath, ConditionsManifest{
		SchemaVersion: SchemaVersion,
		Conditions: []Condition{
			{
				ConditionID:  "with-graph",
				WorkflowMode: WorkflowModeGraphBacked,
				Description:  "full graph-backed kickoff and handoff",
			},
			{
				ConditionID:  "without-graph",
				WorkflowMode: WorkflowModeBaseline,
				Description:  "no graph context; standard delegation only",
			},
		},
		BlockingFactors: []string{
			BlockingFactorTaskFamily,
			BlockingFactorModel,
			BlockingFactorSessionTopology,
		},
	})

	manifest, err := service.LoadConditionsManifest(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("LoadConditionsManifest returned error: %v", err)
	}

	condition, ok := manifest.ConditionByID("with-graph")
	if !ok {
		t.Fatalf("ConditionByID(with-graph) = not found, want found")
	}
	if condition.WorkflowMode != WorkflowModeGraphBacked {
		t.Fatalf("workflow_mode = %q, want %q", condition.WorkflowMode, WorkflowModeGraphBacked)
	}
	if len(manifest.BlockingFactors) != 3 {
		t.Fatalf("blocking factors = %#v, want three declared factors", manifest.BlockingFactors)
	}
	if _, ok := manifest.ConditionByID("missing"); ok {
		t.Fatal("ConditionByID(missing) = found, want not found")
	}
}

func TestServiceLoadSuiteManifestReturnsStructuredValidationErrorForMissingFields(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)

	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	suitePath := filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, repo.LabSuiteManifestName)
	writeFixture(t, suitePath, `{
  "suite_id": "delegated-workflow-evidence-v1",
  "tasks": [
    {
      "family": "delegated-non-trivial-subtask",
      "description": "implement retrieval ranking adjustment",
      "acceptance_criteria_ref": "tasks/task-001/criteria.md"
    }
  ]
}
`)

	_, err := service.LoadSuiteManifest(context.Background(), repoDir)
	if err == nil {
		t.Fatal("expected LoadSuiteManifest to return an error")
	}

	detail, ok := cmdsupport.ErrorDetailFromError(err)
	if !ok {
		t.Fatalf("error = %T, want structured command error", err)
	}
	if detail.Category != "validation_error" {
		t.Fatalf("error.category = %q, want validation_error", detail.Category)
	}
	if detail.Type != "manifest_error" {
		t.Fatalf("error.type = %q, want manifest_error", detail.Type)
	}
	if detail.Code != "manifest_validation_failed" {
		t.Fatalf("error.code = %q, want manifest_validation_failed", detail.Code)
	}
	assertViolationPresent(t, detail, "schema_version", "field is required")
	assertViolationPresent(t, detail, "tasks[0].task_id", "field is required")
}

func TestServiceLoadConditionsManifestReturnsStructuredValidationErrorForUnknownWorkflowMode(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)

	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	conditionsPath := filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, repo.LabConditionsManifestName)
	writeJSONFixture(t, conditionsPath, ConditionsManifest{
		SchemaVersion: SchemaVersion,
		Conditions: []Condition{
			{
				ConditionID:  "with-graph",
				WorkflowMode: "mystery_mode",
				Description:  "unexpected mode",
			},
		},
		BlockingFactors: []string{
			BlockingFactorTaskFamily,
			BlockingFactorModel,
			BlockingFactorSessionTopology,
		},
	})

	_, err := service.LoadConditionsManifest(context.Background(), repoDir)
	if err == nil {
		t.Fatal("expected LoadConditionsManifest to return an error")
	}

	detail, ok := cmdsupport.ErrorDetailFromError(err)
	if !ok {
		t.Fatalf("error = %T, want structured command error", err)
	}
	if detail.Category != "validation_error" {
		t.Fatalf("error.category = %q, want validation_error", detail.Category)
	}
	if detail.Type != "manifest_error" {
		t.Fatalf("error.type = %q, want manifest_error", detail.Type)
	}
	if detail.Code != "manifest_validation_failed" {
		t.Fatalf("error.code = %q, want manifest_validation_failed", detail.Code)
	}
	assertViolationPresent(t, detail, "conditions[0].workflow_mode", "workflow_mode must be recognized")
}

func initLabRepo(t *testing.T) (string, *repo.Manager) {
	t.Helper()

	repoDir := t.TempDir()
	runCommand(t, repoDir, "git", "init")
	runCommand(t, repoDir, "git", "config", "user.email", "test@example.com")
	runCommand(t, repoDir, "git", "config", "user.name", "Test User")
	writeFixture(t, filepath.Join(repoDir, "README.md"), "# test repo\n")
	return repoDir, repo.NewManager(repo.NewGitRepositoryLocator())
}

func runCommand(t *testing.T, dir, name string, args ...string) {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, output)
	}
}

func writeFixture(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}

func writeJSONFixture(t *testing.T, path string, payload any) {
	t.Helper()

	data, err := marshalJSON(payload)
	if err != nil {
		t.Fatalf("marshalJSON(%s): %v", path, err)
	}
	writeFixture(t, path, string(data))
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func readSuiteManifest(t *testing.T, path string) SuiteManifest {
	t.Helper()

	var manifest SuiteManifest
	payload := readFile(t, path)
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatalf("json.Unmarshal suite manifest: %v", err)
	}
	return manifest
}

func readConditionsManifest(t *testing.T, path string) ConditionsManifest {
	t.Helper()

	var manifest ConditionsManifest
	payload := readFile(t, path)
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatalf("json.Unmarshal conditions manifest: %v", err)
	}
	return manifest
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()

	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%s): %v", path, err)
	}
	return payload
}

func assertViolationPresent(t *testing.T, detail cmdsupport.ErrorDetail, field, message string) {
	t.Helper()

	violations, ok := detail.Details["violations"].([]map[string]any)
	if !ok {
		raw, exists := detail.Details["violations"]
		if !exists {
			t.Fatalf("error.details = %#v, missing violations", detail.Details)
		}

		items, ok := raw.([]any)
		if !ok {
			t.Fatalf("violations = %#v, want []any or []map[string]any", raw)
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

	for _, violation := range violations {
		if violation["field"] == field && violation["message"] == message {
			return
		}
	}

	t.Fatalf("violations = %#v, want field=%q message=%q", violations, field, message)
}
