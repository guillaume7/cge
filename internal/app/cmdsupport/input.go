package cmdsupport

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type TextInput struct {
	Value  string
	Source string
}

func ResolveTextInput(stdin io.Reader, inlineValue, filePath, noun, inlineFlag string) (TextInput, error) {
	stdinValue, hasStdin, err := readStdinIfPiped(stdin)
	if err != nil {
		return TextInput{}, NewCommandError(ErrorDetail{
			Category: "operational_error",
			Type:     "input_error",
			Code:     "stdin_read_failed",
			Message:  fmt.Sprintf("%s input could not be read from stdin", noun),
			Details: map[string]any{
				"input":  noun,
				"reason": err.Error(),
			},
		}, err)
	}

	inlineValue = strings.TrimSpace(inlineValue)
	filePath = strings.TrimSpace(filePath)
	acceptedSources := []string{inlineFlag, "--file", "stdin"}

	sourceCount := 0
	if inlineValue != "" {
		sourceCount++
	}
	if filePath != "" {
		sourceCount++
	}
	if hasStdin {
		sourceCount++
	}

	if sourceCount == 0 {
		return TextInput{}, NewCommandError(ErrorDetail{
			Category: "validation_error",
			Type:     "input_error",
			Code:     "missing_input",
			Message:  fmt.Sprintf("%s input is required", noun),
			Details: map[string]any{
				"input":            noun,
				"accepted_sources": acceptedSources,
			},
		}, fmt.Errorf("provide %s via %s, --file, or stdin", noun, inlineFlag))
	}
	if sourceCount > 1 {
		return TextInput{}, NewCommandError(ErrorDetail{
			Category: "validation_error",
			Type:     "input_error",
			Code:     "conflicting_input_sources",
			Message:  fmt.Sprintf("%s input must be provided by exactly one source", noun),
			Details: map[string]any{
				"input":            noun,
				"accepted_sources": acceptedSources,
			},
		}, fmt.Errorf("provide %s using only one of %s, --file, or stdin", noun, inlineFlag))
	}

	if inlineValue != "" {
		return TextInput{Value: inlineValue, Source: "flag"}, nil
	}
	if filePath != "" {
		payload, err := os.ReadFile(filePath)
		if err != nil {
			return TextInput{}, NewCommandError(ErrorDetail{
				Category: "operational_error",
				Type:     "input_error",
				Code:     "input_file_read_failed",
				Message:  fmt.Sprintf("%s file could not be read", noun),
				Details: map[string]any{
					"input":  noun,
					"path":   filePath,
					"reason": err.Error(),
				},
			}, err)
		}

		value := strings.TrimSpace(string(payload))
		if value == "" {
			return TextInput{}, NewCommandError(ErrorDetail{
				Category: "validation_error",
				Type:     "input_error",
				Code:     "empty_input_file",
				Message:  fmt.Sprintf("%s file is empty", noun),
				Details: map[string]any{
					"input": noun,
					"path":  filePath,
				},
			}, fmt.Errorf("%s file %s is empty", noun, filePath))
		}

		return TextInput{Value: value, Source: "file"}, nil
	}

	return TextInput{Value: stdinValue, Source: "stdin"}, nil
}

func readStdinIfPiped(stdin io.Reader) (string, bool, error) {
	if stdin == nil {
		return "", false, nil
	}

	if file, ok := stdin.(*os.File); ok {
		info, err := file.Stat()
		if err != nil {
			return "", false, fmt.Errorf("inspect stdin: %w", err)
		}
		if info.Mode()&os.ModeCharDevice != 0 {
			return "", false, nil
		}
	}

	payload, err := io.ReadAll(stdin)
	if err != nil {
		return "", false, fmt.Errorf("read stdin: %w", err)
	}

	value := strings.TrimSpace(string(payload))
	if value == "" {
		return "", false, nil
	}

	return value, true, nil
}
