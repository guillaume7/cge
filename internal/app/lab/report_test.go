package lab

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/guillaume-galp/cge/internal/app/workflow"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func TestServiceReportAggregatesPairedComparisonsGroupedEffectsAndExplicitLimitations(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeReportSuiteFixture(t, repoDir)
	writeReportBatchPlanFixture(t, repoDir)

	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-model-a-graph-1",
		TaskID:          "task-001",
		ConditionID:     "with-graph",
		Model:           "model-a",
		SessionTopology: "delegated-parallel",
		Seed:            1,
		TotalTokens:     100,
		WallClock:       10,
		Success:         true,
		Quality:         0.90,
		Resumability:    0.80,
	})
	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-model-a-base-1",
		TaskID:          "task-001",
		ConditionID:     "without-graph",
		Model:           "model-a",
		SessionTopology: "delegated-parallel",
		Seed:            1,
		TotalTokens:     150,
		WallClock:       20,
		Success:         true,
		Quality:         0.70,
		Resumability:    0.60,
	})
	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-model-a-graph-2",
		TaskID:          "task-001",
		ConditionID:     "with-graph",
		Model:           "model-a",
		SessionTopology: "delegated-parallel",
		Seed:            2,
		TotalTokens:     110,
		WallClock:       11,
		Success:         true,
		Quality:         0.88,
		Resumability:    0.78,
	})
	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-model-a-base-2",
		TaskID:          "task-001",
		ConditionID:     "without-graph",
		Model:           "model-a",
		SessionTopology: "delegated-parallel",
		Seed:            2,
		TotalTokens:     160,
		WallClock:       22,
		Success:         true,
		Quality:         0.68,
		Resumability:    0.58,
	})
	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-model-b-graph-1",
		TaskID:          "task-001",
		ConditionID:     "with-graph",
		Model:           "model-b",
		SessionTopology: "delegated-parallel",
		Seed:            1,
		TotalTokens:     210,
		WallClock:       26,
		Success:         true,
		Quality:         0.50,
		Resumability:    0.45,
	})
	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-model-b-base-1",
		TaskID:          "task-001",
		ConditionID:     "without-graph",
		Model:           "model-b",
		SessionTopology: "delegated-parallel",
		Seed:            1,
		TotalTokens:     180,
		WallClock:       18,
		Success:         true,
		Quality:         0.60,
		Resumability:    0.50,
	})
	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-model-b-graph-2",
		TaskID:          "task-001",
		ConditionID:     "with-graph",
		Model:           "model-b",
		SessionTopology: "delegated-parallel",
		Seed:            2,
		TotalTokens:     220,
		WallClock:       27,
		Success:         false,
		Quality:         0.48,
		Resumability:    0.43,
	})
	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-model-b-base-2",
		TaskID:          "task-001",
		ConditionID:     "without-graph",
		Model:           "model-b",
		SessionTopology: "delegated-parallel",
		Seed:            2,
		TotalTokens:     185,
		WallClock:       19,
		Success:         true,
		Quality:         0.62,
		Resumability:    0.55,
	})
	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-unscored",
		TaskID:          "task-002",
		ConditionID:     "with-graph",
		Model:           "model-a",
		SessionTopology: "delegated-parallel",
		Seed:            99,
		TotalTokens:     130,
		WallClock:       13,
		WithoutEval:     true,
	})

	service.NowForTest(func() time.Time {
		return time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	})

	result, err := service.Report(context.Background(), repoDir, ReportRequest{})
	if err != nil {
		t.Fatalf("Report returned error: %v", err)
	}

	if result.ReportID != "report-20260401t120000z" {
		t.Fatalf("report_id = %q, want report-20260401t120000z", result.ReportID)
	}
	if got, want := result.ArtifactPath, ".graph/lab/reports/report-20260401t120000z.json"; got != want {
		t.Fatalf("artifact_path = %q, want %q", got, want)
	}
	if result.Report.SourceArtifacts.RunRecordCount != 9 {
		t.Fatalf("run_record_count = %d, want 9", result.Report.SourceArtifacts.RunRecordCount)
	}
	if len(result.Report.SourceArtifacts.BatchPlans) != 1 {
		t.Fatalf("batch_plans = %#v, want one plan", result.Report.SourceArtifacts.BatchPlans)
	}
	if result.Report.RunsUnscored != 1 || len(result.Report.UnscoredRunIDs) != 1 || result.Report.UnscoredRunIDs[0] != "run-unscored" {
		t.Fatalf("unscored runs = %#v, want [run-unscored]", result.Report.UnscoredRunIDs)
	}

	modelA := findPairedComparison(t, result.Report.PairedComparisons, "task-001", "model-a")
	if modelA.PairCount != 2 || modelA.ScoredPairCount != 2 {
		t.Fatalf("model-a pair counts = %#v, want 2 scored pairs", modelA)
	}
	if got := modelA.Metrics["quality_score"].MeanDelta; got <= 0 {
		t.Fatalf("quality_score mean_delta = %v, want positive graph benefit", got)
	}
	if got := modelA.Metrics["total_tokens"].MeanDelta; got >= 0 {
		t.Fatalf("total_tokens mean_delta = %v, want negative token delta for graph benefit", got)
	}
	if len(modelA.Metrics["quality_score"].Uncertainty.Interval95) != 2 {
		t.Fatalf("quality uncertainty = %#v, want explicit interval", modelA.Metrics["quality_score"].Uncertainty)
	}

	modelB := findPairedComparison(t, result.Report.PairedComparisons, "task-001", "model-b")
	if got := modelB.Metrics["quality_score"].MeanDelta; got >= 0 {
		t.Fatalf("model-b quality_score mean_delta = %v, want negative result", got)
	}

	groupedByModel := findGroupedComparison(t, result.Report.GroupedComparisons, groupingModel, "delegated-non-trivial-subtask")
	if groupedByModel.GroupCount != 2 {
		t.Fatalf("group_count = %d, want 2 model groups", groupedByModel.GroupCount)
	}
	if len(groupedByModel.VariationNotes) == 0 {
		t.Fatalf("variation_notes = %#v, want explicit cross-model variation", groupedByModel.VariationNotes)
	}

	if !reportContainsWarning(result.Report.Summary.Warnings, "missing_evaluations") {
		t.Fatalf("warnings = %#v, want missing_evaluations warning", result.Report.Summary.Warnings)
	}
	if !reportContainsFinding(result.Report.Summary.NegativeResults, "quality_score", "model-b") {
		t.Fatalf("negative_results = %#v, want model-b quality_score finding", result.Report.Summary.NegativeResults)
	}
	if len(result.Report.Limitations) == 0 || !strings.Contains(strings.Join(result.Report.Limitations, " "), "unscored") {
		t.Fatalf("limitations = %#v, want explicit unscored/sample-size limitation", result.Report.Limitations)
	}
}

