package workflow

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/app/contextprojector"
	"github.com/guillaume-galp/cge/internal/app/graphhealth"
	"github.com/guillaume-galp/cge/internal/app/retrieval"
	"github.com/guillaume-galp/cge/internal/infra/kuzu"
	"github.com/guillaume-galp/cge/internal/infra/repo"
	"github.com/guillaume-galp/cge/internal/infra/textindex"
)

const (
	RecommendationProceed        = "proceed"
	RecommendationBootstrap      = "bootstrap"
	RecommendationInspectHygiene = "inspect_hygiene"
	RecommendationGatherContext  = "gather_context"

	KickoffCoverageGrounded   = "grounded"
	KickoffCoverageLowContext = "low_context"
	KickoffCoverageAbstained  = "abstained"

	KickoffFamilyWriteProducing     = "write_producing"
	KickoffFamilyTroubleshooting    = "troubleshooting_diagnosis"
	KickoffFamilyVerificationAudit  = "verification_audit"
	KickoffFamilyReportingSynthesis = "reporting_synthesis"
	KickoffFamilyWorkflowContext    = "workflow_context"
	KickoffFamilyAmbiguousTask      = "ambiguous_task"

	KickoffModeInject  = "inject"
	KickoffModeMinimal = "minimal"
	KickoffModeAbstain = "abstain"
	KickoffModeAuto    = "auto"
)

var (
	writeProducingKeywords  = []string{"implement", "add ", "build", "create", "write", "modify", "refactor", "ship"}
	troubleshootingKeywords = []string{"diagnose", "debug", "troubleshoot", "investigate", "broken", "failing", "failure", "error"}
	verificationKeywords    = []string{"audit", "verify", "validation", "validate", "check", "inspect", "review", "snapshot"}
	reportingKeywords       = []string{"report", "summarize", "summary", "synthesis", "synthesize", "debrief", "writeup", "analysis"}
	workflowKeywords        = []string{"workflow", "handoff", "delegated", "delegate", "bootstrap"}
	graphHygieneKeywords    = []string{"hygiene", "contradiction", "contradictory", "duplicate", "duplication", "orphan", "graph stats", "health indicator", "graph health"}
)

type ReadinessReader interface {
	ReadGraph(ctx context.Context, workspace repo.Workspace) (kuzu.Graph, error)
	CurrentRevision(ctx context.Context, workspace repo.Workspace) (kuzu.CurrentRevisionState, error)
}

type KickoffQuerier interface {
	Query(ctx context.Context, workspace repo.Workspace, task string) (retrieval.ResultSet, error)
}

type KickoffProjector interface {
	Project(resultSet retrieval.ResultSet, maxTokens int) (contextprojector.Envelope, error)
}

type StartResult struct {
	Recommendation string          `json:"recommendation"`
	Readiness      ReadinessState  `json:"readiness"`
	Kickoff        KickoffEnvelope `json:"kickoff"`
}

type StartOptions struct {
	KickoffMode string `json:"kickoff_mode,omitempty"`
}

type ReadinessState struct {
	Status     string          `json:"status"`
	Reasons    []string        `json:"reasons"`
	GraphState StartGraphState `json:"graph_state"`
}

type StartGraphState struct {
	WorkspaceInitialized bool                      `json:"workspace_initialized"`
	WorkspacePath        string                    `json:"workspace_path,omitempty"`
	GraphAvailable       bool                      `json:"graph_available"`
	CurrentRevision      kuzu.CurrentRevisionState `json:"current_revision"`
	Health               HealthState               `json:"health"`
}

type HealthState struct {
	Available      bool                   `json:"available"`
	SnapshotAnchor string                 `json:"snapshot_anchor,omitempty"`
	Snapshot       kuzu.GraphStats        `json:"snapshot"`
	Indicators     graphhealth.Indicators `json:"indicators"`
}

type KickoffEnvelope struct {
	Task            KickoffTaskDetails   `json:"task"`
	GraphState      KickoffGraphState    `json:"graph_state"`
	Policy          KickoffPolicyState   `json:"policy"`
	Advisory        KickoffAdvisoryState `json:"advisory"`
	Context         KickoffContext       `json:"context"`
	DelegationBrief DelegationBrief      `json:"delegation_brief"`
}

type KickoffTaskDetails struct {
	Description string `json:"description"`
	MaxTokens   int    `json:"max_tokens"`
	Family      string `json:"family"`
}

type KickoffGraphState struct {
	ReadinessStatus       string                 `json:"readiness_status"`
	Recommendation        string                 `json:"recommendation"`
	WorkspaceInitialized  bool                   `json:"workspace_initialized"`
	GraphAvailable        bool                   `json:"graph_available"`
	CurrentRevisionID     string                 `json:"current_revision_id,omitempty"`
	CurrentRevisionAnchor string                 `json:"current_revision_anchor,omitempty"`
	CurrentRevisionReason string                 `json:"current_revision_reason,omitempty"`
	SnapshotAnchor        string                 `json:"snapshot_anchor,omitempty"`
	Nodes                 int                    `json:"nodes"`
	Relationships         int                    `json:"relationships"`
	Indicators            graphhealth.Indicators `json:"indicators"`
}

type KickoffPolicyState struct {
	Family              string   `json:"family"`
	AllowedEntityKinds  []string `json:"allowed_entity_kinds,omitempty"`
	SuppressedPatterns  []string `json:"suppressed_patterns,omitempty"`
	DefaultKickoffMode  string   `json:"default_kickoff_mode"`
	ClassificationBasis []string `json:"classification_basis,omitempty"`
}

