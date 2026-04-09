package retrieval

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/guillaume-galp/cge/internal/infra/kuzu"
	"github.com/guillaume-galp/cge/internal/infra/repo"
	"github.com/guillaume-galp/cge/internal/infra/textindex"
)

type GraphReader interface {
	ReadGraph(ctx context.Context, workspace repo.Workspace) (kuzu.Graph, error)
}

// ConfidenceEvaluator scores retrieval results for entity confidence.
// Implemented by contextevaluator.Evaluator; defined here as an interface to
// avoid an import cycle (contextevaluator already imports retrieval).
type ConfidenceEvaluator interface {
	EvaluateConfidence(task string, results []Result) []ConfidenceScore
}

// ConfidenceScore holds an evaluator's verdict for a single entity.
type ConfidenceScore struct {
	EntityID  string
	Composite float64
	Fate      string
	Stale     bool
}

type Engine struct {
	graphReader GraphReader
	index       *textindex.Manager
	evaluator   ConfidenceEvaluator
}

var (
	dependentsIntentPattern   = regexp.MustCompile(`\bwhat\s+depends?\s+(?:on|upon)\b`)
	dependenciesIntentPattern = regexp.MustCompile(`\bwhat\s+does\s+.+\s+depend\s+(?:on|upon)\b`)
)

type ResultSet struct {
	IndexStatus string   `json:"index_status"`
	Results     []Result `json:"results"`
}

type Result struct {
	Rank         int            `json:"rank"`
	Score        float64        `json:"score"`
	Scores       ScoreBreakdown `json:"scores"`
	Entity       Entity         `json:"entity"`
	MatchedTerms []string       `json:"matched_terms,omitempty"`
	GraphRefs    []GraphRef     `json:"graph_refs,omitempty"`
	Provenance   Provenance     `json:"provenance"`
}

type ScoreBreakdown struct {
	Text       float64 `json:"text"`
	Structural float64 `json:"structural"`
	// EvaluatorComposite is the evaluator's composite confidence score for this
	// entity. Present only when evaluator-based down-ranking is active.
	EvaluatorComposite float64 `json:"evaluator_composite,omitempty"`
	// EvaluatorFate is the evaluator's verdict ("survived", "trimmed", "rejected").
	// Empty when evaluator-based down-ranking is inactive.
	EvaluatorFate string `json:"evaluator_fate,omitempty"`
	// DownRanked is true when evaluator scoring deprioritized this entity.
	DownRanked bool `json:"down_ranked,omitempty"`
	// DownRankNote explains the reason for down-ranking.
	DownRankNote string `json:"down_rank_note,omitempty"`
}

type Entity struct {
	ID       string         `json:"id"`
	Kind     string         `json:"kind"`
	Title    string         `json:"title,omitempty"`
	Summary  string         `json:"summary,omitempty"`
	Content  string         `json:"content,omitempty"`
	RepoPath string         `json:"repo_path,omitempty"`
	Language string         `json:"language,omitempty"`
	Tags     []string       `json:"tags,omitempty"`
	Props    map[string]any `json:"props,omitempty"`
}

type Provenance struct {
	CreatedAt        string `json:"created_at,omitempty"`
	UpdatedAt        string `json:"updated_at,omitempty"`
	CreatedBy        string `json:"created_by,omitempty"`
	UpdatedBy        string `json:"updated_by,omitempty"`
	CreatedSessionID string `json:"created_session_id,omitempty"`
	UpdatedSessionID string `json:"updated_session_id,omitempty"`
}

type GraphRef struct {
	From       string     `json:"from"`
	To         string     `json:"to"`
	Kind       string     `json:"kind"`
	Direction  string     `json:"direction"`
	Provenance Provenance `json:"provenance"`
}

func NewEngine(graphReader GraphReader, index *textindex.Manager) *Engine {
	if graphReader == nil {
		graphReader = kuzu.NewStore()
	}
	if index == nil {
		index = textindex.NewManager()
	}
	return &Engine{graphReader: graphReader, index: index}
}

