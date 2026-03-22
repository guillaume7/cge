package hygienecmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/guillaume-galp/cge/internal/app/graphhealth"
	graphpayload "github.com/guillaume-galp/cge/internal/domain/payload"
	"github.com/guillaume-galp/cge/internal/infra/kuzu"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func TestHygieneCommandSuggestReturnsOrphanAndDuplicateCandidatesWithoutMutatingGraph(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initHygieneWorkspace(t)
	writeHygieneRevision(t, workspace, noisyGraphPayload)

	store := kuzu.NewStore()
	graphBefore, err := store.ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph before hygiene returned error: %v", err)
	}
	revisionBefore, err := store.CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision before hygiene returned error: %v", err)
	}

	cmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeHygieneSuggestResponse(t, stdout.Bytes())
	if response.Status != "ok" || response.Command != "hygiene" {
		t.Fatalf("response = %#v, want hygiene ok", response)
	}
	if response.Result.Mode != "suggest" {
		t.Fatalf("mode = %q, want suggest", response.Result.Mode)
	}
	if response.Result.Plan.SnapshotAnchor == "" {
		t.Fatal("expected snapshot anchor in hygiene plan")
	}

	orphans := response.Result.Plan.Suggestions.OrphanNodes
	if len(orphans) != 2 {
		t.Fatalf("orphan_nodes = %#v, want 2 orphan suggestions", orphans)
	}
	orphanByID := map[string]graphhealth.OrphanNode{}
	for _, orphan := range orphans {
		if orphan.ActionID == "" || orphan.NodeID == "" || orphan.Reason == "" {
			t.Fatalf("orphan suggestion missing machine-readable fields: %#v", orphan)
		}
		orphanByID[orphan.NodeID] = orphan
	}
	for _, nodeID := range []string{"note:orphan-ideas", "task:backlog-cleanup"} {
		orphan, ok := orphanByID[nodeID]
		if !ok {
			t.Fatalf("missing orphan suggestion for %q in %#v", nodeID, orphans)
		}
		if orphan.ActionID != "orphan:"+nodeID {
			t.Fatalf("orphan action_id = %q, want %q", orphan.ActionID, "orphan:"+nodeID)
		}
		if orphan.Reason != "node has no incoming or outgoing relationships in the current snapshot" {
			t.Fatalf("orphan reason = %q, want machine-readable orphan reason", orphan.Reason)
		}
	}

	duplicateGroups := response.Result.Plan.Suggestions.DuplicateGroups
	if len(duplicateGroups) != 1 {
		t.Fatalf("duplicate_groups = %#v, want 1 duplicate group", duplicateGroups)
	}
	duplicate := duplicateGroups[0]
	if duplicate.ActionID == "" || duplicate.CanonicalNodeID == "" || duplicate.Reason == "" {
		t.Fatalf("duplicate suggestion missing machine-readable fields: %#v", duplicate)
	}
	if !reflect.DeepEqual(duplicate.NodeIDs, []string{"doc:graph-stats-a", "doc:graph-stats-b"}) {
		t.Fatalf("duplicate node_ids = %#v, want graph stats pair", duplicate.NodeIDs)
	}
	if duplicate.CanonicalNodeID != "doc:graph-stats-a" {
		t.Fatalf("canonical_node_id = %q, want doc:graph-stats-a", duplicate.CanonicalNodeID)
	}
	if duplicate.ActionID != "duplicate:doc:graph-stats-a" {
		t.Fatalf("duplicate action_id = %q, want duplicate:doc:graph-stats-a", duplicate.ActionID)
	}
	if duplicate.Reason != "nodes share the same normalized title and text fingerprint" {
		t.Fatalf("duplicate reason = %q, want machine-readable duplicate reason", duplicate.Reason)
	}
	if duplicate.Signature == "" {
		t.Fatal("expected duplicate signature to be populated")
	}

	if response.Result.Plan.Suggestions.Contradictions == nil {
		t.Fatal("expected contradictions suggestions slice, got nil")
	}
	if len(response.Result.Plan.Suggestions.Contradictions) != 0 {
		t.Fatalf("contradictions = %#v, want none", response.Result.Plan.Suggestions.Contradictions)
	}

	graphAfter, err := store.ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph after hygiene returned error: %v", err)
	}
	if !graphsEqual(graphBefore, graphAfter) {
		t.Fatalf("graph changed after hygiene suggest\nbefore: %#v\nafter: %#v", graphBefore, graphAfter)
	}
	revisionAfter, err := store.CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision after hygiene returned error: %v", err)
	}
	if revisionBefore != revisionAfter {
		t.Fatalf("revision changed after hygiene suggest\nbefore: %#v\nafter: %#v", revisionBefore, revisionAfter)
	}
}

