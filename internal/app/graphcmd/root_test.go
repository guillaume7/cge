package graphcmd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guillaume-galp/cge/internal/app/graphcmd"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func TestRootCommandIncludesMVPCommands(t *testing.T) {
	cmd := graphcmd.NewRootCommand(t.TempDir(), repo.NewManager(repo.NewGitRepositoryLocator()))

	available := map[string]bool{}
	for _, subcommand := range cmd.Commands() {
		available[subcommand.Name()] = true
	}

	for _, name := range []string{"init", "write", "query", "context", "explain", "diff", "stats", "hygiene", "workflow", "lab"} {
		if !available[name] {
			t.Fatalf("command %q not registered", name)
		}
	}
}

func TestGraphInitCreatesWorkspace(t *testing.T) {
	repoDir := initGitRepository(t)

	stdout := &bytes.Buffer{}
	err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}

	workspacePath := filepath.Join(repoDir, repo.WorkspaceDirName)
	assertDirExists(t, workspacePath)
	assertDirExists(t, filepath.Join(workspacePath, "kuzu"))
	assertDirExists(t, filepath.Join(workspacePath, "index"))
	assertDirExists(t, filepath.Join(workspacePath, "tmp"))

	configPath := filepath.Join(workspacePath, repo.ConfigFileName)
	payload, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatalf("read config: %v", readErr)
	}

	var config repo.WorkspaceConfig
	if err := json.Unmarshal(payload, &config); err != nil {
		t.Fatalf("parse config: %v", err)
	}

	resolvedRepoDir, err := filepath.EvalSymlinks(repoDir)
	if err != nil {
		t.Fatalf("resolve repo dir: %v", err)
	}

	resolvedGitDir, err := filepath.EvalSymlinks(filepath.Join(repoDir, ".git"))
	if err != nil {
		t.Fatalf("resolve git dir: %v", err)
	}

	if config.SchemaVersion != repo.WorkspaceSchemaVersion {
		t.Fatalf("schema version = %q, want %q", config.SchemaVersion, repo.WorkspaceSchemaVersion)
	}
	if config.Repository.RootPath != resolvedRepoDir {
		t.Fatalf("root path = %q, want %q", config.Repository.RootPath, resolvedRepoDir)
	}
	if config.Repository.GitCommonDir != resolvedGitDir {
		t.Fatalf("git dir = %q, want %q", config.Repository.GitCommonDir, resolvedGitDir)
	}
	if config.Repository.ID == "" {
		t.Fatal("repository id is empty")
	}
	if got := stdout.String(); got == "" || !bytes.Contains(stdout.Bytes(), []byte("initialized graph workspace")) {
		t.Fatalf("stdout = %q, want initialization message", got)
	}
}

func TestGraphInitIsIdempotent(t *testing.T) {
	repoDir := initGitRepository(t)

	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("first graph init returned error: %v", err)
	}

	workspacePath := filepath.Join(repoDir, repo.WorkspaceDirName)
	configPath := filepath.Join(workspacePath, repo.ConfigFileName)
	originalConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read original config: %v", err)
	}

	sentinelPath := filepath.Join(workspacePath, "tmp", "sentinel.txt")
	if err := os.WriteFile(sentinelPath, []byte("keep me"), 0o644); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	stdout := &bytes.Buffer{}
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("second graph init returned error: %v", err)
	}

	updatedConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read updated config: %v", err)
	}

	if !bytes.Equal(updatedConfig, originalConfig) {
		t.Fatalf("config changed on re-run\noriginal: %s\nupdated: %s", originalConfig, updatedConfig)
	}
	if _, err := os.Stat(sentinelPath); err != nil {
		t.Fatalf("sentinel missing after re-run: %v", err)
	}
	if got := stdout.String(); got == "" || !bytes.Contains(stdout.Bytes(), []byte("already exists")) {
		t.Fatalf("stdout = %q, want idempotent message", got)
	}
}

func TestGraphInitFailsOutsideGitRepository(t *testing.T) {
	nonRepoDir := t.TempDir()

	err := graphcmd.Execute(context.Background(), []string{"init"}, nonRepoDir, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error outside git repository, got nil")
	}

	if got := err.Error(); !containsAll(got,
		"repo-scoped initialization could not determine a repository root",
		"not a git repository",
	) {
		t.Fatalf("error = %q, want clear repository-root failure", got)
	}
}

func TestGraphQueryResolvesWorkspaceFromNestedDirectory(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}
	writeHybridFixture(t, repoDir)

	nestedDir := filepath.Join(repoDir, "pkg", "feature")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("mkdir nested dir: %v", err)
	}

	stdout := &bytes.Buffer{}
	err := graphcmd.Execute(context.Background(), []string{"query", "--task", "what depends on auth?"}, nestedDir, nil, stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("graph query returned error: %v", err)
	}

	response := decodeQueryResponse(t, stdout.Bytes())
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if response.Result.Query.Source != "flag" {
		t.Fatalf("query source = %q, want flag", response.Result.Query.Source)
	}
	if response.Result.Query.Task != "what depends on auth?" {
		t.Fatalf("task = %q, want what depends on auth?", response.Result.Query.Task)
	}
	if len(response.Result.Results) == 0 {
		t.Fatal("expected ranked query results")
	}
	if got := response.Result.Results[0].Entity.ID; got != "service:login-api" {
		t.Fatalf("top result id = %q, want service:login-api", got)
	}
}

