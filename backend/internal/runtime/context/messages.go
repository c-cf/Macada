package context

import (
	"encoding/json"

	"github.com/c-cf/macada/internal/domain"
)

// EventsToMessages converts stored events into Anthropic Messages API format.
// Groups consecutive same-role events into single messages.
// Skips span.* and session.* events (metadata, not conversation content).
func EventsToMessages(events []*domain.Event) []Message {
	var messages []Message

	for _, evt := range events {
		role, block := eventToBlock(evt)
		if role == "" {
			continue // skip non-conversation events
		}

		// Group into existing message if same role
		if len(messages) > 0 && messages[len(messages)-1].Role == role {
			messages[len(messages)-1].Content = append(messages[len(messages)-1].Content, block)
		} else {
			messages = append(messages, Message{
				Role:    role,
				Content: []ContentBlock{block},
			})
		}
	}

	return messages
}

// eventToBlock maps an event to (role, ContentBlock).
// Returns empty role for events that don't map to messages.
func eventToBlock(evt *domain.Event) (string, ContentBlock) {
	switch evt.Type {
	case domain.EventTypeUserMessage:
		text := extractTextFromPayload(evt.Payload)
		return "user", ContentBlock{Type: "text", Text: text}

	case domain.EventTypeAgentMessage:
		text := extractTextFromPayload(evt.Payload)
		return "assistant", ContentBlock{Type: "text", Text: text}

	case domain.EventTypeAgentToolUse:
		var p struct {
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		}
		_ = json.Unmarshal(evt.Payload, &p)
		return "assistant", ContentBlock{
			Type:  "tool_use",
			ID:    p.ID,
			Name:  p.Name,
			Input: p.Input,
		}

	case domain.EventTypeAgentToolResult:
		var p struct {
			ToolUseID string          `json:"tool_use_id"`
			Content   json.RawMessage `json:"content"`
			IsError   bool            `json:"is_error"`
		}
		_ = json.Unmarshal(evt.Payload, &p)

		// Content can be string or structured
		text := ""
		var s string
		if json.Unmarshal(p.Content, &s) == nil {
			text = s
		} else {
			text = string(p.Content)
		}

		return "user", ContentBlock{
			Type:      "tool_result",
			ToolUseID: p.ToolUseID,
			Text:      text,
			IsError:   p.IsError,
		}

	default:
		// span.*, session.*, user.interrupt, etc. — not conversation content
		return "", ContentBlock{}
	}
}