type KickoffAdvisoryState struct {
	RequestedMode   string   `json:"requested_mode"`
	EffectiveMode   string   `json:"effective_mode"`
	ConfidenceLevel string   `json:"confidence_level"`
	ConfidenceScore float64  `json:"confidence_score"`
	ReasonCodes     []string `json:"reason_codes,omitempty"`
	NextStep        string   `json:"next_step,omitempty"`
}

type KickoffContext struct {
	Coverage         string                    `json:"coverage"`
	Summary          string                    `json:"summary"`
	Guidance         []string                  `json:"guidance,omitempty"`
	Abstained        bool                      `json:"abstained,omitempty"`
	AbstentionReason string                    `json:"abstention_reason,omitempty"`
	Envelope         contextprojector.Envelope `json:"envelope"`
}

type DelegationBrief struct {
	Status   string   `json:"status"`
	Prompt   string   `json:"prompt"`
	Guidance []string `json:"guidance,omitempty"`
}

type kickoffTaskFamily struct {
	Name                string
	ClassificationBasis []string
}

type kickoffPolicy struct {
	Family             string
	AllowedEntityKinds []string
	SuppressedPatterns []string
	DefaultKickoffMode string
}

type kickoffConfidenceAssessment struct {
	Level       string
	Score       float64
	ReasonCodes []string
}

func (s *Service) Start(ctx context.Context, startDir, task string, maxTokens int) (StartResult, error) {
	return s.StartWithOptions(ctx, startDir, task, maxTokens, StartOptions{KickoffMode: KickoffModeAuto})
}

func (s *Service) StartWithOptions(ctx context.Context, startDir, task string, maxTokens int, options StartOptions) (StartResult, error) {
	if s == nil || s.manager == nil {
		return StartResult{}, errors.New("workflow service is not configured")
	}
	options = normalizeStartOptions(options)
	if s.reader == nil {
		s.reader = kuzu.NewStore()
	}
	if s.querier == nil {
		s.querier = retrieval.NewEngine(nil, nil)
	}
	if s.projector == nil {
		s.projector = contextprojector.NewProjector()
	}
	if err := contextprojector.ValidateMaxTokens(maxTokens); err != nil {
		return StartResult{}, err
	}

	workspace, err := s.manager.OpenWorkspace(ctx, startDir)
	if err != nil {
		if errors.Is(err, repo.ErrWorkspaceNotInitialized) {
			result := bootstrapReadiness()
			taskFamily := classifyKickoffTaskFamily(task)
			policy := kickoffPolicyForFamily(taskFamily.Name)
			advisory := determineKickoffAdvisory(taskFamily, policy, options.KickoffMode, kickoffConfidenceAssessment{Level: "none", Score: 0, ReasonCodes: []string{"workspace_not_initialized"}})
			result.Kickoff = buildKickoffEnvelope(task, maxTokens, result.Recommendation, result.Readiness, taskFamily, policy, advisory, retrieval.ResultSet{}, emptyKickoffContextEnvelope(maxTokens))
			return result, nil
		}
		return StartResult{}, err
	}

	currentRevision, err := s.reader.CurrentRevision(ctx, workspace)
	if err != nil {
		return StartResult{}, classifyReadinessInspectionError("current_revision", err)
	}

	graph, err := s.reader.ReadGraph(ctx, workspace)
	if err != nil {
		return StartResult{}, classifyReadinessInspectionError("graph_state", err)
	}

	analysis, err := graphhealth.AnalyzeGraph(graph)
	if err != nil {
		return StartResult{}, classifyReadinessInspectionError("health_indicators", err)
	}

	graphState := StartGraphState{
		WorkspaceInitialized: true,
		WorkspacePath:        workspace.WorkspacePath,
		GraphAvailable:       graphAvailable(currentRevision, analysis.Snapshot),
		CurrentRevision:      currentRevision,
		Health: HealthState{
			Available:      true,
			SnapshotAnchor: analysis.SnapshotAnchor,
			Snapshot:       analysis.Snapshot,
			Indicators:     analysis.Indicators,
		},
	}
	recommendation, reasons := recommendWorkflowStart(graphState)
	taskFamily := classifyKickoffTaskFamily(task)
	policy := kickoffPolicyForFamily(taskFamily.Name)
	readiness := ReadinessState{
		Status:     readinessStatus(recommendation),
		Reasons:    reasons,
		GraphState: graphState,
	}

	resultSet := retrieval.ResultSet{Results: []retrieval.Result{}}
	contextEnvelope := emptyKickoffContextEnvelope(maxTokens)
	if graphState.GraphAvailable && options.KickoffMode != KickoffModeAbstain && policy.DefaultKickoffMode != KickoffModeAbstain {
		resultSet, err = s.querier.Query(ctx, workspace, task)
		if err != nil {
			return StartResult{}, classifyKickoffContextError("context_query", err)
		}
		resultSet = calibrateKickoffResultSet(taskFamily, policy, resultSet)
		contextEnvelope, err = s.projector.Project(resultSet, maxTokens)
		if err != nil {
			var validationErr *contextprojector.ValidationError
			if errors.As(err, &validationErr) {
				return StartResult{}, err
			}
			return StartResult{}, classifyKickoffContextError("context_projection", err)
		}
	}

	confidence := assessKickoffConfidence(taskFamily, graphState, resultSet, contextEnvelope)
	advisory := determineKickoffAdvisory(taskFamily, policy, options.KickoffMode, confidence)
	contextEnvelope = applyKickoffModeEnvelope(advisory, resultSet, contextEnvelope, maxTokens, s.projector)
	if advisory.EffectiveMode == KickoffModeAbstain {
		resultSet = retrieval.ResultSet{IndexStatus: resultSet.IndexStatus, Results: []retrieval.Result{}}
		contextEnvelope = emptyKickoffContextEnvelope(maxTokens)
	}
	annotateInclusionReasons(taskFamily, resultSet, &contextEnvelope)

	recommendation, readiness.Reasons = calibrateKickoffRecommendation(taskFamily, advisory, recommendation, readiness.Reasons, resultSet, contextEnvelope)
	readiness.Status = readinessStatus(recommendation)
	kickoff := buildKickoffEnvelope(task, maxTokens, recommendation, readiness, taskFamily, policy, advisory, resultSet, contextEnvelope)
	if recommendation == RecommendationProceed && kickoff.Context.Coverage == KickoffCoverageLowContext {
		recommendation = RecommendationGatherContext
		readiness.Status = readinessStatus(recommendation)
		readiness.Reasons = appendReason(readiness.Reasons, "task_context_sparse")
		kickoff = buildKickoffEnvelope(task, maxTokens, recommendation, readiness, taskFamily, policy, advisory, resultSet, contextEnvelope)
	}

	return StartResult{
		Recommendation: recommendation,
		Readiness:      readiness,
		Kickoff:        kickoff,
	}, nil
}

