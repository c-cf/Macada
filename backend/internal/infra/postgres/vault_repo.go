package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/c-cf/macada/internal/domain"
	"github.com/c-cf/macada/pkg/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VaultRepo struct {
	pool *pgxpool.Pool
}

func NewVaultRepo(pool *pgxpool.Pool) *VaultRepo {
	return &VaultRepo{pool: pool}
}

func (r *VaultRepo) Create(ctx context.Context, vault *domain.Vault) error {
	metaJSON, err := json.Marshal(vault.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO vaults (id, workspace_id, display_name, metadata, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		vault.ID, vault.WorkspaceID, vault.DisplayName, metaJSON,
		vault.CreatedAt, vault.UpdatedAt,
	)
	return err
}

func (r *VaultRepo) GetByID(ctx context.Context, id string) (*domain.Vault, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, workspace_id, display_name, metadata, archived_at, created_at, updated_at
		 FROM vaults WHERE id = $1`, id)
	return scanVault(row)
}

func (r *VaultRepo) List(ctx context.Context, params domain.VaultListParams) ([]*domain.Vault, *string, error) {
	limit := domain.DefaultLimit(params.Limit, 20)
	args := []interface{}{}
	conditions := []string{}
	argIdx := 1

	if params.WorkspaceID != "" {
		conditions = append(conditions, fmt.Sprintf("workspace_id = $%d", argIdx))
		args = append(args, params.WorkspaceID)
		argIdx++
	}
	if !params.IncludeArchived {
		conditions = append(conditions, "archived_at IS NULL")
	}
	if params.Page != nil {
		cursorTime, cursorID, err := pagination.DecodeCursor(*params.Page)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid page cursor: %w", err)
		}
		conditions = append(conditions, fmt.Sprintf("(created_at, id) < ($%d, $%d)", argIdx, argIdx+1))
		args = append(args, cursorTime, cursorID)
		argIdx += 2
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + joinAnd(conditions)
	}

	query := fmt.Sprintf(
		`SELECT id, workspace_id, display_name, metadata, archived_at, created_at, updated_at
		 FROM vaults %s ORDER BY created_at DESC, id DESC LIMIT $%d`,
		where, argIdx,
	)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var vaults []*domain.Vault
	for rows.Next() {
		v, err := scanVaultRows(rows)
		if err != nil {
			return nil, nil, err
		}
		vaults = append(vaults, v)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate vaults: %w", err)
	}

	var nextPage *string
	if len(vaults) > limit {
		vaults = vaults[:limit]
		last := vaults[limit-1]
		cursor := pagination.EncodeCursor(last.CreatedAt, last.ID)
		nextPage = &cursor
	}

	return vaults, nextPage, nil
}

// Update atomically updates the vault only if it belongs to the given workspace.
func (r *VaultRepo) Update(ctx context.Context, vault *domain.Vault, workspaceID string) error {
	metaJSON, err := json.Marshal(vault.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	now := time.Now().UTC()
	tag, err := r.pool.Exec(ctx,
		`UPDATE vaults SET display_name=$1, metadata=$2, updated_at=$3
		 WHERE id=$4 AND workspace_id=$5`,
		vault.DisplayName, metaJSON, now, vault.ID, workspaceID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrVaultNotFound
	}
	vault.UpdatedAt = now
	return nil
}

// Archive atomically archives the vault only if it belongs to the given workspace.
func (r *VaultRepo) Archive(ctx context.Context, id, workspaceID string) (*domain.Vault, error) {
	now := time.Now().UTC()
	row := r.pool.QueryRow(ctx,
		`UPDATE vaults SET archived_at=$1, updated_at=$1
		 WHERE id=$2 AND workspace_id=$3
		 RETURNING id, workspace_id, display_name, metadata, archived_at, created_at, updated_at`,
		now, id, workspaceID,
	)
	return scanVault(row)
}

// Delete atomically deletes the vault only if it belongs to the given workspace.
func (r *VaultRepo) Delete(ctx context.Context, id, workspaceID string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM vaults WHERE id = $1 AND workspace_id = $2`,
		id, workspaceID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrVaultNotFound
	}
	return nil
}

// --- Secret operations ---

// SetSecret upserts a secret, verifying vault ownership atomically via a subquery.
func (r *VaultRepo) SetSecret(ctx context.Context, vaultID, workspaceID, key string, encryptedValue, nonce []byte) error {
	now := time.Now().UTC()
	tag, err := r.pool.Exec(ctx,
		`INSERT INTO vault_secrets (vault_id, key, encrypted_value, nonce, created_at, updated_at)
		 SELECT $1, $2, $3, $4, $5, $5
		 FROM vaults WHERE id = $1 AND workspace_id = $6
		 ON CONFLICT (vault_id, key) DO UPDATE SET encrypted_value=$3, nonce=$4, updated_at=$5`,
		vaultID, key, encryptedValue, nonce, now, workspaceID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrVaultNotFound
	}
	return nil
}

// DeleteSecret removes a secret, verifying vault ownership atomically.
func (r *VaultRepo) DeleteSecret(ctx context.Context, vaultID, workspaceID, key string) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM vault_secrets
		 WHERE vault_id = $1 AND key = $2
		   AND vault_id IN (SELECT id FROM vaults WHERE id = $1 AND workspace_id = $3)`,
		vaultID, key, workspaceID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ListSecrets returns secret metadata (never values), verifying vault ownership.
func (r *VaultRepo) ListSecrets(ctx context.Context, vaultID, workspaceID string) ([]*domain.VaultSecret, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT vs.vault_id, vs.key, vs.created_at, vs.updated_at
		 FROM vault_secrets vs
		 JOIN vaults v ON v.id = vs.vault_id
		 WHERE vs.vault_id = $1 AND v.workspace_id = $2
		 ORDER BY vs.key`,
		vaultID, workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var secrets []*domain.VaultSecret
	for rows.Next() {
		var s domain.VaultSecret
		if err := rows.Scan(&s.VaultID, &s.Key, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		secrets = append(secrets, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate secrets: %w", err)
	}
	return secrets, nil
}

func (r *VaultRepo) GetEncryptedSecrets(ctx context.Context, vaultID string) ([]domain.EncryptedSecret, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT key, encrypted_value, nonce
		 FROM vault_secrets WHERE vault_id = $1`,
		vaultID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var secrets []domain.EncryptedSecret
	for rows.Next() {
		var s domain.EncryptedSecret
		if err := rows.Scan(&s.Key, &s.EncryptedValue, &s.Nonce); err != nil {
			return nil, err
		}
		secrets = append(secrets, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate encrypted secrets: %w", err)
	}
	return secrets, nil
}

// --- scan helpers ---

func scanVault(row pgx.Row) (*domain.Vault, error) {
	var v domain.Vault
	var metaJSON []byte
	err := row.Scan(&v.ID, &v.WorkspaceID, &v.DisplayName, &metaJSON,
		&v.ArchivedAt, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	v.Metadata = map[string]string{}
	if err := json.Unmarshal(metaJSON, &v.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	v.Type = "vault"
	return &v, nil
}

func scanVaultRows(rows pgx.Rows) (*domain.Vault, error) {
	var v domain.Vault
	var metaJSON []byte
	err := rows.Scan(&v.ID, &v.WorkspaceID, &v.DisplayName, &metaJSON,
		&v.ArchivedAt, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	v.Metadata = map[string]string{}
	if err := json.Unmarshal(metaJSON, &v.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	v.Type = "vault"
	return &v, nil
}
