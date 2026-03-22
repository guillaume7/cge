package explaincmd

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

func TestExplainCommandReturnsStructuredSuccessEnvelope(t *testing.T) {
	t.Parallel()

	repoDir, manager := initExplainWorkspace(t)
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
					Provenance: retrieval.Provenance{
						CreatedBy:        "developer",
						CreatedSessionID: "sess-42",
					},
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

	response := decodeExplainSuccessResponse(t, stdout.Bytes())
	if response.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want v1", response.SchemaVersion)
	}
	if response.Command != "explain" {
		t.Fatalf("command = %q, want explain", response.Command)
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
	if len(response.Result.Explanation.QueryTerms) == 0 {
		t.Fatalf("query_terms = %#v, want analyzed query terms", response.Result.Explanation.QueryTerms)
	}
	if len(response.Result.Explanation.Results) != 1 {
		t.Fatalf("results = %d, want 1", len(response.Result.Explanation.Results))
	}
	if got := response.Result.Explanation.Results[0].Entity.ID; got != "service:login-api" {
		t.Fatalf("top explained entity = %q, want service:login-api", got)
	}
}

func TestExplainCommandWritesEquivalentStructuredOutputToFile(t *testing.T) {
	t.Parallel()

	repoDir, manager := initExplainWorkspace(t)
	querier := &spyQuerier{
		result: retrieval.ResultSet{
			IndexStatus: "ready",
			Results: []retrieval.Result{{
				Rank:   1,
				Score:  5,
				Entity: retrieval.Entity{ID: "component:authentication", Kind: "Subsystem"},
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

	outputPath := filepath.Join(t.TempDir(), "explain.json")
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

func TestExplainCommandReturnsStructuredValidationError(t *testing.T) {
	t.Parallel()

	repoDir, manager := initExplainWorkspace(t)
	cmd := newCommand(repoDir, manager, &spyQuerier{})
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(bytes.NewBuffer(nil))

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	response := decodeExplainErrorResponse(t, stdout.Bytes())
	if response.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want v1", response.SchemaVersion)
	}
	if response.Command != "explain" {
		t.Fatalf("command = %q, want explain", response.Command)
	}
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Error.Category != "validation_error" {
		t.Fatalf("error category = %q, want validation_error", response.Error.Category)
	}
	if response.Error.Type != "input_error" {
		t.Fatalf("error type = %q, want input_error", response.Error.Type)
	}
	if response.Error.Code != "missing_input" {
		t.Fatalf("error code = %q, want missing_input", response.Error.Code)
	}
	if response.Error.Details["accepted_sources"] == nil {
		t.Fatalf("error details = %#v, want accepted_sources", response.Error.Details)
	}
}

func TestExplainCommandReturnsStructuredOperationalIndexError(t *testing.T) {
	t.Parallel()

	repoDir, manager := initExplainWorkspace(t)
	cmd := newCommand(repoDir, manager, &spyQuerier{err: &textindex.Error{
		Code:    "text_index_corrupt",
		Message: "local text index is unreadable; rebuild is required",
		Details: map[string]any{"rebuild_hint": "remove the corrupted index file and rerun graph query to rebuild from persisted graph data"},
	}})
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--task", "auth"})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	response := decodeExplainErrorResponse(t, stdout.Bytes())
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

type spyQuerier struct {
	result retrieval.ResultSet
	err    error
}

func (s *spyQuerier) Query(_ *cobra.Command, _ repo.Workspace, _ string) (retrieval.ResultSet, error) {
	if s.err != nil {
		return retrieval.ResultSet{}, s.err
	}
	return s.result, nil
}

func initExplainWorkspace(t *testing.T) (string, *repo.Manager) {
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

type explainSuccessResponse struct {
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
		Explanation struct {
			QueryTerms []string `json:"query_terms"`
			Results    []struct {
				Entity struct {
					ID string `json:"id"`
				} `json:"entity"`
			} `json:"results"`
		} `json:"explanation"`
	} `json:"result"`
}

type explainFailureResponse struct {
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

func decodeExplainSuccessResponse(t *testing.T, payload []byte) explainSuccessResponse {
	t.Helper()

	var response explainSuccessResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal explain success response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeExplainErrorResponse(t *testing.T, payload []byte) explainFailureResponse {
	t.Helper()

	var response explainFailureResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal explain error response: %v\npayload: %s", err, payload)
	}
	return response
}