func TestServiceReportSupportsFocusedRunSelection(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeReportSuiteFixture(t, repoDir)
	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-graph",
		TaskID:          "task-001",
		ConditionID:     "with-graph",
		Model:           "model-a",
		SessionTopology: "delegated-parallel",
		Seed:            1,
		TotalTokens:     100,
		WallClock:       10,
		Success:         true,
		Quality:         0.90,
		Resumability:    0.80,
	})
	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-base",
		TaskID:          "task-001",
		ConditionID:     "without-graph",
		Model:           "model-a",
		SessionTopology: "delegated-parallel",
		Seed:            1,
		TotalTokens:     150,
		WallClock:       20,
		Success:         true,
		Quality:         0.70,
		Resumability:    0.60,
	})

	result, err := service.Report(context.Background(), repoDir, ReportRequest{RunIDs: []string{"run-base", "run-graph"}})
	if err != nil {
		t.Fatalf("Report returned error: %v", err)
	}

	if result.Report.Selection.Mode != reportSelectionSelectedRuns {
		t.Fatalf("selection.mode = %q, want %q", result.Report.Selection.Mode, reportSelectionSelectedRuns)
	}
	if result.Report.RunsIncluded != 2 {
		t.Fatalf("runs_included = %d, want 2", result.Report.RunsIncluded)
	}
	if len(result.Report.PairedComparisons) != 1 {
		t.Fatalf("paired comparisons = %#v, want one focused comparison", result.Report.PairedComparisons)
	}
	if got := result.Report.PairedComparisons[0].Metrics["quality_score"].Uncertainty.Method; got != "not_available" {
		t.Fatalf("uncertainty.method = %q, want not_available for a single scored pair", got)
	}
	if len(result.Report.PairedComparisons[0].Limitations) == 0 {
		t.Fatalf("pair limitations = %#v, want explicit sample-size limitation", result.Report.PairedComparisons[0].Limitations)
	}
}

