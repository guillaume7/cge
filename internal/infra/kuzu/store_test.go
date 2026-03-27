package kuzu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	kuzudb "github.com/kuzudb/go-kuzu"

	graphpayload "github.com/guillaume-galp/cge/internal/domain/payload"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func TestStoreWriteRollsBackWhenRevisionAnchorCannotBeRecorded(t *testing.T) {
	t.Parallel()

	repoDir, workspace := initStoreWorkspace(t)
	store := &Store{
		recordRevisionAnchor: func(_ *kuzudb.Connection, _ graphpayload.Metadata, _ WriteSummary) (RevisionWriteSummary, error) {
			return RevisionWriteSummary{}, &PersistenceError{
				Code:    "revision_anchor_unavailable",
				Message: "graph write could not record revision metadata",
				Details: map[string]any{
					"reason": "forced revision anchor failure",
				},
			}
		},
	}

	_, err := store.Write(context.Background(), workspace, graphpayload.Envelope{
		SchemaVersion: graphpayload.SchemaVersionV1,
		Metadata: graphpayload.Metadata{
			AgentID:   "developer",
			SessionID: "sess-42",
			Timestamp: "2026-03-21T14:00:00Z",
			Revision: graphpayload.RevisionMetadata{
				Reason: "Refresh stale story summary",
			},
		},
		Nodes: mustRawMessages(t, []string{`{
      "id": "story:TH1.E2.US3",
      "kind": "UserStory",
      "title": "Support graph updates and revision anchors"
    }`}),
		Edges: nil,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var persistenceErr *PersistenceError
	if !errors.As(err, &persistenceErr) {
		t.Fatalf("error type = %T, want *PersistenceError", err)
	}
	if persistenceErr.Code != "revision_anchor_unavailable" {
		t.Fatalf("error code = %q, want revision_anchor_unavailable", persistenceErr.Code)
	}

	dbPath := filepath.Join(repoDir, repo.WorkspaceDirName, "kuzu", StoreFileName)
	db, conn := openTestConnection(t, dbPath)
	defer conn.Close()
	defer db.Close()

	exists, err := entityExists(conn, "story:TH1.E2.US3")
	if err != nil {
		t.Fatalf("entityExists returned error: %v", err)
	}
	if exists {
		t.Fatal("story entity persisted despite revision anchor failure")
	}

	result, err := conn.Query(`MATCH (e:Entity) WHERE e.kind = 'GraphRevision' RETURN COUNT(e);`)
	if err != nil {
		t.Fatalf("query graph revisions: %v", err)
	}
	defer result.Close()

	if !result.HasNext() {
		t.Fatal("expected graph revision count row")
	}
	tuple, err := result.Next()
	if err != nil {
		t.Fatalf("read graph revision count: %v", err)
	}
	values, err := tuple.GetAsSlice()
	if err != nil {
		t.Fatalf("decode graph revision count: %v", err)
	}
	if got := intValue(values[0]); got != 0 {
		t.Fatalf("graph revision count = %d, want 0", got)
	}
}

func TestStoreWriteUsesRewriteSafeRevisionIDsAfterRevisionDeletion(t *testing.T) {
	t.Parallel()

	repoDir, workspace := initStoreWorkspace(t)
	store := NewStore()

	first, err := store.Write(context.Background(), workspace, graphpayload.Envelope{
		SchemaVersion: graphpayload.SchemaVersionV1,
		Metadata: graphpayload.Metadata{
			AgentID:   "developer",
			SessionID: "sess-42",
			Timestamp: "2026-03-21T14:00:00Z",
			Revision: graphpayload.RevisionMetadata{
				Reason: "Seed initial story state",
			},
		},
		Nodes: mustRawMessages(t, []string{`{
      "id": "story:TH1.E2.US3",
      "kind": "UserStory",
      "title": "Support graph updates",
      "summary": "Initial summary"
    }`}),
		Edges: nil,
	})
	if err != nil {
		t.Fatalf("first Write returned error: %v", err)
	}

	dbPath := filepath.Join(repoDir, repo.WorkspaceDirName, "kuzu", StoreFileName)
	db, conn := openWritableTestConnection(t, dbPath)
	if err := executeQuery(conn, fmt.Sprintf(`MATCH (e:%s {id: '%s'}) DELETE e;`, entityTableName, first.Revision.ID)); err != nil {
		conn.Close()
		db.Close()
		t.Fatalf("delete first revision anchor: %v", err)
	}
	conn.Close()
	db.Close()

	second, err := store.Write(context.Background(), workspace, graphpayload.Envelope{
		SchemaVersion: graphpayload.SchemaVersionV1,
		Metadata: graphpayload.Metadata{
			AgentID:   "developer",
			SessionID: "sess-43",
			Timestamp: "2026-03-21T15:00:00Z",
			Revision: graphpayload.RevisionMetadata{
				Reason: "Refresh story state after deleting old revision anchor",
			},
		},
		Nodes: mustRawMessages(t, []string{`{
      "id": "story:TH1.E2.US3",
      "kind": "UserStory",
      "title": "Support graph updates and revision anchors",
      "summary": "Updated summary"
    }`}),
		Edges: nil,
	})
	if err != nil {
		t.Fatalf("second Write returned error: %v", err)
	}

	if second.Revision.ID == first.Revision.ID {
		t.Fatalf("revision ids collided after deleting older revision anchor: %q", second.Revision.ID)
	}
	if second.Revision.Anchor == "" {
		t.Fatal("second revision anchor was empty")
	}

	verifyDB, verifyConn := openTestConnection(t, dbPath)
	defer verifyConn.Close()
	defer verifyDB.Close()

	result, err := executePrepared(verifyConn,
		fmt.Sprintf(`MATCH (e:%s {id: $id}) RETURN e.id, e.props_json;`, entityTableName),
		map[string]any{"id": second.Revision.ID},
	)
	if err != nil {
		t.Fatalf("query latest revision anchor: %v", err)
	}
	defer result.Close()
	if !result.HasNext() {
		t.Fatalf("revision anchor %q not found", second.Revision.ID)
	}
	tuple, err := result.Next()
	if err != nil {
		t.Fatalf("read latest revision anchor: %v", err)
	}
	values, err := tuple.GetAsSlice()
	if err != nil {
		t.Fatalf("decode latest revision anchor: %v", err)
	}
	props := parseJSONMap(t, stringValue(values[1]))
	if got := props["anchor"]; got != second.Revision.Anchor {
		t.Fatalf("persisted anchor = %#v, want %q", got, second.Revision.Anchor)
	}
	if _, ok := props["snapshot"]; ok {
		t.Fatalf("revision anchor props unexpectedly include full snapshot: %#v", props["snapshot"])
	}
	if _, ok := props["ordinal"]; ok {
		t.Fatalf("revision anchor props unexpectedly include ordinal: %#v", props["ordinal"])
	}
}

func TestStoreWriteDeduplicatesComparableSnapshotRelationships(t *testing.T) {
	t.Parallel()

	repoDir, workspace := initStoreWorkspace(t)
	store := NewStore()

	dbPath := filepath.Join(repoDir, repo.WorkspaceDirName, "kuzu", StoreFileName)
	db, conn := openWritableTestConnection(t, dbPath)
	if err := ensureSchema(conn); err != nil {
		conn.Close()
		db.Close()
		t.Fatalf("ensureSchema returned error: %v", err)
	}
	if err := beginTransaction(conn); err != nil {
		conn.Close()
		db.Close()
		t.Fatalf("beginTransaction returned error: %v", err)
	}

	seedMetadata := graphpayload.Metadata{
		AgentID:   "developer",
		SessionID: "seed-duplicates",
		Timestamp: "2026-03-24T12:00:00Z",
	}
	for _, node := range []entityInput{
		{ID: "reasoning:1", Kind: reasoningUnitKind, Title: "Seed duplicate relationship state", Props: map[string]any{
			"agent_id":   seedMetadata.AgentID,
			"session_id": seedMetadata.SessionID,
			"timestamp":  seedMetadata.Timestamp,
		}},
		{ID: "artifact:1", Kind: "Artifact", Title: "artifact one"},
	} {
		if err := createEntity(conn, seedMetadata, node); err != nil {
			_ = rollbackTransaction(conn)
			conn.Close()
			db.Close()
			t.Fatalf("createEntity returned error: %v", err)
		}
	}
	duplicateEdge := edgeInput{From: "reasoning:1", Kind: "ABOUT", To: "artifact:1", Props: map[string]any{"relation": "changed_artifact"}}
	if err := createRelation(conn, seedMetadata, duplicateEdge); err != nil {
		_ = rollbackTransaction(conn)
		conn.Close()
		db.Close()
		t.Fatalf("first createRelation returned error: %v", err)
	}
	if err := createRelation(conn, seedMetadata, duplicateEdge); err != nil {
		_ = rollbackTransaction(conn)
		conn.Close()
		db.Close()
		t.Fatalf("second createRelation returned error: %v", err)
	}
	if err := commitTransaction(conn); err != nil {
		conn.Close()
		db.Close()
		t.Fatalf("commitTransaction returned error: %v", err)
	}
	conn.Close()
	db.Close()

	summary, err := store.Write(context.Background(), workspace, graphpayload.Envelope{
		SchemaVersion: graphpayload.SchemaVersionV1,
		Metadata: graphpayload.Metadata{
			AgentID:   "developer",
			SessionID: "dedupe-snapshot",
			Timestamp: "2026-03-24T12:30:00Z",
			Revision: graphpayload.RevisionMetadata{
				Reason: "Record a revision despite duplicate comparable relationships",
			},
		},
		Nodes: mustRawMessages(t, []string{`{
      "id": "artifact:1",
      "kind": "Artifact",
      "title": "artifact one",
      "summary": "Updated after duplicate relationship seed"
    }`}),
		Edges: nil,
	})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if summary.Revision.ID == "" {
		t.Fatal("revision id was empty")
	}
	if summary.Revision.EdgeCount != 1 {
		t.Fatalf("revision edge count = %d, want 1 unique relationship", summary.Revision.EdgeCount)
	}
}

func TestCurrentRevisionReturnsNewestMatchingSnapshotWhenTimestampsCollide(t *testing.T) {
	t.Parallel()

	repoDir, workspace := initStoreWorkspace(t)
	dbPath := filepath.Join(repoDir, repo.WorkspaceDirName, "kuzu", StoreFileName)
	db, conn := openWritableTestConnection(t, dbPath)
	defer conn.Close()
	defer db.Close()

	if err := ensureSchema(conn); err != nil {
		t.Fatalf("ensureSchema returned error: %v", err)
	}
	if err := beginTransaction(conn); err != nil {
		t.Fatalf("beginTransaction returned error: %v", err)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		_ = rollbackTransaction(conn)
	}()

	metadata := graphpayload.Metadata{
		AgentID:   "developer",
		SessionID: "timestamp-collision",
		Timestamp: "2026-03-24T12:00:00Z",
	}
	if err := createEntity(conn, metadata, entityInput{
		ID:    "artifact:1",
		Kind:  "Artifact",
		Title: "Initial artifact state",
	}); err != nil {
		t.Fatalf("createEntity returned error: %v", err)
	}

	firstSnapshot, err := readComparableSnapshot(conn)
	if err != nil {
		t.Fatalf("readComparableSnapshot for first revision returned error: %v", err)
	}
	firstAnchor, err := comparableSnapshotAnchor(firstSnapshot)
	if err != nil {
		t.Fatalf("comparableSnapshotAnchor for first revision returned error: %v", err)
	}

	firstRevisionID := "rev:ffffffffffffffffffffffffffffffff"
	if err := createEntity(conn, metadata, entityInput{
		ID:      firstRevisionID,
		Kind:    graphRevisionKind,
		Title:   revisionTitlePrefix + firstRevisionID,
		Summary: "Persist the older colliding revision",
		Props:   revisionProps(firstAnchor, graphpayload.RevisionMetadata{Reason: "Persist the older colliding revision"}, WriteSummary{}, len(firstSnapshot.Nodes), len(firstSnapshot.Edges)),
	}); err != nil {
		t.Fatalf("createEntity for first revision returned error: %v", err)
	}
	if err := persistRevisionSnapshot(conn, firstRevisionID, firstSnapshot); err != nil {
		t.Fatalf("persistRevisionSnapshot for first revision returned error: %v", err)
	}

	if err := updateEntity(conn, metadata, entityInput{
		ID:    "artifact:1",
		Kind:  "Artifact",
		Title: "Newest artifact state",
	}); err != nil {
		t.Fatalf("updateEntity returned error: %v", err)
	}

	secondSnapshot, err := readComparableSnapshot(conn)
	if err != nil {
		t.Fatalf("readComparableSnapshot for second revision returned error: %v", err)
	}
	secondAnchor, err := comparableSnapshotAnchor(secondSnapshot)
	if err != nil {
		t.Fatalf("comparableSnapshotAnchor for second revision returned error: %v", err)
	}

	secondRevisionID := "rev:00000000000000000000000000000000"
	if err := createEntity(conn, metadata, entityInput{
		ID:      secondRevisionID,
		Kind:    graphRevisionKind,
		Title:   revisionTitlePrefix + secondRevisionID,
		Summary: "Persist the newer colliding revision",
		Props:   revisionProps(secondAnchor, graphpayload.RevisionMetadata{Reason: "Persist the newer colliding revision"}, WriteSummary{}, len(secondSnapshot.Nodes), len(secondSnapshot.Edges)),
	}); err != nil {
		t.Fatalf("createEntity for second revision returned error: %v", err)
	}
	if err := persistRevisionSnapshot(conn, secondRevisionID, secondSnapshot); err != nil {
		t.Fatalf("persistRevisionSnapshot for second revision returned error: %v", err)
	}

	if err := commitTransaction(conn); err != nil {
		t.Fatalf("commitTransaction returned error: %v", err)
	}
	committed = true

	currentRevision, err := NewStore().CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision returned error: %v", err)
	}
	if !currentRevision.Exists {
		t.Fatal("expected current revision to exist")
	}
	if currentRevision.Revision.ID != secondRevisionID {
		t.Fatalf("current revision id = %q, want %q", currentRevision.Revision.ID, secondRevisionID)
	}
	if currentRevision.Revision.Anchor != secondAnchor {
		t.Fatalf("current revision anchor = %q, want %q", currentRevision.Revision.Anchor, secondAnchor)
	}
}

func initStoreWorkspace(t *testing.T) (string, repo.Workspace) {
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

	return repoDir, workspace
}

func mustRawMessages(t *testing.T, payloads []string) []json.RawMessage {
	t.Helper()

	values := make([]json.RawMessage, 0, len(payloads))
	for _, payload := range payloads {
		values = append(values, json.RawMessage(payload))
	}
	return values
}

func openTestConnection(t *testing.T, dbPath string) (*kuzudb.Database, *kuzudb.Connection) {
	t.Helper()

	config := kuzudb.DefaultSystemConfig()
	config.ReadOnly = true

	db, err := kuzudb.OpenDatabase(dbPath, config)
	if err != nil {
		t.Fatalf("open kuzu database: %v", err)
	}

	conn, err := kuzudb.OpenConnection(db)
	if err != nil {
		db.Close()
		t.Fatalf("open kuzu connection: %v", err)
	}

	return db, conn
}

func openWritableTestConnection(t *testing.T, dbPath string) (*kuzudb.Database, *kuzudb.Connection) {
	t.Helper()

	db, err := kuzudb.OpenDatabase(dbPath, kuzudb.DefaultSystemConfig())
	if err != nil {
		t.Fatalf("open writable kuzu database: %v", err)
	}

	conn, err := kuzudb.OpenConnection(db)
	if err != nil {
		db.Close()
		t.Fatalf("open writable kuzu connection: %v", err)
	}

	return db, conn
}

func parseJSONMap(t *testing.T, payload string) map[string]any {
	t.Helper()

	if payload == "" {
		return nil
	}

	decoded := map[string]any{}
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("json.Unmarshal props_json: %v", err)
	}
	return decoded
}
