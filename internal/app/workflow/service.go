package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/guillaume-galp/cge/internal/app/attribution"
	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/app/contextprojector"
	"github.com/guillaume-galp/cge/internal/app/contextevaluator"
	"github.com/guillaume-galp/cge/internal/app/decisionengine"
	"github.com/guillaume-galp/cge/internal/app/retrieval"
	"github.com/guillaume-galp/cge/internal/infra/benchmarks"
	"github.com/guillaume-galp/cge/internal/infra/kuzu"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

const ManifestSchemaVersion = "v1"

type Service struct {
	manager        *repo.Manager
	writer         SeedWriter
	reader         ReadinessReader
	querier        KickoffQuerier
	projector      KickoffProjector
	benchmarkStore BenchmarkStore
	evaluator      *contextevaluator.Evaluator
	decisionEngine *decisionengine.Engine
	attribution    *attribution.Recorder
	now            func() time.Time
}

func NewService(manager *repo.Manager) *Service {
	eval := contextevaluator.NewEvaluator(contextevaluator.Config{})
	eng := decisionengine.NewWithDefaults()
	return &Service{
		manager:        manager,
		writer:         kuzu.NewStore(),
		reader:         kuzu.NewStore(),
		querier:        retrieval.NewEngine(nil, nil).WithEvaluator(&eval),
		projector:      contextprojector.NewProjector(),
		benchmarkStore: benchmarks.NewStore(),
		evaluator:      &eval,
		decisionEngine: &eng,
		attribution:    attribution.NewRecorder(),
		now:            func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) NowForTest(now func() time.Time) {
	if s == nil {
		return
	}
	s.now = now
}

func (s *Service) EvaluatorForTest(eval *contextevaluator.Evaluator) {
	if s == nil {
		return
	}
	s.evaluator = eval
}

func (s *Service) DecisionEngineForTest(eng *decisionengine.Engine) {
	if s == nil {
		return
	}
	s.decisionEngine = eng
}

func (s *Service) AttributionRecorderForTest(rec *attribution.Recorder) {
	if s == nil {
		return
	}
	s.attribution = rec
}

func (s *Service) SeedWriterForTest(writer SeedWriter) {
	if s == nil {
		return
	}
	s.writer = writer
}

func (s *Service) ReadinessReaderForTest(reader ReadinessReader) {
	if s == nil {
		return
	}
	s.reader = reader
}

func (s *Service) KickoffQuerierForTest(querier KickoffQuerier) {
	if s == nil {
		return
	}
	s.querier = querier
}

func (s *Service) KickoffProjectorForTest(projector KickoffProjector) {
	if s == nil {
		return
	}
	s.projector = projector
}

func (s *Service) BenchmarkStoreForTest(store BenchmarkStore) {
	if s == nil {
		return
	}
	s.benchmarkStore = store
}

type InitResult struct {
	Workspace WorkspaceState `json:"workspace"`
	Manifest  ManifestState  `json:"manifest"`
	Installed WorkSummary    `json:"installed"`
	Refreshed WorkSummary    `json:"refreshed"`
	Preserved WorkSummary    `json:"preserved"`
	Skipped   WorkSummary    `json:"skipped"`
	Seeded    WorkSummary    `json:"seeded"`
}

type WorkspaceState struct {
	Path               string `json:"path"`
	Initialized        bool   `json:"initialized"`
	AlreadyInitialized bool   `json:"already_initialized"`
}

type ManifestState struct {
	Path          string  `json:"path"`
	SchemaVersion string  `json:"schema_version"`
	Assets        []Asset `json:"assets"`
	Overrides     int     `json:"preserved_override_count"`
	InstalledAt   string  `json:"installed_at"`
	RefreshedAt   string  `json:"refreshed_at"`
}

type WorkSummary struct {
	Count int        `json:"count"`
	Items []WorkItem `json:"items"`
}

type WorkItem struct {
	Kind   string `json:"kind"`
	Path   string `json:"path,omitempty"`
	Source string `json:"source,omitempty"`
	Status string `json:"status,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type Manifest struct {
	SchemaVersion      string     `json:"schema_version"`
	InstalledAt        string     `json:"installed_at"`
	RefreshedAt        string     `json:"refreshed_at"`
	Assets             []Asset    `json:"assets"`
	PreservedOverrides []Override `json:"preserved_overrides"`
}

type Asset struct {
	Path         string `json:"path"`
	Kind         string `json:"kind"`
	Status       string `json:"status"`
	Digest       string `json:"digest,omitempty"`
	OverridePath string `json:"override_path,omitempty"`
}

type Override struct {
	Path      string `json:"path"`
	AssetPath string `json:"asset_path,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

func (s *Service) Init(ctx context.Context, startDir string) (InitResult, error) {
	if s == nil || s.manager == nil {
		return InitResult{}, errors.New("workflow service is not configured")
	}
	if s.now == nil {
		s.now = func() time.Time { return time.Now().UTC() }
	}

	initResult, err := s.manager.InitWorkspace(ctx, startDir)
	if err != nil {
		return InitResult{}, err
	}

	workspace, err := s.manager.OpenWorkspace(ctx, startDir)
	if err != nil {
		return InitResult{}, err
	}

	workflowDir := filepath.Join(workspace.WorkspacePath, repo.WorkflowDirName)
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		return InitResult{}, fmt.Errorf("create workflow directory %s: %w", workflowDir, err)
	}

	manifestPath := filepath.Join(workflowDir, repo.WorkflowManifestName)
	manifest, manifestFound, err := s.prepareManifest(manifestPath)
	if err != nil {
		return InitResult{}, err
	}

	manifest, installed, skipped := s.trackManifestAsset(manifest, manifestFound)
	manifest, assetInstalled, assetRefreshed, assetSkipped, err := s.ensureAssets(workspace, manifest)
	if err != nil {
		return InitResult{}, err
	}
	installed = mergeWorkSummaries(installed, assetInstalled)
	refreshed := assetRefreshed
	skipped = mergeWorkSummaries(skipped, assetSkipped)
	preserved := summarizeOverrides(manifest.PreservedOverrides)

	if err := writeManifest(manifestPath, manifest); err != nil {
		return InitResult{}, classifyManifestWriteError(manifestPath, err)
	}

	seeded, err := s.seedBaseline(ctx, workspace)
	if err != nil {
		return InitResult{}, err
	}

	return InitResult{
		Workspace: WorkspaceState{
			Path:               workspace.WorkspacePath,
			Initialized:        !initResult.AlreadyExists,
			AlreadyInitialized: initResult.AlreadyExists,
		},
		Manifest: ManifestState{
			Path:          manifestPath,
			SchemaVersion: manifest.SchemaVersion,
			Assets:        manifest.Assets,
			Overrides:     len(manifest.PreservedOverrides),
			InstalledAt:   manifest.InstalledAt,
			RefreshedAt:   manifest.RefreshedAt,
		},
		Installed: installed,
		Refreshed: refreshed,
		Preserved: preserved,
		Skipped:   skipped,
		Seeded:    seeded,
	}, nil
}

func (s *Service) prepareManifest(path string) (Manifest, bool, error) {
	now := s.now().Format(time.RFC3339)
	existing, found, err := loadManifest(path)
	if err != nil {
		return Manifest{}, false, classifyManifestLoadError(path, err)
	}

	manifest := Manifest{
		SchemaVersion:      ManifestSchemaVersion,
		InstalledAt:        now,
		RefreshedAt:        now,
		Assets:             []Asset{},
		PreservedOverrides: []Override{},
	}
	if found {
		if existing.SchemaVersion != ManifestSchemaVersion {
			return Manifest{}, false, fmt.Errorf(
				"workflow manifest at %s uses unsupported schema version %q",
				path,
				existing.SchemaVersion,
			)
		}
		manifest = existing
		if manifest.InstalledAt == "" {
			manifest.InstalledAt = now
		}
		manifest.RefreshedAt = now
		if manifest.Assets == nil {
			manifest.Assets = []Asset{}
		}
		if manifest.PreservedOverrides == nil {
			manifest.PreservedOverrides = []Override{}
		}
	}

	manifest.Assets = normalizeAssets(manifest.Assets)
	manifest.PreservedOverrides = normalizeOverrides(manifest.PreservedOverrides)

	return manifest, found, nil
}

func (s *Service) trackManifestAsset(manifest Manifest, found bool) (Manifest, WorkSummary, WorkSummary) {
	installed := WorkSummary{Items: []WorkItem{}}
	skipped := WorkSummary{Items: []WorkItem{}}
	manifestAsset := Asset{
		Path:   filepath.ToSlash(filepath.Join(repo.WorkspaceDirName, repo.WorkflowDirName, repo.WorkflowManifestName)),
		Kind:   "workflow_manifest",
		Status: "installed",
	}
	if index := slices.IndexFunc(manifest.Assets, func(asset Asset) bool {
		return asset.Path == manifestAsset.Path
	}); index >= 0 {
		manifest.Assets[index] = mergeAsset(manifest.Assets[index], manifestAsset)
		if found {
			skipped.Items = append(skipped.Items, WorkItem{
				Kind:   manifest.Assets[index].Kind,
				Path:   manifest.Assets[index].Path,
				Status: "skipped",
				Reason: "workflow manifest already exists",
			})
		}
	} else {
		manifest.Assets = upsertAsset(manifest.Assets, manifestAsset)
		installed.Items = append(installed.Items, WorkItem{
			Kind:   manifestAsset.Kind,
			Path:   manifestAsset.Path,
			Status: "installed",
			Reason: "bootstrapped workflow manifest tracking",
		})
	}

	installed.Count = len(installed.Items)
	skipped.Count = len(skipped.Items)
	return manifest, installed, skipped
}

func loadManifest(path string) (Manifest, bool, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Manifest{}, false, nil
		}
		return Manifest{}, false, fmt.Errorf("read workflow manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return Manifest{}, false, fmt.Errorf("parse workflow manifest: %w", err)
	}

	return manifest, true, nil
}

