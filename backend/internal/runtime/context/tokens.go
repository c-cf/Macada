package context

import (
	"encoding/json"

	"github.com/cchu-code/managed-agents/internal/domain"
)

const (
	charsPerToken   = 4   // rough heuristic for English text
	overheadPerMsg  = 4   // ~4 tokens per message for role/separators
)

// EstimateTokens returns an approximate token count for a text string.
func EstimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	return (len(text) + charsPerToken - 1) / charsPerToken
}

// EstimateEventTokens returns the estimated token count for a single event.
func EstimateEventTokens(evt *domain.Event) int {
	if evt == nil {
		return 0
	}
	tokens := EstimateTokens(evt.Type) + overheadPerMsg
	if len(evt.Payload) > 0 {
		tokens += EstimateTokens(string(evt.Payload))
	}
	return tokens
}

// EstimateEventsTokens returns the total estimated tokens for a slice of events.
func EstimateEventsTokens(events []*domain.Event) int {
	total := 0
	for _, evt := range events {
		total += EstimateEventTokens(evt)
	}
	return total
}

// EstimateMessagesTokens returns the total estimated tokens for Anthropic Messages.
func EstimateMessagesTokens(messages []Message) int {
	total := 0
	for _, msg := range messages {
		total += overheadPerMsg
		for _, block := range msg.Content {
			total += EstimateTokens(block.Text)
			if len(block.Input) > 0 {
				total += EstimateTokens(string(block.Input))
			}
		}
	}
	return total
}

// payloadSize returns the byte length of an event's payload.
func payloadSize(evt *domain.Event) int {
	if evt == nil || len(evt.Payload) == 0 {
		return 0
	}
	// Quick check: if payload is just "{}", treat as empty
	var m map[string]json.RawMessage
	if json.Unmarshal(evt.Payload, &m) == nil {
		total := 0
		for _, v := range m {
			total += len(v)
		}
		return total
	}
	return len(evt.Payload)
}
