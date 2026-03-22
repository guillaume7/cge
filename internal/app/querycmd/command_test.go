package querycmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/guillaume-galp/cge/internal/app/retrieval"
	"github.com/guillaume-galp/cge/internal/infra/repo"
	"github.com/guillaume-galp/cge/internal/infra/textindex"
)

func TestQueryCommandReturnsStructuredSuccessJSONFromFlag(t *testing.T) {
	t.Parallel()

	repoDir, manager := initQueryWorkspace(t)
	querier := &spyQuerier{
		result: retrieval.ResultSet{
			IndexStatus: "rebuilt",
			Results: []retrieval.Result{{
				Rank:  1,
				Score: 14.5,
				Scores: retrieval.ScoreBreakdown{
					Text:       2.5,
					Structural: 12,
				},
				Entity: retrieval.Entity{
					ID:      "service:login-api",
					Kind:    "Service",
					Title:   "Login API",
					Summary: "Depends on authentication",
				},
				MatchedTerms: []string{"auth", "authentication"},
				GraphRefs: []retrieval.GraphRef{{
					From:      "service:login-api",
					To:        "component:authentication",
					Kind:      "DEPENDS_ON",
					Direction: "supports_task",
				}},
				Provenance: retrieval.Provenance{
					CreatedBy:        "developer",
					CreatedSessionID: "sess-42",
				},
			}},
		},
	}

	cmd := newCommand(repoDir, manager, querier)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--task", "what depends on auth?"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if querier.calls != 1 {
		t.Fatalf("query calls = %d, want 1", querier.calls)
	}
	if querier.lastTask != "what depends on auth?" {
		t.Fatalf("task = %q, want what depends on auth?", querier.lastTask)
	}

	response := decodeQuerySuccessResponse(t, stdout.Bytes())
	if response.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want v1", response.SchemaVersion)
	}
	if response.Command != "query" {
		t.Fatalf("command = %q, want query", response.Command)
	}
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if response.Result.Query.Source != "flag" {
		t.Fatalf("query source = %q, want flag", response.Result.Query.Source)
	}
	if response.Result.Index.Status != "rebuilt" {
		t.Fatalf("index status = %q, want rebuilt", response.Result.Index.Status)
	}
	if len(response.Result.Results) != 1 || response.Result.Results[0].Entity.ID != "service:login-api" {
		t.Fatalf("results = %#v, want service:login-api", response.Result.Results)
	}
}

func TestQueryCommandReadsTaskFromStdin(t *testing.T) {
	t.Parallel()

	repoDir, manager := initQueryWorkspace(t)
	querier := &spyQuerier{result: retrieval.ResultSet{IndexStatus: "ready"}}

	cmd := newCommand(repoDir, manager, querier)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(bytes.NewBufferString("auth\n"))

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	if querier.lastTask != "auth" {
		t.Fatalf("task = %q, want auth", querier.lastTask)
	}
	response := decodeQuerySuccessResponse(t, stdout.Bytes())
	if response.Result.Query.Source != "stdin" {
		t.Fatalf("query source = %q, want stdin", response.Result.Query.Source)
	}
	if response.Result.Query.Task != "auth" {
		t.Fatalf("task = %q, want auth", response.Result.Query.Task)
	}
}

func TestQueryCommandReturnsStructuredIndexError(t *testing.T) {
	t.Parallel()

	repoDir, manager := initQueryWorkspace(t)
	querier := &spyQuerier{err: &textindex.Error{
		Code:    "text_index_corrupt",
		Message: "local text index is unreadable; rebuild is required",
		Details: map[string]any{"rebuild_hint": "remove the corrupted index file and rerun graph query to rebuild from persisted graph data"},
	}}

	cmd := newCommand(repoDir, manager, querier)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--task", "auth"})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	response := decodeQueryErrorResponse(t, stdout.Bytes())
	if response.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want v1", response.SchemaVersion)
	}
	if response.Command != "query" {
		t.Fatalf("command = %q, want query", response.Command)
	}
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Error.Category != "operational_error" {
		t.Fatalf("error category = %q, want operational_error", response.Error.Category)
	}
	if response.Error.Type != "index_error" {
		t.Fatalf("error type = %q, want index_error", response.Error.Type)
	}
	if response.Error.Code != "text_index_corrupt" {
		t.Fatalf("error code = %q, want text_index_corrupt", response.Error.Code)
	}
	if response.Error.Details["rebuild_hint"] == nil {
		t.Fatalf("error details = %#v, want rebuild_hint", response.Error.Details)
	}
}

func TestQueryCommandWritesEquivalentStructuredOutputToFile(t *testing.T) {
	t.Parallel()

	repoDir, manager := initQueryWorkspace(t)
	querier := &spyQuerier{
		result: retrieval.ResultSet{
			IndexStatus: "ready",
			Results: []retrieval.Result{{
				Entity: retrieval.Entity{ID: "component:authentication"},
			}},
		},
	}

	stdoutCmd := newCommand(repoDir, manager, querier)
	stdout := &bytes.Buffer{}
	stdoutCmd.SetOut(stdout)
	stdoutCmd.SetErr(&bytes.Buffer{})
	stdoutCmd.SetIn(bytes.NewBuffer(nil))
	stdoutCmd.SetArgs([]string{"--task", "auth"})

	if err := stdoutCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext stdout returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "query.json")
	fileCmd := newCommand(repoDir, manager, querier)
	fileStdout := &bytes.Buffer{}
	fileCmd.SetOut(fileStdout)
	fileCmd.SetErr(&bytes.Buffer{})
	fileCmd.SetIn(bytes.NewBuffer(nil))
	fileCmd.SetArgs([]string{"--task", "auth", "--output", outputPath})

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

type spyQuerier struct {
	calls    int
	lastTask string
	result   retrieval.ResultSet
	err      error
}

func (s *spyQuerier) Query(_ *cobra.Command, _ repo.Workspace, task string) (retrieval.ResultSet, error) {
	s.calls++
	s.lastTask = task
	if s.err != nil {
		return retrieval.ResultSet{}, s.err
	}
	return s.result, nil
}

func initQueryWorkspace(t *testing.T) (string, *repo.Manager) {
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

type querySuccessResponse struct {
	SchemaVersion string `json:"schema_version"`
	Command       string `json:"command"`
	Status        string `json:"status"`
	Result        struct {
		Query struct {
			Task   string `json:"task"`
			Source string `json:"source"`
		} `json:"query"`
		Index struct {
			Status string `json:"status"`
		} `json:"index"`
		Results []struct {
			Entity struct {
				ID string `json:"id"`
			} `json:"entity"`
		} `json:"results"`
	} `json:"result"`
}

type queryFailureResponse struct {
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

func decodeQuerySuccessResponse(t *testing.T, payload []byte) querySuccessResponse {
	t.Helper()

	var response querySuccessResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal success response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeQueryErrorResponse(t *testing.T, payload []byte) queryFailureResponse {
	t.Helper()

	var response queryFailureResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal error response: %v\npayload: %s", err, payload)
	}
	return response
}
