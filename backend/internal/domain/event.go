package domain

import (
	"context"
	"encoding/json"
	"time"
)

// Event type constants matching docs/event-types.md
const (
	EventTypeUserMessage         = "user.message"
	EventTypeUserInterrupt       = "user.interrupt"
	EventTypeUserToolConfirm     = "user.tool_confirmation"
	EventTypeUserCustomToolResult = "user.custom_tool_result"

	EventTypeAgentMessage        = "agent.message"
	EventTypeAgentToolUse        = "agent.tool_use"
	EventTypeAgentToolResult     = "agent.tool_result"
	EventTypeAgentCustomToolUse  = "agent.custom_tool_use"

	EventTypeModelRequestStart   = "span.model_request_start"
	EventTypeModelRequestEnd     = "span.model_request_end"

	EventTypeSessionRunning      = "session.status_running"
	EventTypeSessionIdle         = "session.status_idle"

	EventTypeRuntimeStarted          = "runtime.started"
	EventTypeRuntimeStopped          = "runtime.stopped"
	EventTypeRuntimeHeartbeat        = "runtime.heartbeat"
	EventTypeRuntimeError            = "runtime.error"
	EventTypeRuntimePackagesInstalled = "runtime.packages_installed"
)

type Event struct {
	ID          string          `json:"id"`
	SessionID   string          `json:"-"`
	Type        string          `json:"type"`
	ProcessedAt time.Time       `json:"processed_at"`
	Payload     json.RawMessage `json:"-"` // merged into top-level JSON on serialization
}

// MarshalJSON produces a flat JSON object that merges Payload fields into the top-level event.
func (e Event) MarshalJSON() ([]byte, error) {
	base := map[string]interface{}{
		"id":           e.ID,
		"type":         e.Type,
		"processed_at": e.ProcessedAt.Format(time.RFC3339Nano),
	}

	if len(e.Payload) > 0 && string(e.Payload) != "{}" {
		var extra map[string]interface{}
		if err := json.Unmarshal(e.Payload, &extra); err == nil {
			for k, v := range extra {
				base[k] = v
			}
		}
	}

	return json.Marshal(base)
}

// ModelUsage represents token usage from a model request
type ModelUsage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
}

// EventBus provides pub/sub for real-time event streaming (SSE fan-out)
type EventBus interface {
	Publish(ctx context.Context, sessionID string, event *Event) error
	Subscribe(ctx context.Context, sessionID string) (<-chan *Event, func(), error)
}

type EventRepository interface {
	Create(ctx context.Context, event *Event) error
	ListBySession(ctx context.Context, sessionID string, params EventListParams) ([]*Event, *string, error)
}

type EventListParams struct {
	Limit *int
	Page  *string
	Order *string // "asc" | "desc", default "asc"
}

// SendEventParams describes a single event sent by a client.
type SendEventParams struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content,omitempty"`
	// Additional fields for different event types
	ToolUseID       *string `json:"tool_use_id,omitempty"`
	Result          *string `json:"result,omitempty"`
	DenyMessage     *string `json:"deny_message,omitempty"`
	CustomToolUseID *string `json:"custom_tool_use_id,omitempty"`
	IsError         *bool   `json:"is_error,omitempty"`
}

// SessionRunner is the interface for the agent loop that processes user messages.
type SessionRunner interface {
	Run(ctx context.Context, sessionID string, events []SendEventParams) error
}
