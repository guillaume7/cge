package workflow

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	graphpayload "github.com/guillaume-galp/cge/internal/domain/payload"
	"github.com/guillaume-galp/cge/internal/infra/kuzu"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func TestServiceInitBootstrapsWorkspaceAndManifest(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	service := NewService(manager)
	service.now = func() time.Time { return time.Date(2026, 3, 22, 17, 0, 0, 0, time.UTC) }

	result, err := service.Init(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	if !result.Workspace.Initialized || result.Workspace.AlreadyInitialized {
		t.Fatalf("workspace state = %#v, want initialized and not already initialized", result.Workspace)
	}
	if result.Installed.Count != 5 {
		t.Fatalf("installed count = %d, want 5", result.Installed.Count)
	}
	if result.Refreshed.Count != 0 {
		t.Fatalf("refreshed count = %d, want 0", result.Refreshed.Count)
	}
	if result.Preserved.Count != 0 {
		t.Fatalf("preserved count = %d, want 0", result.Preserved.Count)
	}
	if result.Skipped.Count != 0 {
		t.Fatalf("skipped count = %d, want 0", result.Skipped.Count)
	}
	if result.Seeded.Count != 3 {
		t.Fatalf("seeded count = %d, want 3 missing-source items", result.Seeded.Count)
	}
	for _, item := range result.Seeded.Items {
		if item.Status != "skipped" {
			t.Fatalf("seeded item status = %q, want skipped", item.Status)
		}
	}

	configPath := filepath.Join(repoDir, repo.WorkspaceDirName, repo.ConfigFileName)
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("workspace config stat: %v", err)
	}

	manifest := readManifest(t, filepath.Join(repoDir, repo.WorkspaceDirName, repo.WorkflowDirName, repo.WorkflowManifestName))
	if manifest.SchemaVersion != ManifestSchemaVersion {
		t.Fatalf("manifest schema_version = %q, want %q", manifest.SchemaVersion, ManifestSchemaVersion)
	}
	if manifest.InstalledAt != "2026-03-22T17:00:00Z" || manifest.RefreshedAt != "2026-03-22T17:00:00Z" {
		t.Fatalf("manifest timestamps = %#v, want bootstrap timestamp", manifest)
	}
	if len(manifest.Assets) != 5 {
		t.Fatalf("manifest assets = %#v, want manifest plus managed workflow assets", manifest.Assets)
	}

	expectedAssets := append([]string{".graph/workflow/manifest.json"}, managedWorkflowAssetPaths()...)
	for _, assetPath := range expectedAssets {
		if !manifestHasAsset(manifest, assetPath) {
			t.Fatalf("manifest assets = %#v, missing %q", manifest.Assets, assetPath)
		}
		if _, err := os.Stat(filepath.Join(repoDir, filepath.FromSlash(assetPath))); err != nil && assetPath != ".graph/workflow/manifest.json" {
			t.Fatalf("asset %s stat: %v", assetPath, err)
		}
	}
}

func TestServiceInitRefreshesModifiedWorkflowAssetsIdempotently(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	service := NewService(manager)
	service.now = func() time.Time { return time.Date(2026, 3, 22, 17, 0, 0, 0, time.UTC) }

	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("initial Init returned error: %v", err)
	}

	targetAsset := filepath.Join(repoDir, filepath.FromSlash(managedWorkflowAssetPaths()[0]))
	writeWorkflowFixture(t, targetAsset, "# stale asset\n")

	service.now = func() time.Time { return time.Date(2026, 3, 22, 18, 0, 0, 0, time.UTC) }
	result, err := service.Init(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("refresh Init returned error: %v", err)
	}

	if result.Workspace.Initialized || !result.Workspace.AlreadyInitialized {
		t.Fatalf("workspace state = %#v, want already initialized", result.Workspace)
	}
	if result.Installed.Count != 0 {
		t.Fatalf("installed count = %d, want 0", result.Installed.Count)
	}
	if result.Refreshed.Count != 1 {
		t.Fatalf("refreshed count = %d, want 1", result.Refreshed.Count)
	}
	if result.Preserved.Count != 0 {
		t.Fatalf("preserved count = %d, want 0", result.Preserved.Count)
	}
	if result.Skipped.Count != 4 {
		t.Fatalf("skipped count = %d, want 4", result.Skipped.Count)
	}
	if got := result.Refreshed.Items[0].Path; got != managedWorkflowAssetPaths()[0] {
		t.Fatalf("refreshed path = %q, want %q", got, managedWorkflowAssetPaths()[0])
	}
	if result.Seeded.Count != 3 {
		t.Fatalf("seeded count = %d, want 3 missing-source items", result.Seeded.Count)
	}

	for _, spec := range managedWorkflowAssetSpecs() {
		payload, readErr := os.ReadFile(filepath.Join(repoDir, filepath.FromSlash(spec.Path)))
		if readErr != nil {
			t.Fatalf("os.ReadFile asset %s: %v", spec.Path, readErr)
		}
		if string(payload) != spec.Content {
			t.Fatalf("asset %s content = %q, want refreshed managed content", spec.Path, string(payload))
		}
	}

	manifestPath := filepath.Join(repoDir, repo.WorkspaceDirName, repo.WorkflowDirName, repo.WorkflowManifestName)
	refreshed := readManifest(t, manifestPath)
	if refreshed.RefreshedAt != "2026-03-22T18:00:00Z" {
		t.Fatalf("refreshed_at = %q, want 2026-03-22T18:00:00Z", refreshed.RefreshedAt)
	}
	if refreshed.InstalledAt != "2026-03-22T17:00:00Z" {
		t.Fatalf("installed_at = %q, want original installed timestamp", refreshed.InstalledAt)
	}
}