func bootstrapReadiness() StartResult {
	graphState := StartGraphState{
		WorkspaceInitialized: false,
		GraphAvailable:       false,
		CurrentRevision:      kuzu.CurrentRevisionState{},
		Health: HealthState{
			Available: false,
			Snapshot:  kuzu.GraphStats{},
		},
	}
	return StartResult{
		Recommendation: RecommendationBootstrap,
		Readiness: ReadinessState{
			Status:     readinessStatus(RecommendationBootstrap),
			Reasons:    []string{"workspace_not_initialized"},
			GraphState: graphState,
		},
	}
}

func graphAvailable(revision kuzu.CurrentRevisionState, snapshot kuzu.GraphStats) bool {
	return revision.Exists || snapshot.Nodes > 0 || snapshot.Relationships > 0
}

func recommendWorkflowStart(state StartGraphState) (string, []string) {
	if !state.WorkspaceInitialized {
		return RecommendationBootstrap, []string{"workspace_not_initialized"}
	}
	if !state.GraphAvailable {
		return RecommendationBootstrap, []string{"graph_unavailable"}
	}
	if !state.CurrentRevision.Exists {
		return RecommendationBootstrap, []string{"current_revision_missing"}
	}
	if state.Health.Snapshot.Nodes == 0 {
		return RecommendationBootstrap, []string{"graph_empty"}
	}

	indicators := state.Health.Indicators
	hygieneReasons := make([]string, 0, 3)
	if indicators.ContradictoryFacts > 0 {
		hygieneReasons = append(hygieneReasons, "contradictions_detected")
	}
	if indicators.DuplicationRate >= 0.15 {
		hygieneReasons = append(hygieneReasons, "duplication_rate_high")
	}
	if indicators.OrphanRate >= 0.20 {
		hygieneReasons = append(hygieneReasons, "orphan_rate_high")
	}
	if len(hygieneReasons) > 0 {
		return RecommendationInspectHygiene, hygieneReasons
	}

	contextReasons := make([]string, 0, 3)
	if state.Health.Snapshot.Nodes < 3 {
		contextReasons = append(contextReasons, "graph_too_small")
	}
	if state.Health.Snapshot.Relationships < 2 {
		contextReasons = append(contextReasons, "graph_relationships_sparse")
	}
	if indicators.DensityScore < 0.10 {
		contextReasons = append(contextReasons, "graph_density_low")
	}
	if len(contextReasons) > 0 {
		return RecommendationGatherContext, contextReasons
	}

	return RecommendationProceed, []string{"current_revision_available", "acceptable_health_indicators"}
}

func readinessStatus(recommendation string) string {
	if recommendation == RecommendationProceed {
		return "ready"
	}
	return "action_required"
}

func classifyReadinessInspectionError(stage string, err error) error {
	if _, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return err
	}

	stage = strings.TrimSpace(stage)
	if stage == "" {
		stage = "readiness"
	}

	var persistenceErr *kuzu.PersistenceError
	if errors.As(err, &persistenceErr) {
		details := map[string]any{
			"stage": stage,
		}
		for key, value := range persistenceErr.Details {
			details[key] = value
		}
		return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "operational_error",
			Type:     "persistence_error",
			Code:     persistenceErr.Code,
			Message:  persistenceErr.Message,
			Details:  details,
		}, err)
	}

	return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "workflow_error",
		Code:     "workflow_start_inspection_failed",
		Message:  "workflow readiness inspection failed",
		Details: map[string]any{
			"stage":  stage,
			"reason": err.Error(),
		},
	}, err)
}

