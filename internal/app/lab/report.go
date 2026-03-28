package lab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/app/workflow"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

const (
	reportSelectionAllRuns      = "all_runs"
	reportSelectionSelectedRuns = "selected_runs"

	groupingModel           = "model"
	groupingSessionTopology = "session_topology"
)

type ReportRequest struct {
	RunIDs []string `json:"run_ids,omitempty"`
}

type ReportResult struct {
	ReportID     string         `json:"report_id"`
	ArtifactPath string         `json:"artifact_path"`
	Report       ReportArtifact `json:"report"`
}

type ReportArtifact struct {
	SchemaVersion           string                         `json:"schema_version"`
	ReportID                string                         `json:"report_id"`
	GeneratedAt             string                         `json:"generated_at"`
	SuiteID                 string                         `json:"suite_id"`
	Selection               ReportSelection                `json:"selection"`
	SourceArtifacts         ReportSourceSummary            `json:"source_artifacts"`
	RunsIncluded            int                            `json:"runs_included"`
	RunsScored              int                            `json:"runs_scored"`
	RunsUnscored            int                            `json:"runs_unscored"`
	UnscoredRunIDs          []string                       `json:"unscored_run_ids"`
	PairedComparisons       []PairedComparison             `json:"paired_comparisons"`
	GroupedComparisons      []GroupedComparison            `json:"grouped_comparisons"`
	VerificationAttribution VerificationAttributionSummary `json:"verification_attribution"`
	Summary                 ReportSummary                  `json:"summary"`
	Limitations             []string                       `json:"limitations"`
}

type VerificationAttributionSummary struct {
	Profiles []VerificationProfileSummary `json:"profiles,omitempty"`
	Gate     VerificationRerunGate        `json:"gate"`
}

type VerificationProfileSummary struct {
	Profile                     string         `json:"profile"`
	PairCount                   int            `json:"pair_count"`
	TokenSampleCount            int            `json:"token_sample_count"`
	QualitySampleCount          int            `json:"quality_sample_count"`
	EffectiveModes              map[string]int `json:"effective_modes"`
	AbstentionRate              float64        `json:"abstention_rate"`
	MeanConfidence              float64        `json:"mean_confidence"`
	MeanTokenDelta              float64        `json:"mean_token_delta"`
	MeanQualityDelta            float64        `json:"mean_quality_delta"`
	BaselinePromptMetadataPairs int            `json:"baseline_prompt_metadata_pairs"`
}

type VerificationRerunGate struct {
	Decision string   `json:"decision"`
	Rule     string   `json:"rule"`
	Reasons  []string `json:"reasons,omitempty"`
}

type ReportSelection struct {
	Mode   string   `json:"mode"`
	RunIDs []string `json:"run_ids,omitempty"`
}

type ReportSourceSummary struct {
	RunRecordCount          int               `json:"run_record_count"`
	EvaluationArtifactCount int               `json:"evaluation_artifact_count"`
	BatchPlans              []BatchPlanSource `json:"batch_plans"`
}

type BatchPlanSource struct {
	BatchID         string `json:"batch_id"`
	PlanPath        string `json:"plan_path"`
	PlannedAt       string `json:"planned_at,omitempty"`
	EntryCount      int    `json:"entry_count"`
	Randomized      bool   `json:"randomized"`
	Model           string `json:"model,omitempty"`
	SessionTopology string `json:"session_topology,omitempty"`
}

type PairedComparison struct {
	ComparisonType  string                   `json:"comparison_type"`
	TaskID          string                   `json:"task_id"`
	TaskFamily      string                   `json:"task_family"`
	Model           string                   `json:"model"`
	SessionTopology string                   `json:"session_topology"`
	PairCount       int                      `json:"pair_count"`
	ScoredPairCount int                      `json:"scored_pair_count"`
	Incomplete      bool                     `json:"incomplete"`
	Pairings        []PairedRun              `json:"pairings"`
	Metrics         map[string]MetricSummary `json:"metrics"`
	Limitations     []string                 `json:"limitations"`
}

type PairedRun struct {
	GraphRunID       string             `json:"graph_run_id,omitempty"`
	BaselineRunID    string             `json:"baseline_run_id,omitempty"`
	GraphSeed        *int64             `json:"graph_seed,omitempty"`
	BaselineSeed     *int64             `json:"baseline_seed,omitempty"`
	Scored           bool               `json:"scored"`
	IncompleteReason string             `json:"incomplete_reason,omitempty"`
	Deltas           map[string]float64 `json:"deltas,omitempty"`
}

type MetricSummary struct {
	HigherIsBetter bool              `json:"higher_is_better"`
	Unit           string            `json:"unit,omitempty"`
	SampleSize     int               `json:"sample_size"`
	GraphMean      float64           `json:"graph_mean"`
	BaselineMean   float64           `json:"baseline_mean"`
	MeanDelta      float64           `json:"mean_delta"`
	EffectSize     float64           `json:"effect_size"`
	Uncertainty    MetricUncertainty `json:"uncertainty"`
	Interpretation string            `json:"interpretation"`
}

type MetricUncertainty struct {
	Method     string    `json:"method"`
	Interval95 []float64 `json:"interval_95,omitempty"`
	Note       string    `json:"note,omitempty"`
}

type GroupedComparison struct {
	Grouping       string             `json:"grouping"`
	TaskFamily     string             `json:"task_family"`
	GroupCount     int                `json:"group_count"`
	Groups         []GroupedMetricSet `json:"groups"`
	VariationNotes []string           `json:"variation_notes"`
}

type GroupedMetricSet struct {
	GroupValue  string                   `json:"group_value"`
	PairCount   int                      `json:"pair_count"`
	Metrics     map[string]MetricSummary `json:"metrics"`
	Limitations []string                 `json:"limitations"`
}

