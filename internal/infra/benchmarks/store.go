package benchmarks

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/guillaume-galp/cge/internal/infra/repo"
)

const (
	SchemaVersion = "v1"

	ScenarioArtifactKind = "delegated_workflow_benchmark_scenario"
	RunArtifactKind      = "delegated_workflow_benchmark_run"

	ModeWithGraph    = "with_graph"
	ModeWithoutGraph = "without_graph"
)

var artifactIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

type ValidationError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
	Err     error          `json:"-"`
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type Store struct {
	now func() time.Time
}

type Scenario struct {
	SchemaVersion         string         `json:"schema_version"`
	Kind                  string         `json:"kind"`
	ScenarioID            string         `json:"scenario_id"`
	TaskFamily            string         `json:"task_family"`
	Title                 string         `json:"title,omitempty"`
	AcceptanceCriteriaRef string         `json:"acceptance_criteria_ref,omitempty"`
	CreatedAt             string         `json:"created_at,omitempty"`
	Modes                 []ScenarioMode `json:"modes"`
}

type ScenarioMode struct {
	Mode       string   `json:"mode"`
	TaskPrompt string   `json:"task_prompt"`
	Notes      []string `json:"notes,omitempty"`
}

type ScenarioArtifact struct {
	Path     string   `json:"path"`
	Scenario Scenario `json:"scenario"`
}

