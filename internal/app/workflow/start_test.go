package workflow

import (
	"testing"

	"github.com/guillaume-galp/cge/internal/app/contextprojector"
	"github.com/guillaume-galp/cge/internal/app/retrieval"
)

func TestCalibrateKickoffResultSetSuppressesWorkflowArtifactsForNonWorkflowTasks(t *testing.T) {
	t.Parallel()

	resultSet := retrieval.ResultSet{
		Results: []retrieval.Result{
			{
				Rank:  1,
				Score: 18,
				Entity: retrieval.Entity{
					ID:    "workflow-finish:recent-run:reasoning",
					Kind:  "ReasoningUnit",
					Title: "verify repo-local delegated workflow kickoff and handoff",
				},
			},
			{
				Rank:  2,
				Score: 15,
				Entity: retrieval.Entity{
					ID:    "workflow:delegated-graph-workflow.prompt",
					Kind:  "workflow_prompt",
					Title: "Graph-backed delegated workflow prompt snippet",
				},
			},
			{
				Rank:  3,
				Score: 14,
				Entity: retrieval.Entity{
					ID:    "doc:graph-stats",
					Kind:  "Document",
					Title: "Graph stats indicators",
				},
			},
		},
	}

	taskFamily := classifyKickoffTaskFamily("Verify graph stats counts and health indicators for a seeded graph.")
	policy := kickoffPolicyForFamily(taskFamily.Name)
	calibrated := calibrateKickoffResultSet(taskFamily, policy, resultSet)
	if len(calibrated.Results) != 1 {
		t.Fatalf("result count = %d, want 1 non-workflow result", len(calibrated.Results))
	}
	if got := calibrated.Results[0].Entity.ID; got != "doc:graph-stats" {
		t.Fatalf("remaining result id = %q, want doc:graph-stats", got)
	}
	if got := calibrated.Results[0].Rank; got != 1 {
		t.Fatalf("remaining result rank = %d, want 1", got)
	}
}

func TestCalibrateKickoffResultSetPreservesWorkflowArtifactsForWorkflowTasks(t *testing.T) {
	t.Parallel()

	resultSet := retrieval.ResultSet{
		Results: []retrieval.Result{
			{
				Rank:  1,
				Score: 18,
				Entity: retrieval.Entity{
					ID:    "workflow-finish:recent-run:reasoning",
					Kind:  "ReasoningUnit",
					Title: "implement delegated workflow finish handoff",
				},
			},
			{
				Rank:  2,
				Score: 15,
				Entity: retrieval.Entity{
					ID:    "workflow:delegated-graph-workflow.prompt",
					Kind:  "workflow_prompt",
					Title: "Graph-backed delegated workflow prompt snippet",
				},
			},
		},
	}

	taskFamily := classifyKickoffTaskFamily("Implement delegated workflow finish handoff persistence.")
	policy := kickoffPolicyForFamily(taskFamily.Name)
	calibrated := calibrateKickoffResultSet(taskFamily, policy, resultSet)
	if len(calibrated.Results) != 2 {
		t.Fatalf("result count = %d, want workflow artifacts preserved", len(calibrated.Results))
	}
}

func TestCalibrateKickoffResultSetAppliesFamilyAllowlist(t *testing.T) {
	t.Parallel()

	resultSet := retrieval.ResultSet{
		Results: []retrieval.Result{
			{
				Rank:  1,
				Score: 18,
				Entity: retrieval.Entity{
					ID:    "story:writeback",
					Kind:  "UserStory",
					Title: "Writeback story",
				},
			},
			{
				Rank:  2,
				Score: 17,
				Entity: retrieval.Entity{
					ID:    "doc:audit-provenance",
					Kind:  "Document",
					Title: "Audit provenance guidance",
				},
			},
		},
	}

	taskFamily := classifyKickoffTaskFamily("Audit graph stats counts and verify indicators.")
	policy := kickoffPolicyForFamily(taskFamily.Name)
	calibrated := calibrateKickoffResultSet(taskFamily, policy, resultSet)
	if len(calibrated.Results) != 1 {
		t.Fatalf("result count = %d, want 1 allowlisted result", len(calibrated.Results))
	}
	if got := calibrated.Results[0].Entity.ID; got != "doc:audit-provenance" {
		t.Fatalf("remaining result id = %q, want doc:audit-provenance", got)
	}
}

