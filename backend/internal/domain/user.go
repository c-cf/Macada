package domain

import (
	"context"
	"time"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // never serialized
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Type         string    `json:"type"`
}

type WorkspaceMember struct {
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id"`
	Role        string    `json:"role"` // "owner" | "member"
	CreatedAt   time.Time `json:"created_at"`
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, user *User) error
}

type WorkspaceMemberRepository interface {
	Add(ctx context.Context, member *WorkspaceMember) error
	ListByUser(ctx context.Context, userID string) ([]*WorkspaceMember, error)
	ListByWorkspace(ctx context.Context, workspaceID string) ([]*WorkspaceMember, error)
}