func TestGraphWriteReadsPayloadFromStdin(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}

	stdout := &bytes.Buffer{}
	stdin := strings.NewReader(`{"schema_version":"v1","metadata":{"agent_id":"developer","session_id":"sess-42","timestamp":"2026-03-21T14:00:00Z"},"nodes":[],"edges":[]}`)
	err := graphcmd.Execute(context.Background(), []string{"write"}, repoDir, stdin, stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("graph write returned error: %v", err)
	}

	response := struct {
		SchemaVersion string `json:"schema_version"`
		Command       string `json:"command"`
		Status        string `json:"status"`
		Result        struct {
			Summary struct {
				Nodes struct {
					CreatedCount int `json:"created_count"`
					UpdatedCount int `json:"updated_count"`
				} `json:"nodes"`
				Edges struct {
					CreatedCount int `json:"created_count"`
					UpdatedCount int `json:"updated_count"`
				} `json:"edges"`
			} `json:"summary"`
		} `json:"result"`
	}{}
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal write response: %v\npayload: %s", err, stdout.Bytes())
	}
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if response.SchemaVersion != "v1" || response.Command != "write" {
		t.Fatalf("envelope = (%q, %q), want (v1, write)", response.SchemaVersion, response.Command)
	}
	if response.Result.Summary.Nodes.CreatedCount != 0 || response.Result.Summary.Nodes.UpdatedCount != 0 {
		t.Fatalf("node summary = %#v, want zero counts", response.Result.Summary.Nodes)
	}
	if response.Result.Summary.Edges.CreatedCount != 0 || response.Result.Summary.Edges.UpdatedCount != 0 {
		t.Fatalf("edge summary = %#v, want zero counts", response.Result.Summary.Edges)
	}
}

func TestGraphQueryReadsTaskFromStdin(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}
	writeHybridFixture(t, repoDir)

	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("what depends on auth?\n")
	err := graphcmd.Execute(context.Background(), []string{"query"}, repoDir, stdin, stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("graph query returned error: %v", err)
	}

	response := decodeQueryResponse(t, stdout.Bytes())
	if response.Result.Query.Source != "stdin" {
		t.Fatalf("query source = %q, want stdin", response.Result.Query.Source)
	}
	if response.Result.Query.Task != "what depends on auth?" {
		t.Fatalf("task = %q, want what depends on auth?", response.Result.Query.Task)
	}
	if len(response.Result.Results) == 0 {
		t.Fatal("expected stdin-backed query results")
	}
}

func TestGraphChainAgentPayloadIntoWriteAndTaskIntoQuery(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}

	agentPayload := &bytes.Buffer{}
	agentPayload.WriteString(`{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-42",
    "timestamp": "2026-03-21T14:00:00Z"
  },
  "nodes": [
    {
      "id": "component:authentication",
      "kind": "Subsystem",
      "title": "Authentication subsystem",
      "summary": "Handles user authentication and token validation",
      "tags": ["security", "identity"]
    },
    {
      "id": "service:login-api",
      "kind": "Service",
      "title": "Login API",
      "summary": "Accepts login requests and depends on authentication",
      "tags": ["api"]
    }
  ],
  "edges": [
    {
      "from": "service:login-api",
      "to": "component:authentication",
      "kind": "DEPENDS_ON"
    }
  ]
}`)

	writeStdout := &bytes.Buffer{}
	if err := graphcmd.Execute(context.Background(), []string{"write"}, repoDir, agentPayload, writeStdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph write returned error: %v", err)
	}

	writeResponse := decodeWriteResponse(t, writeStdout.Bytes())
	if writeResponse.Status != "ok" {
		t.Fatalf("write status = %q, want ok", writeResponse.Status)
	}
	if writeResponse.Command != "write" || writeResponse.SchemaVersion != "v1" {
		t.Fatalf("write envelope = (%q, %q), want (v1, write)", writeResponse.SchemaVersion, writeResponse.Command)
	}
	if writeResponse.Result.Summary.Nodes.CreatedCount != 2 {
		t.Fatalf("node created_count = %d, want 2", writeResponse.Result.Summary.Nodes.CreatedCount)
	}
	if writeResponse.Result.Summary.Edges.CreatedCount != 1 {
		t.Fatalf("edge created_count = %d, want 1", writeResponse.Result.Summary.Edges.CreatedCount)
	}

	agentTask := &bytes.Buffer{}
	agentTask.WriteString("what depends on auth?\n")

	queryStdout := &bytes.Buffer{}
	if err := graphcmd.Execute(context.Background(), []string{"query"}, repoDir, agentTask, queryStdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph query returned error: %v", err)
	}

	queryResponse := decodeQueryResponse(t, queryStdout.Bytes())
	if queryResponse.Status != "ok" {
		t.Fatalf("query status = %q, want ok", queryResponse.Status)
	}
	if queryResponse.Result.Query.Source != "stdin" {
		t.Fatalf("query source = %q, want stdin", queryResponse.Result.Query.Source)
	}
	if queryResponse.Result.Query.Task != "what depends on auth?" {
		t.Fatalf("task = %q, want what depends on auth?", queryResponse.Result.Query.Task)
	}
	if len(queryResponse.Result.Results) == 0 {
		t.Fatal("expected chained query results")
	}
	if got := queryResponse.Result.Results[0].Entity.ID; got != "service:login-api" {
		t.Fatalf("top chained result = %q, want service:login-api", got)
	}
}

