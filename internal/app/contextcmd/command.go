package contextcmd

import (
	"errors"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/guillaume-galp/cge/internal/app/attributionrecorder"
	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/app/contextprojector"
	"github.com/guillaume-galp/cge/internal/app/contextevaluator"
	"github.com/guillaume-galp/cge/internal/app/decisionengine"
	"github.com/guillaume-galp/cge/internal/app/retrieval"
	"github.com/guillaume-galp/cge/internal/infra/repo"
	"github.com/guillaume-galp/cge/internal/infra/textindex"
)

type Querier interface {
	Query(cmd *cobra.Command, workspace repo.Workspace, task string) (retrieval.ResultSet, error)
}

type graphQuerier struct {
	engine *retrieval.Engine
}

func (q graphQuerier) Query(cmd *cobra.Command, workspace repo.Workspace, task string) (retrieval.ResultSet, error) {
	if q.engine == nil {
		q.engine = retrieval.NewEngine(nil, nil)
	}
	return q.engine.Query(cmd.Context(), workspace, task)
}

func NewCommand(startDir string, manager *repo.Manager) *cobra.Command {
	return newCommand(startDir, manager, graphQuerier{engine: retrieval.NewEngine(nil, nil)}, contextprojector.NewProjector(), attributionrecorder.New())
}

func newCommand(startDir string, manager *repo.Manager, querier Querier, projector contextprojector.Projector, recorder attributionrecorder.Recorder) *cobra.Command {
	if querier == nil {
		querier = graphQuerier{engine: retrieval.NewEngine(nil, nil)}
	}

	var task string
	var file string
	var maxTokens int
	var output string

	cmd := &cobra.Command{
		Use:           "context",
		Short:         "Project prompt-ready context from the repo-local graph",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, err := cmdsupport.RequireWorkspace(cmd, startDir, manager)
			if err != nil {
				return handleContextError(cmd.OutOrStdout(), output, err)
			}

			input, err := cmdsupport.ResolveTextInput(cmd.InOrStdin(), task, file, "task", "--task")
			if err != nil {
				return handleContextError(cmd.OutOrStdout(), output, err)
			}

			if err := contextprojector.ValidateMaxTokens(maxTokens); err != nil {
				return handleContextError(cmd.OutOrStdout(), output, err)
			}

			rawResults, err := querier.Query(cmd, workspace, input.Value)
			if err != nil {
				return handleContextError(cmd.OutOrStdout(), output, err)
			}

			// Run the evaluator loop: Context Evaluator → Decision Engine →
			// Attribution Recorder. The decision bundle narrows the result set
			// before projection (AC1). Persistence failures do not block the
			// response — the decision envelope is still returned (AC3).
			evaluator := contextevaluator.NewEvaluator(contextevaluator.Config{})
			engine := decisionengine.NewWithDefaults()

			candidates := make([]contextevaluator.Candidate, len(rawResults.Results))
			for i, r := range rawResults.Results {
				candidates[i] = contextevaluator.CandidateFromRetrievalResult(r)
			}
			evalResult := evaluator.Evaluate(contextevaluator.EvaluateRequest{
				Task:       input.Value,
				Candidates: candidates,
			})
			decisionEnvelope, decideErr := engine.Decide(decisionengine.ContextDecisionRequest{
				EvaluationResult: evalResult,
			})
			if decideErr != nil {
				return handleContextError(cmd.OutOrStdout(), output, decideErr)
			}

			// Filter the original result set to those candidates selected by
			// the decision bundle, preserving retrieval metadata.
			effectiveResults := filterResultSetByBundle(rawResults, decisionEnvelope)

			contextEnvelope, err := projector.Project(effectiveResults, maxTokens)
			if err != nil {
				return handleContextError(cmd.OutOrStdout(), output, err)
			}

			// Generate and persist the attribution record. Persistence is
			// best-effort; a failure is logged but does not fail the command.
			sessionID := os.Getenv("GRAPH_SESSION_ID")
			attrRecord := recorder.Generate(decisionEnvelope, input.Value, sessionID)
			_ = recorder.Persist(workspace.WorkspacePath, attrRecord)

			response := resultEnvelope{
				Query: queryEnvelope{
					Task:   input.Value,
					Source: input.Source,
				},
				Index:    indexEnvelope{Status: rawResults.IndexStatus},
				Context:  contextEnvelope,
				Decision: decisionEnvelopeOutput{
					Outcome:       string(decisionEnvelope.Outcome),
					Confidence:    decisionEnvelope.Confidence,
					Attribution:   attrRecord.InlineSummary,
				},
			}

			return cmdsupport.WriteSuccess(cmd.OutOrStdout(), output, "context", response)
		},
	}

	cmd.Flags().StringVar(&task, "task", "", "Task text to project into compact context")
	cmd.Flags().StringVar(&file, "file", "", "File containing the task text")
	cmd.Flags().IntVar(&maxTokens, "max-tokens", 1200, "Maximum approximate token budget for projected context")
	cmd.Flags().StringVar(&output, "output", "", "Write the structured JSON response to a file instead of stdout")

	return cmd
}

