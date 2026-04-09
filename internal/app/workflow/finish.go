package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/guillaume-galp/cge/internal/app/attribution"
	"github.com/guillaume-galp/cge/internal/app/cmdsupport"
	"github.com/guillaume-galp/cge/internal/app/contextevaluator"
	"github.com/guillaume-galp/cge/internal/app/decisionengine"
	graphpayload "github.com/guillaume-galp/cge/internal/domain/payload"
	"github.com/guillaume-galp/cge/internal/infra/kuzu"
	"github.com/guillaume-galp/cge/internal/infra/repo"
)

const (
	FinishWriteStatusApplied = "applied"
	FinishWriteStatusNoOp    = "noop"
	FinishHandoffStatusReady = "handoff_ready"

	ExecutionTelemetryStatusComplete    = "complete"
	ExecutionTelemetryStatusPartial     = "partial"
	ExecutionTelemetryStatusUnavailable = "unavailable"
	ExecutionTelemetryPropertyKey       = "execution_usage"
	ExecutionTelemetrySourceFinishInput = "workflow_finish_payload"
)

type FinishInput struct {
	SchemaVersion    string                  `json:"schema_version"`
	Metadata         graphpayload.Metadata   `json:"metadata"`
	Task             string                  `json:"task"`
	Summary          string                  `json:"summary"`
	Decisions        []FinishDecision        `json:"decisions"`
	ChangedArtifacts []FinishChangedArtifact `json:"changed_artifacts"`
	FollowUp         []FinishFollowUp        `json:"follow_up"`
}

type FinishDecision struct {
	Summary   string `json:"summary"`
	Rationale string `json:"rationale,omitempty"`
	Status    string `json:"status,omitempty"`
}

type FinishChangedArtifact struct {
	Path       string `json:"path"`
	Summary    string `json:"summary"`
	ChangeType string `json:"change_type,omitempty"`
	Language   string `json:"language,omitempty"`
}

type FinishFollowUp struct {
	Summary string `json:"summary"`
	Owner   string `json:"owner,omitempty"`
	Status  string `json:"status,omitempty"`
}

type FinishResult struct {
	BeforeRevision     kuzu.CurrentRevisionState `json:"before_revision"`
	AfterRevision      kuzu.CurrentRevisionState `json:"after_revision"`
	WriteSummary       FinishWriteSummary        `json:"write_summary"`
	HandoffBrief       *FinishHandoffBrief       `json:"handoff_brief,omitempty"`
	NoOp               *FinishNoOpResult         `json:"no_op,omitempty"`
	ExecutionTelemetry *ExecutionTelemetry       `json:"execution_telemetry,omitempty"`
	// WriteGate is populated when the evaluator loop gates a workflow finish
	// write. Nil when write gating is disabled or no mutations were present.
	WriteGate *FinishWriteGate `json:"write_gate,omitempty"`
}

// FinishWriteGate captures the evaluator loop decision for a workflow finish
// memory write (ADR-022 §2). It is included in FinishResult for transparency.
type FinishWriteGate struct {
	// Status is "approved", "deferred", or "skipped".
	Status string `json:"status"`
	// Confidence is the composite confidence score used to make the decision.
	Confidence float64 `json:"confidence"`
	// WriteThreshold is the threshold that was applied.
	WriteThreshold float64 `json:"write_threshold"`
	// DeferThreshold is the defer threshold that was applied.
	DeferThreshold float64 `json:"defer_threshold"`
	// Reason explains why the write was deferred or skipped (empty when approved).
	Reason string `json:"reason,omitempty"`
	// AttributionID is the ID of the persisted attribution record.
	AttributionID string `json:"attribution_id,omitempty"`
}

type FinishWriteSummary struct {
	Status   string                    `json:"status"`
	Nodes    kuzu.NodeWriteSummary     `json:"nodes"`
	Edges    kuzu.EdgeWriteSummary     `json:"edges"`
	Revision kuzu.RevisionWriteSummary `json:"revision,omitempty"`
}

type FinishHandoffBrief struct {
	Status           string   `json:"status"`
	Summary          string   `json:"summary"`
	Prompt           string   `json:"prompt"`
	Decisions        []string `json:"decisions,omitempty"`
	ChangedArtifacts []string `json:"changed_artifacts,omitempty"`
	FollowUp         []string `json:"follow_up,omitempty"`
}

type FinishNoOpResult struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type ExecutionTelemetry struct {
	MeasurementStatus string   `json:"measurement_status"`
	Source            string   `json:"source"`
	Provider          string   `json:"provider,omitempty"`
	InputTokens       *int     `json:"input_tokens,omitempty"`
	OutputTokens      *int     `json:"output_tokens,omitempty"`
	TotalTokens       *int     `json:"total_tokens,omitempty"`
	IncompleteReasons []string `json:"incomplete_reasons,omitempty"`
}

type finishPayloadDTO struct {
	SchemaVersion    *string         `json:"schema_version"`
	Metadata         *finishMetaDTO  `json:"metadata"`
	Task             *string         `json:"task"`
	Summary          *string         `json:"summary"`
	Decisions        json.RawMessage `json:"decisions"`
	ChangedArtifacts json.RawMessage `json:"changed_artifacts"`
	FollowUp         json.RawMessage `json:"follow_up"`
}

type finishMetaDTO struct {
	AgentID   *string                        `json:"agent_id"`
	SessionID *string                        `json:"session_id"`
	Timestamp *string                        `json:"timestamp"`
	Revision  *graphpayload.RevisionMetadata `json:"revision"`
}

