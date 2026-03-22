package writecmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	kuzudb "github.com/kuzudb/go-kuzu"
	"github.com/spf13/cobra"

	graphpayload "github.com/guillaume-galp/cge/internal/domain/payload"
	"github.com/guillaume-galp/cge/internal/infra/kuzu"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func TestWriteCommandAcceptsValidatedPayloadFromStdin(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkspace(t)
	persister := &spyPersister{}

	cmd := newCommand(repoDir, manager, persister)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader(validPayloadJSON))

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if persister.calls != 1 {
		t.Fatalf("persist calls = %d, want 1", persister.calls)
	}
	if persister.lastEnvelope.Metadata.AgentID != "developer" {
		t.Fatalf("agent id = %q, want developer", persister.lastEnvelope.Metadata.AgentID)
	}
	response := decodeSuccessResponse(t, stdout.Bytes())
	if response.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want v1", response.SchemaVersion)
	}
	if response.Command != "write" {
		t.Fatalf("command = %q, want write", response.Command)
	}
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
}

func TestWriteCommandAcceptsValidatedPayloadFromFile(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkspace(t)
	persister := &spyPersister{}
	payloadPath := filepath.Join(t.TempDir(), "payload.json")
	if err := os.WriteFile(payloadPath, []byte(validPayloadJSON), 0o644); err != nil {
		t.Fatalf("write payload file: %v", err)
	}

	cmd := newCommand(repoDir, manager, persister)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--file", payloadPath})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if persister.calls != 1 {
		t.Fatalf("persist calls = %d, want 1", persister.calls)
	}
	response := decodeSuccessResponse(t, stdout.Bytes())
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
}

func TestWriteCommandWritesEquivalentStructuredOutputToFile(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkspace(t)
	persister := &spyPersister{}

	stdoutCmd := newCommand(repoDir, manager, persister)
	stdout := &bytes.Buffer{}
	stdoutCmd.SetOut(stdout)
	stdoutCmd.SetErr(&bytes.Buffer{})
	stdoutCmd.SetIn(strings.NewReader(validPayloadJSON))

	if err := stdoutCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext stdout returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "write.json")
	fileCmd := newCommand(repoDir, manager, persister)
	fileStdout := &bytes.Buffer{}
	fileCmd.SetOut(fileStdout)
	fileCmd.SetErr(&bytes.Buffer{})
	fileCmd.SetIn(strings.NewReader(validPayloadJSON))
	fileCmd.SetArgs([]string{"--output", outputPath})

	if err := fileCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext file returned error: %v", err)
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

func TestWriteCommandRejectsMissingProvenanceWithStructuredError(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkspace(t)
	persister := &spyPersister{}

	cmd := newCommand(repoDir, manager, persister)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader(`{
  "schema_version": "v1",
  "metadata": {
    "timestamp": "2026-03-21T14:00:00Z"
  },
  "nodes": [],
  "edges": []
}`))

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if persister.calls != 0 {
		t.Fatalf("persist calls = %d, want 0", persister.calls)
	}

	response := decodeErrorResponse(t, stdout.Bytes())
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Error.Category != "validation_error" {
		t.Fatalf("error category = %q, want validation_error", response.Error.Category)
	}
	if response.Error.Type != "validation_error" {
		t.Fatalf("error type = %q, want validation_error", response.Error.Type)
	}
	if response.Error.Code != "missing_required_fields" {
		t.Fatalf("error code = %q, want missing_required_fields", response.Error.Code)
	}

	missingFields := asStrings(t, response.Error.Details["missing_fields"])
	wantMissing := []string{"metadata.agent_id", "metadata.session_id"}
	if strings.Join(missingFields, ",") != strings.Join(wantMissing, ",") {
		t.Fatalf("missing fields = %#v, want %#v", missingFields, wantMissing)
	}
}

