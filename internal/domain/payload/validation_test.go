package payload_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/guillaume-galp/cge/internal/domain/payload"
)

func TestParseAndValidateAcceptsValidPayload(t *testing.T) {
	t.Parallel()

	envelope, err := payload.ParseAndValidate(`{
  "schema_version": "v1",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-42",
    "timestamp": "2026-03-21T14:00:00Z",
    "revision": {
      "reason": "Refresh stale story summary",
      "properties": {
        "workflow": "cleanup"
      }
    }
  },
  "nodes": [],
  "edges": []
}`)
	if err != nil {
		t.Fatalf("ParseAndValidate returned error: %v", err)
	}

	if envelope.SchemaVersion != payload.SchemaVersionV1 {
		t.Fatalf("schema version = %q, want %q", envelope.SchemaVersion, payload.SchemaVersionV1)
	}
	if envelope.Metadata.AgentID != "developer" {
		t.Fatalf("agent id = %q, want developer", envelope.Metadata.AgentID)
	}
	if envelope.Metadata.SessionID != "sess-42" {
		t.Fatalf("session id = %q, want sess-42", envelope.Metadata.SessionID)
	}
	if envelope.Metadata.Timestamp != "2026-03-21T14:00:00Z" {
		t.Fatalf("timestamp = %q, want 2026-03-21T14:00:00Z", envelope.Metadata.Timestamp)
	}
	if envelope.Metadata.Revision.Reason != "Refresh stale story summary" {
		t.Fatalf("revision reason = %q, want Refresh stale story summary", envelope.Metadata.Revision.Reason)
	}
	if got := envelope.Metadata.Revision.Properties["workflow"]; got != "cleanup" {
		t.Fatalf("revision workflow = %#v, want cleanup", got)
	}
}

func TestParseAndValidateRejectsMissingProvenance(t *testing.T) {
	t.Parallel()

	_, err := payload.ParseAndValidate(`{
  "schema_version": "v1",
  "metadata": {
    "timestamp": "2026-03-21T14:00:00Z"
  },
  "nodes": [],
  "edges": []
}`)
	validationErr := requireValidationError(t, err)

	if validationErr.Code != "missing_required_fields" {
		t.Fatalf("code = %q, want missing_required_fields", validationErr.Code)
	}

	gotMissing, ok := validationErr.Details["missing_fields"].([]string)
	if !ok {
		t.Fatalf("missing_fields type = %T, want []string", validationErr.Details["missing_fields"])
	}

	wantMissing := []string{"metadata.agent_id", "metadata.session_id"}
	if !reflect.DeepEqual(gotMissing, wantMissing) {
		t.Fatalf("missing fields = %#v, want %#v", gotMissing, wantMissing)
	}
}

func TestParseAndValidateRejectsUnsupportedSchemaVersion(t *testing.T) {
	t.Parallel()

	_, err := payload.ParseAndValidate(`{
  "schema_version": "v2",
  "metadata": {
    "agent_id": "developer",
    "session_id": "sess-42",
    "timestamp": "2026-03-21T14:00:00Z"
  },
  "nodes": [],
  "edges": []
}`)
	validationErr := requireValidationError(t, err)

	if validationErr.Code != "unsupported_schema_version" {
		t.Fatalf("code = %q, want unsupported_schema_version", validationErr.Code)
	}
	if got := validationErr.Details["schema_version"]; got != "v2" {
		t.Fatalf("schema_version detail = %#v, want v2", got)
	}
}

func TestParseAndValidateRejectsMalformedJSON(t *testing.T) {
	t.Parallel()

	_, err := payload.ParseAndValidate(`{"schema_version":"v1"`)
	validationErr := requireValidationError(t, err)

	if validationErr.Code != "malformed_json" {
		t.Fatalf("code = %q, want malformed_json", validationErr.Code)
	}
}

func requireValidationError(t *testing.T, err error) *payload.ValidationError {
	t.Helper()

	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var validationErr *payload.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error type = %T, want *payload.ValidationError", err)
	}

	return validationErr
}
