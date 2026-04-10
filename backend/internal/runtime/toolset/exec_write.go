package toolset

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const maxWriteBytes = 10 * 1024 * 1024 // 10 MB

func executeWriteFile(_ context.Context, workDir string, input json.RawMessage) ToolResult {
	var params struct {
		Path    string `json:"file_path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return ToolResult{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}
	}
	if params.Path == "" {
		return ToolResult{Content: "file_path is required", IsError: true}
	}
	if len(params.Content) > maxWriteBytes {
		return ToolResult{Content: fmt.Sprintf("content too large: %d bytes exceeds %d byte limit", len(params.Content), maxWriteBytes), IsError: true}
	}

	path, err := resolvePath(workDir, params.Path)
	if err != nil {
		return ToolResult{Content: err.Error(), IsError: true}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ToolResult{Content: fmt.Sprintf("mkdir error: %v", err), IsError: true}
	}

	if err := os.WriteFile(path, []byte(params.Content), 0o644); err != nil {
		return ToolResult{Content: fmt.Sprintf("write error: %v", err), IsError: true}
	}

	return ToolResult{Content: fmt.Sprintf("wrote %d bytes to %s", len(params.Content), params.Path), IsError: false}
}
