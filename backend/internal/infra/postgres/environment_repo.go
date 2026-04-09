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

type EnvironmentRepo struct {
	pool *pgxpool.Pool
}

func NewEnvironmentRepo(pool *pgxpool.Pool) *EnvironmentRepo {
	return &EnvironmentRepo{pool: pool}
}

func (r *EnvironmentRepo) Create(ctx context.Context, env *domain.Environment) error {
	configJSON, err := json.Marshal(env.Config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	metaJSON, err := json.Marshal(env.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO environments (id, workspace_id, name, description, config, metadata, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		env.ID, env.WorkspaceID, env.Name, env.Description, configJSON, metaJSON, env.CreatedAt, env.UpdatedAt,
	)
	return err
}

func (r *EnvironmentRepo) GetByID(ctx context.Context, id string) (*domain.Environment, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, workspace_id, name, description, config, metadata, archived_at, created_at, updated_at
		 FROM environments WHERE id = $1`, id)
	return scanEnvironment(row)
}

func (r *EnvironmentRepo) List(ctx context.Context, params domain.ListParams) ([]*domain.Environment, *string, error) {
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
		`SELECT id, workspace_id, name, description, config, metadata, archived_at, created_at, updated_at
		 FROM environments %s ORDER BY created_at DESC, id DESC LIMIT $%d`,
		where, argIdx,
	)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var envs []*domain.Environment
	for rows.Next() {
		env, err := scanEnvironmentRows(rows)
		if err != nil {
			return nil, nil, err
		}
		envs = append(envs, env)
	}

	var nextPage *string
	if len(envs) > limit {
		envs = envs[:limit]
		last := envs[limit-1]
		cursor := pagination.EncodeCursor(last.CreatedAt, last.ID)
		nextPage = &cursor
	}

	return envs, nextPage, nil
}

func (r *EnvironmentRepo) Update(ctx context.Context, env *domain.Environment) error {
	configJSON, err := json.Marshal(env.Config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	metaJSON, err := json.Marshal(env.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	env.UpdatedAt = time.Now().UTC()
	_, err = r.pool.Exec(ctx,
		`UPDATE environments SET name=$1, description=$2, config=$3, metadata=$4, updated_at=$5
		 WHERE id=$6`,
		env.Name, env.Description, configJSON, metaJSON, env.UpdatedAt, env.ID,
	)
	return err
}

func (r *EnvironmentRepo) Archive(ctx context.Context, id string) (*domain.Environment, error) {
	now := time.Now().UTC()
	row := r.pool.QueryRow(ctx,
		`UPDATE environments SET archived_at=$1, updated_at=$1
		 WHERE id=$2
		 RETURNING id, workspace_id, name, description, config, metadata, archived_at, created_at, updated_at`,
		now, id,
	)
	return scanEnvironment(row)
}

func (r *EnvironmentRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM environments WHERE id=$1`, id)
	return err
}

func scanEnvironment(row pgx.Row) (*domain.Environment, error) {
	var env domain.Environment
	var configJSON, metaJSON []byte
	err := row.Scan(
		&env.ID, &env.WorkspaceID, &env.Name, &env.Description,
		&configJSON, &metaJSON,
		&env.ArchivedAt, &env.CreatedAt, &env.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(configJSON, &env.Config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	env.Metadata = map[string]string{}
	if err := json.Unmarshal(metaJSON, &env.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	env.Type = "environment"
	return &env, nil
}

func scanEnvironmentRows(rows pgx.Rows) (*domain.Environment, error) {
	var env domain.Environment
	var configJSON, metaJSON []byte
	err := rows.Scan(
		&env.ID, &env.WorkspaceID, &env.Name, &env.Description,
		&configJSON, &metaJSON,
		&env.ArchivedAt, &env.CreatedAt, &env.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(configJSON, &env.Config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	env.Metadata = map[string]string{}
	if err := json.Unmarshal(metaJSON, &env.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	env.Type = "environment"
	return &env, nil
}

func joinAnd(conds []string) string {
	result := conds[0]
	for _, c := range conds[1:] {
		result += " AND " + c
	}
	return result
}