func TestWriteCommandPersistsMixedPayloadAndReportsSummary(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkspace(t)

	cmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader(`{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-42",
    "timestamp": "2026-03-21T14:00:00Z"
  },
  "nodes": [
    {
      "id": "project:vp1",
      "kind": "ProjectMetadata",
      "title": "VP1 MVP",
      "properties": {
        "summary": "Cognitive graph engine MVP",
        "tags": ["project", "metadata"],
        "owner": "developer"
      }
    },
    {
      "id": "story:TH1.E2.US1",
      "kind": "UserStory",
      "title": "Persist entities and relationships",
      "summary": "Store entity-centric writes",
      "properties": {
        "priority": "high",
        "tags": ["planning", "story"]
      }
    },
    {
      "id": "code:internal/app/writecmd/command.go",
      "kind": "CodeEntity",
      "title": "write command",
      "repo_path": "internal/app/writecmd/command.go",
      "language": "go",
      "properties": {
        "symbol": "newCommand",
        "tags": ["codebase"]
      }
    }
  ],
  "edges": [
    {
      "from": "story:TH1.E2.US1",
      "to": "project:vp1",
      "kind": "PART_OF"
    },
    {
      "from": "code:internal/app/writecmd/command.go",
      "to": "story:TH1.E2.US1",
      "kind": "ABOUT",
      "properties": {
        "confidence": "high"
      }
    }
  ]
}`))

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeSuccessResponse(t, stdout.Bytes())
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if response.Result.Summary.Nodes.CreatedCount != 3 || response.Result.Summary.Nodes.UpdatedCount != 0 {
		t.Fatalf("node summary = %#v, want 3 created and 0 updated", response.Result.Summary.Nodes)
	}
	if response.Result.Summary.Edges.CreatedCount != 2 || response.Result.Summary.Edges.UpdatedCount != 0 {
		t.Fatalf("edge summary = %#v, want 2 created and 0 updated", response.Result.Summary.Edges)
	}

	state := readGraphState(t, repoDir)
	if len(state.Nodes) != 3 {
		t.Fatalf("persisted nodes = %d, want 3", len(state.Nodes))
	}
	if len(state.Edges) != 2 {
		t.Fatalf("persisted edges = %d, want 2", len(state.Edges))
	}
	if got := state.Nodes["project:vp1"].Kind; got != "ProjectMetadata" {
		t.Fatalf("project node kind = %q, want ProjectMetadata", got)
	}
	if got := state.Nodes["story:TH1.E2.US1"].Kind; got != "UserStory" {
		t.Fatalf("story node kind = %q, want UserStory", got)
	}
	if got := state.Nodes["code:internal/app/writecmd/command.go"].Kind; got != "CodeEntity" {
		t.Fatalf("code node kind = %q, want CodeEntity", got)
	}
}

func TestWriteCommandUpsertsExistingEntityByStableID(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkspace(t)

	runWriteCommand(t, repoDir, manager, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-42",
    "timestamp": "2026-03-21T14:00:00Z"
  },
  "nodes": [
    {
      "id": "story:TH1.E2.US1",
      "kind": "UserStory",
      "title": "Persist entities",
      "summary": "Initial summary"
    }
  ],
  "edges": []
}`)

	response := runWriteCommand(t, repoDir, manager, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-43",
    "timestamp": "2026-03-21T15:00:00Z"
  },
  "nodes": [
    {
      "id": "story:TH1.E2.US1",
      "kind": "UserStory",
      "title": "Persist entities and relationships",
      "summary": "Updated summary"
    }
  ],
  "edges": []
}`)

	if response.Result.Summary.Nodes.CreatedCount != 0 || response.Result.Summary.Nodes.UpdatedCount != 1 {
		t.Fatalf("node summary = %#v, want 0 created and 1 updated", response.Result.Summary.Nodes)
	}

	state := readGraphState(t, repoDir)
	if len(state.Nodes) != 1 {
		t.Fatalf("persisted nodes = %d, want 1", len(state.Nodes))
	}

	node := state.Nodes["story:TH1.E2.US1"]
	if node.Title != "Persist entities and relationships" {
		t.Fatalf("node title = %q, want updated title", node.Title)
	}
	if node.Summary != "Updated summary" {
		t.Fatalf("node summary = %q, want updated summary", node.Summary)
	}
	if node.CreatedSessionID != "sess-42" {
		t.Fatalf("created session id = %q, want sess-42", node.CreatedSessionID)
	}
	if node.UpdatedSessionID != "sess-43" {
		t.Fatalf("updated session id = %q, want sess-43", node.UpdatedSessionID)
	}
}