func TestServiceInitPreservesManagedAssetOverridesOnRefresh(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	service := NewService(manager)
	service.now = func() time.Time { return time.Date(2026, 3, 22, 17, 0, 0, 0, time.UTC) }

	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("initial Init returned error: %v", err)
	}

	manifestPath := filepath.Join(repoDir, repo.WorkspaceDirName, repo.WorkflowDirName, repo.WorkflowManifestName)
	manifest := readManifest(t, manifestPath)
	overridePath := filepath.ToSlash(filepath.Join(
		repo.WorkspaceDirName,
		repo.WorkflowDirName,
		workflowAssetsDirName,
		"instructions",
		"delegated-graph-workflow.instructions.md",
	))
	customContent := "# repo override\n\nkeep this custom instruction.\n"
	writeWorkflowFixture(t, filepath.Join(repoDir, filepath.FromSlash(overridePath)), customContent)
	manifest.PreservedOverrides = append(manifest.PreservedOverrides, Override{
		Path:   overridePath,
		Kind:   "workflow_instruction_override",
		Reason: "repo keeps a custom delegated instruction",
	})
	writeManifestFixture(t, manifestPath, manifest)

	service.now = func() time.Time { return time.Date(2026, 3, 22, 18, 0, 0, 0, time.UTC) }
	result, err := service.Init(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("refresh Init returned error: %v", err)
	}

	if result.Workspace.Initialized || !result.Workspace.AlreadyInitialized {
		t.Fatalf("workspace state = %#v, want already initialized", result.Workspace)
	}
	if result.Installed.Count != 0 {
		t.Fatalf("installed count = %d, want 0", result.Installed.Count)
	}
	if result.Refreshed.Count != 0 {
		t.Fatalf("refreshed count = %d, want 0", result.Refreshed.Count)
	}
	if result.Preserved.Count != 1 {
		t.Fatalf("preserved count = %d, want 1", result.Preserved.Count)
	}
	if got := result.Preserved.Items[0].Path; got != overridePath {
		t.Fatalf("preserved path = %q, want %q", got, overridePath)
	}
	if result.Skipped.Count != 4 {
		t.Fatalf("skipped count = %d, want 4", result.Skipped.Count)
	}

	payload, err := os.ReadFile(filepath.Join(repoDir, filepath.FromSlash(overridePath)))
	if err != nil {
		t.Fatalf("os.ReadFile override asset: %v", err)
	}
	if string(payload) != customContent {
		t.Fatalf("override content = %q, want custom content preserved", string(payload))
	}

	refreshedManifest := readManifest(t, manifestPath)
	if len(refreshedManifest.PreservedOverrides) != 1 {
		t.Fatalf("preserved overrides = %#v, want 1 override", refreshedManifest.PreservedOverrides)
	}
	overrideAsset := mustManifestAsset(t, refreshedManifest, overridePath)
	if overrideAsset.Status != "preserved" {
		t.Fatalf("override asset status = %q, want preserved", overrideAsset.Status)
	}
}

func TestServiceInitSeedsBaselineRepoKnowledgeFromStandardArtifacts(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	writeWorkflowFixture(t, filepath.Join(repoDir, "README.md"), `# Cognitive Graph Engine

> A local, chainable graph memory CLI for AI agents.

CGE gives agents a shared, repo-scoped memory they can write to, query, and diff.
`)
	writeWorkflowFixture(t, filepath.Join(repoDir, "docs", "architecture", "components.md"), "# Components\n")
	writeWorkflowFixture(t, filepath.Join(repoDir, "docs", "architecture", "tech-stack.md"), "# Tech Stack\n")
	writeWorkflowFixture(t, filepath.Join(repoDir, "docs", "plan", "backlog.yaml"), "backlog:\n  project: cognitive-graph-engine\n")

	service := NewService(manager)
	service.now = func() time.Time { return time.Date(2026, 3, 22, 19, 0, 0, 0, time.UTC) }

	result, err := service.Init(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	if result.Seeded.Count != 3 {
		t.Fatalf("seeded count = %d, want 3", result.Seeded.Count)
	}
	for _, item := range result.Seeded.Items {
		if item.Status != "seeded" {
			t.Fatalf("seeded item status = %q, want seeded", item.Status)
		}
	}

	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}

	store := kuzu.NewStore()
	graph, err := store.ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph returned error: %v", err)
	}

	if len(graph.Nodes) != 4 {
		t.Fatalf("graph node count = %d, want 4", len(graph.Nodes))
	}
	if len(graph.Edges) != 3 {
		t.Fatalf("graph edge count = %d, want 3", len(graph.Edges))
	}

	nodes := mapGraphNodes(graph)
	readmeNode := mustGraphNode(t, nodes, "README.md")
	if readmeNode.Kind != "Document" {
		t.Fatalf("README node kind = %q, want Document", readmeNode.Kind)
	}
	if readmeNode.CreatedBy != workflowSeedAgentID || readmeNode.UpdatedBy != workflowSeedAgentID {
		t.Fatalf("README provenance = (%q, %q), want workflow-init", readmeNode.CreatedBy, readmeNode.UpdatedBy)
	}
	if !strings.Contains(readmeNode.Summary, "local, chainable graph memory CLI") {
		t.Fatalf("README summary = %q, want README excerpt", readmeNode.Summary)
	}

	architectureNode := mustGraphNode(t, nodes, "docs/architecture")
	if !strings.Contains(architectureNode.Summary, "components.md") || !strings.Contains(architectureNode.Summary, "tech-stack.md") {
		t.Fatalf("architecture summary = %q, want architecture file listing", architectureNode.Summary)
	}

	backlogNode := mustGraphNode(t, nodes, "docs/plan/backlog.yaml")
	if !strings.Contains(backlogNode.Summary, "cognitive-graph-engine") {
		t.Fatalf("backlog summary = %q, want project name", backlogNode.Summary)
	}

	revision, err := store.CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision returned error: %v", err)
	}
	if !revision.Exists {
		t.Fatal("expected seeded revision to exist")
	}
	if revision.Revision.Reason != workflowSeedRevisionReason {
		t.Fatalf("revision reason = %q, want %q", revision.Revision.Reason, workflowSeedRevisionReason)
	}
	if revision.Revision.CreatedBy != workflowSeedAgentID {
		t.Fatalf("revision created_by = %q, want %q", revision.Revision.CreatedBy, workflowSeedAgentID)
	}
}

func TestServiceInitDoesNotReseedUnchangedBaselineRepoKnowledge(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	writeWorkflowFixture(t, filepath.Join(repoDir, "README.md"), `# Cognitive Graph Engine

> A local, chainable graph memory CLI for AI agents.

CGE gives agents a shared, repo-scoped memory they can write to, query, and diff.
`)
	writeWorkflowFixture(t, filepath.Join(repoDir, "docs", "architecture", "components.md"), "# Components\n")
	writeWorkflowFixture(t, filepath.Join(repoDir, "docs", "plan", "backlog.yaml"), "project: cognitive-graph-engine\n")

	service := NewService(manager)
	service.now = func() time.Time { return time.Date(2026, 3, 22, 19, 0, 0, 0, time.UTC) }

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
	if !initialRevision.Exists {
		t.Fatal("expected initial seeded revision to exist")
	}

	service.now = func() time.Time { return time.Date(2026, 3, 22, 19, 30, 0, 0, time.UTC) }
	result, err := service.Init(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("refresh Init returned error: %v", err)
	}

	if result.Seeded.Count != 3 {
		t.Fatalf("seeded count = %d, want 3", result.Seeded.Count)
	}
	for _, item := range result.Seeded.Items {
		if item.Status != "skipped" {
			t.Fatalf("seeded item status = %q, want skipped", item.Status)
		}
		if item.Reason != "baseline repository knowledge already matches discoverable sources" {
			t.Fatalf("seeded item reason = %q, want no-op refresh reason", item.Reason)
		}
	}

	currentRevision, err := store.CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision after refresh returned error: %v", err)
	}
	if !currentRevision.Exists {
		t.Fatal("expected seeded revision to still exist")
	}
	if currentRevision.Revision.ID != initialRevision.Revision.ID {
		t.Fatalf("revision id = %q, want %q on no-op refresh", currentRevision.Revision.ID, initialRevision.Revision.ID)
	}
}

