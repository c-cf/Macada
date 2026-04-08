package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/cchu-code/managed-agents/internal/domain"
	"github.com/cchu-code/managed-agents/pkg/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type APIKeyRepo struct {
	pool *pgxpool.Pool
}

func NewAPIKeyRepo(pool *pgxpool.Pool) *APIKeyRepo {
	return &APIKeyRepo{pool: pool}
}

func (r *APIKeyRepo) Create(ctx context.Context, key *domain.APIKey, keyHash string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO api_keys (id, workspace_id, name, key_hash, key_prefix, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		key.ID, key.WorkspaceID, key.Name, keyHash, key.KeyPrefix,
		key.ExpiresAt, key.CreatedAt,
	)
	return err
}

func (r *APIKeyRepo) GetByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, workspace_id, name, key_prefix, expires_at, revoked_at, last_used_at, created_at
		 FROM api_keys WHERE key_hash = $1`, keyHash)
	return scanAPIKey(row)
}

func (r *APIKeyRepo) ListByWorkspace(ctx context.Context, workspaceID string, params domain.ListParams) ([]*domain.APIKey, *string, error) {
	limit := domain.DefaultLimit(params.Limit, 20)
	args := []interface{}{workspaceID}
	conditions := []string{"workspace_id = $1"}
	argIdx := 2

	if params.Page != nil {
		cursorTime, cursorID, err := pagination.DecodeCursor(*params.Page)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid page cursor: %w", err)
		}
		conditions = append(conditions, fmt.Sprintf("(created_at, id) < ($%d, $%d)", argIdx, argIdx+1))
		args = append(args, cursorTime, cursorID)
		argIdx += 2
	}

	where := "WHERE " + joinAnd(conditions)

	query := fmt.Sprintf(
		`SELECT id, workspace_id, name, key_prefix, expires_at, revoked_at, last_used_at, created_at
		 FROM api_keys %s ORDER BY created_at DESC, id DESC LIMIT $%d`,
		where, argIdx,
	)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var keys []*domain.APIKey
	for rows.Next() {
		k, err := scanAPIKeyRows(rows)
		if err != nil {
			return nil, nil, err
		}
		keys = append(keys, k)
	}

	var nextPage *string
	if len(keys) > limit {
		keys = keys[:limit]
		last := keys[limit-1]
		cursor := pagination.EncodeCursor(last.CreatedAt, last.ID)
		nextPage = &cursor
	}

	return keys, nextPage, nil
}

func (r *APIKeyRepo) Revoke(ctx context.Context, id string, workspaceID string) (*domain.APIKey, error) {
	now := time.Now().UTC()
	row := r.pool.QueryRow(ctx,
		`UPDATE api_keys SET revoked_at=$1
		 WHERE id=$2 AND workspace_id=$3
		 RETURNING id, workspace_id, name, key_prefix, expires_at, revoked_at, last_used_at, created_at`,
		now, id, workspaceID,
	)
	return scanAPIKey(row)
}

func (r *APIKeyRepo) Delete(ctx context.Context, id string, workspaceID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM api_keys WHERE id=$1 AND workspace_id=$2`,
		id, workspaceID,
	)
	return err
}

func (r *APIKeyRepo) TouchLastUsed(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE api_keys SET last_used_at=$1 WHERE id=$2`,
		time.Now().UTC(), id,
	)
	return err
}

func scanAPIKey(row pgx.Row) (*domain.APIKey, error) {
	var k domain.APIKey
	err := row.Scan(
		&k.ID, &k.WorkspaceID, &k.Name, &k.KeyPrefix,
		&k.ExpiresAt, &k.RevokedAt, &k.LastUsedAt, &k.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	k.Type = "api_key"
	return &k, nil
}

func scanAPIKeyRows(rows pgx.Rows) (*domain.APIKey, error) {
	var k domain.APIKey
	err := rows.Scan(
		&k.ID, &k.WorkspaceID, &k.Name, &k.KeyPrefix,
		&k.ExpiresAt, &k.RevokedAt, &k.LastUsedAt, &k.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	k.Type = "api_key"
	return &k, nil
}
