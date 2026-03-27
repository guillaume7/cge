package lab

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/app/workflow"
	"github.com/guillaume-galp/cge/internal/infra/copilot"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

const SchemaVersion = "v1"

const (
	WorkflowModeGraphBacked = "graph_backed"
	WorkflowModeBaseline    = "baseline"

	BlockingFactorTaskFamily      = "task_family"
	BlockingFactorModel           = "model"
	BlockingFactorSessionTopology = "session_topology"
)

type Service struct {
	manager               *repo.Manager
	workflowRunner        WorkflowRunner
	copilotUsageCollector CopilotSessionUsageCollector
	now                   func() time.Time
}

type CopilotSessionUsageCollector interface {
	CollectSessionUsage(ctx context.Context, request copilot.SessionUsageRequest) (copilot.SessionUsage, error)
}

func NewService(manager *repo.Manager) *Service {
	return &Service{
		manager:               manager,
		workflowRunner:        workflow.NewService(manager),
		copilotUsageCollector: copilot.NewSessionStateCollector(""),
		now:                   func() time.Time { return time.Now().UTC() },
	}
}

type InitResult struct {
	Workspace WorkspaceState `json:"workspace"`
	Lab       LabState       `json:"lab"`
	Installed WorkSummary    `json:"installed"`
	Refreshed WorkSummary    `json:"refreshed"`
	Preserved WorkSummary    `json:"preserved"`
	Skipped   WorkSummary    `json:"skipped"`
}

type WorkspaceState struct {
	Path               string `json:"path"`
	Initialized        bool   `json:"initialized"`
	AlreadyInitialized bool   `json:"already_initialized"`
}

type LabState struct {
	Path               string `json:"path"`
	SuiteManifestPath  string `json:"suite_manifest_path"`
	ConditionsPath     string `json:"conditions_manifest_path"`
	SchemaVersion      string `json:"schema_version"`
	AlreadyInitialized bool   `json:"already_initialized"`
}

type WorkSummary struct {
	Count int        `json:"count"`
	Items []WorkItem `json:"items"`
}

