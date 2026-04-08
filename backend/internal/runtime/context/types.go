package context

import (
	"encoding/json"
	"time"

	"github.com/cchu-code/managed-agents/internal/domain"
)

// CompressionConfig holds thresholds for each compression tier.
type CompressionConfig struct {
	// Tier 1: keep N most recent tool results in full
	MaxRecentToolResults int
	// Tier 2: extract memory when estimated tokens exceed this
	MemoryTokenThreshold int
	// Tier 3: full compaction when exceeding this
	CompactTokenThreshold int
	// Tier 3: keep N most recent events in full after compaction
	CompactKeepRecent int
}

// DefaultCompressionConfig returns sensible defaults.
func DefaultCompressionConfig() CompressionConfig {
	return CompressionConfig{
		MaxRecentToolResults:  5,
		MemoryTokenThreshold:  80_000,
		CompactTokenThreshold: 150_000,
		CompactKeepRecent:     10,
	}
}

// SessionMemory stores extracted context from compression.
type SessionMemory struct {
	Summary     string    `json:"summary"`
	CompactedAt time.Time `json:"compacted_at"`
	TurnCount   int       `json:"turn_count"`
}

// CompressResult is the output of the compressor.
type CompressResult struct {
	Events  []*domain.Event // processed events (thinned or truncated)
	Memory  *SessionMemory  // non-nil if memory was created/updated
	Tier    int             // highest tier applied (0=none, 1=thin, 2=memory, 3=compact)
	TokensBefore int
	TokensAfter  int
}

// Message represents an Anthropic messages API message.
type Message struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

// ContentBlock is a single block within a Message.
type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}