func TestCalibrateKickoffRecommendationDowngradesInspectHygieneForGroundedNonHygieneTask(t *testing.T) {
	t.Parallel()

	recommendation, reasons := calibrateKickoffRecommendation(
		classifyKickoffTaskFamily("Report context projection under budget."),
		KickoffAdvisoryState{ReasonCodes: []string{"family_policy_default_no_kickoff"}},
		RecommendationInspectHygiene,
		[]string{"contradictions_detected", "duplication_rate_high"},
		retrieval.ResultSet{
			Results: []retrieval.Result{
				{
					Score:        14,
					MatchedTerms: []string{"context", "budget"},
					Entity: retrieval.Entity{
						ID:    "doc:context-projector",
						Kind:  "Document",
						Title: "Context projector",
					},
				},
			},
		},
		contextprojector.Envelope{
			Results: []contextprojector.Result{
				{
					Rank:  1,
					Score: 14,
					Entity: contextprojector.Entity{
						ID:    "doc:context-projector",
						Kind:  "Document",
						Title: "Context projector",
					},
				},
			},
		},
	)

	if recommendation != RecommendationProceed {
		t.Fatalf("recommendation = %q, want %q", recommendation, RecommendationProceed)
	}
	if len(reasons) != 2 || reasons[0] != "task_specific_context_grounded" || reasons[1] != "graph_hygiene_advisory" {
		t.Fatalf("reasons = %#v, want grounded advisory reasons", reasons)
	}
}

func TestCalibrateKickoffRecommendationKeepsInspectHygieneForHygieneTask(t *testing.T) {
	t.Parallel()

	recommendation, reasons := calibrateKickoffRecommendation(
		classifyKickoffTaskFamily("Verify graph stats counts and health indicators for a seeded graph."),
		KickoffAdvisoryState{},
		RecommendationInspectHygiene,
		[]string{"contradictions_detected", "duplication_rate_high"},
		retrieval.ResultSet{
			Results: []retrieval.Result{
				{
					Score: 12,
					Entity: retrieval.Entity{
						ID:    "doc:graph-stats",
						Kind:  "Document",
						Title: "Graph stats indicators",
					},
				},
			},
		},
		contextprojector.Envelope{
			Results: []contextprojector.Result{
				{
					Rank:  1,
					Score: 12,
					Entity: contextprojector.Entity{
						ID:    "doc:graph-stats",
						Kind:  "Document",
						Title: "Graph stats indicators",
					},
				},
			},
		},
	)

	if recommendation != RecommendationInspectHygiene {
		t.Fatalf("recommendation = %q, want %q", recommendation, RecommendationInspectHygiene)
	}
	if len(reasons) != 2 || reasons[0] != "contradictions_detected" || reasons[1] != "duplication_rate_high" {
		t.Fatalf("reasons = %#v, want unchanged hygiene reasons", reasons)
	}
}

func TestClassifyKickoffTaskFamilyReturnsExpectedFamilies(t *testing.T) {
	t.Parallel()

	cases := []struct {
		task string
		want string
	}{
		{task: "Implement delegated workflow start command", want: KickoffFamilyWorkflowContext},
		{task: "Write the production handler for kickoff classification", want: KickoffFamilyWriteProducing},
		{task: "Diagnose why kickoff returns low-context results", want: KickoffFamilyTroubleshooting},
		{task: "Audit graph stats counts and verify indicators", want: KickoffFamilyVerificationAudit},
		{task: "Produce a synthesis report of campaign findings", want: KickoffFamilyReportingSynthesis},
		{task: "Need help soon", want: KickoffFamilyAmbiguousTask},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			family := classifyKickoffTaskFamily(tc.task)
			if family.Name != tc.want {
				t.Fatalf("family = %q, want %q", family.Name, tc.want)
			}
		})
	}
}