func TestServiceReportBuildsVerificationAttributionAndGate(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeReportSuiteFixture(t, repoDir)
	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "verification-graph-1",
		TaskID:          "task-001",
		ConditionID:     "with-graph",
		Model:           "model-a",
		SessionTopology: "delegated-parallel",
		Seed:            1,
		TotalTokens:     100,
		WallClock:       10,
		Success:         true,
		Quality:         0.90,
		Resumability:    0.80,
		WorkflowStart: &workflow.StartResult{
			Kickoff: workflow.KickoffEnvelope{
				Task: workflow.KickoffTaskDetails{
					Family:     workflow.KickoffFamilyVerificationAudit,
					Subprofile: workflow.KickoffVerificationProfileWorkflow,
				},
				Advisory: workflow.KickoffAdvisoryState{
					EffectiveMode:   workflow.KickoffModeMinimal,
					ConfidenceScore: 0.76,
				},
			},
		},
	})
	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:                  "verification-base-1",
		TaskID:                 "task-001",
		ConditionID:            "without-graph",
		Model:                  "model-a",
		SessionTopology:        "delegated-parallel",
		Seed:                   1,
		TotalTokens:            120,
		WallClock:              12,
		Success:                true,
		Quality:                0.88,
		Resumability:           0.78,
		BaselinePromptMetadata: map[string]any{"task_word_count": 4, "graph_backed": false},
	})

	result, err := service.Report(context.Background(), repoDir, ReportRequest{})
	if err != nil {
		t.Fatalf("Report returned error: %v", err)
	}

	if len(result.Report.VerificationAttribution.Profiles) != 1 {
		t.Fatalf("verification profiles = %#v, want one profile summary", result.Report.VerificationAttribution.Profiles)
	}
	profile := result.Report.VerificationAttribution.Profiles[0]
	if profile.Profile != workflow.KickoffVerificationProfileWorkflow {
		t.Fatalf("profile = %q, want workflow verification", profile.Profile)
	}
	if profile.BaselinePromptMetadataPairs != 1 {
		t.Fatalf("baseline_prompt_metadata_pairs = %d, want 1", profile.BaselinePromptMetadataPairs)
	}
	if result.Report.VerificationAttribution.Gate.Decision == "" {
		t.Fatalf("verification gate = %#v, want explicit decision", result.Report.VerificationAttribution.Gate)
	}
}

type reportFixtureRun struct {
	RunID                  string
	TaskID                 string
	ConditionID            string
	Model                  string
	SessionTopology        string
	Seed                   int64
	TotalTokens            int
	WallClock              int
	Success                bool
	Quality                float64
	Resumability           float64
	WithoutEval            bool
	IncompleteToken        bool
	WorkflowStart          *workflow.StartResult
	BaselinePromptMetadata map[string]any
	AttributionSummary     *RunAttributionSummary
}

func writeReportSuiteFixture(t *testing.T, repoDir string) {
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

func writeReportBatchPlanFixture(t *testing.T, repoDir string) {
	t.Helper()

	writeJSONFixture(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "runs", "batches", "batch-20260401t103000z", "plan.json"), BatchPlanArtifact{
		SchemaVersion:   SchemaVersion,
		BatchID:         "batch-20260401t103000z",
		PlannedAt:       "2026-04-01T10:30:00Z",
		Model:           "model-a",
		SessionTopology: "delegated-parallel",
		Seed:            42,
		Randomized:      true,
		TaskIDs:         []string{"task-001"},
		ConditionIDs:    []string{"with-graph", "without-graph"},
		Entries: []BatchPlanEntry{
			{Order: 1, TaskID: "task-001", ConditionID: "with-graph"},
			{Order: 2, TaskID: "task-001", ConditionID: "without-graph"},
		},
	})
}

