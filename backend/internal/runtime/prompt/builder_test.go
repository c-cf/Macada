package prompt

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/c-cf/macada/internal/domain"
	svcctx "github.com/c-cf/macada/internal/runtime/context"
)

// mockSkillRepo for testing skill resolution
type mockSkillRepo struct {
	skills map[string]*domain.Skill
}

func (m *mockSkillRepo) Create(_ context.Context, s *domain.Skill) error   { return nil }
func (m *mockSkillRepo) GetByID(_ context.Context, _ string) (*domain.Skill, error) {
	return nil, fmt.Errorf("not found")
}
func (m *mockSkillRepo) GetByName(_ context.Context, _ string, name string) (*domain.Skill, error) {
	s, ok := m.skills[name]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return s, nil
}
func (m *mockSkillRepo) List(_ context.Context, _ domain.SkillListParams) ([]*domain.Skill, *string, error) {
	return nil, nil, nil
}
func (m *mockSkillRepo) Update(_ context.Context, _ *domain.Skill) error { return nil }
func (m *mockSkillRepo) Delete(_ context.Context, _ string) error         { return nil }

func TestBuilder_AllSections(t *testing.T) {
	repo := &mockSkillRepo{skills: map[string]*domain.Skill{
		"code-review": {Name: "code-review", Content: "Review code carefully."},
	}}
	resolver := NewSkillResolver(repo)
	builder := NewBuilder(resolver)

	input := PromptInput{
		Agent: AgentSnapshot{
			System: "You are a helpful assistant.",
			Skills: []string{"code-review"},
			Tools:  json.RawMessage(`[{"name":"bash","description":"Run shell commands"}]`),
		},
		SessionMemory: &svcctx.SessionMemory{
			Summary:   "User asked about Go patterns.",
			TurnCount: 3,
		},
	}

	blocks, err := builder.Build(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(blocks) != 4 {
		t.Fatalf("expected 4 blocks, got %d", len(blocks))
	}

	// Block 0: base instructions (no cache)
	if blocks[0].CacheControl != nil {
		t.Error("block 0 should not have cache_control")
	}

	// Block 1: skills (no cache)
	if blocks[1].CacheControl != nil {
		t.Error("block 1 should not have cache_control")
	}

	// Block 2: tools (last static — has cache_control)
	if blocks[2].CacheControl == nil {
		t.Error("block 2 (last static) should have cache_control")
	}

	// Block 3: session memory (dynamic — no cache)
	if blocks[3].CacheControl != nil {
		t.Error("block 3 (dynamic) should not have cache_control")
	}
}

func TestBuilder_NoSkills(t *testing.T) {
	builder := NewBuilder(nil)

	input := PromptInput{
		Agent: AgentSnapshot{
			System: "You are helpful.",
			Tools:  json.RawMessage(`[{"name":"read","description":"Read files"}]`),
		},
	}

	blocks, err := builder.Build(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 2 blocks: system + tools
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	// Last static (tools) should have cache_control
	if blocks[1].CacheControl == nil {
		t.Error("last static block should have cache_control")
	}
}

func TestBuilder_OnlySystem(t *testing.T) {
	builder := NewBuilder(nil)

	input := PromptInput{
		Agent: AgentSnapshot{
			System: "You are a math tutor.",
		},
	}

	blocks, err := builder.Build(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	// Sole block should have cache_control
	if blocks[0].CacheControl == nil {
		t.Error("sole static block should have cache_control")
	}
}

func TestBuilder_EmptySystem(t *testing.T) {
	builder := NewBuilder(nil)

	input := PromptInput{
		Agent: AgentSnapshot{},
	}

	blocks, err := builder.Build(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(blocks) != 0 {
		t.Errorf("expected 0 blocks for empty agent, got %d", len(blocks))
	}
}

func TestBuilder_SessionMemoryOnly(t *testing.T) {
	builder := NewBuilder(nil)

	input := PromptInput{
		Agent: AgentSnapshot{
			System: "Be concise.",
		},
		SessionMemory: &svcctx.SessionMemory{
			Summary: "Previous discussion about databases.",
		},
	}

	blocks, err := builder.Build(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}

	// First block: system (cached)
	if blocks[0].CacheControl == nil {
		t.Error("system block should have cache_control")
	}

	// Second block: session memory (not cached)
	if blocks[1].CacheControl != nil {
		t.Error("session memory block should not have cache_control")
	}
}
