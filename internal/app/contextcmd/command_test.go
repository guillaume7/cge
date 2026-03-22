package contextcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/guillaume-galp/cge/internal/app/contextprojector"
	"github.com/guillaume-galp/cge/internal/app/retrieval"
	"github.com/guillaume-galp/cge/internal/infra/repo"
	"github.com/guillaume-galp/cge/internal/infra/textindex"
)

func TestContextCommandReturnsStructuredSuccessEnvelope(t *testing.T) {
	t.Parallel()

	repoDir, manager := initContextWorkspace(t)
	querier := &spyQuerier{
		result: retrieval.ResultSet{
			IndexStatus: "rebuilt",
			Results: []retrieval.Result{{
				Rank:  1,
				Score: 18,
				Entity: retrieval.Entity{
					ID:      "service:login-api",
					Kind:    "Service",
					Title:   "Login API",
					Summary: "Accepts login requests and depends on auth",
				},
				GraphRefs: []retrieval.GraphRef{{
					From:      "service:login-api",
					To:        "component:authentication",
					Kind:      "DEPENDS_ON",
					Direction: "supports_task",
				}},
				MatchedTerms: []string{"auth"},
				Provenance: retrieval.Provenance{
					CreatedBy:        "developer",
					CreatedSessionID: "sess-42",
				},
			}},
		},
	}

	cmd := newCommand(repoDir, manager, querier, contextprojector.NewProjector())
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--task", "what depends on auth?", "--max-tokens", "80"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}

	response := decodeContextSuccessResponse(t, stdout.Bytes())
	if response.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want v1", response.SchemaVersion)
	}
	if response.Command != "context" {
		t.Fatalf("command = %q, want context", response.Command)
	}
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if response.Result.Query.Task != "what depends on auth?" {
		t.Fatalf("task = %q, want what depends on auth?", response.Result.Query.Task)
	}
	if response.Result.Query.Source != "flag" {
		t.Fatalf("query source = %q, want flag", response.Result.Query.Source)
	}
	if response.Result.Index.Status != "rebuilt" {
		t.Fatalf("index status = %q, want rebuilt", response.Result.Index.Status)
	}
	if len(response.Result.Context.Results) != 1 {
		t.Fatalf("results = %d, want 1", len(response.Result.Context.Results))
	}
	if got := response.Result.Context.Results[0].Entity.ID; got != "service:login-api" {
		t.Fatalf("top context entity = %q, want service:login-api", got)
	}
}

func TestContextCommandWritesEquivalentStructuredOutputToFile(t *testing.T) {
	t.Parallel()

	repoDir, manager := initContextWorkspace(t)
	querier := &spyQuerier{
		result: retrieval.ResultSet{
			IndexStatus: "ready",
			Results: []retrieval.Result{{
				Rank:  1,
				Score: 8,
				Entity: retrieval.Entity{
					ID:    "component:authentication",
					Kind:  "Subsystem",
					Title: "Authentication subsystem",
				},
			}},
		},
	}

	stdoutCmd := newCommand(repoDir, manager, querier, contextprojector.NewProjector())
	stdout := &bytes.Buffer{}
	stdoutCmd.SetOut(stdout)
	stdoutCmd.SetErr(&bytes.Buffer{})
	stdoutCmd.SetIn(bytes.NewBuffer(nil))
	stdoutCmd.SetArgs([]string{"--task", "auth", "--max-tokens", "80"})

	if err := stdoutCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext stdout returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "context.json")
	fileCmd := newCommand(repoDir, manager, querier, contextprojector.NewProjector())
	fileStdout := &bytes.Buffer{}
	fileCmd.SetOut(fileStdout)
	fileCmd.SetErr(&bytes.Buffer{})
	fileCmd.SetIn(bytes.NewBuffer(nil))
	fileCmd.SetArgs([]string{"--task", "auth", "--max-tokens", "80", "--output", outputPath})

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

func TestContextCommandReturnsStructuredValidationError(t *testing.T) {
	t.Parallel()

	repoDir, manager := initContextWorkspace(t)
	cmd := newCommand(repoDir, manager, &spyQuerier{}, contextprojector.NewProjector())
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--task", "auth", "--max-tokens", "0"})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	response := decodeContextErrorResponse(t, stdout.Bytes())
	if response.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want v1", response.SchemaVersion)
	}
	if response.Command != "context" {
		t.Fatalf("command = %q, want context", response.Command)
	}
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Error.Category != "validation_error" {
		t.Fatalf("error category = %q, want validation_error", response.Error.Category)
	}
	if response.Error.Type != "validation_error" {
		t.Fatalf("error type = %q, want validation_error", response.Error.Type)
	}
	if response.Error.Code != "invalid_max_tokens" {
		t.Fatalf("error code = %q, want invalid_max_tokens", response.Error.Code)
	}
	if response.Error.Details["max_tokens"] == nil {
		t.Fatalf("error details = %#v, want max_tokens", response.Error.Details)
	}
}

func TestContextCommandReturnsStructuredOperationalIndexError(t *testing.T) {
	t.Parallel()

	repoDir, manager := initContextWorkspace(t)
	cmd := newCommand(repoDir, manager, &spyQuerier{err: &textindex.Error{
		Code:    "text_index_corrupt",
		Message: "local text index is unreadable; rebuild is required",
		Details: map[string]any{"rebuild_hint": "remove the corrupted index file and rerun graph query to rebuild from persisted graph data"},
	}}, contextprojector.NewProjector())
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(bytes.NewBuffer(nil))
	cmd.SetArgs([]string{"--task", "auth"})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	response := decodeContextErrorResponse(t, stdout.Bytes())
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

func initContextWorkspace(t *testing.T) (string, *repo.Manager) {
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

type contextSuccessResponse struct {
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
		Context struct {
			MaxTokens       int `json:"max_tokens"`
			EstimatedTokens int `json:"estimated_tokens"`
			Results         []struct {
				Entity struct {
					ID string `json:"id"`
				} `json:"entity"`
			} `json:"results"`
		} `json:"context"`
	} `json:"result"`
}

type contextFailureResponse struct {
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

func decodeContextSuccessResponse(t *testing.T, payload []byte) contextSuccessResponse {
	t.Helper()

	var response contextSuccessResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal context success response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeContextErrorResponse(t *testing.T, payload []byte) contextFailureResponse {
	t.Helper()

	var response contextFailureResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal context error response: %v\npayload: %s", err, payload)
	}
	return response
}
