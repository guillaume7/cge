package hygienecmd

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	kuzudb "github.com/kuzudb/go-kuzu"

	"github.com/guillaume-galp/cge/internal/app/diffcmd"
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

func TestHygieneCommandSuggestReturnsStructuredPlanWithAllCandidateTypes(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initHygieneWorkspace(t)
	writeHygieneRevision(t, workspace, combinedGraphPayload)

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
		t.Fatalf("result.mode = %q, want suggest", response.Result.Mode)
	}
	assertNonEmptyHexLikeString(t, "result.plan.snapshot_anchor", response.Result.Plan.SnapshotAnchor)

	raw := decodeHygieneJSONMap(t, stdout.Bytes())
	result := requireJSONObject(t, raw, "result")
	if got := requireJSONString(t, result, "mode"); got != "suggest" {
		t.Fatalf("result.mode = %q, want suggest", got)
	}
	plan := requireJSONObject(t, result, "plan")
	assertNonEmptyHexLikeString(t, "result.plan.snapshot_anchor", requireJSONString(t, plan, "snapshot_anchor"))

	suggestions := requireJSONObject(t, plan, "suggestions")
	duplicateGroups := requireJSONArray(t, suggestions, "duplicate_groups")
	if len(duplicateGroups) == 0 {
		t.Error("result.plan.suggestions.duplicate_groups = [], want at least one duplicate suggestion")
	}
	orphanNodes := requireJSONArray(t, suggestions, "orphan_nodes")
	if len(orphanNodes) == 0 {
		t.Error("result.plan.suggestions.orphan_nodes = [], want at least one orphan suggestion")
	}
	contradictions := requireJSONArray(t, suggestions, "contradictions")
	if len(contradictions) == 0 {
		t.Error("result.plan.suggestions.contradictions = [], want at least one contradiction suggestion")
	}

	actions := requireJSONArray(t, plan, "actions")
	if len(actions) == 0 {
		t.Error("result.plan.actions = [], want at least one action")
	}

	expectedTypes := map[string]struct{}{
		"consolidate_duplicate_nodes": {},
		"prune_orphan_nodes":          {},
		"resolve_contradiction":       {},
	}
	seenTypes := map[string]struct{}{}
	for index, actionValue := range actions {
		action, ok := actionValue.(map[string]any)
		if !ok {
			t.Fatalf("result.plan.actions[%d] = %#v, want object", index, actionValue)
		}
		actionID := requireJSONString(t, action, "action_id")
		if actionID == "" {
			t.Fatalf("result.plan.actions[%d].action_id = %q, want non-empty string", index, actionID)
		}
		actionType := requireJSONString(t, action, "type")
		if _, ok := expectedTypes[actionType]; !ok {
			t.Fatalf("result.plan.actions[%d].type = %q, want one of %#v", index, actionType, keys(expectedTypes))
		}
		seenTypes[actionType] = struct{}{}
		targetIDs := requireStringArray(t, action, "target_ids")
		if len(targetIDs) == 0 {
			t.Fatalf("result.plan.actions[%d].target_ids = %#v, want non-empty string array", index, targetIDs)
		}
		if explanation := requireJSONString(t, action, "explanation"); explanation == "" {
			t.Fatalf("result.plan.actions[%d].explanation = %q, want non-empty string", index, explanation)
		}
	}
	for actionType := range expectedTypes {
		if _, ok := seenTypes[actionType]; !ok {
			t.Fatalf("result.plan.actions types = %#v, want to include %q", keys(seenTypes), actionType)
		}
	}
}

func TestHygieneCommandSuggestReturnsEmptyButValidPlanForCleanGraph(t *testing.T) {
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
		t.Fatalf("result.mode = %q, want suggest", response.Result.Mode)
	}
	assertNonEmptyHexLikeString(t, "result.plan.snapshot_anchor", response.Result.Plan.SnapshotAnchor)
	if response.Result.Plan.Suggestions.DuplicateGroups == nil {
		t.Fatal("result.plan.suggestions.duplicate_groups = nil, want []")
	}
	if len(response.Result.Plan.Suggestions.DuplicateGroups) != 0 {
		t.Fatalf("result.plan.suggestions.duplicate_groups = %#v, want []", response.Result.Plan.Suggestions.DuplicateGroups)
	}
	if response.Result.Plan.Suggestions.OrphanNodes == nil {
		t.Fatal("result.plan.suggestions.orphan_nodes = nil, want []")
	}
	if len(response.Result.Plan.Suggestions.OrphanNodes) != 0 {
		t.Fatalf("result.plan.suggestions.orphan_nodes = %#v, want []", response.Result.Plan.Suggestions.OrphanNodes)
	}
	if response.Result.Plan.Suggestions.Contradictions == nil {
		t.Fatal("result.plan.suggestions.contradictions = nil, want []")
	}
	if len(response.Result.Plan.Suggestions.Contradictions) != 0 {
		t.Fatalf("result.plan.suggestions.contradictions = %#v, want []", response.Result.Plan.Suggestions.Contradictions)
	}
	if response.Result.Plan.Actions == nil {
		t.Fatal("result.plan.actions = nil, want []")
	}
	if len(response.Result.Plan.Actions) != 0 {
		t.Fatalf("result.plan.actions = %#v, want []", response.Result.Plan.Actions)
	}

	raw := decodeHygieneJSONMap(t, stdout.Bytes())
	result := requireJSONObject(t, raw, "result")
	plan := requireJSONObject(t, result, "plan")
	assertNonEmptyHexLikeString(t, "result.plan.snapshot_anchor", requireJSONString(t, plan, "snapshot_anchor"))
	suggestions := requireJSONObject(t, plan, "suggestions")
	if duplicateGroups := requireJSONArray(t, suggestions, "duplicate_groups"); len(duplicateGroups) != 0 {
		t.Fatalf("raw result.plan.suggestions.duplicate_groups = %#v, want []", duplicateGroups)
	}
	if orphanNodes := requireJSONArray(t, suggestions, "orphan_nodes"); len(orphanNodes) != 0 {
		t.Fatalf("raw result.plan.suggestions.orphan_nodes = %#v, want []", orphanNodes)
	}
	if contradictions := requireJSONArray(t, suggestions, "contradictions"); len(contradictions) != 0 {
		t.Fatalf("raw result.plan.suggestions.contradictions = %#v, want []", contradictions)
	}
	if actions := requireJSONArray(t, plan, "actions"); len(actions) != 0 {
		t.Fatalf("raw result.plan.actions = %#v, want []", actions)
	}
}

