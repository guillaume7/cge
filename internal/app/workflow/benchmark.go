package workflow

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/infra/benchmarks"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

type BenchmarkStore interface {
	WriteScenario(workspace repo.Workspace, scenario benchmarks.Scenario) (benchmarks.ScenarioArtifact, error)
	ReadScenario(workspace repo.Workspace, scenarioID string) (benchmarks.ScenarioArtifact, error)
	WriteRun(workspace repo.Workspace, scenario benchmarks.Scenario, report benchmarks.RunReport) (benchmarks.RunArtifact, error)
	LoadScenario(workspace repo.Workspace, scenarioID string) (benchmarks.ScenarioArtifact, error)
	ListScenarios(workspace repo.Workspace) ([]benchmarks.ScenarioArtifact, error)
	ListRuns(workspace repo.Workspace, scenarioID string) ([]benchmarks.RunArtifact, error)
}

type BenchmarkScenarioResult struct {
	Path     string              `json:"path"`
	Scenario benchmarks.Scenario `json:"scenario"`
}

type BenchmarkRunResult struct {
	Path   string               `json:"path"`
	Report benchmarks.RunReport `json:"report"`
}

const (
	BenchmarkSummaryStatusComparable    = "comparable"
	BenchmarkSummaryStatusIncomplete    = "incomplete"
	BenchmarkSummaryStatusNonComparable = "non_comparable"
)

type BenchmarkSummaryResult struct {
	ScenarioID string                     `json:"scenario_id,omitempty"`
	Count      int                        `json:"count"`
	Summaries  []BenchmarkScenarioSummary `json:"summaries"`
}

type BenchmarkScenarioSummary struct {
	Scenario   benchmarks.Scenario     `json:"scenario"`
	Status     string                  `json:"status"`
	Comparable bool                    `json:"comparable"`
	Issues     []BenchmarkSummaryIssue `json:"issues,omitempty"`
	Runs       BenchmarkScenarioRuns   `json:"runs"`
	Comparison *BenchmarkComparison    `json:"comparison,omitempty"`
}

type BenchmarkSummaryIssue struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type BenchmarkScenarioRuns struct {
	WithGraph    BenchmarkModeRuns `json:"with_graph"`
	WithoutGraph BenchmarkModeRuns `json:"without_graph"`
}

type BenchmarkModeRuns struct {
	Count    int                    `json:"count"`
	RunIDs   []string               `json:"run_ids,omitempty"`
	Selected *BenchmarkRunSelection `json:"selected,omitempty"`
}

type BenchmarkRunSelection struct {
	Path   string               `json:"path"`
	Report benchmarks.RunReport `json:"report"`
}

type BenchmarkComparison struct {
	Basis       string                         `json:"basis"`
	Volume      BenchmarkVolumeComparison      `json:"volume"`
	Orientation BenchmarkOrientationComparison `json:"orientation"`
	Outcome     BenchmarkOutcomeComparison     `json:"outcome"`
}

type BenchmarkVolumeComparison struct {
	InputTokens      BenchmarkIntMetricComparison `json:"input_tokens"`
	OutputTokens     BenchmarkIntMetricComparison `json:"output_tokens"`
	PromptCount      BenchmarkIntMetricComparison `json:"prompt_count"`
	PromptCharacters BenchmarkIntMetricComparison `json:"prompt_characters"`
}

type BenchmarkOrientationComparison struct {
	StepCount       BenchmarkIntMetricComparison `json:"step_count"`
	RepoScans       BenchmarkIntMetricComparison `json:"repo_scans"`
	FollowUpPrompts BenchmarkIntMetricComparison `json:"follow_up_prompts"`
	ContextReloads  BenchmarkIntMetricComparison `json:"context_reloads"`
}

type BenchmarkOutcomeComparison struct {
	QualityRating          BenchmarkStringMetricComparison `json:"quality_rating"`
	ResumabilityRating     BenchmarkStringMetricComparison `json:"resumability_rating"`
	AcceptanceChecksPassed BenchmarkIntMetricComparison    `json:"acceptance_checks_passed"`
	AcceptanceChecksTotal  BenchmarkIntMetricComparison    `json:"acceptance_checks_total"`
}

