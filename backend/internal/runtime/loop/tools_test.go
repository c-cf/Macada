package loop

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToolExecutor_ReadFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("line1\nline2\nline3"), 0o644)

	exec := NewToolExecutor(dir)
	result := exec.Execute(context.Background(), "read_file",
		toTestJSON(map[string]string{"file_path": "test.txt"}))

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "line2") {
		t.Errorf("content should contain line2, got: %s", result.Content)
	}
}

func TestToolExecutor_WriteFile(t *testing.T) {
	dir := t.TempDir()
	exec := NewToolExecutor(dir)

	result := exec.Execute(context.Background(), "write_file",
		toTestJSON(map[string]string{"file_path": "output.txt", "content": "hello world"}))

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}

	data, err := os.ReadFile(filepath.Join(dir, "output.txt"))
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("content = %q, want %q", data, "hello world")
	}
}

func TestToolExecutor_ListDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644)
	os.Mkdir(filepath.Join(dir, "subdir"), 0o755)

	exec := NewToolExecutor(dir)
	result := exec.Execute(context.Background(), "list_dir",
		toTestJSON(map[string]string{"path": "."}))

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if !strings.Contains(result.Content, "a.txt") {
		t.Error("should list a.txt")
	}
	if !strings.Contains(result.Content, "subdir/") {
		t.Error("should list subdir/ with trailing slash")
	}
}

func TestToolExecutor_UnknownTool(t *testing.T) {
	exec := NewToolExecutor(t.TempDir())
	result := exec.Execute(context.Background(), "nonexistent", nil)
	if !result.IsError {
		t.Error("should error for unknown tool")
	}
}

func TestToolExecutor_WriteFileCreatesSubdirs(t *testing.T) {
	dir := t.TempDir()
	exec := NewToolExecutor(dir)

	result := exec.Execute(context.Background(), "write_file",
		toTestJSON(map[string]string{"file_path": "deep/nested/file.txt", "content": "ok"}))

	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "deep/nested/file.txt"))
	if string(data) != "ok" {
		t.Errorf("content = %q", data)
	}
}

func toTestJSON(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