func buildKickoffEnvelope(task string, maxTokens int, recommendation string, readiness ReadinessState, taskFamily kickoffTaskFamily, policy kickoffPolicy, advisory KickoffAdvisoryState, resultSet retrieval.ResultSet, contextEnvelope contextprojector.Envelope) KickoffEnvelope {
	contextState := buildKickoffContext(task, maxTokens, recommendation, readiness.GraphState, taskFamily, policy, advisory, resultSet, contextEnvelope)
	kickoff := KickoffEnvelope{
		Task: KickoffTaskDetails{
			Description: strings.TrimSpace(task),
			MaxTokens:   maxTokens,
			Family:      taskFamily.Name,
		},
		GraphState: summarizeKickoffGraphState(readiness, recommendation),
		Policy: KickoffPolicyState{
			Family:              policy.Family,
			AllowedEntityKinds:  append([]string(nil), policy.AllowedEntityKinds...),
			SuppressedPatterns:  append([]string(nil), policy.SuppressedPatterns...),
			DefaultKickoffMode:  policy.DefaultKickoffMode,
			ClassificationBasis: append([]string(nil), taskFamily.ClassificationBasis...),
		},
		Advisory: advisory,
		Context:  contextState,
	}
	kickoff.DelegationBrief = buildDelegationBrief(kickoff)
	return kickoff
}

func summarizeKickoffGraphState(readiness ReadinessState, recommendation string) KickoffGraphState {
	graphState := readiness.GraphState
	summary := KickoffGraphState{
		ReadinessStatus:      readiness.Status,
		Recommendation:       recommendation,
		WorkspaceInitialized: graphState.WorkspaceInitialized,
		GraphAvailable:       graphState.GraphAvailable,
		Nodes:                graphState.Health.Snapshot.Nodes,
		Relationships:        graphState.Health.Snapshot.Relationships,
		Indicators:           graphState.Health.Indicators,
	}
	if graphState.Health.SnapshotAnchor != "" {
		summary.SnapshotAnchor = graphState.Health.SnapshotAnchor
	}
	if graphState.CurrentRevision.Exists {
		summary.CurrentRevisionID = graphState.CurrentRevision.Revision.ID
		summary.CurrentRevisionAnchor = graphState.CurrentRevision.Revision.Anchor
		summary.CurrentRevisionReason = graphState.CurrentRevision.Revision.Reason
	}
	return summary
}

func buildKickoffContext(task string, maxTokens int, recommendation string, graphState StartGraphState, taskFamily kickoffTaskFamily, policy kickoffPolicy, advisory KickoffAdvisoryState, resultSet retrieval.ResultSet, contextEnvelope contextprojector.Envelope) KickoffContext {
	contextEnvelope.MaxTokens = maxTokens
	if contextEnvelope.Results == nil {
		contextEnvelope.Results = []contextprojector.Result{}
	}

	if advisory.EffectiveMode == KickoffModeAbstain {
		return KickoffContext{
			Coverage:         KickoffCoverageAbstained,
			Summary:          abstainedContextSummary(taskFamily, advisory),
			Guidance:         abstainedContextGuidance(taskFamily, advisory),
			Abstained:        true,
			AbstentionReason: firstReason(advisory.ReasonCodes),
			Envelope:         contextEnvelope,
		}
	}

	coverage := KickoffCoverageGrounded
	if !hasGroundedContext(resultSet, contextEnvelope) {
		coverage = KickoffCoverageLowContext
	}

	contextState := KickoffContext{
		Coverage: coverage,
		Envelope: contextEnvelope,
	}
	if coverage == KickoffCoverageLowContext {
		contextState.Summary = lowContextSummary(resultSet, contextEnvelope, graphState)
		contextState.Guidance = lowContextGuidance(task, recommendation, graphState, resultSet, contextEnvelope, advisory)
		return contextState
	}

	contextState.Summary = groundedContextSummary(contextEnvelope)
	return contextState
}

func abstainedContextSummary(taskFamily kickoffTaskFamily, advisory KickoffAdvisoryState) string {
	if containsReason(advisory.ReasonCodes, "explicit_no_kickoff_requested") {
		return "Kickoff context was skipped because the caller explicitly requested no kickoff."
	}
	if containsReason(advisory.ReasonCodes, "low_confidence_abstain") {
		return "Kickoff context abstained because the retrieved evidence did not meet the confidence threshold for this task family."
	}
	if taskFamily.Name == KickoffFamilyReportingSynthesis {
		return "This task family defaults to no kickoff context because reporting and synthesis work is currently more vulnerable to retrieval contamination than helped by pushed graph context."
	}
	return "Kickoff context abstained under the selected family policy."
}

func abstainedContextGuidance(taskFamily kickoffTaskFamily, advisory KickoffAdvisoryState) []string {
	guidance := []string{
		"Start from fresh repository context and pull graph information on demand only if the task later proves to need it.",
	}
	if containsReason(advisory.ReasonCodes, "low_confidence_abstain") {
		guidance = append(guidance, "Use `graph context` on demand once concrete task landmarks emerge, rather than trusting a weak kickoff brief.")
	}
	if taskFamily.Name == KickoffFamilyReportingSynthesis {
		guidance = append(guidance, "Treat reporting and synthesis as an abstention-first family until future evidence justifies broader kickoff injection.")
	}
	return guidance
}

func groundedContextSummary(contextEnvelope contextprojector.Envelope) string {
	resultCount := len(contextEnvelope.Results)
	if contextEnvelope.Truncated {
		return fmt.Sprintf(
			"Projected %s of the highest-value graph results within the token budget; %d additional result(s) were omitted.",
			pluralizeCount(resultCount, "result"),
			contextEnvelope.OmittedResults,
		)
	}
	return fmt.Sprintf("Projected %s of task-relevant graph context within the token budget.", pluralizeCount(resultCount, "result"))
}

