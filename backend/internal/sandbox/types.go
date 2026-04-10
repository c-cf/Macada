package sandbox

import (
	"encoding/json"
	"time"

	"github.com/c-cf/macada/internal/domain"
	svcctx "github.com/c-cf/macada/internal/runtime/context"
)

// SandboxStatus represents the lifecycle state of a sandbox container.
type SandboxStatus string

const (
	SandboxStatusPending  SandboxStatus = "pending"
	SandboxStatusRunning  SandboxStatus = "running"
	SandboxStatusStopped  SandboxStatus = "stopped"
	SandboxStatusError    SandboxStatus = "error"
)

// SandboxInfo tracks a running sandbox container.
type SandboxInfo struct {
	ID              string        `json:"id"`
	SessionID       string        `json:"session_id"`
	ContainerID     string        `json:"container_id"`
	ContainerIP     string        `json:"container_ip"`
	Status          SandboxStatus `json:"status"`
	CreatedAt       time.Time     `json:"created_at"`
	LastHeartbeatAt *time.Time    `json:"last_heartbeat_at,omitempty"`
}

// FileMount describes a file to be mounted into the container.
type FileMount struct {
	MountPath string // absolute path in container (e.g. /mnt/session/uploads/file_xxx)
	Content   []byte // file content
}

// DeployManifest holds all data needed to write config files into a sandbox.
type DeployManifest struct {
	Agent          AgentConfigFile
	SystemPrompt   string
	Model          domain.ModelConfig
	Tools          json.RawMessage
	MCPServers     json.RawMessage
	Skills         []SkillManifest
	Packages       domain.Packages
	SessionMemory  *svcctx.SessionMemory
	FileMounts     []FileMount
}

// SkillManifest holds a resolved skill ready for filesystem deployment.
type SkillManifest struct {
	Name        string
	Description string
	License     string
	Content     string            // SKILL.md body (markdown after frontmatter)
	Files       map[string]string // relative path -> content (scripts/, references/, etc.)
}

// --- Config file schemas (written to sandbox filesystem) ---

// AgentConfigFile is written to /workspace/.claude/agent.json.
// Note: API key is NOT included — LLM calls go through the control plane proxy.
type AgentConfigFile struct {
	ID               string `json:"id"`
	Version          int    `json:"version"`
	Name             string `json:"name"`
	Type             string `json:"type,omitempty"`
	SessionID        string `json:"session_id"`
	ControlPlaneURL  string `json:"control_plane_url"`
	ControlPlaneToken string `json:"control_plane_token"`
}

// SettingsFile is written to /workspace/.claude/settings.json.
type SettingsFile struct {
	Model      domain.ModelConfig     `json:"model"`
	MCPServers json.RawMessage        `json:"mcp_servers,omitempty"`
}

// SessionStateFile is written to /workspace/.claude/session.json.
type SessionStateFile struct {
	Memory *svcctx.SessionMemory `json:"memory,omitempty"`
	Status string                `json:"status"`
}

// PackagesFile is written to /workspace/.claude/packages.json.
type PackagesFile struct {
	Apt   []string `json:"apt,omitempty"`
	Cargo []string `json:"cargo,omitempty"`
	Gem   []string `json:"gem,omitempty"`
	Go    []string `json:"go,omitempty"`
	Npm   []string `json:"npm,omitempty"`
	Pip   []string `json:"pip,omitempty"`
}