func TestHygieneCommandSuggestReturnsEmptyCandidateSetsForTidyGraph(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initHygieneWorkspace(t)
	writeHygieneRevision(t, workspace, tidyGraphPayload)

	cmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeHygieneSuggestResponse(t, stdout.Bytes())
	if response.Status != "ok" || response.Command != "hygiene" {
		t.Fatalf("response = %#v, want hygiene ok", response)
	}
	if response.Result.Mode != "suggest" {
		t.Fatalf("mode = %q, want suggest", response.Result.Mode)
	}
	if response.Result.Plan.Suggestions.OrphanNodes == nil {
		t.Fatal("expected orphan_nodes to be an empty array, got nil")
	}
	if len(response.Result.Plan.Suggestions.OrphanNodes) != 0 {
		t.Fatalf("orphan_nodes = %#v, want empty", response.Result.Plan.Suggestions.OrphanNodes)
	}
	if response.Result.Plan.Suggestions.DuplicateGroups == nil {
		t.Fatal("expected duplicate_groups to be an empty array, got nil")
	}
	if len(response.Result.Plan.Suggestions.DuplicateGroups) != 0 {
		t.Fatalf("duplicate_groups = %#v, want empty", response.Result.Plan.Suggestions.DuplicateGroups)
	}
	if response.Result.Plan.Suggestions.Contradictions == nil {
		t.Fatal("expected contradictions to be an empty array, got nil")
	}
	if len(response.Result.Plan.Suggestions.Contradictions) != 0 {
		t.Fatalf("contradictions = %#v, want empty", response.Result.Plan.Suggestions.Contradictions)
	}
}

func TestHygieneCommandReturnsStructuredOperationalErrorWhenWorkspaceIsMissing(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, output)
	}

	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	hygieneCmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	hygieneCmd.SetOut(stdout)
	hygieneCmd.SetErr(&bytes.Buffer{})

	err := hygieneCmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	response := decodeHygieneErrorResponse(t, stdout.Bytes())
	if response.Error.Category != "operational_error" {
		t.Fatalf("error category = %q, want operational_error", response.Error.Category)
	}
	if response.Error.Code != "workspace_not_initialized" {
		t.Fatalf("error code = %q, want workspace_not_initialized", response.Error.Code)
	}
}

func TestHygieneCommandOutputFlagWritesValidJSONToFile(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initHygieneWorkspace(t)
	writeHygieneRevision(t, workspace, noisyGraphPayload)

	stdoutCmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	stdoutCmd.SetOut(stdout)
	stdoutCmd.SetErr(&bytes.Buffer{})
	if err := stdoutCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("stdout ExecuteContext returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "hygiene.json")
	fileCmd := newCommand(repoDir, manager, nil)
	fileStdout := &bytes.Buffer{}
	fileCmd.SetOut(fileStdout)
	fileCmd.SetErr(&bytes.Buffer{})
	fileCmd.SetArgs([]string{"--output", outputPath})
	if err := fileCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("file ExecuteContext returned error: %v", err)
	}
	if fileStdout.Len() != 0 {
		t.Fatalf("expected no stdout when --output is used, got %q", fileStdout.String())
	}

	payload, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	response := decodeHygieneSuggestResponse(t, payload)
	if response.Status != "ok" || response.Command != "hygiene" || response.Result.Mode != "suggest" {
		t.Fatalf("file response = %#v, want hygiene suggest ok", response)
	}
	if !bytes.Equal(stdout.Bytes(), payload) {
		t.Fatalf("stdout payload != file payload\nstdout: %s\nfile: %s", stdout.Bytes(), payload)
	}
}