func TestHygieneCommandSuggestStdoutMatchesFileOutput(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initHygieneWorkspace(t)
	writeHygieneRevision(t, workspace, combinedGraphPayload)

	stdoutCmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	stdoutCmd.SetOut(stdout)
	stdoutCmd.SetErr(&bytes.Buffer{})
	if err := stdoutCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("stdout ExecuteContext returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "hygiene-plan.json")
	fileCmd := newCommand(repoDir, manager, nil)
	fileStdout := &bytes.Buffer{}
	fileCmd.SetOut(fileStdout)
	fileCmd.SetErr(&bytes.Buffer{})
	fileCmd.SetArgs([]string{"--output", outputPath})
	if err := fileCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("file ExecuteContext returned error: %v", err)
	}
	if fileStdout.Len() != 0 {
		t.Fatalf("stdout with --output = %q, want empty", fileStdout.String())
	}

	filePayload, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !bytes.Equal(stdout.Bytes(), filePayload) {
		t.Fatalf("stdout payload != file payload\nstdout: %s\nfile: %s", stdout.Bytes(), filePayload)
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

func TestHygieneCommandSuggestReturnsContradictionCandidatesWithProposedResolutions(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initHygieneWorkspace(t)
	const contradictionGraphPayload = `{
	  "schema_version": "v1",
	  "metadata": {
	    "agent_id": "developer",
	    "session_id": "sess-contradiction",
	    "timestamp": "2026-03-22T10:00:00Z",
	    "revision": {
	      "reason": "Seed contradiction test"
	    }
	  },
	  "nodes": [
	    {
	      "id": "adr:db-postgres",
	      "kind": "ADR",
	      "title": "Database choice for persistence",
	      "summary": "PostgreSQL selected as primary database",
	      "properties": {
	        "value": "postgresql"
	      }
	    },
	    {
	      "id": "adr:db-mysql",
	      "kind": "ADR",
	      "title": "Database choice for persistence",
	      "summary": "MySQL selected as primary database",
	      "properties": {
	        "value": "mysql"
	      }
	    },
	    {
	      "id": "service:api",
	      "kind": "Service",
	      "title": "API Service",
	      "summary": "REST API service"
	    }
	  ],
	  "edges": [
	    {
	      "from": "adr:db-postgres",
	      "to": "adr:db-mysql",
	      "kind": "CONTRADICTS"
	    },
	    {
	      "from": "service:api",
	      "to": "adr:db-postgres",
	      "kind": "DEPENDS_ON"
	    }
	  ]
	}`
	writeHygieneRevision(t, workspace, contradictionGraphPayload)

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

	contradictions := response.Result.Plan.Suggestions.Contradictions
	if contradictions == nil {
		t.Fatal("expected contradictions suggestions slice, got nil")
	}
	if len(contradictions) == 0 {
		t.Fatal("expected at least one contradiction suggestion")
	}

	foundExpectedNodes := false
	for _, contradiction := range contradictions {
		if contradiction.ActionID == "" || contradiction.Reason == "" {
			t.Fatalf("contradiction suggestion missing machine-readable fields: %#v", contradiction)
		}
		if len(contradiction.NodeIDs) < 2 {
			t.Fatalf("contradiction node_ids = %#v, want at least 2 nodes", contradiction.NodeIDs)
		}
		if contradiction.CanonicalNodeID == "" {
			t.Fatalf("expected canonical node id in contradiction suggestion: %#v", contradiction)
		}
		if len(contradiction.Conflicts) < 2 {
			t.Fatalf("conflicts = %#v, want at least 2 conflicting facts", contradiction.Conflicts)
		}
		if contradiction.Resolution.Strategy == "" || contradiction.Resolution.CanonicalNodeID == "" || contradiction.Resolution.Explanation == "" {
			t.Fatalf("resolution = %#v, want a non-empty structured resolution path", contradiction.Resolution)
		}

		factsByNode := map[string]string{}
		for _, fact := range contradiction.Conflicts {
			if fact.NodeID == "" || fact.Value == "" {
				t.Fatalf("conflict fact missing node_id/value: %#v", fact)
			}
			factsByNode[fact.NodeID] = fact.Value
		}
		if factsByNode["adr:db-postgres"] != "" && factsByNode["adr:db-mysql"] != "" {
			foundExpectedNodes = true
		}
	}
	if !foundExpectedNodes {
		t.Fatalf("expected contradiction involving adr:db-postgres and adr:db-mysql, got %#v", contradictions)
	}

	asString := func(value any) string {
		stringValue, _ := value.(string)
		return stringValue
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal raw hygiene response: %v", err)
	}
	result, ok := payload["result"].(map[string]any)
	if !ok {
		t.Fatalf("raw result = %#v, want object", payload["result"])
	}
	plan, ok := result["plan"].(map[string]any)
	if !ok {
		t.Fatalf("raw plan = %#v, want object", result["plan"])
	}
	suggestions, ok := plan["suggestions"].(map[string]any)
	if !ok {
		t.Fatalf("raw suggestions = %#v, want object", plan["suggestions"])
	}
	rawContradictions, ok := suggestions["contradictions"].([]any)
	if !ok {
		t.Fatalf("raw contradictions = %#v, want array", suggestions["contradictions"])
	}
	if len(rawContradictions) == 0 {
		t.Fatal("expected raw contradictions array to contain at least one entry")
	}
	for _, item := range rawContradictions {
		contradiction, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("raw contradiction = %#v, want object", item)
		}
		if asString(contradiction["action_id"]) == "" {
			t.Fatalf("raw contradiction missing action_id: %#v", contradiction)
		}
		if asString(contradiction["reason"]) == "" {
			t.Fatalf("raw contradiction missing reason: %#v", contradiction)
		}

		factsValue, ok := contradiction["facts"]
		if !ok {
			factsValue, ok = contradiction["conflicts"]
		}
		if !ok {
			t.Fatalf("raw contradiction missing facts/conflicts array: %#v", contradiction)
		}
		facts, ok := factsValue.([]any)
		if !ok || len(facts) == 0 {
			t.Fatalf("raw facts = %#v, want non-empty array", factsValue)
		}
		for _, factValue := range facts {
			fact, ok := factValue.(map[string]any)
			if !ok {
				t.Fatalf("raw fact = %#v, want object", factValue)
			}
			if asString(fact["node_id"]) == "" || asString(fact["value"]) == "" {
				t.Fatalf("raw fact missing node_id/value: %#v", fact)
			}
		}

		resolutionValue, ok := contradiction["proposed_resolution"]
		if !ok {
			resolutionValue, ok = contradiction["resolution"]
		}
		if !ok {
			t.Fatalf("raw contradiction missing proposed_resolution/resolution object: %#v", contradiction)
		}
		resolution, ok := resolutionValue.(map[string]any)
		if !ok || len(resolution) == 0 {
			t.Fatalf("raw resolution = %#v, want non-empty object", resolutionValue)
		}
	}
}

func TestHygieneCommandSuggestReturnsNoContradictionsForCoherentGraph(t *testing.T) {
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
	if response.Result.Plan.Suggestions.Contradictions == nil {
		t.Fatal("expected contradictions to be an empty array, got nil")
	}
	if len(response.Result.Plan.Suggestions.Contradictions) != 0 {
		t.Fatalf("contradictions = %#v, want empty", response.Result.Plan.Suggestions.Contradictions)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal raw hygiene response: %v", err)
	}
	result, ok := payload["result"].(map[string]any)
	if !ok {
		t.Fatalf("raw result = %#v, want object", payload["result"])
	}
	plan, ok := result["plan"].(map[string]any)
	if !ok {
		t.Fatalf("raw plan = %#v, want object", result["plan"])
	}
	suggestions, ok := plan["suggestions"].(map[string]any)
	if !ok {
		t.Fatalf("raw suggestions = %#v, want object", plan["suggestions"])
	}
	rawContradictions, ok := suggestions["contradictions"].([]any)
	if !ok {
		t.Fatalf("raw contradictions = %#v, want empty array", suggestions["contradictions"])
	}
	if len(rawContradictions) != 0 {
		t.Fatalf("raw contradictions = %#v, want empty array", rawContradictions)
	}
}

func TestHygieneCommandSuggestReturnsStructuredErrorWhenContradictionAnalysisFails(t *testing.T) {
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
	if response.Status != "error" || response.Command != "hygiene" {
		t.Fatalf("response = %#v, want hygiene error", response)
	}
	if response.Error.Category != "operational_error" {
		t.Fatalf("error category = %q, want operational_error", response.Error.Category)
	}
	if response.Error.Code != "workspace_not_initialized" {
		t.Fatalf("error code = %q, want workspace_not_initialized", response.Error.Code)
	}
	if response.Error.Message == "" {
		t.Fatal("expected structured error message")
	}
}

func TestHygieneCommandApplyExecutesExplicitPlanAndReturnsRevisionAnchor(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initHygieneWorkspace(t)
	writeHygieneRevision(t, workspace, noisyGraphPayload)

	store := kuzu.NewStore()
	revisionBeforeApply, err := store.CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision before apply returned error: %v", err)
	}
	if !revisionBeforeApply.Exists {
		t.Fatal("expected an initial revision before hygiene apply")
	}

	suggestCmd := newCommand(repoDir, manager, nil)
	suggestStdout := &bytes.Buffer{}
	suggestCmd.SetOut(suggestStdout)
	suggestCmd.SetErr(&bytes.Buffer{})
	if err := suggestCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("suggest ExecuteContext returned error: %v", err)
	}

	suggestResponse := decodeHygieneSuggestResponse(t, suggestStdout.Bytes())
	plan := suggestResponse.Result.Plan

	var duplicateActionID string
	var orphanActionID string
	for _, action := range plan.Actions {
		switch action.Type {
		case graphhealth.ActionConsolidateDuplicate:
			duplicateActionID = action.ID
		case graphhealth.ActionPruneOrphan:
			if len(action.TargetIDs) == 1 && action.TargetIDs[0] == "note:orphan-ideas" {
				orphanActionID = action.ID
			}
		}
	}
	if duplicateActionID == "" {
		t.Fatalf("expected duplicate action in plan, got %#v", plan.Actions)
	}
	if orphanActionID == "" {
		t.Fatalf("expected orphan action for note:orphan-ideas in plan, got %#v", plan.Actions)
	}

	selectedActionIDs := []string{duplicateActionID, orphanActionID}
	plan.SelectedActionIDs = selectedActionIDs
	planPath := filepath.Join(t.TempDir(), "apply-plan.json")
	planPayload, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("json.Marshal plan: %v", err)
	}
	if err := os.WriteFile(planPath, planPayload, 0o644); err != nil {
		t.Fatalf("WriteFile plan: %v", err)
	}

	applyCmd := newCommand(repoDir, manager, nil)
	applyStdout := &bytes.Buffer{}
	applyCmd.SetOut(applyStdout)
	applyCmd.SetErr(&bytes.Buffer{})
	applyCmd.SetArgs([]string{"--apply", "--file", planPath})
	if err := applyCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("apply ExecuteContext returned error: %v", err)
	}

	var response struct {
		SchemaVersion string `json:"schema_version"`
		Command       string `json:"command"`
		Status        string `json:"status"`
		Result        struct {
			Mode              string                     `json:"mode"`
			Applied           graphhealth.AppliedSummary `json:"applied"`
			SelectedActionIDs []string                   `json:"selected_action_ids"`
			BeforeRevision    kuzu.CurrentRevisionState  `json:"before_revision"`
			Revision          kuzu.RevisionWriteSummary  `json:"revision"`
			SyncSummary       kuzu.GraphSyncSummary      `json:"sync_summary"`
		} `json:"result"`
	}
	if err := json.Unmarshal(applyStdout.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal apply response: %v\npayload: %s", err, applyStdout.Bytes())
	}

	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if response.Command != "hygiene" {
		t.Fatalf("command = %q, want hygiene", response.Command)
	}
	if response.Result.Mode != "apply" {
		t.Fatalf("result.mode = %q, want apply", response.Result.Mode)
	}
	assertNonEmptyHexLikeString(t, "result.revision.anchor", response.Result.Revision.Anchor)
	if response.Result.SyncSummary.Revision.Anchor != response.Result.Revision.Anchor {
		t.Fatalf("sync_summary.revision.anchor = %q, want %q", response.Result.SyncSummary.Revision.Anchor, response.Result.Revision.Anchor)
	}
	if !reflect.DeepEqual(response.Result.SelectedActionIDs, selectedActionIDs) {
		t.Fatalf("selected_action_ids = %#v, want %#v", response.Result.SelectedActionIDs, selectedActionIDs)
	}
	if !response.Result.BeforeRevision.Exists {
		t.Fatal("expected before_revision.exists to be true")
	}
	if response.Result.BeforeRevision.Revision.Anchor != revisionBeforeApply.Revision.Anchor {
		t.Fatalf("before_revision.revision.anchor = %q, want %q", response.Result.BeforeRevision.Revision.Anchor, revisionBeforeApply.Revision.Anchor)
	}
	if response.Result.Applied.TotalActions != len(selectedActionIDs) {
		t.Fatalf("applied.total_actions = %d, want %d", response.Result.Applied.TotalActions, len(selectedActionIDs))
	}
	if response.Result.Applied.ConsolidatedDuplicates != 1 {
		t.Fatalf("applied.consolidated_duplicates = %d, want 1", response.Result.Applied.ConsolidatedDuplicates)
	}
	if response.Result.Applied.PrunedOrphans != 1 {
		t.Fatalf("applied.pruned_orphans = %d, want 1", response.Result.Applied.PrunedOrphans)
	}
	if response.Result.Applied.ResolvedContradictions != 0 {
		t.Fatalf("applied.resolved_contradictions = %d, want 0", response.Result.Applied.ResolvedContradictions)
	}

	revisionAfterApply, err := store.CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision after apply returned error: %v", err)
	}
	if !revisionAfterApply.Exists {
		t.Fatal("expected revision after apply")
	}
	if revisionAfterApply.Revision.Anchor != response.Result.Revision.Anchor {
		t.Fatalf("current revision anchor = %q, want %q", revisionAfterApply.Revision.Anchor, response.Result.Revision.Anchor)
	}
}

