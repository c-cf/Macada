package domain

import (
	"context"
	"time"
)

// File represents an uploaded file in the workspace-scoped file system.
type File struct {
	ID           string    `json:"id"`
	WorkspaceID  string    `json:"workspace_id,omitempty"`
	Filename     string    `json:"filename"`
	MimeType     string    `json:"mime_type"`
	SizeBytes    int64     `json:"size_bytes"`
	StoragePath  string    `json:"-"` // internal-only, not exposed in API
	Downloadable bool      `json:"downloadable"`
	CreatedAt    time.Time `json:"created_at"`
	Type         string    `json:"type"`
}

// FileRepository defines file persistence operations.
type FileRepository interface {
	Create(ctx context.Context, file *File) error
	GetByID(ctx context.Context, id string) (*File, error)
	List(ctx context.Context, params FileListParams) ([]*File, *string, error)
	Delete(ctx context.Context, id string) error
}

// FileListParams are pagination/filter params for listing files.
type FileListParams struct {
	ListParams
}

// FileDeleteResponse is the response when a file is deleted.
type FileDeleteResponse struct {
	ID   string `json:"id"`
	Type string `json:"type"` // "file_deleted"
}
