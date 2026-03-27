package workflowcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/app/workflow"
	graphpayload "github.com/guillaume-galp/cge/internal/domain/payload"
	"github.com/guillaume-galp/cge/internal/infra/benchmarks"
	"github.com/guillaume-galp/cge/internal/infra/kuzu"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func TestWorkflowInitCommandBootstrapsMissingWorkspaceAndManifest(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	cmd := newCommand(repoDir, workflow.NewService(manager))
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeWorkflowSuccessResponse(t, stdout.Bytes())
	if response.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want v1", response.SchemaVersion)
	}
	if response.Command != "workflow.init" {
		t.Fatalf("command = %q, want workflow.init", response.Command)
	}
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if !response.Result.Workspace.Initialized || response.Result.Workspace.AlreadyInitialized {
		t.Fatalf("workspace = %#v, want initialized", response.Result.Workspace)
	}
	if response.Result.Installed.Count != 5 {
		t.Fatalf("installed count = %d, want 5", response.Result.Installed.Count)
	}
	if response.Result.Refreshed.Count != 0 {
		t.Fatalf("refreshed count = %d, want 0", response.Result.Refreshed.Count)
	}
	if response.Result.Seeded.Count != 3 {
		t.Fatalf("seeded count = %d, want 3 missing-source items", response.Result.Seeded.Count)
	}
	for _, item := range response.Result.Seeded.Items {
		if item.Status != "skipped" {
			t.Fatalf("seeded item status = %q, want skipped", item.Status)
		}
	}

	manifestPath := filepath.Join(repoDir, repo.WorkspaceDirName, repo.WorkflowDirName, repo.WorkflowManifestName)
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("manifest stat: %v", err)
	}
}

func TestWorkflowInitCommandRefreshesIdempotentlyAndReportsPreservedOverrides(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	service := workflow.NewService(manager)

	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("initial Init returned error: %v", err)
	}

	targetAsset := filepath.Join(repoDir, filepath.FromSlash(".graph/workflow/assets/prompts/delegated-graph-workflow.prompt.md"))
	if err := os.WriteFile(targetAsset, []byte("# stale asset\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile asset: %v", err)
	}

	cmd := newCommand(repoDir, service)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeWorkflowSuccessResponse(t, stdout.Bytes())
	if response.Result.Workspace.Initialized || !response.Result.Workspace.AlreadyInitialized {
		t.Fatalf("workspace = %#v, want already initialized", response.Result.Workspace)
	}
	if response.Result.Installed.Count != 0 {
		t.Fatalf("installed count = %d, want 0", response.Result.Installed.Count)
	}
	if response.Result.Refreshed.Count != 1 {
		t.Fatalf("refreshed count = %d, want 1", response.Result.Refreshed.Count)
	}
	if response.Result.Preserved.Count != 0 {
		t.Fatalf("preserved count = %d, want 0", response.Result.Preserved.Count)
	}
	if response.Result.Skipped.Count != 4 {
		t.Fatalf("skipped count = %d, want 4", response.Result.Skipped.Count)
	}
	if got := response.Result.Refreshed.Items[0].Path; got != ".graph/workflow/assets/prompts/delegated-graph-workflow.prompt.md" {
		t.Fatalf("refreshed path = %q, want prompt asset path", got)
	}
	if response.Result.Seeded.Count != 3 {
		t.Fatalf("seeded count = %d, want 3 missing-source items", response.Result.Seeded.Count)
	}
}

func TestWorkflowInitCommandReportsPreservedWorkflowAssetOverrides(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	service := workflow.NewService(manager)

	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("initial Init returned error: %v", err)
	}

	manifestPath := filepath.Join(repoDir, repo.WorkspaceDirName, repo.WorkflowDirName, repo.WorkflowManifestName)
	manifestPayload := []byte(`{
  "schema_version": "v1",
  "installed_at": "2026-03-22T17:00:00Z",
  "refreshed_at": "2026-03-22T17:00:00Z",
  "assets": [
    {
      "path": ".graph/workflow/manifest.json",
      "kind": "workflow_manifest",
      "status": "installed"
    }
  ],
  "preserved_overrides": [
    {
      "path": ".graph/workflow/assets/instructions/delegated-graph-workflow.instructions.md",
      "kind": "workflow_instruction_override",
      "reason": "repo keeps a custom delegated instruction"
    }
  ]
}`)
	if err := os.WriteFile(manifestPath, append(manifestPayload, '\n'), 0o644); err != nil {
		t.Fatalf("os.WriteFile manifest: %v", err)
	}
	overridePath := filepath.Join(repoDir, ".graph", "workflow", "assets", "instructions", "delegated-graph-workflow.instructions.md")
	if err := os.WriteFile(overridePath, []byte("# repo override\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile override asset: %v", err)
	}

	cmd := newCommand(repoDir, service)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeWorkflowSuccessResponse(t, stdout.Bytes())
	if response.Result.Preserved.Count != 1 {
		t.Fatalf("preserved count = %d, want 1", response.Result.Preserved.Count)
	}
	if got := response.Result.Preserved.Items[0].Path; got != ".graph/workflow/assets/instructions/delegated-graph-workflow.instructions.md" {
		t.Fatalf("preserved path = %q, want override path", got)
	}
}

func TestWorkflowInitCommandDoesNotReseedUnchangedBaselineKnowledge(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	writeWorkflowFixture(t, filepath.Join(repoDir, "README.md"), `# Cognitive Graph Engine

> A local, chainable graph memory CLI for AI agents.
`)
	writeWorkflowFixture(t, filepath.Join(repoDir, "docs", "architecture", "components.md"), "# Components\n")
	writeWorkflowFixture(t, filepath.Join(repoDir, "docs", "plan", "backlog.yaml"), "project: cognitive-graph-engine\n")

	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	service := workflow.NewService(manager)
	service.NowForTest(func() time.Time { return time.Date(2026, 3, 22, 20, 0, 0, 0, time.UTC) })

	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("initial Init returned error: %v", err)
	}

	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}
	store := kuzu.NewStore()
	initialRevision, err := store.CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision returned error: %v", err)
	}

	service.NowForTest(func() time.Time { return time.Date(2026, 3, 22, 20, 30, 0, 0, time.UTC) })
	cmd := newCommand(repoDir, service)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeWorkflowSuccessResponse(t, stdout.Bytes())
	if response.Result.Seeded.Count != 3 {
		t.Fatalf("seeded count = %d, want 3", response.Result.Seeded.Count)
	}
	for _, item := range response.Result.Seeded.Items {
		if item.Status != "skipped" {
			t.Fatalf("seeded item status = %q, want skipped", item.Status)
		}
	}

	currentRevision, err := store.CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision after command returned error: %v", err)
	}
	if currentRevision.Revision.ID != initialRevision.Revision.ID {
		t.Fatalf("revision id = %q, want %q on no-op refresh", currentRevision.Revision.ID, initialRevision.Revision.ID)
	}
}

