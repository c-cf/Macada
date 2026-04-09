package sandbox

import (
	"encoding/json"

	rtctx "github.com/c-cf/macada/internal/runtime/context"
)

// ForwardPayload is the data sent from the control plane to the runtime sandbox.
// It contains pre-compressed conversation history so the runtime is stateless.
//
// The runtime uses:
//   - Memory → injected into system prompt as session context
//   - Messages → conversation history (already in Anthropic message format)
//   - NewEvents → the latest user events that triggered this turn
type ForwardPayload struct {
	// Memory is a compressed summary of older conversation turns.
	// Produced by the backend compressor and stored in session.memory.
	// nil if this is the first turn.
	Memory *rtctx.SessionMemory `json:"memory,omitempty"`

	// Messages are recent conversation turns in Anthropic Messages API format.
	// These are the events that were NOT compressed into Memory.
	Messages []rtctx.Message `json:"messages"`

	// NewEvents are the raw user events that triggered this turn.
	// The runtime converts these to user messages and appends to Messages.
	NewEvents []NewEvent `json:"new_events"`

	// ModelID is the model to use for this request (from agent config).
	ModelID string `json:"model_id"`

	// ContextWindowSize is the model's context window in tokens.
	// Used by runtime to decide when mid-turn compression is needed.
	ContextWindowSize int `json:"context_window_size"`
}

// NewEvent is a simplified user event for forwarding.
type NewEvent struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content,omitempty"`
}
