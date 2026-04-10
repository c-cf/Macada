package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalStorage stores files on the local filesystem.
type LocalStorage struct {
	basePath string
}

// NewLocalStorage creates a LocalStorage and ensures the base directory exists.
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, fmt.Errorf("create storage dir %s: %w", basePath, err)
	}
	return &LocalStorage{basePath: basePath}, nil
}

// Store writes file content and returns the relative storage path and size.
// Layout: {basePath}/{workspaceID}/{fileID}
func (s *LocalStorage) Store(workspaceID, fileID string, reader io.Reader) (string, int64, error) {
	dir := filepath.Join(s.basePath, workspaceID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", 0, fmt.Errorf("create workspace dir: %w", err)
	}

	relPath := filepath.Join(workspaceID, fileID)
	fullPath := filepath.Join(s.basePath, relPath)

	f, err := os.Create(fullPath)
	if err != nil {
		return "", 0, fmt.Errorf("create file: %w", err)
	}
	defer func() { _ = f.Close() }()

	n, err := io.Copy(f, reader)
	if err != nil {
		_ = os.Remove(fullPath)
		return "", 0, fmt.Errorf("write file: %w", err)
	}

	return relPath, n, nil
}

// Read opens a stored file for reading.
func (s *LocalStorage) Read(storagePath string) (io.ReadCloser, error) {
	fullPath := s.FullPath(storagePath)
	f, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("open file %s: %w", storagePath, err)
	}
	return f, nil
}

// ReadAll reads the full content of a stored file into memory.
func (s *LocalStorage) ReadAll(storagePath string) ([]byte, error) {
	fullPath := s.FullPath(storagePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", storagePath, err)
	}
	return data, nil
}

// Delete removes a file from storage.
func (s *LocalStorage) Delete(storagePath string) error {
	fullPath := s.FullPath(storagePath)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file %s: %w", storagePath, err)
	}
	return nil
}

// FullPath returns the absolute filesystem path for a relative storage path.
func (s *LocalStorage) FullPath(storagePath string) string {
	// Prevent path traversal
	clean := filepath.Clean(storagePath)
	clean = strings.ReplaceAll(clean, "..", "")
	return filepath.Join(s.basePath, clean)
}
