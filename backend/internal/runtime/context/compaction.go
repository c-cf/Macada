package context

import (
	"github.com/cchu-code/managed-agents/internal/domain"
)

// Compact reduces a conversation to a summary plus the most recent N events.
// It builds a comprehensive SessionMemory covering all events except the kept tail.
// Returns the updated memory and the kept recent events.
func Compact(events []*domain.Event, keepRecent int) (*SessionMemory, []*domain.Event, error) {
	if len(events) == 0 {
		return &SessionMemory{}, nil, nil
	}

	// Split: events to summarize vs events to keep
	splitIdx := len(events) - keepRecent
	if splitIdx < 0 {
		splitIdx = 0
	}

	toSummarize := events[:splitIdx]
	toKeep := events[splitIdx:]

	// Extract memory from the portion being compacted
	memory, err := ExtractMemory(toSummarize)
	if err != nil {
		return nil, nil, err
	}

	// Copy kept events to avoid mutating the original slice
	kept := make([]*domain.Event, len(toKeep))
	for i, evt := range toKeep {
		kept[i] = copyEvent(evt)
	}

	return memory, kept, nil
}
