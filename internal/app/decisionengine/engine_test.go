package decisionengine

import (
	"encoding/json"
	"testing"

	"github.com/guillaume-galp/cge/internal/app/contextevaluator"
)

// helpers

func ptr(v float64) *float64 { return &v }

func makeScores(items ...struct {
	id        string
	composite float64
	fate      string
}) []contextevaluator.CandidateScore {
	scores := make([]contextevaluator.CandidateScore, len(items))
	for i, item := range items {
		scores[i] = contextevaluator.CandidateScore{
			CandidateID: item.id,
			Composite:   item.composite,
			Fate:        item.fate,
			Scores: contextevaluator.DimensionScores{
				Relevance:   item.composite,
				Consistency: item.composite,
				Usefulness:  item.composite,
			},
		}
	}
	return scores
}

func evalResult(confidence float64, scores []contextevaluator.CandidateScore) contextevaluator.EvaluationResult {
	return contextevaluator.EvaluationResult{
		AggregateConfidence: confidence,
		Scores:              scores,
	}
}

// US1: Five normalized decision outcomes

func TestDecide_Continue_WhenConfidenceAboveInjectionThreshold(t *testing.T) {
	t.Parallel()

	eng := NewWithDefaults()
	scores := makeScores(
		struct {
			id        string
			composite float64
			fate      string
		}{"c1", 0.85, "survived"},
		struct {
			id        string
			composite float64
			fate      string
		}{"c2", 0.80, "survived"},
	)
	envelope, err := eng.Decide(ContextDecisionRequest{
		EvaluationResult: evalResult(0.80, scores),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envelope.Outcome != OutcomeContinue {
		t.Fatalf("outcome = %q, want %q", envelope.Outcome, OutcomeContinue)
	}
	if len(envelope.Bundle) == 0 {
		t.Fatal("bundle must not be empty for continue outcome")
	}
}

func TestDecide_Minimal_WhenConfidenceIsBetweenMinimalAndInjectionThresholds(t *testing.T) {
	t.Parallel()

	eng := NewWithDefaults() // injection=0.70, minimal=0.40
	scores := makeScores(
		struct {
			id        string
			composite float64
			fate      string
		}{"c1", 0.60, "survived"},
		struct {
			id        string
			composite float64
			fate      string
		}{"c2", 0.50, "survived"},
		struct {
			id        string
			composite float64
			fate      string
		}{"c3", 0.45, "trimmed"},
		struct {
			id        string
			composite float64
			fate      string
		}{"c4", 0.30, "rejected"},
	)
	envelope, err := eng.Decide(ContextDecisionRequest{
		EvaluationResult: evalResult(0.55, scores),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envelope.Outcome != OutcomeMinimal {
		t.Fatalf("outcome = %q, want %q", envelope.Outcome, OutcomeMinimal)
	}
	// Only the single highest-scored survived candidate survives.
	if len(envelope.Bundle) != 1 {
		t.Fatalf("bundle length = %d, want 1", len(envelope.Bundle))
	}
	if envelope.Bundle[0].CandidateID != "c1" {
		t.Fatalf("bundle candidate = %q, want %q", envelope.Bundle[0].CandidateID, "c1")
	}
}

func TestDecide_Abstain_WhenConfidenceBelowMinimalThreshold(t *testing.T) {
	t.Parallel()

	eng := NewWithDefaults() // minimal=0.40
	scores := makeScores(
		struct {
			id        string
			composite float64
			fate      string
		}{"c1", 0.25, "rejected"},
	)
	envelope, err := eng.Decide(ContextDecisionRequest{
		EvaluationResult: evalResult(0.25, scores),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envelope.Outcome != OutcomeAbstain {
		t.Fatalf("outcome = %q, want %q", envelope.Outcome, OutcomeAbstain)
	}
	if len(envelope.Bundle) != 0 {
		t.Fatalf("bundle must be empty for abstain outcome, got %d items", len(envelope.Bundle))
	}
}

func TestDecide_Backtrack_WhenQualityRegresses(t *testing.T) {
	t.Parallel()

	eng := NewWithDefaults() // regression_delta=0.10
	scores := makeScores(
		struct {
			id        string
			composite float64
			fate      string
		}{"c1", 0.40, "survived"},
	)
	prior := 0.70
	envelope, err := eng.Decide(ContextDecisionRequest{
		EvaluationResult: evalResult(0.40, scores),
		PriorConfidence:  &prior,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envelope.Outcome != OutcomeBacktrack {
		t.Fatalf("outcome = %q, want %q", envelope.Outcome, OutcomeBacktrack)
	}
	if envelope.PriorConfidence == nil || *envelope.PriorConfidence != prior {
		t.Fatalf("prior_confidence = %v, want %.2f", envelope.PriorConfidence, prior)
	}
	if len(envelope.Bundle) != 0 {
		t.Fatalf("bundle must be empty for backtrack outcome")
	}
}

func TestDecideWrite_Write_WhenOutputConfidenceAboveWriteThreshold(t *testing.T) {
	t.Parallel()

	eng := NewWithDefaults() // write=0.80
	output := contextevaluator.OutputEvaluation{
		CandidateID: "out:memory-candidate",
		Composite:   0.85,
		Scores: contextevaluator.DimensionScores{
			Relevance:   0.90,
			Consistency: 0.85,
			Usefulness:  0.80,
		},
	}
	envelope, err := eng.DecideWrite(WriteDecisionRequest{Output: output})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envelope.Outcome != OutcomeWrite {
		t.Fatalf("outcome = %q, want %q", envelope.Outcome, OutcomeWrite)
	}
}

func TestDecideWrite_Abstain_WhenOutputConfidenceBelowWriteThreshold(t *testing.T) {
	t.Parallel()

	eng := NewWithDefaults() // write=0.80
	output := contextevaluator.OutputEvaluation{
		CandidateID: "out:weak-candidate",
		Composite:   0.60,
	}
	envelope, err := eng.DecideWrite(WriteDecisionRequest{Output: output})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envelope.Outcome != OutcomeAbstain {
		t.Fatalf("outcome = %q, want %q", envelope.Outcome, OutcomeAbstain)
	}
}

// US1 AC2: machine-readable string labels
func TestOutcomeLabels_AreMachineReadable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		outcome Outcome
		want    string
	}{
		{OutcomeContinue, "continue"},
		{OutcomeMinimal, "minimal"},
		{OutcomeAbstain, "abstain"},
		{OutcomeBacktrack, "backtrack"},
		{OutcomeWrite, "write"},
	}
	for _, tc := range cases {
		if string(tc.outcome) != tc.want {
			t.Errorf("Outcome(%q) = %q, want %q", tc.outcome, string(tc.outcome), tc.want)
		}
	}
}

// US2: Configurable thresholds

func TestDecide_DefaultConservativeThresholds_PreferAbstainAtLowConfidence(t *testing.T) {
	t.Parallel()

	eng := NewWithDefaults()
	scores := makeScores(struct {
		id        string
		composite float64
		fate      string
	}{"c1", 0.30, "rejected"})
	envelope, err := eng.Decide(ContextDecisionRequest{
		EvaluationResult: evalResult(0.30, scores),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envelope.Outcome != OutcomeAbstain {
		t.Fatalf("default thresholds with confidence 0.30 should abstain, got %q", envelope.Outcome)
	}
}

func TestDecide_DefaultConservativeThresholds_PreferMinimalAtModerateConfidence(t *testing.T) {
	t.Parallel()

	eng := NewWithDefaults() // injection=0.70, so 0.55 → minimal
	scores := makeScores(struct {
		id        string
		composite float64
		fate      string
	}{"c1", 0.55, "survived"})
	envelope, err := eng.Decide(ContextDecisionRequest{
		EvaluationResult: evalResult(0.55, scores),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envelope.Outcome == OutcomeContinue {
		t.Fatalf("default thresholds with confidence 0.55 should not continue")
	}
}

func TestDecide_InvocationOverride_AppliesForSingleCall(t *testing.T) {
	t.Parallel()

	eng := NewWithDefaults() // default injection=0.70
	override := &Thresholds{
		Injection:       0.50,
		Minimal:         0.30,
		Write:           0.80,
		RegressionDelta: 0.10,
	}
	scores := makeScores(struct {
		id        string
		composite float64
		fate      string
	}{"c1", 0.60, "survived"})
	envelope, err := eng.Decide(ContextDecisionRequest{
		EvaluationResult: evalResult(0.60, scores),
		Thresholds:       override,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// With override injection=0.50, confidence 0.60 >= 0.50 → continue
	if envelope.Outcome != OutcomeContinue {
		t.Fatalf("outcome = %q, want %q (override injection=0.50)", envelope.Outcome, OutcomeContinue)
	}

	// Verify the engine-level defaults are unchanged for a subsequent call.
	envelope2, err := eng.Decide(ContextDecisionRequest{
		EvaluationResult: evalResult(0.60, scores),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envelope2.Outcome != OutcomeMinimal {
		t.Fatalf("engine default injection=0.70 should yield minimal for 0.60, got %q", envelope2.Outcome)
	}
}

func TestDecide_LoadedThresholds_MatchConfiguredValues(t *testing.T) {
	t.Parallel()

	thresholds := Thresholds{
		Injection:       0.75,
		Minimal:         0.45,
		Write:           0.80,
		RegressionDelta: 0.10,
	}
	eng, err := New(thresholds)
	if err != nil {
		t.Fatalf("unexpected error creating engine: %v", err)
	}

	scores := makeScores(struct {
		id        string
		composite float64
		fate      string
	}{"c1", 0.76, "survived"})
	envelope, _ := eng.Decide(ContextDecisionRequest{
		EvaluationResult: evalResult(0.76, scores),
	})
	if envelope.Outcome != OutcomeContinue {
		t.Fatalf("injection=0.75, confidence=0.76 → want continue, got %q", envelope.Outcome)
	}

	scores2 := makeScores(struct {
		id        string
		composite float64
		fate      string
	}{"c1", 0.50, "survived"})
	envelope2, _ := eng.Decide(ContextDecisionRequest{
		EvaluationResult: evalResult(0.50, scores2),
	})
	if envelope2.Outcome != OutcomeMinimal {
		t.Fatalf("minimal=0.45, injection=0.75, confidence=0.50 → want minimal, got %q", envelope2.Outcome)
	}
}

func TestNew_RejectsMisconfiguredThresholds_MinimalGEQInjection(t *testing.T) {
	t.Parallel()

	_, err := New(Thresholds{
		Injection:       0.50,
		Minimal:         0.60, // misconfigured: minimal > injection
		Write:           0.80,
		RegressionDelta: 0.10,
	})
	if err == nil {
		t.Fatal("expected error for minimal >= injection, got nil")
	}
}

func TestNew_RejectsMisconfiguredThresholds_ZeroRegressionDelta(t *testing.T) {
	t.Parallel()

	_, err := New(Thresholds{
		Injection:       0.70,
		Minimal:         0.40,
		Write:           0.80,
		RegressionDelta: 0, // invalid
	})
	if err == nil {
		t.Fatal("expected error for regression_delta=0, got nil")
	}
}

func TestDecide_InvocationOverride_RejectsMisconfigured(t *testing.T) {
	t.Parallel()

	eng := NewWithDefaults()
	override := &Thresholds{
		Injection:       0.40,
		Minimal:         0.60, // invalid: minimal > injection
		Write:           0.80,
		RegressionDelta: 0.10,
	}
	_, err := eng.Decide(ContextDecisionRequest{
		EvaluationResult: evalResult(0.55, nil),
		Thresholds:       override,
	})
	if err == nil {
		t.Fatal("expected error for misconfigured invocation override, got nil")
	}
}

// US2 AC3: regression check does NOT trigger when delta is below threshold
func TestDecide_NoBacktrack_WhenRegressionIsBelowDelta(t *testing.T) {
	t.Parallel()

	eng := NewWithDefaults() // regression_delta=0.10
	scores := makeScores(struct {
		id        string
		composite float64
		fate      string
	}{"c1", 0.65, "survived"})
	prior := 0.70 // drop = 0.05, less than delta 0.10
	envelope, err := eng.Decide(ContextDecisionRequest{
		EvaluationResult: evalResult(0.65, scores),
		PriorConfidence:  &prior,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envelope.Outcome == OutcomeBacktrack {
		t.Fatalf("drop=0.05 < delta=0.10 should not trigger backtrack")
	}
}

// US3: Machine-readable decision envelopes

func TestDecisionEnvelope_Continue_ContainsFullBundle(t *testing.T) {
	t.Parallel()

	eng := NewWithDefaults()
	scores := makeScores(
		struct {
			id        string
			composite float64
			fate      string
		}{"c1", 0.90, "survived"},
		struct {
			id        string
			composite float64
			fate      string
		}{"c2", 0.80, "survived"},
		struct {
			id        string
			composite float64
			fate      string
		}{"c3", 0.75, "survived"},
	)
	envelope, err := eng.Decide(ContextDecisionRequest{
		EvaluationResult: evalResult(0.82, scores),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envelope.Outcome != OutcomeContinue {
		t.Fatalf("outcome = %q, want continue", envelope.Outcome)
	}
	if len(envelope.Scores) != 3 {
		t.Fatalf("scores count = %d, want 3", len(envelope.Scores))
	}
	if len(envelope.Bundle) != 3 {
		t.Fatalf("bundle count = %d, want 3 (full bundle for continue)", len(envelope.Bundle))
	}
	if envelope.Confidence <= 0 {
		t.Fatal("confidence must be > 0")
	}
}

func TestDecisionEnvelope_Minimal_ContainsOnlyTopCandidate(t *testing.T) {
	t.Parallel()

	eng := NewWithDefaults()
	scores := makeScores(
		struct {
			id        string
			composite float64
			fate      string
		}{"c1", 0.65, "survived"},
		struct {
			id        string
			composite float64
			fate      string
		}{"c2", 0.55, "survived"},
		struct {
			id        string
			composite float64
			fate      string
		}{"c3", 0.48, "trimmed"},
		struct {
			id        string
			composite float64
			fate      string
		}{"c4", 0.35, "rejected"},
	)
	envelope, err := eng.Decide(ContextDecisionRequest{
		EvaluationResult: evalResult(0.55, scores),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envelope.Outcome != OutcomeMinimal {
		t.Fatalf("outcome = %q, want minimal", envelope.Outcome)
	}
	if len(envelope.Bundle) != 1 {
		t.Fatalf("bundle count = %d, want 1 for minimal", len(envelope.Bundle))
	}
	if envelope.Bundle[0].CandidateID != "c1" {
		t.Fatalf("bundle[0] = %q, want highest-scored survived c1", envelope.Bundle[0].CandidateID)
	}
}

func TestDecisionEnvelope_Abstain_ContainsScoresButNoBundle(t *testing.T) {
	t.Parallel()

	eng := NewWithDefaults()
	scores := makeScores(struct {
		id        string
		composite float64
		fate      string
	}{"c1", 0.20, "rejected"})
	envelope, err := eng.Decide(ContextDecisionRequest{
		EvaluationResult: evalResult(0.20, scores),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envelope.Outcome != OutcomeAbstain {
		t.Fatalf("outcome = %q, want abstain", envelope.Outcome)
	}
	if len(envelope.Scores) == 0 {
		t.Fatal("scores must be present for abstain outcome")
	}
	if len(envelope.Bundle) != 0 {
		t.Fatalf("bundle must be empty for abstain, got %d items", len(envelope.Bundle))
	}
}

func TestDecisionEnvelope_Backtrack_ContainsBothScoresAndNilBundle(t *testing.T) {
	t.Parallel()

	eng := NewWithDefaults()
	scores := makeScores(struct {
		id        string
		composite float64
		fate      string
	}{"c1", 0.40, "survived"})
	prior := 0.70
	envelope, err := eng.Decide(ContextDecisionRequest{
		EvaluationResult: evalResult(0.40, scores),
		PriorConfidence:  &prior,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envelope.Outcome != OutcomeBacktrack {
		t.Fatalf("outcome = %q, want backtrack", envelope.Outcome)
	}
	if len(envelope.Scores) == 0 {
		t.Fatal("scores must be present for backtrack outcome")
	}
	if envelope.PriorConfidence == nil {
		t.Fatal("prior_confidence must be set for backtrack outcome")
	}
	if *envelope.PriorConfidence != prior {
		t.Fatalf("prior_confidence = %.2f, want %.2f", *envelope.PriorConfidence, prior)
	}
	if len(envelope.Bundle) != 0 {
		t.Fatalf("bundle must be empty for backtrack, got %d items", len(envelope.Bundle))
	}
}

func TestDecisionEnvelope_IsValidJSON(t *testing.T) {
	t.Parallel()

	eng := NewWithDefaults()
	scores := makeScores(
		struct {
			id        string
			composite float64
			fate      string
		}{"c1", 0.80, "survived"},
	)
	for _, confidence := range []float64{0.80, 0.55, 0.20} {
		envelope, err := eng.Decide(ContextDecisionRequest{
			EvaluationResult: evalResult(confidence, scores),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, err := json.Marshal(envelope)
		if err != nil {
			t.Fatalf("envelope is not JSON-marshalable: %v", err)
		}
		var roundtrip DecisionEnvelope
		if err := json.Unmarshal(data, &roundtrip); err != nil {
			t.Fatalf("envelope JSON is not parseable: %v", err)
		}
		if roundtrip.Outcome != envelope.Outcome {
			t.Fatalf("roundtrip outcome = %q, want %q", roundtrip.Outcome, envelope.Outcome)
		}
	}
}

func TestDecisionEnvelope_Write_IsValidJSON(t *testing.T) {
	t.Parallel()

	eng := NewWithDefaults()
	output := contextevaluator.OutputEvaluation{
		CandidateID: "out:candidate",
		Composite:   0.88,
		Scores: contextevaluator.DimensionScores{
			Relevance:   0.90,
			Consistency: 0.85,
			Usefulness:  0.88,
		},
	}
	envelope, err := eng.DecideWrite(WriteDecisionRequest{Output: output})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("write envelope is not JSON-marshalable: %v", err)
	}
	var roundtrip DecisionEnvelope
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatalf("write envelope JSON is not parseable: %v", err)
	}
	if roundtrip.Outcome != OutcomeWrite {
		t.Fatalf("roundtrip outcome = %q, want write", roundtrip.Outcome)
	}
}