type finishDecisionDTO struct {
	Summary   *string `json:"summary"`
	Rationale *string `json:"rationale"`
	Status    *string `json:"status"`
}

type finishChangedArtifactDTO struct {
	Path       *string `json:"path"`
	Summary    *string `json:"summary"`
	ChangeType *string `json:"change_type"`
	Language   *string `json:"language"`
}

type finishFollowUpDTO struct {
	Summary *string `json:"summary"`
	Owner   *string `json:"owner"`
	Status  *string `json:"status"`
}

func (s *Service) Finish(ctx context.Context, startDir, input string) (FinishResult, error) {
	if s == nil || s.manager == nil {
		return FinishResult{}, errors.New("workflow service is not configured")
	}
	if s.writer == nil {
		s.writer = kuzu.NewStore()
	}
	if s.reader == nil {
		s.reader = kuzu.NewStore()
	}

	workspace, err := s.manager.OpenWorkspace(ctx, startDir)
	if err != nil {
		return FinishResult{}, err
	}

	outcome, err := parseFinishInput(input)
	if err != nil {
		return FinishResult{}, err
	}
	executionTelemetry := extractExecutionTelemetry(outcome)

	beforeRevision, err := s.reader.CurrentRevision(ctx, workspace)
	if err != nil {
		return FinishResult{}, classifyFinishStateError("before_revision", err)
	}

	if finishMutationCount(outcome) == 0 {
		return FinishResult{
			BeforeRevision:     beforeRevision,
			AfterRevision:      beforeRevision,
			ExecutionTelemetry: executionTelemetry,
			WriteSummary: FinishWriteSummary{
				Status: FinishWriteStatusNoOp,
			},
			NoOp: &FinishNoOpResult{
				Status: FinishWriteStatusNoOp,
				Reason: "finish payload contains no durable graph updates",
			},
		}, nil
	}

	// Evaluate the write candidate against current graph state before committing.
	writeGate, gateDecision := s.evaluateWriteGate(ctx, workspace, outcome)
	if writeGate != nil && writeGate.Status != "approved" {
		// Write was gated out — persist attribution best-effort, return without writing.
		attrID := s.recordWriteAttribution(workspace, outcome, writeGate, gateDecision)
		writeGate.AttributionID = attrID
		return FinishResult{
			BeforeRevision:     beforeRevision,
			AfterRevision:      beforeRevision,
			ExecutionTelemetry: executionTelemetry,
			WriteSummary: FinishWriteSummary{
				Status: FinishWriteStatusNoOp,
			},
			WriteGate: writeGate,
		}, nil
	}

	envelope, err := s.buildFinishEnvelope(workspace, outcome)
	if err != nil {
		return FinishResult{}, err
	}

	summary, err := s.writer.Write(ctx, workspace, envelope)
	if err != nil {
		return FinishResult{}, classifyFinishPersistenceError(err)
	}

	afterRevision, err := s.reader.CurrentRevision(ctx, workspace)
	if err != nil {
		return FinishResult{}, classifyFinishStateError("after_revision", err)
	}

	result := FinishResult{
		BeforeRevision:     beforeRevision,
		AfterRevision:      afterRevision,
		ExecutionTelemetry: executionTelemetry,
		WriteSummary: FinishWriteSummary{
			Status:   FinishWriteStatusApplied,
			Nodes:    summary.Nodes,
			Edges:    summary.Edges,
			Revision: summary.Revision,
		},
	}
	// Persist attribution for approved writes best-effort; record the ID.
	if writeGate != nil {
		attrID := s.recordWriteAttribution(workspace, outcome, writeGate, gateDecision)
		writeGate.AttributionID = attrID
		result.WriteGate = writeGate
	}
	brief := buildFinishHandoffBrief(outcome, result)
	result.HandoffBrief = &brief
	return result, nil
}

// evaluateWriteGate runs the Context Evaluator and Decision Engine on the
// finish payload to determine whether the write should be approved, deferred,
// or skipped (ADR-022 §1-2). It returns (nil, nil) when the evaluator is not
// configured, preserving backward compatibility.
func (s *Service) evaluateWriteGate(ctx context.Context, workspace repo.Workspace, outcome FinishInput) (*FinishWriteGate, *decisionengine.DecisionEnvelope) {
	if s.evaluator == nil || s.decisionEngine == nil {
		return nil, nil
	}

	// Build task context from the finish payload.
	task := outcome.Task
	if task == "" {
		task = outcome.Summary
	}

	// Load current graph entities for consistency scoring.
	var graphState []contextevaluator.GraphState
	if graph, err := s.reader.ReadGraph(ctx, workspace); err == nil {
		graphState = graphEntitiesToEvaluatorState(graph)
	}

	// Represent the finish payload as an output candidate.
	candidate := contextevaluator.OutputCandidate{
		ID:      finishRootID(workspace, outcome),
		Summary: outcome.Summary,
		Content: finishReasoningContent(outcome),
	}

	evalResult := s.evaluator.EvaluateOutput(contextevaluator.EvaluateOutputRequest{
		Task:       task,
		Candidate:  candidate,
		GraphState: graphState,
	})

	envelope, err := s.decisionEngine.DecideWrite(decisionengine.WriteDecisionRequest{
		Output: evalResult,
	})
	if err != nil {
		// If the engine fails, fail open and proceed with write.
		return nil, nil
	}

	thresholds := s.decisionEngine.Thresholds()
	gate := &FinishWriteGate{
		Status:         envelope.WriteStatus,
		Confidence:     evalResult.Composite,
		WriteThreshold: thresholds.Write,
		DeferThreshold: thresholds.Defer,
		Reason:         envelope.WriteStatusReason,
	}
	return gate, &envelope
}

