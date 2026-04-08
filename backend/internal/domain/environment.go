package domain

import (
	"context"
	"encoding/json"
	"time"
)

type Environment struct {
	ID          string            `json:"id"`
	WorkspaceID string            `json:"workspace_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Config      EnvironmentConfig `json:"config"`
	Metadata    map[string]string `json:"metadata"`
	ArchivedAt  *time.Time        `json:"archived_at"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Type        string            `json:"type"`
}

type EnvironmentConfig struct {
	Type       string          `json:"type"`
	Networking json.RawMessage `json:"networking"`
	Packages   Packages        `json:"packages"`
}

type Packages struct {
	Apt   []string `json:"apt"`
	Cargo []string `json:"cargo"`
	Gem   []string `json:"gem"`
	Go    []string `json:"go"`
	Npm   []string `json:"npm"`
	Pip   []string `json:"pip"`
	Type  *string  `json:"type,omitempty"`
}

type UnrestrictedNetwork struct {
	Type string `json:"type"` // "unrestricted"
}

type LimitedNetwork struct {
	Type                 string   `json:"type"` // "limited"
	AllowMCPServers      *bool    `json:"allow_mcp_servers,omitempty"`
	AllowPackageManagers *bool    `json:"allow_package_managers,omitempty"`
	AllowedHosts         []string `json:"allowed_hosts,omitempty"`
}

type EnvironmentDeleteResponse struct {
	ID   string `json:"id"`
	Type string `json:"type"` // "environment_deleted"
}

type EnvironmentRepository interface {
	Create(ctx context.Context, env *Environment) error
	GetByID(ctx context.Context, id string) (*Environment, error)
	List(ctx context.Context, params ListParams) ([]*Environment, *string, error)
	Update(ctx context.Context, env *Environment) error
	Archive(ctx context.Context, id string) (*Environment, error)
	Delete(ctx context.Context, id string) error
}