type BenchmarkIntMetricComparison struct {
	WithGraph           int `json:"with_graph"`
	WithoutGraph        int `json:"without_graph"`
	DeltaVsWithoutGraph int `json:"delta_vs_without_graph"`
}

type BenchmarkStringMetricComparison struct {
	WithGraph    string `json:"with_graph"`
	WithoutGraph string `json:"without_graph"`
	Matches      bool   `json:"matches"`
}

func (s *Service) StoreBenchmarkScenario(ctx context.Context, startDir string, scenario benchmarks.Scenario) (BenchmarkScenarioResult, error) {
	if s == nil || s.manager == nil {
		return BenchmarkScenarioResult{}, errors.New("workflow service is not configured")
	}
	if s.benchmarkStore == nil {
		s.benchmarkStore = benchmarks.NewStore()
	}

	workspace, err := s.manager.OpenWorkspace(ctx, startDir)
	if err != nil {
		return BenchmarkScenarioResult{}, err
	}

	artifact, err := s.benchmarkStore.WriteScenario(workspace, scenario)
	if err != nil {
		return BenchmarkScenarioResult{}, classifyBenchmarkArtifactError("scenario_write", err)
	}

	return BenchmarkScenarioResult{
		Path:     artifact.Path,
		Scenario: artifact.Scenario,
	}, nil
}

func (s *Service) RecordBenchmarkRun(ctx context.Context, startDir string, report benchmarks.RunReport) (BenchmarkRunResult, error) {
	if s == nil || s.manager == nil {
		return BenchmarkRunResult{}, errors.New("workflow service is not configured")
	}
	if s.benchmarkStore == nil {
		s.benchmarkStore = benchmarks.NewStore()
	}

	workspace, err := s.manager.OpenWorkspace(ctx, startDir)
	if err != nil {
		return BenchmarkRunResult{}, err
	}

	scenarioArtifact, err := s.benchmarkStore.ReadScenario(workspace, report.ScenarioID)
	if err != nil {
		return BenchmarkRunResult{}, classifyBenchmarkArtifactError("scenario_read", err)
	}

	artifact, err := s.benchmarkStore.WriteRun(workspace, scenarioArtifact.Scenario, report)
	if err != nil {
		return BenchmarkRunResult{}, classifyBenchmarkArtifactError("run_write", err)
	}

	return BenchmarkRunResult{
		Path:   artifact.Path,
		Report: artifact.Report,
	}, nil
}

func (s *Service) SummarizeBenchmark(ctx context.Context, startDir, scenarioID string) (BenchmarkSummaryResult, error) {
	if s == nil || s.manager == nil {
		return BenchmarkSummaryResult{}, errors.New("workflow service is not configured")
	}
	if s.benchmarkStore == nil {
		s.benchmarkStore = benchmarks.NewStore()
	}

	workspace, err := s.manager.OpenWorkspace(ctx, startDir)
	if err != nil {
		return BenchmarkSummaryResult{}, err
	}

	scenarioID = strings.TrimSpace(scenarioID)
	scenarios := []benchmarks.ScenarioArtifact{}
	if scenarioID != "" {
		artifact, err := s.benchmarkStore.LoadScenario(workspace, scenarioID)
		if err != nil {
			return BenchmarkSummaryResult{}, classifyBenchmarkArtifactError("scenario_load", err)
		}
		scenarios = append(scenarios, artifact)
	} else {
		scenarios, err = s.benchmarkStore.ListScenarios(workspace)
		if err != nil {
			return BenchmarkSummaryResult{}, classifyBenchmarkArtifactError("scenario_list", err)
		}
	}

	summaries := make([]BenchmarkScenarioSummary, 0, len(scenarios))
	for _, artifact := range scenarios {
		runArtifacts, err := s.benchmarkStore.ListRuns(workspace, benchmarkScenarioID(artifact))
		if err != nil {
			return BenchmarkSummaryResult{}, classifyBenchmarkArtifactError("run_list", err)
		}
		summaries = append(summaries, summarizeBenchmarkScenario(artifact, runArtifacts))
	}

	slices.SortFunc(summaries, func(a, b BenchmarkScenarioSummary) int {
		switch {
		case a.Scenario.ScenarioID < b.Scenario.ScenarioID:
			return -1
		case a.Scenario.ScenarioID > b.Scenario.ScenarioID:
			return 1
		default:
			return 0
		}
	})

	return BenchmarkSummaryResult{
		ScenarioID: scenarioID,
		Count:      len(summaries),
		Summaries:  summaries,
	}, nil
}

