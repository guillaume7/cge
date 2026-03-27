package copilot

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSessionStateCollectorCollectsUsageFromShutdownEvent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sessionDir := filepath.Join(root, "session-123")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "events.jsonl"), []byte(`{"type":"session.start","data":{"sessionId":"session-123"}}
{"type":"session.shutdown","data":{"currentModel":"gpt-5.4","modelMetrics":{"gpt-5.4":{"usage":{"inputTokens":1200,"outputTokens":300,"cacheReadTokens":400,"cacheWriteTokens":0}}}}}
`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	collector := NewSessionStateCollector(root)
	usage, err := collector.CollectSessionUsage(context.Background(), SessionUsageRequest{
		SessionID: "session-123",
		Model:     "gpt-5.4",
	})
	if err != nil {
		t.Fatalf("CollectSessionUsage returned error: %v", err)
	}
	if usage.InputTokens != 1200 {
		t.Fatalf("input_tokens = %d, want 1200", usage.InputTokens)
	}
	if usage.OutputTokens != 300 {
		t.Fatalf("output_tokens = %d, want 300", usage.OutputTokens)
	}
	if usage.TotalTokens != 1500 {
		t.Fatalf("total_tokens = %d, want 1500", usage.TotalTokens)
	}
	if usage.CacheReadTokens != 400 {
		t.Fatalf("cache_read_tokens = %d, want 400", usage.CacheReadTokens)
	}
}

func TestSessionStateCollectorFallsBackToSoleModelMetric(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	sessionDir := filepath.Join(root, "session-123")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "events.jsonl"), []byte(`{"type":"session.shutdown","data":{"modelMetrics":{"gpt-5.4":{"usage":{"inputTokens":700,"outputTokens":200,"cacheReadTokens":0,"cacheWriteTokens":0}}}}}
`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	collector := NewSessionStateCollector(root)
	usage, err := collector.CollectSessionUsage(context.Background(), SessionUsageRequest{
		SessionID: "session-123",
		Model:     "unknown-model",
	})
	if err != nil {
		t.Fatalf("CollectSessionUsage returned error: %v", err)
	}
	if usage.Model != "gpt-5.4" {
		t.Fatalf("model = %q, want gpt-5.4", usage.Model)
	}
}