func TestWorkflowBenchmarkCommandSummarizesComparableScenario(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}

	service := workflow.NewService(manager)
	store := benchmarks.NewStore()
	store.NowForTest(func() time.Time { return time.Date(2026, 3, 24, 9, 0, 0, 0, time.UTC) })
	service.BenchmarkStoreForTest(store)

	scenarioID := "delegated-subtask-summary-001"
	if _, err := service.StoreBenchmarkScenario(context.Background(), repoDir, benchmarks.Scenario{
		ScenarioID:            scenarioID,
		TaskFamily:            "delegated-non-trivial-subtask",
		Title:                 "Summarize benchmark comparison",
		AcceptanceCriteriaRef: "TH3.E3.US2",
		Modes: []benchmarks.ScenarioMode{
			{Mode: benchmarks.ModeWithGraph, TaskPrompt: "Use workflow start context and graph retrieval."},
			{Mode: benchmarks.ModeWithoutGraph, TaskPrompt: "Use repository inspection only."},
		},
	}); err != nil {
		t.Fatalf("StoreBenchmarkScenario returned error: %v", err)
	}
	if _, err := service.RecordBenchmarkRun(context.Background(), repoDir, benchmarks.RunReport{
		RunID:      "run-with-graph-001",
		ScenarioID: scenarioID,
		Mode:       benchmarks.ModeWithGraph,
		Metrics: benchmarks.RunMetrics{
			Volume: benchmarks.VolumeMetrics{
				InputTokens:      1050,
				OutputTokens:     240,
				PromptCount:      3,
				PromptCharacters: 6400,
			},
			Orientation: benchmarks.OrientationMetrics{
				StepCount:       3,
				RepoScans:       1,
				FollowUpPrompts: 1,
				ContextReloads:  0,
			},
			Outcome: benchmarks.OutcomeSignals{
				QualityRating:          "pass",
				ResumabilityRating:     "strong",
				AcceptanceChecksPassed: 3,
				AcceptanceChecksTotal:  3,
			},
		},
	}); err != nil {
		t.Fatalf("RecordBenchmarkRun with_graph returned error: %v", err)
	}
	if _, err := service.RecordBenchmarkRun(context.Background(), repoDir, benchmarks.RunReport{
		RunID:      "run-without-graph-001",
		ScenarioID: scenarioID,
		Mode:       benchmarks.ModeWithoutGraph,
		Metrics: benchmarks.RunMetrics{
			Volume: benchmarks.VolumeMetrics{
				InputTokens:      1420,
				OutputTokens:     295,
				PromptCount:      5,
				PromptCharacters: 9100,
			},
			Orientation: benchmarks.OrientationMetrics{
				StepCount:       6,
				RepoScans:       4,
				FollowUpPrompts: 2,
				ContextReloads:  1,
			},
			Outcome: benchmarks.OutcomeSignals{
				QualityRating:          "pass",
				ResumabilityRating:     "partial",
				AcceptanceChecksPassed: 3,
				AcceptanceChecksTotal:  3,
			},
		},
	}); err != nil {
		t.Fatalf("RecordBenchmarkRun without_graph returned error: %v", err)
	}

	cmd := newCommand(repoDir, service)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"benchmark", "--scenario", scenarioID})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeWorkflowBenchmarkSuccessResponse(t, stdout.Bytes())
	if response.Command != "workflow.benchmark" || response.Status != "ok" {
		t.Fatalf("response = %#v, want workflow.benchmark ok", response)
	}
	if response.Result.Count != 1 {
		t.Fatalf("result.count = %d, want 1", response.Result.Count)
	}
	summary := response.Result.Summaries[0]
	if summary.Status != workflow.BenchmarkSummaryStatusComparable || !summary.Comparable {
		t.Fatalf("summary = %#v, want comparable status", summary)
	}
	if summary.Runs.WithGraph.Count != 1 || summary.Runs.WithoutGraph.Count != 1 {
		t.Fatalf("runs = %#v, want one selected run per mode", summary.Runs)
	}
	if summary.Comparison == nil {
		t.Fatal("expected benchmark comparison payload")
	}
	if got := summary.Comparison.Volume.InputTokens.DeltaVsWithoutGraph; got != -370 {
		t.Fatalf("input token delta = %d, want -370", got)
	}
	if got := summary.Comparison.Orientation.StepCount.DeltaVsWithoutGraph; got != -3 {
		t.Fatalf("step count delta = %d, want -3", got)
	}
	if !summary.Comparison.Outcome.QualityRating.Matches {
		t.Fatalf("quality rating comparison = %#v, want matching pass ratings", summary.Comparison.Outcome.QualityRating)
	}
	if summary.Comparison.Outcome.ResumabilityRating.Matches {
		t.Fatalf("resumability comparison = %#v, want differing ratings", summary.Comparison.Outcome.ResumabilityRating)
	}
}

