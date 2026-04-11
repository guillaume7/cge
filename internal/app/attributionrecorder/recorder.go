// Package attributionrecorder generates, persists, and retrieves
// attribution-first decision records (ADR-021) for every evaluator-loop
// decision.
package attributionrecorder

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/guillaume-galp/cge/internal/app/decisionengine"
)

const (
	schemaVersion  = "v1"
	attributionDir = "attribution"

	// MemoryDecisionWriteApproved means the write outcome was approved.
	MemoryDecisionWriteApproved = "write_approved"
	// MemoryDecisionDeferred means a write was deferred due to moderate confidence.
	MemoryDecisionDeferred = "deferred"
	// MemoryDecisionSkipped means a write was skipped due to low confidence.
	MemoryDecisionSkipped = "skipped"
)

// ErrNotFound is returned when an attribution record does not exist.
var ErrNotFound = errors.New("attribution record not found")

// Record is a full attribution record persisted to the local workspace.
type Record struct {
	SchemaVersion      string          `json:"schema_version"`
	ID                 string          `json:"id"`
	Timestamp          time.Time       `json:"timestamp"`
	TaskContext        string          `json:"task_context"`
	SessionID          string          `json:"session_id,omitempty"`
	Outcome            string          `json:"outcome"`
	Scores             []CandidateScore `json:"scores"`
	CompositeConfidence float64         `json:"composite_confidence"`
	CandidateFates     []CandidateFate  `json:"candidate_fates"`
	MemoryDecision     *MemoryDecision  `json:"memory_decision,omitempty"`
	InlineSummary      InlineSummary    `json:"inline_summary"`
}

// CandidateScore records the per-dimension scores for a single candidate.
type CandidateScore struct {
	CandidateID string  `json:"candidate_id"`
	Relevance   float64 `json:"relevance"`
	Consistency float64 `json:"consistency"`
	Usefulness  float64 `json:"usefulness"`
	Composite   float64 `json:"composite"`
}

// CandidateFate records whether a candidate survived, was trimmed, or rejected.
type CandidateFate struct {
	CandidateID     string `json:"candidate_id"`
	Fate            string `json:"fate"` // "survived", "trimmed", "rejected"
	RejectionReason string `json:"rejection_reason,omitempty"`
}

// MemoryDecision records the outcome of a write decision.
type MemoryDecision struct {
	Decision string  `json:"decision"` // "write_approved", "deferred", "skipped"
	Reason   string  `json:"reason,omitempty"`
	Score    float64 `json:"score"`
}

// InlineSummary is a compact summary suitable for inclusion in a decision
// envelope. It must remain under 500 bytes when serialized.
type InlineSummary struct {
	AttributionID  string  `json:"attribution_id"`
	Outcome        string  `json:"outcome"`
	Confidence     float64 `json:"confidence"`
	CandidateCount int     `json:"candidate_count"`
}

// Filter selects a subset of attribution records.
type Filter struct {
	Outcome     string     `json:"outcome,omitempty"`
	TaskContext string     `json:"task_context,omitempty"`
	From        *time.Time `json:"from,omitempty"`
	To          *time.Time `json:"to,omitempty"`
}

// Recorder generates and persists attribution records.
type Recorder struct {
	now func() time.Time
}

// New returns a Recorder with a real clock.
func New() Recorder {
	return Recorder{now: func() time.Time { return time.Now().UTC() }}
}

// NewWithClock returns a Recorder with a controllable clock for testing.
func NewWithClock(now func() time.Time) Recorder {
	return Recorder{now: now}
}

// GenerateWithMemory builds an AttributionRecord with an explicit memory
// decision, for callers that can determine write eligibility. For read-only
// commands use Generate, which sets MemoryDecision to "skipped".
func (r Recorder) GenerateWithMemory(envelope decisionengine.DecisionEnvelope, taskContext, sessionID string, mem *MemoryDecision) Record {
	rec := r.Generate(envelope, taskContext, sessionID)
	rec.MemoryDecision = mem
	return rec
}
func (r Recorder) Generate(envelope decisionengine.DecisionEnvelope, taskContext, sessionID string) Record {
	now := r.now()
	id := generateID(now)

	fates := buildCandidateFates(envelope)
	scores := buildCandidateScores(envelope)

	rec := Record{
		SchemaVersion:       schemaVersion,
		ID:                  id,
		Timestamp:           now,
		TaskContext:         taskContext,
		SessionID:           sessionID,
		Outcome:             string(envelope.Outcome),
		Scores:              scores,
		CompositeConfidence: envelope.Confidence,
		CandidateFates:      fates,
	}

	// Populate memory decision for write/abstain outcomes that relate to writes.
	if envelope.Outcome == decisionengine.OutcomeWrite {
		var score float64
		if len(envelope.Scores) > 0 {
			score = envelope.Scores[0].Composite
		}
		rec.MemoryDecision = &MemoryDecision{
			Decision: MemoryDecisionWriteApproved,
			Score:    score,
		}
	}

	// For read-only commands, record memory decision as skipped.
	if rec.MemoryDecision == nil {
		rec.MemoryDecision = &MemoryDecision{
			Decision: MemoryDecisionSkipped,
			Reason:   "read-only command",
		}
	}

	rec.InlineSummary = InlineSummary{
		AttributionID:  id,
		Outcome:        string(envelope.Outcome),
		Confidence:     envelope.Confidence,
		CandidateCount: len(envelope.Scores),
	}

	return rec
}

