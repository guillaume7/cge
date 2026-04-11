package attributionrecorder_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/guillaume-galp/cge/internal/app/attributionrecorder"
	"github.com/guillaume-galp/cge/internal/app/contextevaluator"
	"github.com/guillaume-galp/cge/internal/app/decisionengine"
)

// ---- helpers ----

func fixedNow() time.Time {
	return time.Date(2026, 4, 1, 12, 0, 0, 100_000_000, time.UTC)
}

func newRecorder() attributionrecorder.Recorder {
	return attributionrecorder.NewWithClock(fixedNow)
}

func makeEnvelope(outcome decisionengine.Outcome, confidence float64, scores []contextevaluator.CandidateScore) decisionengine.DecisionEnvelope {
	return decisionengine.DecisionEnvelope{
		Outcome:    outcome,
		Confidence: confidence,
		Scores:     scores,
	}
}

func threeScores(fates []string) []contextevaluator.CandidateScore {
	result := make([]contextevaluator.CandidateScore, len(fates))
	for i, fate := range fates {
		result[i] = contextevaluator.CandidateScore{
			CandidateID: fmt.Sprintf("c%d", i+1),
			Scores: contextevaluator.DimensionScores{
				Relevance:   0.8,
				Consistency: 0.9,
				Usefulness:  0.7,
			},
			Composite: 0.8,
			Fate:      fate,
		}
	}
	return result
}

// ---- US1 Tests ----

// Scenario: Attribution record for a continue decision
func TestGenerate_Continue_ThreeSurvivedCandidates(t *testing.T) {
	scores := []contextevaluator.CandidateScore{
		{CandidateID: "c1", Scores: contextevaluator.DimensionScores{Relevance: 0.9, Consistency: 0.85, Usefulness: 0.80}, Composite: 0.86, Fate: "survived"},
		{CandidateID: "c2", Scores: contextevaluator.DimensionScores{Relevance: 0.75, Consistency: 0.80, Usefulness: 0.70}, Composite: 0.75, Fate: "survived"},
		{CandidateID: "c3", Scores: contextevaluator.DimensionScores{Relevance: 0.80, Consistency: 0.78, Usefulness: 0.75}, Composite: 0.78, Fate: "survived"},
	}
	envelope := makeEnvelope(decisionengine.OutcomeContinue, 0.80, scores)
	rec := newRecorder().Generate(envelope, "implement auth module", "session-001")

	if rec.Outcome != "continue" {
		t.Errorf("outcome = %q, want %q", rec.Outcome, "continue")
	}
	if len(rec.CandidateFates) != 3 {
		t.Fatalf("candidate_fates len = %d, want 3", len(rec.CandidateFates))
	}
	for _, fate := range rec.CandidateFates {
		if fate.Fate != "survived" {
			t.Errorf("fate %s = %q, want survived", fate.CandidateID, fate.Fate)
		}
	}
	if rec.CompositeConfidence != 0.80 {
		t.Errorf("composite_confidence = %.2f, want 0.80", rec.CompositeConfidence)
	}
}

