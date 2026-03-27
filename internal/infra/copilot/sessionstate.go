package copilot

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	SessionUsageProvider = "github-copilot-cli"
	SessionUsageSource   = "copilot_session_state"
)

type SessionUsageRequest struct {
	SessionID        string
	Model            string
	SessionStateRoot string
}

type SessionUsage struct {
	SessionID        string
	Model            string
	InputTokens      int
	OutputTokens     int
	TotalTokens      int
	CacheReadTokens  int
	CacheWriteTokens int
}

type SessionStateCollector struct {
	defaultRoot string
}

func NewSessionStateCollector(defaultRoot string) *SessionStateCollector {
	return &SessionStateCollector{defaultRoot: strings.TrimSpace(defaultRoot)}
}

func DefaultSessionStateRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}
	return filepath.Join(home, ".copilot", "session-state"), nil
}

func (c *SessionStateCollector) CollectSessionUsage(ctx context.Context, request SessionUsageRequest) (SessionUsage, error) {
	sessionID := strings.TrimSpace(request.SessionID)
	if sessionID == "" {
		return SessionUsage{}, fmt.Errorf("session_id is required")
	}

	root := strings.TrimSpace(request.SessionStateRoot)
	if root == "" {
		root = c.defaultRoot
	}
	if root == "" {
		var err error
		root, err = DefaultSessionStateRoot()
		if err != nil {
			return SessionUsage{}, err
		}
	}

	path := filepath.Join(root, sessionID, "events.jsonl")
	file, err := os.Open(path)
	if err != nil {
		return SessionUsage{}, fmt.Errorf("open session event log %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	var shutdown *sessionShutdownData
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return SessionUsage{}, ctx.Err()
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event sessionEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return SessionUsage{}, fmt.Errorf("decode session event from %s: %w", path, err)
		}
		if event.Type != "session.shutdown" {
			continue
		}

		data := sessionShutdownData{}
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return SessionUsage{}, fmt.Errorf("decode session shutdown event from %s: %w", path, err)
		}
		shutdown = &data
	}
	if err := scanner.Err(); err != nil {
		return SessionUsage{}, fmt.Errorf("scan session event log %s: %w", path, err)
	}
	if shutdown == nil {
		return SessionUsage{}, fmt.Errorf("session shutdown event not found in %s", path)
	}

	model, metrics, err := resolveModelMetrics(strings.TrimSpace(request.Model), *shutdown)
	if err != nil {
		return SessionUsage{}, err
	}
	if metrics.Usage.InputTokens <= 0 {
		return SessionUsage{}, fmt.Errorf("session %s model %s reported no input tokens", sessionID, model)
	}
	if metrics.Usage.OutputTokens <= 0 {
		return SessionUsage{}, fmt.Errorf("session %s model %s reported no output tokens", sessionID, model)
	}

	return SessionUsage{
		SessionID:        sessionID,
		Model:            model,
		InputTokens:      metrics.Usage.InputTokens,
		OutputTokens:     metrics.Usage.OutputTokens,
		TotalTokens:      metrics.Usage.InputTokens + metrics.Usage.OutputTokens,
		CacheReadTokens:  metrics.Usage.CacheReadTokens,
		CacheWriteTokens: metrics.Usage.CacheWriteTokens,
	}, nil
}

type sessionEvent struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type sessionShutdownData struct {
	CurrentModel string                         `json:"currentModel"`
	ModelMetrics map[string]sessionModelMetrics `json:"modelMetrics"`
}

type sessionModelMetrics struct {
	Usage sessionUsageMetrics `json:"usage"`
}

type sessionUsageMetrics struct {
	InputTokens      int `json:"inputTokens"`
	OutputTokens     int `json:"outputTokens"`
	CacheReadTokens  int `json:"cacheReadTokens"`
	CacheWriteTokens int `json:"cacheWriteTokens"`
}

func resolveModelMetrics(requestedModel string, shutdown sessionShutdownData) (string, sessionModelMetrics, error) {
	if len(shutdown.ModelMetrics) == 0 {
		return "", sessionModelMetrics{}, fmt.Errorf("session shutdown event did not include model metrics")
	}
	if requestedModel != "" {
		if metrics, ok := shutdown.ModelMetrics[requestedModel]; ok {
			return requestedModel, metrics, nil
		}
	}
	if requestedModel == "" {
		if shutdown.CurrentModel != "" {
			if metrics, ok := shutdown.ModelMetrics[shutdown.CurrentModel]; ok {
				return shutdown.CurrentModel, metrics, nil
			}
		}
	}
	if len(shutdown.ModelMetrics) == 1 {
		for model, metrics := range shutdown.ModelMetrics {
			return model, metrics, nil
		}
	}

	available := make([]string, 0, len(shutdown.ModelMetrics))
	for model := range shutdown.ModelMetrics {
		available = append(available, model)
	}
	return "", sessionModelMetrics{}, fmt.Errorf("model metrics for %q not found; available models: %s", requestedModel, strings.Join(available, ", "))
}