func lowContextSummary(resultSet retrieval.ResultSet, contextEnvelope contextprojector.Envelope, graphState StartGraphState) string {
	switch {
	case !graphState.WorkspaceInitialized:
		return "The graph workflow workspace is not initialized, so no kickoff context can be retrieved yet."
	case !graphState.GraphAvailable:
		return "The graph is not available yet, so this kickoff cannot rely on existing graph context."
	case len(resultSet.Results) == 0:
		return "Graph retrieval found no useful task-specific context for this delegation."
	case len(contextEnvelope.Results) == 0:
		return "The requested token budget is too small to carry even the top-ranked graph result."
	default:
		return "Graph retrieval found only weak task-specific signals, so this kickoff should be treated as low context."
	}
}

func lowContextGuidance(task, recommendation string, graphState StartGraphState, resultSet retrieval.ResultSet, contextEnvelope contextprojector.Envelope, advisory KickoffAdvisoryState) []string {
	guidance := []string{}
	switch {
	case !graphState.WorkspaceInitialized:
		guidance = append(guidance, `Run "graph workflow init" before delegating work that depends on graph context.`)
	case !graphState.GraphAvailable:
		guidance = append(guidance, "Bootstrap baseline graph knowledge before relying on a delegated kickoff brief.")
	case len(contextEnvelope.Results) == 0 && len(resultSet.Results) > 0:
		guidance = append(guidance, "Increase --max-tokens so the top-ranked graph result fits into the kickoff context.")
	default:
		guidance = append(guidance, "Inspect repository sources such as README.md, docs/architecture/, and the active backlog or story files before delegating.")
	}
	guidance = append(guidance,
		fmt.Sprintf("Treat %q as a low-context task until concrete graph evidence is gathered.", strings.TrimSpace(task)),
		"Capture newly discovered context explicitly instead of assuming the graph already covers this subtask.",
	)
	if containsReason(advisory.ReasonCodes, "minimal_kickoff_selected") || containsReason(advisory.ReasonCodes, "low_confidence_minimal") {
		guidance = append(guidance, "Proceed with the reduced kickoff brief and pull more graph context only if the task proves to need it.")
	}
	if recommendation == RecommendationInspectHygiene {
		guidance = append(guidance, "Inspect graph hygiene before trusting ambiguous or duplicate-heavy context.")
	}
	return dedupeStrings(guidance)
}

func hasGroundedContext(resultSet retrieval.ResultSet, contextEnvelope contextprojector.Envelope) bool {
	if len(contextEnvelope.Results) == 0 {
		return false
	}
	for _, result := range resultSet.Results {
		if hasHighSignal(result) {
			return true
		}
	}
	return false
}

func hasHighSignal(result retrieval.Result) bool {
	if result.Score >= 10 {
		return true
	}
	if len(result.MatchedTerms) >= 2 {
		return true
	}
	for _, ref := range result.GraphRefs {
		switch ref.Direction {
		case "supports_task", "matched_seed":
			return true
		}
	}
	return false
}

func calibrateKickoffResultSet(taskFamily kickoffTaskFamily, policy kickoffPolicy, resultSet retrieval.ResultSet) retrieval.ResultSet {
	if len(resultSet.Results) == 0 || taskFamily.Name == KickoffFamilyWorkflowContext {
		return resultSet
	}

	filtered := make([]retrieval.Result, 0, len(resultSet.Results))
	for _, result := range resultSet.Results {
		if shouldSuppressWorkflowKickoffResult(result, taskFamily, policy) {
			continue
		}
		filtered = append(filtered, result)
	}
	for i := range filtered {
		filtered[i].Rank = i + 1
	}
	resultSet.Results = filtered
	return resultSet
}

func calibrateKickoffRecommendation(taskFamily kickoffTaskFamily, advisory KickoffAdvisoryState, recommendation string, reasons []string, resultSet retrieval.ResultSet, contextEnvelope contextprojector.Envelope) (string, []string) {
	if recommendation != RecommendationInspectHygiene {
		return recommendation, reasons
	}
	if taskFamily.Name == KickoffFamilyVerificationAudit || taskFamily.Name == KickoffFamilyWorkflowContext {
		return recommendation, reasons
	}
	if !hasGroundedContext(resultSet, contextEnvelope) {
		return RecommendationGatherContext, []string{"task_context_sparse", "graph_hygiene_advisory"}
	}
	return RecommendationProceed, []string{"task_specific_context_grounded", "graph_hygiene_advisory"}
}

func assessKickoffConfidence(taskFamily kickoffTaskFamily, graphState StartGraphState, resultSet retrieval.ResultSet, contextEnvelope contextprojector.Envelope) kickoffConfidenceAssessment {
	reasons := make([]string, 0, 4)
	if !graphState.GraphAvailable {
		return kickoffConfidenceAssessment{Level: "none", Score: 0, ReasonCodes: []string{"graph_unavailable"}}
	}
	if taskFamily.Name == KickoffFamilyAmbiguousTask {
		reasons = append(reasons, "ambiguous_task_family")
	}

	score := 0.0
	if len(resultSet.Results) > 0 {
		score += 0.2
		reasons = append(reasons, "retrieval_results_available")
	}
	if len(contextEnvelope.Results) > 0 {
		score += 0.2
		reasons = append(reasons, "projected_context_available")
	}

	highSignalCount := 0
	for _, result := range resultSet.Results {
		if hasHighSignal(result) {
			highSignalCount++
		}
	}
	switch {
	case highSignalCount >= 2:
		score += 0.4
		reasons = append(reasons, "multiple_high_signal_results")
	case highSignalCount == 1:
		score += 0.25
		reasons = append(reasons, "single_high_signal_result")
	default:
		reasons = append(reasons, "high_signal_results_missing")
	}

	if graphState.Health.Snapshot.Nodes < 3 || graphState.Health.Snapshot.Relationships < 2 {
		score -= 0.15
		reasons = append(reasons, "sparse_graph_state")
	}
	if taskFamily.Name == KickoffFamilyAmbiguousTask {
		score -= 0.15
	}
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	level := "low"
	switch {
	case score >= 0.7:
		level = "high"
	case score >= 0.4:
		level = "medium"
	}
	return kickoffConfidenceAssessment{Level: level, Score: score, ReasonCodes: dedupeStrings(reasons)}
}