func writeManifest(path string, manifest Manifest) error {
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode workflow manifest: %w", err)
	}
	payload = append(payload, '\n')

	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write workflow manifest: %w", err)
	}
	return nil
}

func mergeWorkSummaries(summaries ...WorkSummary) WorkSummary {
	merged := WorkSummary{Items: []WorkItem{}}
	for _, summary := range summaries {
		merged.Items = append(merged.Items, summary.Items...)
	}
	merged.Count = len(merged.Items)
	return merged
}

func summarizeOverrides(overrides []Override) WorkSummary {
	summary := WorkSummary{Items: make([]WorkItem, 0, len(overrides))}
	for _, override := range overrides {
		summary.Items = append(summary.Items, WorkItem{
			Kind:   firstNonEmpty(override.Kind, "override"),
			Path:   override.Path,
			Source: firstNonEmpty(override.AssetPath, "manifest"),
			Status: "preserved",
			Reason: firstNonEmpty(override.Reason, "explicit repo override preserved during refresh"),
		})
	}
	summary.Count = len(summary.Items)
	return summary
}

func classifyManifestLoadError(path string, err error) error {
	return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "workflow_error",
		Code:     "workflow_manifest_load_failed",
		Message:  "workflow bootstrap could not load the workflow manifest",
		Details: map[string]any{
			"path":   filepath.ToSlash(path),
			"reason": err.Error(),
		},
	}, err)
}