func initHygieneWorkspace(t *testing.T) (string, *repo.Manager, repo.Workspace) {
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

func writeHygieneRevision(t *testing.T, workspace repo.Workspace, payload string) kuzu.WriteSummary {
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

type hygieneSuggestResponse struct {
	SchemaVersion string `json:"schema_version"`
	Command       string `json:"command"`
	Status        string `json:"status"`
	Result        struct {
		Mode string                  `json:"mode"`
		Plan graphhealth.HygienePlan `json:"plan"`
	} `json:"result"`
}

type hygieneErrorResponse struct {
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

func decodeHygieneSuggestResponse(t *testing.T, payload []byte) hygieneSuggestResponse {
	t.Helper()

	var response hygieneSuggestResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal hygiene suggest response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeHygieneErrorResponse(t *testing.T, payload []byte) hygieneErrorResponse {
	t.Helper()

	var response hygieneErrorResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal hygiene error response: %v\npayload: %s", err, payload)
	}
	return response
}

func graphsEqual(left, right kuzu.Graph) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftPayload, rightPayload)
}

const noisyGraphPayload = `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-310",
    "timestamp": "2026-03-22T08:00:00Z",
    "revision": {
      "reason": "Seed noisy graph for hygiene suggest tests"
    }
  },
  "nodes": [
    {
      "id": "doc:graph-stats-a",
      "kind": "Document",
      "title": "Graph Stats",
      "summary": "Weekly graph metrics overview",
      "properties": {
        "ticket": "OPS-42"
      }
    },
    {
      "id": "doc:graph-stats-b",
      "kind": "Document",
      "title": "graph stats",
      "summary": "Weekly graph metrics overview"
    },
    {
      "id": "component:metrics-pipeline",
      "kind": "Component",
      "title": "Metrics Pipeline"
    },
    {
      "id": "story:dashboard",
      "kind": "UserStory",
      "title": "Build metrics dashboard"
    },
    {
      "id": "epic:observability",
      "kind": "Epic",
      "title": "Observability"
    },
    {
      "id": "note:orphan-ideas",
      "kind": "Note",
      "title": "Future cleanup ideas"
    },
    {
      "id": "task:backlog-cleanup",
      "kind": "Task",
      "title": "Backlog cleanup"
    }
  ],
  "edges": [
    {
      "from": "doc:graph-stats-a",
      "to": "component:metrics-pipeline",
      "kind": "ABOUT"
    },
    {
      "from": "doc:graph-stats-b",
      "to": "component:metrics-pipeline",
      "kind": "ABOUT"
    },
    {
      "from": "story:dashboard",
      "to": "epic:observability",
      "kind": "PART_OF"
    }
  ]
}`

const tidyGraphPayload = `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-311",
    "timestamp": "2026-03-22T08:05:00Z",
    "revision": {
      "reason": "Seed tidy graph for hygiene suggest tests"
    }
  },
  "nodes": [
    {
      "id": "story:stats-rollup",
      "kind": "UserStory",
      "title": "Implement stats rollup"
    },
    {
      "id": "epic:graph-health",
      "kind": "Epic",
      "title": "Graph health"
    },
    {
      "id": "component:cli",
      "kind": "Component",
      "title": "CLI surface"
    },
    {
      "id": "doc:operator-guide",
      "kind": "Document",
      "title": "Operator guide"
    }
  ],
  "edges": [
    {
      "from": "story:stats-rollup",
      "to": "epic:graph-health",
      "kind": "PART_OF"
    },
    {
      "from": "story:stats-rollup",
      "to": "component:cli",
      "kind": "IMPLEMENTS"
    },
    {
      "from": "doc:operator-guide",
      "to": "component:cli",
      "kind": "ABOUT"
    }
  ]
}`
