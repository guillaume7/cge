package workflow

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/infra/benchmarks"
	"github.com/guillaume-galp/cge/internal/infra/kuzu"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func TestServiceStoresComparableBenchmarkScenarioAndRunPairLocally(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}

	service := NewService(manager)
	service.NowForTest(func() time.Time { return time.Date(2026, 3, 23, 9, 0, 0, 0, time.UTC) })
	if store, ok := service.benchmarkStore.(*benchmarks.Store); ok {
		store.NowForTest(func() time.Time { return time.Date(2026, 3, 23, 9, 0, 0, 0, time.UTC) })
	}

	scenario := benchmarks.Scenario{
		ScenarioID:            "delegated-subtask-001",
		TaskFamily:            "delegated-non-trivial-subtask",
		Title:                 "Implement workflow benchmark artifact storage",
		AcceptanceCriteriaRef: "TH3.E3.US1",
		Modes: []benchmarks.ScenarioMode{
			{
				Mode:       benchmarks.ModeWithGraph,
				TaskPrompt: "Implement the benchmark artifact writer using workflow context and local reports.",
			},
			{
				Mode:       benchmarks.ModeWithoutGraph,
				TaskPrompt: "Implement the benchmark artifact writer using only repository files and local notes.",
			},
		},
	}

	scenarioResult, err := service.StoreBenchmarkScenario(context.Background(), repoDir, scenario)
	if err != nil {
		t.Fatalf("StoreBenchmarkScenario returned error: %v", err)
	}

	withGraphRun, err := service.RecordBenchmarkRun(context.Background(), repoDir, benchmarks.RunReport{
		RunID:      "run-with-graph-001",
		ScenarioID: scenario.ScenarioID,
		Mode:       benchmarks.ModeWithGraph,
		Metrics: benchmarks.RunMetrics{
			Volume: benchmarks.VolumeMetrics{
				InputTokens:      1100,
				OutputTokens:     240,
				PromptCount:      3,
				PromptCharacters: 6800,
			},
			Orientation: benchmarks.OrientationMetrics{
				StepCount:       3,
				RepoScans:       2,
				FollowUpPrompts: 1,
			},
			Outcome: benchmarks.OutcomeSignals{
				QualityRating:          "pass",
				ResumabilityRating:     "strong",
				AcceptanceChecksPassed: 3,
				AcceptanceChecksTotal:  3,
			},
		},
	})
	if err != nil {
		t.Fatalf("RecordBenchmarkRun with graph returned error: %v", err)
	}

	withoutGraphRun, err := service.RecordBenchmarkRun(context.Background(), repoDir, benchmarks.RunReport{
		RunID:      "run-without-graph-001",
		ScenarioID: scenario.ScenarioID,
		Mode:       benchmarks.ModeWithoutGraph,
		Metrics: benchmarks.RunMetrics{
			Volume: benchmarks.VolumeMetrics{
				InputTokens:      1450,
				OutputTokens:     280,
				PromptCount:      5,
				PromptCharacters: 9100,
			},
			Orientation: benchmarks.OrientationMetrics{
				StepCount:       6,
				RepoScans:       5,
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
	})
	if err != nil {
		t.Fatalf("RecordBenchmarkRun without graph returned error: %v", err)
	}

	storedScenario := readBenchmarkScenarioArtifact(t, scenarioResult.Path)
	if storedScenario.ScenarioID != scenario.ScenarioID {
		t.Fatalf("stored scenario_id = %q, want %q", storedScenario.ScenarioID, scenario.ScenarioID)
	}
	if len(storedScenario.Modes) != 2 {
		t.Fatalf("stored scenario modes = %#v, want comparable with_graph and without_graph pair", storedScenario.Modes)
	}

	storedWithGraph := readBenchmarkRunArtifact(t, withGraphRun.Path)
	if storedWithGraph.ScenarioID != scenario.ScenarioID {
		t.Fatalf("with_graph scenario_id = %q, want %q", storedWithGraph.ScenarioID, scenario.ScenarioID)
	}
	if storedWithGraph.Mode != benchmarks.ModeWithGraph {
		t.Fatalf("with_graph mode = %q, want %q", storedWithGraph.Mode, benchmarks.ModeWithGraph)
	}
	if storedWithGraph.Metrics.Orientation.StepCount != 3 {
		t.Fatalf("with_graph orientation step_count = %d, want 3", storedWithGraph.Metrics.Orientation.StepCount)
	}

	storedWithoutGraph := readBenchmarkRunArtifact(t, withoutGraphRun.Path)
	if storedWithoutGraph.ScenarioID != scenario.ScenarioID {
		t.Fatalf("without_graph scenario_id = %q, want %q", storedWithoutGraph.ScenarioID, scenario.ScenarioID)
	}
	if storedWithoutGraph.Mode != benchmarks.ModeWithoutGraph {
		t.Fatalf("without_graph mode = %q, want %q", storedWithoutGraph.Mode, benchmarks.ModeWithoutGraph)
	}
	if storedWithoutGraph.Metrics.Volume.InputTokens <= storedWithGraph.Metrics.Volume.InputTokens {
		t.Fatalf("without_graph input_tokens = %d, want greater than with_graph %d for a comparable contrast", storedWithoutGraph.Metrics.Volume.InputTokens, storedWithGraph.Metrics.Volume.InputTokens)
	}
}

func TestServiceKeepsBenchmarkArtifactsLocalWithoutGraphPersistence(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}

	service := NewService(manager)
	if store, ok := service.benchmarkStore.(*benchmarks.Store); ok {
		store.NowForTest(func() time.Time { return time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC) })
	}

	_, err := service.StoreBenchmarkScenario(context.Background(), repoDir, benchmarks.Scenario{
		ScenarioID: "delegated-subtask-002",
		TaskFamily: "delegated-non-trivial-subtask",
		Modes: []benchmarks.ScenarioMode{
			{Mode: benchmarks.ModeWithGraph, TaskPrompt: "Use workflow start context."},
			{Mode: benchmarks.ModeWithoutGraph, TaskPrompt: "Use repository inspection only."},
		},
	})
	if err != nil {
		t.Fatalf("StoreBenchmarkScenario returned error: %v", err)
	}

	runResult, err := service.RecordBenchmarkRun(context.Background(), repoDir, benchmarks.RunReport{
		RunID:      "run-local-artifact-001",
		ScenarioID: "delegated-subtask-002",
		Mode:       benchmarks.ModeWithGraph,
		Metrics: benchmarks.RunMetrics{
			Volume: benchmarks.VolumeMetrics{
				PromptCount: 2,
			},
			Orientation: benchmarks.OrientationMetrics{
				StepCount: 2,
				RepoScans: 1,
			},
			Outcome: benchmarks.OutcomeSignals{
				QualityRating: "pass",
			},
		},
	})
	if err != nil {
		t.Fatalf("RecordBenchmarkRun returned error: %v", err)
	}

	if got, want := filepath.ToSlash(runResult.Path), ".graph/benchmarks/runs/delegated-subtask-002/with_graph/run-local-artifact-001.json"; filepath.Base(got) != filepath.Base(want) || !containsPathSuffix(got, want) {
		t.Fatalf("run artifact path = %q, want suffix %q", got, want)
	}

	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}

	currentRevision, err := kuzu.NewStore().CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision returned error: %v", err)
	}
	if currentRevision.Exists {
		t.Fatalf("current revision = %#v, want no graph revision for benchmark artifacts", currentRevision)
	}

	graph, err := kuzu.NewStore().ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph returned error: %v", err)
	}
	if len(graph.Nodes) != 0 || len(graph.Edges) != 0 {
		t.Fatalf("graph changed after benchmark recording: %#v", graph)
	}
}

