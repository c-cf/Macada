package context

import (
	"github.com/c-cf/macada/internal/domain"
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

