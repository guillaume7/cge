package payload

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
)

const SchemaVersionV1 = "v1"

var supportedSchemaVersions = []string{SchemaVersionV1}

type Envelope struct {
	SchemaVersion string            `json:"schema_version"`
	Metadata      Metadata          `json:"metadata"`
	Nodes         []json.RawMessage `json:"nodes"`
	Edges         []json.RawMessage `json:"edges"`
}

type Metadata struct {
	AgentID   string           `json:"agent_id"`
	SessionID string           `json:"session_id"`
	Timestamp string           `json:"timestamp"`
	Revision  RevisionMetadata `json:"revision,omitempty"`
}

type RevisionMetadata struct {
	Reason     string         `json:"reason,omitempty"`
	Properties map[string]any `json:"properties,omitempty"`
}

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

func ParseAndValidate(input string) (Envelope, error) {
	dto := envelopeDTO{}
	decoder := json.NewDecoder(strings.NewReader(input))

	if err := decoder.Decode(&dto); err != nil {
		return Envelope{}, malformedJSONError(err)
	}

	if err := ensureSingleJSONValue(decoder); err != nil {
		return Envelope{}, malformedJSONError(err)
	}

	if missing := missingRequiredFields(dto); len(missing) > 0 {
		return Envelope{}, &ValidationError{
			Code:    "missing_required_fields",
			Message: "graph payload is missing required fields",
			Details: map[string]any{
				"missing_fields": missing,
			},
		}
	}

	schemaVersion := strings.TrimSpace(*dto.SchemaVersion)
	if !slices.Contains(supportedSchemaVersions, schemaVersion) {
		return Envelope{}, &ValidationError{
			Code:    "unsupported_schema_version",
			Message: "graph payload schema version is not supported",
			Details: map[string]any{
				"schema_version":            schemaVersion,
				"supported_schema_versions": supportedSchemaVersions,
			},
		}
	}

	timestamp := strings.TrimSpace(*dto.Metadata.Timestamp)
	if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
		return Envelope{}, &ValidationError{
			Code:    "invalid_timestamp",
			Message: "graph payload metadata.timestamp must be a valid RFC3339 timestamp",
			Details: map[string]any{
				"field": "metadata.timestamp",
				"value": timestamp,
			},
		}
	}

	nodes, err := decodeArrayField(dto.Nodes, "nodes")
	if err != nil {
		return Envelope{}, err
	}

	edges, err := decodeArrayField(dto.Edges, "edges")
	if err != nil {
		return Envelope{}, err
	}

	return Envelope{
		SchemaVersion: schemaVersion,
		Metadata: Metadata{
			AgentID:   strings.TrimSpace(*dto.Metadata.AgentID),
			SessionID: strings.TrimSpace(*dto.Metadata.SessionID),
			Timestamp: timestamp,
			Revision:  parseRevisionMetadata(dto.Metadata.Revision),
		},
		Nodes: nodes,
		Edges: edges,
	}, nil
}

type envelopeDTO struct {
	SchemaVersion *string         `json:"schema_version"`
	Metadata      *metadataDTO    `json:"metadata"`
	Nodes         json.RawMessage `json:"nodes"`
	Edges         json.RawMessage `json:"edges"`
}

type metadataDTO struct {
	AgentID   *string      `json:"agent_id"`
	SessionID *string      `json:"session_id"`
	Timestamp *string      `json:"timestamp"`
	Revision  *revisionDTO `json:"revision"`
}

type revisionDTO struct {
	Reason     *string        `json:"reason"`
	Properties map[string]any `json:"properties"`
}

func parseRevisionMetadata(dto *revisionDTO) RevisionMetadata {
	if dto == nil {
		return RevisionMetadata{}
	}

	revision := RevisionMetadata{
		Properties: cloneMap(dto.Properties),
	}
	if dto.Reason != nil {
		revision.Reason = strings.TrimSpace(*dto.Reason)
	}

	return revision
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

func missingRequiredFields(dto envelopeDTO) []string {
	missing := []string{}

	if dto.SchemaVersion == nil || strings.TrimSpace(*dto.SchemaVersion) == "" {
		missing = append(missing, "schema_version")
	}

	if dto.Metadata == nil {
		return append(missing, "metadata.agent_id", "metadata.session_id", "metadata.timestamp")
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

	return missing
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

func malformedJSONError(err error) error {
	return &ValidationError{
		Code:    "malformed_json",
		Message: "graph payload is not valid JSON",
		Details: map[string]any{
			"reason": err.Error(),
		},
	}
}

func decodeArrayField(raw json.RawMessage, field string) ([]json.RawMessage, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, &ValidationError{
			Code:    "invalid_payload_shape",
			Message: fmt.Sprintf("graph payload %s must be a JSON array", field),
			Details: map[string]any{
				"field": field,
			},
		}
	}

	values := []json.RawMessage{}
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, &ValidationError{
			Code:    "invalid_payload_shape",
			Message: fmt.Sprintf("graph payload %s must be a JSON array", field),
			Details: map[string]any{
				"field": field,
			},
		}
	}

	return values, nil
}
