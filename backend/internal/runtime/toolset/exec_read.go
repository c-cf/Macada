package toolset

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const defaultReadLimit = 2000

func executeReadFile(_ context.Context, workDir string, input json.RawMessage) ToolResult {
	var params struct {
		Path   string `json:"file_path"`
		Offset *int   `json:"offset,omitempty"`
		Limit  *int   `json:"limit,omitempty"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return ToolResult{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}
	}
	if params.Path == "" {
		return ToolResult{Content: "file_path is required", IsError: true}
	}

	path, err := resolvePath(workDir, params.Path)
	if err != nil {
		return ToolResult{Content: err.Error(), IsError: true}
	}

	info, err := os.Stat(path)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("read error: %v", err), IsError: true}
	}
	if info.IsDir() {
		return ToolResult{Content: fmt.Sprintf("%s is a directory, not a file. Use list_dir instead.", params.Path), IsError: true}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("read error: %v", err), IsError: true}
	}

	lines := strings.Split(string(data), "\n")
	totalLines := len(lines)

	offset := 0
	if params.Offset != nil && *params.Offset > 0 {
		offset = *params.Offset
	}
	if offset > totalLines {
		offset = totalLines
	}

	limit := defaultReadLimit
	if params.Limit != nil && *params.Limit > 0 {
		limit = *params.Limit
	}
	if offset+limit > totalLines {
		limit = totalLines - offset
	}

	selected := lines[offset : offset+limit]

	var sb strings.Builder
	for i, line := range selected {
		fmt.Fprintf(&sb, "%d\t%s\n", offset+i+1, line)
	}

	result := sb.String()

	if offset+limit < totalLines {
		result += fmt.Sprintf("\n[showing lines %d-%d of %d total]", offset+1, offset+limit, totalLines)
	}

	return ToolResult{Content: truncateOutput(result), IsError: false}
}