type ReportSummary struct {
	Warnings        []ReportWarning `json:"warnings"`
	NullResults     []ReportFinding `json:"null_results"`
	NegativeResults []ReportFinding `json:"negative_results"`
}

type ReportWarning struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	RunIDs  []string `json:"run_ids,omitempty"`
}

type ReportFinding struct {
	Scope           string  `json:"scope"`
	Metric          string  `json:"metric"`
	TaskID          string  `json:"task_id,omitempty"`
	TaskFamily      string  `json:"task_family"`
	Model           string  `json:"model,omitempty"`
	SessionTopology string  `json:"session_topology,omitempty"`
	Grouping        string  `json:"grouping,omitempty"`
	GroupValue      string  `json:"group_value,omitempty"`
	MeanDelta       float64 `json:"mean_delta"`
	SampleSize      int     `json:"sample_size"`
	Reason          string  `json:"reason"`
}

type reportRun struct {
	record                 RunRecord
	task                   SuiteTask
	condition              Condition
	path                   string
	latestEvaluation       *EvaluationRecord
	evaluationCount        int
	workflowStart          *workflow.StartResult
	baselinePromptMetadata map[string]any
}

type pairKey struct {
	taskID          string
	taskFamily      string
	model           string
	sessionTopology string
}

type pairBucket struct {
	key       pairKey
	graphRuns []reportRun
	baseRuns  []reportRun
}

type pairObservation struct {
	taskFamily      string
	model           string
	sessionTopology string
	metrics         map[string]pairedMetricValue
}

type pairedMetricValue struct {
	graph    float64
	baseline float64
	delta    float64
}

type groupAccumulator struct {
	grouping   string
	taskFamily string
	groupValue string
	metrics    map[string][]pairedMetricValue
}

type verificationObservation struct {
	profile                       string
	effectiveMode                 string
	confidenceScore               float64
	abstained                     bool
	tokenDelta                    *float64
	qualityDelta                  *float64
	baselinePromptMetadataPresent bool
}

func (s *Service) Report(ctx context.Context, startDir string, request ReportRequest) (ReportResult, error) {
	if s == nil || s.manager == nil {
		return ReportResult{}, errors.New("lab service is not configured")
	}
	if s.now == nil {
		s.now = func() time.Time { return time.Now().UTC() }
	}

	workspace, err := s.manager.OpenWorkspace(ctx, startDir)
	if err != nil {
		return ReportResult{}, err
	}

	labPath := filepath.Join(workspace.WorkspacePath, repo.LabDirName)
	suite, err := loadSuiteManifest(filepath.Join(labPath, repo.LabSuiteManifestName))
	if err != nil {
		return ReportResult{}, err
	}
	conditions, err := loadConditionsManifest(filepath.Join(labPath, repo.LabConditionsManifestName))
	if err != nil {
		return ReportResult{}, err
	}

	request.RunIDs = normalizeIdentifiers(request.RunIDs)
	allRuns, err := loadReportRuns(filepath.Join(labPath, "runs"), suite, conditions)
	if err != nil {
		return ReportResult{}, err
	}
	selectedRuns, selection, err := selectReportRuns(request, allRuns)
	if err != nil {
		return ReportResult{}, err
	}

	batchPlans, err := loadBatchPlans(filepath.Join(labPath, "runs", "batches"), workspace.RepoRoot)
	if err != nil {
		return ReportResult{}, err
	}

	generatedAt := s.now().UTC()
	report := buildReportArtifact(generatedAt, suite, selection, allRuns, selectedRuns, batchPlans)
	reportID := nextReportID(filepath.Join(labPath, "reports"), generatedAt)
	report.ReportID = reportID
	report.GeneratedAt = generatedAt.Format(time.RFC3339)

	reportPath := filepath.Join(labPath, "reports", reportID+".json")
	if err := writeLedgerJSON(reportPath, report); err != nil {
		return ReportResult{}, err
	}

	return ReportResult{
		ReportID:     reportID,
		ArtifactPath: relativePath(workspace.RepoRoot, reportPath),
		Report:       report,
	}, nil
}

func selectReportRuns(request ReportRequest, allRuns []reportRun) ([]reportRun, ReportSelection, error) {
	if len(request.RunIDs) == 0 {
		return append([]reportRun(nil), allRuns...), ReportSelection{Mode: reportSelectionAllRuns}, nil
	}

	index := make(map[string]reportRun, len(allRuns))
	for _, item := range allRuns {
		index[item.record.RunID] = item
	}

	selected := make([]reportRun, 0, len(request.RunIDs))
	var violations []map[string]any
	for i, runID := range request.RunIDs {
		item, ok := index[runID]
		if !ok {
			violations = append(violations, violationWithValue(fmt.Sprintf("run_ids[%d]", i), "run_id must reference an existing run record", runID))
			continue
		}
		selected = append(selected, item)
	}

	if len(violations) > 0 {
		return nil, ReportSelection{}, cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "validation_error",
			Code:     "lab_report_validation_failed",
			Message:  "lab report request validation failed",
			Details: map[string]any{
				"violations": violations,
			},
		}, fmt.Errorf("lab report request validation failed"))
	}

	return selected, ReportSelection{
		Mode:   reportSelectionSelectedRuns,
		RunIDs: append([]string(nil), request.RunIDs...),
	}, nil
}