// WithEvaluator attaches a ConfidenceEvaluator to the Engine. When set, Query
// applies evaluator-based confidence penalties to down-rank stale or low-quality
// entities in the results.
func (e *Engine) WithEvaluator(ev ConfidenceEvaluator) *Engine {
	e.evaluator = ev
	return e
}

func (e *Engine) Query(ctx context.Context, workspace repo.Workspace, task string) (ResultSet, error) {
	if e == nil {
		e = NewEngine(nil, nil)
	}

	graph, err := e.graphReader.ReadGraph(ctx, workspace)
	if err != nil {
		return ResultSet{}, fmt.Errorf("load graph for query: %w", err)
	}

	docs := documentsFromGraph(graph)
	graphAnchor := textindex.DocumentAnchor(docs)
	indexStatus := "ready"

	index, err := e.index.Load(workspace)
	if err != nil {
		var indexErr *textindex.Error
		if errors.As(err, &indexErr) {
			switch indexErr.Code {
			case "text_index_missing":
				var buildErr error
				index, _, buildErr = e.index.Build(workspace, docs, graphAnchor)
				if buildErr != nil {
					return ResultSet{}, fmt.Errorf("rebuild text index: %w", buildErr)
				}
				indexStatus = "rebuilt"
			default:
				return ResultSet{}, indexErr
			}
		} else {
			return ResultSet{}, err
		}
	}

	if index.GraphAnchor != graphAnchor {
		var buildErr error
		index, _, buildErr = e.index.Build(workspace, docs, graphAnchor)
		if buildErr != nil {
			return ResultSet{}, fmt.Errorf("refresh stale text index: %w", buildErr)
		}
		indexStatus = "rebuilt"
	}

	textResults := e.index.Search(index, task)
	results := mergeResults(task, graph, textResults)

	if e.evaluator != nil {
		results = e.applyEvaluatorDownRanking(ctx, workspace, task, results)
	}

	for i := range results {
		results[i].Rank = i + 1
	}

	return ResultSet{IndexStatus: indexStatus, Results: results}, nil
}

type candidate struct {
	entity       kuzu.EntityRecord
	textScore    float64
	structural   float64
	matchedTerms map[string]struct{}
	graphRefs    map[string]GraphRef
}

