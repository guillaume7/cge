package retrieval

import (
	"sort"

	"github.com/guillaume-galp/cge/internal/infra/textindex"
)

type ExplanationSet struct {
	QueryTerms []string          `json:"query_terms,omitempty"`
	Results    []ExplainedResult `json:"results"`
}

type ExplainedResult struct {
	Rank           int             `json:"rank"`
	Score          float64         `json:"score"`
	Scores         ScoreBreakdown  `json:"scores"`
	Entity         Entity          `json:"entity"`
	TextMatches    []TextMatch     `json:"text_matches,omitempty"`
	GraphPaths     []GraphPath     `json:"graph_paths,omitempty"`
	Provenance     ProvenanceTrace `json:"provenance"`
	RankingReasons []RankingReason `json:"ranking_reasons,omitempty"`
}

type TextMatch struct {
	Terms        []string `json:"terms"`
	Contribution float64  `json:"contribution"`
}

type GraphPath struct {
	Role         string     `json:"role"`
	Contribution float64    `json:"contribution,omitempty"`
	Steps        []GraphRef `json:"steps"`
}

type ProvenanceTrace struct {
	Entity        Provenance `json:"entity"`
	Relationships []GraphRef `json:"relationships,omitempty"`
}

type RankingReason struct {
	Type         string         `json:"type"`
	Contribution float64        `json:"contribution,omitempty"`
	Details      map[string]any `json:"details,omitempty"`
}

func BuildExplanation(task string, resultSet ResultSet) ExplanationSet {
	explanation := ExplanationSet{
		QueryTerms: textindex.AnalyzeQueryTerms(task),
		Results:    make([]ExplainedResult, 0, len(resultSet.Results)),
	}

	for _, result := range resultSet.Results {
		explanation.Results = append(explanation.Results, explainResult(result))
	}

	return explanation
}

func explainResult(result Result) ExplainedResult {
	explained := ExplainedResult{
		Rank:   result.Rank,
		Score:  result.Score,
		Scores: result.Scores,
		Entity: cloneEntity(result.Entity),
		Provenance: ProvenanceTrace{
			Entity:        result.Provenance,
			Relationships: cloneGraphRefs(result.GraphRefs),
		},
	}

	if len(result.MatchedTerms) > 0 || result.Scores.Text > 0 {
		explained.TextMatches = []TextMatch{{
			Terms:        cloneStrings(result.MatchedTerms),
			Contribution: result.Scores.Text,
		}}
		explained.RankingReasons = append(explained.RankingReasons, RankingReason{
			Type:         "text_match",
			Contribution: result.Scores.Text,
			Details: map[string]any{
				"matched_terms": cloneStrings(result.MatchedTerms),
			},
		})
	}

	graphPaths := make([]GraphPath, 0, len(result.GraphRefs))
	structuralByRole := map[string]float64{}
	structuralKindsByRole := map[string]map[string]struct{}{}
	structuralCountByRole := map[string]int{}
	for _, ref := range result.GraphRefs {
		contribution := StructuralContributionForDirection(ref.Direction)
		graphPaths = append(graphPaths, GraphPath{
			Role:         ref.Direction,
			Contribution: contribution,
			Steps:        []GraphRef{cloneGraphRef(ref)},
		})
		if contribution <= 0 {
			continue
		}
		structuralByRole[ref.Direction] += contribution
		if structuralKindsByRole[ref.Direction] == nil {
			structuralKindsByRole[ref.Direction] = map[string]struct{}{}
		}
		structuralKindsByRole[ref.Direction][ref.Kind] = struct{}{}
		structuralCountByRole[ref.Direction]++
	}
	if len(graphPaths) > 0 {
		explained.GraphPaths = graphPaths
	}

	roles := make([]string, 0, len(structuralByRole))
	for role := range structuralByRole {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	for _, role := range roles {
		explained.RankingReasons = append(explained.RankingReasons, RankingReason{
			Type:         "graph_path",
			Contribution: structuralByRole[role],
			Details: map[string]any{
				"role":       role,
				"path_count": structuralCountByRole[role],
				"edge_kinds": sortedKeys(structuralKindsByRole[role]),
			},
		})
	}

	explained.RankingReasons = append(explained.RankingReasons, RankingReason{
		Type: "score_breakdown",
		Details: map[string]any{
			"text":       result.Scores.Text,
			"structural": result.Scores.Structural,
			"total":      result.Score,
		},
	})

	return explained
}

func cloneEntity(entity Entity) Entity {
	return Entity{
		ID:       entity.ID,
		Kind:     entity.Kind,
		Title:    entity.Title,
		Summary:  entity.Summary,
		Content:  entity.Content,
		RepoPath: entity.RepoPath,
		Language: entity.Language,
		Tags:     cloneStrings(entity.Tags),
		Props:    cloneMap(entity.Props),
	}
}

func cloneGraphRefs(refs []GraphRef) []GraphRef {
	if len(refs) == 0 {
		return nil
	}
	cloned := make([]GraphRef, len(refs))
	for i, ref := range refs {
		cloned[i] = cloneGraphRef(ref)
	}
	return cloned
}

func cloneGraphRef(ref GraphRef) GraphRef {
	return GraphRef{
		From:      ref.From,
		To:        ref.To,
		Kind:      ref.Kind,
		Direction: ref.Direction,
		Provenance: Provenance{
			CreatedAt:        ref.Provenance.CreatedAt,
			UpdatedAt:        ref.Provenance.UpdatedAt,
			CreatedBy:        ref.Provenance.CreatedBy,
			UpdatedBy:        ref.Provenance.UpdatedBy,
			CreatedSessionID: ref.Provenance.CreatedSessionID,
			UpdatedSessionID: ref.Provenance.UpdatedSessionID,
		},
	}
}

func sortedKeys(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for value := range values {
		keys = append(keys, value)
	}
	sort.Strings(keys)
	return keys
}
