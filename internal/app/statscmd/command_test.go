package statscmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/guillaume-galp/cge/internal/app/graphhealth"
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
	revisionBefore, err := store.CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision before stats returned error: %v", err)
	}

	cmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeStatsSuccessResponse(t, stdout.Bytes())
	if response.Command != "stats" || response.Status != "ok" {
		t.Fatalf("response = %#v, want stats ok", response)
	}
	if response.Result.Snapshot.Nodes != 2 || response.Result.Snapshot.Relationships != 1 {
		t.Fatalf("snapshot = %#v, want 2 nodes / 1 relationship", response.Result.Snapshot)
	}
	if response.Result.Indicators.DensityScore == 0 {
		t.Fatal("expected non-zero density score")
	}

	graphAfter, err := store.ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph after stats returned error: %v", err)
	}
	if !graphsEqual(graphBefore, graphAfter) {
		t.Fatalf("graph changed after stats\nbefore: %#v\nafter: %#v", graphBefore, graphAfter)
	}
	revisionAfter, err := store.CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision after stats returned error: %v", err)
	}
	if revisionBefore != revisionAfter {
		t.Fatalf("revision changed after stats\nbefore: %#v\nafter: %#v", revisionBefore, revisionAfter)
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
	if response.Result.Snapshot.Nodes != 0 || response.Result.Snapshot.Relationships != 0 {
		t.Fatalf("snapshot = %#v, want zero counts", response.Result.Snapshot)
	}
	if response.Result.Indicators != (graphhealth.Indicators{}) {
		t.Fatalf("indicators = %#v, want zero values", response.Result.Indicators)
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
	if response.Error.Code != "workspace_not_initialized" {
		t.Fatalf("error code = %q, want workspace_not_initialized", response.Error.Code)
	}
}

func TestStatsCommandReturnsStructuredOperationalErrorForReaderFailure(t *testing.T) {
	t.Parallel()

	repoDir, manager, _ := initStatsWorkspace(t)
	reader := &mockStatsReader{err: &kuzu.PersistenceError{
		Code:    "stats_unavailable",
		Message: "graph stats could not be read",
		Details: map[string]any{"reason": "test failure"},
	}}
	cmd := newCommand(repoDir, manager, reader)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	response := decodeStatsErrorResponse(t, stdout.Bytes())
	if response.Error.Code != "stats_unavailable" {
		t.Fatalf("error code = %q, want stats_unavailable", response.Error.Code)
	}
}

func TestStatsCommandWritesEquivalentStructuredOutputToFile(t *testing.T) {
	t.Parallel()

	repoDir, manager, _ := initStatsWorkspace(t)
	reader := &mockStatsReader{analysis: graphhealth.Analysis{
		Snapshot: kuzu.GraphStats{Nodes: 7, Relationships: 11},
		Indicators: graphhealth.Indicators{
			DuplicationRate:    0.25,
			OrphanRate:         0.5,
			ContradictoryFacts: 2,
			DensityScore:       0.3,
			ClusteringScore:    0.75,
		},
	}}

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

	payload, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(stdout.Bytes(), payload) {
		t.Fatalf("stdout payload != file payload\nstdout: %s\nfile: %s", stdout.Bytes(), payload)
	}
}

type mockStatsReader struct {
	analysis      graphhealth.Analysis
	err           error
	calls         int
	lastWorkspace repo.Workspace
}

func (r *mockStatsReader) Analyze(_ *cobra.Command, workspace repo.Workspace) (graphhealth.Analysis, error) {
	r.calls++
	r.lastWorkspace = workspace
	if r.err != nil {
		return graphhealth.Analysis{}, r.err
	}
	return r.analysis, nil
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

type statsSuccessResponse struct {
	SchemaVersion string `json:"schema_version"`
	Command       string `json:"command"`
	Status        string `json:"status"`
	Result        struct {
		Snapshot   kuzu.GraphStats        `json:"snapshot"`
		Indicators graphhealth.Indicators `json:"indicators"`
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

func graphsEqual(left, right kuzu.Graph) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftPayload, rightPayload)
}

var _ = errors.New