func TestHygieneApplyChangesDiffableViaGraphDiff(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initHygieneWorkspace(t)
	initialWrite := writeHygieneRevision(t, workspace, combinedGraphPayload)

	plan := runHygieneSuggestCommand(t, repoDir, manager).Result.Plan
	actionTypes := map[string]struct{}{}
	selectedActionIDs := make([]string, 0, len(plan.Actions))
	for _, action := range plan.Actions {
		actionTypes[action.Type] = struct{}{}
		selectedActionIDs = append(selectedActionIDs, action.ID)
	}
	for _, expectedType := range []string{
		graphhealth.ActionConsolidateDuplicate,
		graphhealth.ActionPruneOrphan,
		graphhealth.ActionResolveContradiction,
	} {
		if _, ok := actionTypes[expectedType]; !ok {
			t.Fatalf("plan action types = %#v, want %q", keys(actionTypes), expectedType)
		}
	}
	plan.SelectedActionIDs = selectedActionIDs

	applyResponse := runHygieneApplyCommand(t, repoDir, manager, plan)
	beforeAnchor := initialWrite.Revision.Anchor
	if applyResponse.Result.BeforeRevision.Exists && strings.TrimSpace(applyResponse.Result.BeforeRevision.Revision.Anchor) != "" {
		beforeAnchor = applyResponse.Result.BeforeRevision.Revision.Anchor
	}
	afterAnchor := applyResponse.Result.Revision.Anchor

	diffResponse := runGraphDiffCommand(t, repoDir, manager, beforeAnchor, afterAnchor)
	if diffResponse.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want v1", diffResponse.SchemaVersion)
	}
	if diffResponse.Command != "diff" {
		t.Fatalf("command = %q, want diff", diffResponse.Command)
	}
	if diffResponse.Status != "ok" {
		t.Fatalf("status = %q, want ok", diffResponse.Status)
	}

	diff := diffResponse.Result.Diff
	if diff.From.Requested != beforeAnchor {
		t.Fatalf("diff.from.requested = %q, want %q", diff.From.Requested, beforeAnchor)
	}
	if diff.To.Requested != afterAnchor {
		t.Fatalf("diff.to.requested = %q, want %q", diff.To.Requested, afterAnchor)
	}
	if diff.Summary.Entities.AddedCount+diff.Summary.Entities.UpdatedCount+diff.Summary.Entities.RemovedCount+diff.Summary.Entities.RetaggedCount+
		diff.Summary.Relationships.AddedCount+diff.Summary.Relationships.UpdatedCount+diff.Summary.Relationships.RemovedCount+diff.Summary.Relationships.RetaggedCount == 0 {
		t.Fatal("expected non-empty diff after hygiene apply")
	}

	removedEntityIDs := map[string]struct{}{}
	for _, entity := range diff.Entities.Removed {
		removedEntityIDs[entity.ID] = struct{}{}
	}
	for _, expectedID := range []string{"note:orphan-idea", "svc:auth-copy"} {
		if _, ok := removedEntityIDs[expectedID]; !ok {
			t.Fatalf("removed entity ids = %#v, want %q to be removed", keys(removedEntityIDs), expectedID)
		}
	}
	if _, removedJWT := removedEntityIDs["adr:use-jwt"]; !removedJWT {
		if _, removedSessions := removedEntityIDs["adr:use-sessions"]; !removedSessions {
			t.Fatalf("removed entity ids = %#v, want one contradictory ADR to be removed", keys(removedEntityIDs))
		}
	}

	foundRemovedContradictionEdge := false
	for _, relationship := range diff.Relationships.Removed {
		if relationship.Kind == "CONTRADICTS" &&
			((relationship.From == "adr:use-jwt" && relationship.To == "adr:use-sessions") ||
				(relationship.From == "adr:use-sessions" && relationship.To == "adr:use-jwt")) {
			foundRemovedContradictionEdge = true
			break
		}
	}
	if !foundRemovedContradictionEdge {
		t.Fatalf("removed relationships = %#v, want contradiction edge between the ADR nodes", diff.Relationships.Removed)
	}
}

