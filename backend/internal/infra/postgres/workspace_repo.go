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

type WorkspaceRepo struct {
	pool *pgxpool.Pool
}

func NewWorkspaceRepo(pool *pgxpool.Pool) *WorkspaceRepo {
	return &WorkspaceRepo{pool: pool}
}

func (r *WorkspaceRepo) Create(ctx context.Context, ws *domain.Workspace) error {
	metaJSON, err := json.Marshal(ws.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO workspaces (id, name, description, metadata, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		ws.ID, ws.Name, ws.Description, metaJSON, ws.CreatedAt, ws.UpdatedAt,
	)
	return err
}

func (r *WorkspaceRepo) GetByID(ctx context.Context, id string) (*domain.Workspace, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, name, description, metadata, archived_at, created_at, updated_at
		 FROM workspaces WHERE id = $1`, id)
	return scanWorkspace(row)
}

func (r *WorkspaceRepo) List(ctx context.Context, params domain.ListParams) ([]*domain.Workspace, *string, error) {
	limit := domain.DefaultLimit(params.Limit, 20)
	args := []interface{}{}
	conditions := []string{}
	argIdx := 1

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
		`SELECT id, name, description, metadata, archived_at, created_at, updated_at
		 FROM workspaces %s ORDER BY created_at DESC, id DESC LIMIT $%d`,
		where, argIdx,
	)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var workspaces []*domain.Workspace
	for rows.Next() {
		ws, err := scanWorkspaceRows(rows)
		if err != nil {
			return nil, nil, err
		}
		workspaces = append(workspaces, ws)
	}

	var nextPage *string
	if len(workspaces) > limit {
		workspaces = workspaces[:limit]
		last := workspaces[limit-1]
		cursor := pagination.EncodeCursor(last.CreatedAt, last.ID)
		nextPage = &cursor
	}

	return workspaces, nextPage, nil
}

func (r *WorkspaceRepo) Update(ctx context.Context, ws *domain.Workspace) error {
	metaJSON, err := json.Marshal(ws.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	ws.UpdatedAt = time.Now().UTC()
	_, err = r.pool.Exec(ctx,
		`UPDATE workspaces SET name=$1, description=$2, metadata=$3, updated_at=$4
		 WHERE id=$5`,
		ws.Name, ws.Description, metaJSON, ws.UpdatedAt, ws.ID,
	)
	return err
}

func (r *WorkspaceRepo) Archive(ctx context.Context, id string) (*domain.Workspace, error) {
	now := time.Now().UTC()
	row := r.pool.QueryRow(ctx,
		`UPDATE workspaces SET archived_at=$1, updated_at=$1
		 WHERE id=$2
		 RETURNING id, name, description, metadata, archived_at, created_at, updated_at`,
		now, id,
	)
	return scanWorkspace(row)
}

func scanWorkspace(row pgx.Row) (*domain.Workspace, error) {
	var ws domain.Workspace
	var metaJSON []byte
	err := row.Scan(
		&ws.ID, &ws.Name, &ws.Description, &metaJSON,
		&ws.ArchivedAt, &ws.CreatedAt, &ws.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	ws.Metadata = map[string]string{}
	if err := json.Unmarshal(metaJSON, &ws.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	ws.Type = "workspace"
	return &ws, nil
}

func scanWorkspaceRows(rows pgx.Rows) (*domain.Workspace, error) {
	var ws domain.Workspace
	var metaJSON []byte
	err := rows.Scan(
		&ws.ID, &ws.Name, &ws.Description, &metaJSON,
		&ws.ArchivedAt, &ws.CreatedAt, &ws.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	ws.Metadata = map[string]string{}
	if err := json.Unmarshal(metaJSON, &ws.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	ws.Type = "workspace"
	return &ws, nil
}