func TestGraphChainContextOutputRemainsConsumableByDownstreamAgent(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}
	writeHybridFixture(t, repoDir)

	contextStdout := &bytes.Buffer{}
	if err := graphcmd.Execute(
		context.Background(),
		[]string{"context", "--max-tokens", "80"},
		repoDir,
		strings.NewReader("what depends on auth?\n"),
		contextStdout,
		&bytes.Buffer{},
	); err != nil {
		t.Fatalf("graph context returned error: %v", err)
	}

	consumed := decodeDownstreamContextConsumption(t, bytes.NewReader(contextStdout.Bytes()))
	if consumed.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want v1", consumed.SchemaVersion)
	}
	if consumed.Command != "context" {
		t.Fatalf("command = %q, want context", consumed.Command)
	}
	if consumed.Status != "ok" {
		t.Fatalf("status = %q, want ok", consumed.Status)
	}
	if consumed.Result.Query.Source != "stdin" {
		t.Fatalf("query source = %q, want stdin", consumed.Result.Query.Source)
	}
	if len(consumed.Result.Context.Results) == 0 {
		t.Fatal("expected downstream-consumable context results")
	}

	top := consumed.Result.Context.Results[0]
	if top.Entity.ID != "service:login-api" {
		t.Fatalf("top downstream context entity = %q, want service:login-api", top.Entity.ID)
	}
	if len(top.Relationships) == 0 || top.Relationships[0].Kind != "DEPENDS_ON" {
		t.Fatalf("relationships = %#v, want DEPENDS_ON relationship", top.Relationships)
	}
	if top.Provenance == nil || top.Provenance.CreatedBy != "developer" {
		t.Fatalf("provenance = %#v, want developer provenance", top.Provenance)
	}
}

func TestGraphChainInvalidPipedPayloadReturnsStructuredError(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}

	invalidAgentPayload := &bytes.Buffer{}
	invalidAgentPayload.WriteString(`{
  "schema_version": "v1",
  "metadata": {
    "timestamp": "2026-03-21T14:00:00Z"
  },
  "nodes": [],
  "edges": []
}`)

	writeStdout := &bytes.Buffer{}
	err := graphcmd.Execute(context.Background(), []string{"write"}, repoDir, invalidAgentPayload, writeStdout, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected graph write error for invalid piped payload, got nil")
	}

	response := decodeWriteErrorResponse(t, writeStdout.Bytes())
	if response.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want v1", response.SchemaVersion)
	}
	if response.Command != "write" {
		t.Fatalf("command = %q, want write", response.Command)
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
	if response.Error.Code != "missing_required_fields" {
		t.Fatalf("error code = %q, want missing_required_fields", response.Error.Code)
	}
	if response.Error.Details["missing_fields"] == nil {
		t.Fatalf("error details = %#v, want missing_fields", response.Error.Details)
	}
}

func TestGraphContextProjectsWithinTokenBudget(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}
	writeHybridFixture(t, repoDir)

	stdout := &bytes.Buffer{}
	err := graphcmd.Execute(context.Background(), []string{"context", "--task", "what depends on auth?", "--max-tokens", "80"}, repoDir, nil, stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("graph context returned error: %v", err)
	}

	response := decodeContextResponse(t, stdout.Bytes())
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if response.Result.Query.Source != "flag" {
		t.Fatalf("query source = %q, want flag", response.Result.Query.Source)
	}
	if response.Result.Context.MaxTokens != 80 {
		t.Fatalf("max_tokens = %d, want 80", response.Result.Context.MaxTokens)
	}
	if response.Result.Context.EstimatedTokens > response.Result.Context.MaxTokens {
		t.Fatalf("estimated_tokens = %d, want <= %d", response.Result.Context.EstimatedTokens, response.Result.Context.MaxTokens)
	}
	if len(response.Result.Context.Results) == 0 {
		t.Fatal("expected projected context results")
	}

	top := response.Result.Context.Results[0]
	if top.Entity.ID != "service:login-api" {
		t.Fatalf("top context entity = %q, want service:login-api", top.Entity.ID)
	}
	if top.Entity.Summary == "" {
		t.Fatalf("top summary = %q, want summary preserved", top.Entity.Summary)
	}
	if len(top.Relationships) == 0 || top.Relationships[0].Kind != "DEPENDS_ON" {
		t.Fatalf("relationships = %#v, want DEPENDS_ON relationship", top.Relationships)
	}
	if top.Provenance == nil || top.Provenance.CreatedBy != "developer" {
		t.Fatalf("provenance = %#v, want developer provenance", top.Provenance)
	}
}

func TestGraphContextTruncatesToHigherValueResultsWithinSmallerBudget(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}
	writeHybridFixture(t, repoDir)

	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("what depends on auth?\n")
	err := graphcmd.Execute(context.Background(), []string{"context", "--max-tokens", "25"}, repoDir, stdin, stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("graph context returned error: %v", err)
	}

	response := decodeContextResponse(t, stdout.Bytes())
	if response.Result.Query.Source != "stdin" {
		t.Fatalf("query source = %q, want stdin", response.Result.Query.Source)
	}
	if response.Result.Context.EstimatedTokens > response.Result.Context.MaxTokens {
		t.Fatalf("estimated_tokens = %d, want <= %d", response.Result.Context.EstimatedTokens, response.Result.Context.MaxTokens)
	}
	if !response.Result.Context.Truncated {
		t.Fatalf("truncated = %v, want true", response.Result.Context.Truncated)
	}
	if response.Result.Context.OmittedResults == 0 {
		t.Fatalf("omitted_results = %d, want positive count", response.Result.Context.OmittedResults)
	}
	if len(response.Result.Context.Results) != 1 {
		t.Fatalf("results = %d, want 1 retained result under tight budget", len(response.Result.Context.Results))
	}
	if got := response.Result.Context.Results[0].Entity.ID; got != "service:login-api" {
		t.Fatalf("retained context entity = %q, want highest-ranked service:login-api", got)
	}
}

