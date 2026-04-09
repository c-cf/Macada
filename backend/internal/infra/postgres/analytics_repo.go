package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/c-cf/macada/pkg/pagination"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UsageDayRow represents a single row of aggregated daily usage data.
type UsageDayRow struct {
	Day                 string `json:"day"`
	Model               string `json:"model"`
	InputTokens         int64  `json:"input_tokens"`
	OutputTokens        int64  `json:"output_tokens"`
	CacheReadTokens     int64  `json:"cache_read_tokens"`
	CacheCreationTokens int64  `json:"cache_creation_tokens"`
	RequestCount        int64  `json:"request_count"`
	WorkspaceID         string `json:"workspace_id"`
}

// LogRow represents a single LLM request log entry.
type LogRow struct {
	ID                  string    `json:"id"`
	SessionID           string    `json:"session_id"`
	AgentID             string    `json:"agent_id"`
	Model               string    `json:"model"`
	InputTokens         int64     `json:"input_tokens"`
	OutputTokens        int64     `json:"output_tokens"`
	CacheReadTokens     int64     `json:"cache_read_tokens"`
	CacheCreationTokens int64     `json:"cache_creation_tokens"`
	LatencyMs           int       `json:"latency_ms"`
	IsError             bool      `json:"is_error"`
	CreatedAt           time.Time `json:"created_at"`
	WorkspaceID         string    `json:"workspace_id"`
}

// AnalyticsRepo handles persistence for analytics data (llm_request_logs and usage_daily).
type AnalyticsRepo struct {
	pool *pgxpool.Pool
}

func NewAnalyticsRepo(pool *pgxpool.Pool) *AnalyticsRepo {
	return &AnalyticsRepo{pool: pool}
}

// InsertRequestLog inserts a new LLM request log row.
func (r *AnalyticsRepo) InsertRequestLog(ctx context.Context, log LogRow) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO llm_request_logs
		 (id, workspace_id, session_id, agent_id, model, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, latency_ms, is_error, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		log.ID, log.WorkspaceID, log.SessionID, log.AgentID, log.Model,
		log.InputTokens, log.OutputTokens, log.CacheReadTokens, log.CacheCreationTokens,
		log.LatencyMs, log.IsError, log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert request log: %w", err)
	}
	return nil
}

// IncrementDailyUsage upserts a row in usage_daily, adding to existing counters.
func (r *AnalyticsRepo) IncrementDailyUsage(ctx context.Context, workspaceID string, day time.Time, model string, inputTokens, outputTokens, cacheReadTokens, cacheCreationTokens int64) error {
	dayStr := day.Format("2006-01-02")
	_, err := r.pool.Exec(ctx,
		`INSERT INTO usage_daily (day, model, workspace_id, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, request_count)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 1)
		 ON CONFLICT (day, model, workspace_id) DO UPDATE SET
		   input_tokens = usage_daily.input_tokens + EXCLUDED.input_tokens,
		   output_tokens = usage_daily.output_tokens + EXCLUDED.output_tokens,
		   cache_read_tokens = usage_daily.cache_read_tokens + EXCLUDED.cache_read_tokens,
		   cache_creation_tokens = usage_daily.cache_creation_tokens + EXCLUDED.cache_creation_tokens,
		   request_count = usage_daily.request_count + 1`,
		dayStr, model, workspaceID, inputTokens, outputTokens, cacheReadTokens, cacheCreationTokens,
	)
	if err != nil {
		return fmt.Errorf("failed to increment daily usage: %w", err)
	}
	return nil
}

// QueryUsage returns daily usage data for a date range, optionally filtered by model.
func (r *AnalyticsRepo) QueryUsage(ctx context.Context, workspaceID string, from, to string, model string) ([]UsageDayRow, error) {
	args := []interface{}{workspaceID, from, to}
	query := `SELECT day, model, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, request_count
	          FROM usage_daily WHERE workspace_id = $1 AND day >= $2 AND day <= $3`

	if model != "" {
		query += " AND model = $4"
		args = append(args, model)
	}
	query += " ORDER BY day ASC, model ASC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage: %w", err)
	}
	defer rows.Close()

	var result []UsageDayRow
	for rows.Next() {
		var row UsageDayRow
		var dayTime time.Time
		if err := rows.Scan(&dayTime, &row.Model, &row.InputTokens, &row.OutputTokens, &row.CacheReadTokens, &row.CacheCreationTokens, &row.RequestCount); err != nil {
			return nil, fmt.Errorf("failed to scan usage row: %w", err)
		}
		row.Day = dayTime.Format("2006-01-02")
		result = append(result, row)
	}
	return result, nil
}

// QueryLogs returns paginated LLM request logs, newest first.
func (r *AnalyticsRepo) QueryLogs(ctx context.Context, workspaceID string, limit int, pageCursor *string) ([]LogRow, *string, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	args := []interface{}{workspaceID}
	conditions := []string{"workspace_id = $1"}
	argIdx := 2

	if pageCursor != nil {
		cursorTime, cursorID, err := pagination.DecodeCursor(*pageCursor)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid page cursor: %w", err)
		}
		conditions = append(conditions, fmt.Sprintf("(created_at, id) < ($%d, $%d)", argIdx, argIdx+1))
		args = append(args, cursorTime, cursorID)
		argIdx += 2
	}

	where := "WHERE " + joinAnd(conditions)

	query := fmt.Sprintf(
		`SELECT id, session_id, agent_id, model, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, latency_ms, is_error, created_at
		 FROM llm_request_logs %s ORDER BY created_at DESC, id DESC LIMIT $%d`,
		where, argIdx,
	)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var logs []LogRow
	for rows.Next() {
		var row LogRow
		if err := rows.Scan(
			&row.ID, &row.SessionID, &row.AgentID, &row.Model,
			&row.InputTokens, &row.OutputTokens, &row.CacheReadTokens, &row.CacheCreationTokens,
			&row.LatencyMs, &row.IsError, &row.CreatedAt,
		); err != nil {
			return nil, nil, fmt.Errorf("failed to scan log row: %w", err)
		}
		logs = append(logs, row)
	}

	var nextPage *string
	if len(logs) > limit {
		logs = logs[:limit]
		last := logs[limit-1]
		cursor := pagination.EncodeCursor(last.CreatedAt, last.ID)
		nextPage = &cursor
	}

	return logs, nextPage, nil
}
