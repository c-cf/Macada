package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cchu-code/managed-agents/internal/domain"
	"github.com/cchu-code/managed-agents/pkg/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepo struct {
	pool *pgxpool.Pool
}

func NewSessionRepo(pool *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{pool: pool}
}

func (r *SessionRepo) Create(ctx context.Context, s *domain.Session) error {
	statsJSON, _ := json.Marshal(s.Stats)
	usageJSON, _ := json.Marshal(s.Usage)
	metaJSON, _ := json.Marshal(s.Metadata)
	vaultJSON, _ := json.Marshal(s.VaultIDs)

	resources := s.Resources
	if resources == nil {
		resources = json.RawMessage("[]")
	}

	memory := s.Memory
	if memory == nil {
		memory = json.RawMessage("{}")
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO sessions (id, workspace_id, agent_snapshot, environment_id, title, status, stats, usage, resources, metadata, vault_ids, memory, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		s.ID, s.WorkspaceID, s.Agent, s.EnvironmentID, s.Title, s.Status,
		statsJSON, usageJSON, resources, metaJSON, vaultJSON, memory,
		s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (r *SessionRepo) GetByID(ctx context.Context, id string) (*domain.Session, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, workspace_id, agent_snapshot, environment_id, title, status, stats, usage, resources, metadata, vault_ids, memory, archived_at, created_at, updated_at
		 FROM sessions WHERE id = $1`, id)
	return scanSession(row)
}

func (r *SessionRepo) List(ctx context.Context, params domain.SessionListParams) ([]*domain.Session, *string, error) {
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
	if params.AgentID != nil {
		conditions = append(conditions, fmt.Sprintf("agent_snapshot->>'id' = $%d", argIdx))
		args = append(args, *params.AgentID)
		argIdx++
	}
	if params.CreatedAtGT != nil {
		conditions = append(conditions, fmt.Sprintf("created_at > $%d", argIdx))
		args = append(args, *params.CreatedAtGT)
		argIdx++
	}
	if params.CreatedAtGTE != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *params.CreatedAtGTE)
		argIdx++
	}
	if params.CreatedAtLT != nil {
		conditions = append(conditions, fmt.Sprintf("created_at < $%d", argIdx))
		args = append(args, *params.CreatedAtLT)
		argIdx++
	}
	if params.CreatedAtLTE != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *params.CreatedAtLTE)
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

	order := "DESC"
	if params.Order != nil && *params.Order == "asc" {
		order = "ASC"
	}

	query := fmt.Sprintf(
		`SELECT id, workspace_id, agent_snapshot, environment_id, title, status, stats, usage, resources, metadata, vault_ids, memory, archived_at, created_at, updated_at
		 FROM sessions %s ORDER BY created_at %s, id %s LIMIT $%d`,
		where, order, order, argIdx,
	)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var sessions []*domain.Session
	for rows.Next() {
		s, err := scanSessionRows(rows)
		if err != nil {
			return nil, nil, err
		}
		sessions = append(sessions, s)
	}

	var nextPage *string
	if len(sessions) > limit {
		sessions = sessions[:limit]
		last := sessions[limit-1]
		cursor := pagination.EncodeCursor(last.CreatedAt, last.ID)
		nextPage = &cursor
	}

	return sessions, nextPage, nil
}

func (r *SessionRepo) Update(ctx context.Context, s *domain.Session) error {
	statsJSON, _ := json.Marshal(s.Stats)
	usageJSON, _ := json.Marshal(s.Usage)
	metaJSON, _ := json.Marshal(s.Metadata)

	s.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE sessions SET title=$1, status=$2, stats=$3, usage=$4, metadata=$5, updated_at=$6
		 WHERE id=$7`,
		s.Title, s.Status, statsJSON, usageJSON, metaJSON, s.UpdatedAt, s.ID,
	)
	return err
}

func (r *SessionRepo) UpdateStatus(ctx context.Context, id string, status domain.SessionStatus) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sessions SET status=$1, updated_at=$2 WHERE id=$3`,
		status, time.Now().UTC(), id,
	)
	return err
}

func (r *SessionRepo) UpdateUsage(ctx context.Context, id string, usage domain.SessionUsage) error {
	usageJSON, _ := json.Marshal(usage)
	_, err := r.pool.Exec(ctx,
		`UPDATE sessions SET usage=$1, updated_at=$2 WHERE id=$3`,
		usageJSON, time.Now().UTC(), id,
	)
	return err
}

func (r *SessionRepo) Archive(ctx context.Context, id string) (*domain.Session, error) {
	now := time.Now().UTC()
	row := r.pool.QueryRow(ctx,
		`UPDATE sessions SET archived_at=$1, updated_at=$1
		 WHERE id=$2
		 RETURNING id, workspace_id, agent_snapshot, environment_id, title, status, stats, usage, resources, metadata, vault_ids, memory, archived_at, created_at, updated_at`,
		now, id,
	)
	return scanSession(row)
}

func (r *SessionRepo) UpdateMemory(ctx context.Context, id string, memory json.RawMessage) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE sessions SET memory=$1, updated_at=$2 WHERE id=$3`,
		memory, time.Now().UTC(), id,
	)
	return err
}

func scanSession(row pgx.Row) (*domain.Session, error) {
	var s domain.Session
	var statsJSON, usageJSON, metaJSON, vaultJSON []byte
	err := row.Scan(
		&s.ID, &s.WorkspaceID, &s.Agent, &s.EnvironmentID, &s.Title, &s.Status,
		&statsJSON, &usageJSON, &s.Resources, &metaJSON, &vaultJSON,
		&s.Memory, &s.ArchivedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(statsJSON, &s.Stats)
	_ = json.Unmarshal(usageJSON, &s.Usage)
	s.Metadata = map[string]string{}
	_ = json.Unmarshal(metaJSON, &s.Metadata)
	s.VaultIDs = []string{}
	_ = json.Unmarshal(vaultJSON, &s.VaultIDs)
	s.Type = "session"
	return &s, nil
}

func scanSessionRows(rows pgx.Rows) (*domain.Session, error) {
	var s domain.Session
	var statsJSON, usageJSON, metaJSON, vaultJSON []byte
	err := rows.Scan(
		&s.ID, &s.WorkspaceID, &s.Agent, &s.EnvironmentID, &s.Title, &s.Status,
		&statsJSON, &usageJSON, &s.Resources, &metaJSON, &vaultJSON,
		&s.Memory, &s.ArchivedAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(statsJSON, &s.Stats)
	_ = json.Unmarshal(usageJSON, &s.Usage)
	s.Metadata = map[string]string{}
	_ = json.Unmarshal(metaJSON, &s.Metadata)
	s.VaultIDs = []string{}
	_ = json.Unmarshal(vaultJSON, &s.VaultIDs)
	s.Type = "session"
	return &s, nil
}
