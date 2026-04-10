package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/c-cf/macada/internal/domain"
	"github.com/c-cf/macada/pkg/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ResourceRepo struct {
	pool *pgxpool.Pool
}

func NewResourceRepo(pool *pgxpool.Pool) *ResourceRepo {
	return &ResourceRepo{pool: pool}
}

func (r *ResourceRepo) Create(ctx context.Context, res *domain.SessionResource) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO session_resources (id, session_id, type, file_id, mount_path, config, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		res.ID, res.SessionID, res.Type, res.FileID, res.MountPath,
		res.Config, res.CreatedAt, res.UpdatedAt,
	)
	return err
}

func (r *ResourceRepo) GetByID(ctx context.Context, id string) (*domain.SessionResource, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, session_id, type, file_id, mount_path, config, created_at, updated_at
		 FROM session_resources WHERE id = $1`, id)
	return scanResource(row)
}

func (r *ResourceRepo) ListBySession(ctx context.Context, sessionID string, params domain.ListParams) ([]*domain.SessionResource, *string, error) {
	limit := domain.DefaultLimit(params.Limit, 100)
	args := []interface{}{sessionID}
	conditions := []string{"session_id = $1"}
	argIdx := 2

	if params.Page != nil {
		cursorTime, cursorID, err := pagination.DecodeCursor(*params.Page)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid page cursor: %w", err)
		}
		conditions = append(conditions, fmt.Sprintf("(created_at, id) > ($%d, $%d)", argIdx, argIdx+1))
		args = append(args, cursorTime, cursorID)
		argIdx += 2
	}

	where := "WHERE " + joinAnd(conditions)

	query := fmt.Sprintf(
		`SELECT id, session_id, type, file_id, mount_path, config, created_at, updated_at
		 FROM session_resources %s ORDER BY created_at ASC, id ASC LIMIT $%d`,
		where, argIdx,
	)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var resources []*domain.SessionResource
	for rows.Next() {
		res, err := scanResourceRows(rows)
		if err != nil {
			return nil, nil, err
		}
		resources = append(resources, res)
	}

	var nextPage *string
	if len(resources) > limit {
		resources = resources[:limit]
		last := resources[limit-1]
		cursor := pagination.EncodeCursor(last.CreatedAt, last.ID)
		nextPage = &cursor
	}

	return resources, nextPage, nil
}

func (r *ResourceRepo) Update(ctx context.Context, res *domain.SessionResource) error {
	res.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE session_resources SET config=$1, mount_path=$2, updated_at=$3 WHERE id=$4`,
		res.Config, res.MountPath, res.UpdatedAt, res.ID,
	)
	return err
}

func (r *ResourceRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM session_resources WHERE id = $1`, id)
	return err
}

func (r *ResourceRepo) ListFileResourcesBySession(ctx context.Context, sessionID string) ([]*domain.SessionResource, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, session_id, type, file_id, mount_path, config, created_at, updated_at
		 FROM session_resources WHERE session_id = $1 AND type = 'file' ORDER BY created_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []*domain.SessionResource
	for rows.Next() {
		res, err := scanResourceRows(rows)
		if err != nil {
			return nil, err
		}
		resources = append(resources, res)
	}
	return resources, nil
}

func scanResource(row pgx.Row) (*domain.SessionResource, error) {
	var res domain.SessionResource
	err := row.Scan(
		&res.ID, &res.SessionID, &res.Type, &res.FileID,
		&res.MountPath, &res.Config, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func scanResourceRows(rows pgx.Rows) (*domain.SessionResource, error) {
	var res domain.SessionResource
	err := rows.Scan(
		&res.ID, &res.SessionID, &res.Type, &res.FileID,
		&res.MountPath, &res.Config, &res.CreatedAt, &res.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
