package loop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/c-cf/macada/internal/runtime/toolset"
)

const (
	toolTimeout     = 30 * time.Second
	maxOutputBytes  = 100 * 1024 // 100 KB
)

// ToolExecutor runs tools inside the sandbox (legacy fallback).
type ToolExecutor struct {
	workDir string
}

// NewToolExecutor creates a new executor rooted at the given working directory.
func NewToolExecutor(workDir string) *ToolExecutor {
	return &ToolExecutor{workDir: workDir}
}

// Execute runs the named tool with the given JSON input.
func (e *ToolExecutor) Execute(ctx context.Context, name string, input json.RawMessage) toolset.ToolResult {
	ctx, cancel := context.WithTimeout(ctx, toolTimeout)
	defer cancel()

	switch name {
	case "bash":
		return e.executeBash(ctx, input)
	case "read_file":
		return e.executeReadFile(input)
	case "write_file":
		return e.executeWriteFile(input)
	case "list_dir":
		return e.executeListDir(input)
	default:
		return toolset.ToolResult{Content: fmt.Sprintf("unknown tool: %s", name), IsError: true}
	}
}

func (e *ToolExecutor) executeBash(ctx context.Context, input json.RawMessage) toolset.ToolResult {
	var params struct {
		Command string `json:"command"`
		Timeout *int   `json:"timeout,omitempty"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return toolset.ToolResult{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}
	}
	if params.Command == "" {
		return toolset.ToolResult{Content: "command is required", IsError: true}
	}

	cmd := exec.CommandContext(ctx, "bash", "-c", params.Command)
	cmd.Dir = e.workDir

	output, err := cmd.CombinedOutput()
	result := truncateOutput(string(output))

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return toolset.ToolResult{Content: result + "\n[command timed out]", IsError: true}
		}
		return toolset.ToolResult{Content: result, IsError: true}
	}

	return toolset.ToolResult{Content: result, IsError: false}
}

func (e *ToolExecutor) executeReadFile(input json.RawMessage) toolset.ToolResult {
	var params struct {
		Path   string `json:"file_path"`
		Offset *int   `json:"offset,omitempty"`
		Limit  *int   `json:"limit,omitempty"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return toolset.ToolResult{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}
	}
	if params.Path == "" {
		return toolset.ToolResult{Content: "file_path is required", IsError: true}
	}

	path := e.resolvePath(params.Path)
	data, err := os.ReadFile(path)
	if err != nil {
		return toolset.ToolResult{Content: fmt.Sprintf("read error: %v", err), IsError: true}
	}

	lines := strings.Split(string(data), "\n")

	offset := 0
	if params.Offset != nil && *params.Offset > 0 {
		offset = *params.Offset
	}
	if offset > len(lines) {
		offset = len(lines)
	}

	limit := len(lines) - offset
	if params.Limit != nil && *params.Limit > 0 && *params.Limit < limit {
		limit = *params.Limit
	}

	selected := lines[offset : offset+limit]

	var sb strings.Builder
	for i, line := range selected {
		fmt.Fprintf(&sb, "%d\t%s\n", offset+i+1, line)
	}

	return toolset.ToolResult{Content: truncateOutput(sb.String()), IsError: false}
}

func (e *ToolExecutor) executeWriteFile(input json.RawMessage) toolset.ToolResult {
	var params struct {
		Path    string `json:"file_path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return toolset.ToolResult{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}
	}
	if params.Path == "" {
		return toolset.ToolResult{Content: "file_path is required", IsError: true}
	}

	path := e.resolvePath(params.Path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return toolset.ToolResult{Content: fmt.Sprintf("mkdir error: %v", err), IsError: true}
	}

	if err := os.WriteFile(path, []byte(params.Content), 0o644); err != nil {
		return toolset.ToolResult{Content: fmt.Sprintf("write error: %v", err), IsError: true}
	}

	return toolset.ToolResult{Content: fmt.Sprintf("wrote %d bytes to %s", len(params.Content), params.Path), IsError: false}
}

func (e *ToolExecutor) executeListDir(input json.RawMessage) toolset.ToolResult {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return toolset.ToolResult{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}
	}

	path := e.resolvePath(params.Path)
	if path == "" {
		path = e.workDir
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return toolset.ToolResult{Content: fmt.Sprintf("readdir error: %v", err), IsError: true}
	}

	var sb strings.Builder
	for _, entry := range entries {
		suffix := ""
		if entry.IsDir() {
			suffix = "/"
		}
		sb.WriteString(entry.Name() + suffix + "\n")
	}

	return toolset.ToolResult{Content: sb.String(), IsError: false}
}

func (e *ToolExecutor) resolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(e.workDir, p)
}

func truncateOutput(s string) string {
	if len(s) <= maxOutputBytes {
		return s
	}
	return s[:maxOutputBytes] + "\n[output truncated]"
}
