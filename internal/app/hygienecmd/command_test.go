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
	"testing"

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

func decodeHygieneSuggestResponse(t *testing.T, payload []byte) hygieneSuggestResponse {
	t.Helper()

	var response hygieneSuggestResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal hygiene suggest response: %v\npayload: %s", err, payload)
	}
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

func decodeHygieneJSONMap(t *testing.T, payload []byte) map[string]any {
	t.Helper()

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json.Unmarshal raw hygiene response: %v\npayload: %s", err, payload)
	}
	return decoded
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