func TestHygieneApplyRevisionMetadataContainsBeforeState(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initHygieneWorkspace(t)
	initialWrite := writeHygieneRevision(t, workspace, noisyGraphPayload)

	plan := runHygieneSuggestCommand(t, repoDir, manager).Result.Plan
	var duplicateActionID string
	var orphanActionID string
	for _, action := range plan.Actions {
		switch action.Type {
		case graphhealth.ActionConsolidateDuplicate:
			duplicateActionID = action.ID
		case graphhealth.ActionPruneOrphan:
			if len(action.TargetIDs) == 1 && action.TargetIDs[0] == "note:orphan-ideas" {
				orphanActionID = action.ID
			}
		}
	}
	if duplicateActionID == "" || orphanActionID == "" {
		t.Fatalf("plan.actions = %#v, want duplicate and orphan actions", plan.Actions)
	}
	plan.SelectedActionIDs = []string{duplicateActionID, orphanActionID}

	applyResponse := runHygieneApplyCommand(t, repoDir, manager, plan)
	if applyResponse.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want v1", applyResponse.SchemaVersion)
	}
	if applyResponse.Command != "hygiene" {
		t.Fatalf("command = %q, want hygiene", applyResponse.Command)
	}
	if applyResponse.Status != "ok" {
		t.Fatalf("status = %q, want ok", applyResponse.Status)
	}
	if applyResponse.Result.Mode != "apply" {
		t.Fatalf("result.mode = %q, want apply", applyResponse.Result.Mode)
	}
	assertNonEmptyHexLikeString(t, "result.revision.anchor", applyResponse.Result.Revision.Anchor)
	if applyResponse.Result.Revision.ID == "" {
		t.Fatal("expected result.revision.id to be populated")
	}
	if applyResponse.Result.Revision.Reason == "" || !strings.Contains(applyResponse.Result.Revision.Reason, "Apply hygiene plan") {
		t.Fatalf("result.revision.reason = %q, want apply reason", applyResponse.Result.Revision.Reason)
	}
	if !reflect.DeepEqual(applyResponse.Result.SelectedActionIDs, plan.SelectedActionIDs) {
		t.Fatalf("selected_action_ids = %#v, want %#v", applyResponse.Result.SelectedActionIDs, plan.SelectedActionIDs)
	}
	if applyResponse.Result.Applied.TotalActions != len(plan.SelectedActionIDs) {
		t.Fatalf("applied.total_actions = %d, want %d", applyResponse.Result.Applied.TotalActions, len(plan.SelectedActionIDs))
	}
	if applyResponse.Result.Applied.ConsolidatedDuplicates != 1 {
		t.Fatalf("applied.consolidated_duplicates = %d, want 1", applyResponse.Result.Applied.ConsolidatedDuplicates)
	}
	if applyResponse.Result.Applied.PrunedOrphans != 1 {
		t.Fatalf("applied.pruned_orphans = %d, want 1", applyResponse.Result.Applied.PrunedOrphans)
	}
	if !applyResponse.Result.BeforeRevision.Exists {
		t.Fatal("expected result.before_revision.exists to be true")
	}
	if applyResponse.Result.BeforeRevision.Revision.ID != initialWrite.Revision.ID {
		t.Fatalf("before_revision.revision.id = %q, want %q", applyResponse.Result.BeforeRevision.Revision.ID, initialWrite.Revision.ID)
	}
	if applyResponse.Result.BeforeRevision.Revision.Anchor != initialWrite.Revision.Anchor {
		t.Fatalf("before_revision.revision.anchor = %q, want %q", applyResponse.Result.BeforeRevision.Revision.Anchor, initialWrite.Revision.Anchor)
	}
	if applyResponse.Result.BeforeRevision.Revision.NodeCount != 7 || applyResponse.Result.BeforeRevision.Revision.EdgeCount != 3 {
		t.Fatalf("before_revision counts = (%d, %d), want (7, 3)", applyResponse.Result.BeforeRevision.Revision.NodeCount, applyResponse.Result.BeforeRevision.Revision.EdgeCount)
	}
	if applyResponse.Result.Revision.NodeCount != 5 || applyResponse.Result.Revision.EdgeCount != 2 {
		t.Fatalf("result.revision counts = (%d, %d), want (5, 2)", applyResponse.Result.Revision.NodeCount, applyResponse.Result.Revision.EdgeCount)
	}
	if applyResponse.Result.SyncSummary.Revision.Anchor != applyResponse.Result.Revision.Anchor {
		t.Fatalf("sync_summary.revision.anchor = %q, want %q", applyResponse.Result.SyncSummary.Revision.Anchor, applyResponse.Result.Revision.Anchor)
	}

	raw := decodeHygieneJSONMap(t, applyResponse.Raw)
	result := requireJSONObject(t, raw, "result")
	beforeRevision := requireJSONObject(t, result, "before_revision")
	if exists, ok := beforeRevision["exists"].(bool); !ok || !exists {
		t.Fatalf("raw result.before_revision.exists = %#v, want true", beforeRevision["exists"])
	}
	revision := requireJSONObject(t, result, "revision")
	assertNonEmptyHexLikeString(t, "raw result.revision.anchor", requireJSONString(t, revision, "anchor"))
}