func determineKickoffAdvisory(taskFamily kickoffTaskFamily, policy kickoffPolicy, requestedMode string, confidence kickoffConfidenceAssessment) KickoffAdvisoryState {
	requestedMode = strings.TrimSpace(requestedMode)
	if requestedMode == "" {
		requestedMode = KickoffModeAuto
	}

	advisory := KickoffAdvisoryState{
		RequestedMode:   requestedMode,
		EffectiveMode:   policy.DefaultKickoffMode,
		ConfidenceLevel: confidence.Level,
		ConfidenceScore: confidence.Score,
		ReasonCodes:     append([]string(nil), confidence.ReasonCodes...),
	}

	switch requestedMode {
	case KickoffModeAbstain:
		advisory.EffectiveMode = KickoffModeAbstain
		advisory.ReasonCodes = appendReason(advisory.ReasonCodes, "explicit_no_kickoff_requested")
	case KickoffModeMinimal:
		if policy.DefaultKickoffMode == KickoffModeAbstain {
			advisory.EffectiveMode = KickoffModeAbstain
			advisory.ReasonCodes = appendReason(advisory.ReasonCodes, "family_policy_default_no_kickoff")
		} else {
			advisory.EffectiveMode = KickoffModeMinimal
			advisory.ReasonCodes = appendReason(advisory.ReasonCodes, "minimal_kickoff_selected")
		}
	default:
		switch {
		case policy.DefaultKickoffMode == KickoffModeAbstain:
			advisory.EffectiveMode = KickoffModeAbstain
			advisory.ReasonCodes = appendReason(advisory.ReasonCodes, "family_policy_default_no_kickoff")
		case confidence.Level == "high":
			advisory.EffectiveMode = KickoffModeInject
		case confidence.Level == "medium":
			advisory.EffectiveMode = KickoffModeMinimal
			advisory.ReasonCodes = appendReason(advisory.ReasonCodes, "minimal_kickoff_selected")
		default:
			if taskFamily.Name == KickoffFamilyAmbiguousTask {
				advisory.EffectiveMode = KickoffModeAbstain
				advisory.ReasonCodes = appendReason(advisory.ReasonCodes, "low_confidence_abstain")
			} else {
				advisory.EffectiveMode = KickoffModeMinimal
				advisory.ReasonCodes = appendReason(advisory.ReasonCodes, "low_confidence_minimal")
			}
		}
	}

	switch advisory.EffectiveMode {
	case KickoffModeInject:
		advisory.NextStep = "proceed_with_kickoff"
	case KickoffModeMinimal:
		advisory.NextStep = "proceed_with_minimal_kickoff"
	default:
		if containsReason(advisory.ReasonCodes, "explicit_no_kickoff_requested") {
			advisory.NextStep = "proceed_with_fresh_context"
		} else {
			advisory.NextStep = "pull_graph_on_demand"
		}
	}
	advisory.ReasonCodes = dedupeStrings(advisory.ReasonCodes)
	return advisory
}

func applyKickoffModeEnvelope(advisory KickoffAdvisoryState, resultSet retrieval.ResultSet, contextEnvelope contextprojector.Envelope, maxTokens int, projector KickoffProjector) contextprojector.Envelope {
	switch advisory.EffectiveMode {
	case KickoffModeAbstain:
		return emptyKickoffContextEnvelope(maxTokens)
	case KickoffModeMinimal:
		minimalBudget := maxTokens
		if minimalBudget > 300 {
			minimalBudget = 300
		}
		if minimalBudget <= 0 {
			minimalBudget = maxTokens
		}
		if projector == nil {
			return contextEnvelope
		}
		projected, err := projector.Project(resultSet, minimalBudget)
		if err != nil {
			return contextEnvelope
		}
		projected.MaxTokens = minimalBudget
		return projected
	default:
		return contextEnvelope
	}
}

func annotateInclusionReasons(taskFamily kickoffTaskFamily, resultSet retrieval.ResultSet, envelope *contextprojector.Envelope) {
	if envelope == nil || len(envelope.Results) == 0 {
		return
	}
	byID := map[string]retrieval.Result{}
	for _, result := range resultSet.Results {
		byID[result.Entity.ID] = result
	}
	for i := range envelope.Results {
		result, ok := byID[envelope.Results[i].Entity.ID]
		if !ok {
			continue
		}
		envelope.Results[i].InclusionReason = buildInclusionReason(taskFamily, result)
	}
}

func buildInclusionReason(taskFamily kickoffTaskFamily, result retrieval.Result) string {
	if len(result.MatchedTerms) > 0 {
		return fmt.Sprintf("Included for the %s family because it matches task terms: %s.", taskFamily.Name, strings.Join(result.MatchedTerms, ", "))
	}
	if len(result.GraphRefs) > 0 {
		return fmt.Sprintf("Included for the %s family because graph relationships connect it to the delegated task context.", taskFamily.Name)
	}
	return fmt.Sprintf("Included for the %s family because it survived the family policy and ranked among the strongest available kickoff candidates.", taskFamily.Name)
}

