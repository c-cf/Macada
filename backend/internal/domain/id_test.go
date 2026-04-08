package domain_test

import (
	"strings"
	"testing"

	"github.com/cchu-code/managed-agents/internal/domain"
)

func TestNewAgentID(t *testing.T) {
	t.Run("starts with agent_ prefix", func(t *testing.T) {
		id := domain.NewAgentID()
		if !strings.HasPrefix(id, "agent_") {
			t.Errorf("expected prefix %q, got %q", "agent_", id)
		}
	})
}

func TestNewSessionID(t *testing.T) {
	t.Run("starts with sesn_ prefix", func(t *testing.T) {
		id := domain.NewSessionID()
		if !strings.HasPrefix(id, "sesn_") {
			t.Errorf("expected prefix %q, got %q", "sesn_", id)
		}
	})
}

func TestNewEnvironmentID(t *testing.T) {
	t.Run("starts with env_ prefix", func(t *testing.T) {
		id := domain.NewEnvironmentID()
		if !strings.HasPrefix(id, "env_") {
			t.Errorf("expected prefix %q, got %q", "env_", id)
		}
	})
}

func TestNewEventID(t *testing.T) {
	t.Run("starts with sevt_ prefix", func(t *testing.T) {
		id := domain.NewEventID()
		if !strings.HasPrefix(id, "sevt_") {
			t.Errorf("expected prefix %q, got %q", "sevt_", id)
		}
	})
}

func TestIDUniqueness(t *testing.T) {
	tests := []struct {
		name   string
		genFn  func() string
		prefix string
	}{
		{"AgentID", domain.NewAgentID, "agent_"},
		{"SessionID", domain.NewSessionID, "sesn_"},
		{"EnvironmentID", domain.NewEnvironmentID, "env_"},
		{"EventID", domain.NewEventID, "sevt_"},
	}

	for _, tc := range tests {
		t.Run(tc.name+" generates unique IDs", func(t *testing.T) {
			seen := make(map[string]struct{}, 100)
			for i := 0; i < 100; i++ {
				id := tc.genFn()
				if _, exists := seen[id]; exists {
					t.Fatalf("duplicate ID generated: %s", id)
				}
				seen[id] = struct{}{}
			}
		})
	}
}

func TestNewID(t *testing.T) {
	t.Run("custom prefix is preserved", func(t *testing.T) {
		id := domain.NewID("custom_")
		if !strings.HasPrefix(id, "custom_") {
			t.Errorf("expected prefix %q, got %q", "custom_", id)
		}
	})

	t.Run("ID has content beyond prefix", func(t *testing.T) {
		id := domain.NewID("pfx_")
		suffix := strings.TrimPrefix(id, "pfx_")
		if len(suffix) == 0 {
			t.Error("expected non-empty ULID suffix after prefix")
		}
		// ULID is 26 characters
		if len(suffix) != 26 {
			t.Errorf("expected ULID suffix length 26, got %d (%q)", len(suffix), suffix)
		}
	})
}