func TestServiceInitSkipsMissingOptionalSeedSources(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	writeWorkflowFixture(t, filepath.Join(repoDir, "README.md"), `# Cognitive Graph Engine

> A local, chainable graph memory CLI for AI agents.
`)

	service := NewService(manager)
	service.now = func() time.Time { return time.Date(2026, 3, 22, 19, 30, 0, 0, time.UTC) }

	result, err := service.Init(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	if result.Seeded.Count != 3 {
		t.Fatalf("seeded count = %d, want 3", result.Seeded.Count)
	}
	if item := findSeededItem(t, result.Seeded, "README.md"); item.Status != "seeded" {
		t.Fatalf("README seeded item status = %q, want seeded", item.Status)
	}
	for _, path := range []string{"docs/architecture", "docs/plan/backlog.yaml"} {
		item := findSeededItem(t, result.Seeded, path)
		if item.Status != "skipped" {
			t.Fatalf("seeded item %s status = %q, want skipped", path, item.Status)
		}
		if item.Reason != "optional seed source not found" {
			t.Fatalf("seeded item %s reason = %q, want explicit missing-source reason", path, item.Reason)
		}
	}

	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}

	graph, err := kuzu.NewStore().ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph returned error: %v", err)
	}
	if len(graph.Nodes) != 2 {
		t.Fatalf("graph node count = %d, want 2", len(graph.Nodes))
	}
	if len(graph.Edges) != 1 {
		t.Fatalf("graph edge count = %d, want 1", len(graph.Edges))
	}
}