// Scenario: Attribution record for a minimal decision with trimmed candidates
func TestGenerate_Minimal_TrimmedCandidates(t *testing.T) {
	scores := []contextevaluator.CandidateScore{
		{CandidateID: "c1", Scores: contextevaluator.DimensionScores{Relevance: 0.7, Consistency: 0.7, Usefulness: 0.7}, Composite: 0.70, Fate: "survived"},
		{CandidateID: "c2", Scores: contextevaluator.DimensionScores{Relevance: 0.7, Consistency: 0.7, Usefulness: 0.7}, Composite: 0.70, Fate: "survived"},
		{CandidateID: "c3", Scores: contextevaluator.DimensionScores{Relevance: 0.4, Consistency: 0.4, Usefulness: 0.4}, Composite: 0.40, Fate: "trimmed", RejectionReason: "below survive threshold"},
		{CandidateID: "c4", Scores: contextevaluator.DimensionScores{Relevance: 0.3, Consistency: 0.3, Usefulness: 0.3}, Composite: 0.30, Fate: "trimmed", RejectionReason: "below survive threshold"},
	}
	envelope := makeEnvelope(decisionengine.OutcomeMinimal, 0.55, scores)
	rec := newRecorder().Generate(envelope, "fix bug in login", "")

	if rec.Outcome != "minimal" {
		t.Errorf("outcome = %q, want minimal", rec.Outcome)
	}

	var survived, trimmed int
	for _, fate := range rec.CandidateFates {
		switch fate.Fate {
		case "survived":
			survived++
		case "trimmed":
			trimmed++
			if fate.RejectionReason == "" {
				t.Errorf("trimmed candidate %s missing rejection reason", fate.CandidateID)
			}
		}
	}
	if survived != 2 {
		t.Errorf("survived = %d, want 2", survived)
	}
	if trimmed != 2 {
		t.Errorf("trimmed = %d, want 2", trimmed)
	}
}

// Scenario: Attribution record for an abstain decision
func TestGenerate_Abstain_AllRejected(t *testing.T) {
	scores := []contextevaluator.CandidateScore{
		{CandidateID: "c1", Scores: contextevaluator.DimensionScores{Relevance: 0.2, Consistency: 0.2, Usefulness: 0.2}, Composite: 0.20, Fate: "rejected", RejectionReason: "below minimum consistency"},
		{CandidateID: "c2", Scores: contextevaluator.DimensionScores{Relevance: 0.15, Consistency: 0.1, Usefulness: 0.1}, Composite: 0.12, Fate: "rejected", RejectionReason: "below minimum relevance"},
	}
	envelope := makeEnvelope(decisionengine.OutcomeAbstain, 0.16, scores)
	rec := newRecorder().Generate(envelope, "run deployment", "session-002")

	if rec.Outcome != "abstain" {
		t.Errorf("outcome = %q, want abstain", rec.Outcome)
	}
	for _, fate := range rec.CandidateFates {
		if fate.Fate != "rejected" {
			t.Errorf("expected all rejected, got %s fate for %s", fate.Fate, fate.CandidateID)
		}
	}
	if len(rec.Scores) != 2 {
		t.Errorf("scores len = %d, want 2", len(rec.Scores))
	}
}

// Scenario: Attribution record includes memory decision for write outcome
func TestGenerate_Write_MemoryDecisionPresent(t *testing.T) {
	scores := []contextevaluator.CandidateScore{
		{CandidateID: "mem-1", Scores: contextevaluator.DimensionScores{Relevance: 0.9, Consistency: 0.85, Usefulness: 0.85}, Composite: 0.87, Fate: "survived"},
	}
	envelope := makeEnvelope(decisionengine.OutcomeWrite, 0.87, scores)
	rec := newRecorder().Generate(envelope, "persist memory", "session-003")

	if rec.Outcome != "write" {
		t.Errorf("outcome = %q, want write", rec.Outcome)
	}
	if rec.MemoryDecision == nil {
		t.Fatal("memory_decision is nil for write outcome")
	}
	if rec.MemoryDecision.Decision != attributionrecorder.MemoryDecisionWriteApproved {
		t.Errorf("memory_decision.decision = %q, want write_approved", rec.MemoryDecision.Decision)
	}
	if rec.MemoryDecision.Score != 0.87 {
		t.Errorf("memory_decision.score = %.2f, want 0.87", rec.MemoryDecision.Score)
	}
}