func writeReportRunFixture(t *testing.T, repoDir string, fixture reportFixtureRun) {
	t.Helper()

	startedAt := "2026-04-01T10:30:00Z"
	finishedAt := "2026-04-01T10:45:00Z"
	record := RunRecord{
		SchemaVersion:   SchemaVersion,
		RunID:           fixture.RunID,
		TaskID:          fixture.TaskID,
		ConditionID:     fixture.ConditionID,
		Model:           fixture.Model,
		SessionTopology: fixture.SessionTopology,
		Seed:            &fixture.Seed,
		PromptVariant:   "default",
		StartedAt:       startedAt,
		FinishedAt:      finishedAt,
		Telemetry: &RunTelemetry{
			MeasurementStatus: "complete",
			Source:            "workflow_finish_payload",
			Provider:          "copilot-cli",
			TotalTokens:       intPointer(fixture.TotalTokens),
			InputTokens:       intPointer(fixture.TotalTokens / 2),
			OutputTokens:      intPointer(fixture.TotalTokens / 2),
			WallClockSeconds:  intPointer(fixture.WallClock),
			RetryCount:        intPointer(0),
			DelegatedSessions: intPointer(1),
		},
		KickoffInputsRef:    "artifacts/task-input.json",
		SessionStructureRef: "artifacts/sessions/",
		WritebackOutputsRef: "artifacts/task-output.json",
		OutcomeArtifactsRef: "artifacts/output/" + fixture.ConditionID + "/",
	}
	if fixture.WorkflowStart != nil {
		record.WorkflowStartResponseRef = "artifacts/workflow-start-response.json"
		writeJSONFixture(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "runs", fixture.RunID, record.WorkflowStartResponseRef), fixture.WorkflowStart)
	}
	if fixture.BaselinePromptMetadata != nil {
		record.BaselinePromptMetadataRef = "artifacts/baseline-prompt-metadata.json"
		writeJSONFixture(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "runs", fixture.RunID, record.BaselinePromptMetadataRef), fixture.BaselinePromptMetadata)
	}
	if fixture.AttributionSummary != nil {
		record.AttributionSummaryRef = "artifacts/attribution-summary.json"
		writeJSONFixture(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "runs", fixture.RunID, record.AttributionSummaryRef), fixture.AttributionSummary)
	}
	if fixture.IncompleteToken {
		record.Telemetry.MeasurementStatus = "partial"
		record.Telemetry.TotalTokens = nil
		record.Telemetry.InputTokens = nil
		record.Telemetry.OutputTokens = nil
		record.Telemetry.IncompleteReasons = []string{"token_measurement_incomplete"}
	}
	writeJSONFixture(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "runs", fixture.RunID, "run.json"), record)

	if fixture.WithoutEval {
		return
	}

	writeJSONFixture(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, "evaluations", fixture.RunID+".json"), EvaluationArtifact{
		SchemaVersion: SchemaVersion,
		RunID:         fixture.RunID,
		Records: []EvaluationRecord{
			{
				SchemaVersion: SchemaVersion,
				RunID:         fixture.RunID,
				Evaluator:     "human:alice",
				EvaluatedAt:   "2026-04-01T11:00:00Z",
				Scores: &EvaluationScores{
					Success:                boolPointer(fixture.Success),
					QualityScore:           &fixture.Quality,
					ResumabilityScore:      &fixture.Resumability,
					HumanInterventionCount: intPointer(0),
				},
			},
		},
	})
}

func findPairedComparison(t *testing.T, comparisons []PairedComparison, taskID, model string) PairedComparison {
	t.Helper()

	for _, comparison := range comparisons {
		if comparison.TaskID == taskID && comparison.Model == model {
			return comparison
		}
	}
	t.Fatalf("paired comparison for task=%s model=%s not found in %#v", taskID, model, comparisons)
	return PairedComparison{}
}

func findGroupedComparison(t *testing.T, comparisons []GroupedComparison, grouping, taskFamily string) GroupedComparison {
	t.Helper()

	for _, comparison := range comparisons {
		if comparison.Grouping == grouping && comparison.TaskFamily == taskFamily {
			return comparison
		}
	}
	t.Fatalf("grouped comparison for grouping=%s task_family=%s not found in %#v", grouping, taskFamily, comparisons)
	return GroupedComparison{}
}

func reportContainsWarning(warnings []ReportWarning, code string) bool {
	for _, warning := range warnings {
		if warning.Code == code {
			return true
		}
	}
	return false
}

func reportContainsFinding(findings []ReportFinding, metric, groupValue string) bool {
	for _, finding := range findings {
		if finding.Metric == metric && (finding.Model == groupValue || finding.GroupValue == groupValue) {
			return true
		}
	}
	return false
}