func TestServiceInitReturnsStructuredSeedPersistenceError(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	writeWorkflowFixture(t, filepath.Join(repoDir, "README.md"), "# Cognitive Graph Engine\n")

	service := NewService(manager)
	service.now = func() time.Time { return time.Date(2026, 3, 22, 20, 0, 0, 0, time.UTC) }
	service.writer = failingSeedWriter{err: &kuzu.PersistenceError{
		Code:    "revision_anchor_unavailable",
		Message: "graph write could not record revision metadata",
		Details: map[string]any{"reason": "forced failure"},
	}}

	_, err := service.Init(context.Background(), repoDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	detail, ok := cmdsupport.ErrorDetailFromError(err)
	if !ok {
		t.Fatalf("error detail missing for %T", err)
	}
	if detail.Code != "workflow_seed_persistence_failed" {
		t.Fatalf("error code = %q, want workflow_seed_persistence_failed", detail.Code)
	}
	if detail.Type != "persistence_error" {
		t.Fatalf("error type = %q, want persistence_error", detail.Type)
	}
	if got := detail.Details["cause_code"]; got != "revision_anchor_unavailable" {
		t.Fatalf("cause_code = %#v, want revision_anchor_unavailable", got)
	}
}

func TestServiceInitReturnsStructuredAssetSyncError(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	blockedPath := filepath.Join(repoDir, filepath.FromSlash(managedWorkflowAssetPaths()[0]))
	if err := os.MkdirAll(blockedPath, 0o755); err != nil {
		t.Fatalf("os.MkdirAll blocked path: %v", err)
	}

	service := NewService(manager)
	service.now = func() time.Time { return time.Date(2026, 3, 22, 20, 30, 0, 0, time.UTC) }

	_, err := service.Init(context.Background(), repoDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	detail, ok := cmdsupport.ErrorDetailFromError(err)
	if !ok {
		t.Fatalf("error detail missing for %T", err)
	}
	if detail.Code != "workflow_asset_sync_failed" {
		t.Fatalf("error code = %q, want workflow_asset_sync_failed", detail.Code)
	}
	if detail.Type != "workflow_error" {
		t.Fatalf("error type = %q, want workflow_error", detail.Type)
	}
	if got := detail.Details["path"]; got != managedWorkflowAssetPaths()[0] {
		t.Fatalf("error path = %#v, want %q", got, managedWorkflowAssetPaths()[0])
	}
}

func TestServiceInitLeavesManagedAssetsUnchangedWhenRefreshFails(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	service := NewService(manager)
	service.now = func() time.Time { return time.Date(2026, 3, 22, 20, 45, 0, 0, time.UTC) }

	if _, err := service.Init(context.Background(), repoDir); err != nil {
		t.Fatalf("initial Init returned error: %v", err)
	}

	staleAssetPath := filepath.Join(repoDir, filepath.FromSlash(managedWorkflowAssetPaths()[0]))
	writeWorkflowFixture(t, staleAssetPath, "# stale asset\n")

	blockedPath := filepath.Join(repoDir, filepath.FromSlash(managedWorkflowAssetPaths()[1]))
	if err := os.Remove(blockedPath); err != nil {
		t.Fatalf("os.Remove blocked file: %v", err)
	}
	if err := os.MkdirAll(blockedPath, 0o755); err != nil {
		t.Fatalf("os.MkdirAll blocked path: %v", err)
	}

	service.now = func() time.Time { return time.Date(2026, 3, 22, 21, 0, 0, 0, time.UTC) }
	_, err := service.Init(context.Background(), repoDir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	detail, ok := cmdsupport.ErrorDetailFromError(err)
	if !ok {
		t.Fatalf("error detail missing for %T", err)
	}
	if detail.Code != "workflow_asset_sync_failed" {
		t.Fatalf("error code = %q, want workflow_asset_sync_failed", detail.Code)
	}
	if got := detail.Details["path"]; got != managedWorkflowAssetPaths()[1] {
		t.Fatalf("error path = %#v, want %q", got, managedWorkflowAssetPaths()[1])
	}

	payload, readErr := os.ReadFile(staleAssetPath)
	if readErr != nil {
		t.Fatalf("os.ReadFile stale asset after failure: %v", readErr)
	}
	if string(payload) != "# stale asset\n" {
		t.Fatalf("stale asset content = %q, want original stale content preserved on failure", string(payload))
	}
}

func TestServiceStartRecommendsProceedForHealthyGraph(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}
	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}

	writeWorkflowRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-301",
    "timestamp": "2026-03-22T21:15:00Z",
    "revision": {
      "reason": "Seed healthy workflow start graph"
    }
  },
  "nodes": [
    {
      "id": "story:workflow-start",
      "kind": "UserStory",
      "title": "Add workflow start readiness inspection",
      "summary": "Inspect graph readiness before delegated kickoff"
    },
    {
      "id": "doc:workflow",
      "kind": "Document",
      "title": "Delegated workflow architecture",
      "summary": "Explains readiness, kickoff, and handoff contracts"
    },
    {
      "id": "adr:workflow-readiness",
      "kind": "ADR",
      "title": "Use graph readiness checks before delegation",
      "summary": "Records why delegated work should inspect graph health first"
    }
  ],
  "edges": [
    {
      "from": "story:workflow-start",
      "to": "doc:workflow",
      "kind": "RELATES_TO"
    },
    {
      "from": "doc:workflow",
      "to": "adr:workflow-readiness",
      "kind": "CITES"
    },
    {
      "from": "adr:workflow-readiness",
      "to": "story:workflow-start",
      "kind": "ABOUT"
    }
  ]
}`)

	service := NewService(manager)
	result, err := service.Start(context.Background(), repoDir, "implement delegated workflow start", 1200)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if result.Recommendation != RecommendationProceed {
		t.Fatalf("recommendation = %q, want %q", result.Recommendation, RecommendationProceed)
	}
	if result.Readiness.Status != "ready" {
		t.Fatalf("status = %q, want ready", result.Readiness.Status)
	}
	if !result.Readiness.GraphState.WorkspaceInitialized {
		t.Fatal("expected workspace_initialized to be true")
	}
	if !result.Readiness.GraphState.GraphAvailable {
		t.Fatal("expected graph_available to be true")
	}
	if !result.Readiness.GraphState.CurrentRevision.Exists {
		t.Fatal("expected current revision to exist")
	}
	if !result.Readiness.GraphState.Health.Available {
		t.Fatal("expected health availability to be true")
	}
	if result.Readiness.GraphState.Health.Snapshot.Nodes != 3 || result.Readiness.GraphState.Health.Snapshot.Relationships != 3 {
		t.Fatalf("snapshot = %#v, want 3 nodes / 3 relationships", result.Readiness.GraphState.Health.Snapshot)
	}
}

func TestServiceStartRecommendsBootstrapForEmptyGraph(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}

	service := NewService(manager)
	result, err := service.Start(context.Background(), repoDir, "implement delegated workflow start", 1200)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if result.Recommendation != RecommendationBootstrap {
		t.Fatalf("recommendation = %q, want %q", result.Recommendation, RecommendationBootstrap)
	}
	if result.Readiness.GraphState.GraphAvailable {
		t.Fatal("expected graph_available to be false for empty graph")
	}
	if result.Readiness.GraphState.CurrentRevision.Exists {
		t.Fatal("expected current revision to be absent for empty graph")
	}
	if got := result.Readiness.Reasons; len(got) != 1 || got[0] != "graph_unavailable" {
		t.Fatalf("reasons = %#v, want [graph_unavailable]", got)
	}
}

func TestServiceStartRecommendsInspectHygieneForDuplicateHeavyGraph(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}
	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}

	writeWorkflowRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-302",
    "timestamp": "2026-03-22T21:30:00Z",
    "revision": {
      "reason": "Seed unhealthy workflow start graph"
    }
  },
  "nodes": [
    {
      "id": "doc:workflow-start-a",
      "kind": "Document",
      "title": "Workflow readiness guide",
      "summary": "Detailed delegated workflow readiness guide"
    },
    {
      "id": "doc:workflow-start-b",
      "kind": "Document",
      "title": "Workflow readiness guide",
      "summary": "Detailed delegated workflow readiness guide"
    },
    {
      "id": "story:handoff",
      "kind": "UserStory",
      "title": "Prepare delegated handoff",
      "summary": "Track delegated workflow handoff work"
    }
  ],
  "edges": [
    {
      "from": "doc:workflow-start-a",
      "to": "story:handoff",
      "kind": "RELATES_TO"
    },
    {
      "from": "doc:workflow-start-b",
      "to": "story:handoff",
      "kind": "RELATES_TO"
    }
  ]
}`)

	service := NewService(manager)
	result, err := service.Start(context.Background(), repoDir, "implement delegated workflow start", 1200)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if result.Recommendation != RecommendationInspectHygiene {
		t.Fatalf("recommendation = %q, want %q", result.Recommendation, RecommendationInspectHygiene)
	}
	if got := result.Readiness.Reasons; len(got) == 0 || got[0] != "duplication_rate_high" {
		t.Fatalf("reasons = %#v, want duplication_rate_high", got)
	}
	if result.Readiness.GraphState.Health.Indicators.DuplicationRate <= 0 {
		t.Fatalf("duplication_rate = %v, want > 0", result.Readiness.GraphState.Health.Indicators.DuplicationRate)
	}
}

