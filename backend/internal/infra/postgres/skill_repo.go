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

type SkillRepo struct {
	pool *pgxpool.Pool
}

func NewSkillRepo(pool *pgxpool.Pool) *SkillRepo {
	return &SkillRepo{pool: pool}
}

func (r *SkillRepo) Create(ctx context.Context, skill *domain.Skill) error {
	metaJSON, err := json.Marshal(skill.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO skills (id, workspace_id, name, description, license, compatibility, allowed_tools, metadata, content, files, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		skill.ID, skill.WorkspaceID, skill.Name, skill.Description, skill.License, skill.Compatibility,
		skill.AllowedTools, metaJSON, skill.Content, skill.Files,
		skill.CreatedAt, skill.UpdatedAt,
	)
	return err
}

func (r *SkillRepo) GetByID(ctx context.Context, id string) (*domain.Skill, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, workspace_id, name, description, license, compatibility, allowed_tools, metadata, content, files, created_at, updated_at
		 FROM skills WHERE id = $1`, id)
	return scanSkill(row)
}

func (r *SkillRepo) GetByName(ctx context.Context, workspaceID string, name string) (*domain.Skill, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, workspace_id, name, description, license, compatibility, allowed_tools, metadata, content, files, created_at, updated_at
		 FROM skills WHERE workspace_id = $1 AND name = $2`, workspaceID, name)
	return scanSkill(row)
}

func (r *SkillRepo) List(ctx context.Context, params domain.SkillListParams) ([]*domain.Skill, *string, error) {
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
		`SELECT id, workspace_id, name, description, license, compatibility, allowed_tools, metadata, content, files, created_at, updated_at
		 FROM skills %s ORDER BY created_at DESC, id DESC LIMIT $%d`,
		where, argIdx,
	)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var skills []*domain.Skill
	for rows.Next() {
		s, err := scanSkillRows(rows)
		if err != nil {
			return nil, nil, err
		}
		skills = append(skills, s)
	}

	var nextPage *string
	if len(skills) > limit {
		skills = skills[:limit]
		last := skills[limit-1]
		cursor := pagination.EncodeCursor(last.CreatedAt, last.ID)
		nextPage = &cursor
	}

	return skills, nextPage, nil
}

func (r *SkillRepo) Update(ctx context.Context, skill *domain.Skill) error {
	metaJSON, err := json.Marshal(skill.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	skill.UpdatedAt = time.Now().UTC()
	_, err = r.pool.Exec(ctx,
		`UPDATE skills SET name=$1, description=$2, license=$3, compatibility=$4, allowed_tools=$5, metadata=$6, content=$7, files=$8, updated_at=$9
		 WHERE id=$10`,
		skill.Name, skill.Description, skill.License, skill.Compatibility,
		skill.AllowedTools, metaJSON, skill.Content, skill.Files,
		skill.UpdatedAt, skill.ID,
	)
	return err
}

func (r *SkillRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM skills WHERE id = $1`, id)
	return err
}

func scanSkill(row pgx.Row) (*domain.Skill, error) {
	var s domain.Skill
	var metaJSON []byte
	err := row.Scan(
		&s.ID, &s.WorkspaceID, &s.Name, &s.Description, &s.License, &s.Compatibility,
		&s.AllowedTools, &metaJSON, &s.Content, &s.Files,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.Metadata = map[string]string{}
	if err := json.Unmarshal(metaJSON, &s.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	s.Type = "skill"
	return &s, nil
}

func scanSkillRows(rows pgx.Rows) (*domain.Skill, error) {
	var s domain.Skill
	var metaJSON []byte
	err := rows.Scan(
		&s.ID, &s.WorkspaceID, &s.Name, &s.Description, &s.License, &s.Compatibility,
		&s.AllowedTools, &metaJSON, &s.Content, &s.Files,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.Metadata = map[string]string{}
	if err := json.Unmarshal(metaJSON, &s.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	s.Type = "skill"
	return &s, nil
}