// recordWriteAttribution persists an attribution record for a write-gate
// decision best-effort. It returns the attribution record ID (or "" on error).
func (s *Service) recordWriteAttribution(workspace repo.Workspace, outcome FinishInput, gate *FinishWriteGate, envelope *decisionengine.DecisionEnvelope) string {
	if s.attribution == nil || gate == nil || envelope == nil {
		return ""
	}

	memDecision := attribution.MemoryDecision(gate.Status)
	var fates []attribution.CandidateFate
	for _, score := range envelope.Scores {
		fates = append(fates, attribution.CandidateFate{
			CandidateID: score.CandidateID,
			Fate:        score.Fate,
			Scores:      score.Scores,
			Composite:   score.Composite,
			Reason:      score.RejectionReason,
		})
	}

	wt := gate.WriteThreshold
	dt := gate.DeferThreshold
	rec := attribution.Record{
		Outcome:              string(envelope.Outcome),
		Task:                 outcome.Task,
		SessionID:            outcome.Metadata.SessionID,
		AggregateConfidence:  gate.Confidence,
		WriteThreshold:       &wt,
		DeferThreshold:       &dt,
		MemoryDecision:       &memDecision,
		MemoryDecisionReason: gate.Reason,
		CandidateFates:       fates,
	}
	recorded, err := s.attribution.Record(workspace, rec)
	if err != nil {
		return ""
	}
	return recorded.ID
}

// graphEntitiesToEvaluatorState converts kuzu graph entities into the
// evaluator's GraphState slice for consistency scoring.
func graphEntitiesToEvaluatorState(graph kuzu.Graph) []contextevaluator.GraphState {
	state := make([]contextevaluator.GraphState, 0, len(graph.Nodes))
	for _, entity := range graph.Nodes {
		gs := contextevaluator.GraphState{
			EntityID: entity.ID,
			Kind:     entity.Kind,
			Title:    entity.Title,
			Summary:  entity.Summary,
			Content:  entity.Content,
			RepoPath: entity.RepoPath,
			Tags:     entity.Tags,
		}
		state = append(state, gs)
	}
	return state
}

func ExtractExecutionTelemetryFromFinishPayload(input string) (*ExecutionTelemetry, error) {
	outcome, err := parseFinishInput(input)
	if err != nil {
		return nil, err
	}
	return extractExecutionTelemetry(outcome), nil
}

func ApplyExecutionTelemetryToFinishPayload(input string, telemetry ExecutionTelemetry) (string, error) {
	outcome, err := parseFinishInput(input)
	if err != nil {
		return "", err
	}
	if outcome.Metadata.Revision.Properties == nil {
		outcome.Metadata.Revision.Properties = map[string]any{}
	}
	outcome.Metadata.Revision.Properties[ExecutionTelemetryPropertyKey] = executionTelemetryProperties(telemetry)
	if outcome.Decisions == nil {
		outcome.Decisions = []FinishDecision{}
	}
	if outcome.ChangedArtifacts == nil {
		outcome.ChangedArtifacts = []FinishChangedArtifact{}
	}
	if outcome.FollowUp == nil {
		outcome.FollowUp = []FinishFollowUp{}
	}
	payload, err := json.MarshalIndent(outcome, "", "  ")
	if err != nil {
		return "", finishValidationError("invalid_finish_payload", "workflow finish payload could not be updated with execution telemetry", map[string]any{
			"reason": err.Error(),
		}, err)
	}
	return string(payload), nil
}

func parseFinishInput(input string) (FinishInput, error) {
	decoder := json.NewDecoder(strings.NewReader(strings.TrimSpace(input)))
	decoder.DisallowUnknownFields()

	dto := finishPayloadDTO{}
	if err := decoder.Decode(&dto); err != nil {
		return FinishInput{}, finishValidationError("malformed_json", "workflow finish payload is not valid JSON", map[string]any{
			"reason": err.Error(),
		}, err)
	}
	if err := ensureSingleJSONValue(decoder); err != nil {
		return FinishInput{}, finishValidationError("malformed_json", "workflow finish payload is not valid JSON", map[string]any{
			"reason": err.Error(),
		}, err)
	}

	if missing := missingFinishRequiredFields(dto); len(missing) > 0 {
		return FinishInput{}, finishValidationError("missing_required_fields", "workflow finish payload is missing required fields", map[string]any{
			"missing_fields": missing,
		}, nil)
	}

	schemaVersion := strings.TrimSpace(*dto.SchemaVersion)
	if schemaVersion != graphpayload.SchemaVersionV1 {
		return FinishInput{}, finishValidationError("unsupported_schema_version", "workflow finish payload schema version is not supported", map[string]any{
			"schema_version":            schemaVersion,
			"supported_schema_versions": []string{graphpayload.SchemaVersionV1},
		}, nil)
	}

	timestamp := strings.TrimSpace(*dto.Metadata.Timestamp)
	if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
		return FinishInput{}, finishValidationError("invalid_timestamp", "workflow finish metadata.timestamp must be a valid RFC3339 timestamp", map[string]any{
			"field": "metadata.timestamp",
			"value": timestamp,
		}, err)
	}

	outcome := FinishInput{
		SchemaVersion: schemaVersion,
		Metadata: graphpayload.Metadata{
			AgentID:   strings.TrimSpace(*dto.Metadata.AgentID),
			SessionID: strings.TrimSpace(*dto.Metadata.SessionID),
			Timestamp: timestamp,
			Revision:  cloneRevisionMetadata(dto.Metadata.Revision),
		},
		Task:    strings.TrimSpace(*dto.Task),
		Summary: strings.TrimSpace(*dto.Summary),
	}

	if err := validateSafeText("task", outcome.Task); err != nil {
		return FinishInput{}, err
	}
	if err := validateSafeText("summary", outcome.Summary); err != nil {
		return FinishInput{}, err
	}

	decisions, err := decodeFinishDecisions(dto.Decisions)
	if err != nil {
		return FinishInput{}, err
	}
	changedArtifacts, err := decodeFinishChangedArtifacts(dto.ChangedArtifacts)
	if err != nil {
		return FinishInput{}, err
	}
	followUp, err := decodeFinishFollowUp(dto.FollowUp)
	if err != nil {
		return FinishInput{}, err
	}

	outcome.Decisions = decisions
	outcome.ChangedArtifacts = changedArtifacts
	outcome.FollowUp = followUp
	return outcome, nil
}