func TestWorkflowBenchmarkCommandFlagsIncompleteComparisonData(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}

	service := workflow.NewService(manager)
	store := benchmarks.NewStore()
	store.NowForTest(func() time.Time { return time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC) })
	service.BenchmarkStoreForTest(store)

	scenarioID := "delegated-subtask-summary-002"
	if _, err := service.StoreBenchmarkScenario(context.Background(), repoDir, benchmarks.Scenario{
		ScenarioID: scenarioID,
		TaskFamily: "delegated-non-trivial-subtask",
		Modes: []benchmarks.ScenarioMode{
			{Mode: benchmarks.ModeWithGraph, TaskPrompt: "Use workflow context."},
			{Mode: benchmarks.ModeWithoutGraph, TaskPrompt: "Use repository inspection only."},
		},
	}); err != nil {
		t.Fatalf("StoreBenchmarkScenario returned error: %v", err)
	}
	if _, err := service.RecordBenchmarkRun(context.Background(), repoDir, benchmarks.RunReport{
		RunID:      "run-with-graph-002",
		ScenarioID: scenarioID,
		Mode:       benchmarks.ModeWithGraph,
		Metrics: benchmarks.RunMetrics{
			Volume: benchmarks.VolumeMetrics{
				InputTokens:      980,
				OutputTokens:     210,
				PromptCount:      3,
				PromptCharacters: 6100,
			},
			Orientation: benchmarks.OrientationMetrics{
				StepCount:       3,
				RepoScans:       1,
				FollowUpPrompts: 1,
			},
			Outcome: benchmarks.OutcomeSignals{
				QualityRating:          "pass",
				ResumabilityRating:     "strong",
				AcceptanceChecksPassed: 2,
				AcceptanceChecksTotal:  2,
			},
		},
	}); err != nil {
		t.Fatalf("RecordBenchmarkRun with_graph returned error: %v", err)
	}
	if _, err := service.RecordBenchmarkRun(context.Background(), repoDir, benchmarks.RunReport{
		RunID:      "run-without-graph-002",
		ScenarioID: scenarioID,
		Mode:       benchmarks.ModeWithoutGraph,
		Metrics: benchmarks.RunMetrics{
			Volume: benchmarks.VolumeMetrics{
				InputTokens: 1200,
			},
			Orientation: benchmarks.OrientationMetrics{
				StepCount: 5,
			},
			Outcome: benchmarks.OutcomeSignals{
				QualityRating: "pass",
			},
		},
	}); err != nil {
		t.Fatalf("RecordBenchmarkRun without_graph returned error: %v", err)
	}

	cmd := newCommand(repoDir, service)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"benchmark", "--scenario", scenarioID})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeWorkflowBenchmarkSuccessResponse(t, stdout.Bytes())
	summary := response.Result.Summaries[0]
	if summary.Status != workflow.BenchmarkSummaryStatusIncomplete || summary.Comparable {
		t.Fatalf("summary = %#v, want incomplete non-comparable response", summary)
	}
	if summary.Comparison != nil {
		t.Fatalf("comparison = %#v, want nil for incomplete result", summary.Comparison)
	}
	if len(summary.Issues) == 0 || summary.Issues[0].Code != "missing_comparison_fields" {
		t.Fatalf("issues = %#v, want missing_comparison_fields", summary.Issues)
	}
}

func TestWorkflowBenchmarkCommandReturnsStructuredArtifactLoadError(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}

	service := workflow.NewService(manager)
	store := benchmarks.NewStore()
	service.BenchmarkStoreForTest(store)

	scenarioID := "delegated-subtask-summary-003"
	if _, err := service.StoreBenchmarkScenario(context.Background(), repoDir, benchmarks.Scenario{
		ScenarioID: scenarioID,
		TaskFamily: "delegated-non-trivial-subtask",
		Modes: []benchmarks.ScenarioMode{
			{Mode: benchmarks.ModeWithGraph, TaskPrompt: "Use workflow context."},
			{Mode: benchmarks.ModeWithoutGraph, TaskPrompt: "Use repository inspection only."},
		},
	}); err != nil {
		t.Fatalf("StoreBenchmarkScenario returned error: %v", err)
	}

	brokenPath := filepath.Join(repoDir, repo.WorkspaceDirName, repo.BenchmarksDirName, "runs", scenarioID, benchmarks.ModeWithGraph, "broken.json")
	if err := os.MkdirAll(filepath.Dir(brokenPath), 0o755); err != nil {
		t.Fatalf("os.MkdirAll broken path: %v", err)
	}
	if err := os.WriteFile(brokenPath, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("os.WriteFile broken run artifact: %v", err)
	}

	cmd := newCommand(repoDir, service)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"benchmark", "--scenario", scenarioID})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var silentErr *cmdsupport.SilentError
	if !errors.As(err, &silentErr) {
		t.Fatalf("error type = %T, want *cmdsupport.SilentError", err)
	}

	response := decodeWorkflowFailureResponse(t, stdout.Bytes())
	if response.Command != "workflow.benchmark" || response.Status != "error" {
		t.Fatalf("response = %#v, want workflow.benchmark error", response)
	}
	if response.Error.Code != "benchmark_artifact_read_failed" {
		t.Fatalf("error code = %q, want benchmark_artifact_read_failed", response.Error.Code)
	}
	if got := response.Error.Details["stage"]; got != "run_list" {
		t.Fatalf("error details stage = %#v, want run_list", got)
	}
}

func TestWorkflowInitCommandReturnsStructuredRepositoryScopeError(t *testing.T) {
	t.Parallel()

	startDir := t.TempDir()
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	cmd := newCommand(startDir, workflow.NewService(manager))
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init"})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	response := decodeWorkflowFailureResponse(t, stdout.Bytes())
	if response.Command != "workflow.init" {
		t.Fatalf("command = %q, want workflow.init", response.Command)
	}
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Error.Code != "repository_root_not_found" {
		t.Fatalf("error code = %q, want repository_root_not_found", response.Error.Code)
	}
	if _, statErr := os.Stat(filepath.Join(startDir, repo.WorkspaceDirName)); !os.IsNotExist(statErr) {
		t.Fatalf("workspace dir unexpectedly exists: stat err = %v", statErr)
	}
}

func TestWorkflowInitCommandReturnsStructuredSeedPersistenceError(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	writeWorkflowFixture(t, filepath.Join(repoDir, "README.md"), "# Cognitive Graph Engine\n")
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	service := workflow.NewService(manager)
	service.NowForTest(func() time.Time { return time.Date(2026, 3, 22, 20, 15, 0, 0, time.UTC) })
	service.SeedWriterForTest(failingWorkflowSeedWriter{err: &kuzu.PersistenceError{
		Code:    "revision_anchor_unavailable",
		Message: "graph write could not record revision metadata",
		Details: map[string]any{"reason": "forced failure"},
	}})

	cmd := newCommand(repoDir, service)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init"})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var silentErr *cmdsupport.SilentError
	if !errors.As(err, &silentErr) {
		t.Fatalf("error type = %T, want *cmdsupport.SilentError", err)
	}

	response := decodeWorkflowFailureResponse(t, stdout.Bytes())
	if response.Error.Code != "workflow_seed_persistence_failed" {
		t.Fatalf("error code = %q, want workflow_seed_persistence_failed", response.Error.Code)
	}
	if response.Error.Type != "persistence_error" {
		t.Fatalf("error type = %q, want persistence_error", response.Error.Type)
	}
	if got := response.Error.Details["cause_code"]; got != "revision_anchor_unavailable" {
		t.Fatalf("error details cause_code = %#v, want revision_anchor_unavailable", got)
	}
}

