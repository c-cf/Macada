package postgres

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	return pool, nil
}

func MigrateUp(ctx context.Context, pool *pgxpool.Pool) error {
	files := []string{
		"migrations/001_init.up.sql",
		"migrations/002_analytics.up.sql",
		"migrations/003_skills.up.sql",
		"migrations/004_session_memory.up.sql",
		"migrations/005_sandbox.up.sql",
		"migrations/006_workspaces.up.sql",
		"migrations/007_users.up.sql",
	}
	for _, file := range files {
		data, err := migrationsFS.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", file, err)
		}

		statements := splitStatements(string(data))
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := pool.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("failed to exec migration statement: %w\nSQL: %s", err, stmt)
			}
		}
	}
	return nil
}

func MigrateDown(ctx context.Context, pool *pgxpool.Pool) error {
	// Run down migrations in reverse order
	files := []string{
		"migrations/007_users.down.sql",
		"migrations/006_workspaces.down.sql",
		"migrations/005_sandbox.down.sql",
		"migrations/004_session_memory.down.sql",
		"migrations/003_skills.down.sql",
		"migrations/002_analytics.down.sql",
		"migrations/001_init.down.sql",
	}
	for _, file := range files {
		data, err := migrationsFS.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", file, err)
		}

		statements := splitStatements(string(data))
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := pool.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("failed to exec migration statement: %w\nSQL: %s", err, stmt)
			}
		}
	}
	return nil
}

func splitStatements(sql string) []string {
	// Simple split on semicolons. Works for our straightforward DDL.
	var result []string
	for _, s := range strings.Split(sql, ";") {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}
