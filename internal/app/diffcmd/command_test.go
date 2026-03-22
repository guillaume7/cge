package diffcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	kuzudb "github.com/kuzudb/go-kuzu"

	graphpayload "github.com/guillaume-galp/cge/internal/domain/payload"
	"github.com/guillaume-galp/cge/internal/infra/kuzu"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func TestDiffCommandReturnsStructuredValidationErrorForMissingAnchors(t *testing.T) {
	t.Parallel()

	repoDir, manager, _ := initDiffWorkspace(t)
	cmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	response := decodeDiffErrorResponse(t, stdout.Bytes())
	if response.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want v1", response.SchemaVersion)
	}
	if response.Command != "diff" {
		t.Fatalf("command = %q, want diff", response.Command)
	}
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Error.Category != "validation_error" {
		t.Fatalf("error category = %q, want validation_error", response.Error.Category)
	}
	if response.Error.Code != "missing_revision_anchors" {
		t.Fatalf("error code = %q, want missing_revision_anchors", response.Error.Code)
	}
}

func TestDiffCommandComparesTwoPersistedRevisionsAndReturnsStructuredSuccess(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initDiffWorkspace(t)

	first := writeRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-42",
    "timestamp": "2026-03-21T14:00:00Z",
    "revision": {
      "reason": "Seed initial graph state"
    }
  },
  "nodes": [
    {
      "id": "story:TH1.E4.US2",
      "kind": "UserStory",
      "title": "Compare graph revisions with diff",
      "summary": "Initial story summary",
      "properties": {
        "priority": "medium",
        "status": "draft",
        "tags": ["story", "draft"]
      }
    }
  ],
  "edges": []
}`)

	second := writeRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-43",
    "timestamp": "2026-03-21T15:00:00Z",
    "revision": {
      "reason": "Refine story and connect it to the epic"
    }
  },
  "nodes": [
    {
      "id": "story:TH1.E4.US2",
      "kind": "UserStory",
      "title": "Compare graph revisions with diff",
      "summary": "Structured graph diff across persisted revisions",
      "properties": {
        "priority": "medium",
        "status": "ready",
        "tags": ["story", "draft"]
      }
    },
    {
      "id": "epic:E4",
      "kind": "Epic",
      "title": "Diff, trust, and interoperability"
    }
  ],
  "edges": [
    {
      "from": "story:TH1.E4.US2",
      "to": "epic:E4",
      "kind": "PART_OF",
      "properties": {
        "source": "planning"
      }
    }
  ]
}`)

	cmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--from", first.Revision.Anchor, "--to", second.Revision.Anchor})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeDiffSuccessResponse(t, stdout.Bytes())
	if response.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want v1", response.SchemaVersion)
	}
	if response.Command != "diff" {
		t.Fatalf("command = %q, want diff", response.Command)
	}
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}

	diff := response.Result.Diff
	if diff.From.Requested != first.Revision.Anchor {
		t.Fatalf("from requested = %q, want %q", diff.From.Requested, first.Revision.Anchor)
	}
	if diff.From.ID != first.Revision.ID {
		t.Fatalf("from id = %q, want %q", diff.From.ID, first.Revision.ID)
	}
	if diff.From.Reason != "Seed initial graph state" {
		t.Fatalf("from reason = %q, want Seed initial graph state", diff.From.Reason)
	}
	if diff.To.ID != second.Revision.ID {
		t.Fatalf("to id = %q, want %q", diff.To.ID, second.Revision.ID)
	}
	if diff.To.Anchor != second.Revision.Anchor {
		t.Fatalf("to anchor = %q, want %q", diff.To.Anchor, second.Revision.Anchor)
	}
	if diff.To.Reason != "Refine story and connect it to the epic" {
		t.Fatalf("to reason = %q, want Refine story and connect it to the epic", diff.To.Reason)
	}

	if diff.Summary.Entities.AddedCount != 1 || diff.Summary.Entities.UpdatedCount != 1 || diff.Summary.Entities.RemovedCount != 0 || diff.Summary.Entities.RetaggedCount != 0 {
		t.Fatalf("entity summary = %#v, want added=1 updated=1 removed=0 retagged=0", diff.Summary.Entities)
	}
	if diff.Summary.Relationships.AddedCount != 1 || diff.Summary.Relationships.UpdatedCount != 0 || diff.Summary.Relationships.RemovedCount != 0 || diff.Summary.Relationships.RetaggedCount != 0 {
		t.Fatalf("relationship summary = %#v, want added=1 updated=0 removed=0 retagged=0", diff.Summary.Relationships)
	}

	if len(diff.Entities.Added) != 1 || diff.Entities.Added[0].ID != "epic:E4" {
		t.Fatalf("added entities = %#v, want epic:E4", diff.Entities.Added)
	}
	if len(diff.Entities.Updated) != 1 {
		t.Fatalf("updated entities = %#v, want 1 updated entity", diff.Entities.Updated)
	}
	updated := diff.Entities.Updated[0]
	if updated.Before.ID != "story:TH1.E4.US2" || updated.After.ID != "story:TH1.E4.US2" {
		t.Fatalf("updated entity ids = (%q, %q), want story:TH1.E4.US2", updated.Before.ID, updated.After.ID)
	}
	if !slices.Equal(updated.ChangedFields, []string{"summary", "props"}) {
		t.Fatalf("updated changed_fields = %#v, want [summary props]", updated.ChangedFields)
	}
	if got := updated.Changes["summary"].From; got != "Initial story summary" {
		t.Fatalf("summary from = %#v, want Initial story summary", got)
	}
	if got := updated.Changes["summary"].To; got != "Structured graph diff across persisted revisions" {
		t.Fatalf("summary to = %#v, want Structured graph diff across persisted revisions", got)
	}

	if len(diff.Relationships.Added) != 1 {
		t.Fatalf("added relationships = %#v, want 1 relationship", diff.Relationships.Added)
	}
	addedEdge := diff.Relationships.Added[0]
	if addedEdge.From != "story:TH1.E4.US2" || addedEdge.Kind != "PART_OF" || addedEdge.To != "epic:E4" {
		t.Fatalf("added relationship = %#v, want story PART_OF epic", addedEdge)
	}
}

