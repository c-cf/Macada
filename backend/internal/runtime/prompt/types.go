package prompt

import (
	"encoding/json"

	"github.com/cchu-code/managed-agents/internal/domain"
	svcctx "github.com/cchu-code/managed-agents/internal/runtime/context"
)

// SystemBlock is one block in the Anthropic system prompt array.
type SystemBlock struct {
	Type         string        `json:"type"`
	Text         string        `json:"text"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// CacheControl instructs the Anthropic API to cache up to this block.
type CacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

// EphemeralCache is a convenience value for cache_control.
var EphemeralCache = &CacheControl{Type: "ephemeral"}

// AgentSnapshot is the parsed agent data from session.agent_snapshot JSONB.
type AgentSnapshot struct {
	ID          string            `json:"id"`
	Version     int               `json:"version"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Model       domain.ModelConfig `json:"model"`
	System      string            `json:"system"`
	Tools       json.RawMessage   `json:"tools"`
	MCPServers  json.RawMessage   `json:"mcp_servers"`
	Skills      []string          `json:"skills"`
}

// ResolvedSkill pairs a skill name with its resolved content.
type ResolvedSkill struct {
	Name    string
	Content string
}

// PromptInput gathers all inputs needed to build the system prompt.
type PromptInput struct {
	Agent          AgentSnapshot
	WorkspaceID    string
	ResolvedSkills []ResolvedSkill
	SessionMemory  *svcctx.SessionMemory
}
