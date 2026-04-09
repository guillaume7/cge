package contextevaluator

import (
	"math"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/guillaume-galp/cge/internal/app/retrieval"
	"github.com/guillaume-galp/cge/internal/infra/textindex"
)

const (
	fateSurvived = "survived"
	fateTrimmed  = "trimmed"
	fateRejected = "rejected"
)

var contradictionGroups = []struct {
	positive []string
	negative []string
}{
	{
		positive: []string{"enabled", "enable", "active", "current", "present", "exists", "supported", "healthy", "ready"},
		negative: []string{"disabled", "disable", "inactive", "deprecated", "legacy", "absent", "missing", "removed", "unsupported", "stale", "broken"},
	},
	{
		positive: []string{"allow", "allowed", "allows", "available"},
		negative: []string{"deny", "denied", "denies", "blocked", "forbidden", "unavailable"},
	},
}

var contradictionTokens = buildContradictionTokenSet()

type Evaluator struct {
	config Config
}

type Config struct {
	Weights             DimensionWeights    `json:"weights,omitempty"`
	Thresholds          CandidateThresholds `json:"thresholds,omitempty"`
	StaleAfter          time.Duration       `json:"-"`
	RegressionTolerance float64             `json:"regression_tolerance,omitempty"`
}

type DimensionWeights struct {
	Relevance   float64 `json:"relevance"`
	Consistency float64 `json:"consistency"`
	Usefulness  float64 `json:"usefulness"`
}

type CandidateThresholds struct {
	SurviveComposite   float64 `json:"survive_composite,omitempty"`
	TrimComposite      float64 `json:"trim_composite,omitempty"`
	MinimumRelevance   float64 `json:"minimum_relevance,omitempty"`
	MinimumConsistency float64 `json:"minimum_consistency,omitempty"`
	MinimumUsefulness  float64 `json:"minimum_usefulness,omitempty"`
}

type EvaluateRequest struct {
	Task       string       `json:"task"`
	Candidates []Candidate  `json:"candidates"`
	GraphState []GraphState `json:"graph_state,omitempty"`
}

type Candidate struct {
	ID           string               `json:"id"`
	Kind         string               `json:"kind,omitempty"`
	Title        string               `json:"title,omitempty"`
	Summary      string               `json:"summary,omitempty"`
	Content      string               `json:"content,omitempty"`
	RepoPath     string               `json:"repo_path,omitempty"`
	Tags         []string             `json:"tags,omitempty"`
	MatchedTerms []string             `json:"matched_terms,omitempty"`
	GraphRefs    []retrieval.GraphRef `json:"graph_refs,omitempty"`
	Provenance   retrieval.Provenance `json:"provenance,omitempty"`
}

type GraphState struct {
	EntityID   string               `json:"entity_id"`
	Kind       string               `json:"kind,omitempty"`
	Title      string               `json:"title,omitempty"`
	Summary    string               `json:"summary,omitempty"`
	Content    string               `json:"content,omitempty"`
	RepoPath   string               `json:"repo_path,omitempty"`
	Tags       []string             `json:"tags,omitempty"`
	Provenance retrieval.Provenance `json:"provenance,omitempty"`
}

type DimensionScores struct {
	Relevance   float64 `json:"relevance"`
	Consistency float64 `json:"consistency"`
	Usefulness  float64 `json:"usefulness"`
}

type ScoreMetadata struct {
	MatchedTaskTerms         []string `json:"matched_task_terms,omitempty"`
	StructuralSignals        []string `json:"structural_signals,omitempty"`
	ConflictingGraphEntities []string `json:"conflicting_graph_entities,omitempty"`
	ConflictingCandidates    []string `json:"conflicting_candidates,omitempty"`
	Stale                    bool     `json:"stale,omitempty"`
	AgeDays                  float64  `json:"age_days,omitempty"`
}