func shouldSuppressWorkflowKickoffResult(result retrieval.Result, taskFamily kickoffTaskFamily, policy kickoffPolicy) bool {
	kind := strings.ToLower(strings.TrimSpace(result.Entity.Kind))
	id := strings.ToLower(strings.TrimSpace(result.Entity.ID))
	if len(policy.AllowedEntityKinds) > 0 && !stringInSliceFold(policy.AllowedEntityKinds, kind) {
		return true
	}
	if containsAnyFold(kind, policy.SuppressedPatterns) || containsAnyFold(id, policy.SuppressedPatterns) {
		return true
	}
	if taskFamily.Name != KickoffFamilyWorkflowContext {
		if strings.HasPrefix(kind, "workflow_") || strings.HasPrefix(id, "workflow-finish:") {
			return true
		}
	}
	return false
}

func classifyKickoffTaskFamily(task string) kickoffTaskFamily {
	normalized := normalizeKickoffTask(task)
	switch {
	case containsAnyFold(normalized, workflowKeywords):
		return kickoffTaskFamily{Name: KickoffFamilyWorkflowContext, ClassificationBasis: matchedKeywords(normalized, workflowKeywords)}
	case containsAnyFold(normalized, reportingKeywords):
		return kickoffTaskFamily{Name: KickoffFamilyReportingSynthesis, ClassificationBasis: matchedKeywords(normalized, reportingKeywords)}
	case containsAnyFold(normalized, troubleshootingKeywords):
		return kickoffTaskFamily{Name: KickoffFamilyTroubleshooting, ClassificationBasis: matchedKeywords(normalized, troubleshootingKeywords)}
	case containsAnyFold(normalized, verificationKeywords) || containsAnyFold(normalized, graphHygieneKeywords):
		basis := append(matchedKeywords(normalized, verificationKeywords), matchedKeywords(normalized, graphHygieneKeywords)...)
		return kickoffTaskFamily{Name: KickoffFamilyVerificationAudit, ClassificationBasis: dedupeStrings(basis)}
	case containsAnyFold(normalized, writeProducingKeywords):
		return kickoffTaskFamily{Name: KickoffFamilyWriteProducing, ClassificationBasis: matchedKeywords(normalized, writeProducingKeywords)}
	default:
		return kickoffTaskFamily{Name: KickoffFamilyAmbiguousTask, ClassificationBasis: []string{"no_family_keywords_matched"}}
	}
}

func kickoffPolicyForFamily(family string) kickoffPolicy {
	switch family {
	case KickoffFamilyWorkflowContext:
		return kickoffPolicy{Family: family, DefaultKickoffMode: KickoffModeInject}
	case KickoffFamilyWriteProducing:
		return kickoffPolicy{
			Family:             family,
			AllowedEntityKinds: []string{"document", "decision", "story", "reasoningunit", "coderef", "file", "type", "function"},
			SuppressedPatterns: []string{"workflow_", "workflow-finish:"},
			DefaultKickoffMode: KickoffModeInject,
		}
	case KickoffFamilyTroubleshooting:
		return kickoffPolicy{
			Family:             family,
			AllowedEntityKinds: []string{"document", "decision", "reasoningunit", "coderef", "file", "type", "function", "researchfinding"},
			SuppressedPatterns: []string{"workflow_", "workflow-finish:"},
			DefaultKickoffMode: KickoffModeInject,
		}
	case KickoffFamilyVerificationAudit:
		return kickoffPolicy{
			Family:             family,
			AllowedEntityKinds: []string{"document", "decision", "coderef", "file", "type", "function", "researchfinding"},
			SuppressedPatterns: []string{"workflow_", "workflow-finish:"},
			DefaultKickoffMode: KickoffModeInject,
		}
	case KickoffFamilyReportingSynthesis:
		return kickoffPolicy{
			Family:             family,
			SuppressedPatterns: []string{"workflow_", "workflow-finish:"},
			DefaultKickoffMode: KickoffModeAbstain,
		}
	default:
		return kickoffPolicy{
			Family:             KickoffFamilyAmbiguousTask,
			AllowedEntityKinds: []string{"document", "decision", "story", "coderef", "file"},
			SuppressedPatterns: []string{"workflow_", "workflow-finish:"},
			DefaultKickoffMode: KickoffModeMinimal,
		}
	}
}

func normalizeKickoffTask(task string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(task))), " ")
}

