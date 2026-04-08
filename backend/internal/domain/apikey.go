package domain

import (
	"context"
	"time"
)

type APIKey struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspace_id"`
	Name        string     `json:"name"`
	KeyPrefix   string     `json:"key_prefix"`
	ExpiresAt   *time.Time `json:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at"`
	Type        string     `json:"type"`
}

// APIKeyCreateResult is returned only on creation — contains the plaintext key.
type APIKeyCreateResult struct {
	APIKey
	Key string `json:"key"`
}

type APIKeyRepository interface {
	Create(ctx context.Context, key *APIKey, keyHash string) error
	GetByHash(ctx context.Context, keyHash string) (*APIKey, error)
	ListByWorkspace(ctx context.Context, workspaceID string, params ListParams) ([]*APIKey, *string, error)
	Revoke(ctx context.Context, id string, workspaceID string) (*APIKey, error)
	Delete(ctx context.Context, id string, workspaceID string) error
	TouchLastUsed(ctx context.Context, id string) error
}