func TestServiceStartBuildsGroundedKickoffEnvelopeForWellCoveredTask(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}
	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}

	writeWorkflowRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-303",
    "timestamp": "2026-03-22T21:45:00Z",
    "revision": {
      "reason": "Seed workflow finish kickoff context"
    }
  },
  "nodes": [
    {
      "id": "story:workflow-finish",
      "kind": "UserStory",
      "title": "Implement delegated workflow finish",
      "summary": "Persist structured handoff envelopes after delegated work completes"
    },
    {
      "id": "doc:kickoff-envelope",
      "kind": "Document",
      "title": "Kickoff envelope and delegation brief",
      "summary": "Describes task details, compact context, and prompt-ready briefing for workflow start"
    },
    {
      "id": "adr:delegated-workflow",
      "kind": "ADR",
      "title": "Thin delegated workflow orchestration",
      "summary": "Explains how kickoff and finish commands reuse graph retrieval and context projection"
    }
  ],
  "edges": [
    {
      "from": "story:workflow-finish",
      "to": "doc:kickoff-envelope",
      "kind": "RELATES_TO"
    },
    {
      "from": "doc:kickoff-envelope",
      "to": "adr:delegated-workflow",
      "kind": "CITES"
    },
    {
      "from": "adr:delegated-workflow",
      "to": "story:workflow-finish",
      "kind": "ABOUT"
    }
  ]
}`)

	service := NewService(manager)
	result, err := service.Start(context.Background(), repoDir, "implement delegated workflow finish", 1200)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if result.Recommendation != RecommendationProceed {
		t.Fatalf("recommendation = %q, want %q", result.Recommendation, RecommendationProceed)
	}
	if result.Kickoff.Task.Description != "implement delegated workflow finish" {
		t.Fatalf("kickoff task description = %q, want task text", result.Kickoff.Task.Description)
	}
	if result.Kickoff.Task.MaxTokens != 1200 {
		t.Fatalf("kickoff task max_tokens = %d, want 1200", result.Kickoff.Task.MaxTokens)
	}
	if result.Kickoff.Context.Coverage != KickoffCoverageGrounded {
		t.Fatalf("kickoff context coverage = %q, want %q", result.Kickoff.Context.Coverage, KickoffCoverageGrounded)
	}
	if len(result.Kickoff.Context.Envelope.Results) == 0 {
		t.Fatal("expected kickoff context results")
	}
	if got := result.Kickoff.Context.Envelope.Results[0].Entity.ID; got != "story:workflow-finish" {
		t.Fatalf("top kickoff context result = %q, want story:workflow-finish", got)
	}
	if result.Kickoff.GraphState.Nodes != 3 || result.Kickoff.GraphState.Relationships != 3 {
		t.Fatalf("kickoff graph state = %#v, want 3 nodes / 3 relationships", result.Kickoff.GraphState)
	}
	if result.Kickoff.DelegationBrief.Status != KickoffCoverageGrounded {
		t.Fatalf("delegation brief status = %q, want %q", result.Kickoff.DelegationBrief.Status, KickoffCoverageGrounded)
	}
	if !strings.Contains(result.Kickoff.DelegationBrief.Prompt, "Task: implement delegated workflow finish") {
		t.Fatalf("delegation brief prompt = %q, want task heading", result.Kickoff.DelegationBrief.Prompt)
	}
	if !strings.Contains(result.Kickoff.DelegationBrief.Prompt, "Kickoff envelope and delegation brief") {
		t.Fatalf("delegation brief prompt = %q, want retrieved context title", result.Kickoff.DelegationBrief.Prompt)
	}
}

func TestServiceStartBoundsKickoffContextByTokenBudget(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}
	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}

	writeWorkflowRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-304",
    "timestamp": "2026-03-22T21:50:00Z",
    "revision": {
      "reason": "Seed workflow finish budgeted kickoff context"
    }
  },
  "nodes": [
    {
      "id": "story:workflow-finish",
      "kind": "UserStory",
      "title": "Implement delegated workflow finish",
      "summary": "Persist structured handoff envelopes after delegated work completes and publish a machine-readable writeback result"
    },
    {
      "id": "doc:handoff-contract",
      "kind": "Document",
      "title": "Structured handoff contract",
      "summary": "Covers writeback payload fields, compact handoff notes, and revision-aware persistence for delegated completion"
    },
    {
      "id": "doc:kickoff-brief",
      "kind": "Document",
      "title": "Delegation brief examples",
      "summary": "Shows prompt-ready kickoff briefs, context prioritization, and concise summaries for delegated tasks"
    },
    {
      "id": "adr:workflow-finish",
      "kind": "ADR",
      "title": "Record structured handoff envelopes",
      "summary": "Explains why completion writeback should reuse compact context and structured envelopes instead of ad hoc prose"
    }
  ],
  "edges": [
    {
      "from": "story:workflow-finish",
      "to": "doc:handoff-contract",
      "kind": "RELATES_TO"
    },
    {
      "from": "story:workflow-finish",
      "to": "doc:kickoff-brief",
      "kind": "RELATES_TO"
    },
    {
      "from": "doc:handoff-contract",
      "to": "adr:workflow-finish",
      "kind": "CITES"
    },
    {
      "from": "doc:kickoff-brief",
      "to": "adr:workflow-finish",
      "kind": "CITES"
    }
  ]
}`)

	service := NewService(manager)
	result, err := service.Start(context.Background(), repoDir, "implement delegated workflow finish", 30)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if result.Kickoff.Context.Envelope.MaxTokens != 30 {
		t.Fatalf("kickoff context max_tokens = %d, want 30", result.Kickoff.Context.Envelope.MaxTokens)
	}
	if result.Kickoff.Context.Envelope.EstimatedTokens > 30 {
		t.Fatalf("estimated_tokens = %d, want <= 30", result.Kickoff.Context.Envelope.EstimatedTokens)
	}
	if !result.Kickoff.Context.Envelope.Truncated {
		t.Fatalf("kickoff context = %#v, want truncated envelope", result.Kickoff.Context.Envelope)
	}
	if result.Kickoff.Context.Envelope.OmittedResults == 0 {
		t.Fatalf("kickoff context omitted_results = %d, want > 0", result.Kickoff.Context.Envelope.OmittedResults)
	}
	if len(result.Kickoff.Context.Envelope.Results) == 0 {
		t.Fatal("expected at least one kickoff context result within budget")
	}
	if got := result.Kickoff.Context.Envelope.Results[0].Entity.ID; got != "story:workflow-finish" {
		t.Fatalf("top kickoff context result = %q, want story:workflow-finish", got)
	}
}