func TestWorkflowInitCommandReturnsStructuredAssetSyncError(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	blockedPath := filepath.Join(repoDir, ".graph", "workflow", "assets", "prompts", "delegated-graph-workflow.prompt.md")
	if err := os.MkdirAll(blockedPath, 0o755); err != nil {
		t.Fatalf("os.MkdirAll blocked path: %v", err)
	}

	cmd := newCommand(repoDir, workflow.NewService(manager))
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init"})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var silentErr *cmdsupport.SilentError
	if !errors.As(err, &silentErr) {
		t.Fatalf("error type = %T, want *cmdsupport.SilentError", err)
	}

	response := decodeWorkflowFailureResponse(t, stdout.Bytes())
	if response.Error.Code != "workflow_asset_sync_failed" {
		t.Fatalf("error code = %q, want workflow_asset_sync_failed", response.Error.Code)
	}
	if got := response.Error.Details["path"]; got != ".graph/workflow/assets/prompts/delegated-graph-workflow.prompt.md" {
		t.Fatalf("error details path = %#v, want prompt asset path", got)
	}
}

func TestWorkflowInitCommandDoesNotPartiallyRefreshAssetsOnFailure(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	service := workflow.NewService(manager)

	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("initial Init returned error: %v", err)
	}

	staleAssetPath := filepath.Join(repoDir, ".graph", "workflow", "assets", "prompts", "delegated-graph-workflow.prompt.md")
	if err := os.WriteFile(staleAssetPath, []byte("# stale asset\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile stale asset: %v", err)
	}

	blockedPath := filepath.Join(repoDir, ".graph", "workflow", "assets", "instructions", "delegated-graph-workflow.instructions.md")
	if err := os.Remove(blockedPath); err != nil {
		t.Fatalf("os.Remove blocked file: %v", err)
	}
	if err := os.MkdirAll(blockedPath, 0o755); err != nil {
		t.Fatalf("os.MkdirAll blocked path: %v", err)
	}

	cmd := newCommand(repoDir, service)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"init"})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var silentErr *cmdsupport.SilentError
	if !errors.As(err, &silentErr) {
		t.Fatalf("error type = %T, want *cmdsupport.SilentError", err)
	}

	response := decodeWorkflowFailureResponse(t, stdout.Bytes())
	if response.Error.Code != "workflow_asset_sync_failed" {
		t.Fatalf("error code = %q, want workflow_asset_sync_failed", response.Error.Code)
	}
	if got := response.Error.Details["path"]; got != ".graph/workflow/assets/instructions/delegated-graph-workflow.instructions.md" {
		t.Fatalf("error details path = %#v, want instructions asset path", got)
	}

	payload, readErr := os.ReadFile(staleAssetPath)
	if readErr != nil {
		t.Fatalf("os.ReadFile stale asset after failure: %v", readErr)
	}
	if string(payload) != "# stale asset\n" {
		t.Fatalf("stale asset content = %q, want stale content preserved after failure", string(payload))
	}
}

func TestWorkflowStartCommandReturnsProceedRecommendationForHealthyGraph(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}
	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}

	writeWorkflowGraph(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-401",
    "timestamp": "2026-03-22T22:00:00Z",
    "revision": {
      "reason": "Seed workflow start command test"
    }
  },
  "nodes": [
    {
      "id": "story:workflow-start",
      "kind": "UserStory",
      "title": "Implement delegated workflow start",
      "summary": "Inspect readiness before delegated work begins"
    },
    {
      "id": "doc:workflow-contract",
      "kind": "Document",
      "title": "Workflow start contract",
      "summary": "Documents readiness recommendations for delegated kickoff"
    },
    {
      "id": "adr:delegation",
      "kind": "ADR",
      "title": "Delegate with graph-backed readiness",
      "summary": "Explains why the current revision and health indicators matter"
    }
  ],
  "edges": [
    {
      "from": "story:workflow-start",
      "to": "doc:workflow-contract",
      "kind": "RELATES_TO"
    },
    {
      "from": "doc:workflow-contract",
      "to": "adr:delegation",
      "kind": "CITES"
    },
    {
      "from": "adr:delegation",
      "to": "story:workflow-start",
      "kind": "ABOUT"
    }
  ]
}`)

	cmd := newCommand(repoDir, workflow.NewService(manager))
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"start", "--task", "implement delegated workflow start", "--max-tokens", "1200"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeWorkflowStartSuccessResponse(t, stdout.Bytes())
	if response.Command != "workflow.start" || response.Status != "ok" {
		t.Fatalf("response = %#v, want workflow.start ok", response)
	}
	if response.Result.Task.Value != "implement delegated workflow start" {
		t.Fatalf("task.value = %q, want task text", response.Result.Task.Value)
	}
	if response.Result.Task.Source != "flag" {
		t.Fatalf("task.source = %q, want flag", response.Result.Task.Source)
	}
	if response.Result.Recommendation != workflow.RecommendationProceed {
		t.Fatalf("recommendation = %q, want %q", response.Result.Recommendation, workflow.RecommendationProceed)
	}
	if !response.Result.Readiness.GraphState.CurrentRevision.Exists {
		t.Fatal("expected current revision to exist")
	}
	if response.Result.Readiness.GraphState.Health.Snapshot.Nodes != 3 {
		t.Fatalf("snapshot.nodes = %d, want 3", response.Result.Readiness.GraphState.Health.Snapshot.Nodes)
	}
	if response.Result.Kickoff.Task.Description != "implement delegated workflow start" {
		t.Fatalf("kickoff.task.description = %q, want task text", response.Result.Kickoff.Task.Description)
	}
	if response.Result.Kickoff.Task.Family != workflow.KickoffFamilyWorkflowContext {
		t.Fatalf("kickoff.task.family = %q, want %q", response.Result.Kickoff.Task.Family, workflow.KickoffFamilyWorkflowContext)
	}
	if response.Result.Kickoff.Policy.Family != workflow.KickoffFamilyWorkflowContext {
		t.Fatalf("kickoff.policy.family = %q, want %q", response.Result.Kickoff.Policy.Family, workflow.KickoffFamilyWorkflowContext)
	}
	if response.Result.Kickoff.Advisory.EffectiveMode != workflow.KickoffModeInject {
		t.Fatalf("kickoff.advisory.effective_mode = %q, want %q", response.Result.Kickoff.Advisory.EffectiveMode, workflow.KickoffModeInject)
	}
	if response.Result.Kickoff.Context.Coverage != workflow.KickoffCoverageGrounded {
		t.Fatalf("kickoff.context.coverage = %q, want %q", response.Result.Kickoff.Context.Coverage, workflow.KickoffCoverageGrounded)
	}
	if len(response.Result.Kickoff.Context.Envelope.Results) == 0 {
		t.Fatal("expected kickoff context results")
	}
	if response.Result.Kickoff.Context.Envelope.Results[0].InclusionReason == "" {
		t.Fatal("expected inclusion reason in kickoff context result")
	}
	if response.Result.Kickoff.DelegationBrief.Status != workflow.KickoffCoverageGrounded {
		t.Fatalf("delegation_brief.status = %q, want %q", response.Result.Kickoff.DelegationBrief.Status, workflow.KickoffCoverageGrounded)
	}
}

func TestWorkflowStartCommandReturnsBootstrapRecommendationForMissingGraph(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}

	cmd := newCommand(repoDir, workflow.NewService(manager))
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"start", "--task", "implement delegated workflow start"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeWorkflowStartSuccessResponse(t, stdout.Bytes())
	if response.Result.Recommendation != workflow.RecommendationBootstrap {
		t.Fatalf("recommendation = %q, want %q", response.Result.Recommendation, workflow.RecommendationBootstrap)
	}
	if response.Result.Readiness.GraphState.GraphAvailable {
		t.Fatal("expected graph_available to be false")
	}
}

func TestWorkflowStartCommandAbstainsForReportingTasks(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}
	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}

	writeWorkflowGraph(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-402",
    "timestamp": "2026-03-27T20:00:00Z",
    "revision": {"reason": "Seed reporting abstention test"}
  },
  "nodes": [
    {
      "id": "report:campaign",
      "kind": "Document",
      "title": "Campaign report",
      "summary": "Summarizes the VP5 campaign findings"
    },
    {
      "id": "doc:reporting-policy",
      "kind": "Document",
      "title": "Reporting policy",
      "summary": "Documents abstention-first kickoff for reporting tasks"
    },
    {
      "id": "adr:precision-kickoff",
      "kind": "ADR",
      "title": "Precision-governed kickoff",
      "summary": "Explains why reporting defaults to no kickoff"
    }
  ],
  "edges": [
    {
      "from": "report:campaign",
      "to": "doc:reporting-policy",
      "kind": "RELATES_TO"
    },
    {
      "from": "doc:reporting-policy",
      "to": "adr:precision-kickoff",
      "kind": "CITES"
    }
  ]
}`)

	cmd := newCommand(repoDir, workflow.NewService(manager))
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"start", "--task", "Produce a synthesis report of campaign findings", "--max-tokens", "1200"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeWorkflowStartSuccessResponse(t, stdout.Bytes())
	if response.Result.Recommendation != workflow.RecommendationProceed {
		t.Fatalf("recommendation = %q, want %q", response.Result.Recommendation, workflow.RecommendationProceed)
	}
	if response.Result.Kickoff.Task.Family != workflow.KickoffFamilyReportingSynthesis {
		t.Fatalf("kickoff.task.family = %q, want %q", response.Result.Kickoff.Task.Family, workflow.KickoffFamilyReportingSynthesis)
	}
	if response.Result.Kickoff.Context.Coverage != workflow.KickoffCoverageAbstained {
		t.Fatalf("kickoff.context.coverage = %q, want %q", response.Result.Kickoff.Context.Coverage, workflow.KickoffCoverageAbstained)
	}
	if !response.Result.Kickoff.Context.Abstained {
		t.Fatal("expected abstained kickoff context")
	}
	if response.Result.Kickoff.Advisory.EffectiveMode != workflow.KickoffModeAbstain {
		t.Fatalf("effective mode = %q, want %q", response.Result.Kickoff.Advisory.EffectiveMode, workflow.KickoffModeAbstain)
	}
	if got := len(response.Result.Kickoff.Context.Envelope.Results); got != 0 {
		t.Fatalf("result count = %d, want 0 for abstained reporting kickoff", got)
	}
}