func classifyBenchmarkArtifactError(stage string, err error) error {
	if _, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return err
	}

	var validationErr *benchmarks.ValidationError
	if errors.As(err, &validationErr) {
		return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "validation_error",
			Code:     validationErr.Code,
			Message:  validationErr.Message,
			Details:  withBenchmarkStage(validationErr.Details, stage),
		}, err)
	}

	var benchmarkErr *benchmarks.Error
	if errors.As(err, &benchmarkErr) {
		return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "operational_error",
			Type:     "benchmark_error",
			Code:     benchmarkErr.Code,
			Message:  benchmarkErr.Message,
			Details:  withBenchmarkStage(benchmarkErr.Details, stage),
		}, err)
	}

	return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "benchmark_error",
		Code:     "benchmark_artifact_failed",
		Message:  "benchmark artifact operation failed",
		Details: map[string]any{
			"stage":  stage,
			"reason": err.Error(),
		},
	}, err)
}

func withBenchmarkStage(details map[string]any, stage string) map[string]any {
	merged := map[string]any{
		"stage": stage,
	}
	for key, value := range details {
		merged[key] = value
	}
	return merged
}

func summarizeBenchmarkScenario(artifact benchmarks.ScenarioArtifact, runArtifacts []benchmarks.RunArtifact) BenchmarkScenarioSummary {
	scenario := artifact.Scenario
	scenarioID := benchmarkScenarioID(artifact)
	if scenario.ScenarioID == "" {
		scenario.ScenarioID = scenarioID
	}

	summary := BenchmarkScenarioSummary{
		Scenario:   scenario,
		Status:     BenchmarkSummaryStatusComparable,
		Comparable: true,
	}
	addIssue := func(severity, code, message string, details map[string]any) {
		summary.Issues = append(summary.Issues, BenchmarkSummaryIssue{
			Code:    code,
			Message: message,
			Details: details,
		})
		summary.Status = elevateBenchmarkSummaryStatus(summary.Status, severity)
	}

	if scenario.SchemaVersion != "" && scenario.SchemaVersion != benchmarks.SchemaVersion {
		addIssue(BenchmarkSummaryStatusNonComparable, "unsupported_scenario_schema", "benchmark scenario schema_version is unsupported for summary comparison", map[string]any{
			"scenario_id":     scenarioID,
			"schema_version":  scenario.SchemaVersion,
			"supported_value": benchmarks.SchemaVersion,
		})
	}
	if scenario.Kind != "" && scenario.Kind != benchmarks.ScenarioArtifactKind {
		addIssue(BenchmarkSummaryStatusNonComparable, "unsupported_scenario_kind", "benchmark scenario kind is unsupported for summary comparison", map[string]any{
			"scenario_id":     scenarioID,
			"kind":            scenario.Kind,
			"supported_value": benchmarks.ScenarioArtifactKind,
		})
	}

	modeCounts := map[string]int{}
	for _, mode := range scenario.Modes {
		modeCounts[mode.Mode]++
		switch mode.Mode {
		case benchmarks.ModeWithGraph, benchmarks.ModeWithoutGraph:
		case "":
			addIssue(BenchmarkSummaryStatusNonComparable, "scenario_mode_missing", "benchmark scenario is missing a workflow mode identifier", map[string]any{
				"scenario_id": scenarioID,
			})
		default:
			addIssue(BenchmarkSummaryStatusNonComparable, "unsupported_scenario_mode", "benchmark scenario defines an unsupported workflow mode", map[string]any{
				"scenario_id": scenarioID,
				"mode":        mode.Mode,
			})
		}
	}
	for _, requiredMode := range []string{benchmarks.ModeWithGraph, benchmarks.ModeWithoutGraph} {
		switch modeCounts[requiredMode] {
		case 0:
			addIssue(BenchmarkSummaryStatusNonComparable, "scenario_mode_missing", "benchmark scenario does not define both comparable workflow modes", map[string]any{
				"scenario_id": scenarioID,
				"mode":        requiredMode,
			})
		case 1:
		default:
			addIssue(BenchmarkSummaryStatusNonComparable, "duplicate_scenario_mode", "benchmark scenario defines a comparable workflow mode more than once", map[string]any{
				"scenario_id": scenarioID,
				"mode":        requiredMode,
				"count":       modeCounts[requiredMode],
			})
		}
	}

	modeRuns := map[string][]benchmarks.RunArtifact{
		benchmarks.ModeWithGraph:    {},
		benchmarks.ModeWithoutGraph: {},
	}
	for _, artifact := range runArtifacts {
		run := artifact.Report
		runID := benchmarkRunID(artifact)
		if run.SchemaVersion != "" && run.SchemaVersion != benchmarks.SchemaVersion {
			addIssue(BenchmarkSummaryStatusNonComparable, "unsupported_run_schema", "benchmark run schema_version is unsupported for summary comparison", map[string]any{
				"scenario_id":     scenarioID,
				"run_id":          runID,
				"schema_version":  run.SchemaVersion,
				"supported_value": benchmarks.SchemaVersion,
			})
		}
		if run.Kind != "" && run.Kind != benchmarks.RunArtifactKind {
			addIssue(BenchmarkSummaryStatusNonComparable, "unsupported_run_kind", "benchmark run kind is unsupported for summary comparison", map[string]any{
				"scenario_id":     scenarioID,
				"run_id":          runID,
				"kind":            run.Kind,
				"supported_value": benchmarks.RunArtifactKind,
			})
		}
		if run.ScenarioID == "" {
			addIssue(BenchmarkSummaryStatusNonComparable, "run_scenario_id_missing", "benchmark run is missing scenario_id metadata", map[string]any{
				"scenario_id": scenarioID,
				"run_id":      runID,
				"path":        filepath.ToSlash(artifact.Path),
			})
			continue
		}
		if scenarioID != "" && run.ScenarioID != scenarioID {
			addIssue(BenchmarkSummaryStatusNonComparable, "run_scenario_mismatch", "benchmark run scenario_id does not match the requested scenario", map[string]any{
				"scenario_id":          scenarioID,
				"run_id":               runID,
				"reported_scenario_id": run.ScenarioID,
				"path":                 filepath.ToSlash(artifact.Path),
			})
			continue
		}
		switch run.Mode {
		case benchmarks.ModeWithGraph, benchmarks.ModeWithoutGraph:
			modeRuns[run.Mode] = append(modeRuns[run.Mode], artifact)
		default:
			addIssue(BenchmarkSummaryStatusNonComparable, "unsupported_run_mode", "benchmark run uses an unsupported workflow mode", map[string]any{
				"scenario_id": scenarioID,
				"run_id":      runID,
				"mode":        run.Mode,
				"path":        filepath.ToSlash(artifact.Path),
			})
		}
	}

	summary.Runs.WithGraph = summarizeBenchmarkModeRuns(modeRuns[benchmarks.ModeWithGraph])
	summary.Runs.WithoutGraph = summarizeBenchmarkModeRuns(modeRuns[benchmarks.ModeWithoutGraph])
	for _, mode := range []struct {
		name string
		runs []benchmarks.RunArtifact
	}{
		{name: benchmarks.ModeWithGraph, runs: modeRuns[benchmarks.ModeWithGraph]},
		{name: benchmarks.ModeWithoutGraph, runs: modeRuns[benchmarks.ModeWithoutGraph]},
	} {
		switch len(mode.runs) {
		case 0:
			addIssue(BenchmarkSummaryStatusIncomplete, "missing_mode_run", "benchmark summary is missing a run for one workflow mode", map[string]any{
				"scenario_id": scenarioID,
				"mode":        mode.name,
			})
		case 1:
		default:
			addIssue(BenchmarkSummaryStatusNonComparable, "multiple_mode_runs", "benchmark summary found multiple runs for a single workflow mode", map[string]any{
				"scenario_id": scenarioID,
				"mode":        mode.name,
				"count":       len(mode.runs),
				"run_ids":     summaryRunIDs(mode.runs),
			})
		}
	}

	if summary.Status == BenchmarkSummaryStatusNonComparable {
		summary.Comparable = false
		return summary
	}

	selectedWithGraph := summary.Runs.WithGraph.Selected
	selectedWithoutGraph := summary.Runs.WithoutGraph.Selected
	if selectedWithGraph == nil || selectedWithoutGraph == nil {
		summary.Comparable = false
		return summary
	}

	for _, selection := range []struct {
		mode string
		run  *BenchmarkRunSelection
	}{
		{mode: benchmarks.ModeWithGraph, run: selectedWithGraph},
		{mode: benchmarks.ModeWithoutGraph, run: selectedWithoutGraph},
	} {
		if missingFields := missingBenchmarkComparisonFields(selection.run.Report); len(missingFields) > 0 {
			addIssue(BenchmarkSummaryStatusIncomplete, "missing_comparison_fields", "benchmark run is missing fields required for summary comparison", map[string]any{
				"scenario_id": scenarioID,
				"mode":        selection.mode,
				"run_id":      selection.run.Report.RunID,
				"fields":      missingFields,
			})
		}
	}

	if summary.Status == BenchmarkSummaryStatusComparable {
		summary.Comparison = buildBenchmarkComparison(selectedWithGraph.Report, selectedWithoutGraph.Report)
	}
	summary.Comparable = summary.Status == BenchmarkSummaryStatusComparable
	return summary
}