type CandidateScore struct {
	CandidateID     string          `json:"candidate_id"`
	Scores          DimensionScores `json:"scores"`
	Composite       float64         `json:"composite"`
	Fate            string          `json:"fate"`
	RejectionReason string          `json:"rejection_reason,omitempty"`
	Metadata        ScoreMetadata   `json:"metadata,omitempty"`
}

type EvaluationResult struct {
	TaskContext         string           `json:"task_context"`
	CandidateCount      int              `json:"candidate_count"`
	Scores              []CandidateScore `json:"scores"`
	AggregateConfidence float64          `json:"aggregate_confidence"`
	Survivors           []string         `json:"survivors,omitempty"`
	Trimmed             []string         `json:"trimmed,omitempty"`
	Rejected            []string         `json:"rejected,omitempty"`
}

type EvaluateOutputRequest struct {
	Task        string           `json:"task"`
	Candidate   OutputCandidate  `json:"candidate"`
	PriorOutput *OutputCandidate `json:"prior_output,omitempty"`
	GraphState  []GraphState     `json:"graph_state,omitempty"`
}

type OutputCandidate struct {
	ID      string `json:"id"`
	Summary string `json:"summary,omitempty"`
	Content string `json:"content,omitempty"`
}

type BaselineComparison struct {
	PriorScores        DimensionScores `json:"prior_scores"`
	PriorComposite     float64         `json:"prior_composite"`
	CompositeDelta     float64         `json:"composite_delta"`
	RegressionDetected bool            `json:"regression_detected"`
	Reasons            []string        `json:"reasons,omitempty"`
}

type OutputEvaluation struct {
	CandidateID string              `json:"candidate_id"`
	Scores      DimensionScores     `json:"scores"`
	Composite   float64             `json:"composite"`
	Metadata    ScoreMetadata       `json:"metadata,omitempty"`
	Baseline    *BaselineComparison `json:"baseline,omitempty"`
}

func NewEvaluator(config Config) Evaluator {
	if config.StaleAfter <= 0 {
		config.StaleAfter = 30 * 24 * time.Hour
	}
	if config.RegressionTolerance <= 0 {
		config.RegressionTolerance = 0.05
	}
	config.Weights = normalizeWeights(config.Weights)
	config.Thresholds = normalizeThresholds(config.Thresholds)
	return Evaluator{config: config}
}

func CandidateFromRetrievalResult(result retrieval.Result) Candidate {
	return Candidate{
		ID:           strings.TrimSpace(result.Entity.ID),
		Kind:         strings.TrimSpace(result.Entity.Kind),
		Title:        strings.TrimSpace(result.Entity.Title),
		Summary:      strings.TrimSpace(result.Entity.Summary),
		Content:      strings.TrimSpace(result.Entity.Content),
		RepoPath:     strings.TrimSpace(result.Entity.RepoPath),
		Tags:         cloneStrings(result.Entity.Tags),
		MatchedTerms: cloneStrings(result.MatchedTerms),
		GraphRefs:    cloneGraphRefs(result.GraphRefs),
		Provenance:   result.Provenance,
	}
}

// EvaluateConfidence implements retrieval.ConfidenceEvaluator. It converts
// retrieval results to evaluator candidates, runs Evaluate, and returns a slim
// ConfidenceScore slice that the retrieval engine can use for down-ranking
// without importing this package.
func (e Evaluator) EvaluateConfidence(task string, results []retrieval.Result) []retrieval.ConfidenceScore {
	candidates := make([]Candidate, 0, len(results))
	for _, r := range results {
		candidates = append(candidates, CandidateFromRetrievalResult(r))
	}

	evalResult := e.Evaluate(EvaluateRequest{Task: task, Candidates: candidates})

	scores := make([]retrieval.ConfidenceScore, 0, len(evalResult.Scores))
	for _, cs := range evalResult.Scores {
		scores = append(scores, retrieval.ConfidenceScore{
			EntityID:  cs.CandidateID,
			Composite: cs.Composite,
			Fate:      cs.Fate,
			Stale:     cs.Metadata.Stale,
		})
	}
	return scores
}