func TestDiffCommandDetectsRetaggedEntityMetadataChanges(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initDiffWorkspace(t)

	first := writeRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-50",
    "timestamp": "2026-03-21T16:00:00Z",
    "revision": {
      "reason": "Seed tagged skill"
    }
  },
  "nodes": [
    {
      "id": "artifact:graph-diff-skill",
      "kind": "Skill",
      "title": "Graph diff skill",
      "properties": {
        "tags": ["cli", "graph"]
      }
    }
  ],
  "edges": []
}`)

	second := writeRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-51",
    "timestamp": "2026-03-21T17:00:00Z",
    "revision": {
      "reason": "Retag graph diff skill"
    }
  },
  "nodes": [
    {
      "id": "artifact:graph-diff-skill",
      "kind": "Instruction",
      "title": "Graph diff skill",
      "properties": {
        "tags": ["agent", "graph"]
      }
    }
  ],
  "edges": []
}`)

	cmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--from", first.Revision.Anchor, "--to", second.Revision.Anchor})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeDiffSuccessResponse(t, stdout.Bytes())
	diff := response.Result.Diff
	if diff.Summary.Entities.RetaggedCount != 1 {
		t.Fatalf("retagged entity count = %d, want 1", diff.Summary.Entities.RetaggedCount)
	}
	if diff.Summary.Entities.UpdatedCount != 0 {
		t.Fatalf("updated entity count = %d, want 0", diff.Summary.Entities.UpdatedCount)
	}
	if len(diff.Entities.Retagged) != 1 {
		t.Fatalf("retagged entities = %#v, want 1 entity", diff.Entities.Retagged)
	}
	retagged := diff.Entities.Retagged[0]
	if !slices.Equal(retagged.ChangedFields, []string{"kind", "tags"}) {
		t.Fatalf("retagged changed_fields = %#v, want [kind tags]", retagged.ChangedFields)
	}
	if retagged.Changes["kind"].From != "Skill" || retagged.Changes["kind"].To != "Instruction" {
		t.Fatalf("kind change = %#v, want Skill -> Instruction", retagged.Changes["kind"])
	}
	if !slices.Equal(retagged.Changes["tags"].Added, []string{"agent"}) {
		t.Fatalf("added tags = %#v, want [agent]", retagged.Changes["tags"].Added)
	}
	if !slices.Equal(retagged.Changes["tags"].Removed, []string{"cli"}) {
		t.Fatalf("removed tags = %#v, want [cli]", retagged.Changes["tags"].Removed)
	}
}

