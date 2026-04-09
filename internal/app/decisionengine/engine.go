// Package decisionengine translates Context Evaluator confidence scores into
// one of five normalized outcomes and composes machine-readable decision
// envelopes (ADR-019).
package decisionengine

import (
	"fmt"

	"github.com/guillaume-galp/cge/internal/app/contextevaluator"
)

// Outcome is one of the five normalized decision outcomes.
type Outcome string

const (
	OutcomeContinue  Outcome = "continue"
	OutcomeMinimal   Outcome = "minimal"
	OutcomeAbstain   Outcome = "abstain"
	OutcomeBacktrack Outcome = "backtrack"
	OutcomeWrite     Outcome = "write"
)

// Thresholds configures the confidence levels that drive outcome selection.
// Conservative defaults prefer abstain and minimal over aggressive injection
// (ADR-019 §2).
type Thresholds struct {
	// Injection is the minimum aggregate confidence for a "continue" outcome.
	Injection float64 `json:"injection"`
	// Minimal is the minimum aggregate confidence for a "minimal" outcome.
	// Must be strictly less than Injection.
	Minimal float64 `json:"minimal"`
	// Write is the minimum composite confidence for a "write" outcome.
	Write float64 `json:"write"`
	// RegressionDelta is the confidence drop that triggers a "backtrack" outcome.
	// A positive value; e.g. 0.10 means a drop of 0.10 or more triggers backtrack.
	RegressionDelta float64 `json:"regression_delta"`
}

// DefaultThresholds returns the conservative VP8 defaults (ADR-019 §2).
func DefaultThresholds() Thresholds {
	return Thresholds{
		Injection:       0.70,
		Minimal:         0.40,
		Write:           0.80,
		RegressionDelta: 0.10,
	}
}

// Validate returns an error if the thresholds are logically inconsistent.
func (t Thresholds) Validate() error {
	if t.Minimal >= t.Injection {
		return fmt.Errorf(
			"decision engine: threshold misconfiguration: minimal (%.4f) must be less than injection (%.4f)",
			t.Minimal, t.Injection,
		)
	}
	if t.Injection <= 0 || t.Injection > 1 {
		return fmt.Errorf("decision engine: threshold misconfiguration: injection (%.4f) must be in (0, 1]", t.Injection)
	}
	if t.Minimal < 0 || t.Minimal >= 1 {
		return fmt.Errorf("decision engine: threshold misconfiguration: minimal (%.4f) must be in [0, 1)", t.Minimal)
	}
	if t.Write <= 0 || t.Write > 1 {
		return fmt.Errorf("decision engine: threshold misconfiguration: write (%.4f) must be in (0, 1]", t.Write)
	}
	if t.RegressionDelta <= 0 {
		return fmt.Errorf("decision engine: threshold misconfiguration: regression_delta (%.4f) must be positive", t.RegressionDelta)
	}
	return nil
}

// Engine selects a normalized decision outcome by comparing evaluator
// confidence scores against configurable thresholds.
type Engine struct {
	thresholds Thresholds
}

// New returns a Decision Engine with the given thresholds. It validates the
// thresholds and returns an error if they are misconfigured.
func New(thresholds Thresholds) (Engine, error) {
	if err := thresholds.Validate(); err != nil {
		return Engine{}, err
	}
	return Engine{thresholds: thresholds}, nil
}

// NewWithDefaults returns a Decision Engine with the conservative VP8 defaults.
func NewWithDefaults() Engine {
	return Engine{thresholds: DefaultThresholds()}
}

// DecisionEnvelope is the machine-readable output of a decision pass (ADR-019 §3).
// It always contains the selected outcome and the evaluator scores that motivated it.
// The Bundle is populated only for "continue" and "minimal" outcomes.
type DecisionEnvelope struct {
	// Outcome is the selected normalized decision outcome.
	Outcome Outcome `json:"outcome"`
	// Confidence is the aggregate confidence that drove the outcome selection.
	Confidence float64 `json:"confidence"`
	// PriorConfidence is the aggregate confidence from the previous evaluation
	// pass. Populated only for "backtrack" outcomes.
	PriorConfidence *float64 `json:"prior_confidence,omitempty"`
	// Scores contains the per-candidate dimension scores from the evaluation.
	Scores []contextevaluator.CandidateScore `json:"scores"`
	// Bundle contains the scored candidates delivered to the consumer.
	// Populated only for "continue" (full bundle) and "minimal" (top candidate).
	Bundle []contextevaluator.CandidateScore `json:"bundle,omitempty"`
}