func TestWriteCommandRecordsRevisionAnchorForUpdatedEntity(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkspace(t)

	first := runWriteCommand(t, repoDir, manager, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-42",
    "timestamp": "2026-03-21T14:00:00Z",
    "revision": {
      "reason": "Seed the initial story record",
      "properties": {
        "workflow": "seed"
      }
    }
  },
  "nodes": [
    {
      "id": "story:TH1.E2.US3",
      "kind": "UserStory",
      "title": "Support graph updates",
      "summary": "Original summary"
    }
  ],
  "edges": []
}`)

	second := runWriteCommand(t, repoDir, manager, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-43",
    "timestamp": "2026-03-21T15:00:00Z",
    "revision": {
      "reason": "Refresh stale story summary",
      "properties": {
        "workflow": "cleanup",
        "source": "agent-review"
      }
    }
  },
  "nodes": [
    {
      "id": "story:TH1.E2.US3",
      "kind": "UserStory",
      "title": "Support graph updates and revision anchors",
      "summary": "Updated summary"
    }
  ],
  "edges": []
}`)

	if first.Result.Summary.Revision.ID == "" || first.Result.Summary.Revision.Anchor == "" {
		t.Fatalf("first revision summary = %#v, want populated anchor metadata", first.Result.Summary.Revision)
	}
	if second.Result.Summary.Revision.ID == "" || second.Result.Summary.Revision.Anchor == "" {
		t.Fatalf("second revision summary = %#v, want populated anchor metadata", second.Result.Summary.Revision)
	}
	if first.Result.Summary.Revision.Anchor == second.Result.Summary.Revision.Anchor {
		t.Fatalf("revision anchors should differ across updated graph states, both were %q", second.Result.Summary.Revision.Anchor)
	}
	if second.Result.Summary.Revision.Reason != "Refresh stale story summary" {
		t.Fatalf("revision reason = %q, want Refresh stale story summary", second.Result.Summary.Revision.Reason)
	}

	state := readGraphState(t, repoDir)
	node := state.Nodes["story:TH1.E2.US3"]
	if node.Title != "Support graph updates and revision anchors" {
		t.Fatalf("node title = %q, want updated title", node.Title)
	}
	if node.UpdatedSessionID != "sess-43" {
		t.Fatalf("updated session id = %q, want sess-43", node.UpdatedSessionID)
	}

	anchors := readRevisionAnchors(t, repoDir)
	if len(anchors) != 2 {
		t.Fatalf("revision anchors = %d, want 2", len(anchors))
	}

	latest := requireRevisionAnchor(t, anchors, second.Result.Summary.Revision.ID)
	if latest.ID != second.Result.Summary.Revision.ID {
		t.Fatalf("latest revision id = %q, want %q", latest.ID, second.Result.Summary.Revision.ID)
	}
	if latest.Summary != "Refresh stale story summary" {
		t.Fatalf("latest revision summary = %q, want Refresh stale story summary", latest.Summary)
	}
	if latest.CreatedBy != "developer" || latest.CreatedSessionID != "sess-43" {
		t.Fatalf("latest revision provenance = %#v, want created_by=developer created_session_id=sess-43", latest)
	}
	if got := stringProp(t, latest.Props, "anchor"); got != second.Result.Summary.Revision.Anchor {
		t.Fatalf("revision anchor prop = %q, want %q", got, second.Result.Summary.Revision.Anchor)
	}
	if got := stringProp(t, latest.Props, "reason"); got != "Refresh stale story summary" {
		t.Fatalf("revision reason prop = %q, want Refresh stale story summary", got)
	}
	if got := latest.Props["workflow"]; got != "cleanup" {
		t.Fatalf("revision workflow prop = %#v, want cleanup", got)
	}
	if got := latest.Props["source"]; got != "agent-review" {
		t.Fatalf("revision source prop = %#v, want agent-review", got)
	}
	if _, ok := latest.Props["snapshot"]; ok {
		t.Fatalf("revision props unexpectedly contain full snapshot: %#v", latest.Props["snapshot"])
	}
	if _, ok := latest.Props["ordinal"]; ok {
		t.Fatalf("revision props unexpectedly contain ordinal: %#v", latest.Props["ordinal"])
	}
}

