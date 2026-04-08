package domain

import (
	"context"
	"encoding/json"
	"time"
)

type Skill struct {
	ID            string            `json:"id"`
	WorkspaceID   string            `json:"workspace_id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	License       string            `json:"license,omitempty"`
	Compatibility string            `json:"compatibility,omitempty"`
	AllowedTools  string            `json:"allowed_tools,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Content       string            `json:"content"`
	Files         json.RawMessage   `json:"files,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	Type          string            `json:"type"`
}

type SkillRepository interface {
	Create(ctx context.Context, skill *Skill) error
	GetByID(ctx context.Context, id string) (*Skill, error)
	GetByName(ctx context.Context, workspaceID string, name string) (*Skill, error)
	List(ctx context.Context, params SkillListParams) ([]*Skill, *string, error)
	Update(ctx context.Context, skill *Skill) error
	Delete(ctx context.Context, id string) error
}

type SkillListParams struct {
	ListParams
}
