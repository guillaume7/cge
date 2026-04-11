package attribution

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/guillaume-galp/cge/internal/infra/repo"
)

func TestRecorder_Record_CreatesFileUnderAttributionDir(t *testing.T) {
	t.Parallel()

	workspace := initAttributionWorkspace(t)
	recorder := NewRecorder()
	recorder.NowForTest(func() time.Time {
		return time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	})

	rec, err := recorder.Record(workspace, Record{
		Outcome:             "write",
		Task:                "implement auth module",
		SessionID:           "sess-test",
		AggregateConfidence: 0.88,
	})

	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if rec.ID == "" {
		t.Fatal("expected non-empty ID")
	}

	dir := filepath.Join(workspace.WorkspacePath, attributionDirName)
	path := filepath.Join(dir, rec.ID+".json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("attribution file not found at %s: %v", path, err)
	}
}

func TestRecorder_LoadRecord_RoundTripsAllFields(t *testing.T) {
	t.Parallel()

	workspace := initAttributionWorkspace(t)
	recorder := NewRecorder()
	recorder.NowForTest(func() time.Time {
		return time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	})

	wt := 0.80
	dt := 0.60
	md := MemoryDecisionDeferred
	written, err := recorder.Record(workspace, Record{
		Outcome:              "abstain",
		Task:                 "add caching layer",
		SessionID:            "sess-42",
		AggregateConfidence:  0.72,
		WriteThreshold:       &wt,
		DeferThreshold:       &dt,
		MemoryDecision:       &md,
		MemoryDecisionReason: "composite below write threshold; may retry",
	})
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}

	loaded, err := LoadRecord(workspace, written.ID)
	if err != nil {
		t.Fatalf("LoadRecord returned error: %v", err)
	}

	if loaded.Outcome != "abstain" {
		t.Errorf("Outcome = %q, want abstain", loaded.Outcome)
	}
	if loaded.Task != "add caching layer" {
		t.Errorf("Task = %q, want add caching layer", loaded.Task)
	}
	if loaded.MemoryDecision == nil || *loaded.MemoryDecision != MemoryDecisionDeferred {
		t.Errorf("MemoryDecision = %v, want deferred", loaded.MemoryDecision)
	}
	if loaded.MemoryDecisionReason == "" {
		t.Error("MemoryDecisionReason should be non-empty")
	}
	if loaded.WriteThreshold == nil || *loaded.WriteThreshold != 0.80 {
		t.Errorf("WriteThreshold = %v, want 0.80", loaded.WriteThreshold)
	}
	if loaded.DeferThreshold == nil || *loaded.DeferThreshold != 0.60 {
		t.Errorf("DeferThreshold = %v, want 0.60", loaded.DeferThreshold)
	}
	if loaded.InlineSummary.Outcome != "abstain" {
		t.Errorf("InlineSummary.Outcome = %q, want abstain", loaded.InlineSummary.Outcome)
	}
}

func TestRecorder_LoadRecord_ReturnsErrorForMissingRecord(t *testing.T) {
	t.Parallel()

	workspace := initAttributionWorkspace(t)

	_, err := LoadRecord(workspace, "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for missing record, got nil")
	}
}

func TestListRecords_ReturnsAllRecordsWhenNoFilter(t *testing.T) {
	t.Parallel()

	workspace := initAttributionWorkspace(t)
	recorder := NewRecorder()
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	recorder.NowForTest(func() time.Time { return now })

	for i, outcome := range []string{"write", "abstain", "abstain"} {
		_ = i
		if _, err := recorder.Record(workspace, Record{
			Outcome:             outcome,
			Task:                "task " + outcome,
			AggregateConfidence: 0.50,
		}); err != nil {
			t.Fatalf("Record returned error: %v", err)
		}
		now = now.Add(time.Second)
		recorder.NowForTest(func() time.Time { return now })
	}

	all, err := ListRecords(workspace, ListFilter{})
	if err != nil {
		t.Fatalf("ListRecords returned error: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("ListRecords = %d records, want 3", len(all))
	}
}

func TestListRecords_FiltersByOutcome(t *testing.T) {
	t.Parallel()

	workspace := initAttributionWorkspace(t)
	recorder := NewRecorder()
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	recorder.NowForTest(func() time.Time { return now })

	for _, outcome := range []string{"write", "abstain", "write"} {
		if _, err := recorder.Record(workspace, Record{
			Outcome: outcome,
			Task:    "task",
		}); err != nil {
			t.Fatalf("Record returned error: %v", err)
		}
		now = now.Add(time.Second)
		recorder.NowForTest(func() time.Time { return now })
	}

	writeRecords, err := ListRecords(workspace, ListFilter{Outcome: "write"})
	if err != nil {
		t.Fatalf("ListRecords returned error: %v", err)
	}
	if len(writeRecords) != 2 {
		t.Fatalf("ListRecords(write) = %d records, want 2", len(writeRecords))
	}
	for _, rec := range writeRecords {
		if rec.Outcome != "write" {
			t.Errorf("unexpected outcome %q in filtered results", rec.Outcome)
		}
	}
}

func TestListRecords_ReturnsEmptySliceForMissingDirectory(t *testing.T) {
	t.Parallel()

	workspace := initAttributionWorkspace(t)

	records, err := ListRecords(workspace, ListFilter{})
	if err != nil {
		t.Fatalf("ListRecords on empty workspace returned error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected 0 records, got %d", len(records))
	}
}

func TestRecorder_Record_InlineSummary_ReflectsFields(t *testing.T) {
	t.Parallel()

	workspace := initAttributionWorkspace(t)
	recorder := NewRecorder()
	md := MemoryDecisionApproved

	rec, err := recorder.Record(workspace, Record{
		Outcome:             "write",
		Task:                "refactor api layer",
		AggregateConfidence: 0.92,
		MemoryDecision:      &md,
		CandidateFates: []CandidateFate{
			{CandidateID: "c1", Fate: "survived", Composite: 0.92},
			{CandidateID: "c2", Fate: "trimmed", Composite: 0.65},
		},
	})
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}

	if rec.InlineSummary.Outcome != "write" {
		t.Errorf("InlineSummary.Outcome = %q, want write", rec.InlineSummary.Outcome)
	}
	if rec.InlineSummary.Confidence != 0.92 {
		t.Errorf("InlineSummary.Confidence = %.2f, want 0.92", rec.InlineSummary.Confidence)
	}
	if rec.InlineSummary.CandidateCount != 2 {
		t.Errorf("InlineSummary.CandidateCount = %d, want 2", rec.InlineSummary.CandidateCount)
	}
}

// helpers

func initAttributionWorkspace(t *testing.T) repo.Workspace {
	t.Helper()

	repoDir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	manager := repo.NewManager(repo.NewGitRepositoryLocator())
	if _, err := manager.InitWorkspace(context.Background(), repoDir); err != nil {
		t.Fatalf("InitWorkspace: %v", err)
	}
	workspace, err := manager.OpenWorkspace(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("OpenWorkspace: %v", err)
	}
	return workspace
}