func buildReportArtifact(
	generatedAt time.Time,
	suite SuiteManifest,
	selection ReportSelection,
	allRuns []reportRun,
	selectedRuns []reportRun,
	batchPlans []BatchPlanSource,
) ReportArtifact {
	pairedComparisons, pairObservations, verificationObservations, unscoredRunIDs := buildPairedComparisons(selectedRuns)
	groupedComparisons := buildGroupedComparisons(pairObservations)
	verificationAttribution := buildVerificationAttribution(verificationObservations)
	sort.Strings(unscoredRunIDs)

	warnings := []ReportWarning{}
	if len(unscoredRunIDs) > 0 {
		warnings = append(warnings, ReportWarning{
			Code:    "missing_evaluations",
			Message: "some selected runs do not have evaluation records; scored comparisons exclude those runs",
			RunIDs:  append([]string(nil), unscoredRunIDs...),
		})
	}
	if len(selectedRuns) == 0 {
		warnings = append(warnings, ReportWarning{
			Code:    "no_runs_selected",
			Message: "the report selection did not match any run records",
		})
	}
	if tokenSparse := countTokenSparseComparisons(pairedComparisons); tokenSparse > 0 {
		warnings = append(warnings, ReportWarning{
			Code:    "incomplete_token_telemetry",
			Message: "some scored comparisons lack complete token telemetry; token metrics exclude incomplete measurements",
		})
	}

	limitations := []string{}
	if len(unscoredRunIDs) > 0 {
		limitations = append(limitations, fmt.Sprintf("%d selected runs are unscored; quality, success, and resumability metrics only reflect scored runs.", len(unscoredRunIDs)))
	}
	if sparse := countSparseComparisons(pairedComparisons); sparse > 0 {
		limitations = append(limitations, fmt.Sprintf("%d paired comparisons have fewer than two scored pairs, so interval estimates are missing or highly uncertain.", sparse))
	}
	if len(selectedRuns) == 0 {
		limitations = append(limitations, "No run records matched the report selection.")
	}
	if tokenSparse := countTokenSparseComparisons(pairedComparisons); tokenSparse > 0 {
		limitations = append(limitations, fmt.Sprintf("%d paired comparisons lack complete token telemetry for at least one scored pair, so token metrics exclude incomplete measurements.", tokenSparse))
	}

	runsScored := 0
	for _, item := range selectedRuns {
		if item.latestEvaluation != nil {
			runsScored++
		}
	}

	return ReportArtifact{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   generatedAt.UTC().Format(time.RFC3339),
		SuiteID:       suite.SuiteID,
		Selection:     selection,
		SourceArtifacts: ReportSourceSummary{
			RunRecordCount:          len(allRuns),
			EvaluationArtifactCount: countEvaluationArtifacts(allRuns),
			BatchPlans:              batchPlans,
		},
		RunsIncluded:            len(selectedRuns),
		RunsScored:              runsScored,
		RunsUnscored:            len(selectedRuns) - runsScored,
		UnscoredRunIDs:          unscoredRunIDs,
		PairedComparisons:       pairedComparisons,
		GroupedComparisons:      groupedComparisons,
		VerificationAttribution: verificationAttribution,
		Summary: ReportSummary{
			Warnings:        warnings,
			NullResults:     collectFindings(pairedComparisons, groupedComparisons, true),
			NegativeResults: collectFindings(pairedComparisons, groupedComparisons, false),
		},
		Limitations: limitations,
	}
}

func buildPairedComparisons(selectedRuns []reportRun) ([]PairedComparison, []pairObservation, []verificationObservation, []string) {
	buckets := map[pairKey]*pairBucket{}
	unscoredSet := map[string]struct{}{}

	for _, item := range selectedRuns {
		if item.latestEvaluation == nil {
			unscoredSet[item.record.RunID] = struct{}{}
		}

		key := pairKey{
			taskID:          item.record.TaskID,
			taskFamily:      item.task.Family,
			model:           item.record.Model,
			sessionTopology: item.record.SessionTopology,
		}
		bucket := buckets[key]
		if bucket == nil {
			bucket = &pairBucket{key: key}
			buckets[key] = bucket
		}

		if item.condition.WorkflowMode == WorkflowModeGraphBacked {
			bucket.graphRuns = append(bucket.graphRuns, item)
			continue
		}
		if item.condition.WorkflowMode == WorkflowModeBaseline {
			bucket.baseRuns = append(bucket.baseRuns, item)
		}
	}

	keys := make([]pairKey, 0, len(buckets))
	for key := range buckets {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].taskFamily != keys[j].taskFamily {
			return keys[i].taskFamily < keys[j].taskFamily
		}
		if keys[i].taskID != keys[j].taskID {
			return keys[i].taskID < keys[j].taskID
		}
		if keys[i].model != keys[j].model {
			return keys[i].model < keys[j].model
		}
		return keys[i].sessionTopology < keys[j].sessionTopology
	})

	comparisons := make([]PairedComparison, 0, len(keys))
	observations := []pairObservation{}
	verificationObservations := []verificationObservation{}
	for _, key := range keys {
		bucket := buckets[key]
		sortReportRuns(bucket.graphRuns)
		sortReportRuns(bucket.baseRuns)

		pairCount := len(bucket.graphRuns)
		if len(bucket.baseRuns) > pairCount {
			pairCount = len(bucket.baseRuns)
		}

		pairings := make([]PairedRun, 0, pairCount)
		metricValues := map[string][]pairedMetricValue{}
		limitations := []string{}
		incomplete := false

		for i := 0; i < pairCount; i++ {
			var graphRun *reportRun
			var baseRun *reportRun
			if i < len(bucket.graphRuns) {
				graphRun = &bucket.graphRuns[i]
			}
			if i < len(bucket.baseRuns) {
				baseRun = &bucket.baseRuns[i]
			}

			pair := PairedRun{}
			if graphRun != nil {
				pair.GraphRunID = graphRun.record.RunID
				pair.GraphSeed = graphRun.record.Seed
			}
			if baseRun != nil {
				pair.BaselineRunID = baseRun.record.RunID
				pair.BaselineSeed = baseRun.record.Seed
			}

			switch {
			case graphRun == nil || baseRun == nil:
				incomplete = true
				pair.IncompleteReason = "missing_condition_match"
			case graphRun.latestEvaluation == nil || baseRun.latestEvaluation == nil:
				incomplete = true
				pair.IncompleteReason = "missing_evaluation"
			default:
				pair.Scored = true
				pair.Deltas = runMetricDeltas(*graphRun, *baseRun)
				appendMetricValues(metricValues, *graphRun, *baseRun)
				observations = append(observations, pairObservation{
					taskFamily:      key.taskFamily,
					model:           key.model,
					sessionTopology: key.sessionTopology,
					metrics:         pairMetricValues(*graphRun, *baseRun),
				})
				if verificationObservation, ok := buildVerificationObservation(*graphRun, *baseRun); ok {
					verificationObservations = append(verificationObservations, verificationObservation)
				}
			}

			if pair.IncompleteReason != "" {
				pair.Scored = false
			}
			pairings = append(pairings, pair)
		}

		if incomplete {
			limitations = append(limitations, "At least one graph/baseline pair is incomplete due to a missing counterpart or evaluation.")
		}
		if scored := countScoredPairs(pairings); scored < 2 {
			limitations = append(limitations, "Fewer than two scored pairs are available, so interval estimates are omitted or unstable.")
		}
		if sampleSizeForMetric(metricValues, "total_tokens") < countScoredPairs(pairings) {
			limitations = append(limitations, "At least one scored pair lacks complete token telemetry, so token metrics exclude incomplete measurements.")
		}

		comparisons = append(comparisons, PairedComparison{
			ComparisonType:  "paired_task",
			TaskID:          key.taskID,
			TaskFamily:      key.taskFamily,
			Model:           key.model,
			SessionTopology: key.sessionTopology,
			PairCount:       pairCount,
			ScoredPairCount: countScoredPairs(pairings),
			Incomplete:      incomplete,
			Pairings:        pairings,
			Metrics:         summarizeMetricValues(metricValues),
			Limitations:     limitations,
		})
	}

	unscoredRunIDs := make([]string, 0, len(unscoredSet))
	for runID := range unscoredSet {
		unscoredRunIDs = append(unscoredRunIDs, runID)
	}

	return comparisons, observations, verificationObservations, unscoredRunIDs
}

