package domain

import (
	"context"
	"encoding/json"
	"time"
)

// SessionResource represents a resource (file or github repo) mounted into a session container.
type SessionResource struct {
	ID        string          `json:"id"`
	SessionID string          `json:"session_id,omitempty"`
	Type      string          `json:"type"`                // "file" or "github_repository"
	FileID    *string         `json:"file_id,omitempty"`   // set when type is "file"
	MountPath string          `json:"mount_path"`
	Config    json.RawMessage `json:"config,omitempty"`    // extensible per-type data
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// ResourceRepository defines session resource persistence operations.
type ResourceRepository interface {
	Create(ctx context.Context, resource *SessionResource) error
	GetByID(ctx context.Context, id string) (*SessionResource, error)
	ListBySession(ctx context.Context, sessionID string, params ListParams) ([]*SessionResource, *string, error)
	Update(ctx context.Context, resource *SessionResource) error
	Delete(ctx context.Context, id string) error

	// ListFileResourcesBySession returns all file-type resources for a session (no pagination).
	// Used by the orchestrator to determine which files to mount.
	ListFileResourcesBySession(ctx context.Context, sessionID string) ([]*SessionResource, error)
}

// DeleteResourceResponse is the response when a resource is deleted.
type DeleteResourceResponse struct {
	ID   string `json:"id"`
	Type string `json:"type"` // "session_resource_deleted"
}