func TestHygieneApplyRevisionDiffReturnsStructuredErrorWhenSnapshotIsUnavailable(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initHygieneWorkspace(t)
	initialWrite := writeHygieneRevision(t, workspace, noisyGraphPayload)

	plan := runHygieneSuggestCommand(t, repoDir, manager).Result.Plan
	var orphanActionID string
	for _, action := range plan.Actions {
		if action.Type == graphhealth.ActionPruneOrphan && len(action.TargetIDs) == 1 && action.TargetIDs[0] == "note:orphan-ideas" {
			orphanActionID = action.ID
			break
		}
	}
	if orphanActionID == "" {
		t.Fatalf("plan.actions = %#v, want orphan prune action", plan.Actions)
	}
	plan.SelectedActionIDs = []string{orphanActionID}

	applyResponse := runHygieneApplyCommand(t, repoDir, manager, plan)
	mutateHygieneGraphState(t, repoDir,
		"MATCH (s:RevisionNodeState) WHERE s.revision_id = '"+applyResponse.Result.Revision.ID+"' AND s.entity_id = 'task:backlog-cleanup' DELETE s;",
	)

	diffCmd := diffcmd.NewCommand(repoDir, manager)
	stdout := &bytes.Buffer{}
	diffCmd.SetOut(stdout)
	diffCmd.SetErr(&bytes.Buffer{})
	diffCmd.SetArgs([]string{"--from", initialWrite.Revision.Anchor, "--to", applyResponse.Result.Revision.Anchor})

	err := diffCmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected diff error, got nil")
	}

	response := decodeHygieneDiffErrorResponse(t, stdout.Bytes())
	if response.SchemaVersion != "v1" {
		t.Fatalf("schema_version = %q, want v1", response.SchemaVersion)
	}
	if response.Command != "diff" {
		t.Fatalf("command = %q, want diff", response.Command)
	}
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Error.Category != "operational_error" {
		t.Fatalf("error.category = %q, want operational_error", response.Error.Category)
	}
	if response.Error.Type != "diff_error" {
		t.Fatalf("error.type = %q, want diff_error", response.Error.Type)
	}
	if response.Error.Code != "revision_snapshot_unavailable" {
		t.Fatalf("error.code = %q, want revision_snapshot_unavailable", response.Error.Code)
	}
	if response.Error.Details["anchor"] != applyResponse.Result.Revision.Anchor {
		t.Fatalf("error.details.anchor = %#v, want %q", response.Error.Details["anchor"], applyResponse.Result.Revision.Anchor)
	}
}

func TestHygieneCommandApplyRejectsWhenNoFileProvided(t *testing.T) {
	t.Parallel()

	repoDir, manager, _ := initHygieneWorkspace(t)

	cmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--apply"})

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	response := decodeHygieneErrorResponse(t, stdout.Bytes())
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Command != "hygiene" {
		t.Fatalf("command = %q, want hygiene", response.Command)
	}
	if response.Error.Category != "validation_error" {
		t.Fatalf("error.category = %q, want validation_error", response.Error.Category)
	}
	if response.Error.Type != "input_error" {
		t.Fatalf("error.type = %q, want input_error", response.Error.Type)
	}
	if response.Error.Code != "missing_hygiene_plan" {
		t.Fatalf("error.code = %q, want missing_hygiene_plan", response.Error.Code)
	}
}

func TestHygieneCommandApplyRejectsWhenNoActionsSelected(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initHygieneWorkspace(t)
	writeHygieneRevision(t, workspace, noisyGraphPayload)

	suggestCmd := newCommand(repoDir, manager, nil)
	suggestStdout := &bytes.Buffer{}
	suggestCmd.SetOut(suggestStdout)
	suggestCmd.SetErr(&bytes.Buffer{})
	if err := suggestCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("suggest ExecuteContext returned error: %v", err)
	}

	plan := decodeHygieneSuggestResponse(t, suggestStdout.Bytes()).Result.Plan
	plan.SelectedActionIDs = []string{}

	planPath := filepath.Join(t.TempDir(), "empty-selection-plan.json")
	planPayload, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("json.Marshal plan: %v", err)
	}
	if err := os.WriteFile(planPath, planPayload, 0o644); err != nil {
		t.Fatalf("WriteFile plan: %v", err)
	}

	applyCmd := newCommand(repoDir, manager, nil)
	applyStdout := &bytes.Buffer{}
	applyCmd.SetOut(applyStdout)
	applyCmd.SetErr(&bytes.Buffer{})
	applyCmd.SetArgs([]string{"--apply", "--file", planPath})

	err = applyCmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	response := decodeHygieneErrorResponse(t, applyStdout.Bytes())
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Command != "hygiene" {
		t.Fatalf("command = %q, want hygiene", response.Command)
	}
	if response.Error.Category != "validation_error" {
		t.Fatalf("error.category = %q, want validation_error", response.Error.Category)
	}
	if response.Error.Type != "input_error" {
		t.Fatalf("error.type = %q, want input_error", response.Error.Type)
	}
	if response.Error.Code != "no_selected_hygiene_actions" {
		t.Fatalf("error.code = %q, want no_selected_hygiene_actions", response.Error.Code)
	}
}

func TestHygieneCommandApplyRejectsStalePlan(t *testing.T) {
	repoDir, manager, workspace := initHygieneWorkspace(t)
	writeHygieneRevision(t, workspace, noisyGraphPayload)

	stalePlan := runHygieneSuggestCommand(t, repoDir, manager).Result.Plan
	if len(stalePlan.Actions) == 0 {
		t.Fatalf("plan.actions = %#v, want at least one action", stalePlan.Actions)
	}
	stalePlan.SelectedActionIDs = []string{stalePlan.Actions[0].ID}
	mutateHygieneGraphState(t, repoDir,
		"MATCH (e:Entity {id: 'doc:graph-stats-a'}) SET e.summary = 'Weekly graph metrics overview (mutated after suggest)';",
	)

	store := kuzu.NewStore()
	graphBefore, err := store.ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph before stale apply returned error: %v", err)
	}
	revisionBefore, err := store.CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision before stale apply returned error: %v", err)
	}
	revisionCountBefore := countHygieneRevisionNodes(t, workspace)

	response := runHygieneApplyCommandExpectError(t, repoDir, manager, stalePlan)
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Command != "hygiene" {
		t.Fatalf("command = %q, want hygiene", response.Command)
	}
	if response.Error.Category != "validation_error" {
		t.Fatalf("error.category = %q, want validation_error", response.Error.Category)
	}
	if response.Error.Type != "input_error" {
		t.Fatalf("error.type = %q, want input_error", response.Error.Type)
	}
	if response.Error.Code != "stale_hygiene_plan" {
		t.Fatalf("error.code = %q, want stale_hygiene_plan", response.Error.Code)
	}

	graphAfter, err := store.ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph after stale apply returned error: %v", err)
	}
	if !graphsEqual(graphBefore, graphAfter) {
		t.Fatalf("graph changed after stale hygiene apply rejection\nbefore: %#v\nafter: %#v", graphBefore, graphAfter)
	}

	revisionAfter, err := store.CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision after stale apply returned error: %v", err)
	}
	if !reflect.DeepEqual(revisionBefore, revisionAfter) {
		t.Fatalf("revision changed after stale hygiene apply rejection\nbefore: %#v\nafter: %#v", revisionBefore, revisionAfter)
	}

	revisionCountAfter := countHygieneRevisionNodes(t, workspace)
	if revisionCountAfter != revisionCountBefore {
		t.Fatalf("revision count changed after stale hygiene apply rejection\nbefore: %d\nafter: %d", revisionCountBefore, revisionCountAfter)
	}
}

