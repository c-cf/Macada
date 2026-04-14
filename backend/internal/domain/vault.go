package domain

import (
	"context"
	"fmt"
	"time"
)

// Vault stores credentials for use by agents during sessions.
type Vault struct {
	ID          string            `json:"id"`
	WorkspaceID string            `json:"-"`
	DisplayName string            `json:"display_name"`
	Metadata    map[string]string `json:"metadata"`
	ArchivedAt  *time.Time        `json:"archived_at"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Type        string            `json:"type"`
}

// VaultSecret is a single key-value secret within a vault.
// The Value field is always empty in API responses; it is only populated
// internally when injecting secrets into sandbox containers.
type VaultSecret struct {
	VaultID   string    `json:"-"`
	Key       string    `json:"key"`
	Value     string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DeletedVault is the response when a vault is permanently deleted.
type DeletedVault struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// VaultRepository defines storage operations for vaults and their secrets.
// Write methods accept workspaceID to enforce ownership atomically in SQL.
type VaultRepository interface {
	Create(ctx context.Context, vault *Vault) error
	GetByID(ctx context.Context, id string) (*Vault, error)
	List(ctx context.Context, params VaultListParams) ([]*Vault, *string, error)
	Update(ctx context.Context, vault *Vault, workspaceID string) error
	Archive(ctx context.Context, id, workspaceID string) (*Vault, error)
	Delete(ctx context.Context, id, workspaceID string) error

	// Secret operations — workspaceID enforces vault ownership atomically.
	SetSecret(ctx context.Context, vaultID, workspaceID, key string, encryptedValue, nonce []byte) error
	DeleteSecret(ctx context.Context, vaultID, workspaceID, key string) error
	ListSecrets(ctx context.Context, vaultID, workspaceID string) ([]*VaultSecret, error)
	GetEncryptedSecrets(ctx context.Context, vaultID string) ([]EncryptedSecret, error)
}

// ErrVaultNotFound is returned when a vault does not exist or does not belong
// to the specified workspace.
var ErrVaultNotFound = fmt.Errorf("vault not found")

// EncryptedSecret holds the raw encrypted data as stored in the database.
type EncryptedSecret struct {
	Key            string
	EncryptedValue []byte
	Nonce          []byte
}

// VaultListParams controls vault listing filters.
type VaultListParams struct {
	ListParams
}