// filterResultSetByBundle returns a ResultSet containing only the results
// whose entity IDs appear in the decision bundle. If the bundle is empty
// (e.g. abstain outcome), an empty ResultSet is returned. If the bundle
// covers all candidates, the original set is returned unchanged.
func filterResultSetByBundle(original retrieval.ResultSet, envelope decisionengine.DecisionEnvelope) retrieval.ResultSet {
	if len(envelope.Bundle) == 0 {
		return retrieval.ResultSet{IndexStatus: original.IndexStatus, Results: []retrieval.Result{}}
	}
	bundleIDs := make(map[string]struct{}, len(envelope.Bundle))
	for _, cs := range envelope.Bundle {
		bundleIDs[cs.CandidateID] = struct{}{}
	}
	filtered := make([]retrieval.Result, 0, len(envelope.Bundle))
	for _, r := range original.Results {
		if _, ok := bundleIDs[r.Entity.ID]; ok {
			filtered = append(filtered, r)
		}
	}
	return retrieval.ResultSet{IndexStatus: original.IndexStatus, Results: filtered}
}

type resultEnvelope struct {
	Query    queryEnvelope             `json:"query"`
	Index    indexEnvelope             `json:"index"`
	Context  contextprojector.Envelope `json:"context"`
	Decision decisionEnvelopeOutput    `json:"decision"`
}

// decisionEnvelopeOutput is the decision metadata included in the response.
// Existing consumers that do not parse this field remain unaffected (AC4).
type decisionEnvelopeOutput struct {
	Outcome    string                          `json:"outcome"`
	Confidence float64                         `json:"confidence"`
	Attribution attributionrecorder.InlineSummary `json:"attribution"`
}

type queryEnvelope struct {
	Task   string `json:"task"`
	Source string `json:"source"`
}

type indexEnvelope struct {
	Status string `json:"status"`
}

func handleContextError(w io.Writer, outputPath string, err error) error {
	if detail, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return cmdsupport.WriteFailure(w, outputPath, "context", detail, err)
	}

	var validationErr *contextprojector.ValidationError
	if errors.As(err, &validationErr) {
		return cmdsupport.WriteFailure(w, outputPath, "context", cmdsupport.ErrorDetail{
			Category: "validation_error",
			Type:     "validation_error",
			Code:     validationErr.Code,
			Message:  validationErr.Message,
			Details:  validationErr.Details,
		}, err)
	}

	var indexErr *textindex.Error
	if errors.As(err, &indexErr) {
		return cmdsupport.WriteFailure(w, outputPath, "context", cmdsupport.ErrorDetail{
			Category: "operational_error",
			Type:     "index_error",
			Code:     indexErr.Code,
			Message:  indexErr.Message,
			Details:  indexErr.Details,
		}, err)
	}

	return cmdsupport.WriteFailure(w, outputPath, "context", cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "context_error",
		Code:     "context_failed",
		Message:  "graph context projection failed",
		Details: map[string]any{
			"reason": err.Error(),
		},
	}, err)
}