func TestHygieneCommandApplyRejectsUnsupportedActionType(t *testing.T) {
	repoDir, manager, workspace := initHygieneWorkspace(t)
	writeHygieneRevision(t, workspace, noisyGraphPayload)

	validPlan := runHygieneSuggestCommand(t, repoDir, manager).Result.Plan
	unsupportedPlan := graphhealth.HygienePlan{
		SnapshotAnchor: validPlan.SnapshotAnchor,
		Snapshot:       validPlan.Snapshot,
		Suggestions:    validPlan.Suggestions,
		Actions: []graphhealth.HygieneAction{
			{
				ID:          "bad:action",
				Type:        "delete_everything",
				TargetIDs:   []string{"doc:graph-stats-a"},
				Explanation: "bad action",
			},
		},
		SelectedActionIDs: []string{"bad:action"},
	}

	store := kuzu.NewStore()
	graphBefore, err := store.ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph before unsupported apply returned error: %v", err)
	}
	revisionCountBefore := countHygieneRevisionNodes(t, workspace)

	response := runHygieneApplyCommandExpectError(t, repoDir, manager, unsupportedPlan)
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Command != "hygiene" {
		t.Fatalf("command = %q, want hygiene", response.Command)
	}
	if response.Error.Category != "validation_error" {
		t.Fatalf("error.category = %q, want validation_error", response.Error.Category)
	}
	if response.Error.Type != "input_error" {
		t.Fatalf("error.type = %q, want input_error", response.Error.Type)
	}
	if response.Error.Code != "unsupported_hygiene_action" {
		t.Fatalf("error.code = %q, want unsupported_hygiene_action", response.Error.Code)
	}

	graphAfter, err := store.ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph after unsupported apply returned error: %v", err)
	}
	if !graphsEqual(graphBefore, graphAfter) {
		t.Fatalf("graph changed after unsupported hygiene apply rejection\nbefore: %#v\nafter: %#v", graphBefore, graphAfter)
	}

	revisionCountAfter := countHygieneRevisionNodes(t, workspace)
	if revisionCountAfter != revisionCountBefore {
		t.Fatalf("revision count changed after unsupported hygiene apply rejection\nbefore: %d\nafter: %d", revisionCountBefore, revisionCountAfter)
	}
}

func TestHygieneCommandApplyRejectsMissingTargetEntity(t *testing.T) {
	repoDir, manager, workspace := initHygieneWorkspace(t)
	writeHygieneRevision(t, workspace, noisyGraphPayload)

	validPlan := runHygieneSuggestCommand(t, repoDir, manager).Result.Plan
	missingTargetPlan := graphhealth.HygienePlan{
		SnapshotAnchor: validPlan.SnapshotAnchor,
		Snapshot:       validPlan.Snapshot,
		Suggestions:    validPlan.Suggestions,
		Actions: []graphhealth.HygieneAction{
			{
				ID:          "missing:node",
				Type:        graphhealth.ActionPruneOrphan,
				TargetIDs:   []string{"node:missing"},
				Explanation: "missing target",
			},
		},
		SelectedActionIDs: []string{"missing:node"},
	}

	store := kuzu.NewStore()
	graphBefore, err := store.ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph before missing-target apply returned error: %v", err)
	}
	revisionCountBefore := countHygieneRevisionNodes(t, workspace)

	response := runHygieneApplyCommandExpectError(t, repoDir, manager, missingTargetPlan)
	if response.Status != "error" {
		t.Fatalf("status = %q, want error", response.Status)
	}
	if response.Command != "hygiene" {
		t.Fatalf("command = %q, want hygiene", response.Command)
	}
	if response.Error.Category != "validation_error" {
		t.Fatalf("error.category = %q, want validation_error", response.Error.Category)
	}
	if response.Error.Type != "input_error" {
		t.Fatalf("error.type = %q, want input_error", response.Error.Type)
	}
	if response.Error.Code != "unsafe_hygiene_plan" {
		t.Fatalf("error.code = %q, want unsafe_hygiene_plan", response.Error.Code)
	}
	reason, _ := response.Error.Details["reason"].(string)
	if !strings.Contains(reason, "references missing node") {
		t.Fatalf("error.details.reason = %#v, want missing node detail", response.Error.Details["reason"])
	}

	graphAfter, err := store.ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph after missing-target apply returned error: %v", err)
	}
	if !graphsEqual(graphBefore, graphAfter) {
		t.Fatalf("graph changed after missing-target hygiene apply rejection\nbefore: %#v\nafter: %#v", graphBefore, graphAfter)
	}

	revisionCountAfter := countHygieneRevisionNodes(t, workspace)
	if revisionCountAfter != revisionCountBefore {
		t.Fatalf("revision count changed after missing-target hygiene apply rejection\nbefore: %d\nafter: %d", revisionCountBefore, revisionCountAfter)
	}
}

func TestHygieneCommandApplyPreservesGraphStateAfterRejection(t *testing.T) {
	repoDir, manager, workspace := initHygieneWorkspace(t)
	writeHygieneRevision(t, workspace, noisyGraphPayload)

	validPlan := runHygieneSuggestCommand(t, repoDir, manager).Result.Plan
	rejectedPlan := graphhealth.HygienePlan{
		SnapshotAnchor: validPlan.SnapshotAnchor,
		Snapshot:       validPlan.Snapshot,
		Suggestions:    validPlan.Suggestions,
		Actions: []graphhealth.HygieneAction{
			{
				ID:          "bad:action",
				Type:        "delete_everything",
				TargetIDs:   []string{"doc:graph-stats-a"},
				Explanation: "bad action",
			},
		},
		SelectedActionIDs: []string{"bad:action"},
	}

	store := kuzu.NewStore()
	graphBefore, err := store.ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph before rejected apply returned error: %v", err)
	}
	revisionBefore, err := store.CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision before rejected apply returned error: %v", err)
	}
	revisionCountBefore := countHygieneRevisionNodes(t, workspace)

	response := runHygieneApplyCommandExpectError(t, repoDir, manager, rejectedPlan)
	if response.Error.Code != "unsupported_hygiene_action" {
		t.Fatalf("error.code = %q, want unsupported_hygiene_action", response.Error.Code)
	}

	graphAfter, err := store.ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph after rejected apply returned error: %v", err)
	}
	if !graphsEqual(graphBefore, graphAfter) {
		t.Fatalf("graph changed after rejected hygiene apply\nbefore: %#v\nafter: %#v", graphBefore, graphAfter)
	}

	revisionAfter, err := store.CurrentRevision(context.Background(), workspace)
	if err != nil {
		t.Fatalf("CurrentRevision after rejected apply returned error: %v", err)
	}
	if !reflect.DeepEqual(revisionBefore, revisionAfter) {
		t.Fatalf("revision changed after rejected hygiene apply\nbefore: %#v\nafter: %#v", revisionBefore, revisionAfter)
	}

	revisionCountAfter := countHygieneRevisionNodes(t, workspace)
	if revisionCountAfter != revisionCountBefore {
		t.Fatalf("revision count changed after rejected hygiene apply\nbefore: %d\nafter: %d", revisionCountBefore, revisionCountAfter)
	}
}

