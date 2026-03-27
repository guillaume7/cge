package labcmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type repoDogfoodingBaseline struct {
	DocumentationRef  string `json:"documentation_ref"`
	SuiteManifestPath string `json:"suite_manifest_path"`
	ConditionsPath    string `json:"conditions_manifest_path"`
	Tasks             []struct {
		TaskID                string `json:"task_id"`
		AcceptanceCriteriaRef string `json:"acceptance_criteria_ref"`
	} `json:"tasks"`
	BaselineArtifacts struct {
		BatchPlanPath   string   `json:"batch_plan_path"`
		RunIDs          []string `json:"run_ids"`
		EvaluationPaths []string `json:"evaluation_paths"`
		ReportPath      string   `json:"report_path"`
	} `json:"baseline_artifacts"`
	Limitations []string `json:"limitations"`
}

type repoDogfoodingSuite struct {
	Tasks []struct {
		TaskID                string `json:"task_id"`
		AcceptanceCriteriaRef string `json:"acceptance_criteria_ref"`
	} `json:"tasks"`
}

type repoDogfoodingConditions struct {
	Conditions []struct {
		ConditionID string `json:"condition_id"`
	} `json:"conditions"`
}

type repoDogfoodingReport struct {
	PairedComparisons []any    `json:"paired_comparisons"`
	Limitations       []string `json:"limitations"`
	Summary           struct {
		NegativeResults []any `json:"negative_results"`
	} `json:"summary"`
}

func TestRepoDogfoodingHarnessIncludesTwoRepoTasksAndArtifactRefs(t *testing.T) {
	t.Parallel()

	repoRoot := repoDogfoodingRepoRoot(t)
	baseline := readJSON[repoDogfoodingBaseline](t, filepath.Join(repoRoot, ".graph", "lab", "dogfooding", "baseline-v1.json"))
	suite := readJSON[repoDogfoodingSuite](t, filepath.Join(repoRoot, baseline.SuiteManifestPath))
	conditions := readJSON[repoDogfoodingConditions](t, filepath.Join(repoRoot, baseline.ConditionsPath))

	if len(suite.Tasks) < 2 {
		t.Fatalf("suite task count = %d, want at least 2 repo tasks", len(suite.Tasks))
	}
	for _, task := range suite.Tasks {
		if task.AcceptanceCriteriaRef == "" {
			t.Fatalf("task %#v missing acceptance criteria ref", task)
		}
		if !strings.HasPrefix(task.AcceptanceCriteriaRef, "docs/") {
			t.Fatalf("acceptance_criteria_ref = %q, want repo-local docs path", task.AcceptanceCriteriaRef)
		}
		if _, err := os.Stat(filepath.Join(repoRoot, task.AcceptanceCriteriaRef)); err != nil {
			t.Fatalf("acceptance criteria ref %s does not exist: %v", task.AcceptanceCriteriaRef, err)
		}
	}

	conditionIDs := map[string]bool{}
	for _, condition := range conditions.Conditions {
		conditionIDs[condition.ConditionID] = true
	}
	for _, required := range []string{"with-graph", "without-graph"} {
		if !conditionIDs[required] {
			t.Fatalf("condition_ids = %#v, missing %q", conditionIDs, required)
		}
	}

	if len(baseline.BaselineArtifacts.RunIDs) != 4 {
		t.Fatalf("run_ids = %#v, want four baseline runs", baseline.BaselineArtifacts.RunIDs)
	}
	for _, path := range append([]string{baseline.BaselineArtifacts.BatchPlanPath, baseline.BaselineArtifacts.ReportPath}, baseline.BaselineArtifacts.EvaluationPaths...) {
		if _, err := os.Stat(filepath.Join(repoRoot, path)); err != nil {
			t.Fatalf("baseline artifact %s does not exist: %v", path, err)
		}
	}
}

func TestRepoDogfoodingDocumentationExplainsLifecycleAndLimitations(t *testing.T) {
	t.Parallel()

	repoRoot := repoDogfoodingRepoRoot(t)
	baseline := readJSON[repoDogfoodingBaseline](t, filepath.Join(repoRoot, ".graph", "lab", "dogfooding", "baseline-v1.json"))
	payload, err := os.ReadFile(filepath.Join(repoRoot, baseline.DocumentationRef))
	if err != nil {
		t.Fatalf("os.ReadFile(%s): %v", baseline.DocumentationRef, err)
	}
	text := string(payload)

	requiredSnippets := []string{
		"graph lab init",
		"graph lab run",
		"graph lab evaluate",
		"graph lab report",
		"illustrative baseline",
		"tiny",
		"not be presented as proof",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(text, snippet) {
			t.Fatalf("documentation missing %q\n%s", snippet, text)
		}
	}
}

func TestRepoDogfoodingBaselineReportSurfacesPairedEvidenceAndExplicitLimitations(t *testing.T) {
	t.Parallel()

	repoRoot := repoDogfoodingRepoRoot(t)
	baseline := readJSON[repoDogfoodingBaseline](t, filepath.Join(repoRoot, ".graph", "lab", "dogfooding", "baseline-v1.json"))
	report := readJSON[repoDogfoodingReport](t, filepath.Join(repoRoot, baseline.BaselineArtifacts.ReportPath))

	if len(report.PairedComparisons) < 2 {
		t.Fatalf("paired comparisons = %d, want at least 2", len(report.PairedComparisons))
	}
	if len(report.Summary.NegativeResults) == 0 {
		t.Fatal("expected report to surface at least one negative result for honesty")
	}
	if len(report.Limitations) == 0 {
		t.Fatal("expected report to include explicit limitations")
	}

	joined := strings.ToLower(strings.Join(report.Limitations, " "))
	if !strings.Contains(joined, "uncertain") && !strings.Contains(joined, "fewer than two") {
		t.Fatalf("limitations = %#v, want explicit small-sample warning", report.Limitations)
	}
}

func repoDogfoodingRepoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func readJSON[T any](t *testing.T, path string) T {
	t.Helper()

	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%s): %v", path, err)
	}
	var value T
	if err := json.Unmarshal(payload, &value); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", path, err)
	}
	return value
}