func (e Evaluator) CompositeConfidence(scores DimensionScores) float64 {
	config := ensureConfig(e.config)
	return clamp01(
		scores.Relevance*config.Weights.Relevance +
			scores.Consistency*config.Weights.Consistency +
			scores.Usefulness*config.Weights.Usefulness,
	)
}

func (e Evaluator) AggregateConfidence(scores []CandidateScore) float64 {
	if len(scores) == 0 {
		return 0
	}
	var picked []float64
	for _, score := range scores {
		if score.Fate == fateSurvived {
			picked = append(picked, score.Composite)
		}
	}
	if len(picked) == 0 {
		for _, score := range scores {
			if score.Fate == fateTrimmed {
				picked = append(picked, score.Composite)
			}
		}
	}
	if len(picked) == 0 {
		for _, score := range scores {
			picked = append(picked, score.Composite)
		}
	}
	return clamp01(mean(picked))
}

func (e Evaluator) Evaluate(request EvaluateRequest) EvaluationResult {
	config := ensureConfig(e.config)
	candidates := normalizeCandidates(request.Candidates)
	graphState := normalizeGraphState(request.GraphState)
	task := strings.TrimSpace(request.Task)
	taskTerms := textindex.AnalyzedTerms(task)

	result := EvaluationResult{
		TaskContext:    task,
		CandidateCount: len(candidates),
		Scores:         make([]CandidateScore, 0, len(candidates)),
	}
	if len(candidates) == 0 {
		return result
	}

	for index, candidate := range candidates {
		score := scoreContextCandidate(config, taskTerms, candidate, candidates, graphState, index)
		result.Scores = append(result.Scores, score)
		switch score.Fate {
		case fateSurvived:
			result.Survivors = append(result.Survivors, score.CandidateID)
		case fateTrimmed:
			result.Trimmed = append(result.Trimmed, score.CandidateID)
		default:
			result.Rejected = append(result.Rejected, score.CandidateID)
		}
	}

	result.AggregateConfidence = e.AggregateConfidence(result.Scores)
	return result
}

func (e Evaluator) EvaluateOutput(request EvaluateOutputRequest) OutputEvaluation {
	config := ensureConfig(e.config)
	task := strings.TrimSpace(request.Task)
	taskTerms := textindex.AnalyzedTerms(task)
	graphState := normalizeGraphState(request.GraphState)

	candidate := Candidate{
		ID:      strings.TrimSpace(request.Candidate.ID),
		Kind:    "task_output",
		Summary: strings.TrimSpace(request.Candidate.Summary),
		Content: strings.TrimSpace(request.Candidate.Content),
	}
	if strings.TrimSpace(candidate.Summary) == "" && strings.TrimSpace(candidate.Content) == "" {
		return OutputEvaluation{CandidateID: candidate.ID}
	}
	score := scoreContextCandidate(config, taskTerms, candidate, []Candidate{candidate}, graphState, 0)
	result := OutputEvaluation{
		CandidateID: score.CandidateID,
		Scores:      score.Scores,
		Composite:   score.Composite,
		Metadata:    score.Metadata,
	}

	if request.PriorOutput == nil {
		return result
	}

	priorCandidate := Candidate{
		ID:      strings.TrimSpace(request.PriorOutput.ID),
		Kind:    "task_output",
		Summary: strings.TrimSpace(request.PriorOutput.Summary),
		Content: strings.TrimSpace(request.PriorOutput.Content),
	}
	priorScore := scoreContextCandidate(config, taskTerms, priorCandidate, []Candidate{priorCandidate}, graphState, 0)
	delta := roundMetric(result.Composite - priorScore.Composite)
	reasons := []string{}
	if delta < -config.RegressionTolerance {
		reasons = append(reasons, "composite confidence regressed")
	}
	if result.Scores.Relevance+config.RegressionTolerance < priorScore.Scores.Relevance {
		reasons = append(reasons, "relevance regressed")
	}
	if result.Scores.Usefulness+config.RegressionTolerance < priorScore.Scores.Usefulness {
		reasons = append(reasons, "usefulness regressed")
	}
	result.Baseline = &BaselineComparison{
		PriorScores:        priorScore.Scores,
		PriorComposite:     priorScore.Composite,
		CompositeDelta:     delta,
		RegressionDetected: len(reasons) > 0,
		Reasons:            reasons,
	}
	return result
}