func mergeResults(task string, graph kuzu.Graph, textResults []textindex.SearchResult) []Result {
	entities := map[string]kuzu.EntityRecord{}
	for _, node := range graph.Nodes {
		entities[node.ID] = node
	}

	candidates := map[string]*candidate{}
	seedSet := map[string]struct{}{}
	for _, textResult := range textResults {
		entity, ok := entities[textResult.DocumentID]
		if !ok {
			continue
		}
		cand := ensureCandidate(candidates, entity)
		cand.textScore += textResult.Score
		for _, term := range textResult.MatchedTerms {
			cand.matchedTerms[term] = struct{}{}
		}
		seedSet[textResult.DocumentID] = struct{}{}
	}

	intent := detectDependencyIntent(task)
	for _, edge := range graph.Edges {
		_, fromSeed := seedSet[edge.From]
		_, toSeed := seedSet[edge.To]

		switch {
		case intent == dependencyIntentDependentsOfMatch && strings.EqualFold(edge.Kind, "DEPENDS_ON") && toSeed:
			if entity, ok := entities[edge.From]; ok {
				cand := ensureCandidate(candidates, entity)
				cand.structural += StructuralContributionForDirection("supports_task")
				cand.graphRefs[edgeKey(edge)] = graphRef(edge, "supports_task")
			}
			if entity, ok := entities[edge.To]; ok {
				cand := ensureCandidate(candidates, entity)
				cand.graphRefs[edgeKey(edge)] = graphRef(edge, "matched_seed")
			}
		case intent == dependencyIntentDependenciesOfMatch && strings.EqualFold(edge.Kind, "DEPENDS_ON") && fromSeed:
			if entity, ok := entities[edge.To]; ok {
				cand := ensureCandidate(candidates, entity)
				cand.structural += StructuralContributionForDirection("supports_task")
				cand.graphRefs[edgeKey(edge)] = graphRef(edge, "supports_task")
			}
			if entity, ok := entities[edge.From]; ok {
				cand := ensureCandidate(candidates, entity)
				cand.graphRefs[edgeKey(edge)] = graphRef(edge, "matched_seed")
			}
		case fromSeed:
			if entity, ok := entities[edge.To]; ok {
				cand := ensureCandidate(candidates, entity)
				cand.structural += StructuralContributionForDirection("outgoing_neighbor")
				cand.graphRefs[edgeKey(edge)] = graphRef(edge, "outgoing_neighbor")
			}
		case toSeed:
			if entity, ok := entities[edge.From]; ok {
				cand := ensureCandidate(candidates, entity)
				cand.structural += StructuralContributionForDirection("incoming_neighbor")
				cand.graphRefs[edgeKey(edge)] = graphRef(edge, "incoming_neighbor")
			}
		}
	}

	results := make([]Result, 0, len(candidates))
	for _, cand := range candidates {
		matchedTerms := make([]string, 0, len(cand.matchedTerms))
		for term := range cand.matchedTerms {
			matchedTerms = append(matchedTerms, term)
		}
		sort.Strings(matchedTerms)

		refs := make([]GraphRef, 0, len(cand.graphRefs))
		for _, ref := range cand.graphRefs {
			refs = append(refs, ref)
		}
		sort.Slice(refs, func(i, j int) bool {
			if refs[i].Kind != refs[j].Kind {
				return refs[i].Kind < refs[j].Kind
			}
			if refs[i].From != refs[j].From {
				return refs[i].From < refs[j].From
			}
			return refs[i].To < refs[j].To
		})

		results = append(results, Result{
			Score: cand.textScore + cand.structural,
			Scores: ScoreBreakdown{
				Text:       cand.textScore,
				Structural: cand.structural,
			},
			Entity: Entity{
				ID:       cand.entity.ID,
				Kind:     cand.entity.Kind,
				Title:    cand.entity.Title,
				Summary:  cand.entity.Summary,
				Content:  cand.entity.Content,
				RepoPath: cand.entity.RepoPath,
				Language: cand.entity.Language,
				Tags:     cloneStrings(cand.entity.Tags),
				Props:    cloneMap(cand.entity.Props),
			},
			MatchedTerms: matchedTerms,
			GraphRefs:    refs,
			Provenance:   provenanceFromEntity(cand.entity),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Entity.ID < results[j].Entity.ID
		}
		return results[i].Score > results[j].Score
	})

	return results
}

type dependencyIntent int

const (
	dependencyIntentNone dependencyIntent = iota
	dependencyIntentDependentsOfMatch
	dependencyIntentDependenciesOfMatch
)

func StructuralContributionForDirection(direction string) float64 {
	switch direction {
	case "supports_task":
		return 20
	case "outgoing_neighbor", "incoming_neighbor":
		return 2
	default:
		return 0
	}
}

func detectDependencyIntent(task string) dependencyIntent {
	normalized := strings.ToLower(strings.TrimSpace(task))
	switch {
	case dependenciesIntentPattern.MatchString(normalized):
		return dependencyIntentDependenciesOfMatch
	case dependentsIntentPattern.MatchString(normalized):
		return dependencyIntentDependentsOfMatch
	default:
		return dependencyIntentNone
	}
}

func documentsFromGraph(graph kuzu.Graph) []textindex.Document {
	docs := make([]textindex.Document, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		docs = append(docs, textindex.Document{
			ID:       node.ID,
			Kind:     node.Kind,
			Title:    node.Title,
			Summary:  node.Summary,
			Content:  node.Content,
			RepoPath: node.RepoPath,
			Tags:     cloneStrings(node.Tags),
			Aliases:  aliasesFromProps(node.Props),
		})
	}
	return docs
}

