package domain

import (
	"context"
	"encoding/json"
	"time"
)

type SessionStatus string

const (
	SessionStatusRunning      SessionStatus = "running"
	SessionStatusIdle         SessionStatus = "idle"
	SessionStatusTerminated   SessionStatus = "terminated"
	SessionStatusRescheduling SessionStatus = "rescheduling"
)

type Session struct {
	ID            string            `json:"id"`
	WorkspaceID   string            `json:"workspace_id"`
	Agent         json.RawMessage   `json:"agent"`
	EnvironmentID string            `json:"environment_id"`
	Title         string            `json:"title"`
	Status        SessionStatus     `json:"status"`
	Stats         SessionStats      `json:"stats"`
	Usage         SessionUsage      `json:"usage"`
	Resources     json.RawMessage   `json:"resources"`
	Metadata      map[string]string `json:"metadata"`
	VaultIDs      []string          `json:"vault_ids"`
	Memory        json.RawMessage   `json:"memory"`
	ArchivedAt    *time.Time        `json:"archived_at"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	Type          string            `json:"type"`
}

type SessionStats struct {
	ActiveSeconds   *float64 `json:"active_seconds"`
	DurationSeconds *float64 `json:"duration_seconds"`
}

type SessionUsage struct {
	InputTokens          *int64              `json:"input_tokens"`
	OutputTokens         *int64              `json:"output_tokens"`
	CacheReadInputTokens *int64              `json:"cache_read_input_tokens"`
	CacheCreation        *CacheCreationUsage `json:"cache_creation"`
}

type CacheCreationUsage struct {
	Ephemeral1hInputTokens *int64 `json:"ephemeral_1h_input_tokens"`
	Ephemeral5mInputTokens *int64 `json:"ephemeral_5m_input_tokens"`
}

type SessionRepository interface {
	Create(ctx context.Context, session *Session) error
	GetByID(ctx context.Context, id string) (*Session, error)
	List(ctx context.Context, params SessionListParams) ([]*Session, *string, error)
	Update(ctx context.Context, session *Session) error
	UpdateStatus(ctx context.Context, id string, status SessionStatus) error
	UpdateUsage(ctx context.Context, id string, usage SessionUsage) error
	UpdateMemory(ctx context.Context, id string, memory json.RawMessage) error
	Archive(ctx context.Context, id string) (*Session, error)
}

type SessionListParams struct {
	ListParams
	AgentID      *string
	AgentVersion *int
	CreatedAtGT  *time.Time
	CreatedAtGTE *time.Time
	CreatedAtLT  *time.Time
	CreatedAtLTE *time.Time
	Order        *string // "asc" | "desc"
}