func TestGraphContextRejectsInvalidTokenBudget(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}
	writeHybridFixture(t, repoDir)

	for _, maxTokens := range []string{"0", "-5"} {
		t.Run(maxTokens, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			err := graphcmd.Execute(context.Background(), []string{"context", "--task", "auth", "--max-tokens", maxTokens}, repoDir, nil, stdout, &bytes.Buffer{})
			if err == nil {
				t.Fatalf("expected validation error for max_tokens=%s, got nil", maxTokens)
			}

			response := decodeContextErrorResponse(t, stdout.Bytes())
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
				t.Fatalf("error details = %#v, want max_tokens detail", response.Error.Details)
			}
		})
	}
}

func TestGraphExplainReturnsStructuredReasonsForDependencyQuery(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}
	writeHybridFixture(t, repoDir)

	stdout := &bytes.Buffer{}
	if err := graphcmd.Execute(context.Background(), []string{"explain", "--task", "what depends on auth?"}, repoDir, nil, stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph explain returned error: %v", err)
	}

	response := decodeExplainResponse(t, stdout.Bytes())
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if response.Result.Query.Source != "flag" {
		t.Fatalf("query source = %q, want flag", response.Result.Query.Source)
	}
	if len(response.Result.Explanation.QueryTerms) == 0 {
		t.Fatalf("query_terms = %#v, want analyzed terms", response.Result.Explanation.QueryTerms)
	}
	if len(response.Result.Explanation.Results) < 2 {
		t.Fatalf("results = %d, want at least 2 explained candidates", len(response.Result.Explanation.Results))
	}

	top := response.Result.Explanation.Results[0]
	if top.Entity.ID != "service:login-api" {
		t.Fatalf("top explain entity = %q, want service:login-api", top.Entity.ID)
	}
	if len(top.GraphPaths) == 0 || len(top.GraphPaths[0].Steps) == 0 {
		t.Fatalf("graph_paths = %#v, want path steps", top.GraphPaths)
	}
	if top.GraphPaths[0].Steps[0].Kind != "DEPENDS_ON" {
		t.Fatalf("graph path step kind = %q, want DEPENDS_ON", top.GraphPaths[0].Steps[0].Kind)
	}
	if len(top.RankingReasons) == 0 {
		t.Fatalf("ranking_reasons = %#v, want structured ranking reasons", top.RankingReasons)
	}

	auth := findExplainedResultByID(t, response.Result.Explanation.Results, "component:authentication")
	if len(auth.TextMatches) == 0 || len(auth.TextMatches[0].Terms) == 0 {
		t.Fatalf("text_matches = %#v, want text match evidence", auth.TextMatches)
	}
}

func TestGraphExplainReadsTaskFromStdin(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}
	writeHybridFixture(t, repoDir)

	stdout := &bytes.Buffer{}
	stdin := strings.NewReader("what depends on auth?\n")
	if err := graphcmd.Execute(context.Background(), []string{"explain"}, repoDir, stdin, stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph explain returned error: %v", err)
	}

	response := decodeExplainResponse(t, stdout.Bytes())
	if response.Result.Query.Source != "stdin" {
		t.Fatalf("query source = %q, want stdin", response.Result.Query.Source)
	}
	if response.Result.Query.Task != "what depends on auth?" {
		t.Fatalf("task = %q, want what depends on auth?", response.Result.Query.Task)
	}
	if len(response.Result.Explanation.Results) == 0 {
		t.Fatal("expected stdin-backed explanation results")
	}
}

func TestGraphExplainSupportsDependencySubjectPhraseVariant(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}
	writeHybridFixture(t, repoDir)

	stdout := &bytes.Buffer{}
	if err := graphcmd.Execute(context.Background(), []string{"explain", "--task", "what does login-api depend on?"}, repoDir, nil, stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph explain returned error: %v", err)
	}

	response := decodeExplainResponse(t, stdout.Bytes())
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if response.Result.Query.Task != "what does login-api depend on?" {
		t.Fatalf("task = %q, want subject-phrase variant preserved", response.Result.Query.Task)
	}
	if len(response.Result.Explanation.Results) == 0 {
		t.Fatal("expected explanation results for subject-phrase variant")
	}
	if got := response.Result.Explanation.Results[0].Entity.ID; got != "component:authentication" {
		t.Fatalf("top explain entity = %q, want component:authentication", got)
	}
}

func TestGraphExplainRejectsMissingTaskWithStructuredValidationError(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}
	writeHybridFixture(t, repoDir)

	stdout := &bytes.Buffer{}
	err := graphcmd.Execute(context.Background(), []string{"explain"}, repoDir, nil, stdout, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected validation error for missing task, got nil")
	}

	response := decodeExplainErrorResponse(t, stdout.Bytes())
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