// ContextDecisionRequest is the input for bundle-level decisions.
type ContextDecisionRequest struct {
	// EvaluationResult is the scored context bundle from the Context Evaluator.
	EvaluationResult contextevaluator.EvaluationResult
	// PriorConfidence is the aggregate confidence from the previous evaluation
	// pass. When provided, a significant confidence regression triggers "backtrack".
	PriorConfidence *float64
	// Thresholds overrides the engine-level thresholds for this invocation only.
	Thresholds *Thresholds
}

// Decide selects a normalized outcome for a context bundle evaluation and
// returns a machine-readable decision envelope.
//
// Outcome selection precedence (checked in order):
//  1. backtrack — when PriorConfidence is provided and the drop exceeds RegressionDelta
//  2. continue  — when aggregate confidence >= Injection threshold
//  3. minimal   — when aggregate confidence >= Minimal threshold
//  4. abstain   — otherwise
func (e Engine) Decide(req ContextDecisionRequest) (DecisionEnvelope, error) {
	thresholds := e.thresholds
	if req.Thresholds != nil {
		thresholds = *req.Thresholds
	}
	if err := thresholds.Validate(); err != nil {
		return DecisionEnvelope{}, err
	}

	confidence := req.EvaluationResult.AggregateConfidence
	scores := req.EvaluationResult.Scores

	// 1. Backtrack: significant confidence regression detected.
	if req.PriorConfidence != nil {
		prior := *req.PriorConfidence
		if prior-confidence >= thresholds.RegressionDelta {
			return DecisionEnvelope{
				Outcome:         OutcomeBacktrack,
				Confidence:      confidence,
				PriorConfidence: req.PriorConfidence,
				Scores:          scores,
			}, nil
		}
	}

	// 2. Continue: confidence is above the injection threshold.
	if confidence >= thresholds.Injection {
		return DecisionEnvelope{
			Outcome:    OutcomeContinue,
			Confidence: confidence,
			Scores:     scores,
			Bundle:     survivedScores(scores),
		}, nil
	}

	// 3. Minimal: confidence is moderate; deliver only the top-scored candidate.
	if confidence >= thresholds.Minimal {
		return DecisionEnvelope{
			Outcome:    OutcomeMinimal,
			Confidence: confidence,
			Scores:     scores,
			Bundle:     topScoredCandidate(scores),
		}, nil
	}

	// 4. Abstain: confidence is too low to deliver any context.
	return DecisionEnvelope{
		Outcome:    OutcomeAbstain,
		Confidence: confidence,
		Scores:     scores,
	}, nil
}

// WriteDecisionRequest is the input for memory-write decisions.
type WriteDecisionRequest struct {
	// Output is the scored output candidate from the Context Evaluator.
	Output contextevaluator.OutputEvaluation
	// Thresholds overrides the engine-level thresholds for this invocation only.
	Thresholds *Thresholds
}

// DecideWrite selects either "write" or "abstain" for a candidate output.
// It returns "write" when the output's composite confidence meets or exceeds
// the Write threshold, and "abstain" otherwise.
func (e Engine) DecideWrite(req WriteDecisionRequest) (DecisionEnvelope, error) {
	thresholds := e.thresholds
	if req.Thresholds != nil {
		thresholds = *req.Thresholds
	}
	if err := thresholds.Validate(); err != nil {
		return DecisionEnvelope{}, err
	}

	score := contextevaluator.CandidateScore{
		CandidateID: req.Output.CandidateID,
		Scores:      req.Output.Scores,
		Composite:   req.Output.Composite,
	}
	scores := []contextevaluator.CandidateScore{score}

	if req.Output.Composite >= thresholds.Write {
		return DecisionEnvelope{
			Outcome:    OutcomeWrite,
			Confidence: req.Output.Composite,
			Scores:     scores,
			Bundle:     scores,
		}, nil
	}

	return DecisionEnvelope{
		Outcome:    OutcomeAbstain,
		Confidence: req.Output.Composite,
		Scores:     scores,
	}, nil
}

// survivedScores returns candidates with fate "survived" from the scored set.
// If none survived, all candidates are returned.
func survivedScores(scores []contextevaluator.CandidateScore) []contextevaluator.CandidateScore {
	var survived []contextevaluator.CandidateScore
	for _, s := range scores {
		if s.Fate == "survived" {
			survived = append(survived, s)
		}
	}
	if len(survived) == 0 {
		return scores
	}
	return survived
}

// topScoredCandidate returns a single-element slice containing the
// highest-composite survived candidate. If no candidates survived, the
// highest-composite candidate overall is returned.
func topScoredCandidate(scores []contextevaluator.CandidateScore) []contextevaluator.CandidateScore {
	if len(scores) == 0 {
		return nil
	}
	candidates := survivedScores(scores)
	best := candidates[0]
	for _, s := range candidates[1:] {
		if s.Composite > best.Composite {
			best = s
		}
	}
	return []contextevaluator.CandidateScore{best}
}