func TestServiceReportSurfacesHarnessAwareTokenDeclineComparisons(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeReportSuiteFixture(t, repoDir)

	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-harness-1",
		TaskID:          "task-001",
		ConditionID:     "with-harness",
		Model:           "model-a",
		SessionTopology: "delegated-parallel",
		Seed:            42,
		TotalTokens:     500,
		WallClock:       60,
		Success:         true,
		Quality:         0.8,
		Resumability:    0.7,
	})
	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-without-harness-1",
		TaskID:          "task-001",
		ConditionID:     "without-harness",
		Model:           "model-a",
		SessionTopology: "delegated-parallel",
		Seed:            42,
		TotalTokens:     1000,
		WallClock:       90,
		Success:         true,
		Quality:         0.8,
		Resumability:    0.7,
	})

	service.NowForTest(func() time.Time { return time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC) })
	result, err := service.Report(context.Background(), repoDir, ReportRequest{})
	if err != nil {
		t.Fatalf("Report returned error: %v", err)
	}

	m := result.Report.HarnessAwareMetrics
	if !m.HasHarnessData {
		t.Fatalf("harness_aware_metrics.has_harness_data = false, want true")
	}
	if m.TokenDecline.RunsWithHarness != 1 {
		t.Fatalf("runs_with_harness = %d, want 1", m.TokenDecline.RunsWithHarness)
	}
	if m.TokenDecline.RunsWithoutHarness != 1 {
		t.Fatalf("runs_without_harness = %d, want 1", m.TokenDecline.RunsWithoutHarness)
	}
	if m.TokenDecline.PrimaryComparison == nil {
		t.Fatal("primary_comparison = nil, want token comparison between with-harness and without-harness")
	}
	if m.TokenDecline.PrimaryComparison.ConditionA != "with-harness" || m.TokenDecline.PrimaryComparison.ConditionB != "without-harness" {
		t.Fatalf("primary_comparison conditions = %q/%q, want with-harness/without-harness",
			m.TokenDecline.PrimaryComparison.ConditionA, m.TokenDecline.PrimaryComparison.ConditionB)
	}
	if m.TokenDecline.PrimaryComparison.SampleSize != 1 {
		t.Fatalf("primary_comparison.sample_size = %d, want 1", m.TokenDecline.PrimaryComparison.SampleSize)
	}
	// delta = without - with = 1000 - 500 = 500 (positive = with-harness uses fewer tokens)
	if m.TokenDecline.PrimaryComparison.MeanAbsoluteDelta != 500 {
		t.Fatalf("mean_absolute_delta = %v, want 500", m.TokenDecline.PrimaryComparison.MeanAbsoluteDelta)
	}
	if len(m.TokenDecline.PerTaskDistributions) == 0 {
		t.Fatal("per_task_distributions is empty, want at least one entry")
	}
}

func TestServiceReportDetectsOverAbstentionWhenTokenSavingsCoincideWithQualityRegression(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeReportSuiteFixture(t, repoDir)

	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-harness-abstain-1",
		TaskID:          "task-001",
		ConditionID:     "with-harness",
		Model:           "model-a",
		SessionTopology: "delegated-parallel",
		Seed:            42,
		TotalTokens:     400, // fewer tokens than without-harness
		WallClock:       60,
		Success:         false,
		Quality:         0.3, // lower quality → quality regression
		Resumability:    0.3,
		AttributionSummary: &RunAttributionSummary{
			SchemaVersion:  SchemaVersion,
			RunID:          "run-harness-abstain-1",
			OutcomeCounts:  map[string]int{"abstain": 3, "proceed": 5},
			TotalDecisions: 8,
		},
	})
	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-without-harness-qa-1",
		TaskID:          "task-001",
		ConditionID:     "without-harness",
		Model:           "model-a",
		SessionTopology: "delegated-parallel",
		Seed:            42,
		TotalTokens:     1000, // more tokens
		WallClock:       90,
		Success:         true,
		Quality:         0.9, // higher quality
		Resumability:    0.8,
	})

	service.NowForTest(func() time.Time { return time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC) })
	result, err := service.Report(context.Background(), repoDir, ReportRequest{})
	if err != nil {
		t.Fatalf("Report returned error: %v", err)
	}

	oa := result.Report.HarnessAwareMetrics.OverAbstentionAnalysis
	if !oa.TokenSavingsPresent {
		t.Fatal("token_savings_present = false, want true when harness tokens < without-harness tokens")
	}
	if !oa.QualityRegressionPresent {
		t.Fatal("quality_regression_present = false, want true when harness quality < without-harness quality")
	}
	if !oa.OverAbstentionWarning {
		t.Fatal("over_abstention_warning = false, want true when both token savings and quality regression present")
	}
	if oa.AbstentionCount != 3 {
		t.Fatalf("abstention_count = %d, want 3", oa.AbstentionCount)
	}
}