func scoreContextCandidate(config Config, taskTerms []string, candidate Candidate, candidates []Candidate, graphState []GraphState, index int) CandidateScore {
	candidate = normalizeCandidate(candidate)
	otherCandidates := make([]Candidate, 0, len(candidates)-1)
	for i, other := range candidates {
		if i == index {
			continue
		}
		otherCandidates = append(otherCandidates, normalizeCandidate(other))
	}

	candidateTerms := candidateAnalyzedTerms(candidate)
	taskMatches := overlapTerms(taskTerms, append(cloneStrings(candidateTerms), candidate.MatchedTerms...))
	matchedCoverage := ratio(len(taskMatches), len(taskTerms))
	weightedCoverage := weightedTaskCoverage(taskTerms, candidateDocument(candidate))
	structuralSignals := structuralSignals(candidate.GraphRefs)
	structuralScore := structuralSupportScore(candidate.GraphRefs)

	graphConflicts := conflictingGraphEntities(candidate, graphState)
	candidateConflicts := conflictingCandidateIDs(candidate, otherCandidates)
	recencyScore, stale, ageDays := provenanceRecency(candidate.Provenance, config.StaleAfter)
	coherence := clamp01(0.45 + 0.55*structuralScore)
	conflictPenalty := math.Min(0.75, 0.4*float64(len(graphConflicts)+len(candidateConflicts)))

	relevance := clamp01(0.5*matchedCoverage + 0.35*weightedCoverage + 0.15*structuralScore)
	consistency := clamp01(0.45*recencyScore + 0.35*coherence + 0.20*matchedCoverage - conflictPenalty)
	if len(graphConflicts) == 0 && len(candidateConflicts) == 0 && matchedCoverage > 0.6 {
		consistency = clamp01(consistency + 0.1)
	}

	focus := ratio(len(taskMatches), len(candidateTerms))
	signalDensity := candidateSignalDensity(candidate)
	usefulness := clamp01(0.55*relevance + 0.25*focus + 0.20*signalDensity)
	if stale {
		usefulness = clamp01(usefulness - 0.1)
	}
	if len(graphConflicts)+len(candidateConflicts) > 0 {
		usefulness = clamp01(usefulness - 0.15)
	}

	scores := DimensionScores{
		Relevance:   roundScore(relevance),
		Consistency: roundScore(consistency),
		Usefulness:  roundScore(usefulness),
	}
	composite := roundScore(Evaluator{config: config}.CompositeConfidence(scores))
	fate, rejectionReason := classifyScore(config.Thresholds, scores, composite)

	return CandidateScore{
		CandidateID:     candidate.ID,
		Scores:          scores,
		Composite:       composite,
		Fate:            fate,
		RejectionReason: rejectionReason,
		Metadata: ScoreMetadata{
			MatchedTaskTerms:         taskMatches,
			StructuralSignals:        structuralSignals,
			ConflictingGraphEntities: graphConflicts,
			ConflictingCandidates:    candidateConflicts,
			Stale:                    stale,
			AgeDays:                  roundMetric(ageDays),
		},
	}
}

