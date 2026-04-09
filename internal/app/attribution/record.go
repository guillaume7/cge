// Package attribution persists attribution-first decision records (ADR-021) to
// the local workspace under .graph/attribution/. Records explain why the
// evaluator loop approved, deferred, or skipped a write, or why context was
// injected, minimized, or suppressed.
package attribution

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/guillaume-galp/cge/internal/app/contextevaluator"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

const (
	attributionDirName = "attribution"

	// MemoryDecisionApproved indicates the write was committed to the graph.
	MemoryDecisionApproved MemoryDecision = "approved"
	// MemoryDecisionDeferred indicates the write was held back but may be
	// retried with additional evidence (moderate confidence).
	MemoryDecisionDeferred MemoryDecision = "deferred"
	// MemoryDecisionSkipped indicates the write was suppressed and will not be
	// retried automatically (low confidence).
	MemoryDecisionSkipped MemoryDecision = "skipped"
)

// MemoryDecision is the outcome of an evaluated write gate.
type MemoryDecision string

// CandidateFate records how the evaluator treated a single candidate.
type CandidateFate struct {
	CandidateID  string                          `json:"candidate_id"`
	Fate         string                          `json:"fate"`
	Scores       contextevaluator.DimensionScores `json:"scores"`
	Composite    float64                          `json:"composite"`
	Reason       string                           `json:"reason,omitempty"`
	DownRanked   bool                             `json:"down_ranked,omitempty"`
	DownRankNote string                           `json:"down_rank_note,omitempty"`
}

// Record is a structured attribution record produced by the evaluator loop.
// It captures the decision context, per-candidate fates, and memory decision so
// that lab experiments can reconstruct and analyze evaluator behavior (ADR-021).
type Record struct {
	// ID is a unique, file-safe identifier for this record.
	ID string `json:"id"`
	// Outcome is the Decision Engine outcome that drove the record.
	Outcome string `json:"outcome"`
	// Task is the task context that the evaluator scored against.
	Task string `json:"task"`
	// SessionID is the agent session that produced the decision.
	SessionID string `json:"session_id,omitempty"`
	// Timestamp is the RFC-3339 time the record was generated.
	Timestamp string `json:"timestamp"`
	// AggregateConfidence is the bundle-level composite confidence.
	AggregateConfidence float64 `json:"aggregate_confidence"`
	// WriteThreshold is the write-confidence threshold that was applied.
	WriteThreshold *float64 `json:"write_threshold,omitempty"`
	// DeferThreshold is the defer-confidence threshold that was applied.
	DeferThreshold *float64 `json:"defer_threshold,omitempty"`
	// MemoryDecision is set for write-gate records and indicates whether the
	// write was approved, deferred, or skipped.
	MemoryDecision *MemoryDecision `json:"memory_decision,omitempty"`
	// MemoryDecisionReason explains why the write was deferred or skipped.
	MemoryDecisionReason string `json:"memory_decision_reason,omitempty"`
	// CandidateFates contains per-candidate evaluation outcomes.
	CandidateFates []CandidateFate `json:"candidate_fates,omitempty"`
	// InlineSummary is a compact representation suitable for inclusion in a
	// decision envelope (≤500 bytes).
	InlineSummary InlineSummary `json:"inline_summary"`
}

// InlineSummary is the compact portion of a Record, designed to remain small
// enough to include inline in decision envelopes (target ≤500 bytes).
type InlineSummary struct {
	Outcome        string  `json:"outcome"`
	Confidence     float64 `json:"confidence"`
	CandidateCount int     `json:"candidate_count"`
}

// Recorder persists attribution records to the local workspace.
type Recorder struct {
	now func() time.Time
}

// NewRecorder returns a Recorder that uses the current time.
func NewRecorder() *Recorder {
	return &Recorder{now: func() time.Time { return time.Now().UTC() }}
}

// NowForTest replaces the clock with a deterministic function for tests.
func (r *Recorder) NowForTest(now func() time.Time) {
	if r == nil {
		return
	}
	r.now = now
}

// Record generates, persists, and returns an attribution record at the given
// workspace path. The record file is named <id>.json under
// .graph/attribution/.
func (r *Recorder) Record(workspace repo.Workspace, rec Record) (Record, error) {
	if r == nil {
		r = NewRecorder()
	}
	if rec.Timestamp == "" {
		rec.Timestamp = r.now().Format(time.RFC3339)
	}
	if rec.ID == "" {
		rec.ID = generateID(rec.Timestamp, rec.Outcome, rec.Task)
	}
	rec.InlineSummary = InlineSummary{
		Outcome:        rec.Outcome,
		Confidence:     rec.AggregateConfidence,
		CandidateCount: len(rec.CandidateFates),
	}

	dir := filepath.Join(workspace.WorkspacePath, attributionDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Record{}, fmt.Errorf("create attribution directory %s: %w", dir, err)
	}

	payload, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return Record{}, fmt.Errorf("marshal attribution record: %w", err)
	}
	payload = append(payload, '\n')

	path := filepath.Join(dir, rec.ID+".json")
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return Record{}, fmt.Errorf("write attribution record %s: %w", path, err)
	}

	return rec, nil
}

// ListFilter controls which attribution records ListRecords returns.
type ListFilter struct {
	// Outcome filters to records with this decision outcome (e.g. "write").
	// Empty means no filter.
	Outcome string
	// After and Before bound the Timestamp range (RFC-3339).
	After  string
	Before string
}

// ListRecords loads all attribution records from the workspace that match f.
func ListRecords(workspace repo.Workspace, f ListFilter) ([]Record, error) {
	dir := filepath.Join(workspace.WorkspacePath, attributionDirName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("list attribution records: %w", err)
	}

	var records []Record
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		rec, err := loadRecord(path)
		if err != nil {
			continue
		}
		if !matchesFilter(rec, f) {
			continue
		}
		records = append(records, rec)
	}
	return records, nil
}

// LoadRecord returns the attribution record with the given ID from the
// workspace. It returns an error if the record does not exist.
func LoadRecord(workspace repo.Workspace, id string) (Record, error) {
	path := filepath.Join(workspace.WorkspacePath, attributionDirName, id+".json")
	rec, err := loadRecord(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Record{}, fmt.Errorf("attribution record %q not found", id)
		}
		return Record{}, err
	}
	return rec, nil
}

func loadRecord(path string) (Record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Record{}, err
	}
	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return Record{}, fmt.Errorf("parse attribution record %s: %w", path, err)
	}
	return rec, nil
}

func matchesFilter(rec Record, f ListFilter) bool {
	if f.Outcome != "" && rec.Outcome != f.Outcome {
		return false
	}
	if f.After != "" && rec.Timestamp <= f.After {
		return false
	}
	if f.Before != "" && rec.Timestamp >= f.Before {
		return false
	}
	return true
}

// generateID produces a file-safe attribution record identifier from the
// timestamp, outcome, and task. It is safe to use as a filename.
func generateID(timestamp, outcome, task string) string {
	ts := strings.ReplaceAll(timestamp, ":", "")
	ts = strings.ReplaceAll(ts, "-", "")
	if len(ts) > 15 {
		ts = ts[:15]
	}
	slug := sanitizeSlug(task)
	if len(slug) > 24 {
		slug = slug[:24]
	}
	return fmt.Sprintf("attr-%s-%s-%s", ts, outcome, slug)
}

func sanitizeSlug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ', r == '-', r == '_', r == '/':
			b.WriteByte('-')
		}
	}
	result := b.String()
	result = strings.Trim(result, "-")
	return result
}