func TestServiceStartReturnsExplicitLowContextGuidanceWhenRetrievalIsSparse(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}
	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}

	writeWorkflowRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-305",
    "timestamp": "2026-03-22T22:00:00Z",
    "revision": {
      "reason": "Seed unrelated workflow context"
    }
  },
  "nodes": [
    {
      "id": "service:login-api",
      "kind": "Service",
      "title": "Login API",
      "summary": "Accepts login requests and delegates authentication checks"
    },
    {
      "id": "component:authentication",
      "kind": "Component",
      "title": "Authentication component",
      "summary": "Stores credential verification and token issuance logic"
    },
    {
      "id": "doc:auth-runbook",
      "kind": "Document",
      "title": "Authentication runbook",
      "summary": "Operational steps for login and token debugging"
    }
  ],
  "edges": [
    {
      "from": "service:login-api",
      "to": "component:authentication",
      "kind": "DEPENDS_ON"
    },
    {
      "from": "component:authentication",
      "to": "doc:auth-runbook",
      "kind": "DOCUMENTED_BY"
    },
    {
      "from": "doc:auth-runbook",
      "to": "service:login-api",
      "kind": "ABOUT"
    }
  ]
}`)

	service := NewService(manager)
	result, err := service.Start(context.Background(), repoDir, "investigate a new benchmark scenario", 1200)
	if err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	if result.Recommendation != RecommendationGatherContext {
		t.Fatalf("recommendation = %q, want %q", result.Recommendation, RecommendationGatherContext)
	}
	if got := result.Readiness.Reasons; len(got) == 0 || got[len(got)-1] != "task_context_sparse" {
		t.Fatalf("reasons = %#v, want task_context_sparse", got)
	}
	if result.Kickoff.Context.Coverage != KickoffCoverageLowContext {
		t.Fatalf("kickoff context coverage = %q, want %q", result.Kickoff.Context.Coverage, KickoffCoverageLowContext)
	}
	if len(result.Kickoff.Context.Envelope.Results) != 0 {
		t.Fatalf("kickoff context results = %#v, want empty low-context envelope", result.Kickoff.Context.Envelope.Results)
	}
	if len(result.Kickoff.Context.Guidance) == 0 {
		t.Fatal("expected explicit low-context guidance")
	}
	if result.Kickoff.DelegationBrief.Status != KickoffCoverageLowContext {
		t.Fatalf("delegation brief status = %q, want %q", result.Kickoff.DelegationBrief.Status, KickoffCoverageLowContext)
	}
	if !strings.Contains(result.Kickoff.DelegationBrief.Prompt, "Kickoff context (low_context)") {
		t.Fatalf("delegation brief prompt = %q, want low-context section", result.Kickoff.DelegationBrief.Prompt)
	}
	if !strings.Contains(result.Kickoff.DelegationBrief.Prompt, "Inspect repository sources") {
		t.Fatalf("delegation brief prompt = %q, want explicit next-best-action guidance", result.Kickoff.DelegationBrief.Prompt)
	}
}

func TestServiceStartReturnsStructuredErrorWhenInspectionFails(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}

	service := NewService(manager)
	service.ReadinessReaderForTest(failingReadinessReader{err: &kuzu.PersistenceError{
		Code:    "graph_read_failed",
		Message: "graph state could not be read",
		Details: map[string]any{"reason": "forced failure"},
	}})

	_, err := service.Start(context.Background(), repoDir, "implement delegated workflow start", 1200)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	detail, ok := cmdsupport.ErrorDetailFromError(err)
	if !ok {
		t.Fatalf("error detail missing for %T", err)
	}
	if detail.Code != "graph_read_failed" {
		t.Fatalf("error code = %q, want graph_read_failed", detail.Code)
	}
	if detail.Type != "persistence_error" {
		t.Fatalf("error type = %q, want persistence_error", detail.Type)
	}
	if got := detail.Details["stage"]; got != "current_revision" {
		t.Fatalf("error stage = %#v, want current_revision", got)
	}
}

func TestServiceFinishPersistsDelegatedOutcomeAndReturnsHandoffEnvelope(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}

	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}
	writeWorkflowRevision(t, workspace, `{
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
      "summary": "Seed graph state for workflow finish tests"
    }
  ],
  "edges": []
}`)

	service := NewService(manager)
	result, err := service.Finish(context.Background(), repoDir, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "finish-session",
    "timestamp": "2026-03-22T17:00:00Z"
  },
  "task": "implement workflow finish",
  "summary": "Persisted delegated handoff memory using the existing revision-aware write path.",
  "decisions": [
    {
      "summary": "Reuse the normal graph write flow for workflow finish",
      "rationale": "Keeps revisions and graph persistence consistent",
      "status": "accepted"
    }
  ],
  "changed_artifacts": [
    {
      "path": "internal/app/workflow/finish.go",
      "summary": "Added the workflow finish service and payload validation",
      "change_type": "updated",
      "language": "go"
    }
  ],
  "follow_up": [
    {
      "summary": "Wire repo-level workflow guidance to call graph workflow finish by default",
      "owner": "next-agent",
      "status": "open"
    }
  ]
}`)
	if err != nil {
		t.Fatalf("Finish returned error: %v", err)
	}

	if !result.BeforeRevision.Exists || !result.AfterRevision.Exists {
		t.Fatalf("revision state = %#v, want before and after revisions to exist", result)
	}
	if result.BeforeRevision.Revision.ID == result.AfterRevision.Revision.ID {
		t.Fatalf("before revision id = %q, want a new revision after finish", result.BeforeRevision.Revision.ID)
	}
	if result.WriteSummary.Status != FinishWriteStatusApplied {
		t.Fatalf("write status = %q, want %q", result.WriteSummary.Status, FinishWriteStatusApplied)
	}
	if result.WriteSummary.Nodes.CreatedCount != 4 || result.WriteSummary.Edges.CreatedCount != 3 {
		t.Fatalf("write summary = %#v, want 4 created nodes and 3 created edges", result.WriteSummary)
	}
	if result.HandoffBrief == nil {
		t.Fatal("expected handoff brief, got nil")
	}
	if result.HandoffBrief.Status != FinishHandoffStatusReady {
		t.Fatalf("handoff status = %q, want %q", result.HandoffBrief.Status, FinishHandoffStatusReady)
	}
	if !strings.Contains(result.HandoffBrief.Prompt, "Next-agent brief:") {
		t.Fatalf("handoff prompt = %q, want next-agent section", result.HandoffBrief.Prompt)
	}
	if result.NoOp != nil {
		t.Fatalf("no_op = %#v, want nil", result.NoOp)
	}

	graph, err := kuzu.NewStore().ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph returned error: %v", err)
	}
	if len(graph.Nodes) != 5 {
		t.Fatalf("graph node count = %d, want 5", len(graph.Nodes))
	}
	if len(graph.Edges) != 3 {
		t.Fatalf("graph edge count = %d, want 3", len(graph.Edges))
	}

	nodesByPath := mapGraphNodes(graph)
	artifactNode := mustGraphNode(t, nodesByPath, "internal/app/workflow/finish.go")
	if artifactNode.Kind != "Artifact" {
		t.Fatalf("artifact kind = %q, want Artifact", artifactNode.Kind)
	}
	if got := artifactNode.Props["workflow_command"]; got != "workflow.finish" {
		t.Fatalf("artifact workflow_command = %#v, want workflow.finish", got)
	}

	var reasoningNode kuzu.EntityRecord
	for _, node := range graph.Nodes {
		if node.Kind == "ReasoningUnit" && node.Title == "implement workflow finish" {
			reasoningNode = node
			break
		}
	}
	if reasoningNode.ID == "" {
		t.Fatal("expected workflow finish reasoning node to be persisted")
	}
	if reasoningNode.CreatedBy != "developer" || reasoningNode.UpdatedBy != "developer" {
		t.Fatalf("reasoning node provenance = (%q, %q), want developer", reasoningNode.CreatedBy, reasoningNode.UpdatedBy)
	}

	currentRevision, err := kuzu.NewStore().CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision returned error: %v", err)
	}
	if currentRevision.Revision.ID != result.AfterRevision.Revision.ID {
		t.Fatalf("current revision id = %q, want %q", currentRevision.Revision.ID, result.AfterRevision.Revision.ID)
	}
}

