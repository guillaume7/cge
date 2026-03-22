package cmdsupport

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const ResponseSchemaVersion = "v1"

type ResponseEnvelope struct {
	SchemaVersion string       `json:"schema_version"`
	Command       string       `json:"command"`
	Status        string       `json:"status"`
	Result        any          `json:"result,omitempty"`
	Error         *ErrorDetail `json:"error,omitempty"`
}

type ErrorDetail struct {
	Category string         `json:"category"`
	Type     string         `json:"type"`
	Code     string         `json:"code"`
	Message  string         `json:"message"`
	Details  map[string]any `json:"details,omitempty"`
}

type CommandError struct {
	Detail ErrorDetail
	Err    error
}

func NewCommandError(detail ErrorDetail, err error) *CommandError {
	return &CommandError{
		Detail: detail,
		Err:    err,
	}
}

func (e *CommandError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Detail.Message
}

func (e *CommandError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func ErrorDetailFromError(err error) (ErrorDetail, bool) {
	var commandErr *CommandError
	if !errors.As(err, &commandErr) {
		return ErrorDetail{}, false
	}
	return commandErr.Detail, true
}

func SuccessResponse(command string, result any) ResponseEnvelope {
	return ResponseEnvelope{
		SchemaVersion: ResponseSchemaVersion,
		Command:       command,
		Status:        "ok",
		Result:        result,
	}
}

func FailureResponse(command string, detail ErrorDetail) ResponseEnvelope {
	return ResponseEnvelope{
		SchemaVersion: ResponseSchemaVersion,
		Command:       command,
		Status:        "error",
		Error:         &detail,
	}
}

func WriteSuccess(w io.Writer, outputPath, command string, result any) error {
	return WriteJSON(w, outputPath, SuccessResponse(command, result))
}

func WriteFailure(w io.Writer, outputPath, command string, detail ErrorDetail, err error) error {
	if writeErr := WriteJSON(w, outputPath, FailureResponse(command, detail)); writeErr != nil {
		return writeErr
	}
	return &SilentError{Err: err}
}

func WriteJSON(w io.Writer, outputPath string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}
	data = append(data, '\n')

	outputPath = strings.TrimSpace(outputPath)
	if outputPath == "" {
		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("write response: %w", err)
		}
		return nil
	}

	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("write response file %s: %w", outputPath, err)
	}
	return nil
}