func classifyScore(thresholds CandidateThresholds, scores DimensionScores, composite float64) (string, string) {
	switch {
	case scores.Relevance < thresholds.MinimumRelevance:
		return fateRejected, "below relevance threshold"
	case scores.Consistency < thresholds.MinimumConsistency:
		return fateRejected, "below consistency threshold"
	case scores.Usefulness < thresholds.MinimumUsefulness:
		return fateRejected, "below usefulness threshold"
	case composite >= thresholds.SurviveComposite:
		return fateSurvived, ""
	case composite >= thresholds.TrimComposite:
		return fateTrimmed, ""
	default:
		return fateRejected, "below composite threshold"
	}
}

func ensureConfig(config Config) Config {
	return NewEvaluator(config).config
}

func normalizeWeights(weights DimensionWeights) DimensionWeights {
	values := []float64{
		math.Max(0, weights.Relevance),
		math.Max(0, weights.Consistency),
		math.Max(0, weights.Usefulness),
	}
	total := values[0] + values[1] + values[2]
	if total == 0 {
		return DimensionWeights{Relevance: 1.0 / 3.0, Consistency: 1.0 / 3.0, Usefulness: 1.0 / 3.0}
	}
	return DimensionWeights{
		Relevance:   values[0] / total,
		Consistency: values[1] / total,
		Usefulness:  values[2] / total,
	}
}

func normalizeThresholds(thresholds CandidateThresholds) CandidateThresholds {
	if thresholds.SurviveComposite <= 0 {
		thresholds.SurviveComposite = 0.65
	}
	if thresholds.TrimComposite <= 0 {
		thresholds.TrimComposite = 0.4
	}
	if thresholds.MinimumRelevance <= 0 {
		thresholds.MinimumRelevance = 0.25
	}
	if thresholds.MinimumConsistency <= 0 {
		thresholds.MinimumConsistency = 0.35
	}
	if thresholds.MinimumUsefulness <= 0 {
		thresholds.MinimumUsefulness = 0.2
	}
	thresholds.SurviveComposite = clamp01(thresholds.SurviveComposite)
	thresholds.TrimComposite = clamp01(math.Min(thresholds.TrimComposite, thresholds.SurviveComposite))
	thresholds.MinimumRelevance = clamp01(thresholds.MinimumRelevance)
	thresholds.MinimumConsistency = clamp01(thresholds.MinimumConsistency)
	thresholds.MinimumUsefulness = clamp01(thresholds.MinimumUsefulness)
	return thresholds
}

func normalizeCandidates(candidates []Candidate) []Candidate {
	normalized := make([]Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		normalized = append(normalized, normalizeCandidate(candidate))
	}
	return normalized
}

func normalizeCandidate(candidate Candidate) Candidate {
	return Candidate{
		ID:           strings.TrimSpace(candidate.ID),
		Kind:         strings.TrimSpace(candidate.Kind),
		Title:        strings.TrimSpace(candidate.Title),
		Summary:      strings.TrimSpace(candidate.Summary),
		Content:      strings.TrimSpace(candidate.Content),
		RepoPath:     strings.TrimSpace(candidate.RepoPath),
		Tags:         cloneStrings(candidate.Tags),
		MatchedTerms: cloneStrings(candidate.MatchedTerms),
		GraphRefs:    cloneGraphRefs(candidate.GraphRefs),
		Provenance:   candidate.Provenance,
	}
}

func normalizeGraphState(items []GraphState) []GraphState {
	normalized := make([]GraphState, 0, len(items))
	for _, item := range items {
		normalized = append(normalized, GraphState{
			EntityID:   strings.TrimSpace(item.EntityID),
			Kind:       strings.TrimSpace(item.Kind),
			Title:      strings.TrimSpace(item.Title),
			Summary:    strings.TrimSpace(item.Summary),
			Content:    strings.TrimSpace(item.Content),
			RepoPath:   strings.TrimSpace(item.RepoPath),
			Tags:       cloneStrings(item.Tags),
			Provenance: item.Provenance,
		})
	}
	return normalized
}