func TestWriteCommandStoresSupersedingRelationshipAndRevisionAnchor(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkspace(t)

	response := runWriteCommand(t, repoDir, manager, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-42",
    "timestamp": "2026-03-21T14:00:00Z",
    "revision": {
      "reason": "Supersede stale ADR summary",
      "properties": {
        "workflow": "supersede"
      }
    }
  },
  "nodes": [
    {
      "id": "adr:old",
      "kind": "ADR",
      "title": "Old ADR",
      "summary": "Outdated guidance"
    },
    {
      "id": "adr:new",
      "kind": "ADR",
      "title": "New ADR",
      "summary": "Updated guidance"
    }
  ],
  "edges": [
    {
      "from": "adr:new",
      "to": "adr:old",
      "kind": "SUPERSEDES",
      "properties": {
        "confidence": "high"
      }
    }
  ]
}`)

	if response.Result.Summary.Edges.CreatedCount != 1 {
		t.Fatalf("edge summary = %#v, want 1 created edge", response.Result.Summary.Edges)
	}
	if response.Result.Summary.Revision.ID == "" || response.Result.Summary.Revision.Anchor == "" {
		t.Fatalf("revision summary = %#v, want populated anchor metadata", response.Result.Summary.Revision)
	}

	state := readGraphState(t, repoDir)
	edge := state.Edges["adr:new\x00SUPERSEDES\x00adr:old"]
	if edge.Kind != "SUPERSEDES" {
		t.Fatalf("relationship kind = %q, want SUPERSEDES", edge.Kind)
	}
	if got := stringProp(t, edge.Props, "confidence"); got != "high" {
		t.Fatalf("relationship confidence = %q, want high", got)
	}

	anchors := readRevisionAnchors(t, repoDir)
	if len(anchors) != 1 {
		t.Fatalf("revision anchors = %d, want 1", len(anchors))
	}
	anchor := requireRevisionAnchor(t, anchors, response.Result.Summary.Revision.ID)
	if got := stringProp(t, anchor.Props, "anchor"); got != response.Result.Summary.Revision.Anchor {
		t.Fatalf("revision anchor prop = %q, want %q", got, response.Result.Summary.Revision.Anchor)
	}
	if got := anchor.Props["workflow"]; got != "supersede" {
		t.Fatalf("revision workflow prop = %#v, want supersede", got)
	}
}

func TestWriteCommandRejectsUnresolvedRelationshipEndpoints(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkspace(t)

	cmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader(`{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-42",
    "timestamp": "2026-03-21T14:00:00Z"
  },
  "nodes": [
    {
      "id": "story:TH1.E2.US1",
      "kind": "UserStory",
      "title": "Persist entities and relationships"
    }
  ],
  "edges": [
    {
      "from": "story:TH1.E2.US1",
      "to": "project:vp1",
      "kind": "PART_OF"
    }
  ]
}`))

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	response := decodeErrorResponse(t, stdout.Bytes())
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Error.Type != "persistence_error" {
		t.Fatalf("error type = %q, want persistence_error", response.Error.Type)
	}
	if response.Error.Code != "unresolved_relationship_endpoint" {
		t.Fatalf("error code = %q, want unresolved_relationship_endpoint", response.Error.Code)
	}

	missingEndpoints := response.Error.Details["missing_endpoints"]
	if missingEndpoints == nil {
		t.Fatalf("details = %#v, want missing_endpoints", response.Error.Details)
	}

	statePath := filepath.Join(repoDir, repo.WorkspaceDirName, "kuzu", kuzu.StoreFileName)
	if _, statErr := os.Stat(statePath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("graph state file error = %v, want not exist after failed write", statErr)
	}
}

func TestWriteCommandPersistsReasoningUnitAndSessionWithProvenance(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkspace(t)

	response := runWriteCommand(t, repoDir, manager, fmt.Sprintf(`{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-42",
    "timestamp": "2026-03-21T14:00:00Z"
  },
  "nodes": [
    {
      "id": "sess-42",
      "kind": "AgentSession",
      "title": "TH1.E2.US2 implementation session",
      "summary": "Captured reasoning provenance for persistence work",
      "properties": {
        "agent_id": "developer",
        "started_at": "2026-03-21T13:45:00Z",
        "ended_at": "2026-03-21T14:00:00Z",
        "repo_root": %q
      }
    },
    {
      "id": "ru:session-provenance",
      "kind": "ReasoningUnit",
      "title": "Validate reasoning/session persistence",
      "summary": "Store record-level provenance alongside entity-centric graph data",
      "properties": {
        "task": "Implement TH1.E2.US2",
        "agent_id": "developer",
        "session_id": "sess-42",
        "timestamp": "2026-03-21T14:00:00Z"
      }
    }
  ],
  "edges": [
    {
      "from": "ru:session-provenance",
      "to": "sess-42",
      "kind": "GENERATED_IN"
    }
  ]
}`, repoDir))

	if response.Result.Summary.Nodes.CreatedCount != 2 || response.Result.Summary.Nodes.UpdatedCount != 0 {
		t.Fatalf("node summary = %#v, want 2 created and 0 updated", response.Result.Summary.Nodes)
	}
	if response.Result.Summary.Edges.CreatedCount != 1 || response.Result.Summary.Edges.UpdatedCount != 0 {
		t.Fatalf("edge summary = %#v, want 1 created and 0 updated", response.Result.Summary.Edges)
	}

	state := readGraphState(t, repoDir)

	session := state.Nodes["sess-42"]
	if session.Kind != "AgentSession" {
		t.Fatalf("session kind = %q, want AgentSession", session.Kind)
	}
	if got := stringProp(t, session.Props, "agent_id"); got != "developer" {
		t.Fatalf("session agent_id = %q, want developer", got)
	}
	if got := stringProp(t, session.Props, "started_at"); got != "2026-03-21T13:45:00Z" {
		t.Fatalf("session started_at = %q, want 2026-03-21T13:45:00Z", got)
	}
	if got := stringProp(t, session.Props, "ended_at"); got != "2026-03-21T14:00:00Z" {
		t.Fatalf("session ended_at = %q, want 2026-03-21T14:00:00Z", got)
	}
	if got := stringProp(t, session.Props, "repo_root"); got != repoDir {
		t.Fatalf("session repo_root = %q, want %q", got, repoDir)
	}
	if session.CreatedBy != "developer" || session.CreatedSessionID != "sess-42" || session.CreatedAt != "2026-03-21T14:00:00Z" {
		t.Fatalf("session provenance = %#v, want created_by=developer created_session_id=sess-42 created_at=2026-03-21T14:00:00Z", session)
	}

	reasoning := state.Nodes["ru:session-provenance"]
	if reasoning.Kind != "ReasoningUnit" {
		t.Fatalf("reasoning kind = %q, want ReasoningUnit", reasoning.Kind)
	}
	if got := stringProp(t, reasoning.Props, "task"); got != "Implement TH1.E2.US2" {
		t.Fatalf("reasoning task = %q, want Implement TH1.E2.US2", got)
	}
	if got := stringProp(t, reasoning.Props, "agent_id"); got != "developer" {
		t.Fatalf("reasoning agent_id = %q, want developer", got)
	}
	if got := stringProp(t, reasoning.Props, "session_id"); got != "sess-42" {
		t.Fatalf("reasoning session_id = %q, want sess-42", got)
	}
	if got := stringProp(t, reasoning.Props, "timestamp"); got != "2026-03-21T14:00:00Z" {
		t.Fatalf("reasoning timestamp = %q, want 2026-03-21T14:00:00Z", got)
	}
	if reasoning.CreatedBy != "developer" || reasoning.CreatedSessionID != "sess-42" || reasoning.CreatedAt != "2026-03-21T14:00:00Z" {
		t.Fatalf("reasoning provenance = %#v, want created_by=developer created_session_id=sess-42 created_at=2026-03-21T14:00:00Z", reasoning)
	}

	edge := state.Edges["ru:session-provenance\x00GENERATED_IN\x00sess-42"]
	if edge.Kind != "GENERATED_IN" {
		t.Fatalf("relationship kind = %q, want GENERATED_IN", edge.Kind)
	}
	if edge.CreatedBy != "developer" || edge.CreatedSessionID != "sess-42" || edge.CreatedAt != "2026-03-21T14:00:00Z" {
		t.Fatalf("relationship provenance = %#v, want created_by=developer created_session_id=sess-42 created_at=2026-03-21T14:00:00Z", edge)
	}
}

func TestWriteCommandPreservesTypedReasoningLinkToExistingProjectArtefact(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkspace(t)

	runWriteCommand(t, repoDir, manager, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-41",
    "timestamp": "2026-03-21T13:30:00Z"
  },
  "nodes": [
    {
      "id": "adr:ADR-004",
      "kind": "ADR",
      "title": "Entity-centric provenance model",
      "summary": "Keep a compact schema for graph persistence"
    }
  ],
  "edges": []
}`)

	response := runWriteCommand(t, repoDir, manager, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-42",
    "timestamp": "2026-03-21T14:00:00Z"
  },
  "nodes": [
    {
      "id": "ru:link-adr",
      "kind": "ReasoningUnit",
      "title": "Connect reasoning to ADR",
      "summary": "Preserve a typed provenance link to an existing project artefact",
      "properties": {
        "task": "Implement TH1.E2.US2",
        "agent_id": "developer",
        "session_id": "sess-42",
        "timestamp": "2026-03-21T14:00:00Z"
      }
    }
  ],
  "edges": [
    {
      "from": "ru:link-adr",
      "to": "adr:ADR-004",
      "kind": "ABOUT",
      "properties": {
        "evidence": "ADR-004"
      }
    }
  ]
}`)

	if response.Result.Summary.Nodes.CreatedCount != 1 || response.Result.Summary.Nodes.UpdatedCount != 0 {
		t.Fatalf("node summary = %#v, want 1 created and 0 updated", response.Result.Summary.Nodes)
	}
	if response.Result.Summary.Edges.CreatedCount != 1 || response.Result.Summary.Edges.UpdatedCount != 0 {
		t.Fatalf("edge summary = %#v, want 1 created and 0 updated", response.Result.Summary.Edges)
	}

	state := readGraphState(t, repoDir)
	if got := state.Nodes["adr:ADR-004"].Kind; got != "ADR" {
		t.Fatalf("artefact kind = %q, want ADR", got)
	}

	edge := state.Edges["ru:link-adr\x00ABOUT\x00adr:ADR-004"]
	if edge.Kind != "ABOUT" {
		t.Fatalf("relationship kind = %q, want ABOUT", edge.Kind)
	}
	if got := stringProp(t, edge.Props, "evidence"); got != "ADR-004" {
		t.Fatalf("relationship evidence = %q, want ADR-004", got)
	}
}

func TestWriteCommandRejectsReasoningUnitWithoutSessionProvenance(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkspace(t)

	cmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader(`{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-42",
    "timestamp": "2026-03-21T14:00:00Z"
  },
  "nodes": [
    {
      "id": "ru:missing-session",
      "kind": "ReasoningUnit",
      "title": "Missing session provenance",
      "properties": {
        "task": "Implement TH1.E2.US2",
        "agent_id": "developer",
        "timestamp": "2026-03-21T14:00:00Z"
      }
    }
  ],
  "edges": []
}`))

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	response := decodeErrorResponse(t, stdout.Bytes())
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Error.Type != "persistence_error" {
		t.Fatalf("error type = %q, want persistence_error", response.Error.Type)
	}
	if response.Error.Code != "incomplete_provenance" {
		t.Fatalf("error code = %q, want incomplete_provenance", response.Error.Code)
	}
	if got := response.Error.Details["kind"]; got != "ReasoningUnit" {
		t.Fatalf("kind detail = %#v, want ReasoningUnit", got)
	}

	missingFields := asStrings(t, response.Error.Details["missing_fields"])
	if strings.Join(missingFields, ",") != "session_id" {
		t.Fatalf("missing fields = %#v, want [session_id]", missingFields)
	}

	statePath := filepath.Join(repoDir, repo.WorkspaceDirName, "kuzu", kuzu.StoreFileName)
	if _, statErr := os.Stat(statePath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("graph state file error = %v, want not exist after failed write", statErr)
	}
}

func TestWriteCommandRejectsAgentSessionWithIncompleteProvenance(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkspace(t)

	cmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader(fmt.Sprintf(`{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-42",
    "timestamp": "2026-03-21T14:00:00Z"
  },
  "nodes": [
    {
      "id": "sess-42",
      "kind": "AgentSession",
      "title": "Incomplete session record",
      "properties": {
        "agent_id": "developer",
        "ended_at": "2026-03-21T14:00:00Z",
        "repo_root": %q
      }
    }
  ],
  "edges": []
}`, repoDir)))

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	response := decodeErrorResponse(t, stdout.Bytes())
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Error.Type != "persistence_error" {
		t.Fatalf("error type = %q, want persistence_error", response.Error.Type)
	}
	if response.Error.Code != "incomplete_provenance" {
		t.Fatalf("error code = %q, want incomplete_provenance", response.Error.Code)
	}
	if got := response.Error.Details["kind"]; got != "AgentSession" {
		t.Fatalf("kind detail = %#v, want AgentSession", got)
	}

	missingFields := asStrings(t, response.Error.Details["missing_fields"])
	if strings.Join(missingFields, ",") != "started_at" {
		t.Fatalf("missing fields = %#v, want [started_at]", missingFields)
	}
}

func TestWriteCommandReturnsStructuredErrorWhenRevisionAnchorCannotBeRecorded(t *testing.T) {
	t.Parallel()

	repoDir, manager := initWorkspace(t)

	cmd := newCommand(repoDir, manager, failingRevisionPersister{})
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader(`{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-42",
    "timestamp": "2026-03-21T14:00:00Z",
    "revision": {
      "reason": "Refresh stale story summary"
    }
  },
  "nodes": [
    {
      "id": "story:TH1.E2.US3",
      "kind": "UserStory",
      "title": "Support graph updates and revision anchors"
    }
  ],
  "edges": []
}`))

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	response := decodeErrorResponse(t, stdout.Bytes())
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Error.Type != "persistence_error" {
		t.Fatalf("error type = %q, want persistence_error", response.Error.Type)
	}
	if response.Error.Code != "revision_anchor_unavailable" {
		t.Fatalf("error code = %q, want revision_anchor_unavailable", response.Error.Code)
	}
	if got := response.Error.Details["reason"]; got != "revision snapshot encoding failed" {
		t.Fatalf("reason detail = %#v, want revision snapshot encoding failed", got)
	}
}

type spyPersister struct {
	calls        int
	lastEnvelope graphpayload.Envelope
}

func (s *spyPersister) Write(_ *cobra.Command, _ repo.Workspace, envelope graphpayload.Envelope) (kuzu.WriteSummary, error) {
	s.calls++
	s.lastEnvelope = envelope
	return kuzu.WriteSummary{}, nil
}

type failingRevisionPersister struct{}

func (failingRevisionPersister) Write(_ *cobra.Command, _ repo.Workspace, _ graphpayload.Envelope) (kuzu.WriteSummary, error) {
	return kuzu.WriteSummary{}, &kuzu.PersistenceError{
		Code:    "revision_anchor_unavailable",
		Message: "graph write could not record revision metadata",
		Details: map[string]any{
			"reason": "revision snapshot encoding failed",
		},
	}
}

type structuredErrorResponse struct {
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

type structuredSuccessResponse struct {
	SchemaVersion string `json:"schema_version"`
	Command       string `json:"command"`
	Status        string `json:"status"`
	Result        struct {
		Summary kuzu.WriteSummary `json:"summary"`
	} `json:"result"`
}

const validPayloadJSON = `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-42",
    "timestamp": "2026-03-21T14:00:00Z"
  },
  "nodes": [],
  "edges": []
}`

func initWorkspace(t *testing.T) (string, *repo.Manager) {
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

	return repoDir, manager
}

func decodeErrorResponse(t *testing.T, payload []byte) structuredErrorResponse {
	t.Helper()

	response := structuredErrorResponse{}
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal error response: %v\npayload: %s", err, payload)
	}

	return response
}

func decodeSuccessResponse(t *testing.T, payload []byte) structuredSuccessResponse {
	t.Helper()

	response := structuredSuccessResponse{}
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal success response: %v\npayload: %s", err, payload)
	}

	return response
}

func runWriteCommand(t *testing.T, repoDir string, manager *repo.Manager, payload string) structuredSuccessResponse {
	t.Helper()

	cmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader(payload))

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	return decodeSuccessResponse(t, stdout.Bytes())
}

type persistedGraphState struct {
	Nodes map[string]persistedNode     `json:"nodes"`
	Edges map[string]persistedRelation `json:"edges"`
}

type persistedRevisionAnchor struct {
	ID               string         `json:"id"`
	Title            string         `json:"title"`
	Summary          string         `json:"summary"`
	CreatedAt        string         `json:"created_at"`
	CreatedBy        string         `json:"created_by"`
	CreatedSessionID string         `json:"created_session_id"`
	Props            map[string]any `json:"props_json"`
}

type persistedNode struct {
	ID               string         `json:"id"`
	Kind             string         `json:"kind"`
	Title            string         `json:"title"`
	Summary          string         `json:"summary"`
	CreatedAt        string         `json:"created_at"`
	UpdatedAt        string         `json:"updated_at"`
	CreatedBy        string         `json:"created_by"`
	UpdatedBy        string         `json:"updated_by"`
	CreatedSessionID string         `json:"created_session_id"`
	UpdatedSessionID string         `json:"updated_session_id"`
	Props            map[string]any `json:"props_json"`
}

type persistedRelation struct {
	From             string         `json:"from"`
	To               string         `json:"to"`
	Kind             string         `json:"kind"`
	CreatedAt        string         `json:"created_at"`
	UpdatedAt        string         `json:"updated_at"`
	CreatedBy        string         `json:"created_by"`
	UpdatedBy        string         `json:"updated_by"`
	CreatedSessionID string         `json:"created_session_id"`
	UpdatedSessionID string         `json:"updated_session_id"`
	Props            map[string]any `json:"props_json"`
}

func readGraphState(t *testing.T, repoDir string) persistedGraphState {
	t.Helper()

	dbPath := filepath.Join(repoDir, repo.WorkspaceDirName, "kuzu", kuzu.StoreFileName)
	config := kuzudb.DefaultSystemConfig()
	config.ReadOnly = true

	db, err := kuzudb.OpenDatabase(dbPath, config)
	if err != nil {
		t.Fatalf("open kuzu database: %v", err)
	}
	defer db.Close()

	conn, err := kuzudb.OpenConnection(db)
	if err != nil {
		t.Fatalf("open kuzu connection: %v", err)
	}
	defer conn.Close()

	state := persistedGraphState{
		Nodes: map[string]persistedNode{},
		Edges: map[string]persistedRelation{},
	}

	nodeResult, err := conn.Query(`MATCH (e:Entity)