func buildGroupedComparisons(observations []pairObservation) []GroupedComparison {
	type groupedKey struct {
		grouping   string
		taskFamily string
	}

	grouped := map[groupedKey]map[string]*groupAccumulator{}
	for _, observation := range observations {
		for _, item := range []struct {
			grouping string
			value    string
		}{
			{grouping: groupingModel, value: observation.model},
			{grouping: groupingSessionTopology, value: observation.sessionTopology},
		} {
			key := groupedKey{grouping: item.grouping, taskFamily: observation.taskFamily}
			if grouped[key] == nil {
				grouped[key] = map[string]*groupAccumulator{}
			}
			acc := grouped[key][item.value]
			if acc == nil {
				acc = &groupAccumulator{
					grouping:   item.grouping,
					taskFamily: observation.taskFamily,
					groupValue: item.value,
					metrics:    map[string][]pairedMetricValue{},
				}
				grouped[key][item.value] = acc
			}
			for name, value := range observation.metrics {
				acc.metrics[name] = append(acc.metrics[name], value)
			}
		}
	}

	keys := make([]groupedKey, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].grouping != keys[j].grouping {
			return keys[i].grouping < keys[j].grouping
		}
		return keys[i].taskFamily < keys[j].taskFamily
	})

	comparisons := make([]GroupedComparison, 0, len(keys))
	for _, key := range keys {
		groupValues := make([]string, 0, len(grouped[key]))
		for value := range grouped[key] {
			groupValues = append(groupValues, value)
		}
		sort.Strings(groupValues)

		groups := make([]GroupedMetricSet, 0, len(groupValues))
		for _, value := range groupValues {
			acc := grouped[key][value]
			metrics := summarizeMetricValues(acc.metrics)
			limitations := []string{}
			if sampleSizeForMetrics(metrics) < 2 {
				limitations = append(limitations, "This grouped comparison is based on fewer than two scored pairs.")
			}
			groups = append(groups, GroupedMetricSet{
				GroupValue:  value,
				PairCount:   pairCountForMetricValues(acc.metrics),
				Metrics:     metrics,
				Limitations: limitations,
			})
		}

		comparisons = append(comparisons, GroupedComparison{
			Grouping:       key.grouping,
			TaskFamily:     key.taskFamily,
			GroupCount:     len(groups),
			Groups:         groups,
			VariationNotes: buildVariationNotes(key.grouping, groups),
		})
	}

	return comparisons
}

func loadReportRuns(runsPath string, suite SuiteManifest, conditions ConditionsManifest) ([]reportRun, error) {
	runPaths, err := listRunRecordPaths(runsPath)
	if err != nil {
		return nil, err
	}

	items := make([]reportRun, 0, len(runPaths))
	for _, path := range runPaths {
		record, err := loadRunRecordFromPath(path)
		if err != nil {
			return nil, err
		}

		task, ok := suite.TaskByID(record.TaskID)
		if !ok {
			task = SuiteTask{TaskID: record.TaskID, Family: "unknown"}
		}
		condition, ok := conditions.ConditionByID(record.ConditionID)
		if !ok {
			condition = Condition{ConditionID: record.ConditionID}
		}

		evaluation, count, err := loadLatestEvaluation(filepath.Join(filepath.Dir(runsPath), "evaluations", record.RunID+".json"), record.RunID)
		if err != nil {
			return nil, err
		}
		workflowStartPath := ""
		if record.WorkflowStartResponseRef != "" {
			workflowStartPath = filepath.Join(filepath.Dir(path), record.WorkflowStartResponseRef)
		}
		workflowStart, err := loadWorkflowStartArtifact(workflowStartPath)
		if err != nil {
			return nil, err
		}
		baselinePromptPath := ""
		if record.BaselinePromptMetadataRef != "" {
			baselinePromptPath = filepath.Join(filepath.Dir(path), record.BaselinePromptMetadataRef)
		}
		baselinePromptMetadata, err := loadGenericArtifact(baselinePromptPath)
		if err != nil {
			return nil, err
		}

		items = append(items, reportRun{
			record:                 record,
			task:                   task,
			condition:              condition,
			path:                   path,
			latestEvaluation:       evaluation,
			evaluationCount:        count,
			workflowStart:          workflowStart,
			baselinePromptMetadata: baselinePromptMetadata,
		})
	}

	sortReportRuns(items)
	return items, nil
}

