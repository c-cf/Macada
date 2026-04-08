package domain

import (
	"context"
	"encoding/json"
	"time"
)

type Agent struct {
	ID          string            `json:"id"`
	WorkspaceID string            `json:"workspace_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Model       ModelConfig       `json:"model"`
	System      string            `json:"system"`
	Tools       json.RawMessage   `json:"tools"`
	MCPServers  json.RawMessage   `json:"mcp_servers"`
	Skills      json.RawMessage   `json:"skills"`
	Metadata    map[string]string `json:"metadata"`
	Version     int               `json:"version"`
	ArchivedAt  *time.Time        `json:"archived_at"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Type        string            `json:"type"`
}

type ModelConfig struct {
	ID    string  `json:"id"`
	Speed *string `json:"speed,omitempty"`
}

type AgentRepository interface {
	Create(ctx context.Context, agent *Agent) error
	GetByID(ctx context.Context, id string) (*Agent, error)
	List(ctx context.Context, params AgentListParams) ([]*Agent, *string, error)
	Update(ctx context.Context, agent *Agent) error
	Archive(ctx context.Context, id string) (*Agent, error)
}

type AgentListParams struct {
	ListParams
	CreatedAtGTE *time.Time
	CreatedAtLTE *time.Time
}