func TestGraphExplainIncludesProvenanceAndRankingReasons(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}
	writeHybridFixture(t, repoDir)

	stdout := &bytes.Buffer{}
	if err := graphcmd.Execute(context.Background(), []string{"explain", "--task", "what depends on auth?"}, repoDir, nil, stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph explain returned error: %v", err)
	}

	response := decodeExplainResponse(t, stdout.Bytes())
	top := response.Result.Explanation.Results[0]
	if top.Provenance.Entity.CreatedBy != "developer" || top.Provenance.Entity.CreatedSessionID != "sess-42" {
		t.Fatalf("entity provenance = %#v, want developer/sess-42", top.Provenance.Entity)
	}
	if len(top.Provenance.Relationships) == 0 || top.Provenance.Relationships[0].Provenance.CreatedBy != "developer" {
		t.Fatalf("relationship provenance = %#v, want relationship provenance", top.Provenance.Relationships)
	}
	if top.Provenance.Relationships[0].Provenance.CreatedSessionID != "sess-42" {
		t.Fatalf("relationship provenance = %#v, want sess-42 relationship session", top.Provenance.Relationships)
	}

	graphPathReason := findExplainReason(t, top.RankingReasons, "graph_path")
	if graphPathReason.Contribution != 20 {
		t.Fatalf("graph_path contribution = %v, want 20", graphPathReason.Contribution)
	}
	if got := graphPathReason.Details["role"]; got != "supports_task" {
		t.Fatalf("graph_path role = %#v, want supports_task", got)
	}
	if got := graphPathReason.Details["path_count"]; got != float64(1) {
		t.Fatalf("graph_path path_count = %#v, want 1", got)
	}

	scoreBreakdown := findExplainReason(t, top.RankingReasons, "score_breakdown")
	if got := scoreBreakdown.Details["structural"]; got != float64(20) {
		t.Fatalf("score_breakdown structural = %#v, want 20", got)
	}
	if total, ok := scoreBreakdown.Details["total"].(float64); !ok || total < 20 {
		t.Fatalf("score_breakdown total = %#v, want >= 20", scoreBreakdown.Details["total"])
	}

	auth := findExplainedResultByID(t, response.Result.Explanation.Results, "component:authentication")
	if len(auth.TextMatches) == 0 || auth.TextMatches[0].Contribution <= 0 {
		t.Fatalf("text_matches = %#v, want positive text contribution", auth.TextMatches)
	}
	textReason := findExplainReason(t, auth.RankingReasons, "text_match")
	if textReason.Contribution <= 0 {
		t.Fatalf("text_match contribution = %v, want positive", textReason.Contribution)
	}
	matchedTerms, ok := textReason.Details["matched_terms"].([]any)
	if !ok || len(matchedTerms) == 0 {
		t.Fatalf("text_match details = %#v, want matched_terms", textReason.Details)
	}
}

func TestGraphQueryCombinesStructuralAndTextRetrievalWithProvenance(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}
	writeHybridFixture(t, repoDir)

	stdout := &bytes.Buffer{}
	if err := graphcmd.Execute(context.Background(), []string{"query", "--task", "what depends on auth?"}, repoDir, nil, stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph query returned error: %v", err)
	}

	response := decodeQueryResponse(t, stdout.Bytes())
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if response.Result.Index.Status != "rebuilt" {
		t.Fatalf("index status = %q, want rebuilt on first query", response.Result.Index.Status)
	}
	if len(response.Result.Results) < 2 {
		t.Fatalf("results = %d, want at least 2 ranked candidates", len(response.Result.Results))
	}

	top := response.Result.Results[0]
	if top.Entity.ID != "service:login-api" {
		t.Fatalf("top result id = %q, want service:login-api", top.Entity.ID)
	}
	if top.Scores.Structural <= 0 {
		t.Fatalf("top structural score = %v, want positive score", top.Scores.Structural)
	}
	if len(top.GraphRefs) == 0 || top.GraphRefs[0].Kind != "DEPENDS_ON" {
		t.Fatalf("graph refs = %#v, want DEPENDS_ON provenance", top.GraphRefs)
	}
	if top.Provenance.CreatedBy != "developer" || top.Provenance.CreatedSessionID != "sess-42" {
		t.Fatalf("provenance = %#v, want developer/sess-42", top.Provenance)
	}

	auth := findResultByID(t, response.Result.Results, "component:authentication")
	if auth.Scores.Text <= 0 {
		t.Fatalf("authentication text score = %v, want positive score", auth.Scores.Text)
	}
	if len(auth.MatchedTerms) == 0 {
		t.Fatalf("authentication matched_terms = %#v, want auth/authentication matches", auth.MatchedTerms)
	}
}

func TestGraphQuerySurfacesAuthenticationForAuthTerminologyMismatch(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}
	writeHybridFixture(t, repoDir)

	stdout := &bytes.Buffer{}
	if err := graphcmd.Execute(context.Background(), []string{"query", "--task", "auth"}, repoDir, nil, stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph query returned error: %v", err)
	}

	response := decodeQueryResponse(t, stdout.Bytes())
	if len(response.Result.Results) == 0 {
		t.Fatal("expected text-ranked results for auth")
	}
	if got := response.Result.Results[0].Entity.ID; got != "component:authentication" {
		t.Fatalf("top result id = %q, want component:authentication", got)
	}
	if !containsString(response.Result.Results[0].MatchedTerms, "auth") && !containsString(response.Result.Results[0].MatchedTerms, "authentication") {
		t.Fatalf("matched_terms = %#v, want auth/authentication", response.Result.Results[0].MatchedTerms)
	}
}