func candidateDocument(candidate Candidate) textindex.Document {
	return textindex.Document{
		ID:       candidate.ID,
		Kind:     candidate.Kind,
		Title:    candidate.Title,
		Summary:  candidate.Summary,
		Content:  candidate.Content,
		RepoPath: candidate.RepoPath,
		Tags:     cloneStrings(candidate.Tags),
	}
}

func graphStateDocument(item GraphState) textindex.Document {
	return textindex.Document{
		ID:       item.EntityID,
		Kind:     item.Kind,
		Title:    item.Title,
		Summary:  item.Summary,
		Content:  item.Content,
		RepoPath: item.RepoPath,
		Tags:     cloneStrings(item.Tags),
	}
}

func candidateAnalyzedTerms(candidate Candidate) []string {
	return textindex.AnalyzedTerms(strings.Join([]string{
		candidate.ID,
		candidate.Kind,
		candidate.Title,
		candidate.Summary,
		candidate.Content,
		candidate.RepoPath,
		strings.Join(candidate.Tags, " "),
		strings.Join(candidate.MatchedTerms, " "),
	}, " "))
}

func weightedTaskCoverage(taskTerms []string, doc textindex.Document) float64 {
	if len(taskTerms) == 0 {
		return 0
	}
	weights := textindex.WeightedTerms(doc)
	var overlap float64
	for _, term := range taskTerms {
		overlap += weights[term]
	}
	return clamp01(overlap / (3 * float64(len(taskTerms))))
}

func structuralSignals(refs []retrieval.GraphRef) []string {
	if len(refs) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	signals := make([]string, 0, len(refs))
	for _, ref := range refs {
		direction := strings.TrimSpace(ref.Direction)
		if direction == "" {
			continue
		}
		if _, ok := seen[direction]; ok {
			continue
		}
		seen[direction] = struct{}{}
		signals = append(signals, direction)
	}
	sort.Strings(signals)
	return signals
}

func structuralSupportScore(refs []retrieval.GraphRef) float64 {
	var raw float64
	for _, ref := range refs {
		switch strings.TrimSpace(ref.Direction) {
		case "supports_task":
			raw += 1.0
		case "matched_seed":
			raw += 0.6
		case "outgoing_neighbor", "incoming_neighbor":
			raw += 0.35
		default:
			raw += 0.15
		}
	}
	return clamp01(1 - math.Exp(-raw))
}

func conflictingGraphEntities(candidate Candidate, graphState []GraphState) []string {
	candidateTerms := candidateAnalyzedTerms(candidate)
	conflicts := []string{}
	for _, item := range graphState {
		if item.EntityID == "" {
			continue
		}
		otherTerms := textindex.AnalyzedTerms(strings.Join([]string{
			item.EntityID,
			item.Kind,
			item.Title,
			item.Summary,
			item.Content,
			item.RepoPath,
			strings.Join(item.Tags, " "),
		}, " "))
		sameTarget := sameEntity(candidate, item) || subjectOverlap(candidateTerms, otherTerms) > 0
		if sameTarget && contradictoryTerms(candidateTerms, otherTerms) {
			conflicts = append(conflicts, item.EntityID)
		}
	}
	sort.Strings(conflicts)
	return conflicts
}

func conflictingCandidateIDs(candidate Candidate, others []Candidate) []string {
	candidateTerms := candidateAnalyzedTerms(candidate)
	conflicts := []string{}
	for _, other := range others {
		otherTerms := candidateAnalyzedTerms(other)
		sameTarget := sameCandidateSubject(candidate, other) || subjectOverlap(candidateTerms, otherTerms) > 0
		if sameTarget && contradictoryTerms(candidateTerms, otherTerms) {
			conflicts = append(conflicts, other.ID)
		}
	}
	sort.Strings(conflicts)
	return conflicts
}

