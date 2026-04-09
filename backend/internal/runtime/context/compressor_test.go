package context

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cchu-code/managed-agents/internal/domain"
)

func makeEvents(count int) []*domain.Event {
	events := make([]*domain.Event, 0, count)
	for i := 0; i < count; i++ {
		switch i % 3 {
		case 0:
			events = append(events, makeEvent(domain.EventTypeUserMessage,
				`{"content":[{"type":"text","text":"`+strings.Repeat("x", 200)+`"}]}`))
		case 1:
			events = append(events, makeEvent(domain.EventTypeAgentMessage,
				`{"content":[{"type":"text","text":"`+strings.Repeat("y", 200)+`"}]}`))
		default:
			events = append(events, makeEvent(domain.EventTypeAgentToolResult,
				`{"tool_use_id":"t`+json.Number(string(rune('0'+i))).String()+`","content":"`+strings.Repeat("z", 500)+`"}`))
		}
	}
	return events
}

func TestCompressor_SmallEventSet(t *testing.T) {
	c := NewCompressor(CompressionConfig{
		MaxRecentToolResults:  5,
		MemoryTokenThreshold:  80_000,
		CompactTokenThreshold: 150_000,
		CompactKeepRecent:     10,
	})

	events := makeEvents(5)
	result, err := c.Compress(events, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Tier != 1 {
		t.Errorf("tier = %d, want 1 (thin only)", result.Tier)
	}
	if result.Memory != nil {
		t.Error("expected no memory for small event set")
	}
}

func TestCompressor_MemoryThreshold(t *testing.T) {
	c := NewCompressor(CompressionConfig{
		MaxRecentToolResults:  2,
		MemoryTokenThreshold:  100, // very low threshold to trigger
		CompactTokenThreshold: 999_999,
		CompactKeepRecent:     5,
	})

	events := makeEvents(30)
	result, err := c.Compress(events, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Tier < 2 {
		t.Errorf("tier = %d, want >= 2 (memory extraction)", result.Tier)
	}
	if result.Memory == nil {
		t.Error("expected memory to be extracted")
	}
}

func TestCompressor_CompactThreshold(t *testing.T) {
	c := NewCompressor(CompressionConfig{
		MaxRecentToolResults:  2,
		MemoryTokenThreshold:  100,
		CompactTokenThreshold: 100, // very low threshold
		CompactKeepRecent:     3,
	})

	events := makeEvents(30)
	result, err := c.Compress(events, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Tier != 3 {
		t.Errorf("tier = %d, want 3 (full compaction)", result.Tier)
	}
	if result.Memory == nil {
		t.Error("expected memory after compaction")
	}
	if len(result.Events) > 3 {
		t.Errorf("expected at most 3 events after compaction, got %d", len(result.Events))
	}
}

func TestCompressor_ExistingMemoryPreserved(t *testing.T) {
	c := NewCompressor(CompressionConfig{
		MaxRecentToolResults:  5,
		MemoryTokenThreshold:  999_999,
		CompactTokenThreshold: 999_999,
		CompactKeepRecent:     10,
	})

	existing := &SessionMemory{Summary: "previous context", TurnCount: 5}
	events := makeEvents(3)
	result, err := c.Compress(events, existing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Memory == nil {
		t.Fatal("expected existing memory to be preserved")
	}
	if result.Memory.Summary != "previous context" {
		t.Errorf("memory summary = %q, want %q", result.Memory.Summary, "previous context")
	}
}

func TestCompressor_EmptyEvents(t *testing.T) {
	c := NewCompressor(DefaultCompressionConfig())
	result, err := c.Compress(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Tier != 0 {
		t.Errorf("tier = %d, want 0 for empty events", result.Tier)
	}
}