func TestHygieneCommandApplyOnlySelectedSubset(t *testing.T) {
	t.Parallel()

	repoDir, manager, workspace := initHygieneWorkspace(t)
	writeHygieneRevision(t, workspace, noisyGraphPayload)

	suggestCmd := newCommand(repoDir, manager, nil)
	suggestStdout := &bytes.Buffer{}
	suggestCmd.SetOut(suggestStdout)
	suggestCmd.SetErr(&bytes.Buffer{})
	if err := suggestCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("suggest ExecuteContext returned error: %v", err)
	}

	plan := decodeHygieneSuggestResponse(t, suggestStdout.Bytes()).Result.Plan
	var orphanActionID string
	for _, action := range plan.Actions {
		if action.Type == graphhealth.ActionPruneOrphan && len(action.TargetIDs) == 1 && action.TargetIDs[0] == "note:orphan-ideas" {
			orphanActionID = action.ID
			break
		}
	}
	if orphanActionID == "" {
		t.Fatalf("expected orphan action for note:orphan-ideas in plan, got %#v", plan.Actions)
	}
	plan.SelectedActionIDs = []string{orphanActionID}

	planPath := filepath.Join(t.TempDir(), "orphan-only-plan.json")
	planPayload, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("json.Marshal plan: %v", err)
	}
	if err := os.WriteFile(planPath, planPayload, 0o644); err != nil {
		t.Fatalf("WriteFile plan: %v", err)
	}

	applyCmd := newCommand(repoDir, manager, nil)
	applyStdout := &bytes.Buffer{}
	applyCmd.SetOut(applyStdout)
	applyCmd.SetErr(&bytes.Buffer{})
	applyCmd.SetArgs([]string{"--apply", "--file", planPath})
	if err := applyCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("apply ExecuteContext returned error: %v", err)
	}

	var response struct {
		Status  string `json:"status"`
		Command string `json:"command"`
		Result  struct {
			Mode              string                     `json:"mode"`
			Applied           graphhealth.AppliedSummary `json:"applied"`
			SelectedActionIDs []string                   `json:"selected_action_ids"`
			SyncSummary       kuzu.GraphSyncSummary      `json:"sync_summary"`
		} `json:"result"`
	}
	if err := json.Unmarshal(applyStdout.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal apply response: %v\npayload: %s", err, applyStdout.Bytes())
	}
	if response.Status != "ok" {
		t.Fatalf("status = %q, want ok", response.Status)
	}
	if response.Command != "hygiene" {
		t.Fatalf("command = %q, want hygiene", response.Command)
	}
	if response.Result.Mode != "apply" {
		t.Fatalf("result.mode = %q, want apply", response.Result.Mode)
	}
	if !reflect.DeepEqual(response.Result.SelectedActionIDs, []string{orphanActionID}) {
		t.Fatalf("selected_action_ids = %#v, want %#v", response.Result.SelectedActionIDs, []string{orphanActionID})
	}
	if response.Result.Applied.TotalActions != 1 {
		t.Fatalf("applied.total_actions = %d, want 1", response.Result.Applied.TotalActions)
	}
	if response.Result.Applied.PrunedOrphans != 1 {
		t.Fatalf("applied.pruned_orphans = %d, want 1", response.Result.Applied.PrunedOrphans)
	}
	if response.Result.Applied.ConsolidatedDuplicates != 0 {
		t.Fatalf("applied.consolidated_duplicates = %d, want 0", response.Result.Applied.ConsolidatedDuplicates)
	}
	if response.Result.SyncSummary.Nodes.RemovedCount != 1 {
		t.Fatalf("sync_summary.nodes.removed_count = %d, want 1", response.Result.SyncSummary.Nodes.RemovedCount)
	}
	if len(response.Result.SyncSummary.Nodes.Removed) != 1 || response.Result.SyncSummary.Nodes.Removed[0].ID != "note:orphan-ideas" {
		t.Fatalf("sync_summary.nodes.removed = %#v, want only note:orphan-ideas", response.Result.SyncSummary.Nodes.Removed)
	}

	graphAfterApply, err := kuzu.NewStore().ReadGraph(context.Background(), workspace)
	if err != nil {
		t.Fatalf("ReadGraph after apply returned error: %v", err)
	}
	nodeIDs := map[string]struct{}{}
	for _, node := range graphAfterApply.Nodes {
		nodeIDs[node.ID] = struct{}{}
	}
	if _, ok := nodeIDs["note:orphan-ideas"]; ok {
		t.Fatal("expected note:orphan-ideas to be pruned")
	}
	if _, ok := nodeIDs["task:backlog-cleanup"]; !ok {
		t.Fatal("expected unselected orphan task:backlog-cleanup to remain")
	}
	if _, ok := nodeIDs["doc:graph-stats-a"]; !ok {
		t.Fatal("expected doc:graph-stats-a to remain because duplicate action was not selected")
	}
	if _, ok := nodeIDs["doc:graph-stats-b"]; !ok {
		t.Fatal("expected doc:graph-stats-b to remain because duplicate action was not selected")
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

type hygieneApplyResponse struct {
	SchemaVersion string `json:"schema_version"`
	Command       string `json:"command"`
	Status        string `json:"status"`
	Result        struct {
		Mode              string                     `json:"mode"`
		Applied           graphhealth.AppliedSummary `json:"applied"`
		SelectedActionIDs []string                   `json:"selected_action_ids"`
		BeforeRevision    kuzu.CurrentRevisionState  `json:"before_revision"`
		Revision          kuzu.RevisionWriteSummary  `json:"revision"`
		SyncSummary       kuzu.GraphSyncSummary      `json:"sync_summary"`
	} `json:"result"`
	Raw []byte `json:"-"`
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

type hygieneDiffSuccessResponse struct {
	SchemaVersion string `json:"schema_version"`
	Command       string `json:"command"`
	Status        string `json:"status"`
	Result        struct {
		Diff kuzu.GraphDiff `json:"diff"`
	} `json:"result"`
}

type hygieneDiffErrorResponse struct {
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

func decodeHygieneApplyResponse(t *testing.T, payload []byte) hygieneApplyResponse {
	t.Helper()

	var response hygieneApplyResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal hygiene apply response: %v\npayload: %s", err, payload)
	}
	response.Raw = append([]byte(nil), payload...)
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

func decodeHygieneDiffSuccessResponse(t *testing.T, payload []byte) hygieneDiffSuccessResponse {
	t.Helper()

	var response hygieneDiffSuccessResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal diff success response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeHygieneDiffErrorResponse(t *testing.T, payload []byte) hygieneDiffErrorResponse {
	t.Helper()

	var response hygieneDiffErrorResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal diff error response: %v\npayload: %s", err, payload)
	}
	return response
}

func decodeHygieneJSONMap(t *testing.T, payload []byte) map[string]any {
	t.Helper()

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json.Unmarshal raw hygiene response: %v\npayload: %s", err, payload)
	}
	return decoded
}

func runHygieneSuggestCommand(t *testing.T, repoDir string, manager *repo.Manager) hygieneSuggestResponse {
	t.Helper()

	cmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("hygiene suggest ExecuteContext returned error: %v", err)
	}
	return decodeHygieneSuggestResponse(t, stdout.Bytes())
}

func runHygieneApplyCommand(t *testing.T, repoDir string, manager *repo.Manager, plan graphhealth.HygienePlan) hygieneApplyResponse {
	t.Helper()

	planPath := filepath.Join(t.TempDir(), "hygiene-apply-plan.json")
	payload, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("json.Marshal plan: %v", err)
	}
	if err := os.WriteFile(planPath, payload, 0o644); err != nil {
		t.Fatalf("WriteFile plan: %v", err)
	}

	cmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--apply", "--file", planPath})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("hygiene apply ExecuteContext returned error: %v", err)
	}
	return decodeHygieneApplyResponse(t, stdout.Bytes())
}

