package contextprojector

import (
	"testing"

	"github.com/guillaume-galp/cge/internal/app/retrieval"
)

func TestProjectPreservesRelationshipsAndProvenanceAheadOfSummaryUnderTightBudget(t *testing.T) {
	projector := NewProjector()
	result := retrieval.Result{
		Rank:  1,
		Score: 42,
		Entity: retrieval.Entity{
			ID:      "service:login-api",
			Kind:    "Service",
			Title:   "Login API",
			Summary: "accepts login requests and depends on the authentication subsystem for every protected request path",
		},
		GraphRefs: []retrieval.GraphRef{{
			From:      "service:login-api",
			To:        "component:authentication",
			Kind:      "DEPENDS_ON",
			Direction: "supports_task",
		}},
		Provenance: retrieval.Provenance{
			CreatedBy:        "developer",
			CreatedSessionID: "sess-42",
			CreatedAt:        "2026-03-21T14:00:00Z",
		},
	}

	budget := estimateTokens(Result{
		Rank:  result.Rank,
		Score: result.Score,
		Entity: Entity{
			ID:    result.Entity.ID,
			Kind:  result.Entity.Kind,
			Title: result.Entity.Title,
		},
		Relationships: []Relationship{{
			Kind:      "DEPENDS_ON",
			Direction: "supports_task",
			Peer:      "component:authentication",
		}},
		Provenance: &retrieval.Provenance{
			CreatedBy:        "developer",
			CreatedSessionID: "sess-42",
			CreatedAt:        "2026-03-21T14:00:00Z",
		},
	})

	if tokensWithSummary := estimateTokens(Result{
		Rank:  result.Rank,
		Score: result.Score,
		Entity: Entity{
			ID:      result.Entity.ID,
			Kind:    result.Entity.Kind,
			Title:   result.Entity.Title,
			Summary: result.Entity.Summary,
		},
		Relationships: []Relationship{{
			Kind:      "DEPENDS_ON",
			Direction: "supports_task",
			Peer:      "component:authentication",
		}},
		Provenance: &retrieval.Provenance{
			CreatedBy:        "developer",
			CreatedSessionID: "sess-42",
			CreatedAt:        "2026-03-21T14:00:00Z",
		},
	}); tokensWithSummary <= budget {
		t.Fatalf("test setup invalid: full summary should exceed tight budget, tokens=%d budget=%d", tokensWithSummary, budget)
	}

	envelope, err := projector.Project(retrieval.ResultSet{Results: []retrieval.Result{result}}, budget)
	if err != nil {
		t.Fatalf("Project returned error: %v", err)
	}
	if len(envelope.Results) != 1 {
		t.Fatalf("results = %d, want 1", len(envelope.Results))
	}

	projected := envelope.Results[0]
	if len(projected.Relationships) != 1 {
		t.Fatalf("relationships = %#v, want preserved critical relationship", projected.Relationships)
	}
	if projected.Provenance == nil || projected.Provenance.CreatedBy != "developer" {
		t.Fatalf("provenance = %#v, want preserved provenance", projected.Provenance)
	}
	if projected.Entity.Summary != "" {
		t.Fatalf("summary = %q, want omitted when it competes with relationship/provenance", projected.Entity.Summary)
	}
	if envelope.EstimatedTokens > envelope.MaxTokens {
		t.Fatalf("estimated_tokens = %d, want <= %d", envelope.EstimatedTokens, envelope.MaxTokens)
	}
}

func TestProjectDoesNotEmitLowerPriorityRelationshipAfterHigherPrioritySkipped(t *testing.T) {
	projector := NewProjector()
	result := retrieval.Result{
		Rank:  1,
		Score: 21,
		Entity: retrieval.Entity{
			ID:    "service:gateway",
			Kind:  "Service",
			Title: "Gateway",
		},
		GraphRefs: []retrieval.GraphRef{
			{
				From:      "service:gateway",
				To:        "component:critical-dependency-with-many-segments",
				Kind:      "DEPENDS_ON",
				Direction: "supports_task",
			},
			{
				From:      "component:x",
				To:        "service:gateway",
				Kind:      "RELATED_TO",
				Direction: "incoming_neighbor",
			},
		},
	}

	budget := estimateTokens(Result{
		Rank:  result.Rank,
		Score: result.Score,
		Entity: Entity{
			ID:    result.Entity.ID,
			Kind:  result.Entity.Kind,
			Title: result.Entity.Title,
		},
		Relationships: []Relationship{{
			Kind:      "RELATED_TO",
			Direction: "incoming_neighbor",
			Peer:      "component:x",
		}},
	})

	if highPriorityTokens := estimateTokens(Result{
		Rank:  result.Rank,
		Score: result.Score,
		Entity: Entity{
			ID:    result.Entity.ID,
			Kind:  result.Entity.Kind,
			Title: result.Entity.Title,
		},
		Relationships: []Relationship{{
			Kind:      "DEPENDS_ON",
			Direction: "supports_task",
			Peer:      "component:critical-dependency-with-many-segments",
		}},
	}); highPriorityTokens <= budget {
		t.Fatalf("test setup invalid: higher-priority relationship should exceed budget, tokens=%d budget=%d", highPriorityTokens, budget)
	}

	envelope, err := projector.Project(retrieval.ResultSet{Results: []retrieval.Result{result}}, budget)
	if err != nil {
		t.Fatalf("Project returned error: %v", err)
	}
	if len(envelope.Results) != 1 {
		t.Fatalf("results = %d, want 1", len(envelope.Results))
	}

	projected := envelope.Results[0]
	if len(projected.Relationships) != 0 {
		t.Fatalf("relationships = %#v, want none because lower-priority detail must not leapfrog skipped higher-priority detail", projected.Relationships)
	}
	if projected.OmittedRelationships != 2 {
		t.Fatalf("omitted_relationships = %d, want 2", projected.OmittedRelationships)
	}
}