func TestServiceFinishReturnsExplicitNoOpWhenNoDurableUpdatesAreRequested(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}

	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}
	writeWorkflowRevision(t, workspace, `{
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
      "summary": "Seed graph state for workflow finish no-op tests"
    }
  ],
  "edges": []
}`)

	initialGraph, err := kuzu.NewStore().ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph before finish returned error: %v", err)
	}
	initialRevision, err := kuzu.NewStore().CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision before finish returned error: %v", err)
	}

	service := NewService(manager)
	result, err := service.Finish(context.Background(), repoDir, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "finish-session",
    "timestamp": "2026-03-22T17:15:00Z"
  },
  "task": "document workflow finish handoff",
  "summary": "No durable graph updates were needed for this delegated closeout.",
  "decisions": [],
  "changed_artifacts": [],
  "follow_up": []
}`)
	if err != nil {
		t.Fatalf("Finish returned error: %v", err)
	}

	if result.WriteSummary.Status != FinishWriteStatusNoOp {
		t.Fatalf("write status = %q, want %q", result.WriteSummary.Status, FinishWriteStatusNoOp)
	}
	if result.NoOp == nil {
		t.Fatal("expected explicit no_op result")
	}
	if result.HandoffBrief != nil {
		t.Fatalf("handoff_brief = %#v, want nil on no-op", result.HandoffBrief)
	}
	if result.BeforeRevision.Revision.ID != initialRevision.Revision.ID || result.AfterRevision.Revision.ID != initialRevision.Revision.ID {
		t.Fatalf("revision ids = (%q, %q), want unchanged revision %q", result.BeforeRevision.Revision.ID, result.AfterRevision.Revision.ID, initialRevision.Revision.ID)
	}

	currentRevision, err := kuzu.NewStore().CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision after finish returned error: %v", err)
	}
	if currentRevision.Revision.ID != initialRevision.Revision.ID {
		t.Fatalf("current revision id = %q, want %q", currentRevision.Revision.ID, initialRevision.Revision.ID)
	}

	graph, err := kuzu.NewStore().ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph after finish returned error: %v", err)
	}
	if len(graph.Nodes) != len(initialGraph.Nodes) || len(graph.Edges) != len(initialGraph.Edges) {
		t.Fatalf("graph changed on no-op finish: before=%#v after=%#v", initialGraph, graph)
	}
}

func TestServiceFinishExtractsExecutionTelemetryFromFinishPayloadMetadata(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}

	service := NewService(manager)
	result, err := service.Finish(context.Background(), repoDir, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "finish-session",
    "timestamp": "2026-03-22T17:15:00Z",
    "revision": {
      "properties": {
        "execution_usage": {
          "measurement_status": "complete",
          "source": "workflow_finish_payload",
          "provider": "copilot-cli",
          "input_tokens": 810,
          "output_tokens": 190,
          "total_tokens": 1000
        }
      }
    }
  },
  "task": "document workflow finish handoff",
  "summary": "No durable graph updates were needed for this delegated closeout.",
  "decisions": [],
  "changed_artifacts": [],
  "follow_up": []
}`)
	if err != nil {
		t.Fatalf("Finish returned error: %v", err)
	}

	if result.ExecutionTelemetry == nil {
		t.Fatal("expected execution telemetry, got nil")
	}
	if got := result.ExecutionTelemetry.MeasurementStatus; got != ExecutionTelemetryStatusComplete {
		t.Fatalf("measurement_status = %q, want %q", got, ExecutionTelemetryStatusComplete)
	}
	if result.ExecutionTelemetry.TotalTokens == nil || *result.ExecutionTelemetry.TotalTokens != 1000 {
		t.Fatalf("total_tokens = %#v, want 1000", result.ExecutionTelemetry.TotalTokens)
	}
	if got := result.ExecutionTelemetry.Provider; got != "copilot-cli" {
		t.Fatalf("provider = %q, want copilot-cli", got)
	}
}

func TestApplyExecutionTelemetryToFinishPayloadSetsExecutionUsageMetadata(t *testing.T) {
	t.Parallel()

	payload, err := ApplyExecutionTelemetryToFinishPayload(`{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "finish-session",
    "timestamp": "2026-03-22T17:15:00Z"
  },
  "task": "document workflow finish handoff",
  "summary": "No durable graph updates were needed for this delegated closeout.",
  "decisions": [],
  "changed_artifacts": [],
  "follow_up": []
}`, ExecutionTelemetry{
		MeasurementStatus: ExecutionTelemetryStatusComplete,
		Source:            "copilot_session_state",
		Provider:          "github-copilot-cli",
		InputTokens:       intPointer(810),
		OutputTokens:      intPointer(190),
		TotalTokens:       intPointer(1000),
	})
	if err != nil {
		t.Fatalf("ApplyExecutionTelemetryToFinishPayload returned error: %v", err)
	}

	telemetry, err := ExtractExecutionTelemetryFromFinishPayload(payload)
	if err != nil {
		t.Fatalf("ExtractExecutionTelemetryFromFinishPayload returned error: %v", err)
	}
	if telemetry == nil || telemetry.TotalTokens == nil || *telemetry.TotalTokens != 1000 {
		t.Fatalf("telemetry = %#v, want total_tokens=1000", telemetry)
	}
	if got := telemetry.Source; got != "copilot_session_state" {
		t.Fatalf("source = %q, want copilot_session_state", got)
	}
}

