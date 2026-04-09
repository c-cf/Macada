package context

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cchu-code/managed-agents/internal/domain"
)

const maxExcerptLen = 200

// ExtractMemory creates a SessionMemory summary from events.
// This is a deterministic extraction (no LLM call) that captures:
// - User questions/requests
// - Agent responses
// - Tool usage patterns
// - Turn count
func ExtractMemory(events []*domain.Event) (*SessionMemory, error) {
	if len(events) == 0 {
		return &SessionMemory{Summary: "", CompactedAt: time.Now().UTC()}, nil
	}

	var userRequests []string
	var agentResponses []string
	toolUsage := map[string]int{}
	turnCount := 0

	for _, evt := range events {
		switch evt.Type {
		case domain.EventTypeUserMessage:
			turnCount++
			text := extractTextFromPayload(evt.Payload)
			userRequests = append(userRequests, truncate(text, maxExcerptLen))

		case domain.EventTypeAgentMessage:
			text := extractTextFromPayload(evt.Payload)
			agentResponses = append(agentResponses, truncate(text, maxExcerptLen))

		case domain.EventTypeAgentToolUse:
			name := extractToolName(evt.Payload)
			if name != "" {
				toolUsage[name]++
			}
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "## Conversation Summary (%d turns)\n\n", turnCount)

	if len(userRequests) > 0 {
		sb.WriteString("### User Requests\n")
		for i, req := range userRequests {
			fmt.Fprintf(&sb, "- Turn %d: %q\n", i+1, req)
		}
		sb.WriteString("\n")
	}

	if len(toolUsage) > 0 {
		sb.WriteString("### Tools Used\n")
		for name, count := range toolUsage {
			fmt.Fprintf(&sb, "- %s (%d calls)\n", name, count)
		}
		sb.WriteString("\n")
	}

	if len(agentResponses) > 0 {
		sb.WriteString("### Agent Responses\n")
		for i, resp := range agentResponses {
			fmt.Fprintf(&sb, "- Turn %d: %q\n", i+1, resp)
		}
	}

	return &SessionMemory{
		Summary:     sb.String(),
		CompactedAt: time.Now().UTC(),
		TurnCount:   turnCount,
	}, nil
}

// extractTextFromPayload extracts the first text content from an event payload.
func extractTextFromPayload(payload json.RawMessage) string {
	// Try content array format: {"content": [{"type":"text","text":"..."}]}
	var withContent struct {
		Content json.RawMessage `json:"content"`
	}
	if json.Unmarshal(payload, &withContent) == nil && len(withContent.Content) > 0 {
		// Try as array of blocks
		var blocks []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if json.Unmarshal(withContent.Content, &blocks) == nil {
			for _, b := range blocks {
				if b.Type == "text" && b.Text != "" {
					return b.Text
				}
			}
		}
		// Try as plain string
		var text string
		if json.Unmarshal(withContent.Content, &text) == nil {
			return text
		}
	}

	// Try direct text field
	var direct struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(payload, &direct) == nil && direct.Text != "" {
		return direct.Text
	}

	return ""
}

// extractToolName extracts the tool name from a tool_use event payload.
func extractToolName(payload json.RawMessage) string {
	var p struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(payload, &p) == nil {
		return p.Name
	}
	return ""
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
