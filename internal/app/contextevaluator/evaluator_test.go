package contextevaluator

import (
	"slices"
	"testing"

	"github.com/guillaume-galp/cge/internal/app/retrieval"
)

func TestEvaluateScoresCandidatesAndPreservesInputOrder(t *testing.T) {
	t.Parallel()

	evaluator := NewEvaluator(Config{})
	result := evaluator.Evaluate(EvaluateRequest{
		Task: "implement a new cli command structure for graph context output",
		Candidates: []Candidate{
			{
				ID:       "cmd:graph-context",
				Kind:     "Command",
				Title:    "CLI command structure for graph context",
				Summary:  "Describes command flags, output handling, and context command wiring",
				Content:  "The command structure defines cobra usage, flags, and graph context output behavior.",
				RepoPath: "internal/app/contextcmd/command.go",
				Tags:     []string{"cli", "command", "context"},
			},
			{
				ID:       "report:quarterly",
				Kind:     "Report",
				Title:    "Quarterly finance reporting",
				Summary:  "Summarizes payment reconciliation and reporting metrics",
				RepoPath: "internal/app/reporting/report.go",
				Tags:     []string{"finance", "reporting"},
			},
		},
	})

	if len(result.Scores) != 2 {
		t.Fatalf("score count = %d, want 2", len(result.Scores))
	}
	if result.Scores[0].CandidateID != "cmd:graph-context" || result.Scores[1].CandidateID != "report:quarterly" {
		t.Fatalf("candidate order = [%s, %s], want input order", result.Scores[0].CandidateID, result.Scores[1].CandidateID)
	}
	if got := result.Scores[0].Scores.Relevance; got <= 0.7 {
		t.Fatalf("relevance = %.3f, want > 0.7", got)
	}
	if got := result.Scores[1].Scores.Relevance; got >= 0.3 {
		t.Fatalf("irrelevant relevance = %.3f, want < 0.3", got)
	}
	for _, score := range result.Scores {
		for name, value := range map[string]float64{
			"relevance":   score.Scores.Relevance,
			"consistency": score.Scores.Consistency,
			"usefulness":  score.Scores.Usefulness,
			"composite":   score.Composite,
		} {
			if value < 0 || value > 1 {
				t.Fatalf("%s = %.3f, want normalized float", name, value)
			}
		}
	}
}

func TestEvaluateFlagsContradictingGraphState(t *testing.T) {
	t.Parallel()

	evaluator := NewEvaluator(Config{})
	result := evaluator.Evaluate(EvaluateRequest{
		Task: "stabilize the authentication subsystem for login requests",
		Candidates: []Candidate{{
			ID:      "component:authentication",
			Kind:    "Subsystem",
			Title:   "Authentication subsystem",
			Summary: "Authentication is disabled for protected requests",
			Content: "The current authentication subsystem is disabled and unavailable to login requests.",
			Tags:    []string{"auth", "security"},
			Provenance: retrieval.Provenance{
				UpdatedAt: "2026-04-01T10:00:00Z",
			},
		}},
		GraphState: []GraphState{{
			EntityID: "component:authentication",
			Kind:     "Subsystem",
			Title:    "Authentication subsystem",
			Summary:  "Authentication is active for protected requests",
			Content:  "The current authentication subsystem is active and available to login requests.",
			Tags:     []string{"auth", "security"},
		}},
	})

	score := result.Scores[0]
	if score.Scores.Consistency >= 0.4 {
		t.Fatalf("consistency = %.3f, want < 0.4", score.Scores.Consistency)
	}
	if !slices.Contains(score.Metadata.ConflictingGraphEntities, "component:authentication") {
		t.Fatalf("graph conflicts = %#v, want component:authentication", score.Metadata.ConflictingGraphEntities)
	}
}

func TestCompositeConfidenceUsesConfiguredWeights(t *testing.T) {
	t.Parallel()

	defaultEvaluator := NewEvaluator(Config{})
	scores := DimensionScores{Relevance: 0.9, Consistency: 0.6, Usefulness: 0.8}
	if got := roundScore(defaultEvaluator.CompositeConfidence(scores)); got != 0.767 {
		t.Fatalf("default composite = %.3f, want 0.767", got)
	}

	customEvaluator := NewEvaluator(Config{
		Weights: DimensionWeights{
			Relevance:   0.5,
			Consistency: 0.3,
			Usefulness:  0.2,
		},
	})
	if got := roundScore(customEvaluator.CompositeConfidence(scores)); got != 0.79 {
		t.Fatalf("custom composite = %.3f, want 0.79", got)
	}
}

func TestAggregateConfidenceUsesSurvivors(t *testing.T) {
	t.Parallel()

	evaluator := NewEvaluator(Config{})
	aggregate := evaluator.AggregateConfidence([]CandidateScore{
		{CandidateID: "a", Composite: 0.85, Fate: fateSurvived},
		{CandidateID: "b", Composite: 0.51, Fate: fateTrimmed},
		{CandidateID: "c", Composite: 0.12, Fate: fateRejected},
	})
	if aggregate != 0.85 {
		t.Fatalf("aggregate = %.3f, want 0.85", aggregate)
	}
}

func TestEvaluateOutputDetectsRegression(t *testing.T) {
	t.Parallel()

	evaluator := NewEvaluator(Config{})
	result := evaluator.EvaluateOutput(EvaluateOutputRequest{
		Task: "update the auth middleware and add tests for the login flow",
		Candidate: OutputCandidate{
			ID:      "candidate-2",
			Summary: "Updates auth middleware",
			Content: "Changed auth middleware only.",
		},
		PriorOutput: &OutputCandidate{
			ID:      "candidate-1",
			Summary: "Updates auth middleware and tests",
			Content: "Changed auth middleware and added login flow tests with coverage for failure cases.",
		},
	})

	if result.Baseline == nil {
		t.Fatal("expected baseline comparison")
	}
	if !result.Baseline.RegressionDetected {
		t.Fatalf("regression detected = false, want true")
	}
	if result.Composite >= result.Baseline.PriorComposite {
		t.Fatalf("composite = %.3f, prior = %.3f, want regression", result.Composite, result.Baseline.PriorComposite)
	}
}

func TestEvaluateOutputReturnsZeroForEmptyCandidate(t *testing.T) {
	t.Parallel()

	evaluator := NewEvaluator(Config{})
	result := evaluator.EvaluateOutput(EvaluateOutputRequest{
		Task: "implement multiple requirements",
		Candidate: OutputCandidate{
			ID: "empty",
		},
	})

	if result.Scores != (DimensionScores{}) {
		t.Fatalf("scores = %#v, want zero values", result.Scores)
	}
	if result.Composite != 0 {
		t.Fatalf("composite = %.3f, want 0", result.Composite)
	}
}

func TestEvaluateReturnsEmptyBundleWithoutError(t *testing.T) {
	t.Parallel()

	evaluator := NewEvaluator(Config{})
	result := evaluator.Evaluate(EvaluateRequest{Task: "anything"})

	if result.CandidateCount != 0 {
		t.Fatalf("candidate count = %d, want 0", result.CandidateCount)
	}
	if len(result.Scores) != 0 {
		t.Fatalf("scores = %d, want 0", len(result.Scores))
	}
	if result.AggregateConfidence != 0 {
		t.Fatalf("aggregate confidence = %.3f, want 0", result.AggregateConfidence)
	}
}