func provenanceRecency(provenance retrieval.Provenance, staleAfter time.Duration) (score float64, stale bool, ageDays float64) {
	timestamp := strings.TrimSpace(provenance.UpdatedAt)
	if timestamp == "" {
		timestamp = strings.TrimSpace(provenance.CreatedAt)
	}
	if timestamp == "" {
		return 0.65, false, 0
	}
	parsed, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return 0.65, false, 0
	}
	age := time.Since(parsed)
	if age < 0 {
		age = 0
	}
	ageDays = age.Hours() / 24
	if age <= staleAfter {
		freshness := 1 - 0.15*(float64(age)/float64(staleAfter))
		return clamp01(freshness), false, ageDays
	}
	overdue := float64(age-staleAfter) / float64(staleAfter)
	score = clamp01(0.85 - 0.45*overdue)
	return score, true, ageDays
}

func candidateSignalDensity(candidate Candidate) float64 {
	var signals float64
	for _, value := range []string{candidate.Title, candidate.Summary, candidate.Content, candidate.RepoPath} {
		if strings.TrimSpace(value) != "" {
			signals++
		}
	}
	signals += math.Min(float64(len(candidate.Tags)), 2)
	signals += math.Min(float64(len(candidate.GraphRefs)), 2)
	return clamp01(signals / 8)
}

func sameEntity(candidate Candidate, item GraphState) bool {
	switch {
	case candidate.ID != "" && candidate.ID == item.EntityID:
		return true
	case candidate.RepoPath != "" && candidate.RepoPath == item.RepoPath:
		return true
	default:
		return false
	}
}

func sameCandidateSubject(left, right Candidate) bool {
	switch {
	case left.ID != "" && left.ID == right.ID:
		return true
	case left.RepoPath != "" && left.RepoPath == right.RepoPath:
		return true
	default:
		return false
	}
}

func contradictoryTerms(left, right []string) bool {
	if len(left) == 0 || len(right) == 0 {
		return false
	}
	leftSet := toSet(left)
	rightSet := toSet(right)
	if subjectOverlap(left, right) == 0 {
		return false
	}
	for _, group := range contradictionGroups {
		leftPositive := containsAny(leftSet, group.positive)
		leftNegative := containsAny(leftSet, group.negative)
		rightPositive := containsAny(rightSet, group.positive)
		rightNegative := containsAny(rightSet, group.negative)
		if (leftPositive && rightNegative) || (leftNegative && rightPositive) {
			return true
		}
	}
	return false
}

func subjectOverlap(left, right []string) int {
	leftSet := toSet(left)
	count := 0
	for _, term := range right {
		if _, noisy := contradictionTokens[term]; noisy {
			continue
		}
		if _, ok := leftSet[term]; ok {
			count++
		}
	}
	return count
}

func overlapTerms(left, right []string) []string {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	rightSet := toSet(right)
	overlap := []string{}
	for _, term := range left {
		if _, ok := rightSet[term]; !ok {
			continue
		}
		overlap = append(overlap, term)
	}
	overlap = slices.Compact(overlap)
	sort.Strings(overlap)
	return overlap
}

func buildContradictionTokenSet() map[string]struct{} {
	set := map[string]struct{}{}
	for _, group := range contradictionGroups {
		for _, token := range group.positive {
			set[token] = struct{}{}
		}
		for _, token := range group.negative {
			set[token] = struct{}{}
		}
	}
	return set
}

func containsAny(set map[string]struct{}, values []string) bool {
	for _, value := range values {
		if _, ok := set[value]; ok {
			return true
		}
	}
	return false
}

func toSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		set[value] = struct{}{}
	}
	return set
}

func ratio(numerator, denominator int) float64 {
	if denominator == 0 || numerator <= 0 {
		return 0
	}
	return clamp01(float64(numerator) / float64(denominator))
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var total float64
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func roundScore(value float64) float64 {
	return math.Round(clamp01(value)*1000) / 1000
}

func roundMetric(value float64) float64 {
	return math.Round(value*1000) / 1000
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func cloneGraphRefs(values []retrieval.GraphRef) []retrieval.GraphRef {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]retrieval.GraphRef, len(values))
	copy(cloned, values)
	return cloned
}