// Persist writes the record to the workspace's attribution directory.
// The file is named <record.ID>.json under <workspacePath>/attribution/.
func (r Recorder) Persist(workspacePath string, record Record) error {
	dir := filepath.Join(workspacePath, attributionDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("attributionrecorder: create attribution dir: %w", err)
	}

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("attributionrecorder: marshal record %s: %w", record.ID, err)
	}

	path := filepath.Join(dir, record.ID+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("attributionrecorder: write record %s: %w", record.ID, err)
	}
	return nil
}

// LoadAttribution loads a single persisted record by ID.
// It returns ErrNotFound if no record with that ID exists.
func (r Recorder) LoadAttribution(workspacePath, id string) (Record, error) {
	if strings.TrimSpace(id) == "" {
		return Record{}, fmt.Errorf("attributionrecorder: attribution ID must not be empty")
	}
	path := filepath.Join(workspacePath, attributionDir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Record{}, fmt.Errorf("%w: %s", ErrNotFound, id)
		}
		return Record{}, fmt.Errorf("attributionrecorder: load record %s: %w", id, err)
	}

	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return Record{}, fmt.Errorf("attributionrecorder: parse record %s: %w", id, err)
	}
	return rec, nil
}

// ListAttributions returns persisted attribution records matching the filter.
// Results are returned in ascending timestamp order.
func (r Recorder) ListAttributions(workspacePath string, filter Filter) ([]Record, error) {
	dir := filepath.Join(workspacePath, attributionDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Record{}, nil
		}
		return nil, fmt.Errorf("attributionrecorder: read attribution dir: %w", err)
	}

	var records []Record
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var rec Record
		if err := json.Unmarshal(data, &rec); err != nil {
			continue
		}

		if matchesFilter(rec, filter) {
			records = append(records, rec)
		}
	}

	sortByTimestamp(records)
	return records, nil
}

// matchesFilter returns true if the record matches all active filter criteria.
func matchesFilter(rec Record, filter Filter) bool {
	if filter.Outcome != "" && rec.Outcome != filter.Outcome {
		return false
	}
	if filter.TaskContext != "" && !strings.Contains(rec.TaskContext, filter.TaskContext) {
		return false
	}
	if filter.From != nil && rec.Timestamp.Before(*filter.From) {
		return false
	}
	if filter.To != nil && rec.Timestamp.After(*filter.To) {
		return false
	}
	return true
}

// sortByTimestamp sorts records ascending by timestamp (oldest first).
func sortByTimestamp(records []Record) {
	for i := 1; i < len(records); i++ {
		for j := i; j > 0 && records[j].Timestamp.Before(records[j-1].Timestamp); j-- {
			records[j], records[j-1] = records[j-1], records[j]
		}
	}
}

// generateID produces a unique attribution ID of the form attr-YYYYMMDD-HHMMSS-<random8>.
// The random suffix ensures collision safety even with sub-millisecond calls.
func generateID(t time.Time) string {
	var buf [4]byte
	_, _ = rand.Read(buf[:])
	return fmt.Sprintf("attr-%s-%s",
		t.UTC().Format("20060102-150405"),
		hex.EncodeToString(buf[:]),
	)
}

// buildCandidateFates derives per-candidate fates from the decision envelope.
func buildCandidateFates(envelope decisionengine.DecisionEnvelope) []CandidateFate {
	fates := make([]CandidateFate, 0, len(envelope.Scores))
	for _, cs := range envelope.Scores {
		fates = append(fates, CandidateFate{
			CandidateID:     cs.CandidateID,
			Fate:            cs.Fate,
			RejectionReason: cs.RejectionReason,
		})
	}
	return fates
}

// buildCandidateScores extracts per-dimension scores from the decision envelope.
func buildCandidateScores(envelope decisionengine.DecisionEnvelope) []CandidateScore {
	scores := make([]CandidateScore, 0, len(envelope.Scores))
	for _, cs := range envelope.Scores {
		scores = append(scores, CandidateScore{
			CandidateID: cs.CandidateID,
			Relevance:   cs.Scores.Relevance,
			Consistency: cs.Scores.Consistency,
			Usefulness:  cs.Scores.Usefulness,
			Composite:   cs.Composite,
		})
	}
	return scores
}