func TestGraphQuerySupportsDependsUponPhraseVariant(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}
	writeHybridFixture(t, repoDir)

	stdout := &bytes.Buffer{}
	if err := graphcmd.Execute(context.Background(), []string{"query", "--task", "what depends upon auth?"}, repoDir, nil, stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph query returned error: %v", err)
	}

	response := decodeQueryResponse(t, stdout.Bytes())
	if len(response.Result.Results) == 0 {
		t.Fatal("expected ranked results")
	}
	if got := response.Result.Results[0].Entity.ID; got != "service:login-api" {
		t.Fatalf("top result id = %q, want service:login-api", got)
	}
}

func TestGraphQuerySupportsDependencyLookupFromSubjectPhrase(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}
	writeHybridFixture(t, repoDir)

	stdout := &bytes.Buffer{}
	if err := graphcmd.Execute(context.Background(), []string{"query", "--task", "what does login-api depend on?"}, repoDir, nil, stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph query returned error: %v", err)
	}

	response := decodeQueryResponse(t, stdout.Bytes())
	if len(response.Result.Results) == 0 {
		t.Fatal("expected ranked results")
	}
	if got := response.Result.Results[0].Entity.ID; got != "component:authentication" {
		t.Fatalf("top result id = %q, want component:authentication", got)
	}
}

func TestGraphQueryRebuildsMissingTextIndex(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}
	writeHybridFixture(t, repoDir)

	if err := os.Remove(filepath.Join(repoDir, repo.WorkspaceDirName, "index", "text-index-v1.json")); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove text index: %v", err)
	}

	stdout := &bytes.Buffer{}
	if err := graphcmd.Execute(context.Background(), []string{"query", "--task", "auth"}, repoDir, nil, stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph query returned error: %v", err)
	}

	response := decodeQueryResponse(t, stdout.Bytes())
	if response.Result.Index.Status != "rebuilt" {
		t.Fatalf("index status = %q, want rebuilt", response.Result.Index.Status)
	}
	if len(response.Result.Results) == 0 {
		t.Fatal("expected results after rebuilding missing text index")
	}
}

func TestGraphQueryRefreshesStaleTextIndexAfterGraphWrite(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}
	writeHybridFixture(t, repoDir)

	stdout := &bytes.Buffer{}
	if err := graphcmd.Execute(context.Background(), []string{"query", "--task", "auth"}, repoDir, nil, stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("initial graph query returned error: %v", err)
	}

	updatePayload := `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-43",
    "timestamp": "2026-03-21T15:00:00Z"
  },
  "nodes": [
    {
      "id": "component:authentication",
      "kind": "Subsystem",
      "title": "Authentication subsystem",
      "summary": "Authentication now also covers admin auth flows",
      "tags": ["security", "identity", "admin"]
    }
  ],
  "edges": []
}`
	if err := graphcmd.Execute(context.Background(), []string{"write"}, repoDir, strings.NewReader(updatePayload), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph write returned error: %v", err)
	}

	stdout.Reset()
	if err := graphcmd.Execute(context.Background(), []string{"query", "--task", "admin auth"}, repoDir, nil, stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph query after graph update returned error: %v", err)
	}

	response := decodeQueryResponse(t, stdout.Bytes())
	if response.Result.Index.Status != "rebuilt" {
		t.Fatalf("index status = %q, want rebuilt after graph change", response.Result.Index.Status)
	}
	if got := response.Result.Results[0].Entity.ID; got != "component:authentication" {
		t.Fatalf("top result id = %q, want component:authentication", got)
	}
	if !containsString(response.Result.Results[0].MatchedTerms, "admin") {
		t.Fatalf("matched_terms = %#v, want admin after stale-index refresh", response.Result.Results[0].MatchedTerms)
	}
}