// Scenario: Attribution for backtrack outcome
func TestGenerate_Backtrack_RecordedCorrectly(t *testing.T) {
	scores := []contextevaluator.CandidateScore{
		{CandidateID: "c1", Scores: contextevaluator.DimensionScores{Relevance: 0.5, Consistency: 0.5, Usefulness: 0.5}, Composite: 0.50, Fate: "survived"},
	}
	prior := 0.75
	envelope := decisionengine.DecisionEnvelope{
		Outcome:         decisionengine.OutcomeBacktrack,
		Confidence:      0.50,
		PriorConfidence: &prior,
		Scores:          scores,
	}
	rec := newRecorder().Generate(envelope, "refactor auth", "")

	if rec.Outcome != "backtrack" {
		t.Errorf("outcome = %q, want backtrack", rec.Outcome)
	}
}

// Scenario: Inline summary is compact (< 500 bytes)
func TestGenerate_InlineSummaryIsCompact(t *testing.T) {
	scores := []contextevaluator.CandidateScore{
		{CandidateID: "c1", Scores: contextevaluator.DimensionScores{Relevance: 0.8, Consistency: 0.8, Usefulness: 0.8}, Composite: 0.80, Fate: "survived"},
	}
	envelope := makeEnvelope(decisionengine.OutcomeContinue, 0.80, scores)
	rec := newRecorder().Generate(envelope, "any task", "")

	data, err := json.Marshal(rec.InlineSummary)
	if err != nil {
		t.Fatalf("marshal inline summary: %v", err)
	}
	if len(data) >= 500 {
		t.Errorf("inline summary size = %d bytes, must be < 500", len(data))
	}
	if rec.InlineSummary.Outcome == "" {
		t.Error("inline summary outcome must not be empty")
	}
	if rec.InlineSummary.AttributionID == "" {
		t.Error("inline summary attribution_id must not be empty")
	}
}

// Scenario: All five outcome types generate records
func TestGenerate_AllFiveOutcomes(t *testing.T) {
	outcomes := []decisionengine.Outcome{
		decisionengine.OutcomeContinue,
		decisionengine.OutcomeMinimal,
		decisionengine.OutcomeAbstain,
		decisionengine.OutcomeBacktrack,
		decisionengine.OutcomeWrite,
	}
	r := newRecorder()
	for _, outcome := range outcomes {
		scores := []contextevaluator.CandidateScore{
			{CandidateID: "c1", Scores: contextevaluator.DimensionScores{Relevance: 0.5, Consistency: 0.5, Usefulness: 0.5}, Composite: 0.50, Fate: "survived"},
		}
		envelope := makeEnvelope(outcome, 0.50, scores)
		rec := r.Generate(envelope, "task", "")
		if rec.Outcome != string(outcome) {
			t.Errorf("outcome %s: got %q", outcome, rec.Outcome)
		}
		if rec.ID == "" {
			t.Errorf("outcome %s: ID must not be empty", outcome)
		}
	}
}

// ---- US2 Tests ----

// Scenario: Persist an attribution record to the workspace
func TestPersist_WritesFileToAttributionDir(t *testing.T) {
	dir := t.TempDir()
	r := newRecorder()
	scores := []contextevaluator.CandidateScore{
		{CandidateID: "c1", Scores: contextevaluator.DimensionScores{Relevance: 0.8, Consistency: 0.8, Usefulness: 0.8}, Composite: 0.80, Fate: "survived"},
	}
	envelope := makeEnvelope(decisionengine.OutcomeMinimal, 0.55, scores)
	rec := r.Generate(envelope, "write tests", "")

	if err := r.Persist(dir, rec); err != nil {
		t.Fatalf("persist: %v", err)
	}

	path := filepath.Join(dir, "attribution", rec.ID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	var loaded attributionrecorder.Record
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if loaded.ID != rec.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, rec.ID)
	}
	if loaded.Outcome != "minimal" {
		t.Errorf("outcome = %q, want minimal", loaded.Outcome)
	}
}