func loadWorkflowStartArtifact(path string) (*workflow.StartResult, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read workflow start artifact %s: %w", path, err)
	}
	var result workflow.StartResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decode workflow start artifact %s: %w", path, err)
	}
	return &result, nil
}

func loadGenericArtifact(path string) (map[string]any, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read artifact %s: %w", path, err)
	}
	payload := map[string]any{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode artifact %s: %w", path, err)
	}
	return payload, nil
}

func listRunRecordPaths(runsPath string) ([]string, error) {
	info, err := os.Stat(runsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("inspect run ledger root %s: %w", runsPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("run ledger root %s is not a directory", runsPath)
	}

	paths := []string{}
	err = filepath.WalkDir(runsPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path == filepath.Join(runsPath, "batches") {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Base(path) == "run.json" || (filepath.Dir(path) == runsPath && strings.HasSuffix(path, ".json")) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list run ledgers under %s: %w", runsPath, err)
	}
	sort.Strings(paths)
	return paths, nil
}

func loadBatchPlans(batchesPath, repoRoot string) ([]BatchPlanSource, error) {
	info, err := os.Stat(batchesPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []BatchPlanSource{}, nil
		}
		return nil, fmt.Errorf("inspect batch plan root %s: %w", batchesPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("batch plan root %s is not a directory", batchesPath)
	}

	plans := []BatchPlanSource{}
	err = filepath.WalkDir(batchesPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Base(path) != "plan.json" {
			return nil
		}

		var plan BatchPlanArtifact
		if err := loadManifestJSON(path, &plan); err != nil {
			return err
		}
		plans = append(plans, BatchPlanSource{
			BatchID:         plan.BatchID,
			PlanPath:        relativePath(repoRoot, path),
			PlannedAt:       plan.PlannedAt,
			EntryCount:      len(plan.Entries),
			Randomized:      plan.Randomized,
			Model:           plan.Model,
			SessionTopology: plan.SessionTopology,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("load batch plans from %s: %w", batchesPath, err)
	}

	sort.Slice(plans, func(i, j int) bool {
		return plans[i].BatchID < plans[j].BatchID
	})
	return plans, nil
}

func loadLatestEvaluation(path, runID string) (*EvaluationRecord, int, error) {
	artifact, err := loadEvaluationArtifact(path, runID)
	if err != nil {
		return nil, 0, err
	}
	if len(artifact.Records) == 0 {
		return nil, 0, nil
	}
	latest := artifact.Records[len(artifact.Records)-1]
	return &latest, len(artifact.Records), nil
}

func sortReportRuns(items []reportRun) {
	sort.Slice(items, func(i, j int) bool {
		return compareReportRuns(items[i], items[j])
	})
}

func compareReportRuns(left, right reportRun) bool {
	leftSeed := int64(math.MinInt64)
	if left.record.Seed != nil {
		leftSeed = *left.record.Seed
	}
	rightSeed := int64(math.MinInt64)
	if right.record.Seed != nil {
		rightSeed = *right.record.Seed
	}
	if leftSeed != rightSeed {
		return leftSeed < rightSeed
	}
	if left.record.StartedAt != right.record.StartedAt {
		return left.record.StartedAt < right.record.StartedAt
	}
	return left.record.RunID < right.record.RunID
}

func runMetricDeltas(graphRun, baseRun reportRun) map[string]float64 {
	values := pairMetricValues(graphRun, baseRun)
	deltas := make(map[string]float64, len(values))
	for name, value := range values {
		deltas[name] = value.delta
	}
	return deltas
}

func pairMetricValues(graphRun, baseRun reportRun) map[string]pairedMetricValue {
	graphScores := graphRun.latestEvaluation.Scores
	baseScores := baseRun.latestEvaluation.Scores

	values := map[string]pairedMetricValue{
		"success": {
			graph:    boolAsFloat(graphScores.Success),
			baseline: boolAsFloat(baseScores.Success),
			delta:    boolAsFloat(graphScores.Success) - boolAsFloat(baseScores.Success),
		},
		"quality_score": {
			graph:    floatPointerValue(graphScores.QualityScore),
			baseline: floatPointerValue(baseScores.QualityScore),
			delta:    floatPointerValue(graphScores.QualityScore) - floatPointerValue(baseScores.QualityScore),
		},
		"resumability_score": {
			graph:    floatPointerValue(graphScores.ResumabilityScore),
			baseline: floatPointerValue(baseScores.ResumabilityScore),
			delta:    floatPointerValue(graphScores.ResumabilityScore) - floatPointerValue(baseScores.ResumabilityScore),
		},
		"wall_clock_seconds": {
			graph:    intMetric(graphRun.record.Telemetry, func(t *RunTelemetry) *int { return t.WallClockSeconds }),
			baseline: intMetric(baseRun.record.Telemetry, func(t *RunTelemetry) *int { return t.WallClockSeconds }),
			delta:    intMetric(graphRun.record.Telemetry, func(t *RunTelemetry) *int { return t.WallClockSeconds }) - intMetric(baseRun.record.Telemetry, func(t *RunTelemetry) *int { return t.WallClockSeconds }),
		},
	}
	graphTotal, graphOK := comparableTokenMetric(graphRun.record.Telemetry, func(t *RunTelemetry) *int { return t.TotalTokens })
	baseTotal, baseOK := comparableTokenMetric(baseRun.record.Telemetry, func(t *RunTelemetry) *int { return t.TotalTokens })
	if graphOK && baseOK {
		values["total_tokens"] = pairedMetricValue{
			graph:    graphTotal,
			baseline: baseTotal,
			delta:    graphTotal - baseTotal,
		}
	}
	return values
}

func appendMetricValues(target map[string][]pairedMetricValue, graphRun, baseRun reportRun) {
	for name, value := range pairMetricValues(graphRun, baseRun) {
		target[name] = append(target[name], value)
	}
}

func summarizeMetricValues(metricValues map[string][]pairedMetricValue) map[string]MetricSummary {
	summaries := map[string]MetricSummary{}
	for _, name := range []string{"success", "quality_score", "resumability_score", "total_tokens", "wall_clock_seconds"} {
		values := metricValues[name]
		summaries[name] = summarizeMetric(name, values)
	}
	return summaries
}

func summarizeMetric(name string, values []pairedMetricValue) MetricSummary {
	higherIsBetter := metricHigherIsBetter(name)
	graphValues := make([]float64, 0, len(values))
	baseValues := make([]float64, 0, len(values))
	deltas := make([]float64, 0, len(values))
	for _, value := range values {
		graphValues = append(graphValues, value.graph)
		baseValues = append(baseValues, value.baseline)
		deltas = append(deltas, value.delta)
	}

	graphMean := meanFloat64(graphValues)
	baseMean := meanFloat64(baseValues)
	deltaMean := meanFloat64(deltas)
	uncertainty := summarizeUncertainty(deltas)

	return MetricSummary{
		HigherIsBetter: higherIsBetter,
		Unit:           metricUnit(name),
		SampleSize:     len(deltas),
		GraphMean:      graphMean,
		BaselineMean:   baseMean,
		MeanDelta:      deltaMean,
		EffectSize:     deltaMean,
		Uncertainty:    uncertainty,
		Interpretation: interpretMetricDelta(name, deltaMean),
	}
}

func summarizeUncertainty(deltas []float64) MetricUncertainty {
	switch len(deltas) {
	case 0:
		return MetricUncertainty{
			Method: "not_available",
			Note:   "no scored pairs were available for this metric",
		}
	case 1:
		return MetricUncertainty{
			Method: "not_available",
			Note:   "only one scored pair was available, so no interval could be estimated",
		}
	default:
		mean := meanFloat64(deltas)
		sd := sampleStandardDeviation(deltas, mean)
		margin := 1.96 * sd / math.Sqrt(float64(len(deltas)))
		return MetricUncertainty{
			Method:     "paired_mean_normal_approximation",
			Interval95: []float64{mean - margin, mean + margin},
			Note:       "interval uses a simple normal approximation over paired deltas",
		}
	}
}

func buildVariationNotes(grouping string, groups []GroupedMetricSet) []string {
	if len(groups) < 2 {
		return []string{}
	}

	notes := []string{}
	for _, metric := range []string{"quality_score", "resumability_score", "total_tokens", "wall_clock_seconds", "success"} {
		minDelta := math.Inf(1)
		maxDelta := math.Inf(-1)
		hasValue := false
		for _, group := range groups {
			summary, ok := group.Metrics[metric]
			if !ok || summary.SampleSize == 0 {
				continue
			}
			hasValue = true
			if summary.MeanDelta < minDelta {
				minDelta = summary.MeanDelta
			}
			if summary.MeanDelta > maxDelta {
				maxDelta = summary.MeanDelta
			}
		}
		if !hasValue || approxEqual(minDelta, maxDelta) {
			continue
		}
		notes = append(notes, fmt.Sprintf("graph effect for %s varies across %s groups (mean delta range %.3f to %.3f)", metric, grouping, minDelta, maxDelta))
	}
	return notes
}

func collectFindings(paired []PairedComparison, grouped []GroupedComparison, nullResults bool) []ReportFinding {
	findings := []ReportFinding{}
	for _, comparison := range paired {
		for _, metric := range []string{"success", "quality_score", "resumability_score", "total_tokens", "wall_clock_seconds"} {
			summary := comparison.Metrics[metric]
			if summary.SampleSize == 0 {
				continue
			}
			if classifyFinding(metric, summary.MeanDelta, nullResults) {
				reason := "graph matched baseline on this metric"
				if !nullResults {
					reason = "graph underperformed the baseline on this metric"
				}
				findings = append(findings, ReportFinding{
					Scope:           "paired_task",
					Metric:          metric,
					TaskID:          comparison.TaskID,
					TaskFamily:      comparison.TaskFamily,
					Model:           comparison.Model,
					SessionTopology: comparison.SessionTopology,
					MeanDelta:       summary.MeanDelta,
					SampleSize:      summary.SampleSize,
					Reason:          reason,
				})
			}
		}
	}

	for _, comparison := range grouped {
		for _, group := range comparison.Groups {
			for _, metric := range []string{"success", "quality_score", "resumability_score", "total_tokens", "wall_clock_seconds"} {
				summary := group.Metrics[metric]
				if summary.SampleSize == 0 {
					continue
				}
				if classifyFinding(metric, summary.MeanDelta, nullResults) {
					reason := "grouped graph effect was null for this metric"
					if !nullResults {
						reason = "grouped graph effect was negative for this metric"
					}
					findings = append(findings, ReportFinding{
						Scope:      "grouped_comparison",
						Metric:     metric,
						TaskFamily: comparison.TaskFamily,
						Grouping:   comparison.Grouping,
						GroupValue: group.GroupValue,
						MeanDelta:  summary.MeanDelta,
						SampleSize: summary.SampleSize,
						Reason:     reason,
					})
				}
			}
		}
	}

	return findings
}

func classifyFinding(metric string, meanDelta float64, nullResults bool) bool {
	if nullResults {
		return approxEqual(meanDelta, 0)
	}
	if metricHigherIsBetter(metric) {
		return meanDelta < 0
	}
	return meanDelta > 0
}

func countSparseComparisons(comparisons []PairedComparison) int {
	count := 0
	for _, comparison := range comparisons {
		if comparison.ScoredPairCount < 2 {
			count++
		}
	}
	return count
}

func countEvaluationArtifacts(runs []reportRun) int {
	count := 0
	for _, item := range runs {
		if item.evaluationCount > 0 {
			count++
		}
	}
	return count
}

func countScoredPairs(pairings []PairedRun) int {
	count := 0
	for _, pair := range pairings {
		if pair.Scored {
			count++
		}
	}
	return count
}

func pairCountForMetricValues(metricValues map[string][]pairedMetricValue) int {
	for _, metric := range []string{"success", "quality_score", "resumability_score", "total_tokens", "wall_clock_seconds"} {
		if values := metricValues[metric]; len(values) > 0 {
			return len(values)
		}
	}
	return 0
}

func sampleSizeForMetrics(metrics map[string]MetricSummary) int {
	for _, metric := range []string{"success", "quality_score", "resumability_score", "total_tokens", "wall_clock_seconds"} {
		if summary, ok := metrics[metric]; ok && summary.SampleSize > 0 {
			return summary.SampleSize
		}
	}
	return 0
}

func metricHigherIsBetter(metric string) bool {
	switch metric {
	case "total_tokens", "wall_clock_seconds":
		return false
	default:
		return true
	}
}

func metricUnit(metric string) string {
	switch metric {
	case "total_tokens":
		return "tokens"
	case "wall_clock_seconds":
		return "seconds"
	default:
		return "score"
	}
}

func interpretMetricDelta(metric string, meanDelta float64) string {
	if approxEqual(meanDelta, 0) {
		return "no_clear_difference"
	}
	if metricHigherIsBetter(metric) {
		if meanDelta > 0 {
			return "graph_better"
		}
		return "baseline_better"
	}
	if meanDelta < 0 {
		return "graph_better"
	}
	return "baseline_better"
}

func nextReportID(reportsPath string, now time.Time) string {
	base := "report-" + strings.ToLower(now.UTC().Format("20060102T150405Z"))
	candidate := base
	for index := 2; ; index++ {
		if _, err := os.Stat(filepath.Join(reportsPath, candidate+".json")); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%02d", base, index)
	}
}

func meanFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func sampleStandardDeviation(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		delta := value - mean
		total += delta * delta
	}
	return math.Sqrt(total / float64(len(values)-1))
}

func approxEqual(left, right float64) bool {
	return math.Abs(left-right) < 1e-9
}

func boolAsFloat(value *bool) float64 {
	if value == nil {
		return 0
	}
	if *value {
		return 1
	}
	return 0
}

func floatPointerValue(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func intMetric(telemetry *RunTelemetry, selector func(*RunTelemetry) *int) float64 {
	if telemetry == nil {
		return 0
	}
	value := selector(telemetry)
	if value == nil {
		return 0
	}
	return float64(*value)
}

func comparableTokenMetric(telemetry *RunTelemetry, selector func(*RunTelemetry) *int) (float64, bool) {
	if telemetry == nil || telemetry.MeasurementStatus != "complete" {
		return 0, false
	}
	value := selector(telemetry)
	if value == nil {
		return 0, false
	}
	return float64(*value), true
}

func sampleSizeForMetric(metricValues map[string][]pairedMetricValue, name string) int {
	return len(metricValues[name])
}

func countTokenSparseComparisons(comparisons []PairedComparison) int {
	count := 0
	for _, comparison := range comparisons {
		metric := comparison.Metrics["total_tokens"]
		if metric.SampleSize < comparison.ScoredPairCount {
			count++
		}
	}
	return count
}

func buildVerificationObservation(graphRun, baseRun reportRun) (verificationObservation, bool) {
	if graphRun.workflowStart == nil {
		return verificationObservation{}, false
	}
	if graphRun.workflowStart.Kickoff.Task.Family != workflow.KickoffFamilyVerificationAudit {
		return verificationObservation{}, false
	}
	profile := strings.TrimSpace(graphRun.workflowStart.Kickoff.Task.Subprofile)
	if profile == "" {
		profile = strings.TrimSpace(graphRun.workflowStart.Kickoff.Policy.Subprofile)
	}
	if profile == "" {
		profile = "unspecified"
	}
	observation := verificationObservation{
		profile:                       profile,
		effectiveMode:                 graphRun.workflowStart.Kickoff.Advisory.EffectiveMode,
		confidenceScore:               graphRun.workflowStart.Kickoff.Advisory.ConfidenceScore,
		abstained:                     graphRun.workflowStart.Kickoff.Context.Abstained,
		baselinePromptMetadataPresent: len(baseRun.baselinePromptMetadata) > 0,
	}
	if tokenDelta, ok := pairedComparableTokenDelta(graphRun.record.Telemetry, baseRun.record.Telemetry); ok {
		observation.tokenDelta = &tokenDelta
	}
	if qualityDelta, ok := pairedQualityDelta(graphRun.latestEvaluation, baseRun.latestEvaluation); ok {
		observation.qualityDelta = &qualityDelta
	}
	return observation, true
}

func pairedComparableTokenDelta(graphTelemetry, baseTelemetry *RunTelemetry) (float64, bool) {
	graphTotal, graphOK := comparableTokenMetric(graphTelemetry, func(t *RunTelemetry) *int { return t.TotalTokens })
	baseTotal, baseOK := comparableTokenMetric(baseTelemetry, func(t *RunTelemetry) *int { return t.TotalTokens })
	if !graphOK || !baseOK {
		return 0, false
	}
	return graphTotal - baseTotal, true
}

func pairedQualityDelta(graphEvaluation, baseEvaluation *EvaluationRecord) (float64, bool) {
	if graphEvaluation == nil || baseEvaluation == nil || graphEvaluation.Scores == nil || baseEvaluation.Scores == nil {
		return 0, false
	}
	if graphEvaluation.Scores.QualityScore == nil || baseEvaluation.Scores.QualityScore == nil {
		return 0, false
	}
	return *graphEvaluation.Scores.QualityScore - *baseEvaluation.Scores.QualityScore, true
}

func buildVerificationAttribution(observations []verificationObservation) VerificationAttributionSummary {
	if len(observations) == 0 {
		return VerificationAttributionSummary{
			Gate: VerificationRerunGate{
				Decision: "not_applicable",
				Rule:     "A verification rerun gate is only computed when selected runs include verification-family kickoff artifacts.",
			},
		}
	}

	grouped := map[string][]verificationObservation{}
	for _, observation := range observations {
		grouped[observation.profile] = append(grouped[observation.profile], observation)
	}
	profiles := make([]string, 0, len(grouped))
	for profile := range grouped {
		profiles = append(profiles, profile)
	}
	sort.Strings(profiles)

	summaries := make([]VerificationProfileSummary, 0, len(profiles))
	for _, profile := range profiles {
		items := grouped[profile]
		effectiveModes := map[string]int{}
		confidenceTotal := 0.0
		abstainedCount := 0
		tokenDeltas := []float64{}
		qualityDeltas := []float64{}
		baselinePromptMetadataPairs := 0
		for _, item := range items {
			effectiveModes[item.effectiveMode]++
			confidenceTotal += item.confidenceScore
			if item.abstained {
				abstainedCount++
			}
			if item.tokenDelta != nil {
				tokenDeltas = append(tokenDeltas, *item.tokenDelta)
			}
			if item.qualityDelta != nil {
				qualityDeltas = append(qualityDeltas, *item.qualityDelta)
			}
			if item.baselinePromptMetadataPresent {
				baselinePromptMetadataPairs++
			}
		}
		summaries = append(summaries, VerificationProfileSummary{
			Profile:                     profile,
			PairCount:                   len(items),
			TokenSampleCount:            len(tokenDeltas),
			QualitySampleCount:          len(qualityDeltas),
			EffectiveModes:              effectiveModes,
			AbstentionRate:              meanFloat64FromObservations(float64(abstainedCount), len(items)),
			MeanConfidence:              meanFloat64FromObservations(confidenceTotal, len(items)),
			MeanTokenDelta:              meanFloat64(tokenDeltas),
			MeanQualityDelta:            meanFloat64(qualityDeltas),
			BaselinePromptMetadataPairs: baselinePromptMetadataPairs,
		})
	}
	return VerificationAttributionSummary{
		Profiles: summaries,
		Gate:     buildVerificationGate(summaries),
	}
}

func meanFloat64FromObservations(total float64, count int) float64 {
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func buildVerificationGate(summaries []VerificationProfileSummary) VerificationRerunGate {
	rule := "Proceed only when verification profiles have comparable token telemetry, remain roughly neutral on token delta, and do not show quality regression."
	if len(summaries) == 0 {
		return VerificationRerunGate{Decision: "not_applicable", Rule: rule}
	}

	reasons := []string{}
	overallTokenDeltas := []float64{}
	hasIncompleteMetrics := false
	hasPositiveTokenRegression := false
	hasCatastrophicRegression := false
	hasQualityRegression := false
	for _, summary := range summaries {
		if summary.TokenSampleCount == 0 || summary.QualitySampleCount == 0 {
			hasIncompleteMetrics = true
			reasons = append(reasons, fmt.Sprintf("profile %s lacks comparable token or quality samples", summary.Profile))
			continue
		}
		overallTokenDeltas = append(overallTokenDeltas, summary.MeanTokenDelta)
		if summary.MeanTokenDelta > 25000 {
			hasPositiveTokenRegression = true
			reasons = append(reasons, fmt.Sprintf("profile %s still shows a positive token delta of %.0f", summary.Profile, summary.MeanTokenDelta))
		}
		if summary.MeanTokenDelta > 100000 || summary.MeanQualityDelta < -0.10 {
			hasCatastrophicRegression = true
			reasons = append(reasons, fmt.Sprintf("profile %s remains materially regressed", summary.Profile))
		}
		if summary.MeanQualityDelta < 0 {
			hasQualityRegression = true
			reasons = append(reasons, fmt.Sprintf("profile %s still loses quality versus baseline", summary.Profile))
		}
	}

	switch {
	case hasCatastrophicRegression:
		return VerificationRerunGate{Decision: "stop-and-recalibrate", Rule: rule, Reasons: dedupeReportStrings(reasons)}
	case hasIncompleteMetrics:
		return VerificationRerunGate{Decision: "hold", Rule: rule, Reasons: dedupeReportStrings(reasons)}
	case meanFloat64(overallTokenDeltas) <= 25000 && !hasPositiveTokenRegression && !hasQualityRegression:
		return VerificationRerunGate{
			Decision: "proceed",
			Rule:     rule,
			Reasons:  []string{"Verification profiles stayed roughly neutral on token delta and did not regress quality."},
		}
	default:
		return VerificationRerunGate{Decision: "hold", Rule: rule, Reasons: dedupeReportStrings(reasons)}
	}
}

func dedupeReportStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	deduped := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		deduped = append(deduped, value)
	}
	return deduped
}