type WorkItem struct {
	Kind   string `json:"kind"`
	Path   string `json:"path,omitempty"`
	Status string `json:"status,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type SuiteManifest struct {
	SchemaVersion string      `json:"schema_version"`
	SuiteID       string      `json:"suite_id"`
	Tasks         []SuiteTask `json:"tasks"`
}

type SuiteTask struct {
	TaskID                string `json:"task_id"`
	Family                string `json:"family"`
	Description           string `json:"description"`
	AcceptanceCriteriaRef string `json:"acceptance_criteria_ref"`
}

type ConditionsManifest struct {
	SchemaVersion   string      `json:"schema_version"`
	Conditions      []Condition `json:"conditions"`
	BlockingFactors []string    `json:"blocking_factors"`
}

type Condition struct {
	ConditionID  string `json:"condition_id"`
	WorkflowMode string `json:"workflow_mode"`
	Description  string `json:"description"`
}

func (m SuiteManifest) TaskByID(taskID string) (SuiteTask, bool) {
	for _, task := range m.Tasks {
		if task.TaskID == taskID {
			return task, true
		}
	}
	return SuiteTask{}, false
}

func (m ConditionsManifest) ConditionByID(conditionID string) (Condition, bool) {
	for _, condition := range m.Conditions {
		if condition.ConditionID == conditionID {
			return condition, true
		}
	}
	return Condition{}, false
}

func (s *Service) Init(ctx context.Context, startDir string) (InitResult, error) {
	if s == nil || s.manager == nil {
		return InitResult{}, errors.New("lab service is not configured")
	}

	workspaceInit, err := s.manager.InitWorkspace(ctx, startDir)
	if err != nil {
		return InitResult{}, err
	}

	workspace, err := s.manager.OpenWorkspace(ctx, startDir)
	if err != nil {
		return InitResult{}, err
	}

	labPath := filepath.Join(workspace.WorkspacePath, repo.LabDirName)
	suitePath := filepath.Join(labPath, repo.LabSuiteManifestName)
	conditionsPath := filepath.Join(labPath, repo.LabConditionsManifestName)

	preservedArtifacts, err := collectPreservedArtifacts(workspace.RepoRoot, labPath)
	if err != nil {
		return InitResult{}, err
	}

	var installed []WorkItem
	var refreshed []WorkItem
	var skipped []WorkItem

	for _, dir := range []struct {
		path string
		kind string
	}{
		{path: labPath, kind: "lab_directory"},
		{path: filepath.Join(labPath, "runs"), kind: "run_ledger_directory"},
		{path: filepath.Join(labPath, "evaluations"), kind: "evaluation_directory"},
		{path: filepath.Join(labPath, "reports"), kind: "report_directory"},
	} {
		action, item, err := ensureDir(workspace.RepoRoot, dir.path, dir.kind)
		if err != nil {
			return InitResult{}, err
		}
		switch action {
		case "installed":
			installed = append(installed, item)
		default:
			skipped = append(skipped, item)
		}
	}

	for _, file := range []struct {
		path    string
		kind    string
		payload any
		check   func(string) error
	}{
		{
			path:    suitePath,
			kind:    "suite_manifest",
			payload: defaultSuiteManifest(),
			check: func(path string) error {
				_, err := loadSuiteManifest(path)
				return err
			},
		},
		{
			path:    conditionsPath,
			kind:    "conditions_manifest",
			payload: defaultConditionsManifest(),
			check: func(path string) error {
				_, err := loadConditionsManifest(path)
				return err
			},
		},
	} {
		action, item, err := ensureManifestFile(workspace.RepoRoot, file.path, file.kind, file.payload, file.check)
		if err != nil {
			return InitResult{}, err
		}
		switch action {
		case "installed":
			installed = append(installed, item)
		case "refreshed":
			refreshed = append(refreshed, item)
		default:
			skipped = append(skipped, item)
		}
	}

	return InitResult{
		Workspace: WorkspaceState{
			Path:               workspace.WorkspacePath,
			Initialized:        !workspaceInit.AlreadyExists,
			AlreadyInitialized: workspaceInit.AlreadyExists,
		},
		Lab: LabState{
			Path:               labPath,
			SuiteManifestPath:  suitePath,
			ConditionsPath:     conditionsPath,
			SchemaVersion:      SchemaVersion,
			AlreadyInitialized: pathExists(labPath) && len(installed) == 0,
		},
		Installed: summarize(installed),
		Refreshed: summarize(refreshed),
		Preserved: summarize(preservedArtifacts),
		Skipped:   summarize(skipped),
	}, nil
}

func (s *Service) LoadSuiteManifest(ctx context.Context, startDir string) (SuiteManifest, error) {
	if s == nil || s.manager == nil {
		return SuiteManifest{}, errors.New("lab service is not configured")
	}

	workspace, err := s.manager.OpenWorkspace(ctx, startDir)
	if err != nil {
		return SuiteManifest{}, err
	}

	path := filepath.Join(workspace.WorkspacePath, repo.LabDirName, repo.LabSuiteManifestName)
	return loadSuiteManifest(path)
}

func (s *Service) LoadConditionsManifest(ctx context.Context, startDir string) (ConditionsManifest, error) {
	if s == nil || s.manager == nil {
		return ConditionsManifest{}, errors.New("lab service is not configured")
	}

	workspace, err := s.manager.OpenWorkspace(ctx, startDir)
	if err != nil {
		return ConditionsManifest{}, err
	}

	path := filepath.Join(workspace.WorkspacePath, repo.LabDirName, repo.LabConditionsManifestName)
	return loadConditionsManifest(path)
}

func defaultSuiteManifest() SuiteManifest {
	return SuiteManifest{
		SchemaVersion: SchemaVersion,
		SuiteID:       "delegated-workflow-evidence-v1",
		Tasks:         []SuiteTask{},
	}
}

func defaultConditionsManifest() ConditionsManifest {
	return ConditionsManifest{
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
		BlockingFactors: []string{BlockingFactorTaskFamily, BlockingFactorModel, BlockingFactorSessionTopology},
	}
}

func loadSuiteManifest(path string) (SuiteManifest, error) {
	var manifest SuiteManifest
	if err := loadManifestJSON(path, &manifest); err != nil {
		return SuiteManifest{}, err
	}
	if err := validateSuiteManifest(path, manifest); err != nil {
		return SuiteManifest{}, err
	}
	return manifest, nil
}

func loadConditionsManifest(path string) (ConditionsManifest, error) {
	var manifest ConditionsManifest
	if err := loadManifestJSON(path, &manifest); err != nil {
		return ConditionsManifest{}, err
	}
	if err := validateConditionsManifest(path, manifest); err != nil {
		return ConditionsManifest{}, err
	}
	return manifest, nil
}

func loadManifestJSON(path string, target any) error {
	payload, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "manifest_error",
			Code:     "manifest_parse_failed",
			Message:  "manifest could not be parsed",
			Details: map[string]any{
				"path":   path,
				"reason": err.Error(),
			},
		}, fmt.Errorf("parse manifest %s: %w", path, err))
	}
	return nil
}

func validateSuiteManifest(path string, manifest SuiteManifest) error {
	var violations []map[string]any

	if manifest.SchemaVersion == "" {
		violations = append(violations, violation("schema_version", "field is required"))
	} else if manifest.SchemaVersion != SchemaVersion {
		violations = append(violations, violationWithAllowed("schema_version", "unsupported schema version", manifest.SchemaVersion, []string{SchemaVersion}))
	}
	if manifest.SuiteID == "" {
		violations = append(violations, violation("suite_id", "field is required"))
	}
	if manifest.Tasks == nil {
		violations = append(violations, violation("tasks", "field is required"))
	}

	seenTaskIDs := map[string]struct{}{}
	for i, task := range manifest.Tasks {
		fieldPrefix := fmt.Sprintf("tasks[%d]", i)
		if task.TaskID == "" {
			violations = append(violations, violation(fieldPrefix+".task_id", "field is required"))
		} else {
			if _, exists := seenTaskIDs[task.TaskID]; exists {
				violations = append(violations, violation(fieldPrefix+".task_id", "task_id must be unique"))
			}
			seenTaskIDs[task.TaskID] = struct{}{}
		}
		if task.Family == "" {
			violations = append(violations, violation(fieldPrefix+".family", "field is required"))
		}
		if task.Description == "" {
			violations = append(violations, violation(fieldPrefix+".description", "field is required"))
		}
		if task.AcceptanceCriteriaRef == "" {
			violations = append(violations, violation(fieldPrefix+".acceptance_criteria_ref", "field is required"))
		}
	}

	if len(violations) > 0 {
		return manifestValidationError("suite", path, violations)
	}
	return nil
}

func validateConditionsManifest(path string, manifest ConditionsManifest) error {
	var violations []map[string]any

	if manifest.SchemaVersion == "" {
		violations = append(violations, violation("schema_version", "field is required"))
	} else if manifest.SchemaVersion != SchemaVersion {
		violations = append(violations, violationWithAllowed("schema_version", "unsupported schema version", manifest.SchemaVersion, []string{SchemaVersion}))
	}
	if manifest.Conditions == nil {
		violations = append(violations, violation("conditions", "field is required"))
	}
	if manifest.BlockingFactors == nil {
		violations = append(violations, violation("blocking_factors", "field is required"))
	}

	seenConditionIDs := map[string]struct{}{}
	for i, condition := range manifest.Conditions {
		fieldPrefix := fmt.Sprintf("conditions[%d]", i)
		if condition.ConditionID == "" {
			violations = append(violations, violation(fieldPrefix+".condition_id", "field is required"))
		} else {
			if _, exists := seenConditionIDs[condition.ConditionID]; exists {
				violations = append(violations, violation(fieldPrefix+".condition_id", "condition_id must be unique"))
			}
			seenConditionIDs[condition.ConditionID] = struct{}{}
		}
		if condition.WorkflowMode == "" {
			violations = append(violations, violation(fieldPrefix+".workflow_mode", "field is required"))
		} else if !isAllowedWorkflowMode(condition.WorkflowMode) {
			violations = append(violations, violationWithAllowed(fieldPrefix+".workflow_mode", "workflow_mode must be recognized", condition.WorkflowMode, []string{WorkflowModeGraphBacked, WorkflowModeBaseline}))
		}
		if condition.Description == "" {
			violations = append(violations, violation(fieldPrefix+".description", "field is required"))
		}
	}

	if manifest.BlockingFactors != nil {
		seenFactors := map[string]struct{}{}
		requiredFactors := map[string]struct{}{
			BlockingFactorTaskFamily:      {},
			BlockingFactorModel:           {},
			BlockingFactorSessionTopology: {},
		}
		for i, factor := range manifest.BlockingFactors {
			field := fmt.Sprintf("blocking_factors[%d]", i)
			if factor == "" {
				violations = append(violations, violation(field, "field is required"))
				continue
			}
			if _, exists := seenFactors[factor]; exists {
				violations = append(violations, violation(field, "blocking factor must be unique"))
				continue
			}
			seenFactors[factor] = struct{}{}
			if _, allowed := requiredFactors[factor]; !allowed {
				violations = append(violations, violationWithAllowed(field, "blocking factor must be recognized", factor, []string{BlockingFactorTaskFamily, BlockingFactorModel, BlockingFactorSessionTopology}))
				continue
			}
			delete(requiredFactors, factor)
		}
		for factor := range requiredFactors {
			violations = append(violations, violation("blocking_factors", fmt.Sprintf("missing required blocking factor %q", factor)))
		}
	}

	if len(violations) > 0 {
		return manifestValidationError("conditions", path, violations)
	}
	return nil
}

func isAllowedWorkflowMode(mode string) bool {
	switch mode {
	case WorkflowModeGraphBacked, WorkflowModeBaseline:
		return true
	default:
		return false
	}
}

func manifestValidationError(kind, path string, violations []map[string]any) error {
	return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
		Category: "validation_error",
		Type:     "manifest_error",
		Code:     "manifest_validation_failed",
		Message:  fmt.Sprintf("%s manifest validation failed", kind),
		Details: map[string]any{
			"manifest_kind": kind,
			"path":          path,
			"violations":    violations,
		},
	}, fmt.Errorf("%s manifest validation failed", kind))
}

func violation(field, message string) map[string]any {
	return map[string]any{
		"field":   field,
		"message": message,
	}
}

func violationWithAllowed(field, message string, value any, allowed []string) map[string]any {
	entry := violation(field, message)
	entry["value"] = value
	entry["allowed_values"] = allowed
	return entry
}

func ensureDir(repoRoot, path, kind string) (string, WorkItem, error) {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return "", WorkItem{}, fmt.Errorf("%s path %s already exists and is not a directory", kind, path)
		}
		return "skipped", WorkItem{Kind: kind, Path: relativePath(repoRoot, path), Status: "unchanged"}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", WorkItem{}, fmt.Errorf("inspect %s %s: %w", kind, path, err)
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", WorkItem{}, fmt.Errorf("create %s %s: %w", kind, path, err)
	}
	return "installed", WorkItem{Kind: kind, Path: relativePath(repoRoot, path), Status: "installed"}, nil
}

func ensureJSONFile(repoRoot, path, kind string, payload any) (string, WorkItem, error) {
	desired, err := marshalJSON(payload)
	if err != nil {
		return "", WorkItem{}, fmt.Errorf("encode %s: %w", kind, err)
	}

	existing, err := os.ReadFile(path)
	if err == nil {
		if bytes.Equal(existing, desired) {
			return "skipped", WorkItem{Kind: kind, Path: relativePath(repoRoot, path), Status: "unchanged"}, nil
		}
		if err := os.WriteFile(path, desired, 0o644); err != nil {
			return "", WorkItem{}, fmt.Errorf("refresh %s %s: %w", kind, path, err)
		}
		return "refreshed", WorkItem{Kind: kind, Path: relativePath(repoRoot, path), Status: "refreshed"}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", WorkItem{}, fmt.Errorf("inspect %s %s: %w", kind, path, err)
	}
	if err := os.WriteFile(path, desired, 0o644); err != nil {
		return "", WorkItem{}, fmt.Errorf("write %s %s: %w", kind, path, err)
	}
	return "installed", WorkItem{Kind: kind, Path: relativePath(repoRoot, path), Status: "installed"}, nil
}

func ensureManifestFile(
	repoRoot, path, kind string,
	payload any,
	validateExisting func(string) error,
) (string, WorkItem, error) {
	desired, err := marshalJSON(payload)
	if err != nil {
		return "", WorkItem{}, fmt.Errorf("encode %s: %w", kind, err)
	}

	existing, err := os.ReadFile(path)
	if err == nil {
		if bytes.Equal(existing, desired) {
			return "skipped", WorkItem{Kind: kind, Path: relativePath(repoRoot, path), Status: "unchanged"}, nil
		}
		if validateExisting != nil && validateExisting(path) == nil {
			return "skipped", WorkItem{
				Kind:   kind,
				Path:   relativePath(repoRoot, path),
				Status: "preserved",
				Reason: "existing valid customized manifest preserved",
			}, nil
		}
		if err := os.WriteFile(path, desired, 0o644); err != nil {
			return "", WorkItem{}, fmt.Errorf("refresh %s %s: %w", kind, path, err)
		}
		return "refreshed", WorkItem{
			Kind:   kind,
			Path:   relativePath(repoRoot, path),
			Status: "refreshed",
			Reason: "existing manifest was invalid or stale",
		}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", WorkItem{}, fmt.Errorf("inspect %s %s: %w", kind, path, err)
	}
	if err := os.WriteFile(path, desired, 0o644); err != nil {
		return "", WorkItem{}, fmt.Errorf("write %s %s: %w", kind, path, err)
	}
	return "installed", WorkItem{Kind: kind, Path: relativePath(repoRoot, path), Status: "installed"}, nil
}

func marshalJSON(payload any) ([]byte, error) {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func collectPreservedArtifacts(repoRoot, labPath string) ([]WorkItem, error) {
	var preserved []WorkItem
	for _, dir := range []struct {
		path string
		kind string
	}{
		{path: filepath.Join(labPath, "runs"), kind: "run_artifact"},
		{path: filepath.Join(labPath, "evaluations"), kind: "evaluation_artifact"},
		{path: filepath.Join(labPath, "reports"), kind: "report_artifact"},
	} {
		items, err := collectFiles(repoRoot, dir.path, dir.kind)
		if err != nil {
			return nil, err
		}
		preserved = append(preserved, items...)
	}

	slices.SortFunc(preserved, func(a, b WorkItem) int {
		return cmpString(a.Path, b.Path)
	})
	return preserved, nil
}

func collectFiles(repoRoot, root, kind string) ([]WorkItem, error) {
	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("inspect %s root %s: %w", kind, root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s root %s is not a directory", kind, root)
	}

	var items []WorkItem
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		items = append(items, WorkItem{
			Kind:   kind,
			Path:   relativePath(repoRoot, path),
			Status: "preserved",
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("collect %s artifacts from %s: %w", kind, root, err)
	}
	return items, nil
}

func summarize(items []WorkItem) WorkSummary {
	if items == nil {
		items = []WorkItem{}
	}
	return WorkSummary{
		Count: len(items),
		Items: items,
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func cmpString(left, right string) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func relativePath(repoRoot, path string) string {
	relative, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relative)
}