func TestDiffCommandReturnsStructuredErrorForMissingRevisionAnchor(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initDiffWorkspace(t)
	first := writeRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-60",
    "timestamp": "2026-03-21T18:00:00Z",
    "revision": {
      "reason": "Only known revision"
    }
  },
  "nodes": [
    {
      "id": "story:known",
      "kind": "UserStory",
      "title": "Known revision"
    }
  ],
  "edges": []
}`)

	cmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--from", first.Revision.Anchor, "--to", "missing-anchor"})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	response := decodeDiffErrorResponse(t, stdout.Bytes())
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Error.Category != "operational_error" {
		t.Fatalf("error category = %q, want operational_error", response.Error.Category)
	}
	if response.Error.Type != "diff_error" {
		t.Fatalf("error type = %q, want diff_error", response.Error.Type)
	}
	if response.Error.Code != "revision_anchor_not_found" {
		t.Fatalf("error code = %q, want revision_anchor_not_found", response.Error.Code)
	}
	if response.Error.Details["flag"] != "to" {
		t.Fatalf("error flag = %#v, want to", response.Error.Details["flag"])
	}
	if response.Error.Details["anchor"] != "missing-anchor" {
		t.Fatalf("error anchor = %#v, want missing-anchor", response.Error.Details["anchor"])
	}
}

func TestDiffCommandDetectsRetaggedRelationshipChanges(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initDiffWorkspace(t)
	first := writeRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-80",
    "timestamp": "2026-03-21T21:00:00Z",
    "revision": {
      "reason": "Seed relationship retag test"
    }
  },
  "nodes": [
    {
      "id": "story:rel-retag",
      "kind": "UserStory",
      "title": "Relationship retag"
    },
    {
      "id": "epic:rel-retag",
      "kind": "Epic",
      "title": "Relationship retag epic"
    }
  ],
  "edges": [
    {
      "from": "story:rel-retag",
      "to": "epic:rel-retag",
      "kind": "PART_OF",
      "properties": {
        "workflow": "seed"
      }
    }
  ]
}`)

	mutateGraphState(t, repoDir,
		`MATCH (from:Entity {id: 'story:rel-retag'})-[r:EntityRelation {kind: 'PART_OF'}]->(to:Entity {id: 'epic:rel-retag'}) DELETE r;`,
		`MATCH (from:Entity {id: 'story:rel-retag'}), (to:Entity {id: 'epic:rel-retag'})
CREATE (from)-[:EntityRelation {kind: 'DEPENDS_ON', props_json: '{"workflow":"cleanup"}'}]->(to);`,
	)

	second := writeRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-81",
    "timestamp": "2026-03-21T22:00:00Z",
    "revision": {
      "reason": "Record relationship retag"
    }
  },
  "nodes": [
    {
      "id": "story:rel-retag",
      "kind": "UserStory",
      "title": "Relationship retag"
    },
    {
      "id": "epic:rel-retag",
      "kind": "Epic",
      "title": "Relationship retag epic"
    }
  ],
  "edges": []
}`)

	response := runDiffCommand(t, repoDir, manager, first.Revision.Anchor, second.Revision.Anchor)
	diff := response.Result.Diff
	if diff.Summary.Relationships.RetaggedCount != 1 {
		t.Fatalf("retagged relationship count = %d, want 1", diff.Summary.Relationships.RetaggedCount)
	}
	if diff.Summary.Relationships.AddedCount != 0 || diff.Summary.Relationships.RemovedCount != 0 {
		t.Fatalf("relationship summary = %#v, want added=0 removed=0", diff.Summary.Relationships)
	}
	if len(diff.Relationships.Retagged) != 1 {
		t.Fatalf("retagged relationships = %#v, want 1 relationship", diff.Relationships.Retagged)
	}
	retagged := diff.Relationships.Retagged[0]
	if !slices.Equal(retagged.ChangedFields, []string{"kind", "props"}) {
		t.Fatalf("retagged changed_fields = %#v, want [kind props]", retagged.ChangedFields)
	}
	if retagged.Before.Kind != "PART_OF" || retagged.After.Kind != "DEPENDS_ON" {
		t.Fatalf("relationship kinds = (%q, %q), want PART_OF -> DEPENDS_ON", retagged.Before.Kind, retagged.After.Kind)
	}
}

func TestDiffCommandDetectsRemovedRelationship(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initDiffWorkspace(t)
	first := writeRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-82",
    "timestamp": "2026-03-21T21:00:00Z",
    "revision": {
      "reason": "Seed relationship removal test"
    }
  },
  "nodes": [
    {
      "id": "story:rel-remove",
      "kind": "UserStory",
      "title": "Relationship removal"
    },
    {
      "id": "epic:rel-remove",
      "kind": "Epic",
      "title": "Relationship removal epic"
    }
  ],
  "edges": [
    {
      "from": "story:rel-remove",
      "to": "epic:rel-remove",
      "kind": "PART_OF"
    }
  ]
}`)

	mutateGraphState(t, repoDir,
		`MATCH (from:Entity {id: 'story:rel-remove'})-[r:EntityRelation {kind: 'PART_OF'}]->(to:Entity {id: 'epic:rel-remove'}) DELETE r;`,
	)

	second := writeRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-83",
    "timestamp": "2026-03-21T22:00:00Z",
    "revision": {
      "reason": "Record relationship removal"
    }
  },
  "nodes": [
    {
      "id": "story:rel-remove",
      "kind": "UserStory",
      "title": "Relationship removal"
    },
    {
      "id": "epic:rel-remove",
      "kind": "Epic",
      "title": "Relationship removal epic"
    }
  ],
  "edges": []
}`)

	response := runDiffCommand(t, repoDir, manager, first.Revision.Anchor, second.Revision.Anchor)
	diff := response.Result.Diff
	if diff.Summary.Relationships.RemovedCount != 1 {
		t.Fatalf("removed relationship count = %d, want 1", diff.Summary.Relationships.RemovedCount)
	}
	if len(diff.Relationships.Removed) != 1 {
		t.Fatalf("removed relationships = %#v, want 1 relationship", diff.Relationships.Removed)
	}
	removed := diff.Relationships.Removed[0]
	if removed.From != "story:rel-remove" || removed.Kind != "PART_OF" || removed.To != "epic:rel-remove" {
		t.Fatalf("removed relationship = %#v, want story PART_OF epic", removed)
	}
}

func TestDiffCommandDetectsUpdatedRelationshipProperties(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initDiffWorkspace(t)
	first := writeRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-84",
    "timestamp": "2026-03-21T21:00:00Z",
    "revision": {
      "reason": "Seed relationship update test"
    }
  },
  "nodes": [
    {
      "id": "story:rel-update",
      "kind": "UserStory",
      "title": "Relationship update"
    },
    {
      "id": "epic:rel-update",
      "kind": "Epic",
      "title": "Relationship update epic"
    }
  ],
  "edges": [
    {
      "from": "story:rel-update",
      "to": "epic:rel-update",
      "kind": "PART_OF",
      "properties": {
        "workflow": "seed"
      }
    }
  ]
}`)

	second := writeRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-85",
    "timestamp": "2026-03-21T22:00:00Z",
    "revision": {
      "reason": "Update relationship properties"
    }
  },
  "nodes": [
    {
      "id": "story:rel-update",
      "kind": "UserStory",
      "title": "Relationship update"
    },
    {
      "id": "epic:rel-update",
      "kind": "Epic",
      "title": "Relationship update epic"
    }
  ],
  "edges": [
    {
      "from": "story:rel-update",
      "to": "epic:rel-update",
      "kind": "PART_OF",
      "properties": {
        "workflow": "cleanup"
      }
    }
  ]
}`)

	response := runDiffCommand(t, repoDir, manager, first.Revision.Anchor, second.Revision.Anchor)
	diff := response.Result.Diff
	if diff.Summary.Relationships.UpdatedCount != 1 {
		t.Fatalf("updated relationship count = %d, want 1", diff.Summary.Relationships.UpdatedCount)
	}
	if len(diff.Relationships.Updated) != 1 {
		t.Fatalf("updated relationships = %#v, want 1 relationship", diff.Relationships.Updated)
	}
	updated := diff.Relationships.Updated[0]
	if !slices.Equal(updated.ChangedFields, []string{"props"}) {
		t.Fatalf("updated changed_fields = %#v, want [props]", updated.ChangedFields)
	}
	if got := updated.Changes["props"].From.(map[string]any)["workflow"]; got != "seed" {
		t.Fatalf("relationship props.from.workflow = %#v, want seed", got)
	}
	if got := updated.Changes["props"].To.(map[string]any)["workflow"]; got != "cleanup" {
		t.Fatalf("relationship props.to.workflow = %#v, want cleanup", got)
	}
}

func TestDiffCommandDoesNotFabricateRetaggedRelationshipWhenMultipleCandidatesShareEndpoints(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initDiffWorkspace(t)
	first := writeRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-86",
    "timestamp": "2026-03-21T21:00:00Z",
    "revision": {
      "reason": "Seed ambiguous retag test"
    }
  },
  "nodes": [
    {
      "id": "story:rel-ambiguous",
      "kind": "UserStory",
      "title": "Ambiguous relationship retag"
    },
    {
      "id": "epic:rel-ambiguous",
      "kind": "Epic",
      "title": "Ambiguous relationship retag epic"
    }
  ],
  "edges": [
    {
      "from": "story:rel-ambiguous",
      "to": "epic:rel-ambiguous",
      "kind": "PART_OF"
    }
  ]
}`)

	mutateGraphState(t, repoDir,
		`MATCH (from:Entity {id: 'story:rel-ambiguous'})-[r:EntityRelation {kind: 'PART_OF'}]->(to:Entity {id: 'epic:rel-ambiguous'}) DELETE r;`,
		`MATCH (from:Entity {id: 'story:rel-ambiguous'}), (to:Entity {id: 'epic:rel-ambiguous'})
CREATE (from)-[:EntityRelation {kind: 'DEPENDS_ON'}]->(to);`,
		`MATCH (from:Entity {id: 'story:rel-ambiguous'}), (to:Entity {id: 'epic:rel-ambiguous'})
CREATE (from)-[:EntityRelation {kind: 'RELATES_TO'}]->(to);`,
	)

	second := writeRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-87",
    "timestamp": "2026-03-21T22:00:00Z",
    "revision": {
      "reason": "Record ambiguous relationship change"
    }
  },
  "nodes": [
    {
      "id": "story:rel-ambiguous",
      "kind": "UserStory",
      "title": "Ambiguous relationship retag"
    },
    {
      "id": "epic:rel-ambiguous",
      "kind": "Epic",
      "title": "Ambiguous relationship retag epic"
    }
  ],
  "edges": []
}`)

	response := runDiffCommand(t, repoDir, manager, first.Revision.Anchor, second.Revision.Anchor)
	diff := response.Result.Diff
	if diff.Summary.Relationships.RetaggedCount != 0 {
		t.Fatalf("retagged relationship count = %d, want 0", diff.Summary.Relationships.RetaggedCount)
	}
	if diff.Summary.Relationships.RemovedCount != 1 {
		t.Fatalf("removed relationship count = %d, want 1", diff.Summary.Relationships.RemovedCount)
	}
	if diff.Summary.Relationships.AddedCount != 2 {
		t.Fatalf("added relationship count = %d, want 2", diff.Summary.Relationships.AddedCount)
	}
	if len(diff.Relationships.Removed) != 1 {
		t.Fatalf("removed relationships = %#v, want 1 relationship", diff.Relationships.Removed)
	}
	if len(diff.Relationships.Added) != 2 {
		t.Fatalf("added relationships = %#v, want 2 relationships", diff.Relationships.Added)
	}
	gotKinds := []string{diff.Relationships.Added[0].Kind, diff.Relationships.Added[1].Kind}
	if !slices.Equal(gotKinds, []string{"DEPENDS_ON", "RELATES_TO"}) {
		t.Fatalf("added relationship kinds = %#v, want [DEPENDS_ON RELATES_TO]", gotKinds)
	}
}

func initDiffWorkspace(t *testing.T) (string, *repo.Manager, repo.Workspace) {
	t.Helper()

	repoDir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, output)
	}

	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace returned error: %v", err)
	}
	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace returned error: %v", err)
	}

	return repoDir, manager, workspace
}

func writeRevision(t *testing.T, workspace repo.Workspace, payload string) kuzu.WriteSummary {
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

func runDiffCommand(t *testing.T, repoDir string, manager *repo.Manager, from, to string) diffSuccessResponse {
	t.Helper()

	cmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--from", from, "--to", to})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	return decodeDiffSuccessResponse(t, stdout.Bytes())
}

func mutateGraphState(t *testing.T, repoDir string, queries ...string) {
	t.Helper()

	dbPath := filepath.Join(repoDir, repo.WorkspaceDirName, "kuzu", kuzu.StoreFileName)
	db, err := kuzudb.OpenDatabase(dbPath, kuzudb.DefaultSystemConfig())
	if err != nil {
		t.Fatalf("open writable kuzu database: %v", err)
	}
	defer db.Close()

	conn, err := kuzudb.OpenConnection(db)
	if err != nil {
		t.Fatalf("open writable kuzu connection: %v", err)
	}
	defer conn.Close()

	for _, query := range queries {
		result, err := conn.Query(query)
		if result != nil {
			result.Close()
		}
		if err != nil {
			t.Fatalf("execute mutation query %q: %v", query, err)
		}
	}
}

type diffSuccessResponse struct {
	SchemaVersion string `json:"schema_version"`
	Command       string `json:"command"`
	Status        string `json:"status"`
	Result        struct {
		Diff kuzu.GraphDiff `json:"diff"`
	} `json:"result"`
}

type diffErrorResponse struct {
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

func decodeDiffSuccessResponse(t *testing.T, payload []byte) diffSuccessResponse {
	t.Helper()

	var response diffSuccessResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal diff success response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeDiffErrorResponse(t *testing.T, payload []byte) diffErrorResponse {
	t.Helper()

	var response diffErrorResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal diff error response: %v\npayload: %s", err, payload)
	}
	return response
}

func TestDiffCommandWritesEquivalentStructuredSuccessToFile(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initDiffWorkspace(t)
	first := writeRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-70",
    "timestamp": "2026-03-21T19:00:00Z",
    "revision": {
      "reason": "Seed file-mode diff test"
    }
  },
  "nodes": [
    {
      "id": "story:file-mode",
      "kind": "UserStory",
      "title": "File mode"
    }
  ],
  "edges": []
}`)
	second := writeRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-71",
    "timestamp": "2026-03-21T20:00:00Z",
    "revision": {
      "reason": "Add relation for file-mode diff test"
    }
  },
  "nodes": [
    {
      "id": "story:file-mode",
      "kind": "UserStory",
      "title": "File mode updated"
    },
    {
      "id": "epic:file-mode",
      "kind": "Epic",
      "title": "Epic"
    }
  ],
  "edges": [
    {
      "from": "story:file-mode",
      "to": "epic:file-mode",
      "kind": "PART_OF"
    }
  ]
}`)

	stdoutCmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	stdoutCmd.SetOut(stdout)
	stdoutCmd.SetErr(&bytes.Buffer{})
	stdoutCmd.SetArgs([]string{"--from", first.Revision.Anchor, "--to", second.Revision.Anchor})
	if err := stdoutCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("stdout ExecuteContext returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "diff.json")
	fileCmd := newCommand(repoDir, manager, nil)
	fileStdout := &bytes.Buffer{}
	fileCmd.SetOut(fileStdout)
	fileCmd.SetErr(&bytes.Buffer{})
	fileCmd.SetArgs([]string{"--from", first.Revision.Anchor, "--to", second.Revision.Anchor, "--output", outputPath})
	if err := fileCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("file ExecuteContext returned error: %v", err)
	}

	if got := fileStdout.String(); got != "" {
		t.Fatalf("stdout in file mode = %q, want empty", got)
	}

	filePayload, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("os.ReadFile output: %v", err)
	}
	if !bytes.Equal(filePayload, stdout.Bytes()) {
		t.Fatalf("file payload != stdout payload\nstdout: %s\nfile: %s", stdout.Bytes(), filePayload)
	}
}
