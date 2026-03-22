package statscmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	kuzudb "github.com/kuzudb/go-kuzu"
	"github.com/spf13/cobra"

	graphpayload "github.com/guillaume-galp/cge/internal/domain/payload"
	"github.com/guillaume-galp/cge/internal/infra/kuzu"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func TestStatsCommandReturnsSnapshotCountsForInitializedGraphWithoutMutatingState(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initStatsWorkspace(t)
	writeStatsRevision(t, workspace, `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-90",
    "timestamp": "2026-03-21T23:00:00Z",
    "revision": {
      "reason": "Seed graph stats test"
    }
  },
  "nodes": [
    {
      "id": "story:TH2.E1.US1",
      "kind": "UserStory",
      "title": "Add graph stats command"
    },
    {
      "id": "epic:E1",
      "kind": "Epic",
      "title": "Graph stats snapshot"
    }
  ],
  "edges": [
    {
      "from": "story:TH2.E1.US1",
      "to": "epic:E1",
      "kind": "PART_OF"
    }
  ]
}`)

	store := kuzu.NewStore()
	graphBefore, err := store.ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph before stats returned error: %v", err)
	}
	revisionCountBefore := queryRevisionCount(t, repoDir)

	cmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeStatsSuccessResponse(t, stdout.Bytes())
	if response.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want v1", response.SchemaVersion)
	}
	if response.Command != "stats" {
		t.Fatalf("command = %q, want stats", response.Command)
	}
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if response.Result.Snapshot.Nodes != 2 {
		t.Fatalf("snapshot.nodes = %d, want 2", response.Result.Snapshot.Nodes)
	}
	if response.Result.Snapshot.Relationships != 1 {
		t.Fatalf("snapshot.relationships = %d, want 1", response.Result.Snapshot.Relationships)
	}

	graphAfter, err := store.ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph after stats returned error: %v", err)
	}
	if !reflect.DeepEqual(graphAfter, graphBefore) {
		t.Fatalf("graph content changed after stats\nbefore: %#v\nafter: %#v", graphBefore, graphAfter)
	}

	revisionCountAfter := queryRevisionCount(t, repoDir)
	if revisionCountAfter != revisionCountBefore {
		t.Fatalf("revision count after stats = %d, want %d", revisionCountAfter, revisionCountBefore)
	}
}

func TestStatsCommandReturnsZeroCountsForInitializedButEmptyGraph(t *testing.T) {
	t.Parallel()

	repoDir, manager, _ := initStatsWorkspace(t)

	cmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeStatsSuccessResponse(t, stdout.Bytes())
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if response.Result.Snapshot.Nodes != 0 {
		t.Fatalf("snapshot.nodes = %d, want 0", response.Result.Snapshot.Nodes)
	}
	if response.Result.Snapshot.Relationships != 0 {
		t.Fatalf("snapshot.relationships = %d, want 0", response.Result.Snapshot.Relationships)
	}
}

func TestStatsCommandReturnsStructuredErrorForMissingWorkspace(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, output)
	}

	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	statsCmd := newCommand(repoDir, manager, &mockStatsReader{})
	stdout := &bytes.Buffer{}
	statsCmd.SetOut(stdout)
	statsCmd.SetErr(&bytes.Buffer{})

	err := statsCmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	response := decodeStatsErrorResponse(t, stdout.Bytes())
	if response.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want v1", response.SchemaVersion)
	}
	if response.Command != "stats" {
		t.Fatalf("command = %q, want stats", response.Command)
	}
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Error.Category != "operational_error" {
		t.Fatalf("error category = %q, want operational_error", response.Error.Category)
	}
	if response.Error.Type != "workspace_error" {
		t.Fatalf("error type = %q, want workspace_error", response.Error.Type)
	}
	if response.Error.Code != "workspace_not_initialized" {
		t.Fatalf("error code = %q, want workspace_not_initialized", response.Error.Code)
	}
}