// Scenario: Load a persisted record by ID
func TestLoadAttribution_ReturnsPersistedRecord(t *testing.T) {
	dir := t.TempDir()
	r := newRecorder()
	scores := []contextevaluator.CandidateScore{
		{CandidateID: "c1", Scores: contextevaluator.DimensionScores{Relevance: 0.9, Consistency: 0.9, Usefulness: 0.9}, Composite: 0.90, Fate: "survived"},
	}
	envelope := makeEnvelope(decisionengine.OutcomeContinue, 0.90, scores)
	rec := r.Generate(envelope, "load test", "")
	_ = r.Persist(dir, rec)

	loaded, err := r.LoadAttribution(dir, rec.ID)
	if err != nil {
		t.Fatalf("LoadAttribution: %v", err)
	}
	if loaded.ID != rec.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, rec.ID)
	}
	if loaded.TaskContext != "load test" {
		t.Errorf("task_context = %q", loaded.TaskContext)
	}
}

// Scenario: Handle missing attribution ID gracefully
func TestLoadAttribution_NotFound(t *testing.T) {
	dir := t.TempDir()
	r := newRecorder()

	_, err := r.LoadAttribution(dir, "attr-nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, attributionrecorder.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// Scenario: List records filtered by outcome type
func TestListAttributions_FilterByOutcome(t *testing.T) {
	dir := t.TempDir()
	r := newRecorder()

	persist := func(outcome decisionengine.Outcome, task string, offset time.Duration) {
		scores := []contextevaluator.CandidateScore{
			{CandidateID: "c1", Scores: contextevaluator.DimensionScores{Relevance: 0.5, Consistency: 0.5, Usefulness: 0.5}, Composite: 0.5, Fate: "survived"},
		}
		rr := attributionrecorder.NewWithClock(func() time.Time { return fixedNow().Add(offset) })
		rec := rr.Generate(makeEnvelope(outcome, 0.5, scores), task, "")
		_ = r.Persist(dir, rec)
	}

	persist(decisionengine.OutcomeContinue, "task-a", 0)
	persist(decisionengine.OutcomeContinue, "task-b", time.Millisecond)
	persist(decisionengine.OutcomeAbstain, "task-c", 2*time.Millisecond)
	persist(decisionengine.OutcomeAbstain, "task-d", 3*time.Millisecond)
	persist(decisionengine.OutcomeMinimal, "task-e", 4*time.Millisecond)

	records, err := r.ListAttributions(dir, attributionrecorder.Filter{Outcome: "abstain"})
	if err != nil {
		t.Fatalf("ListAttributions: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("len = %d, want 2", len(records))
	}
	for _, rec := range records {
		if rec.Outcome != "abstain" {
			t.Errorf("outcome = %q, want abstain", rec.Outcome)
		}
	}
}

// Scenario: List records filtered by time range
func TestListAttributions_FilterByTimeRange(t *testing.T) {
	dir := t.TempDir()
	r := newRecorder()

	day1 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)

	scores := []contextevaluator.CandidateScore{
		{CandidateID: "c1", Scores: contextevaluator.DimensionScores{Relevance: 0.5, Consistency: 0.5, Usefulness: 0.5}, Composite: 0.5, Fate: "survived"},
	}
	for i, day := range []time.Time{day1, day2, day3} {
		d := day
		rr := attributionrecorder.NewWithClock(func() time.Time { return d })
		rec := rr.Generate(makeEnvelope(decisionengine.OutcomeContinue, 0.7, scores), fmt.Sprintf("task-%d", i), "")
		_ = r.Persist(dir, rec)
	}

	from := day2.Add(-time.Minute)
	to := day2.Add(time.Minute)
	records, err := r.ListAttributions(dir, attributionrecorder.Filter{From: &from, To: &to})
	if err != nil {
		t.Fatalf("ListAttributions: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("len = %d, want 1", len(records))
	}
}

// Scenario: Empty attribution dir returns empty slice, not error
func TestListAttributions_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	r := newRecorder()

	records, err := r.ListAttributions(dir, attributionrecorder.Filter{})
	if err != nil {
		t.Fatalf("ListAttributions: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}
