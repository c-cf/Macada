package context

import (
	"encoding/json"
	"testing"

	"github.com/c-cf/macada/internal/domain"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name string
		text string
		want int // approximate
	}{
		{"empty", "", 0},
		{"short", "hello", 2},
		{"medium", "The quick brown fox jumps over the lazy dog.", 11},
		{"long", string(make([]byte, 400)), 100},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := EstimateTokens(tc.text)
			if got != tc.want {
				t.Errorf("EstimateTokens(%q) = %d, want %d", tc.text[:min(len(tc.text), 20)], got, tc.want)
			}
		})
	}
}

func TestEstimateEventsTokens(t *testing.T) {
	events := []*domain.Event{
		{Type: "user.message", Payload: json.RawMessage(`{"content":"hello world"}`)},
		{Type: "agent.message", Payload: json.RawMessage(`{"content":"hi there"}`)},
	}

	tokens := EstimateEventsTokens(events)
	if tokens <= 0 {
		t.Errorf("expected positive token count, got %d", tokens)
	}
}

func TestEstimateMessagesTokens(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: []ContentBlock{{Type: "text", Text: "What is Go?"}}},
		{Role: "assistant", Content: []ContentBlock{{Type: "text", Text: "Go is a programming language."}}},
	}

	tokens := EstimateMessagesTokens(messages)
	if tokens <= 0 {
		t.Errorf("expected positive token count, got %d", tokens)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
