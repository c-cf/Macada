package toolset

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultBashTimeout = 30 * time.Second
	maxBashTimeout     = 10 * time.Minute
	maxOutputBytes     = 100 * 1024 // 100 KB
)

func executeBash(ctx context.Context, workDir string, input json.RawMessage) ToolResult {
	var params struct {
		Command     string `json:"command"`
		Description string `json:"description,omitempty"`
		Timeout     *int   `json:"timeout,omitempty"` // milliseconds
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return ToolResult{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}
	}
	if params.Command == "" {
		return ToolResult{Content: "command is required", IsError: true}
	}

	timeout := defaultBashTimeout
	if params.Timeout != nil && *params.Timeout > 0 {
		timeout = time.Duration(*params.Timeout) * time.Millisecond
		if timeout > maxBashTimeout {
			timeout = maxBashTimeout
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", params.Command)
	cmd.Dir = workDir

	output, err := cmd.CombinedOutput()
	result := truncateOutput(string(output))

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return ToolResult{Content: result + "\n[command timed out]", IsError: true}
		}
		return ToolResult{Content: result, IsError: true}
	}

	return ToolResult{Content: result, IsError: false}
}

// resolvePath resolves p relative to workDir and enforces that the result
// stays inside workDir (prevents path traversal attacks).
func resolvePath(workDir, p string) (string, error) {
	var abs string
	if filepath.IsAbs(p) {
		abs = filepath.Clean(p)
	} else {
		abs = filepath.Clean(filepath.Join(workDir, p))
	}
	clean := filepath.Clean(workDir)
	if abs != clean && !strings.HasPrefix(abs, clean+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes workspace %q", p, workDir)
	}
	return abs, nil
}

func truncateOutput(s string) string {
	if len(s) <= maxOutputBytes {
		return s
	}
	return s[:maxOutputBytes] + "\n[output truncated]"
}