type RunReport struct {
	SchemaVersion string            `json:"schema_version"`
	Kind          string            `json:"kind"`
	RunID         string            `json:"run_id"`
	ScenarioID    string            `json:"scenario_id"`
	Mode          string            `json:"mode"`
	RecordedAt    string            `json:"recorded_at,omitempty"`
	Metrics       RunMetrics        `json:"metrics"`
	Notes         []string          `json:"notes,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
}

type RunMetrics struct {
	Volume      VolumeMetrics      `json:"volume"`
	Orientation OrientationMetrics `json:"orientation"`
	Outcome     OutcomeSignals     `json:"outcome"`
}

type VolumeMetrics struct {
	InputTokens      int `json:"input_tokens,omitempty"`
	OutputTokens     int `json:"output_tokens,omitempty"`
	PromptCount      int `json:"prompt_count,omitempty"`
	PromptCharacters int `json:"prompt_characters,omitempty"`
}

type OrientationMetrics struct {
	StepCount       int `json:"step_count"`
	RepoScans       int `json:"repo_scans,omitempty"`
	FollowUpPrompts int `json:"follow_up_prompts,omitempty"`
	ContextReloads  int `json:"context_reloads,omitempty"`
}

type OutcomeSignals struct {
	QualityRating          string `json:"quality_rating,omitempty"`
	ResumabilityRating     string `json:"resumability_rating,omitempty"`
	AcceptanceChecksPassed int    `json:"acceptance_checks_passed,omitempty"`
	AcceptanceChecksTotal  int    `json:"acceptance_checks_total,omitempty"`
}

type RunArtifact struct {
	Path   string    `json:"path"`
	Report RunReport `json:"report"`
}

func NewStore() *Store {
	return &Store{
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (s *Store) NowForTest(now func() time.Time) {
	if s == nil {
		return
	}
	s.now = now
}

func (s *Store) WriteScenario(workspace repo.Workspace, scenario Scenario) (ScenarioArtifact, error) {
	if s == nil {
		return ScenarioArtifact{}, &Error{
			Code:    "benchmark_store_unavailable",
			Message: "benchmark store is not configured",
		}
	}
	if s.now == nil {
		s.now = func() time.Time { return time.Now().UTC() }
	}

	scenario, err := normalizeScenario(scenario, s.now())
	if err != nil {
		return ScenarioArtifact{}, err
	}

	path := scenarioPath(workspace, scenario.ScenarioID)
	if err := writeJSON(path, scenario); err != nil {
		return ScenarioArtifact{}, classifyWriteError(path, "scenario", err)
	}

	return ScenarioArtifact{
		Path:     path,
		Scenario: scenario,
	}, nil
}

func (s *Store) ReadScenario(workspace repo.Workspace, scenarioID string) (ScenarioArtifact, error) {
	scenarioID = strings.TrimSpace(scenarioID)
	if err := validateArtifactID("scenario_id", scenarioID); err != nil {
		return ScenarioArtifact{}, err
	}

	path := scenarioPath(workspace, scenarioID)
	payload, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ScenarioArtifact{}, &ValidationError{
				Code:    "benchmark_scenario_not_found",
				Message: "benchmark scenario does not exist",
				Details: map[string]any{
					"scenario_id": scenarioID,
					"path":        filepath.ToSlash(path),
				},
			}
		}
		return ScenarioArtifact{}, classifyReadError(path, "scenario", err)
	}

	var scenario Scenario
	if err := json.Unmarshal(payload, &scenario); err != nil {
		return ScenarioArtifact{}, classifyReadError(path, "scenario", fmt.Errorf("parse benchmark scenario: %w", err))
	}
	scenario, err = normalizeScenario(scenario, time.Time{})
	if err != nil {
		return ScenarioArtifact{}, err
	}

	return ScenarioArtifact{
		Path:     path,
		Scenario: scenario,
	}, nil
}

func (s *Store) LoadScenario(workspace repo.Workspace, scenarioID string) (ScenarioArtifact, error) {
	scenarioID = strings.TrimSpace(scenarioID)
	if err := validateArtifactID("scenario_id", scenarioID); err != nil {
		return ScenarioArtifact{}, err
	}

	path := scenarioPath(workspace, scenarioID)
	scenario, err := loadScenarioArtifact(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ScenarioArtifact{}, &ValidationError{
				Code:    "benchmark_scenario_not_found",
				Message: "benchmark scenario does not exist",
				Details: map[string]any{
					"scenario_id": scenarioID,
					"path":        filepath.ToSlash(path),
				},
			}
		}
		return ScenarioArtifact{}, classifyReadError(path, "scenario", err)
	}

	return ScenarioArtifact{
		Path:     path,
		Scenario: scenario,
	}, nil
}

func (s *Store) WriteRun(workspace repo.Workspace, scenario Scenario, report RunReport) (RunArtifact, error) {
	if s == nil {
		return RunArtifact{}, &Error{
			Code:    "benchmark_store_unavailable",
			Message: "benchmark store is not configured",
		}
	}
	if s.now == nil {
		s.now = func() time.Time { return time.Now().UTC() }
	}

	scenario, err := normalizeScenario(scenario, time.Time{})
	if err != nil {
		return RunArtifact{}, err
	}
	report, err = normalizeRunReport(report, scenario, s.now())
	if err != nil {
		return RunArtifact{}, err
	}

	path := runPath(workspace, report.ScenarioID, report.Mode, report.RunID)
	if err := writeJSON(path, report); err != nil {
		return RunArtifact{}, classifyWriteError(path, "run_report", err)
	}

	return RunArtifact{
		Path:   path,
		Report: report,
	}, nil
}

func (s *Store) ListScenarios(workspace repo.Workspace) ([]ScenarioArtifact, error) {
	dir := filepath.Join(workspace.WorkspacePath, repo.BenchmarksDirName, "scenarios")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []ScenarioArtifact{}, nil
		}
		return nil, classifyReadError(dir, "scenario", err)
	}

	artifacts := make([]ScenarioArtifact, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		scenario, err := loadScenarioArtifact(path)
		if err != nil {
			return nil, classifyReadError(path, "scenario", err)
		}
		artifacts = append(artifacts, ScenarioArtifact{
			Path:     path,
			Scenario: scenario,
		})
	}

	slices.SortFunc(artifacts, func(a, b ScenarioArtifact) int {
		aKey := scenarioSortKey(a)
		bKey := scenarioSortKey(b)
		switch {
		case aKey < bKey:
			return -1
		case aKey > bKey:
			return 1
		default:
			return 0
		}
	})

	return artifacts, nil
}

func (s *Store) ListRuns(workspace repo.Workspace, scenarioID string) ([]RunArtifact, error) {
	scenarioID = strings.TrimSpace(scenarioID)
	if err := validateArtifactID("scenario_id", scenarioID); err != nil {
		return nil, err
	}

	dir := filepath.Join(workspace.WorkspacePath, repo.BenchmarksDirName, "runs", scenarioID)
	runPaths := []string{}
	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".json" {
			return nil
		}
		runPaths = append(runPaths, path)
		return nil
	}); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []RunArtifact{}, nil
		}
		return nil, classifyReadError(dir, "run_report", err)
	}

	slices.Sort(runPaths)
	artifacts := make([]RunArtifact, 0, len(runPaths))
	for _, path := range runPaths {
		report, err := loadRunArtifact(path)
		if err != nil {
			return nil, classifyReadError(path, "run_report", err)
		}
		artifacts = append(artifacts, RunArtifact{
			Path:   path,
			Report: report,
		})
	}

	slices.SortFunc(artifacts, func(a, b RunArtifact) int {
		switch {
		case a.Report.Mode < b.Report.Mode:
			return -1
		case a.Report.Mode > b.Report.Mode:
			return 1
		case a.Report.RecordedAt < b.Report.RecordedAt:
			return -1
		case a.Report.RecordedAt > b.Report.RecordedAt:
			return 1
		case a.Report.RunID < b.Report.RunID:
			return -1
		case a.Report.RunID > b.Report.RunID:
			return 1
		case a.Path < b.Path:
			return -1
		case a.Path > b.Path:
			return 1
		default:
			return 0
		}
	})

	return artifacts, nil
}

func normalizeScenario(scenario Scenario, now time.Time) (Scenario, error) {
	scenario.SchemaVersion = firstNonEmpty(strings.TrimSpace(scenario.SchemaVersion), SchemaVersion)
	scenario.Kind = firstNonEmpty(strings.TrimSpace(scenario.Kind), ScenarioArtifactKind)
	scenario.ScenarioID = strings.TrimSpace(scenario.ScenarioID)
	scenario.TaskFamily = strings.TrimSpace(scenario.TaskFamily)
	scenario.Title = strings.TrimSpace(scenario.Title)
	scenario.AcceptanceCriteriaRef = strings.TrimSpace(scenario.AcceptanceCriteriaRef)
	scenario.CreatedAt = strings.TrimSpace(scenario.CreatedAt)

	if scenario.SchemaVersion != SchemaVersion {
		return Scenario{}, &ValidationError{
			Code:    "invalid_benchmark_scenario",
			Message: "benchmark scenario schema_version is unsupported",
			Details: map[string]any{
				"schema_version": scenario.SchemaVersion,
				"supported":      SchemaVersion,
			},
		}
	}
	if scenario.Kind != ScenarioArtifactKind {
		return Scenario{}, &ValidationError{
			Code:    "invalid_benchmark_scenario",
			Message: "benchmark scenario kind is unsupported",
			Details: map[string]any{
				"kind":      scenario.Kind,
				"supported": ScenarioArtifactKind,
			},
		}
	}
	if err := validateArtifactID("scenario_id", scenario.ScenarioID); err != nil {
		return Scenario{}, err
	}
	if scenario.TaskFamily == "" {
		return Scenario{}, &ValidationError{
			Code:    "invalid_benchmark_scenario",
			Message: "benchmark scenario task_family is required",
			Details: map[string]any{
				"scenario_id": scenario.ScenarioID,
				"field":       "task_family",
			},
		}
	}

	if len(scenario.Modes) != 2 {
		return Scenario{}, &ValidationError{
			Code:    "invalid_benchmark_scenario",
			Message: "benchmark scenario must define exactly two comparable workflow modes",
			Details: map[string]any{
				"scenario_id": scenario.ScenarioID,
				"required":    []string{ModeWithGraph, ModeWithoutGraph},
				"mode_count":  len(scenario.Modes),
			},
		}
	}

	normalizedModes := make([]ScenarioMode, 0, len(scenario.Modes))
	seenModes := map[string]struct{}{}
	for _, mode := range scenario.Modes {
		mode.Mode = normalizeMode(mode.Mode)
		mode.TaskPrompt = strings.TrimSpace(mode.TaskPrompt)
		mode.Notes = normalizeNotes(mode.Notes)

		if mode.Mode != ModeWithGraph && mode.Mode != ModeWithoutGraph {
			return Scenario{}, &ValidationError{
				Code:    "invalid_benchmark_scenario",
				Message: "benchmark scenario mode is unsupported",
				Details: map[string]any{
					"scenario_id": scenario.ScenarioID,
					"mode":        mode.Mode,
				},
			}
		}
		if _, ok := seenModes[mode.Mode]; ok {
			return Scenario{}, &ValidationError{
				Code:    "invalid_benchmark_scenario",
				Message: "benchmark scenario mode must be unique",
				Details: map[string]any{
					"scenario_id": scenario.ScenarioID,
					"mode":        mode.Mode,
				},
			}
		}
		seenModes[mode.Mode] = struct{}{}
		if mode.TaskPrompt == "" {
			return Scenario{}, &ValidationError{
				Code:    "invalid_benchmark_scenario",
				Message: "benchmark scenario mode requires a task_prompt",
				Details: map[string]any{
					"scenario_id": scenario.ScenarioID,
					"mode":        mode.Mode,
					"field":       "task_prompt",
				},
			}
		}
		normalizedModes = append(normalizedModes, mode)
	}

	for _, requiredMode := range []string{ModeWithGraph, ModeWithoutGraph} {
		if _, ok := seenModes[requiredMode]; !ok {
			return Scenario{}, &ValidationError{
				Code:    "invalid_benchmark_scenario",
				Message: "benchmark scenario must include with_graph and without_graph modes",
				Details: map[string]any{
					"scenario_id": scenario.ScenarioID,
					"missing":     requiredMode,
				},
			}
		}
	}

	slices.SortFunc(normalizedModes, func(a, b ScenarioMode) int {
		switch {
		case a.Mode < b.Mode:
			return -1
		case a.Mode > b.Mode:
			return 1
		default:
			return 0
		}
	})
	scenario.Modes = normalizedModes
	if scenario.CreatedAt == "" && !now.IsZero() {
		scenario.CreatedAt = now.Format(time.RFC3339)
	}
	return scenario, nil
}

func normalizeRunReport(report RunReport, scenario Scenario, now time.Time) (RunReport, error) {
	report.SchemaVersion = firstNonEmpty(strings.TrimSpace(report.SchemaVersion), SchemaVersion)
	report.Kind = firstNonEmpty(strings.TrimSpace(report.Kind), RunArtifactKind)
	report.RunID = strings.TrimSpace(report.RunID)
	report.ScenarioID = strings.TrimSpace(report.ScenarioID)
	report.Mode = normalizeMode(report.Mode)
	report.RecordedAt = strings.TrimSpace(report.RecordedAt)
	report.Notes = normalizeNotes(report.Notes)

	if report.SchemaVersion != SchemaVersion {
		return RunReport{}, &ValidationError{
			Code:    "invalid_benchmark_report",
			Message: "benchmark report schema_version is unsupported",
			Details: map[string]any{
				"schema_version": report.SchemaVersion,
				"supported":      SchemaVersion,
			},
		}
	}
	if report.Kind != RunArtifactKind {
		return RunReport{}, &ValidationError{
			Code:    "invalid_benchmark_report",
			Message: "benchmark report kind is unsupported",
			Details: map[string]any{
				"kind":      report.Kind,
				"supported": RunArtifactKind,
			},
		}
	}
	if err := validateArtifactID("run_id", report.RunID); err != nil {
		return RunReport{}, err
	}
	if report.ScenarioID == "" {
		report.ScenarioID = scenario.ScenarioID
	}
	if report.ScenarioID != scenario.ScenarioID {
		return RunReport{}, &ValidationError{
			Code:    "invalid_benchmark_report",
			Message: "benchmark report scenario_id does not match the stored scenario",
			Details: map[string]any{
				"scenario_id":       report.ScenarioID,
				"expected_scenario": scenario.ScenarioID,
			},
		}
	}
	if report.Mode != ModeWithGraph && report.Mode != ModeWithoutGraph {
		return RunReport{}, &ValidationError{
			Code:    "invalid_benchmark_report",
			Message: "benchmark report mode is unsupported",
			Details: map[string]any{
				"scenario_id": report.ScenarioID,
				"mode":        report.Mode,
			},
		}
	}
	if !scenarioSupportsMode(scenario, report.Mode) {
		return RunReport{}, &ValidationError{
			Code:    "invalid_benchmark_report",
			Message: "benchmark report mode is not defined by the stored scenario",
			Details: map[string]any{
				"scenario_id": report.ScenarioID,
				"mode":        report.Mode,
			},
		}
	}

	volume := report.Metrics.Volume
	if volume.InputTokens < 0 || volume.OutputTokens < 0 || volume.PromptCount < 0 || volume.PromptCharacters < 0 {
		return RunReport{}, &ValidationError{
			Code:    "invalid_benchmark_report",
			Message: "benchmark report volume metrics must be zero or greater",
			Details: map[string]any{
				"scenario_id": report.ScenarioID,
				"mode":        report.Mode,
			},
		}
	}
	if volume.InputTokens+volume.OutputTokens == 0 && volume.PromptCount == 0 && volume.PromptCharacters == 0 {
		return RunReport{}, &ValidationError{
			Code:    "invalid_benchmark_report",
			Message: "benchmark report requires token or prompt volume metrics",
			Details: map[string]any{
				"scenario_id": report.ScenarioID,
				"mode":        report.Mode,
				"field":       "metrics.volume",
			},
		}
	}

	orientation := report.Metrics.Orientation
	if orientation.StepCount <= 0 || orientation.RepoScans < 0 || orientation.FollowUpPrompts < 0 || orientation.ContextReloads < 0 {
		return RunReport{}, &ValidationError{
			Code:    "invalid_benchmark_report",
			Message: "benchmark report requires non-empty orientation effort metrics",
			Details: map[string]any{
				"scenario_id": report.ScenarioID,
				"mode":        report.Mode,
				"field":       "metrics.orientation",
			},
		}
	}

	outcome := report.Metrics.Outcome
	outcome.QualityRating = strings.TrimSpace(outcome.QualityRating)
	outcome.ResumabilityRating = strings.TrimSpace(outcome.ResumabilityRating)
	if outcome.QualityRating == "" && outcome.ResumabilityRating == "" {
		return RunReport{}, &ValidationError{
			Code:    "invalid_benchmark_report",
			Message: "benchmark report requires quality or resumability signals",
			Details: map[string]any{
				"scenario_id": report.ScenarioID,
				"mode":        report.Mode,
				"field":       "metrics.outcome",
			},
		}
	}
	if outcome.AcceptanceChecksPassed < 0 || outcome.AcceptanceChecksTotal < 0 || outcome.AcceptanceChecksPassed > outcome.AcceptanceChecksTotal {
		return RunReport{}, &ValidationError{
			Code:    "invalid_benchmark_report",
			Message: "benchmark report acceptance check counts are invalid",
			Details: map[string]any{
				"scenario_id": report.ScenarioID,
				"mode":        report.Mode,
				"field":       "metrics.outcome.acceptance_checks",
			},
		}
	}
	report.Metrics.Outcome = outcome

	if report.RecordedAt == "" && !now.IsZero() {
		report.RecordedAt = now.Format(time.RFC3339)
	}
	return report, nil
}

func scenarioSupportsMode(scenario Scenario, mode string) bool {
	for _, candidate := range scenario.Modes {
		if candidate.Mode == mode {
			return true
		}
	}
	return false
}

func scenarioPath(workspace repo.Workspace, scenarioID string) string {
	return filepath.Join(workspace.WorkspacePath, repo.BenchmarksDirName, "scenarios", scenarioID+".json")
}

func runPath(workspace repo.Workspace, scenarioID, mode, runID string) string {
	return filepath.Join(workspace.WorkspacePath, repo.BenchmarksDirName, "runs", scenarioID, mode, runID+".json")
}

func writeJSON(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func normalizeMode(mode string) string {
	return strings.ToLower(strings.TrimSpace(mode))
}

func normalizeNotes(notes []string) []string {
	if len(notes) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(notes))
	for _, note := range notes {
		note = strings.TrimSpace(note)
		if note == "" {
			continue
		}
		normalized = append(normalized, note)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func validateArtifactID(field, value string) error {
	if value == "" {
		return &ValidationError{
			Code:    "invalid_benchmark_artifact_id",
			Message: "benchmark artifact identifier is required",
			Details: map[string]any{
				"field": field,
			},
		}
	}
	if !artifactIDPattern.MatchString(value) {
		return &ValidationError{
			Code:    "invalid_benchmark_artifact_id",
			Message: "benchmark artifact identifier must use safe path characters",
			Details: map[string]any{
				"field": field,
				"value": value,
			},
		}
	}
	return nil
}

func classifyWriteError(path, artifactType string, err error) error {
	return &Error{
		Code:    "benchmark_artifact_write_failed",
		Message: "benchmark artifact could not be written",
		Details: map[string]any{
			"path":     filepath.ToSlash(path),
			"artifact": artifactType,
		},
		Err: err,
	}
}

func classifyReadError(path, artifactType string, err error) error {
	return &Error{
		Code:    "benchmark_artifact_read_failed",
		Message: "benchmark artifact could not be loaded",
		Details: map[string]any{
			"path":     filepath.ToSlash(path),
			"artifact": artifactType,
		},
		Err: err,
	}
}

func loadScenarioArtifact(path string) (Scenario, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return Scenario{}, err
	}

	var scenario Scenario
	if err := json.Unmarshal(payload, &scenario); err != nil {
		return Scenario{}, fmt.Errorf("parse benchmark scenario: %w", err)
	}

	return normalizeScenarioLoose(scenario), nil
}

func loadRunArtifact(path string) (RunReport, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return RunReport{}, err
	}

	var report RunReport
	if err := json.Unmarshal(payload, &report); err != nil {
		return RunReport{}, fmt.Errorf("parse benchmark run report: %w", err)
	}

	return normalizeRunReportLoose(report), nil
}

func normalizeScenarioLoose(scenario Scenario) Scenario {
	scenario.SchemaVersion = strings.TrimSpace(scenario.SchemaVersion)
	scenario.Kind = strings.TrimSpace(scenario.Kind)
	scenario.ScenarioID = strings.TrimSpace(scenario.ScenarioID)
	scenario.TaskFamily = strings.TrimSpace(scenario.TaskFamily)
	scenario.Title = strings.TrimSpace(scenario.Title)
	scenario.AcceptanceCriteriaRef = strings.TrimSpace(scenario.AcceptanceCriteriaRef)
	scenario.CreatedAt = strings.TrimSpace(scenario.CreatedAt)

	if len(scenario.Modes) == 0 {
		scenario.Modes = []ScenarioMode{}
		return scenario
	}

	modes := make([]ScenarioMode, 0, len(scenario.Modes))
	for _, mode := range scenario.Modes {
		mode.Mode = normalizeMode(mode.Mode)
		mode.TaskPrompt = strings.TrimSpace(mode.TaskPrompt)
		mode.Notes = normalizeNotes(mode.Notes)
		modes = append(modes, mode)
	}
	slices.SortFunc(modes, func(a, b ScenarioMode) int {
		switch {
		case a.Mode < b.Mode:
			return -1
		case a.Mode > b.Mode:
			return 1
		case a.TaskPrompt < b.TaskPrompt:
			return -1
		case a.TaskPrompt > b.TaskPrompt:
			return 1
		default:
			return 0
		}
	})
	scenario.Modes = modes
	return scenario
}

func normalizeRunReportLoose(report RunReport) RunReport {
	report.SchemaVersion = strings.TrimSpace(report.SchemaVersion)
	report.Kind = strings.TrimSpace(report.Kind)
	report.RunID = strings.TrimSpace(report.RunID)
	report.ScenarioID = strings.TrimSpace(report.ScenarioID)
	report.Mode = normalizeMode(report.Mode)
	report.RecordedAt = strings.TrimSpace(report.RecordedAt)
	report.Notes = normalizeNotes(report.Notes)
	report.Labels = normalizeLabels(report.Labels)
	report.Metrics.Outcome.QualityRating = strings.TrimSpace(report.Metrics.Outcome.QualityRating)
	report.Metrics.Outcome.ResumabilityRating = strings.TrimSpace(report.Metrics.Outcome.ResumabilityRating)
	return report
}

func normalizeLabels(labels map[string]string) map[string]string {
	if len(labels) == 0 {
		return nil
	}

	normalized := map[string]string{}
	for key, value := range labels {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		normalized[key] = value
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func scenarioSortKey(artifact ScenarioArtifact) string {
	if artifact.Scenario.ScenarioID != "" {
		return artifact.Scenario.ScenarioID
	}
	return strings.TrimSuffix(filepath.Base(artifact.Path), filepath.Ext(artifact.Path))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