func TestServiceFinishRejectsUnknownNestedFieldsWithoutMutation(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}

	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}
	writeWorkflowRevision(t, workspace, `{
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
      "summary": "Seed graph state for workflow finish validation tests"
    }
  ],
  "edges": []
}`)

	initialGraph, err := kuzu.NewStore().ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph before finish returned error: %v", err)
	}
	initialRevision, err := kuzu.NewStore().CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision before finish returned error: %v", err)
	}

	service := NewService(manager)
	_, err = service.Finish(context.Background(), repoDir, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "finish-session",
    "timestamp": "2026-03-22T17:30:00Z"
  },
  "task": "reject unknown nested workflow finish payload",
  "summary": "This payload should fail validation without mutating the graph.",
  "decisions": [
    {
      "summary": "Reuse the workflow finish write path",
      "unexpected": "reject me"
    }
  ],
  "changed_artifacts": [],
  "follow_up": []
}`)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	detail, ok := cmdsupport.ErrorDetailFromError(err)
	if !ok {
		t.Fatalf("error detail missing for %T", err)
	}
	if detail.Code != "invalid_finish_payload" {
		t.Fatalf("error code = %q, want invalid_finish_payload", detail.Code)
	}
	if detail.Type != "validation_error" {
		t.Fatalf("error type = %q, want validation_error", detail.Type)
	}
	if got := detail.Details["field"]; got != "decisions" {
		t.Fatalf("error field = %#v, want decisions", got)
	}
	if got := detail.Details["index"]; got != 0 {
		t.Fatalf("error index = %#v, want 0", got)
	}
	if got, _ := detail.Details["reason"].(string); !strings.Contains(got, `unknown field "unexpected"`) {
		t.Fatalf("error reason = %#v, want unknown field detail", detail.Details["reason"])
	}

	currentRevision, err := kuzu.NewStore().CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision after finish returned error: %v", err)
	}
	if currentRevision.Revision.ID != initialRevision.Revision.ID {
		t.Fatalf("current revision id = %q, want %q after failed validation", currentRevision.Revision.ID, initialRevision.Revision.ID)
	}

	graph, err := kuzu.NewStore().ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph after finish returned error: %v", err)
	}
	if len(graph.Nodes) != len(initialGraph.Nodes) || len(graph.Edges) != len(initialGraph.Edges) {
		t.Fatalf("graph changed after failed validation: before=%#v after=%#v", initialGraph, graph)
	}
}

func TestServiceFinishRejectsWindowsTraversalPathsWithoutMutation(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkflowRepo(t)
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}

	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}
	writeWorkflowRevision(t, workspace, `{
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
      "summary": "Seed graph state for workflow finish validation tests"
    }
  ],
  "edges": []
}`)

	initialGraph, err := kuzu.NewStore().ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph before finish returned error: %v", err)
	}
	initialRevision, err := kuzu.NewStore().CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision before finish returned error: %v", err)
	}

	service := NewService(manager)
	_, err = service.Finish(context.Background(), repoDir, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "finish-session",
    "timestamp": "2026-03-22T17:30:00Z"
  },
  "task": "reject unsafe workflow finish payload",
  "summary": "This payload should fail validation without mutating the graph.",
  "decisions": [],
  "changed_artifacts": [
    {
      "path": "..\\secret.txt",
      "summary": "unsafe path"
    }
  ],
  "follow_up": []
}`)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	detail, ok := cmdsupport.ErrorDetailFromError(err)
	if !ok {
		t.Fatalf("error detail missing for %T", err)
	}
	if detail.Code != "unsafe_repo_path" {
		t.Fatalf("error code = %q, want unsafe_repo_path", detail.Code)
	}
	if detail.Type != "validation_error" {
		t.Fatalf("error type = %q, want validation_error", detail.Type)
	}
	if got := detail.Details["index"]; got != 0 {
		t.Fatalf("error index = %#v, want 0", got)
	}

	currentRevision, err := kuzu.NewStore().CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision after finish returned error: %v", err)
	}
	if currentRevision.Revision.ID != initialRevision.Revision.ID {
		t.Fatalf("current revision id = %q, want %q after failed validation", currentRevision.Revision.ID, initialRevision.Revision.ID)
	}

	graph, err := kuzu.NewStore().ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph after finish returned error: %v", err)
	}
	if len(graph.Nodes) != len(initialGraph.Nodes) || len(graph.Edges) != len(initialGraph.Edges) {
		t.Fatalf("graph changed after failed validation: before=%#v after=%#v", initialGraph, graph)
	}
}

func initWorkflowRepo(t *testing.T) (string, *repo.Manager) {
	t.Helper()

	repoDir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, output)
	}

	return repoDir, repo.NewManager(repo.NewGitRepositoryLocator())
}

func readManifest(t *testing.T, path string) Manifest {
	t.Helper()

	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile manifest: %v", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatalf("json.Unmarshal manifest: %v\npayload: %s", err, payload)
	}
	return manifest
}

func writeManifestFixture(t *testing.T, path string, manifest Manifest) {
	t.Helper()

	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent manifest: %v", err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("os.WriteFile manifest: %v", err)
	}
}

type failingSeedWriter struct {
	err error
}

func (w failingSeedWriter) Write(_ context.Context, _ repo.Workspace, _ graphpayload.Envelope) (kuzu.WriteSummary, error) {
	return kuzu.WriteSummary{}, w.err
}

type failingReadinessReader struct {
	err error
}

func (r failingReadinessReader) ReadGraph(_ context.Context, _ repo.Workspace) (kuzu.Graph, error) {
	return kuzu.Graph{}, r.err
}

func (r failingReadinessReader) CurrentRevision(_ context.Context, _ repo.Workspace) (kuzu.CurrentRevisionState, error) {
	return kuzu.CurrentRevisionState{}, r.err
}

func writeWorkflowRevision(t *testing.T, workspace repo.Workspace, payload string) kuzu.WriteSummary {
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

func mapGraphNodes(graph kuzu.Graph) map[string]kuzu.EntityRecord {
	nodes := make(map[string]kuzu.EntityRecord, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodes[node.RepoPath] = node
	}
	return nodes
}

func mustGraphNode(t *testing.T, nodes map[string]kuzu.EntityRecord, repoPath string) kuzu.EntityRecord {
	t.Helper()

	node, ok := nodes[repoPath]
	if !ok {
		t.Fatalf("graph node for repo_path %q not found", repoPath)
	}
	return node
}

func findSeededItem(t *testing.T, summary WorkSummary, path string) WorkItem {
	t.Helper()

	for _, item := range summary.Items {
		if item.Path == path {
			return item
		}
	}
	t.Fatalf("seeded item %q not found", path)
	return WorkItem{}
}

func manifestHasAsset(manifest Manifest, assetPath string) bool {
	for _, asset := range manifest.Assets {
		if asset.Path == assetPath {
			return true
		}
	}
	return false
}

func mustManifestAsset(t *testing.T, manifest Manifest, assetPath string) Asset {
	t.Helper()

	for _, asset := range manifest.Assets {
		if asset.Path == assetPath {
			return asset
		}
	}
	t.Fatalf("manifest asset %q not found", assetPath)
	return Asset{}
}