func summarizeBenchmarkModeRuns(artifacts []benchmarks.RunArtifact) BenchmarkModeRuns {
	runs := BenchmarkModeRuns{
		Count:  len(artifacts),
		RunIDs: summaryRunIDs(artifacts),
	}
	if len(artifacts) == 1 {
		runs.Selected = &BenchmarkRunSelection{
			Path:   filepath.ToSlash(artifacts[0].Path),
			Report: artifacts[0].Report,
		}
	}
	if len(runs.RunIDs) == 0 {
		runs.RunIDs = nil
	}
	return runs
}

func summaryRunIDs(artifacts []benchmarks.RunArtifact) []string {
	if len(artifacts) == 0 {
		return nil
	}
	runIDs := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		runIDs = append(runIDs, benchmarkRunID(artifact))
	}
	return runIDs
}

func missingBenchmarkComparisonFields(report benchmarks.RunReport) []string {
	missing := []string{}
	if report.Metrics.Volume.InputTokens <= 0 {
		missing = append(missing, "metrics.volume.input_tokens")
	}
	if report.Metrics.Volume.OutputTokens <= 0 {
		missing = append(missing, "metrics.volume.output_tokens")
	}
	if report.Metrics.Volume.PromptCount <= 0 {
		missing = append(missing, "metrics.volume.prompt_count")
	}
	if report.Metrics.Volume.PromptCharacters <= 0 {
		missing = append(missing, "metrics.volume.prompt_characters")
	}
	if report.Metrics.Orientation.StepCount <= 0 {
		missing = append(missing, "metrics.orientation.step_count")
	}
	if strings.TrimSpace(report.Metrics.Outcome.QualityRating) == "" {
		missing = append(missing, "metrics.outcome.quality_rating")
	}
	if strings.TrimSpace(report.Metrics.Outcome.ResumabilityRating) == "" {
		missing = append(missing, "metrics.outcome.resumability_rating")
	}
	if report.Metrics.Outcome.AcceptanceChecksTotal <= 0 {
		missing = append(missing, "metrics.outcome.acceptance_checks_total")
	}
	if report.Metrics.Outcome.AcceptanceChecksPassed < 0 || report.Metrics.Outcome.AcceptanceChecksPassed > report.Metrics.Outcome.AcceptanceChecksTotal {
		missing = append(missing, "metrics.outcome.acceptance_checks_passed")
	}
	return missing
}