func ensureSingleJSONValue(decoder *json.Decoder) error {
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	return fmt.Errorf("unexpected trailing content after JSON payload")
}

func missingFinishRequiredFields(dto finishPayloadDTO) []string {
	missing := []string{}
	if dto.SchemaVersion == nil || strings.TrimSpace(*dto.SchemaVersion) == "" {
		missing = append(missing, "schema_version")
	}
	if dto.Metadata == nil {
		return append(missing, "metadata.agent_id", "metadata.session_id", "metadata.timestamp", "task", "summary")
	}
	if dto.Metadata.AgentID == nil || strings.TrimSpace(*dto.Metadata.AgentID) == "" {
		missing = append(missing, "metadata.agent_id")
	}
	if dto.Metadata.SessionID == nil || strings.TrimSpace(*dto.Metadata.SessionID) == "" {
		missing = append(missing, "metadata.session_id")
	}
	if dto.Metadata.Timestamp == nil || strings.TrimSpace(*dto.Metadata.Timestamp) == "" {
		missing = append(missing, "metadata.timestamp")
	}
	if dto.Task == nil || strings.TrimSpace(*dto.Task) == "" {
		missing = append(missing, "task")
	}
	if dto.Summary == nil || strings.TrimSpace(*dto.Summary) == "" {
		missing = append(missing, "summary")
	}
	return missing
}

func decodeFinishDecisions(raw json.RawMessage) ([]FinishDecision, error) {
	items, err := decodeOptionalFinishArray[finishDecisionDTO](raw, "decisions")
	if err != nil {
		return nil, err
	}
	decisions := make([]FinishDecision, 0, len(items))
	for index, item := range items {
		if item.Summary == nil || strings.TrimSpace(*item.Summary) == "" {
			return nil, finishValidationError("invalid_finish_payload", "workflow finish decisions must include a non-empty summary", map[string]any{
				"field": "decisions.summary",
				"index": index,
			}, nil)
		}
		decision := FinishDecision{
			Summary: strings.TrimSpace(*item.Summary),
		}
		if err := validateSafeText("decisions.summary", decision.Summary); err != nil {
			return nil, withIndex(err, index)
		}
		if item.Rationale != nil {
			decision.Rationale = strings.TrimSpace(*item.Rationale)
			if err := validateSafeOptionalText("decisions.rationale", decision.Rationale); err != nil {
				return nil, withIndex(err, index)
			}
		}
		if item.Status != nil {
			decision.Status = strings.TrimSpace(*item.Status)
			if err := validateSafeOptionalText("decisions.status", decision.Status); err != nil {
				return nil, withIndex(err, index)
			}
		}
		decisions = append(decisions, decision)
	}
	return decisions, nil
}

