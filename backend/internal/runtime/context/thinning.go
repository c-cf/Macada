package context

import (
	"encoding/json"

	"github.com/cchu-code/managed-agents/internal/domain"
)

// ThinEvents returns a new slice of events with old tool_result payloads replaced.
// The most recent keepRecent tool_result events retain full content.
// Older tool_result events have their payload replaced with a cleared marker.
// Non-tool_result events are copied unchanged. Original events are never mutated.
func ThinEvents(events []*domain.Event, keepRecent int) []*domain.Event {
	if len(events) == 0 {
		return nil
	}

	// Find indices of tool_result events (in order)
	var toolResultIndices []int
	for i, evt := range events {
		if evt.Type == domain.EventTypeAgentToolResult {
			toolResultIndices = append(toolResultIndices, i)
		}
	}

	// Determine which tool_result indices to thin
	thinSet := map[int]bool{}
	if len(toolResultIndices) > keepRecent {
		for _, idx := range toolResultIndices[:len(toolResultIndices)-keepRecent] {
			thinSet[idx] = true
		}
	}

	result := make([]*domain.Event, len(events))
	for i, evt := range events {
		if thinSet[i] {
			result[i] = thinToolResult(evt)
		} else {
			result[i] = copyEvent(evt)
		}
	}
	return result
}

// thinToolResult creates a new event with the tool_result payload replaced.
func thinToolResult(evt *domain.Event) *domain.Event {
	// Preserve tool_use_id from original payload
	toolUseID := extractToolUseID(evt.Payload)

	thinned := map[string]interface{}{
		"tool_use_id": toolUseID,
		"content":     "[content cleared]",
		"is_error":    false,
	}
	payload, _ := json.Marshal(thinned)

	return &domain.Event{
		ID:          evt.ID,
		SessionID:   evt.SessionID,
		Type:        evt.Type,
		ProcessedAt: evt.ProcessedAt,
		Payload:     payload,
	}
}

// extractToolUseID extracts the tool_use_id from a tool_result payload.
func extractToolUseID(payload json.RawMessage) string {
	var p struct {
		ToolUseID string `json:"tool_use_id"`
	}
	if json.Unmarshal(payload, &p) == nil {
		return p.ToolUseID
	}
	return ""
}

// copyEvent creates a shallow copy of an event (new struct, shared payload bytes).
func copyEvent(evt *domain.Event) *domain.Event {
	return &domain.Event{
		ID:          evt.ID,
		SessionID:   evt.SessionID,
		Type:        evt.Type,
		ProcessedAt: evt.ProcessedAt,
		Payload:     evt.Payload,
	}
}