func buildBenchmarkComparison(withGraph, withoutGraph benchmarks.RunReport) *BenchmarkComparison {
	return &BenchmarkComparison{
		Basis: "single_run_per_mode",
		Volume: BenchmarkVolumeComparison{
			InputTokens:      compareBenchmarkInts(withGraph.Metrics.Volume.InputTokens, withoutGraph.Metrics.Volume.InputTokens),
			OutputTokens:     compareBenchmarkInts(withGraph.Metrics.Volume.OutputTokens, withoutGraph.Metrics.Volume.OutputTokens),
			PromptCount:      compareBenchmarkInts(withGraph.Metrics.Volume.PromptCount, withoutGraph.Metrics.Volume.PromptCount),
			PromptCharacters: compareBenchmarkInts(withGraph.Metrics.Volume.PromptCharacters, withoutGraph.Metrics.Volume.PromptCharacters),
		},
		Orientation: BenchmarkOrientationComparison{
			StepCount:       compareBenchmarkInts(withGraph.Metrics.Orientation.StepCount, withoutGraph.Metrics.Orientation.StepCount),
			RepoScans:       compareBenchmarkInts(withGraph.Metrics.Orientation.RepoScans, withoutGraph.Metrics.Orientation.RepoScans),
			FollowUpPrompts: compareBenchmarkInts(withGraph.Metrics.Orientation.FollowUpPrompts, withoutGraph.Metrics.Orientation.FollowUpPrompts),
			ContextReloads:  compareBenchmarkInts(withGraph.Metrics.Orientation.ContextReloads, withoutGraph.Metrics.Orientation.ContextReloads),
		},
		Outcome: BenchmarkOutcomeComparison{
			QualityRating:          compareBenchmarkStrings(withGraph.Metrics.Outcome.QualityRating, withoutGraph.Metrics.Outcome.QualityRating),
			ResumabilityRating:     compareBenchmarkStrings(withGraph.Metrics.Outcome.ResumabilityRating, withoutGraph.Metrics.Outcome.ResumabilityRating),
			AcceptanceChecksPassed: compareBenchmarkInts(withGraph.Metrics.Outcome.AcceptanceChecksPassed, withoutGraph.Metrics.Outcome.AcceptanceChecksPassed),
			AcceptanceChecksTotal:  compareBenchmarkInts(withGraph.Metrics.Outcome.AcceptanceChecksTotal, withoutGraph.Metrics.Outcome.AcceptanceChecksTotal),
		},
	}
}

