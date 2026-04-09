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

type AgentRepo struct {
	pool *pgxpool.Pool
}

func NewAgentRepo(pool *pgxpool.Pool) *AgentRepo {
	return &AgentRepo{pool: pool}
}

func (r *AgentRepo) Create(ctx context.Context, agent *domain.Agent) error {
	modelJSON, err := json.Marshal(agent.Model)
	if err != nil {
		return fmt.Errorf("marshal model: %w", err)
	}
	metaJSON, err := json.Marshal(agent.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO agents (id, workspace_id, name, description, model, system, tools, mcp_servers, skills, metadata, version, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		agent.ID, agent.WorkspaceID, agent.Name, agent.Description, modelJSON, agent.System,
		agent.Tools, agent.MCPServers, agent.Skills, metaJSON,
		agent.Version, agent.CreatedAt, agent.UpdatedAt,
	)
	return err
}

func (r *AgentRepo) GetByID(ctx context.Context, id string) (*domain.Agent, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, workspace_id, name, description, model, system, tools, mcp_servers, skills, metadata, version, archived_at, created_at, updated_at
		 FROM agents WHERE id = $1`, id)
	return scanAgent(row)
}

func (r *AgentRepo) List(ctx context.Context, params domain.AgentListParams) ([]*domain.Agent, *string, error) {
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
	if params.CreatedAtGTE != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *params.CreatedAtGTE)
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

	query := fmt.Sprintf(
		`SELECT id, workspace_id, name, description, model, system, tools, mcp_servers, skills, metadata, version, archived_at, created_at, updated_at
		 FROM agents %s ORDER BY created_at DESC, id DESC LIMIT $%d`,
		where, argIdx,
	)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var agents []*domain.Agent
	for rows.Next() {
		a, err := scanAgentRows(rows)
		if err != nil {
			return nil, nil, err
		}
		agents = append(agents, a)
	}

	var nextPage *string
	if len(agents) > limit {
		agents = agents[:limit]
		last := agents[limit-1]
		cursor := pagination.EncodeCursor(last.CreatedAt, last.ID)
		nextPage = &cursor
	}

	return agents, nextPage, nil
}

func (r *AgentRepo) Update(ctx context.Context, agent *domain.Agent) error {
	modelJSON, err := json.Marshal(agent.Model)
	if err != nil {
		return fmt.Errorf("marshal model: %w", err)
	}
	metaJSON, err := json.Marshal(agent.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	agent.UpdatedAt = time.Now().UTC()
	_, err = r.pool.Exec(ctx,
		`UPDATE agents SET name=$1, description=$2, model=$3, system=$4, tools=$5, mcp_servers=$6, skills=$7, metadata=$8, version=$9, updated_at=$10
		 WHERE id=$11`,
		agent.Name, agent.Description, modelJSON, agent.System,
		agent.Tools, agent.MCPServers, agent.Skills, metaJSON,
		agent.Version, agent.UpdatedAt, agent.ID,
	)
	return err
}

func (r *AgentRepo) Archive(ctx context.Context, id string) (*domain.Agent, error) {
	now := time.Now().UTC()
	row := r.pool.QueryRow(ctx,
		`UPDATE agents SET archived_at=$1, updated_at=$1
		 WHERE id=$2
		 RETURNING id, workspace_id, name, description, model, system, tools, mcp_servers, skills, metadata, version, archived_at, created_at, updated_at`,
		now, id,
	)
	return scanAgent(row)
}

func scanAgent(row pgx.Row) (*domain.Agent, error) {
	var a domain.Agent
	var modelJSON, metaJSON []byte
	err := row.Scan(
		&a.ID, &a.WorkspaceID, &a.Name, &a.Description, &modelJSON, &a.System,
		&a.Tools, &a.MCPServers, &a.Skills, &metaJSON,
		&a.Version, &a.ArchivedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(modelJSON, &a.Model); err != nil {
		return nil, fmt.Errorf("unmarshal model: %w", err)
	}
	a.Metadata = map[string]string{}
	if err := json.Unmarshal(metaJSON, &a.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	a.Type = "agent"
	return &a, nil
}

func scanAgentRows(rows pgx.Rows) (*domain.Agent, error) {
	var a domain.Agent
	var modelJSON, metaJSON []byte
	err := rows.Scan(
		&a.ID, &a.WorkspaceID, &a.Name, &a.Description, &modelJSON, &a.System,
		&a.Tools, &a.MCPServers, &a.Skills, &metaJSON,
		&a.Version, &a.ArchivedAt, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(modelJSON, &a.Model); err != nil {
		return nil, fmt.Errorf("unmarshal model: %w", err)
	}
	a.Metadata = map[string]string{}
	if err := json.Unmarshal(metaJSON, &a.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	a.Type = "agent"
	return &a, nil
}
