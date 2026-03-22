package textindex

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/guillaume-galp/cge/internal/infra/repo"
)

const (
	SchemaVersion = "v1"
	FileName      = "text-index-v1.json"
)

var tokenPattern = regexp.MustCompile(`[a-z0-9]+`)

var stopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "does": {}, "for": {}, "how": {},
	"in": {}, "is": {}, "of": {}, "on": {}, "or": {}, "the": {}, "to": {},
	"what": {}, "which": {}, "who": {}, "with": {},
}

var synonymGroups = [][]string{{"auth", "authentication"}}

type Manager struct{}

type Document struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Title    string   `json:"title,omitempty"`
	Summary  string   `json:"summary,omitempty"`
	Content  string   `json:"content,omitempty"`
	RepoPath string   `json:"repo_path,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Aliases  []string `json:"aliases,omitempty"`
}

type Index struct {
	SchemaVersion string               `json:"schema_version"`
	GraphAnchor   string               `json:"graph_anchor"`
	BuiltAt       string               `json:"built_at"`
	DocumentCount int                  `json:"document_count"`
	Documents     []documentDescriptor `json:"documents"`
	Inverted      []termPostings       `json:"inverted"`
}

type documentDescriptor struct {
	ID string `json:"id"`
}

type termPostings struct {
	Term     string        `json:"term"`
	Postings []termPosting `json:"postings"`
}

type termPosting struct {
	DocumentID string  `json:"document_id"`
	Weight     float64 `json:"weight"`
}

type BuildSummary struct {
	DocumentCount int    `json:"document_count"`
	GraphAnchor   string `json:"graph_anchor"`
}

type SearchResult struct {
	DocumentID   string   `json:"document_id"`
	Score        float64  `json:"score"`
	MatchedTerms []string `json:"matched_terms,omitempty"`
}

type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func NewManager() *Manager {
	return &Manager{}
}

func IndexPath(workspace repo.Workspace) string {
	return filepath.Join(workspace.WorkspacePath, "index", FileName)
}

func (m *Manager) Load(workspace repo.Workspace) (Index, error) {
	payload, err := os.ReadFile(IndexPath(workspace))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Index{}, &Error{
				Code:    "text_index_missing",
				Message: "local text index is missing",
				Details: map[string]any{
					"index_path": IndexPath(workspace),
				},
			}
		}
		return Index{}, fmt.Errorf("read text index: %w", err)
	}

	var index Index
	if err := json.Unmarshal(payload, &index); err != nil {
		return Index{}, &Error{
			Code:    "text_index_corrupt",
			Message: "local text index is unreadable; rebuild is required",
			Details: map[string]any{
				"index_path":   IndexPath(workspace),
				"rebuild_hint": "remove the corrupted index file and rerun graph query to rebuild from persisted graph data",
			},
		}
	}

	if index.SchemaVersion != SchemaVersion {
		return Index{}, &Error{
			Code:    "text_index_corrupt",
			Message: "local text index schema is not supported; rebuild is required",
			Details: map[string]any{
				"index_path":         IndexPath(workspace),
				"schema_version":     index.SchemaVersion,
				"supported_versions": []string{SchemaVersion},
				"rebuild_hint":       "remove the incompatible index file and rerun graph query to rebuild from persisted graph data",
			},
		}
	}

	if index.DocumentCount != len(index.Documents) {
		return Index{}, &Error{
			Code:    "text_index_corrupt",
			Message: "local text index metadata is inconsistent; rebuild is required",
			Details: map[string]any{
				"index_path":   IndexPath(workspace),
				"rebuild_hint": "remove the inconsistent index file and rerun graph query to rebuild from persisted graph data",
			},
		}
	}

	return index, nil
}

func (m *Manager) Build(workspace repo.Workspace, docs []Document, graphAnchor string) (Index, BuildSummary, error) {
	if err := os.MkdirAll(filepath.Dir(IndexPath(workspace)), 0o755); err != nil {
		return Index{}, BuildSummary{}, fmt.Errorf("create text index directory: %w", err)
	}

	prepared := make([]Document, 0, len(docs))
	for _, doc := range docs {
		prepared = append(prepared, normalizeDocument(doc))
	}
	sort.Slice(prepared, func(i, j int) bool { return prepared[i].ID < prepared[j].ID })

	inverted := map[string]map[string]float64{}
	descriptors := make([]documentDescriptor, 0, len(prepared))
	for _, doc := range prepared {
		descriptors = append(descriptors, documentDescriptor{ID: doc.ID})
		weights := weightedTerms(doc)
		for term, weight := range weights {
			postings := inverted[term]
			if postings == nil {
				postings = map[string]float64{}
				inverted[term] = postings
			}
			postings[doc.ID] += weight
		}
	}

	terms := make([]string, 0, len(inverted))
	for term := range inverted {
		terms = append(terms, term)
	}
	sort.Strings(terms)

	persistedTerms := make([]termPostings, 0, len(terms))
	for _, term := range terms {
		docIDs := make([]string, 0, len(inverted[term]))
		for docID := range inverted[term] {
			docIDs = append(docIDs, docID)
		}
		sort.Strings(docIDs)

		postings := make([]termPosting, 0, len(docIDs))
		for _, docID := range docIDs {
			postings = append(postings, termPosting{DocumentID: docID, Weight: inverted[term][docID]})
		}
		persistedTerms = append(persistedTerms, termPostings{Term: term, Postings: postings})
	}

	index := Index{
		SchemaVersion: SchemaVersion,
		GraphAnchor:   graphAnchor,
		BuiltAt:       time.Now().UTC().Format(time.RFC3339),
		DocumentCount: len(prepared),
		Documents:     descriptors,
		Inverted:      persistedTerms,
	}

	payload, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return Index{}, BuildSummary{}, fmt.Errorf("encode text index: %w", err)
	}
	payload = append(payload, '\n')

	tmpPath := filepath.Join(filepath.Dir(IndexPath(workspace)), ".tmp-"+FileName)
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return Index{}, BuildSummary{}, fmt.Errorf("write text index: %w", err)
	}
	if err := os.Rename(tmpPath, IndexPath(workspace)); err != nil {
		_ = os.Remove(tmpPath)
		return Index{}, BuildSummary{}, fmt.Errorf("replace text index: %w", err)
	}

	return index, BuildSummary{DocumentCount: len(prepared), GraphAnchor: graphAnchor}, nil
}

func AnalyzeQueryTerms(query string) []string {
	return analyzedTerms(query)
}

func (m *Manager) Search(index Index, query string) []SearchResult {
	if index.DocumentCount == 0 {
		return nil
	}

	postingsByTerm := make(map[string][]termPosting, len(index.Inverted))
	for _, entry := range index.Inverted {
		postingsByTerm[entry.Term] = entry.Postings
	}

	terms := analyzedTerms(query)
	scores := map[string]float64{}
	matched := map[string]map[string]struct{}{}
	for _, term := range terms {
		postings := postingsByTerm[term]
		if len(postings) == 0 {
			continue
		}
		idf := math.Log(1 + float64(index.DocumentCount)/(1+float64(len(postings))))
		for _, posting := range postings {
			scores[posting.DocumentID] += posting.Weight * idf
			if matched[posting.DocumentID] == nil {
				matched[posting.DocumentID] = map[string]struct{}{}
			}
			matched[posting.DocumentID][term] = struct{}{}
		}
	}

	results := make([]SearchResult, 0, len(scores))
	for docID, score := range scores {
		terms := make([]string, 0, len(matched[docID]))
		for term := range matched[docID] {
			terms = append(terms, term)
		}
		sort.Strings(terms)
		results = append(results, SearchResult{DocumentID: docID, Score: score, MatchedTerms: terms})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].DocumentID < results[j].DocumentID
		}
		return results[i].Score > results[j].Score
	})

	return results
}

func DocumentAnchor(docs []Document) string {
	prepared := make([]Document, 0, len(docs))
	for _, doc := range docs {
		prepared = append(prepared, normalizeDocument(doc))
	}
	sort.Slice(prepared, func(i, j int) bool { return prepared[i].ID < prepared[j].ID })

	payload, err := json.Marshal(prepared)
	if err != nil {
		return ""
	}

	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func normalizeDocument(doc Document) Document {
	return Document{
		ID:       strings.TrimSpace(doc.ID),
		Kind:     strings.TrimSpace(doc.Kind),
		Title:    strings.TrimSpace(doc.Title),
		Summary:  strings.TrimSpace(doc.Summary),
		Content:  strings.TrimSpace(doc.Content),
		RepoPath: strings.TrimSpace(doc.RepoPath),
		Tags:     normalizedStrings(doc.Tags),
		Aliases:  normalizedStrings(doc.Aliases),
	}
}

func weightedTerms(doc Document) map[string]float64 {
	weights := map[string]float64{}
	addTerms(weights, doc.ID, 0.25)
	addTerms(weights, doc.Kind, 0.5)
	addTerms(weights, doc.Title, 4)
	addTerms(weights, doc.Summary, 2.5)
	addTerms(weights, doc.Content, 1)
	addTerms(weights, doc.RepoPath, 0.5)
	for _, tag := range doc.Tags {
		addTerms(weights, tag, 3)
	}
	for _, alias := range doc.Aliases {
		addTerms(weights, alias, 3.5)
	}
	return weights
}

func addTerms(weights map[string]float64, text string, multiplier float64) {
	for _, term := range analyzedTerms(text) {
		weights[term] += multiplier
	}
}

func analyzedTerms(text string) []string {
	rawTerms := tokenPattern.FindAllString(strings.ToLower(text), -1)
	seen := map[string]struct{}{}
	terms := make([]string, 0, len(rawTerms))
	for _, raw := range rawTerms {
		if _, stop := stopWords[raw]; stop {
			continue
		}
		for _, term := range expandTerm(raw) {
			if _, ok := seen[term]; ok {
				continue
			}
			seen[term] = struct{}{}
			terms = append(terms, term)
		}
	}
	sort.Strings(terms)
	return terms
}

func expandTerm(term string) []string {
	if term == "" {
		return nil
	}
	expanded := map[string]struct{}{term: {}}
	for _, group := range synonymGroups {
		contains := false
		for _, candidate := range group {
			if candidate == term {
				contains = true
				break
			}
		}
		if !contains {
			continue
		}
		for _, candidate := range group {
			expanded[candidate] = struct{}{}
		}
	}
	terms := make([]string, 0, len(expanded))
	for candidate := range expanded {
		terms = append(terms, candidate)
	}
	sort.Strings(terms)
	return terms
}

func normalizedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}
