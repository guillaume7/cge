package contextprojector

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"github.com/guillaume-galp/cge/internal/app/retrieval"
)

var estimateTokenPattern = regexp.MustCompile(`[A-Za-z0-9_]+`)

type ValidationError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type Projector struct{}

type Envelope struct {
	MaxTokens       int      `json:"max_tokens"`
	EstimatedTokens int      `json:"estimated_tokens"`
	Truncated       bool     `json:"truncated"`
	OmittedResults  int      `json:"omitted_results,omitempty"`
	Results         []Result `json:"results"`
}

type Result struct {
	Rank                 int                   `json:"rank"`
	Score                float64               `json:"score"`
	Entity               Entity                `json:"entity"`
	Relationships        []Relationship        `json:"relationships,omitempty"`
	MatchedTerms         []string              `json:"matched_terms,omitempty"`
	Provenance           *retrieval.Provenance `json:"provenance,omitempty"`
	OmittedRelationships int                   `json:"omitted_relationships,omitempty"`
}

type Entity struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Title   string `json:"title,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type Relationship struct {
	Kind      string `json:"kind"`
	Direction string `json:"direction,omitempty"`
	Peer      string `json:"peer,omitempty"`
}

func NewProjector() Projector {
	return Projector{}
}

func ValidateMaxTokens(maxTokens int) error {
	if maxTokens <= 0 {
		return &ValidationError{
			Code:    "invalid_max_tokens",
			Message: "max_tokens must be greater than zero",
			Details: map[string]any{"max_tokens": maxTokens},
		}
	}
	return nil
}

func (p Projector) Project(resultSet retrieval.ResultSet, maxTokens int) (Envelope, error) {
	if err := ValidateMaxTokens(maxTokens); err != nil {
		return Envelope{}, err
	}

	envelope := Envelope{
		MaxTokens: maxTokens,
		Results:   make([]Result, 0, len(resultSet.Results)),
	}

	usedTokens := 0
	for i, result := range resultSet.Results {
		projected, tokens, ok := p.projectResult(result, maxTokens-usedTokens)
		if !ok {
			envelope.Truncated = true
			envelope.OmittedResults = len(resultSet.Results) - i
			break
		}
		envelope.Results = append(envelope.Results, projected)
		usedTokens += tokens
	}

	envelope.EstimatedTokens = usedTokens
	return envelope, nil
}

func (p Projector) projectResult(result retrieval.Result, remaining int) (Result, int, bool) {
	projected := Result{
		Rank:  result.Rank,
		Score: result.Score,
		Entity: Entity{
			ID:    result.Entity.ID,
			Kind:  result.Entity.Kind,
			Title: strings.TrimSpace(result.Entity.Title),
		},
	}

	used := estimateTokens(projected)
	if used > remaining {
		return Result{}, 0, false
	}

	relationships := prioritizeRelationships(result.GraphRefs, result.Entity.ID)
	omittedRelationships := 0
	for index, relationship := range relationships {
		candidate := projected
		candidate.Relationships = append(cloneRelationships(projected.Relationships), relationship)
		candidateTokens := estimateTokens(candidate)
		if candidateTokens > remaining {
			omittedRelationships += len(relationships) - index
			break
		}
		projected = candidate
		used = candidateTokens
	}

	if provenance, tokens, ok := fitProvenance(projected, result.Provenance, remaining); ok {
		projected.Provenance = provenance
		used = tokens
	}

	if summary := strings.TrimSpace(result.Entity.Summary); summary != "" {
		if trimmed, tokens, ok := fitSummary(projected, summary, remaining); ok {
			projected.Entity.Summary = trimmed
			used = tokens
		}
	}

	for _, term := range result.MatchedTerms {
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}
		candidate := projected
		candidate.MatchedTerms = append(cloneStrings(projected.MatchedTerms), term)
		candidateTokens := estimateTokens(candidate)
		if candidateTokens > remaining {
			break
		}
		projected = candidate
		used = candidateTokens
	}

	if omittedRelationships > 0 {
		candidate := projected
		candidate.OmittedRelationships = omittedRelationships
		candidateTokens := estimateTokens(candidate)
		if candidateTokens <= remaining {
			projected = candidate
			used = candidateTokens
		}
	}

	return projected, used, true
}

func fitSummary(base Result, summary string, remaining int) (string, int, bool) {
	summary = strings.Join(strings.Fields(summary), " ")
	candidate := base
	candidate.Entity.Summary = summary
	if tokens := estimateTokens(candidate); tokens <= remaining {
		return summary, tokens, true
	}

	words := strings.Fields(summary)
	if len(words) == 0 {
		return "", 0, false
	}

	bestSummary := ""
	bestTokens := 0
	low, high := 1, len(words)
	for low <= high {
		mid := low + (high-low)/2
		trimmed := strings.Join(words[:mid], " ")
		if mid < len(words) {
			trimmed += "…"
		}
		candidate.Entity.Summary = trimmed
		tokens := estimateTokens(candidate)
		if tokens <= remaining {
			bestSummary = trimmed
			bestTokens = tokens
			low = mid + 1
			continue
		}
		high = mid - 1
	}

	if bestSummary == "" {
		return "", 0, false
	}
	return bestSummary, bestTokens, true
}

func fitProvenance(base Result, source retrieval.Provenance, remaining int) (*retrieval.Provenance, int, bool) {
	candidate := retrieval.Provenance{}
	projected := base
	applied := false

	for _, step := range []struct {
		value string
		apply func(*retrieval.Provenance, string)
	}{
		{value: strings.TrimSpace(source.CreatedBy), apply: func(p *retrieval.Provenance, value string) { p.CreatedBy = value }},
		{value: strings.TrimSpace(source.CreatedSessionID), apply: func(p *retrieval.Provenance, value string) { p.CreatedSessionID = value }},
		{value: strings.TrimSpace(source.UpdatedBy), apply: func(p *retrieval.Provenance, value string) { p.UpdatedBy = value }},
		{value: strings.TrimSpace(source.UpdatedSessionID), apply: func(p *retrieval.Provenance, value string) { p.UpdatedSessionID = value }},
		{value: strings.TrimSpace(source.CreatedAt), apply: func(p *retrieval.Provenance, value string) { p.CreatedAt = value }},
		{value: strings.TrimSpace(source.UpdatedAt), apply: func(p *retrieval.Provenance, value string) { p.UpdatedAt = value }},
	} {
		if step.value == "" {
			continue
		}
		next := candidate
		step.apply(&next, step.value)
		projected.Provenance = &next
		tokens := estimateTokens(projected)
		if tokens > remaining {
			continue
		}
		candidate = next
		applied = true
	}

	if !applied {
		return nil, 0, false
	}

	projected.Provenance = &candidate
	return &candidate, estimateTokens(projected), true
}

func prioritizeRelationships(refs []retrieval.GraphRef, entityID string) []Relationship {
	if len(refs) == 0 {
		return nil
	}

	type rankedRelationship struct {
		relationship Relationship
		priority     int
		from         string
		to           string
	}

	ranked := make([]rankedRelationship, 0, len(refs))
	for _, ref := range refs {
		ranked = append(ranked, rankedRelationship{
			relationship: Relationship{
				Kind:      ref.Kind,
				Direction: ref.Direction,
				Peer:      relationshipPeer(ref, entityID),
			},
			priority: relationshipPriority(ref.Direction),
			from:     ref.From,
			to:       ref.To,
		})
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].priority != ranked[j].priority {
			return ranked[i].priority < ranked[j].priority
		}
		if ranked[i].relationship.Kind != ranked[j].relationship.Kind {
			return ranked[i].relationship.Kind < ranked[j].relationship.Kind
		}
		if ranked[i].from != ranked[j].from {
			return ranked[i].from < ranked[j].from
		}
		return ranked[i].to < ranked[j].to
	})

	projected := make([]Relationship, 0, len(ranked))
	for _, item := range ranked {
		projected = append(projected, item.relationship)
	}
	return projected
}

func relationshipPriority(direction string) int {
	switch direction {
	case "supports_task":
		return 0
	case "matched_seed":
		return 1
	case "outgoing_neighbor":
		return 2
	case "incoming_neighbor":
		return 3
	default:
		return 4
	}
}

func relationshipPeer(ref retrieval.GraphRef, entityID string) string {
	switch {
	case ref.From == entityID:
		return ref.To
	case ref.To == entityID:
		return ref.From
	default:
		return ""
	}
}

func estimateTokens(value any) int {
	payload, err := json.Marshal(value)
	if err != nil {
		return 0
	}
	return len(estimateTokenPattern.FindAll(payload, -1))
}

func cloneRelationships(values []Relationship) []Relationship {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]Relationship, len(values))
	copy(cloned, values)
	return cloned
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}