func decodeFinishChangedArtifacts(raw json.RawMessage) ([]FinishChangedArtifact, error) {
	items, err := decodeOptionalFinishArray[finishChangedArtifactDTO](raw, "changed_artifacts")
	if err != nil {
		return nil, err
	}
	artifacts := make([]FinishChangedArtifact, 0, len(items))
	for index, item := range items {
		if item.Path == nil || strings.TrimSpace(*item.Path) == "" {
			return nil, finishValidationError("invalid_finish_payload", "workflow finish changed_artifacts must include a non-empty path", map[string]any{
				"field": "changed_artifacts.path",
				"index": index,
			}, nil)
		}
		if item.Summary == nil || strings.TrimSpace(*item.Summary) == "" {
			return nil, finishValidationError("invalid_finish_payload", "workflow finish changed_artifacts must include a non-empty summary", map[string]any{
				"field": "changed_artifacts.summary",
				"index": index,
			}, nil)
		}
		path, err := normalizeFinishRepoPath(strings.TrimSpace(*item.Path))
		if err != nil {
			return nil, withIndex(err, index)
		}
		artifact := FinishChangedArtifact{
			Path:    path,
			Summary: strings.TrimSpace(*item.Summary),
		}
		if err := validateSafeText("changed_artifacts.summary", artifact.Summary); err != nil {
			return nil, withIndex(err, index)
		}
		if item.ChangeType != nil {
			artifact.ChangeType = strings.TrimSpace(*item.ChangeType)
			if err := validateSafeOptionalText("changed_artifacts.change_type", artifact.ChangeType); err != nil {
				return nil, withIndex(err, index)
			}
		}
		if item.Language != nil {
			artifact.Language = strings.TrimSpace(*item.Language)
			if err := validateSafeOptionalText("changed_artifacts.language", artifact.Language); err != nil {
				return nil, withIndex(err, index)
			}
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts, nil
}

func decodeFinishFollowUp(raw json.RawMessage) ([]FinishFollowUp, error) {
	items, err := decodeOptionalFinishArray[finishFollowUpDTO](raw, "follow_up")
	if err != nil {
		return nil, err
	}
	followUp := make([]FinishFollowUp, 0, len(items))
	for index, item := range items {
		if item.Summary == nil || strings.TrimSpace(*item.Summary) == "" {
			return nil, finishValidationError("invalid_finish_payload", "workflow finish follow_up items must include a non-empty summary", map[string]any{
				"field": "follow_up.summary",
				"index": index,
			}, nil)
		}
		next := FinishFollowUp{Summary: strings.TrimSpace(*item.Summary)}
		if err := validateSafeText("follow_up.summary", next.Summary); err != nil {
			return nil, withIndex(err, index)
		}
		if item.Owner != nil {
			next.Owner = strings.TrimSpace(*item.Owner)
			if err := validateSafeOptionalText("follow_up.owner", next.Owner); err != nil {
				return nil, withIndex(err, index)
			}
		}
		if item.Status != nil {
			next.Status = strings.TrimSpace(*item.Status)
			if err := validateSafeOptionalText("follow_up.status", next.Status); err != nil {
				return nil, withIndex(err, index)
			}
		}
		followUp = append(followUp, next)
	}
	return followUp, nil
}

func decodeOptionalFinishArray[T any](raw json.RawMessage, field string) ([]T, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}

	decoder := json.NewDecoder(strings.NewReader(trimmed))
	var rawItems []json.RawMessage
	if err := decoder.Decode(&rawItems); err != nil {
		return nil, finishValidationError("invalid_finish_payload", fmt.Sprintf("workflow finish payload %s must be a JSON array", field), map[string]any{
			"field":  field,
			"reason": err.Error(),
		}, err)
	}
	if err := ensureSingleJSONValue(decoder); err != nil {
		return nil, finishValidationError("invalid_finish_payload", fmt.Sprintf("workflow finish payload %s must be a JSON array", field), map[string]any{
			"field":  field,
			"reason": err.Error(),
		}, err)
	}

	items := make([]T, 0, len(rawItems))
	for index, rawItem := range rawItems {
		item, err := decodeStrictFinishItem[T](rawItem, field)
		if err != nil {
			return nil, withIndex(err, index)
		}
		items = append(items, item)
	}
	return items, nil
}

func validateSafeText(field, value string) error {
	if strings.ContainsRune(value, '\x00') {
		return finishValidationError("unsafe_finish_payload", "workflow finish payload contains unsafe text content", map[string]any{
			"field": field,
		}, nil)
	}
	return nil
}

func validateSafeOptionalText(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return validateSafeText(field, value)
}

func extractExecutionTelemetry(outcome FinishInput) *ExecutionTelemetry {
	properties := outcome.Metadata.Revision.Properties
	if len(properties) == 0 {
		return nil
	}

	raw, ok := properties[ExecutionTelemetryPropertyKey]
	if !ok {
		return nil
	}

	payload, ok := raw.(map[string]any)
	if !ok {
		return &ExecutionTelemetry{
			MeasurementStatus: ExecutionTelemetryStatusUnavailable,
			Source:            ExecutionTelemetrySourceFinishInput,
			IncompleteReasons: []string{"invalid_execution_usage_shape"},
		}
	}

	telemetry := &ExecutionTelemetry{
		MeasurementStatus: strings.TrimSpace(stringValue(payload["measurement_status"])),
		Source:            strings.TrimSpace(stringValue(payload["source"])),
		Provider:          strings.TrimSpace(stringValue(payload["provider"])),
		IncompleteReasons: stringSliceValue(payload["incomplete_reasons"]),
	}
	if telemetry.Source == "" {
		telemetry.Source = ExecutionTelemetrySourceFinishInput
	}
	if value, ok := intValue(payload["input_tokens"]); ok {
		telemetry.InputTokens = intPointer(value)
	}
	if value, ok := intValue(payload["output_tokens"]); ok {
		telemetry.OutputTokens = intPointer(value)
	}
	if value, ok := intValue(payload["total_tokens"]); ok {
		telemetry.TotalTokens = intPointer(value)
	}
	if telemetry.MeasurementStatus == "" {
		switch {
		case telemetry.TotalTokens != nil && telemetry.InputTokens != nil && telemetry.OutputTokens != nil:
			telemetry.MeasurementStatus = ExecutionTelemetryStatusComplete
		case telemetry.TotalTokens != nil || telemetry.InputTokens != nil || telemetry.OutputTokens != nil:
			telemetry.MeasurementStatus = ExecutionTelemetryStatusPartial
		default:
			telemetry.MeasurementStatus = ExecutionTelemetryStatusUnavailable
		}
	}
	if telemetry.MeasurementStatus != ExecutionTelemetryStatusComplete && len(telemetry.IncompleteReasons) == 0 {
		telemetry.IncompleteReasons = []string{"token_measurement_incomplete"}
	}
	return telemetry
}

func executionTelemetryProperties(telemetry ExecutionTelemetry) map[string]any {
	properties := map[string]any{
		"measurement_status": strings.TrimSpace(telemetry.MeasurementStatus),
		"source":             strings.TrimSpace(telemetry.Source),
	}
	if telemetry.Provider != "" {
		properties["provider"] = strings.TrimSpace(telemetry.Provider)
	}
	if telemetry.InputTokens != nil {
		properties["input_tokens"] = *telemetry.InputTokens
	}
	if telemetry.OutputTokens != nil {
		properties["output_tokens"] = *telemetry.OutputTokens
	}
	if telemetry.TotalTokens != nil {
		properties["total_tokens"] = *telemetry.TotalTokens
	}
	if len(telemetry.IncompleteReasons) > 0 {
		properties["incomplete_reasons"] = append([]string(nil), telemetry.IncompleteReasons...)
	}
	return properties
}

func cloneRevisionMetadata(metadata *graphpayload.RevisionMetadata) graphpayload.RevisionMetadata {
	if metadata == nil {
		return graphpayload.RevisionMetadata{}
	}
	return graphpayload.RevisionMetadata{
		Reason:     strings.TrimSpace(metadata.Reason),
		Properties: cloneProperties(metadata.Properties),
	}
}

func cloneProperties(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func stringSliceValue(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if ok {
				text = strings.TrimSpace(text)
				if text != "" {
					items = append(items, text)
				}
			}
		}
		if len(items) == 0 {
			return nil
		}
		return items
	default:
		return nil
	}
}

