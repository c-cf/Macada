package config

import (
	"encoding/json"

	"github.com/c-cf/macada/internal/domain"
	svcctx "github.com/c-cf/macada/internal/runtime/context"
)

// RuntimeConfig is the aggregate config loaded from sandbox filesystem.
type RuntimeConfig struct {
	Agent        AgentConfig
	SystemPrompt string
	Settings     SettingsConfig
	Tools        json.RawMessage
	Skills       []LoadedSkill
	Packages     PackagesConfig
	Session      SessionState
}

// AgentConfig corresponds to /workspace/.claude/agent.json.
// Note: no API key — LLM calls go through the control plane proxy.
type AgentConfig struct {
	ID                string `json:"id"`
	Version           int    `json:"version"`
	Name              string `json:"name"`
	Type              string `json:"type,omitempty"`
	SessionID         string `json:"session_id"`
	ControlPlaneURL   string `json:"control_plane_url"`
	ControlPlaneToken string `json:"control_plane_token"`
}

// SettingsConfig corresponds to /workspace/.claude/settings.json.
type SettingsConfig struct {
	Model      domain.ModelConfig `json:"model"`
	MCPServers json.RawMessage    `json:"mcp_servers,omitempty"`
}

// LoadedSkill represents a skill discovered on the filesystem.
type LoadedSkill struct {
	Name        string
	Description string
	Content     string // SKILL.md body (markdown after frontmatter)
	Dir         string // absolute path to skill directory
}

// PackagesConfig corresponds to /workspace/.claude/packages.json.
type PackagesConfig struct {
	Apt   []string `json:"apt,omitempty"`
	Cargo []string `json:"cargo,omitempty"`
	Gem   []string `json:"gem,omitempty"`
	Go    []string `json:"go,omitempty"`
	Npm   []string `json:"npm,omitempty"`
	Pip   []string `json:"pip,omitempty"`
}

// SessionState corresponds to /workspace/.claude/session.json.
type SessionState struct {
	Memory *svcctx.SessionMemory `json:"memory,omitempty"`
	Status string                `json:"status"`
}