WHERE e.kind <> 'GraphRevision'
RETURN e.id, e.kind, e.title, e.summary, e.created_at, e.updated_at, e.created_by, e.updated_by, e.created_session_id, e.updated_session_id, e.props_json
ORDER BY e.id;`)
	if err != nil {
		t.Fatalf("query persisted nodes: %v", err)
	}
	defer nodeResult.Close()

	for nodeResult.HasNext() {
		tuple, err := nodeResult.Next()
		if err != nil {
			t.Fatalf("read persisted node tuple: %v", err)
		}
		values, err := tuple.GetAsSlice()
		if err != nil {
			t.Fatalf("decode persisted node tuple: %v", err)
		}

		node := persistedNode{
			ID:               stringValue(values[0]),
			Kind:             stringValue(values[1]),
			Title:            stringValue(values[2]),
			Summary:          stringValue(values[3]),
			CreatedAt:        stringValue(values[4]),
			UpdatedAt:        stringValue(values[5]),
			CreatedBy:        stringValue(values[6]),
			UpdatedBy:        stringValue(values[7]),
			CreatedSessionID: stringValue(values[8]),
			UpdatedSessionID: stringValue(values[9]),
			Props:            parseJSONMap(t, stringValue(values[10])),
		}
		state.Nodes[node.ID] = node
	}

	edgeResult, err := conn.Query(`MATCH (from:Entity)-[r:EntityRelation]->(to:Entity)
