package retrieval

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	graphpayload "github.com/guillaume-galp/cge/internal/domain/payload"
	"github.com/guillaume-galp/cge/internal/infra/kuzu"
	"github.com/guillaume-galp/cge/internal/infra/repo"
	"github.com/guillaume-galp/cge/internal/infra/textindex"
)

func TestEngineQueryCombinesHybridRetrievalForAuthScenario(t *testing.T) {
	t.Parallel()

	_, workspace := initRetrievalWorkspace(t)
	writeRetrievalFixture(t, workspace)

	engine := NewEngine(nil, nil)
	result, err := engine.Query(context.Background(), workspace, "what depends on auth?")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	if result.IndexStatus != "rebuilt" {
		t.Fatalf("index status = %q, want rebuilt", result.IndexStatus)
	}
	if len(result.Results) < 2 {
		t.Fatalf("results = %d, want at least 2", len(result.Results))
	}
	if got := result.Results[0].Entity.ID; got != "service:login-api" {
		t.Fatalf("top result id = %q, want service:login-api", got)
	}
	if result.Results[0].Scores.Structural <= 0 {
		t.Fatalf("top structural score = %v, want positive", result.Results[0].Scores.Structural)
	}

	auth := requireResult(t, result.Results, "component:authentication")
	if auth.Scores.Text <= 0 {
		t.Fatalf("authentication text score = %v, want positive", auth.Scores.Text)
	}
	if !contains(auth.MatchedTerms, "auth") && !contains(auth.MatchedTerms, "authentication") {
		t.Fatalf("matched terms = %#v, want auth/authentication", auth.MatchedTerms)
	}
	if auth.Provenance.CreatedBy != "developer" || auth.Provenance.CreatedSessionID != "sess-42" {
		t.Fatalf("provenance = %#v, want developer/sess-42", auth.Provenance)
	}
}

func TestEngineQuerySupportsDependsUponPhraseVariant(t *testing.T) {
	t.Parallel()

	_, workspace := initRetrievalWorkspace(t)
	writeRetrievalFixture(t, workspace)

	engine := NewEngine(nil, nil)
	result, err := engine.Query(context.Background(), workspace, "what depends upon auth?")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	if len(result.Results) == 0 {
		t.Fatal("expected ranked results")
	}
	if got := result.Results[0].Entity.ID; got != "service:login-api" {
		t.Fatalf("top result id = %q, want service:login-api", got)
	}
}

func TestEngineQuerySupportsDependencyLookupFromSubjectPhrase(t *testing.T) {
	t.Parallel()

	_, workspace := initRetrievalWorkspace(t)
	writeRetrievalFixture(t, workspace)

	engine := NewEngine(nil, nil)
	result, err := engine.Query(context.Background(), workspace, "what does login-api depend on?")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}

	if len(result.Results) == 0 {
		t.Fatal("expected ranked results")
	}
	if got := result.Results[0].Entity.ID; got != "component:authentication" {
		t.Fatalf("top result id = %q, want component:authentication", got)
	}
	if result.Results[0].Scores.Structural <= 0 {
		t.Fatalf("top structural score = %v, want positive", result.Results[0].Scores.Structural)
	}
}

func TestEngineQueryRebuildsMissingIndexFromPersistedGraph(t *testing.T) {
	t.Parallel()

	repoDir, workspace := initRetrievalWorkspace(t)
	writeRetrievalFixture(t, workspace)

	engine := NewEngine(nil, nil)
	result, err := engine.Query(context.Background(), workspace, "auth")
	if err != nil {
		t.Fatalf("first Query returned error: %v", err)
	}
	if result.IndexStatus != "rebuilt" {
		t.Fatalf("index status = %q, want rebuilt", result.IndexStatus)
	}

	indexPath := filepath.Join(repoDir, repo.WorkspaceDirName, "index", textindex.FileName)
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("stat text index: %v", err)
	}
}

func TestEngineQueryFailsClearlyWhenTextIndexIsCorrupted(t *testing.T) {
	t.Parallel()

	repoDir, workspace := initRetrievalWorkspace(t)
	writeRetrievalFixture(t, workspace)

	engine := NewEngine(nil, nil)
	if _, err := engine.Query(context.Background(), workspace, "auth"); err != nil {
		t.Fatalf("initial Query returned error: %v", err)
	}

	indexPath := filepath.Join(repoDir, repo.WorkspaceDirName, "index", textindex.FileName)
	if err := os.WriteFile(indexPath, []byte("{broken"), 0o644); err != nil {
		t.Fatalf("corrupt text index: %v", err)
	}

	_, err := engine.Query(context.Background(), workspace, "auth")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var indexErr *textindex.Error
	if !errors.As(err, &indexErr) {
		t.Fatalf("error type = %T, want *textindex.Error", err)
	}
	if indexErr.Code != "text_index_corrupt" {
		t.Fatalf("error code = %q, want text_index_corrupt", indexErr.Code)
	}
	if indexErr.Details["rebuild_hint"] == nil {
		t.Fatalf("error details = %#v, want rebuild_hint", indexErr.Details)
	}
}

func initRetrievalWorkspace(t *testing.T) (string, repo.Workspace) {
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

func writeRetrievalFixture(t *testing.T, workspace repo.Workspace) {
	t.Helper()

	store := kuzu.NewStore()
	_, err := store.Write(context.Background(), workspace, graphpayload.Envelope{
		SchemaVersion: graphpayload.SchemaVersionV1,
		Metadata: graphpayload.Metadata{
			AgentID:   "developer",
			SessionID: "sess-42",
			Timestamp: "2026-03-21T14:00:00Z",
		},
		Nodes: mustRawMessages(t, []string{`{
      "id": "component:authentication",
      "kind": "Subsystem",
      "title": "Authentication subsystem",
      "summary": "Handles user authentication, tokens, and session checks for the platform",
      "content": "Authentication verifies identity before protected requests are accepted.",
      "properties": {
        "aliases": ["identity-service"]
      },
      "tags": ["security", "identity"]
    }`, `{
      "id": "service:login-api",
      "kind": "Service",
      "title": "Login API",
      "summary": "Accepts login requests and depends on the authentication subsystem",
      "tags": ["api"]
    }`, `{
      "id": "service:billing",
      "kind": "Service",
      "title": "Billing service",
      "summary": "Handles invoices and payment reconciliation",
      "tags": ["payments"]
    }`}),
		Edges: mustRawMessages(t, []string{`{
      "from": "service:login-api",
      "to": "component:authentication",
      "kind": "DEPENDS_ON"
    }`}),
	})
	if err != nil {
		t.Fatalf("store.Write returned error: %v", err)
	}
}

func mustRawMessages(t *testing.T, payloads []string) []json.RawMessage {
	t.Helper()

	values := make([]json.RawMessage, 0, len(payloads))
	for _, payload := range payloads {
		values = append(values, json.RawMessage(payload))
	}
	return values
}

func requireResult(t *testing.T, results []Result, id string) Result {
	t.Helper()
	for _, result := range results {
		if result.Entity.ID == id {
			return result
		}
	}
	t.Fatalf("result %q not found in %#v", id, results)
	return Result{}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