func TestServiceReportNoOverAbstentionFlagWhenQualityHolds(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeReportSuiteFixture(t, repoDir)

	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-harness-good-1",
		TaskID:          "task-001",
		ConditionID:     "with-harness",
		Model:           "model-a",
		SessionTopology: "delegated-parallel",
		Seed:            42,
		TotalTokens:     500, // fewer tokens
		WallClock:       60,
		Success:         true,
		Quality:         0.9, // better quality — no regression
		Resumability:    0.8,
	})
	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-without-harness-good-1",
		TaskID:          "task-001",
		ConditionID:     "without-harness",
		Model:           "model-a",
		SessionTopology: "delegated-parallel",
		Seed:            42,
		TotalTokens:     1000, // more tokens
		WallClock:       90,
		Success:         true,
		Quality:         0.7, // lower quality
		Resumability:    0.6,
	})

	service.NowForTest(func() time.Time { return time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC) })
	result, err := service.Report(context.Background(), repoDir, ReportRequest{})
	if err != nil {
		t.Fatalf("Report returned error: %v", err)
	}

	oa := result.Report.HarnessAwareMetrics.OverAbstentionAnalysis
	if !oa.TokenSavingsPresent {
		t.Fatal("token_savings_present = false, want true")
	}
	if oa.QualityRegressionPresent {
		t.Fatal("quality_regression_present = true, want false when harness quality >= without-harness quality")
	}
	if oa.OverAbstentionWarning {
		t.Fatal("over_abstention_warning = true, want false when quality does not regress")
	}
}

func TestServiceReportAggregatesAttributionOutcomeDistribution(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	writeReportSuiteFixture(t, repoDir)

	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-attr-1",
		TaskID:          "task-001",
		ConditionID:     "with-harness",
		Model:           "model-a",
		SessionTopology: "delegated-parallel",
		Seed:            42,
		TotalTokens:     600,
		WallClock:       60,
		Success:         true,
		Quality:         0.8,
		Resumability:    0.7,
		AttributionSummary: &RunAttributionSummary{
			SchemaVersion:  SchemaVersion,
			RunID:          "run-attr-1",
			OutcomeCounts:  map[string]int{"proceed": 7, "abstain": 2, "defer": 1},
			TotalDecisions: 10,
		},
	})
	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-attr-2",
		TaskID:          "task-002",
		ConditionID:     "with-harness",
		Model:           "model-a",
		SessionTopology: "delegated-parallel",
		Seed:            42,
		TotalTokens:     700,
		WallClock:       65,
		Success:         true,
		Quality:         0.75,
		Resumability:    0.65,
		AttributionSummary: &RunAttributionSummary{
			SchemaVersion:  SchemaVersion,
			RunID:          "run-attr-2",
			OutcomeCounts:  map[string]int{"proceed": 5, "abstain": 3},
			TotalDecisions: 8,
		},
	})
	// One harness run without attribution (should be counted in TotalRunsWithoutAttribution).
	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-attr-3",
		TaskID:          "task-001",
		ConditionID:     "with-harness",
		Model:           "model-b",
		SessionTopology: "delegated-parallel",
		Seed:            99,
		TotalTokens:     550,
		WallClock:       55,
		Success:         true,
		Quality:         0.78,
		Resumability:    0.68,
	})

	service.NowForTest(func() time.Time { return time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC) })
	result, err := service.Report(context.Background(), repoDir, ReportRequest{})
	if err != nil {
		t.Fatalf("Report returned error: %v", err)
	}

	agg := result.Report.HarnessAwareMetrics.AttributionAggregation
	if agg.TotalRunsWithAttribution != 2 {
		t.Fatalf("total_runs_with_attribution = %d, want 2", agg.TotalRunsWithAttribution)
	}
	if agg.TotalRunsWithoutAttribution != 1 {
		t.Fatalf("total_runs_without_attribution = %d, want 1", agg.TotalRunsWithoutAttribution)
	}
	if agg.OverallOutcomeDistribution == nil {
		t.Fatal("overall_outcome_distribution = nil, want aggregated outcome counts")
	}
	// proceed: 7 + 5 = 12, abstain: 2 + 3 = 5, defer: 1
	if got := agg.OverallOutcomeDistribution["proceed"].Count; got != 12 {
		t.Fatalf("overall proceed count = %d, want 12", got)
	}
	if got := agg.OverallOutcomeDistribution["abstain"].Count; got != 5 {
		t.Fatalf("overall abstain count = %d, want 5", got)
	}
}