func TestWorkflowStartCommandSupportsExplicitNoKickoffMode(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}
	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}

	writeWorkflowGraph(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-403",
    "timestamp": "2026-03-27T20:10:00Z",
    "revision": {"reason": "Seed explicit no-kickoff test"}
  },
  "nodes": [
    {
      "id": "doc:kickoff-policy",
      "kind": "Document",
      "title": "Kickoff policy",
      "summary": "Documents kickoff modes"
    },
    {
      "id": "story:kickoff-controls",
      "kind": "UserStory",
      "title": "Kickoff controls",
      "summary": "Adds explicit no-kickoff and minimal-kickoff controls"
    },
    {
      "id": "adr:advisory-kickoff",
      "kind": "ADR",
      "title": "Advisory kickoff",
      "summary": "Explains freedom-preserving kickoff behavior"
    }
  ],
  "edges": [
    {
      "from": "story:kickoff-controls",
      "to": "doc:kickoff-policy",
      "kind": "RELATES_TO"
    },
    {
      "from": "doc:kickoff-policy",
      "to": "adr:advisory-kickoff",
      "kind": "CITES"
    }
  ]
}`)

	cmd := newCommand(repoDir, workflow.NewService(manager))
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"start", "--task", "Write the production handler for kickoff classification", "--kickoff-mode", "none"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeWorkflowStartSuccessResponse(t, stdout.Bytes())
	if response.Result.Recommendation != workflow.RecommendationProceed {
		t.Fatalf("recommendation = %q, want %q", response.Result.Recommendation, workflow.RecommendationProceed)
	}
	if response.Result.Kickoff.Advisory.RequestedMode != workflow.KickoffModeAbstain {
		t.Fatalf("requested mode = %q, want %q", response.Result.Kickoff.Advisory.RequestedMode, workflow.KickoffModeAbstain)
	}
	if response.Result.Kickoff.Advisory.EffectiveMode != workflow.KickoffModeAbstain {
		t.Fatalf("effective mode = %q, want %q", response.Result.Kickoff.Advisory.EffectiveMode, workflow.KickoffModeAbstain)
	}
	if response.Result.Kickoff.Context.Coverage != workflow.KickoffCoverageAbstained {
		t.Fatalf("coverage = %q, want %q", response.Result.Kickoff.Context.Coverage, workflow.KickoffCoverageAbstained)
	}
}

func TestWorkflowStartCommandReturnsStructuredOperationalErrorOnInspectionFailure(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	starter := mockWorkflowStarter{err: cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "persistence_error",
		Code:     "graph_read_failed",
		Message:  "graph state could not be read",
		Details: map[string]any{
			"stage":  "graph_state",
			"reason": "forced failure",
		},
	}, errors.New("forced failure"))}

	cmd := &bytes.Buffer{}
	command := &bytes.Buffer{}
	workflowCmd := newCommand(repoDir, starterWithInit{Initializer: workflow.NewService(manager), Starter: starter})
	workflowCmd.SetOut(command)
	workflowCmd.SetErr(cmd)
	workflowCmd.SetArgs([]string{"start", "--task", "implement delegated workflow start"})

	err := workflowCmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var silentErr *cmdsupport.SilentError
	if !errors.As(err, &silentErr) {
		t.Fatalf("error type = %T, want *cmdsupport.SilentError", err)
	}

	response := decodeWorkflowFailureResponse(t, command.Bytes())
	if response.Command != "workflow.start" {
		t.Fatalf("command = %q, want workflow.start", response.Command)
	}
	if response.Error.Code != "graph_read_failed" {
		t.Fatalf("error code = %q, want graph_read_failed", response.Error.Code)
	}
}

func TestWorkflowFinishCommandPersistsDelegatedOutcomeFromFileAndReturnsHandoffEnvelope(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}
	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}
	writeWorkflowGraph(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "seed-session",
    "timestamp": "2026-03-22T16:00:00Z"
  },
  "nodes": [
    {
      "id": "project:cge",
      "kind": "ProjectMetadata",
      "title": "CGE",
      "summary": "Seed graph state for workflow finish command tests"
    }
  ],
  "edges": []
}`)

	payloadPath := filepath.Join(t.TempDir(), "task-outcome.json")
	writeWorkflowFixture(t, payloadPath, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "finish-session",
    "timestamp": "2026-03-22T17:00:00Z"
  },
  "task": "implement workflow finish",
  "summary": "Persisted delegated task outcome in the graph and returned a handoff brief.",
  "decisions": [
    {
      "summary": "Reuse the normal graph write path for workflow finish",
      "rationale": "Keeps graph revisions consistent",
      "status": "accepted"
    }
  ],
  "changed_artifacts": [
    {
      "path": "internal/app/workflow/finish.go",
      "summary": "Added the workflow finish implementation",
      "change_type": "updated",
      "language": "go"
    }
  ],
  "follow_up": [
    {
      "summary": "Use the returned handoff brief when delegating the next task",
      "owner": "next-agent",
      "status": "open"
    }
  ]
}`)

	cmd := newCommand(repoDir, workflow.NewService(manager))
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"finish", "--file", payloadPath})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeWorkflowFinishSuccessResponse(t, stdout.Bytes())
	if response.Command != "workflow.finish" {
		t.Fatalf("command = %q, want workflow.finish", response.Command)
	}
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if !response.Result.BeforeRevision.Exists || !response.Result.AfterRevision.Exists {
		t.Fatalf("revision state = %#v, want before and after revisions", response.Result)
	}
	if response.Result.BeforeRevision.Revision.ID == response.Result.AfterRevision.Revision.ID {
		t.Fatalf("before revision id = %q, want a new revision", response.Result.BeforeRevision.Revision.ID)
	}
	if response.Result.WriteSummary.Status != workflow.FinishWriteStatusApplied {
		t.Fatalf("write status = %q, want %q", response.Result.WriteSummary.Status, workflow.FinishWriteStatusApplied)
	}
	if response.Result.HandoffBrief == nil {
		t.Fatal("expected handoff brief, got nil")
	}
	if response.Result.HandoffBrief.Status != workflow.FinishHandoffStatusReady {
		t.Fatalf("handoff status = %q, want %q", response.Result.HandoffBrief.Status, workflow.FinishHandoffStatusReady)
	}
	if response.Result.NoOp != nil {
		t.Fatalf("no_op = %#v, want nil", response.Result.NoOp)
	}
}

func TestWorkflowFinishCommandReturnsExplicitNoOpWhenPayloadHasNoDurableUpdates(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}
	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}
	writeWorkflowGraph(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "seed-session",
    "timestamp": "2026-03-22T16:15:00Z"
  },
  "nodes": [
    {
      "id": "project:cge",
      "kind": "ProjectMetadata",
      "title": "CGE",
      "summary": "Seed graph state for workflow finish no-op command tests"
    }
  ],
  "edges": []
}`)
	initialRevision, err := kuzu.NewStore().CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision before finish returned error: %v", err)
	}

	payloadPath := filepath.Join(t.TempDir(), "task-outcome.json")
	writeWorkflowFixture(t, payloadPath, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "finish-session",
    "timestamp": "2026-03-22T17:15:00Z"
  },
  "task": "document no-op workflow finish",
  "summary": "This finish payload is valid but intentionally contains no durable graph updates.",
  "decisions": [],
  "changed_artifacts": [],
  "follow_up": []
}`)

	cmd := newCommand(repoDir, workflow.NewService(manager))
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"finish", "--file", payloadPath})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeWorkflowFinishSuccessResponse(t, stdout.Bytes())
	if response.Result.WriteSummary.Status != workflow.FinishWriteStatusNoOp {
		t.Fatalf("write status = %q, want %q", response.Result.WriteSummary.Status, workflow.FinishWriteStatusNoOp)
	}
	if response.Result.NoOp == nil {
		t.Fatal("expected explicit no-op result")
	}
	if response.Result.HandoffBrief != nil {
		t.Fatalf("handoff_brief = %#v, want nil on no-op", response.Result.HandoffBrief)
	}
	if response.Result.BeforeRevision.Revision.ID != initialRevision.Revision.ID || response.Result.AfterRevision.Revision.ID != initialRevision.Revision.ID {
		t.Fatalf("revision ids = (%q, %q), want unchanged %q", response.Result.BeforeRevision.Revision.ID, response.Result.AfterRevision.Revision.ID, initialRevision.Revision.ID)
	}
}

func TestWorkflowFinishCommandRejectsUnknownNestedFieldsWithoutMutation(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}
	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}
	writeWorkflowGraph(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "seed-session",
    "timestamp": "2026-03-22T16:30:00Z"
  },
  "nodes": [
    {
      "id": "project:cge",
      "kind": "ProjectMetadata",
      "title": "CGE",
      "summary": "Seed graph state for workflow finish validation command tests"
    }
  ],
  "edges": []
}`)
	initialRevision, err := kuzu.NewStore().CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision before finish returned error: %v", err)
	}

	payloadPath := filepath.Join(t.TempDir(), "task-outcome.json")
	writeWorkflowFixture(t, payloadPath, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "finish-session",
    "timestamp": "2026-03-22T17:30:00Z"
  },
  "task": "reject invalid workflow finish payload",
  "summary": "This payload should be rejected without mutating the graph.",
  "decisions": [],
  "changed_artifacts": [],
  "follow_up": [
    {
      "summary": "Hand the task back to the next agent",
      "owner": "next-agent",
      "status": "open",
      "unexpected": "reject me"
    }
  ]
}`)

	cmd := newCommand(repoDir, workflow.NewService(manager))
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"finish", "--file", payloadPath})

	err = cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var silentErr *cmdsupport.SilentError
	if !errors.As(err, &silentErr) {
		t.Fatalf("error type = %T, want *cmdsupport.SilentError", err)
	}

	response := decodeWorkflowFailureResponse(t, stdout.Bytes())
	if response.Command != "workflow.finish" {
		t.Fatalf("command = %q, want workflow.finish", response.Command)
	}
	if response.Error.Code != "invalid_finish_payload" {
		t.Fatalf("error code = %q, want invalid_finish_payload", response.Error.Code)
	}
	if response.Error.Type != "validation_error" {
		t.Fatalf("error type = %q, want validation_error", response.Error.Type)
	}
	if got := response.Error.Details["field"]; got != "follow_up" {
		t.Fatalf("error details field = %#v, want follow_up", got)
	}
	if got := response.Error.Details["index"]; got != float64(0) {
		t.Fatalf("error details index = %#v, want 0", got)
	}
	if got, _ := response.Error.Details["reason"].(string); !strings.Contains(got, `unknown field "unexpected"`) {
		t.Fatalf("error details reason = %#v, want unknown field detail", response.Error.Details["reason"])
	}

	currentRevision, err := kuzu.NewStore().CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision after finish returned error: %v", err)
	}
	if currentRevision.Revision.ID != initialRevision.Revision.ID {
		t.Fatalf("current revision id = %q, want %q after failed finish", currentRevision.Revision.ID, initialRevision.Revision.ID)
	}
}

func TestWorkflowFinishCommandRejectsWindowsTraversalPathsWithoutMutation(t *testing.T) {
	t.Parallel()

	repoDir := initGitRepository(t)
	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}
	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}
	writeWorkflowGraph(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "seed-session",
    "timestamp": "2026-03-22T16:30:00Z"
  },
  "nodes": [
    {
      "id": "project:cge",
      "kind": "ProjectMetadata",
      "title": "CGE",
      "summary": "Seed graph state for workflow finish validation command tests"
    }
  ],
  "edges": []
}`)
	initialRevision, err := kuzu.NewStore().CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision before finish returned error: %v", err)
	}

	payloadPath := filepath.Join(t.TempDir(), "task-outcome.json")
	writeWorkflowFixture(t, payloadPath, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "finish-session",
    "timestamp": "2026-03-22T17:30:00Z"
  },
  "task": "reject invalid workflow finish payload",
  "summary": "This payload should be rejected without mutating the graph.",
  "decisions": [],
  "changed_artifacts": [
    {
      "path": "..\\secret.txt",
      "summary": "unsafe path"
    }
  ],
  "follow_up": []
}`)

	cmd := newCommand(repoDir, workflow.NewService(manager))
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"finish", "--file", payloadPath})

	err = cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var silentErr *cmdsupport.SilentError
	if !errors.As(err, &silentErr) {
		t.Fatalf("error type = %T, want *cmdsupport.SilentError", err)
	}

	response := decodeWorkflowFailureResponse(t, stdout.Bytes())
	if response.Command != "workflow.finish" {
		t.Fatalf("command = %q, want workflow.finish", response.Command)
	}
	if response.Error.Code != "unsafe_repo_path" {
		t.Fatalf("error code = %q, want unsafe_repo_path", response.Error.Code)
	}
	if response.Error.Type != "validation_error" {
		t.Fatalf("error type = %q, want validation_error", response.Error.Type)
	}
	if got := response.Error.Details["index"]; got != float64(0) {
		t.Fatalf("error details index = %#v, want 0", got)
	}

	currentRevision, err := kuzu.NewStore().CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision after finish returned error: %v", err)
	}
	if currentRevision.Revision.ID != initialRevision.Revision.ID {
		t.Fatalf("current revision id = %q, want %q after failed finish", currentRevision.Revision.ID, initialRevision.Revision.ID)
	}
}

func initGitRepository(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, output)
	}
	return repoDir
}

type workflowSuccessResponse struct {
	SchemaVersion string `json:"schema_version"`
	Command       string `json:"command"`
	Status        string `json:"status"`
	Result        struct {
		Workspace struct {
			Initialized        bool `json:"initialized"`
			AlreadyInitialized bool `json:"already_initialized"`
		} `json:"workspace"`
		Installed struct {
			Count int `json:"count"`
			Items []struct {
				Path string `json:"path"`
			} `json:"items"`
		} `json:"installed"`
		Refreshed struct {
			Count int `json:"count"`
			Items []struct {
				Path string `json:"path"`
			} `json:"items"`
		} `json:"refreshed"`
		Preserved struct {
			Count int `json:"count"`
			Items []struct {
				Path string `json:"path"`
			} `json:"items"`
		} `json:"preserved"`
		Skipped struct {
			Count int `json:"count"`
		} `json:"skipped"`
		Seeded struct {
			Count int `json:"count"`
			Items []struct {
				Path   string `json:"path"`
				Status string `json:"status"`
			} `json:"items"`
		} `json:"seeded"`
	} `json:"result"`
}

type workflowFailureResponse struct {
	SchemaVersion string `json:"schema_version"`
	Command       string `json:"command"`
	Status        string `json:"status"`
	Error         struct {
		Category string         `json:"category"`
		Type     string         `json:"type"`
		Code     string         `json:"code"`
		Message  string         `json:"message"`
		Details  map[string]any `json:"details"`
	} `json:"error"`
}

type workflowStartSuccessResponse struct {
	SchemaVersion string `json:"schema_version"`
	Command       string `json:"command"`
	Status        string `json:"status"`
	Result        struct {
		Task struct {
			Value  string `json:"value"`
			Source string `json:"source"`
		} `json:"task"`
		Recommendation string `json:"recommendation"`
		Readiness      struct {
			Status     string   `json:"status"`
			Reasons    []string `json:"reasons"`
			GraphState struct {
				WorkspaceInitialized bool                      `json:"workspace_initialized"`
				GraphAvailable       bool                      `json:"graph_available"`
				CurrentRevision      kuzu.CurrentRevisionState `json:"current_revision"`
				Health               struct {
					Available bool            `json:"available"`
					Snapshot  kuzu.GraphStats `json:"snapshot"`
				} `json:"health"`
			} `json:"graph_state"`
		} `json:"readiness"`
		Kickoff struct {
			Task struct {
				Description string `json:"description"`
				MaxTokens   int    `json:"max_tokens"`
				Family      string `json:"family"`
			} `json:"task"`
			Policy struct {
				Family             string   `json:"family"`
				AllowedEntityKinds []string `json:"allowed_entity_kinds"`
				SuppressedPatterns []string `json:"suppressed_patterns"`
				DefaultKickoffMode string   `json:"default_kickoff_mode"`
			} `json:"policy"`
			Advisory struct {
				RequestedMode   string   `json:"requested_mode"`
				EffectiveMode   string   `json:"effective_mode"`
				ConfidenceLevel string   `json:"confidence_level"`
				ConfidenceScore float64  `json:"confidence_score"`
				ReasonCodes     []string `json:"reason_codes"`
				NextStep        string   `json:"next_step"`
			} `json:"advisory"`
			Context struct {
				Coverage         string `json:"coverage"`
				Abstained        bool   `json:"abstained"`
				AbstentionReason string `json:"abstention_reason"`
				Envelope         struct {
					MaxTokens       int  `json:"max_tokens"`
					EstimatedTokens int  `json:"estimated_tokens"`
					Truncated       bool `json:"truncated"`
					Results         []struct {
						InclusionReason string `json:"inclusion_reason"`
						Entity          struct {
							ID string `json:"id"`
						} `json:"entity"`
					} `json:"results"`
				} `json:"envelope"`
			} `json:"context"`
			DelegationBrief struct {
				Status string `json:"status"`
				Prompt string `json:"prompt"`
			} `json:"delegation_brief"`
		} `json:"kickoff"`
	} `json:"result"`
}

type workflowFinishSuccessResponse struct {
	SchemaVersion string `json:"schema_version"`
	Command       string `json:"command"`
	Status        string `json:"status"`
	Result        struct {
		BeforeRevision kuzu.CurrentRevisionState    `json:"before_revision"`
		AfterRevision  kuzu.CurrentRevisionState    `json:"after_revision"`
		WriteSummary   workflow.FinishWriteSummary  `json:"write_summary"`
		HandoffBrief   *workflow.FinishHandoffBrief `json:"handoff_brief"`
		NoOp           *workflow.FinishNoOpResult   `json:"no_op"`
	} `json:"result"`
}

type workflowBenchmarkSuccessResponse struct {
	SchemaVersion string `json:"schema_version"`
	Command       string `json:"command"`
	Status        string `json:"status"`
	Result        struct {
		ScenarioID string `json:"scenario_id"`
		Count      int    `json:"count"`
		Summaries  []struct {
			Scenario struct {
				ScenarioID string `json:"scenario_id"`
			} `json:"scenario"`
			Status     string `json:"status"`
			Comparable bool   `json:"comparable"`
			Issues     []struct {
				Code string `json:"code"`
			} `json:"issues"`
			Runs struct {
				WithGraph struct {
					Count int `json:"count"`
				} `json:"with_graph"`
				WithoutGraph struct {
					Count int `json:"count"`
				} `json:"without_graph"`
			} `json:"runs"`
			Comparison *struct {
				Volume struct {
					InputTokens struct {
						DeltaVsWithoutGraph int `json:"delta_vs_without_graph"`
					} `json:"input_tokens"`
				} `json:"volume"`
				Orientation struct {
					StepCount struct {
						DeltaVsWithoutGraph int `json:"delta_vs_without_graph"`
					} `json:"step_count"`
				} `json:"orientation"`
				Outcome struct {
					QualityRating struct {
						Matches bool `json:"matches"`
					} `json:"quality_rating"`
					ResumabilityRating struct {
						Matches bool `json:"matches"`
					} `json:"resumability_rating"`
				} `json:"outcome"`
			} `json:"comparison"`
		} `json:"summaries"`
	} `json:"result"`
}

func decodeWorkflowSuccessResponse(t *testing.T, payload []byte) workflowSuccessResponse {
	t.Helper()

	var response workflowSuccessResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal success response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeWorkflowFailureResponse(t *testing.T, payload []byte) workflowFailureResponse {
	t.Helper()

	var response workflowFailureResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal failure response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeWorkflowStartSuccessResponse(t *testing.T, payload []byte) workflowStartSuccessResponse {
	t.Helper()

	var response workflowStartSuccessResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal workflow start success response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeWorkflowFinishSuccessResponse(t *testing.T, payload []byte) workflowFinishSuccessResponse {
	t.Helper()

	var response workflowFinishSuccessResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal workflow finish success response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeWorkflowBenchmarkSuccessResponse(t *testing.T, payload []byte) workflowBenchmarkSuccessResponse {
	t.Helper()

	var response workflowBenchmarkSuccessResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal workflow benchmark success response: %v\npayload: %s", err, payload)
	}
	return response
}

type failingWorkflowSeedWriter struct {
	err error
}

func (w failingWorkflowSeedWriter) Write(_ context.Context, _ repo.Workspace, _ graphpayload.Envelope) (kuzu.WriteSummary, error) {
	return kuzu.WriteSummary{}, w.err
}

type starterWithInit struct {
	Initializer
	Starter
	Finisher
}

func (s starterWithInit) Init(ctx context.Context, startDir string) (workflow.InitResult, error) {
	return s.Initializer.Init(ctx, startDir)
}

func (s starterWithInit) Start(ctx context.Context, startDir, task string, maxTokens int) (workflow.StartResult, error) {
	return s.Starter.Start(ctx, startDir, task, maxTokens)
}

func (s starterWithInit) Finish(ctx context.Context, startDir, input string) (workflow.FinishResult, error) {
	if s.Finisher == nil {
		return workflow.FinishResult{}, errors.New("workflow finisher is not configured")
	}
	return s.Finisher.Finish(ctx, startDir, input)
}

type mockWorkflowStarter struct {
	result workflow.StartResult
	err    error
}

func (s mockWorkflowStarter) Start(_ context.Context, _ string, _ string, _ int) (workflow.StartResult, error) {
	if s.err != nil {
		return workflow.StartResult{}, s.err
	}
	return s.result, nil
}

type mockWorkflowFinisher struct {
	result workflow.FinishResult
	err    error
}

func (s mockWorkflowFinisher) Finish(_ context.Context, _ string, _ string) (workflow.FinishResult, error) {
	if s.err != nil {
		return workflow.FinishResult{}, s.err
	}
	return s.result, nil
}

func writeWorkflowGraph(t *testing.T, workspace repo.Workspace, payload string) kuzu.WriteSummary {
	t.Helper()

	envelope, err := graphpayload.ParseAndValidate(payload)
	if err != nil {
		t.Fatalf("ParseAndValidate returned error: %v", err)
	}
	summary, err := kuzu.NewStore().Write(context.Background(), workspace, envelope)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	return summary
}

func writeWorkflowFixture(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll fixture dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile fixture: %v", err)
	}
}
