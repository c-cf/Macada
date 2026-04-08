package context

import (
	"github.com/cchu-code/managed-agents/internal/domain"
)

// Compressor orchestrates the three tiers of context compression.
type Compressor struct {
	config CompressionConfig
}

// NewCompressor creates a new Compressor with the given config.
func NewCompressor(config CompressionConfig) *Compressor {
	return &Compressor{config: config}
}

// Compress applies the appropriate compression tier(s) to the event history.
//
// Tiers are applied progressively:
//  1. Thinning — always applied; clears old tool result payloads
//  2. Memory extraction — if tokens > MemoryTokenThreshold and no existing memory
//  3. Full compaction — if tokens > CompactTokenThreshold
//
// Returns the processed events, any memory updates, and which tier was applied.
func (c *Compressor) Compress(events []*domain.Event, existingMemory *SessionMemory) (*CompressResult, error) {
	if len(events) == 0 {
		return &CompressResult{Events: events, Tier: 0}, nil
	}

	tokensBefore := EstimateEventsTokens(events)

	// Tier 1: Always thin old tool results
	thinned := ThinEvents(events, c.config.MaxRecentToolResults)
	tier := 1

	tokensAfterThin := EstimateEventsTokens(thinned)

	// Tier 2: Extract memory if threshold exceeded and no existing memory
	var memory *SessionMemory
	if tokensAfterThin > c.config.MemoryTokenThreshold && existingMemory == nil {
		var err error
		memory, err = ExtractMemory(thinned)
		if err != nil {
			return nil, err
		}
		tier = 2
	}

	// Tier 3: Full compaction if threshold exceeded
	resultEvents := thinned
	if tokensAfterThin > c.config.CompactTokenThreshold {
		compactMemory, kept, err := Compact(thinned, c.config.CompactKeepRecent)
		if err != nil {
			return nil, err
		}
		memory = compactMemory
		resultEvents = kept
		tier = 3
	}

	// If no new memory was created, carry forward existing
	if memory == nil && existingMemory != nil {
		memory = existingMemory
	}

	tokensAfter := EstimateEventsTokens(resultEvents)

	return &CompressResult{
		Events:       resultEvents,
		Memory:       memory,
		Tier:         tier,
		TokensBefore: tokensBefore,
		TokensAfter:  tokensAfter,
	}, nil
}