func TestServiceRejectsBenchmarkScenarioWithoutComparableWorkflowModes(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}

	service := NewService(manager)

	_, err := service.StoreBenchmarkScenario(context.Background(), repoDir, benchmarks.Scenario{
		ScenarioID: "delegated-subtask-invalid",
		TaskFamily: "delegated-non-trivial-subtask",
		Modes: []benchmarks.ScenarioMode{
			{Mode: benchmarks.ModeWithGraph, TaskPrompt: "Use workflow context."},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	detail, ok := cmdsupport.ErrorDetailFromError(err)
	if !ok {
		t.Fatalf("expected structured error detail for %T", err)
	}
	if detail.Category != "validation_error" {
		t.Fatalf("error category = %q, want validation_error", detail.Category)
	}
	if detail.Code != "invalid_benchmark_scenario" {
		t.Fatalf("error code = %q, want invalid_benchmark_scenario", detail.Code)
	}
}

func TestServiceRejectsBenchmarkReportMissingRequiredComparisonMetrics(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}

	service := NewService(manager)
	if _, err := service.StoreBenchmarkScenario(context.Background(), repoDir, benchmarks.Scenario{
		ScenarioID: "delegated-subtask-003",
		TaskFamily: "delegated-non-trivial-subtask",
		Modes: []benchmarks.ScenarioMode{
			{Mode: benchmarks.ModeWithGraph, TaskPrompt: "Use workflow context."},
			{Mode: benchmarks.ModeWithoutGraph, TaskPrompt: "Use repo inspection only."},
		},
	}); err != nil {
		t.Fatalf("StoreBenchmarkScenario returned error: %v", err)
	}

	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}

	_, err = service.RecordBenchmarkRun(context.Background(), repoDir, benchmarks.RunReport{
		RunID:      "run-invalid-001",
		ScenarioID: "delegated-subtask-003",
		Mode:       benchmarks.ModeWithGraph,
		Metrics: benchmarks.RunMetrics{
			Volume: benchmarks.VolumeMetrics{
				InputTokens: 700,
			},
			Outcome: benchmarks.OutcomeSignals{
				QualityRating: "pass",
			},
		},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	detail, ok := cmdsupport.ErrorDetailFromError(err)
	if !ok {
		t.Fatalf("expected structured error detail for %T", err)
	}
	if detail.Category != "validation_error" {
		t.Fatalf("error category = %q, want validation_error", detail.Category)
	}
	if detail.Code != "invalid_benchmark_report" {
		t.Fatalf("error code = %q, want invalid_benchmark_report", detail.Code)
	}
	if got := detail.Details["field"]; got != "metrics.orientation" {
		t.Fatalf("error field = %#v, want metrics.orientation", got)
	}

	runPath := filepath.Join(workspace.WorkspacePath, repo.BenchmarksDirName, "runs", "delegated-subtask-003", benchmarks.ModeWithGraph, "run-invalid-001.json")
	if _, statErr := os.Stat(runPath); !os.IsNotExist(statErr) {
		t.Fatalf("invalid run artifact stat err = %v, want not exist", statErr)
	}
}

func TestServiceSummarizesComparableBenchmarkRuns(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}

	service := NewService(manager)
	store := benchmarks.NewStore()
	store.NowForTest(func() time.Time { return time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC) })
	service.BenchmarkStoreForTest(store)

	scenarioID := "delegated-subtask-summary-service-001"
	if _, err := service.StoreBenchmarkScenario(context.Background(), repoDir, benchmarks.Scenario{
		ScenarioID: scenarioID,
		TaskFamily: "delegated-non-trivial-subtask",
		Modes: []benchmarks.ScenarioMode{
			{Mode: benchmarks.ModeWithGraph, TaskPrompt: "Use workflow start context."},
			{Mode: benchmarks.ModeWithoutGraph, TaskPrompt: "Use repository inspection only."},
		},
	}); err != nil {
		t.Fatalf("StoreBenchmarkScenario returned error: %v", err)
	}
	for _, report := range []benchmarks.RunReport{
		{
			RunID:      "run-with-graph-service-001",
			ScenarioID: scenarioID,
			Mode:       benchmarks.ModeWithGraph,
			Metrics: benchmarks.RunMetrics{
				Volume: benchmarks.VolumeMetrics{
					InputTokens:      1020,
					OutputTokens:     220,
					PromptCount:      3,
					PromptCharacters: 6200,
				},
				Orientation: benchmarks.OrientationMetrics{
					StepCount:       3,
					RepoScans:       2,
					FollowUpPrompts: 1,
				},
				Outcome: benchmarks.OutcomeSignals{
					QualityRating:          "pass",
					ResumabilityRating:     "strong",
					AcceptanceChecksPassed: 3,
					AcceptanceChecksTotal:  3,
				},
			},
		},
		{
			RunID:      "run-without-graph-service-001",
			ScenarioID: scenarioID,
			Mode:       benchmarks.ModeWithoutGraph,
			Metrics: benchmarks.RunMetrics{
				Volume: benchmarks.VolumeMetrics{
					InputTokens:      1340,
					OutputTokens:     260,
					PromptCount:      5,
					PromptCharacters: 8700,
				},
				Orientation: benchmarks.OrientationMetrics{
					StepCount:       5,
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
		},
	} {
		if _, err := service.RecordBenchmarkRun(context.Background(), repoDir, report); err != nil {
			t.Fatalf("RecordBenchmarkRun(%s) returned error: %v", report.Mode, err)
		}
	}

	result, err := service.SummarizeBenchmark(context.Background(), repoDir, scenarioID)
	if err != nil {
		t.Fatalf("SummarizeBenchmark returned error: %v", err)
	}

	if result.Count != 1 {
		t.Fatalf("summary count = %d, want 1", result.Count)
	}
	summary := result.Summaries[0]
	if summary.Status != BenchmarkSummaryStatusComparable || !summary.Comparable {
		t.Fatalf("summary = %#v, want comparable", summary)
	}
	if summary.Comparison == nil {
		t.Fatal("expected comparison payload")
	}
	if got := summary.Comparison.Volume.InputTokens.DeltaVsWithoutGraph; got != -320 {
		t.Fatalf("input token delta = %d, want -320", got)
	}
	if got := summary.Comparison.Orientation.StepCount.DeltaVsWithoutGraph; got != -2 {
		t.Fatalf("step count delta = %d, want -2", got)
	}
	if !summary.Comparison.Outcome.QualityRating.Matches {
		t.Fatalf("quality rating comparison = %#v, want matching pass ratings", summary.Comparison.Outcome.QualityRating)
	}
}

func TestServiceFlagsNonComparableBenchmarkRunsWhenModeHasDuplicates(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}

	service := NewService(manager)
	store := benchmarks.NewStore()
	store.NowForTest(func() time.Time { return time.Date(2026, 3, 24, 13, 0, 0, 0, time.UTC) })
	service.BenchmarkStoreForTest(store)

	scenarioID := "delegated-subtask-summary-service-002"
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

	reports := []benchmarks.RunReport{
		{
			RunID:      "run-with-graph-a",
			ScenarioID: scenarioID,
			Mode:       benchmarks.ModeWithGraph,
			Metrics: benchmarks.RunMetrics{
				Volume:      benchmarks.VolumeMetrics{InputTokens: 900, OutputTokens: 200, PromptCount: 3, PromptCharacters: 5800},
				Orientation: benchmarks.OrientationMetrics{StepCount: 3},
				Outcome:     benchmarks.OutcomeSignals{QualityRating: "pass", ResumabilityRating: "strong", AcceptanceChecksPassed: 2, AcceptanceChecksTotal: 2},
			},
		},
		{
			RunID:      "run-with-graph-b",
			ScenarioID: scenarioID,
			Mode:       benchmarks.ModeWithGraph,
			Metrics: benchmarks.RunMetrics{
				Volume:      benchmarks.VolumeMetrics{InputTokens: 910, OutputTokens: 205, PromptCount: 3, PromptCharacters: 5900},
				Orientation: benchmarks.OrientationMetrics{StepCount: 4},
				Outcome:     benchmarks.OutcomeSignals{QualityRating: "pass", ResumabilityRating: "strong", AcceptanceChecksPassed: 2, AcceptanceChecksTotal: 2},
			},
		},
		{
			RunID:      "run-without-graph-a",
			ScenarioID: scenarioID,
			Mode:       benchmarks.ModeWithoutGraph,
			Metrics: benchmarks.RunMetrics{
				Volume:      benchmarks.VolumeMetrics{InputTokens: 1300, OutputTokens: 260, PromptCount: 5, PromptCharacters: 8400},
				Orientation: benchmarks.OrientationMetrics{StepCount: 5},
				Outcome:     benchmarks.OutcomeSignals{QualityRating: "pass", ResumabilityRating: "partial", AcceptanceChecksPassed: 2, AcceptanceChecksTotal: 2},
			},
		},
	}
	for _, report := range reports {
		if _, err := service.RecordBenchmarkRun(context.Background(), repoDir, report); err != nil {
			t.Fatalf("RecordBenchmarkRun(%s) returned error: %v", report.RunID, err)
		}
	}

	result, err := service.SummarizeBenchmark(context.Background(), repoDir, scenarioID)
	if err != nil {
		t.Fatalf("SummarizeBenchmark returned error: %v", err)
	}

	summary := result.Summaries[0]
	if summary.Status != BenchmarkSummaryStatusNonComparable || summary.Comparable {
		t.Fatalf("summary = %#v, want non_comparable", summary)
	}
	if summary.Comparison != nil {
		t.Fatalf("comparison = %#v, want nil for non-comparable data", summary.Comparison)
	}
	if len(summary.Issues) == 0 || summary.Issues[0].Code != "multiple_mode_runs" {
		t.Fatalf("issues = %#v, want multiple_mode_runs", summary.Issues)
	}
}

func readBenchmarkScenarioArtifact(t *testing.T, path string) benchmarks.Scenario {
	t.Helper()

	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile scenario artifact: %v", err)
	}

	var scenario benchmarks.Scenario
	if err := json.Unmarshal(payload, &scenario); err != nil {
		t.Fatalf("json.Unmarshal scenario artifact: %v\npayload: %s", err, payload)
	}
	return scenario
}

func readBenchmarkRunArtifact(t *testing.T, path string) benchmarks.RunReport {
	t.Helper()

	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile run artifact: %v", err)
	}

	var report benchmarks.RunReport
	if err := json.Unmarshal(payload, &report); err != nil {
		t.Fatalf("json.Unmarshal run artifact: %v\npayload: %s", err, payload)
	}
	return report
}

func containsPathSuffix(path, suffix string) bool {
	return len(path) >= len(suffix) && path[len(path)-len(suffix):] == suffix
}
