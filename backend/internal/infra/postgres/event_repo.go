package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cchu-code/managed-agents/internal/domain"
	"github.com/cchu-code/managed-agents/pkg/pagination"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventRepo struct {
	pool *pgxpool.Pool
}

func NewEventRepo(pool *pgxpool.Pool) *EventRepo {
	return &EventRepo{pool: pool}
}

func (r *EventRepo) Create(ctx context.Context, event *domain.Event) error {
	payload := event.Payload
	if payload == nil {
		payload = json.RawMessage("{}")
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO events (id, session_id, type, payload, processed_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		event.ID, event.SessionID, event.Type, payload, event.ProcessedAt,
	)
	return err
}

func (r *EventRepo) ListBySession(ctx context.Context, sessionID string, params domain.EventListParams) ([]*domain.Event, *string, error) {
	limit := domain.DefaultLimit(params.Limit, 100)
	args := []interface{}{sessionID}
	conditions := []string{"session_id = $1"}
	argIdx := 2

	order := "ASC"
	if params.Order != nil && *params.Order == "desc" {
		order = "DESC"
	}

	if params.Page != nil {
		cursorTime, cursorID, err := pagination.DecodeCursor(*params.Page)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid page cursor: %w", err)
		}
		if order == "ASC" {
			conditions = append(conditions, fmt.Sprintf("(created_at, id) > ($%d, $%d)", argIdx, argIdx+1))
		} else {
			conditions = append(conditions, fmt.Sprintf("(created_at, id) < ($%d, $%d)", argIdx, argIdx+1))
		}
		args = append(args, cursorTime, cursorID)
		argIdx += 2
	}

	query := fmt.Sprintf(
		`SELECT id, session_id, type, payload, processed_at
		 FROM events WHERE %s ORDER BY created_at %s, id %s LIMIT $%d`,
		joinAnd(conditions), order, order, argIdx,
	)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		var e domain.Event
		if err := rows.Scan(&e.ID, &e.SessionID, &e.Type, &e.Payload, &e.ProcessedAt); err != nil {
			return nil, nil, err
		}
		events = append(events, &e)
	}

	var nextPage *string
	if len(events) > limit {
		events = events[:limit]
		last := events[limit-1]
		cursor := pagination.EncodeCursor(last.ProcessedAt, last.ID)
		nextPage = &cursor
	}

	return events, nextPage, nil
}