func aliasesFromProps(props map[string]any) []string {
	if len(props) == 0 {
		return nil
	}

	aliases := []string{}
	switch raw := props["aliases"].(type) {
	case []string:
		aliases = append(aliases, raw...)
	case []any:
		for _, item := range raw {
			if value, ok := item.(string); ok && strings.TrimSpace(value) != "" {
				aliases = append(aliases, value)
			}
		}
	case string:
		if strings.TrimSpace(raw) != "" {
			aliases = append(aliases, raw)
		}
	}
	if alias, ok := props["alias"].(string); ok && strings.TrimSpace(alias) != "" {
		aliases = append(aliases, alias)
	}
	if len(aliases) == 0 {
		return nil
	}
	sort.Strings(aliases)
	return aliases
}

func ensureCandidate(candidates map[string]*candidate, entity kuzu.EntityRecord) *candidate {
	if candidates[entity.ID] == nil {
		candidates[entity.ID] = &candidate{
			entity:       entity,
			matchedTerms: map[string]struct{}{},
			graphRefs:    map[string]GraphRef{},
		}
	}
	return candidates[entity.ID]
}

func graphRef(edge kuzu.RelationRecord, direction string) GraphRef {
	return GraphRef{
		From:      edge.From,
		To:        edge.To,
		Kind:      edge.Kind,
		Direction: direction,
		Provenance: Provenance{
			CreatedAt:        edge.CreatedAt,
			UpdatedAt:        edge.UpdatedAt,
			CreatedBy:        edge.CreatedBy,
			UpdatedBy:        edge.UpdatedBy,
			CreatedSessionID: edge.CreatedSessionID,
			UpdatedSessionID: edge.UpdatedSessionID,
		},
	}
}

func provenanceFromEntity(entity kuzu.EntityRecord) Provenance {
	return Provenance{
		CreatedAt:        entity.CreatedAt,
		UpdatedAt:        entity.UpdatedAt,
		CreatedBy:        entity.CreatedBy,
		UpdatedBy:        entity.UpdatedBy,
		CreatedSessionID: entity.CreatedSessionID,
		UpdatedSessionID: entity.UpdatedSessionID,
	}
}

func edgeKey(edge kuzu.RelationRecord) string {
	return edge.From + "\x00" + edge.Kind + "\x00" + edge.To
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

// applyEvaluatorDownRanking evaluates each result's entity for confidence and
// applies a bounded score penalty to stale or low-quality entities, then
// re-sorts the slice by the adjusted score. The original text+structural scores
// are preserved in ScoreBreakdown; only the top-level Score is adjusted.
//
// Penalty factors by evaluator fate:
//   - "survived": 1.0 (no change)
//   - "trimmed": 0.7
//   - "rejected": 0.4
func (e *Engine) applyEvaluatorDownRanking(_ context.Context, _ repo.Workspace, task string, results []Result) []Result {
	if len(results) == 0 || e.evaluator == nil {
		return results
	}

	scores := e.evaluator.EvaluateConfidence(task, results)

	scoreByID := make(map[string]ConfidenceScore, len(scores))
	for _, cs := range scores {
		scoreByID[cs.EntityID] = cs
	}

	for i := range results {
		cs, ok := scoreByID[results[i].Entity.ID]
		if !ok {
			continue
		}

		factor := penaltyFactor(cs.Fate)
		results[i].Scores.EvaluatorComposite = cs.Composite
		results[i].Scores.EvaluatorFate = cs.Fate

		if factor < 1.0 {
			results[i].Score = results[i].Score * factor
			results[i].Scores.DownRanked = true
			results[i].Scores.DownRankNote = downRankNote(cs)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Entity.ID < results[j].Entity.ID
		}
		return results[i].Score > results[j].Score
	})

	return results
}

// penaltyFactor returns the score multiplier for a given evaluator fate.
func penaltyFactor(fate string) float64 {
	switch fate {
	case "trimmed":
		return 0.7
	case "rejected":
		return 0.4
	default:
		return 1.0
	}
}

// downRankNote returns a human-readable explanation for a down-ranked entity.
func downRankNote(cs ConfidenceScore) string {
	if cs.Stale {
		return fmt.Sprintf("stale entity (composite=%.2f, fate=%s)", cs.Composite, cs.Fate)
	}
	return fmt.Sprintf("low confidence (composite=%.2f, fate=%s)", cs.Composite, cs.Fate)
}

func cloneMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