func compareBenchmarkInts(withGraph, withoutGraph int) BenchmarkIntMetricComparison {
	return BenchmarkIntMetricComparison{
		WithGraph:           withGraph,
		WithoutGraph:        withoutGraph,
		DeltaVsWithoutGraph: withGraph - withoutGraph,
	}
}

func compareBenchmarkStrings(withGraph, withoutGraph string) BenchmarkStringMetricComparison {
	return BenchmarkStringMetricComparison{
		WithGraph:    withGraph,
		WithoutGraph: withoutGraph,
		Matches:      withGraph == withoutGraph,
	}
}

func elevateBenchmarkSummaryStatus(current, candidate string) string {
	if benchmarkSummaryStatusRank(candidate) > benchmarkSummaryStatusRank(current) {
		return candidate
	}
	return current
}

func benchmarkSummaryStatusRank(status string) int {
	switch status {
	case BenchmarkSummaryStatusNonComparable:
		return 2
	case BenchmarkSummaryStatusIncomplete:
		return 1
	default:
		return 0
	}
}

func benchmarkScenarioID(artifact benchmarks.ScenarioArtifact) string {
	if artifact.Scenario.ScenarioID != "" {
		return artifact.Scenario.ScenarioID
	}
	return strings.TrimSuffix(filepath.Base(artifact.Path), filepath.Ext(artifact.Path))
}

func benchmarkRunID(artifact benchmarks.RunArtifact) string {
	if artifact.Report.RunID != "" {
		return artifact.Report.RunID
	}
	return strings.TrimSuffix(filepath.Base(artifact.Path), filepath.Ext(artifact.Path))
}