func intValue(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		if typed == float64(int(typed)) {
			return int(typed), true
		}
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil {
			return int(parsed), true
		}
	}
	return 0, false
}

func intPointer(value int) *int {
	return &value
}

func normalizeFinishRepoPath(path string) (string, error) {
	if strings.ContainsRune(path, '\x00') {
		return "", finishValidationError("unsafe_repo_path", "workflow finish changed_artifacts path is unsafe", map[string]any{
			"field": "changed_artifacts.path",
			"path":  path,
		}, nil)
	}
	repoPath := strings.ReplaceAll(path, `\`, "/")
	if filepath.IsAbs(repoPath) {
		return "", finishValidationError("unsafe_repo_path", "workflow finish changed_artifacts path must be repo-relative", map[string]any{
			"field": "changed_artifacts.path",
			"path":  path,
		}, nil)
	}
	normalized := filepath.ToSlash(filepath.Clean(repoPath))
	if normalized == "." || normalized == "" || normalized == ".." || strings.HasPrefix(normalized, "../") {
		return "", finishValidationError("unsafe_repo_path", "workflow finish changed_artifacts path must stay within the repository", map[string]any{
			"field": "changed_artifacts.path",
			"path":  path,
		}, nil)
	}
	return normalized, nil
}

func decodeStrictFinishItem[T any](raw json.RawMessage, field string) (T, error) {
	var item T

	decoder := json.NewDecoder(strings.NewReader(strings.TrimSpace(string(raw))))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&item); err != nil {
		return item, finishValidationError("invalid_finish_payload", fmt.Sprintf("workflow finish payload %s contains invalid items", field), map[string]any{
			"field":  field,
			"reason": err.Error(),
		}, err)
	}
	if err := ensureSingleJSONValue(decoder); err != nil {
		return item, finishValidationError("invalid_finish_payload", fmt.Sprintf("workflow finish payload %s contains invalid items", field), map[string]any{
			"field":  field,
			"reason": err.Error(),
		}, err)
	}
	return item, nil
}

func finishValidationError(code, message string, details map[string]any, err error) error {
	return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
		Category: "validation_error",
		Type:     "validation_error",
		Code:     code,
		Message:  message,
		Details:  details,
	}, err)
}

func withIndex(err error, index int) error {
	detail, ok := cmdsupport.ErrorDetailFromError(err)
	if !ok {
		return err
	}
	if detail.Details == nil {
		detail.Details = map[string]any{}
	}
	if _, exists := detail.Details["index"]; !exists {
		detail.Details["index"] = index
	}
	return cmdsupport.NewCommandError(detail, err)
}

func finishMutationCount(input FinishInput) int {
	return len(input.Decisions) + len(input.ChangedArtifacts) + len(input.FollowUp)
}

func (s *Service) buildFinishEnvelope(workspace repo.Workspace, outcome FinishInput) (graphpayload.Envelope, error) {
	nodes, edges := finishObjects(workspace, outcome)
	rawNodes, err := marshalSeedObjects(nodes)
	if err != nil {
		return graphpayload.Envelope{}, classifyFinishEnvelopeError(err)
	}
	rawEdges, err := marshalSeedObjects(edges)
	if err != nil {
		return graphpayload.Envelope{}, classifyFinishEnvelopeError(err)
	}

	return graphpayload.Envelope{
		SchemaVersion: graphpayload.SchemaVersionV1,
		Metadata: graphpayload.Metadata{
			AgentID:   outcome.Metadata.AgentID,
			SessionID: outcome.Metadata.SessionID,
			Timestamp: outcome.Metadata.Timestamp,
			Revision: graphpayload.RevisionMetadata{
				Reason: fmt.Sprintf("Persist delegated workflow finish outcome for %s", outcome.Task),
				Properties: map[string]any{
					"workflow": map[string]any{
						"command":                "workflow.finish",
						"task":                   outcome.Task,
						"decision_count":         len(outcome.Decisions),
						"changed_artifact_count": len(outcome.ChangedArtifacts),
						"follow_up_count":        len(outcome.FollowUp),
					},
				},
			},
		},
		Nodes: rawNodes,
		Edges: rawEdges,
	}, nil
}

func finishObjects(workspace repo.Workspace, outcome FinishInput) ([]seedNode, []seedEdge) {
	rootID := finishRootID(workspace, outcome)
	nodes := make([]seedNode, 0, 1+len(outcome.Decisions)+len(outcome.ChangedArtifacts)+len(outcome.FollowUp))
	edges := make([]seedEdge, 0, len(outcome.Decisions)+len(outcome.ChangedArtifacts)+len(outcome.FollowUp))

	nodes = append(nodes, seedNode{
		ID:      rootID,
		Kind:    "ReasoningUnit",
		Title:   excerptText(outcome.Task, 120),
		Summary: excerptText(outcome.Summary, 240),
		Content: finishReasoningContent(outcome),
		Tags:    []string{"workflow", "handoff", "delegated"},
		Properties: map[string]any{
			"workflow_command":       "workflow.finish",
			"task":                   outcome.Task,
			"summary":                outcome.Summary,
			"decision_count":         len(outcome.Decisions),
			"changed_artifact_count": len(outcome.ChangedArtifacts),
			"follow_up_count":        len(outcome.FollowUp),
			"agent_id":               outcome.Metadata.AgentID,
			"session_id":             outcome.Metadata.SessionID,
			"timestamp":              outcome.Metadata.Timestamp,
		},
	})

	for index, decision := range outcome.Decisions {
		decisionID := fmt.Sprintf("%s:decision:%02d", rootID, index+1)
		nodes = append(nodes, seedNode{
			ID:      decisionID,
			Kind:    "Decision",
			Title:   excerptText(decision.Summary, 120),
			Summary: excerptText(firstNonEmpty(decision.Rationale, decision.Summary), 240),
			Tags:    []string{"workflow", "decision"},
			Properties: map[string]any{
				"workflow_command": "workflow.finish",
				"decision":         decision.Summary,
				"rationale":        decision.Rationale,
				"status":           decision.Status,
			},
		})
		edges = append(edges, seedEdge{
			From: rootID,
			Kind: "RELATES_TO",
			To:   decisionID,
			Properties: map[string]any{
				"relation": "decision",
			},
		})
	}

	for index, artifact := range outcome.ChangedArtifacts {
		artifactID := fmt.Sprintf("%s:artifact:%02d", rootID, index+1)
		node := seedNode{
			ID:       artifactID,
			Kind:     "Artifact",
			Title:    artifact.Path,
			Summary:  excerptText(artifact.Summary, 240),
			RepoPath: artifact.Path,
			Tags:     []string{"workflow", "artifact"},
			Properties: map[string]any{
				"workflow_command": "workflow.finish",
				"path":             artifact.Path,
				"change_type":      artifact.ChangeType,
				"summary":          artifact.Summary,
			},
		}
		if artifact.Language != "" {
			node.Properties["language"] = artifact.Language
			node.Language = artifact.Language
		}
		nodes = append(nodes, node)
		edges = append(edges, seedEdge{
			From: rootID,
			Kind: "ABOUT",
			To:   artifactID,
			Properties: map[string]any{
				"relation": "changed_artifact",
			},
		})
	}

	for index, next := range outcome.FollowUp {
		followUpID := fmt.Sprintf("%s:follow-up:%02d", rootID, index+1)
		nodes = append(nodes, seedNode{
			ID:      followUpID,
			Kind:    "Task",
			Title:   excerptText(next.Summary, 120),
			Summary: excerptText(next.Summary, 240),
			Tags:    []string{"workflow", "follow-up"},
			Properties: map[string]any{
				"workflow_command": "workflow.finish",
				"owner":            next.Owner,
				"status":           next.Status,
			},
		})
		edges = append(edges, seedEdge{
			From: rootID,
			Kind: "DEPENDS_ON",
			To:   followUpID,
			Properties: map[string]any{
				"relation": "follow_up",
			},
		})
	}

	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].Kind != edges[j].Kind {
			return edges[i].Kind < edges[j].Kind
		}
		return edges[i].To < edges[j].To
	})
	return nodes, edges
}

func finishRootID(workspace repo.Workspace, outcome FinishInput) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		workspace.Config.Repository.ID,
		outcome.Metadata.SessionID,
		outcome.Metadata.Timestamp,
		outcome.Task,
		outcome.Summary,
	}, "\n")))
	token := sanitizeIDToken(outcome.Metadata.SessionID)
	if token == "" {
		token = "session"
	}
	return fmt.Sprintf("workflow-finish:%s:%s:%s", workspace.Config.Repository.ID, token, hex.EncodeToString(sum[:6]))
}

func sanitizeIDToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	builder := strings.Builder{}
	builder.Grow(len(value))
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		case r == '-', r == '_', r == ':', r == '/':
			if !lastDash && builder.Len() > 0 {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	token := strings.Trim(builder.String(), "-")
	if len(token) > 24 {
		return token[:24]
	}
	return token
}

func finishReasoningContent(outcome FinishInput) string {
	sections := []string{fmt.Sprintf("Task: %s", outcome.Task), "", fmt.Sprintf("Summary: %s", outcome.Summary)}
	if len(outcome.Decisions) > 0 {
		sections = append(sections, "", "Decisions:")
		for _, decision := range outcome.Decisions {
			line := "- " + decision.Summary
			if decision.Rationale != "" {
				line += " — " + decision.Rationale
			}
			sections = append(sections, line)
		}
	}
	if len(outcome.ChangedArtifacts) > 0 {
		sections = append(sections, "", "Changed artifacts:")
		for _, artifact := range outcome.ChangedArtifacts {
			line := fmt.Sprintf("- %s: %s", artifact.Path, artifact.Summary)
			sections = append(sections, line)
		}
	}
	if len(outcome.FollowUp) > 0 {
		sections = append(sections, "", "Follow-up:")
		for _, next := range outcome.FollowUp {
			sections = append(sections, "- "+next.Summary)
		}
	}
	return strings.Join(sections, "\n")
}

func buildFinishHandoffBrief(outcome FinishInput, result FinishResult) FinishHandoffBrief {
	decisionLines := make([]string, 0, len(outcome.Decisions))
	for _, decision := range outcome.Decisions {
		line := decision.Summary
		if decision.Status != "" {
			line = fmt.Sprintf("%s (%s)", line, decision.Status)
		}
		decisionLines = append(decisionLines, line)
	}
	artifactLines := make([]string, 0, len(outcome.ChangedArtifacts))
	for _, artifact := range outcome.ChangedArtifacts {
		line := fmt.Sprintf("%s — %s", artifact.Path, artifact.Summary)
		if artifact.ChangeType != "" {
			line = fmt.Sprintf("%s [%s]", line, artifact.ChangeType)
		}
		artifactLines = append(artifactLines, line)
	}
	followUpLines := make([]string, 0, len(outcome.FollowUp))
	for _, next := range outcome.FollowUp {
		line := next.Summary
		if next.Owner != "" {
			line = fmt.Sprintf("%s (owner: %s)", line, next.Owner)
		}
		if next.Status != "" {
			line = fmt.Sprintf("%s [%s]", line, next.Status)
		}
		followUpLines = append(followUpLines, line)
	}

	summary := fmt.Sprintf(
		"Persisted delegated task outcome for %q from revision %s to %s.",
		outcome.Task,
		finishRevisionLabel(result.BeforeRevision),
		finishRevisionLabel(result.AfterRevision),
	)
	lines := []string{
		fmt.Sprintf("Task complete: %s", outcome.Task),
		"",
		fmt.Sprintf("Outcome summary: %s", outcome.Summary),
		"",
		"Revision anchors:",
		fmt.Sprintf("- before: %s", finishRevisionLabel(result.BeforeRevision)),
		fmt.Sprintf("- after: %s", finishRevisionLabel(result.AfterRevision)),
		"",
		"Graph write summary:",
		fmt.Sprintf("- nodes created: %d", result.WriteSummary.Nodes.CreatedCount),
		fmt.Sprintf("- nodes updated: %d", result.WriteSummary.Nodes.UpdatedCount),
		fmt.Sprintf("- edges created: %d", result.WriteSummary.Edges.CreatedCount),
		fmt.Sprintf("- edges updated: %d", result.WriteSummary.Edges.UpdatedCount),
	}
	if len(decisionLines) > 0 {
		lines = append(lines, "", "Key decisions:")
		for _, line := range decisionLines {
			lines = append(lines, "- "+line)
		}
	}
	if len(artifactLines) > 0 {
		lines = append(lines, "", "Changed artifacts:")
		for _, line := range artifactLines {
			lines = append(lines, "- "+line)
		}
	}
	lines = append(lines, "", "Next-agent brief:")
	if len(followUpLines) == 0 {
		lines = append(lines, "- No explicit follow-up items were captured in this handoff.")
	} else {
		for _, line := range followUpLines {
			lines = append(lines, "- "+line)
		}
	}

	return FinishHandoffBrief{
		Status:           FinishHandoffStatusReady,
		Summary:          summary,
		Prompt:           strings.Join(lines, "\n"),
		Decisions:        decisionLines,
		ChangedArtifacts: artifactLines,
		FollowUp:         followUpLines,
	}
}

func finishRevisionLabel(state kuzu.CurrentRevisionState) string {
	if !state.Exists {
		return "none"
	}
	if state.Revision.ID != "" && state.Revision.Anchor != "" {
		return fmt.Sprintf("%s (%s)", state.Revision.ID, state.Revision.Anchor)
	}
	if state.Revision.ID != "" {
		return state.Revision.ID
	}
	if state.Revision.Anchor != "" {
		return state.Revision.Anchor
	}
	return "present"
}

func classifyFinishEnvelopeError(err error) error {
	if _, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return err
	}
	return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "workflow_error",
		Code:     "workflow_finish_envelope_failed",
		Message:  "workflow finish could not assemble a graph write envelope",
		Details: map[string]any{
			"reason": err.Error(),
		},
	}, err)
}

func classifyFinishPersistenceError(err error) error {
	if _, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return err
	}
	var persistenceErr *kuzu.PersistenceError
	if errors.As(err, &persistenceErr) {
		details := map[string]any{}
		for key, value := range persistenceErr.Details {
			details[key] = value
		}
		details["cause_code"] = persistenceErr.Code
		return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
			Category: "operational_error",
			Type:     "persistence_error",
			Code:     "workflow_finish_persistence_failed",
			Message:  "workflow finish could not persist durable graph memory",
			Details:  details,
		}, err)
	}
	return cmdsupport.NewCommandError(cmdsupport.ErrorDetail{
		Category: "operational_error",
		Type:     "workflow_error",
		Code:     "workflow_finish_failed",
		Message:  "workflow finish failed",
		Details: map[string]any{
			"reason": err.Error(),
		},
	}, err)
}

func classifyFinishStateError(stage string, err error) error {
	if _, ok := cmdsupport.ErrorDetailFromError(err); ok {
		return err
	}
	var persistenceErr *kuzu.PersistenceError
	if errors.As(err, &persistenceErr) {
		details := map[string]any{"stage": stage}
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
		Code:     "workflow_finish_state_failed",
		Message:  "workflow finish could not inspect graph revision state",
		Details: map[string]any{
			"stage":  stage,
			"reason": err.Error(),
		},
	}, err)
}