func TestBuildKickoffContextAbstainsForReportingFamily(t *testing.T) {
	t.Parallel()

	taskFamily := classifyKickoffTaskFamily("Produce a synthesis report of campaign findings.")
	policy := kickoffPolicyForFamily(taskFamily.Name)
	contextState := buildKickoffContext(
		"Produce a synthesis report of campaign findings.",
		1200,
		RecommendationProceed,
		StartGraphState{WorkspaceInitialized: true, GraphAvailable: true},
		taskFamily,
		policy,
		KickoffAdvisoryState{
			RequestedMode:   KickoffModeAuto,
			EffectiveMode:   KickoffModeAbstain,
			ConfidenceLevel: "medium",
			ConfidenceScore: 0.5,
			ReasonCodes:     []string{"family_policy_default_no_kickoff"},
		},
		retrieval.ResultSet{},
		contextprojector.Envelope{MaxTokens: 1200, Results: []contextprojector.Result{}},
	)

	if contextState.Coverage != KickoffCoverageAbstained {
		t.Fatalf("coverage = %q, want %q", contextState.Coverage, KickoffCoverageAbstained)
	}
	if !contextState.Abstained {
		t.Fatal("expected abstained context state")
	}
	if contextState.AbstentionReason != "family_policy_default_no_kickoff" {
		t.Fatalf("abstention_reason = %q, want family policy reason", contextState.AbstentionReason)
	}
}

func TestDetermineKickoffAdvisorySupportsMinimalAndExplicitNone(t *testing.T) {
	t.Parallel()

	writeFamily := classifyKickoffTaskFamily("Write the production handler for kickoff classification.")
	policy := kickoffPolicyForFamily(writeFamily.Name)
	confidence := kickoffConfidenceAssessment{Level: "high", Score: 0.9}

	minimal := determineKickoffAdvisory(writeFamily, policy, KickoffModeMinimal, confidence)
	if minimal.EffectiveMode != KickoffModeMinimal {
		t.Fatalf("minimal effective mode = %q, want %q", minimal.EffectiveMode, KickoffModeMinimal)
	}

	none := determineKickoffAdvisory(writeFamily, policy, KickoffModeAbstain, confidence)
	if none.EffectiveMode != KickoffModeAbstain {
		t.Fatalf("none effective mode = %q, want %q", none.EffectiveMode, KickoffModeAbstain)
	}
	if none.NextStep != "proceed_with_fresh_context" {
		t.Fatalf("none next step = %q, want proceed_with_fresh_context", none.NextStep)
	}
}

func TestAnnotateInclusionReasonsAddsMachineReadableReasons(t *testing.T) {
	t.Parallel()

	taskFamily := classifyKickoffTaskFamily("Write the production handler for kickoff classification.")
	resultSet := retrieval.ResultSet{
		Results: []retrieval.Result{
			{
				Rank:         1,
				Score:        14,
				MatchedTerms: []string{"write", "kickoff"},
				Entity: retrieval.Entity{
					ID:    "doc:kickoff-policy",
					Kind:  "Document",
					Title: "Kickoff policy",
				},
			},
		},
	}
	envelope := contextprojector.Envelope{
		MaxTokens: 1200,
		Results: []contextprojector.Result{
			{
				Rank:  1,
				Score: 14,
				Entity: contextprojector.Entity{
					ID:    "doc:kickoff-policy",
					Kind:  "Document",
					Title: "Kickoff policy",
				},
			},
		},
	}

	annotateInclusionReasons(taskFamily, resultSet, &envelope)
	if envelope.Results[0].InclusionReason == "" {
		t.Fatal("expected inclusion reason to be populated")
	}
}