func TestGraphQueryFailsClearlyWhenTextIndexIsCorrupted(t *testing.T) {
	repoDir := initGitRepository(t)
	if err := graphcmd.Execute(context.Background(), []string{"init"}, repoDir, nil, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph init returned error: %v", err)
	}
	writeHybridFixture(t, repoDir)

	stdout := &bytes.Buffer{}
	if err := graphcmd.Execute(context.Background(), []string{"query", "--task", "auth"}, repoDir, nil, stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("initial graph query returned error: %v", err)
	}

	indexPath := filepath.Join(repoDir, repo.WorkspaceDirName, "index", "text-index-v1.json")
	if err := os.WriteFile(indexPath, []byte("{broken"), 0o644); err != nil {
		t.Fatalf("corrupt text index: %v", err)
	}

	stdout.Reset()
	err := graphcmd.Execute(context.Background(), []string{"query", "--task", "auth"}, repoDir, nil, stdout, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected corrupted text index error, got nil")
	}

	response := decodeQueryErrorResponse(t, stdout.Bytes())
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

func TestGraphCommandsFailClearlyWhenWorkspaceMissing(t *testing.T) {
	repoDir := initGitRepository(t)

	tests := []struct {
		name string
		args []string
	}{
		{name: "write", args: []string{"write"}},
		{name: "query", args: []string{"query"}},
		{name: "context", args: []string{"context"}},
		{name: "explain", args: []string{"explain"}},
		{name: "diff", args: []string{"diff"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stdout := &bytes.Buffer{}
			err := graphcmd.Execute(context.Background(), tc.args, repoDir, nil, stdout, &bytes.Buffer{})
			if err == nil {
				t.Fatalf("expected error for graph %s without workspace, got nil", tc.name)
			}

			switch tc.name {
			case "query":
				response := decodeQueryErrorResponse(t, stdout.Bytes())
				if response.Error.Category != "operational_error" || response.Error.Type != "workspace_error" {
					t.Fatalf("query error = %#v, want operational workspace error", response.Error)
				}
				if response.Error.Code != "workspace_not_initialized" {
					t.Fatalf("query error code = %q, want workspace_not_initialized", response.Error.Code)
				}
			case "context":
				response := decodeContextErrorResponse(t, stdout.Bytes())
				if response.Error.Category != "operational_error" || response.Error.Type != "workspace_error" {
					t.Fatalf("context error = %#v, want operational workspace error", response.Error)
				}
			case "explain":
				response := decodeExplainErrorResponse(t, stdout.Bytes())
				if response.Error.Category != "operational_error" || response.Error.Type != "workspace_error" {
					t.Fatalf("explain error = %#v, want operational workspace error", response.Error)
				}
			case "write", "diff":
				var response struct {
					Error struct {
						Category string         `json:"category"`
						Type     string         `json:"type"`
						Code     string         `json:"code"`
						Details  map[string]any `json:"details"`
					} `json:"error"`
				}
				if unmarshalErr := json.Unmarshal(stdout.Bytes(), &response); unmarshalErr != nil {
					t.Fatalf("json.Unmarshal workspace error response: %v\npayload: %s", unmarshalErr, stdout.Bytes())
				}
				if response.Error.Category != "operational_error" || response.Error.Type != "workspace_error" {
					t.Fatalf("%s error = %#v, want operational workspace error", tc.name, response.Error)
				}
				if response.Error.Code != "workspace_not_initialized" {
					t.Fatalf("%s error code = %q, want workspace_not_initialized", tc.name, response.Error.Code)
				}
			}
		})
	}
}

func initGitRepository(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git init failed: %v\n%s", err, output)
	}

	return repoDir
}

func assertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", path)
	}
}

func containsAll(s string, fragments ...string) bool {
	for _, fragment := range fragments {
		if !bytes.Contains([]byte(s), []byte(fragment)) {
			return false
		}
	}
	return true
}

type queryResponse struct {
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
		Results []queryResult `json:"results"`
	} `json:"result"`
}

type queryResult struct {
	Entity struct {
		ID string `json:"id"`
	} `json:"entity"`
	Scores struct {
		Text       float64 `json:"text"`
		Structural float64 `json:"structural"`
	} `json:"scores"`
	MatchedTerms []string `json:"matched_terms"`
	GraphRefs    []struct {
		Kind string `json:"kind"`
	} `json:"graph_refs"`
	Provenance struct {
		CreatedBy        string `json:"created_by"`
		CreatedSessionID string `json:"created_session_id"`
	} `json:"provenance"`
}

type queryErrorResponse struct {
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

type writeResponse struct {
	SchemaVersion string `json:"schema_version"`
	Command       string `json:"command"`
	Status        string `json:"status"`
	Result        struct {
		Summary struct {
			Nodes struct {
				CreatedCount int `json:"created_count"`
				UpdatedCount int `json:"updated_count"`
			} `json:"nodes"`
			Edges struct {
				CreatedCount int `json:"created_count"`
				UpdatedCount int `json:"updated_count"`
			} `json:"edges"`
		} `json:"summary"`
	} `json:"result"`
}

type writeErrorResponse struct {
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

type contextResponse struct {
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
			MaxTokens       int             `json:"max_tokens"`
			EstimatedTokens int             `json:"estimated_tokens"`
			Truncated       bool            `json:"truncated"`
			OmittedResults  int             `json:"omitted_results"`
			Results         []contextResult `json:"results"`
		} `json:"context"`
	} `json:"result"`
}

type contextResult struct {
	Entity struct {
		ID      string `json:"id"`
		Summary string `json:"summary"`
	} `json:"entity"`
	Relationships []struct {
		Kind string `json:"kind"`
		Peer string `json:"peer"`
	} `json:"relationships"`
	Provenance *struct {
		CreatedBy string `json:"created_by"`
	} `json:"provenance"`
}

type explainResponse struct {
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
			QueryTerms []string          `json:"query_terms"`
			Results    []explainedResult `json:"results"`
		} `json:"explanation"`
	} `json:"result"`
}

type explainedResult struct {
	Score  float64 `json:"score"`
	Entity struct {
		ID string `json:"id"`
	} `json:"entity"`
	TextMatches []struct {
		Terms        []string `json:"terms"`
		Contribution float64  `json:"contribution"`
	} `json:"text_matches"`
	GraphPaths []struct {
		Role         string  `json:"role"`
		Contribution float64 `json:"contribution"`
		Steps        []struct {
			Kind string `json:"kind"`
		} `json:"steps"`
	} `json:"graph_paths"`
	Provenance struct {
		Entity struct {
			CreatedBy        string `json:"created_by"`
			CreatedSessionID string `json:"created_session_id"`
		} `json:"entity"`
		Relationships []struct {
			Provenance struct {
				CreatedBy        string `json:"created_by"`
				CreatedSessionID string `json:"created_session_id"`
			} `json:"provenance"`
		} `json:"relationships"`
	} `json:"provenance"`
	RankingReasons []struct {
		Type         string         `json:"type"`
		Contribution float64        `json:"contribution"`
		Details      map[string]any `json:"details"`
	} `json:"ranking_reasons"`
}