func TestStatsCommandReturnsStructuredOperationalErrorForReaderFailure(t *testing.T) {
	t.Parallel()

	repoDir, manager, _ := initStatsWorkspace(t)
	reader := &mockStatsReader{
		err: &kuzu.PersistenceError{
			Code:    "stats_unavailable",
			Message: "graph stats could not be read",
			Details: map[string]any{
				"reason": "test failure",
			},
		},
	}

	cmd := newCommand(repoDir, manager, reader)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if reader.calls != 1 {
		t.Fatalf("reader calls = %d, want 1", reader.calls)
	}

	response := decodeStatsErrorResponse(t, stdout.Bytes())
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Error.Category != "operational_error" {
		t.Fatalf("error category = %q, want operational_error", response.Error.Category)
	}
	if response.Error.Type != "persistence_error" {
		t.Fatalf("error type = %q, want persistence_error", response.Error.Type)
	}
	if response.Error.Code != "stats_unavailable" {
		t.Fatalf("error code = %q, want stats_unavailable", response.Error.Code)
	}
	if response.Error.Details["reason"] != "test failure" {
		t.Fatalf("error details reason = %#v, want test failure", response.Error.Details["reason"])
	}
}

func TestStatsCommandWritesValidJSONToFile(t *testing.T) {
	t.Parallel()

	repoDir, manager, _ := initStatsWorkspace(t)
	reader := &mockStatsReader{
		stats: kuzu.GraphStats{
			Nodes:         7,
			Relationships: 11,
		},
	}

	stdoutCmd := newCommand(repoDir, manager, reader)
	stdout := &bytes.Buffer{}
	stdoutCmd.SetOut(stdout)
	stdoutCmd.SetErr(&bytes.Buffer{})
	if err := stdoutCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("stdout ExecuteContext returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "stats.json")
	fileCmd := newCommand(repoDir, manager, reader)
	fileStdout := &bytes.Buffer{}
	fileCmd.SetOut(fileStdout)
	fileCmd.SetErr(&bytes.Buffer{})
	fileCmd.SetArgs([]string{"--output", outputPath})
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

	response := decodeStatsSuccessResponse(t, filePayload)
	if response.Result.Snapshot.Nodes != 7 || response.Result.Snapshot.Relationships != 11 {
		t.Fatalf("file response snapshot = %#v, want nodes=7 relationships=11", response.Result.Snapshot)
	}
}

type mockStatsReader struct {
	stats         kuzu.GraphStats
	err           error
	calls         int
	lastWorkspace repo.Workspace
}

func (r *mockStatsReader) Stats(_ *cobra.Command, workspace repo.Workspace) (kuzu.GraphStats, error) {
	r.calls++
	r.lastWorkspace = workspace
	if r.err != nil {
		return kuzu.GraphStats{}, r.err
	}
	return r.stats, nil
}

func initStatsWorkspace(t *testing.T) (string, *repo.Manager, repo.Workspace) {
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

func writeStatsRevision(t *testing.T, workspace repo.Workspace, payload string) kuzu.WriteSummary {
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

func queryRevisionCount(t *testing.T, repoDir string) int {
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

	result, err := conn.Query(`MATCH (e:Entity) WHERE e.kind = 'GraphRevision' RETURN count(e);`)
	if err != nil {
		t.Fatalf("query revision count: %v", err)
	}
	defer result.Close()

	if !result.HasNext() {
		return 0
	}

	tuple, err := result.Next()
	if err != nil {
		t.Fatalf("read revision count tuple: %v", err)
	}
	values, err := tuple.GetAsSlice()
	if err != nil {
		t.Fatalf("decode revision count tuple: %v", err)
	}

	return countValue(values[0])
}

type statsSuccessResponse struct {
	SchemaVersion string `json:"schema_version"`
	Command       string `json:"command"`
	Status        string `json:"status"`
	Result        struct {
		Snapshot kuzu.GraphStats `json:"snapshot"`
	} `json:"result"`
}

type statsErrorResponse struct {
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

func decodeStatsSuccessResponse(t *testing.T, payload []byte) statsSuccessResponse {
	t.Helper()

	var response statsSuccessResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal stats success response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeStatsErrorResponse(t *testing.T, payload []byte) statsErrorResponse {
	t.Helper()

	var response statsErrorResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal stats error response: %v\npayload: %s", err, payload)
	}
	return response
}

func countValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int8:
		return int(typed)
	case int16:
		return int(typed)
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case uint:
		return int(typed)
	case uint8:
		return int(typed)
	case uint16:
		return int(typed)
	case uint32:
		return int(typed)
	case uint64:
		return int(typed)
	case float32:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}