func buildDelegationBrief(kickoff KickoffEnvelope) DelegationBrief {
	guidance := append([]string(nil), kickoff.Context.Guidance...)
	status := kickoff.Context.Coverage

	lines := []string{
		fmt.Sprintf("Task: %s", kickoff.Task.Description),
		fmt.Sprintf("Kickoff family: %s", kickoff.Task.Family),
		"",
		"Kickoff advisory:",
		fmt.Sprintf("- requested mode: %s", kickoff.Advisory.RequestedMode),
		fmt.Sprintf("- effective mode: %s", kickoff.Advisory.EffectiveMode),
		fmt.Sprintf("- confidence: %s (%.2f)", kickoff.Advisory.ConfidenceLevel, kickoff.Advisory.ConfidenceScore),
		fmt.Sprintf("- next step: %s", kickoff.Advisory.NextStep),
		"",
		"Graph state:",
		fmt.Sprintf("- readiness: %s", kickoff.GraphState.ReadinessStatus),
		fmt.Sprintf("- recommendation: %s", kickoff.GraphState.Recommendation),
		fmt.Sprintf("- graph available: %s", strconv.FormatBool(kickoff.GraphState.GraphAvailable)),
		fmt.Sprintf("- snapshot: %d node(s), %d relationship(s)", kickoff.GraphState.Nodes, kickoff.GraphState.Relationships),
	}
	if kickoff.GraphState.CurrentRevisionID != "" {
		lines = append(lines, fmt.Sprintf("- current revision: %s", kickoff.GraphState.CurrentRevisionID))
	}
	lines = append(lines, "", fmt.Sprintf("Kickoff context (%s):", kickoff.Context.Coverage))
	if kickoff.Context.Coverage == KickoffCoverageAbstained || kickoff.Context.Coverage == KickoffCoverageLowContext {
		lines = append(lines, kickoff.Context.Summary, "", "Next best action:")
		for _, item := range guidance {
			lines = append(lines, "- "+item)
		}
	} else {
		lines = append(lines, kickoff.Context.Summary)
		for _, item := range kickoff.Context.Envelope.Results {
			lines = append(lines, formatKickoffContextResult(item))
		}
		if kickoff.Context.Envelope.Truncated {
			lines = append(lines, fmt.Sprintf("- additional omitted results: %d", kickoff.Context.Envelope.OmittedResults))
		}
	}

	return DelegationBrief{
		Status:   status,
		Prompt:   strings.Join(lines, "\n"),
		Guidance: guidance,
	}
}

func formatKickoffContextResult(result contextprojector.Result) string {
	label := strings.TrimSpace(result.Entity.Title)
	if label == "" {
		label = result.Entity.ID
	}
	details := []string{fmt.Sprintf("- %s [%s]", label, result.Entity.Kind)}
	if summary := strings.TrimSpace(result.Entity.Summary); summary != "" {
		details[0] += ": " + summary
	}

	relationshipLabels := make([]string, 0, len(result.Relationships))
	for _, relationship := range result.Relationships {
		label := relationship.Kind
		if relationship.Peer != "" {
			label += " -> " + relationship.Peer
		}
		relationshipLabels = append(relationshipLabels, label)
	}
	sort.Strings(relationshipLabels)
	if len(relationshipLabels) > 0 {
		details = append(details, "  relationships: "+strings.Join(relationshipLabels, ", "))
	}
	if len(result.MatchedTerms) > 0 {
		details = append(details, "  matched terms: "+strings.Join(result.MatchedTerms, ", "))
	}
	if reason := strings.TrimSpace(result.InclusionReason); reason != "" {
		details = append(details, "  included because: "+reason)
	}
	return strings.Join(details, "\n")
}

func emptyKickoffContextEnvelope(maxTokens int) contextprojector.Envelope {
	return contextprojector.Envelope{
		MaxTokens: maxTokens,
		Results:   []contextprojector.Result{},
	}
}

func classifyKickoffContextError(stage string, err error) error {
	if _, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return err
	}

	var indexErr *textindex.Error
	if errors.As(err, &indexErr) {
		details := map[string]any{
			"stage": stage,
		}
		for key, value := range indexErr.Details {
			details[key] = value
		}
		return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "operational_error",
			Type:     "index_error",
			Code:     indexErr.Code,
			Message:  indexErr.Message,
			Details:  details,
		}, err)
	}

	return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "workflow_error",
		Code:     "workflow_kickoff_context_failed",
		Message:  "workflow kickoff context assembly failed",
		Details: map[string]any{
			"stage":  stage,
			"reason": err.Error(),
		},
	}, err)
}

func appendReason(reasons []string, reason string) []string {
	if reason == "" {
		return reasons
	}
	for _, existing := range reasons {
		if existing == reason {
			return reasons
		}
	}
	return append(reasons, reason)
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	deduped := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		deduped = append(deduped, value)
	}
	if len(deduped) == 0 {
		return nil
	}
	return deduped
}

func containsAnyFold(value string, needles []string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, needle := range needles {
		needle = strings.ToLower(strings.TrimSpace(needle))
		if needle != "" && strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func matchedKeywords(value string, keywords []string) []string {
	value = strings.ToLower(strings.TrimSpace(value))
	matches := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if keyword != "" && strings.Contains(value, keyword) {
			matches = append(matches, keyword)
		}
	}
	return dedupeStrings(matches)
}

func stringInSliceFold(values []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == target {
			return true
		}
	}
	return false
}

func normalizeStartOptions(options StartOptions) StartOptions {
	options.KickoffMode = strings.ToLower(strings.TrimSpace(options.KickoffMode))
	switch options.KickoffMode {
	case "none":
		options.KickoffMode = KickoffModeAbstain
	case "", KickoffModeAuto, KickoffModeMinimal, KickoffModeAbstain:
	default:
		options.KickoffMode = KickoffModeAuto
	}
	if options.KickoffMode == "" {
		options.KickoffMode = KickoffModeAuto
	}
	return options
}

func appendReasons(reasons []string, extras ...string) []string {
	for _, reason := range extras {
		reasons = appendReason(reasons, reason)
	}
	return reasons
}

func containsReason(reasons []string, target string) bool {
	for _, reason := range reasons {
		if reason == target {
			return true
		}
	}
	return false
}

func firstReason(reasons []string) string {
	if len(reasons) == 0 {
		return ""
	}
	return reasons[0]
}

func pluralizeCount(count int, noun string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", count, noun)
}