WHERE from.kind <> 'GraphRevision' AND to.kind <> 'GraphRevision'
RETURN from.id, to.id, r.kind, r.created_at, r.updated_at, r.created_by, r.updated_by, r.created_session_id, r.updated_session_id, r.props_json
ORDER BY from.id, r.kind, to.id;`)
	if err != nil {
		t.Fatalf("query persisted relationships: %v", err)
	}
	defer edgeResult.Close()

	for edgeResult.HasNext() {
		tuple, err := edgeResult.Next()
		if err != nil {
			t.Fatalf("read persisted relationship tuple: %v", err)
		}
		values, err := tuple.GetAsSlice()
		if err != nil {
			t.Fatalf("decode persisted relationship tuple: %v", err)
		}
		relation := persistedRelation{
			From:             stringValue(values[0]),
			To:               stringValue(values[1]),
			Kind:             stringValue(values[2]),
			CreatedAt:        stringValue(values[3]),
			UpdatedAt:        stringValue(values[4]),
			CreatedBy:        stringValue(values[5]),
			UpdatedBy:        stringValue(values[6]),
			CreatedSessionID: stringValue(values[7]),
			UpdatedSessionID: stringValue(values[8]),
			Props:            parseJSONMap(t, stringValue(values[9])),
		}
		state.Edges[relation.From+"\x00"+relation.Kind+"\x00"+relation.To] = relation
	}

	return state
}

func readRevisionAnchors(t *testing.T, repoDir string) []persistedRevisionAnchor {
	t.Helper()

	dbPath := filepath.Join(repoDir, repo.WorkspaceDirName, "kuzu", kuzu.StoreFileName)
	config := kuzudb.DefaultSystemConfig()
	config.ReadOnly = true

	db, err := kuzudb.OpenDatabase(dbPath, config)
	if err != nil {
		t.Fatalf("open kuzu database: %v", err)
	}
	defer db.Close()

	conn, err := kuzudb.OpenConnection(db)
	if err != nil {
		t.Fatalf("open kuzu connection: %v", err)
	}
	defer conn.Close()

	result, err := conn.Query(`MATCH (e:Entity)