func TestServiceReportPerFamilyDecisionPatterns(t *testing.T) {
	t.Parallel()

	repoDir, manager := initLabRepo(t)
	service := NewService(manager)
	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	// Custom suite with two different task families.
	writeJSONFixture(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.LabDirName, repo.LabSuiteManifestName), SuiteManifest{
		SchemaVersion: SchemaVersion,
		SuiteID:       "delegated-workflow-evidence-v1",
		Tasks: []SuiteTask{
			{
				TaskID:                "task-fam-a",
				Family:                "family-alpha",
				Description:           "alpha task",
				AcceptanceCriteriaRef: "tasks/task-fam-a/criteria.md",
			},
			{
				TaskID:                "task-fam-b",
				Family:                "family-beta",
				Description:           "beta task",
				AcceptanceCriteriaRef: "tasks/task-fam-b/criteria.md",
			},
		},
	})

	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-fam-alpha-1",
		TaskID:          "task-fam-a",
		ConditionID:     "with-harness",
		Model:           "model-a",
		SessionTopology: "delegated-parallel",
		Seed:            1,
		TotalTokens:     500,
		WallClock:       60,
		Success:         true,
		Quality:         0.8,
		Resumability:    0.7,
		AttributionSummary: &RunAttributionSummary{
			SchemaVersion:  SchemaVersion,
			RunID:          "run-fam-alpha-1",
			OutcomeCounts:  map[string]int{"proceed": 8, "abstain": 2},
			TotalDecisions: 10,
		},
	})
	writeReportRunFixture(t, repoDir, reportFixtureRun{
		RunID:           "run-fam-beta-1",
		TaskID:          "task-fam-b",
		ConditionID:     "with-harness",
		Model:           "model-a",
		SessionTopology: "delegated-parallel",
		Seed:            2,
		TotalTokens:     600,
		WallClock:       70,
		Success:         true,
		Quality:         0.75,
		Resumability:    0.65,
		AttributionSummary: &RunAttributionSummary{
			SchemaVersion:  SchemaVersion,
			RunID:          "run-fam-beta-1",
			OutcomeCounts:  map[string]int{"proceed": 3, "abstain": 7},
			TotalDecisions: 10,
		},
	})

	service.NowForTest(func() time.Time { return time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC) })
	result, err := service.Report(context.Background(), repoDir, ReportRequest{})
	if err != nil {
		t.Fatalf("Report returned error: %v", err)
	}

	perFamily := result.Report.HarnessAwareMetrics.AttributionAggregation.PerFamilyOutcomes
	if len(perFamily) != 2 {
		t.Fatalf("per_family_outcomes count = %d, want 2 (one per family)", len(perFamily))
	}
	// Sorted by family name: family-alpha, family-beta.
	if perFamily[0].TaskFamily != "family-alpha" {
		t.Fatalf("per_family_outcomes[0].task_family = %q, want family-alpha", perFamily[0].TaskFamily)
	}
	if perFamily[1].TaskFamily != "family-beta" {
		t.Fatalf("per_family_outcomes[1].task_family = %q, want family-beta", perFamily[1].TaskFamily)
	}
	if got := perFamily[0].OutcomeDistribution["abstain"].Count; got != 2 {
		t.Fatalf("family-alpha abstain count = %d, want 2", got)
	}
	if got := perFamily[1].OutcomeDistribution["abstain"].Count; got != 7 {
		t.Fatalf("family-beta abstain count = %d, want 7", got)
	}
}
