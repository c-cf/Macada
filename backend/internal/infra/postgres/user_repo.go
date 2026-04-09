package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/c-cf/macada/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, name, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		user.ID, user.Email, user.PasswordHash, user.Name, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, name, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	u.Type = "user"
	return &u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, name, created_at, updated_at
		 FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	u.Type = "user"
	return &u, nil
}

func (r *UserRepo) Update(ctx context.Context, user *domain.User) error {
	user.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET email=$1, name=$2, updated_at=$3 WHERE id=$4`,
		user.Email, user.Name, user.UpdatedAt, user.ID,
	)
	return err
}

// WorkspaceMemberRepo implements domain.WorkspaceMemberRepository.

type WorkspaceMemberRepo struct {
	pool *pgxpool.Pool
}

func NewWorkspaceMemberRepo(pool *pgxpool.Pool) *WorkspaceMemberRepo {
	return &WorkspaceMemberRepo{pool: pool}
}

func (r *WorkspaceMemberRepo) Add(ctx context.Context, member *domain.WorkspaceMember) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO workspace_members (workspace_id, user_id, role, created_at)
		 VALUES ($1, $2, $3, $4)`,
		member.WorkspaceID, member.UserID, member.Role, member.CreatedAt,
	)
	return err
}

func (r *WorkspaceMemberRepo) ListByUser(ctx context.Context, userID string) ([]*domain.WorkspaceMember, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT workspace_id, user_id, role, created_at
		 FROM workspace_members WHERE user_id = $1 ORDER BY created_at ASC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*domain.WorkspaceMember
	for rows.Next() {
		var m domain.WorkspaceMember
		if err := rows.Scan(&m.WorkspaceID, &m.UserID, &m.Role, &m.CreatedAt); err != nil {
			return nil, err
		}
		members = append(members, &m)
	}
	return members, nil
}

func (r *WorkspaceMemberRepo) ListByWorkspace(ctx context.Context, workspaceID string) ([]*domain.WorkspaceMember, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT workspace_id, user_id, role, created_at
		 FROM workspace_members WHERE workspace_id = $1 ORDER BY created_at ASC`, workspaceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*domain.WorkspaceMember
	for rows.Next() {
		var m domain.WorkspaceMember
		if err := rows.Scan(&m.WorkspaceID, &m.UserID, &m.Role, &m.CreatedAt); err != nil {
			return nil, err
		}
		members = append(members, &m)
	}
	return members, nil
}