func classifyManifestWriteError(path string, err error) error {
	return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "workflow_error",
		Code:     "workflow_manifest_write_failed",
		Message:  "workflow bootstrap could not write the workflow manifest",
		Details: map[string]any{
			"path":   filepath.ToSlash(path),
			"reason": err.Error(),
		},
	}, err)
}

func normalizeAssets(assets []Asset) []Asset {
	if len(assets) == 0 {
		return []Asset{}
	}

	normalized := make([]Asset, 0, len(assets))
	for _, asset := range assets {
		asset.Path = normalizeManifestPath(asset.Path)
		asset.OverridePath = normalizeManifestPath(asset.OverridePath)
		if asset.Path == "" {
			continue
		}
		normalized = upsertAsset(normalized, asset)
	}
	slices.SortFunc(normalized, func(a, b Asset) int {
		switch {
		case a.Path < b.Path:
			return -1
		case a.Path > b.Path:
			return 1
		default:
			return 0
		}
	})
	return normalized
}

func normalizeOverrides(overrides []Override) []Override {
	if len(overrides) == 0 {
		return []Override{}
	}

	normalized := make([]Override, 0, len(overrides))
	seen := map[string]struct{}{}
	for _, override := range overrides {
		override.Path = normalizeManifestPath(override.Path)
		override.AssetPath = normalizeManifestPath(override.AssetPath)
		if override.Path == "" {
			continue
		}
		key := override.Path + "\n" + override.AssetPath + "\n" + override.Kind
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, override)
	}
	slices.SortFunc(normalized, func(a, b Override) int {
		switch {
		case a.Path < b.Path:
			return -1
		case a.Path > b.Path:
			return 1
		default:
			return 0
		}
	})
	return normalized
}

func upsertAsset(assets []Asset, candidate Asset) []Asset {
	for i, asset := range assets {
		if asset.Path != candidate.Path {
			continue
		}
		assets[i] = mergeAsset(asset, candidate)
		return assets
	}
	return append(assets, candidate)
}

func mergeAsset(existing, candidate Asset) Asset {
	return Asset{
		Path:         firstNonEmpty(candidate.Path, existing.Path),
		Kind:         firstNonEmpty(candidate.Kind, existing.Kind),
		Status:       firstNonEmpty(candidate.Status, existing.Status),
		Digest:       firstNonEmpty(candidate.Digest, existing.Digest),
		OverridePath: firstNonEmpty(candidate.OverridePath, existing.OverridePath),
	}
}

func normalizeManifestPath(path string) string {
	path = filepath.ToSlash(filepath.Clean(path))
	switch path {
	case ".", "":
		return ""
	default:
		return path
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
