package context

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/cchu-code/managed-agents/internal/domain"
)

func makeEvent(typ string, payload string) *domain.Event {
	return &domain.Event{
		ID:          "test-" + typ,
		Type:        typ,
		ProcessedAt: time.Now().UTC(),
		Payload:     json.RawMessage(payload),
	}
}

func TestThinEvents_FewerThanKeep(t *testing.T) {
	events := []*domain.Event{
		makeEvent(domain.EventTypeUserMessage, `{"content":"hello"}`),
		makeEvent(domain.EventTypeAgentToolResult, `{"tool_use_id":"t1","content":"result1"}`),
		makeEvent(domain.EventTypeAgentToolResult, `{"tool_use_id":"t2","content":"result2"}`),
	}

	result := ThinEvents(events, 5)
	if len(result) != 3 {
		t.Fatalf("expected 3 events, got %d", len(result))
	}

	// All tool results should be preserved
	for _, evt := range result {
		if evt.Type == domain.EventTypeAgentToolResult {
			if strings.Contains(string(evt.Payload), "[content cleared]") {
				t.Error("tool result should not be thinned when fewer than keepRecent")
			}
		}
	}
}

func TestThinEvents_ThinsOldResults(t *testing.T) {
	events := []*domain.Event{
		makeEvent(domain.EventTypeAgentToolResult, `{"tool_use_id":"t1","content":"old result 1"}`),
		makeEvent(domain.EventTypeAgentToolResult, `{"tool_use_id":"t2","content":"old result 2"}`),
		makeEvent(domain.EventTypeAgentToolResult, `{"tool_use_id":"t3","content":"old result 3"}`),
		makeEvent(domain.EventTypeUserMessage, `{"content":"next question"}`),
		makeEvent(domain.EventTypeAgentToolResult, `{"tool_use_id":"t4","content":"recent result"}`),
	}

	result := ThinEvents(events, 2)

	// First 2 tool results should be thinned, last 2 preserved
	thinnedCount := 0
	preservedCount := 0
	for _, evt := range result {
		if evt.Type == domain.EventTypeAgentToolResult {
			if strings.Contains(string(evt.Payload), "[content cleared]") {
				thinnedCount++
			} else {
				preservedCount++
			}
		}
	}

	if thinnedCount != 2 {
		t.Errorf("expected 2 thinned results, got %d", thinnedCount)
	}
	if preservedCount != 2 {
		t.Errorf("expected 2 preserved results, got %d", preservedCount)
	}
}

func TestThinEvents_PreservesToolUseID(t *testing.T) {
	events := []*domain.Event{
		makeEvent(domain.EventTypeAgentToolResult, `{"tool_use_id":"keep-this","content":"big payload"}`),
		makeEvent(domain.EventTypeAgentToolResult, `{"tool_use_id":"recent","content":"new"}`),
	}

	result := ThinEvents(events, 1)

	// First event should be thinned but keep tool_use_id
	var p struct {
		ToolUseID string `json:"tool_use_id"`
	}
	json.Unmarshal(result[0].Payload, &p)
	if p.ToolUseID != "keep-this" {
		t.Errorf("tool_use_id = %q, want %q", p.ToolUseID, "keep-this")
	}
}

func TestThinEvents_DoesNotMutateOriginal(t *testing.T) {
	original := makeEvent(domain.EventTypeAgentToolResult, `{"tool_use_id":"t1","content":"original"}`)
	originalPayload := string(original.Payload)

	events := []*domain.Event{original, makeEvent(domain.EventTypeAgentToolResult, `{"tool_use_id":"t2","content":"recent"}`)}
	ThinEvents(events, 1)

	if string(original.Payload) != originalPayload {
		t.Error("original event payload was mutated")
	}
}

func TestThinEvents_Empty(t *testing.T) {
	result := ThinEvents(nil, 5)
	if result != nil {
		t.Errorf("expected nil for empty input, got %v", result)
	}
}