WHERE e.kind = 'GraphRevision'
RETURN e.id, e.title, e.summary, e.created_at, e.created_by, e.created_session_id, e.props_json
ORDER BY e.created_at, e.id;`)
	if err != nil {
		t.Fatalf("query revision anchors: %v", err)
	}
	defer result.Close()

	anchors := []persistedRevisionAnchor{}
	for result.HasNext() {
		tuple, err := result.Next()
		if err != nil {
			t.Fatalf("read revision anchor tuple: %v", err)
		}
		values, err := tuple.GetAsSlice()
		if err != nil {
			t.Fatalf("decode revision anchor tuple: %v", err)
		}

		anchors = append(anchors, persistedRevisionAnchor{
			ID:               stringValue(values[0]),
			Title:            stringValue(values[1]),
			Summary:          stringValue(values[2]),
			CreatedAt:        stringValue(values[3]),
			CreatedBy:        stringValue(values[4]),
			CreatedSessionID: stringValue(values[5]),
			Props:            parseJSONMap(t, stringValue(values[6])),
		})
	}

	return anchors
}

func requireRevisionAnchor(t *testing.T, anchors []persistedRevisionAnchor, id string) persistedRevisionAnchor {
	t.Helper()

	for _, anchor := range anchors {
		if anchor.ID == id {
			return anchor
		}
	}

	t.Fatalf("revision anchor %q not found in %#v", id, anchors)
	return persistedRevisionAnchor{}
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	str, _ := value.(string)
	return str
}

func parseJSONMap(t *testing.T, payload string) map[string]any {
	t.Helper()

	if strings.TrimSpace(payload) == "" {
		return nil
	}

	value := map[string]any{}
	if err := json.Unmarshal([]byte(payload), &value); err != nil {
		t.Fatalf("json.Unmarshal props_json: %v\npayload: %s", err, payload)
	}
	return value
}

func asStrings(t *testing.T, value any) []string {
	t.Helper()

	values, ok := value.([]any)
	if !ok {
		t.Fatalf("value type = %T, want []any", value)
	}

	result := make([]string, 0, len(values))
	for _, item := range values {
		str, ok := item.(string)
		if !ok {
			t.Fatalf("item type = %T, want string", item)
		}
		result = append(result, str)
	}

	return result
}

func stringProp(t *testing.T, props map[string]any, key string) string {
	t.Helper()

	value, ok := props[key]
	if !ok {
		t.Fatalf("props[%q] missing from %#v", key, props)
	}

	str, ok := value.(string)
	if !ok {
		t.Fatalf("props[%q] type = %T, want string", key, value)
	}

	return str
}
