package postgres

import (
	"context"
	"fmt"

	"github.com/c-cf/macada/internal/domain"
	"github.com/c-cf/macada/pkg/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FileRepo struct {
	pool *pgxpool.Pool
}

func NewFileRepo(pool *pgxpool.Pool) *FileRepo {
	return &FileRepo{pool: pool}
}

func (r *FileRepo) Create(ctx context.Context, file *domain.File) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO files (id, workspace_id, filename, mime_type, size_bytes, storage_path, downloadable, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		file.ID, file.WorkspaceID, file.Filename, file.MimeType,
		file.SizeBytes, file.StoragePath, file.Downloadable, file.CreatedAt,
	)
	return err
}

func (r *FileRepo) GetByID(ctx context.Context, id string) (*domain.File, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, workspace_id, filename, mime_type, size_bytes, storage_path, downloadable, created_at
		 FROM files WHERE id = $1`, id)
	return scanFile(row)
}

func (r *FileRepo) List(ctx context.Context, params domain.FileListParams) ([]*domain.File, *string, error) {
	limit := domain.DefaultLimit(params.Limit, 20)
	args := []interface{}{}
	conditions := []string{}
	argIdx := 1

	if params.WorkspaceID != "" {
		conditions = append(conditions, fmt.Sprintf("workspace_id = $%d", argIdx))
		args = append(args, params.WorkspaceID)
		argIdx++
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
		`SELECT id, workspace_id, filename, mime_type, size_bytes, storage_path, downloadable, created_at
		 FROM files %s ORDER BY created_at DESC, id DESC LIMIT $%d`,
		where, argIdx,
	)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var files []*domain.File
	for rows.Next() {
		f, err := scanFileRows(rows)
		if err != nil {
			return nil, nil, err
		}
		files = append(files, f)
	}

	var nextPage *string
	if len(files) > limit {
		files = files[:limit]
		last := files[limit-1]
		cursor := pagination.EncodeCursor(last.CreatedAt, last.ID)
		nextPage = &cursor
	}

	return files, nextPage, nil
}

func (r *FileRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM files WHERE id = $1`, id)
	return err
}

func scanFile(row pgx.Row) (*domain.File, error) {
	var f domain.File
	err := row.Scan(
		&f.ID, &f.WorkspaceID, &f.Filename, &f.MimeType,
		&f.SizeBytes, &f.StoragePath, &f.Downloadable, &f.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	f.Type = "file"
	return &f, nil
}

func scanFileRows(rows pgx.Rows) (*domain.File, error) {
	var f domain.File
	err := rows.Scan(
		&f.ID, &f.WorkspaceID, &f.Filename, &f.MimeType,
		&f.SizeBytes, &f.StoragePath, &f.Downloadable, &f.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	f.Type = "file"
	return &f, nil
}