type explainErrorResponse struct {
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

type contextErrorResponse struct {
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

type downstreamContextConsumption struct {
	SchemaVersion string `json:"schema_version"`
	Command       string `json:"command"`
	Status        string `json:"status"`
	Result        struct {
		Query struct {
			Task   string `json:"task"`
			Source string `json:"source"`
		} `json:"query"`
		Context struct {
			Results []struct {
				Entity struct {
					ID string `json:"id"`
				} `json:"entity"`
				Relationships []struct {
					Kind string `json:"kind"`
					Peer string `json:"peer"`
				} `json:"relationships"`
				Provenance *struct {
					CreatedBy string `json:"created_by"`
				} `json:"provenance"`
			} `json:"results"`
		} `json:"context"`
	} `json:"result"`
}

func writeHybridFixture(t *testing.T, repoDir string) {
	t.Helper()

	payload := `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-42",
    "timestamp": "2026-03-21T14:00:00Z"
  },
  "nodes": [
    {
      "id": "component:authentication",
      "kind": "Subsystem",
      "title": "Authentication subsystem",
      "summary": "Handles user authentication, tokens, and session checks for the platform",
      "content": "Authentication verifies identity before protected requests are accepted.",
      "properties": {
        "aliases": ["identity-service"]
      },
      "tags": ["security", "identity"]
    },
    {
      "id": "service:login-api",
      "kind": "Service",
      "title": "Login API",
      "summary": "Accepts login requests and depends on the authentication subsystem",
      "tags": ["api"]
    },
    {
      "id": "service:billing",
      "kind": "Service",
      "title": "Billing service",
      "summary": "Handles invoices and payment reconciliation",
      "tags": ["payments"]
    }
  ],
  "edges": [
    {
      "from": "service:login-api",
      "to": "component:authentication",
      "kind": "DEPENDS_ON"
    }
  ]
}`

	stdout := &bytes.Buffer{}
	if err := graphcmd.Execute(context.Background(), []string{"write"}, repoDir, strings.NewReader(payload), stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("graph write returned error: %v", err)
	}
}

func decodeQueryResponse(t *testing.T, payload []byte) queryResponse {
	t.Helper()

	var response queryResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal query response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeWriteResponse(t *testing.T, payload []byte) writeResponse {
	t.Helper()

	var response writeResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal write response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeWriteErrorResponse(t *testing.T, payload []byte) writeErrorResponse {
	t.Helper()

	var response writeErrorResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal write error response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeQueryErrorResponse(t *testing.T, payload []byte) queryErrorResponse {
	t.Helper()

	var response queryErrorResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal query error response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeDownstreamContextConsumption(t *testing.T, r *bytes.Reader) downstreamContextConsumption {
	t.Helper()

	var response downstreamContextConsumption
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&response); err != nil {
		t.Fatalf("json.Decode downstream context payload: %v", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("downstream context payload contains trailing content, second decode error = %v", err)
	}
	return response
}

func decodeContextResponse(t *testing.T, payload []byte) contextResponse {
	t.Helper()

	var response contextResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal context response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeExplainResponse(t *testing.T, payload []byte) explainResponse {
	t.Helper()

	var response explainResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal explain response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeExplainErrorResponse(t *testing.T, payload []byte) explainErrorResponse {
	t.Helper()

	var response explainErrorResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal explain error response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeContextErrorResponse(t *testing.T, payload []byte) contextErrorResponse {
	t.Helper()

	var response contextErrorResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal context error response: %v\npayload: %s", err, payload)
	}
	return response
}

func findExplainedResultByID(t *testing.T, results []explainedResult, id string) explainedResult {
	t.Helper()
	for _, result := range results {
		if result.Entity.ID == id {
			return result
		}
	}
	t.Fatalf("explained result %q not found in %#v", id, results)
	return explainedResult{}
}

func containsExplainReason(reasons []struct {
	Type string `json:"type"`
}, want string) bool {
	for _, reason := range reasons {
		if reason.Type == want {
			return true
		}
	}
	return false
}

func findExplainReason(t *testing.T, reasons []struct {
	Type         string         `json:"type"`
	Contribution float64        `json:"contribution"`
	Details      map[string]any `json:"details"`
}, want string) struct {
	Type         string         `json:"type"`
	Contribution float64        `json:"contribution"`
	Details      map[string]any `json:"details"`
} {
	t.Helper()
	for _, reason := range reasons {
		if reason.Type == want {
			return reason
		}
	}
	t.Fatalf("ranking reason %q not found in %#v", want, reasons)
	return struct {
		Type         string         `json:"type"`
		Contribution float64        `json:"contribution"`
		Details      map[string]any `json:"details"`
	}{}
}

func findResultByID(t *testing.T, results []queryResult, id string) queryResult {
	t.Helper()
	for _, result := range results {
		if result.Entity.ID == id {
			return result
		}
	}
	t.Fatalf("result %q not found in %#v", id, results)
	return queryResult{}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