func runHygieneApplyCommandExpectError(t *testing.T, repoDir string, manager *repo.Manager, plan graphhealth.HygienePlan) hygieneErrorResponse {
	t.Helper()

	planPath := filepath.Join(t.TempDir(), "hygiene-apply-plan-error.json")
	payload, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("json.Marshal plan: %v", err)
	}
	if err := os.WriteFile(planPath, payload, 0o644); err != nil {
		t.Fatalf("WriteFile plan: %v", err)
	}

	cmd := newCommand(repoDir, manager, nil)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--apply", "--file", planPath})
	err = cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected hygiene apply error, got nil")
	}
	return decodeHygieneErrorResponse(t, stdout.Bytes())
}

func runGraphDiffCommand(t *testing.T, repoDir string, manager *repo.Manager, from, to string) hygieneDiffSuccessResponse {
	t.Helper()

	cmd := diffcmd.NewCommand(repoDir, manager)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--from", from, "--to", to})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("graph diff ExecuteContext returned error: %v", err)
	}
	return decodeHygieneDiffSuccessResponse(t, stdout.Bytes())
}

func mutateHygieneGraphState(t *testing.T, repoDir string, queries ...string) {
	t.Helper()

	dbPath := filepath.Join(repoDir, repo.WorkspaceDirName, "kuzu", kuzu.StoreFileName)
	db, err := kuzudb.OpenDatabase(dbPath, kuzudb.DefaultSystemConfig())
	if err != nil {
		t.Fatalf("open writable kuzu database: %v", err)
	}
	defer db.Close()

	conn, err := kuzudb.OpenConnection(db)
	if err != nil {
		t.Fatalf("open writable kuzu connection: %v", err)
	}
	defer conn.Close()

	for _, query := range queries {
		result, err := conn.Query(query)
		if result != nil {
			result.Close()
		}
		if err != nil {
			t.Fatalf("execute mutation query %q: %v", query, err)
		}
	}
}

func countHygieneRevisionNodes(t *testing.T, workspace repo.Workspace) int {
	t.Helper()

	dbPath := filepath.Join(workspace.WorkspacePath, "kuzu", kuzu.StoreFileName)
	config := kuzudb.DefaultSystemConfig()
	config.ReadOnly = true

	db, err := kuzudb.OpenDatabase(dbPath, config)
	if err != nil {
		t.Fatalf("open read-only kuzu database: %v", err)
	}
	defer db.Close()

	conn, err := kuzudb.OpenConnection(db)
	if err != nil {
		t.Fatalf("open read-only kuzu connection: %v", err)
	}
	defer conn.Close()

	result, err := conn.Query(`MATCH (e:Entity) WHERE e.kind = 'GraphRevision' RETURN COUNT(e);`)
	if err != nil {
		t.Fatalf("query graph revision count: %v", err)
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

	switch value := values[0].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case int32:
		return int(value)
	case uint64:
		return int(value)
	case uint32:
		return int(value)
	default:
		t.Fatalf("graph revision count = %#v, want integer", values[0])
		return 0
	}
}

func graphsEqual(left, right kuzu.Graph) bool {
	leftPayload, leftErr := json.Marshal(left)
	rightPayload, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftPayload, rightPayload)
}

func requireJSONObject(t *testing.T, parent map[string]any, key string) map[string]any {
	t.Helper()

	value, ok := parent[key]
	if !ok {
		t.Fatalf("missing key %q in %#v", key, parent)
	}
	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s = %#v, want object", key, value)
	}
	return object
}

func requireJSONArray(t *testing.T, parent map[string]any, key string) []any {
	t.Helper()

	value, ok := parent[key]
	if !ok {
		t.Fatalf("missing key %q in %#v", key, parent)
	}
	array, ok := value.([]any)
	if !ok {
		t.Fatalf("%s = %#v, want array", key, value)
	}
	return array
}

func requireJSONString(t *testing.T, parent map[string]any, key string) string {
	t.Helper()

	value, ok := parent[key]
	if !ok {
		t.Fatalf("missing key %q in %#v", key, parent)
	}
	text, ok := value.(string)
	if !ok {
		t.Fatalf("%s = %#v, want string", key, value)
	}
	return text
}

func requireStringArray(t *testing.T, parent map[string]any, key string) []string {
	t.Helper()

	values := requireJSONArray(t, parent, key)
	result := make([]string, 0, len(values))
	for index, value := range values {
		text, ok := value.(string)
		if !ok {
			t.Fatalf("%s[%d] = %#v, want string", key, index, value)
		}
		if text == "" {
			t.Fatalf("%s[%d] = %q, want non-empty string", key, index, text)
		}
		result = append(result, text)
	}
	return result
}

func assertNonEmptyHexLikeString(t *testing.T, label string, value string) {
	t.Helper()

	if value == "" {
		t.Fatalf("%s = %q, want non-empty string", label, value)
	}
	if _, err := hex.DecodeString(value); err != nil {
		t.Fatalf("%s = %q, want hex-like string: %v", label, value, err)
	}
}

func keys[V any](input map[string]V) []string {
	result := make([]string, 0, len(input))
	for key := range input {
		result = append(result, key)
	}
	return result
}

const combinedGraphPayload = `{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-plan-test",
    "timestamp": "2026-03-22T12:00:00Z",
    "revision": {
      "reason": "Seed plan envelope test"
    }
  },
  "nodes": [
    {
      "id": "svc:auth",
      "kind": "Service",
      "title": "Authentication Service",
      "summary": "Handles auth"
    },
    {
      "id": "svc:auth-copy",
      "kind": "Service",
      "title": "Authentication Service",
      "summary": "Handles auth"
    },
    {
      "id": "adr:use-jwt",
      "kind": "ADR",
      "title": "Authentication strategy",
      "summary": "Use JWT tokens for all auth"
    },
    {
      "id": "adr:use-sessions",
      "kind": "ADR",
      "title": "Authentication strategy",
      "summary": "Use session cookies for all auth"
    },
    {
      "id": "note:orphan-idea",
      "kind": "Note",
      "title": "Random thought",
      "summary": "Unconnected idea"
    }
  ],
  "edges": [
    {
      "from": "svc:auth",
      "to": "adr:use-jwt",
      "kind": "DEPENDS_ON"
    },
    {
      "from": "adr:use-jwt",
      "to": "adr:use-sessions",
      "kind": "CONTRADICTS"
    }
  ]
}`

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
