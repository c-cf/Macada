package toolset

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const maxDirEntries = 1000

func executeListDir(_ context.Context, workDir string, input json.RawMessage) ToolResult {
	var params struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return ToolResult{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}
	}

	var path string
	if params.Path == "" {
		path = workDir
	} else {
		var err error
		path, err = resolvePath(workDir, params.Path)
		if err != nil {
			return ToolResult{Content: err.Error(), IsError: true}
		}
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("readdir error: %v", err), IsError: true}
	}

	truncated := false
	if len(entries) > maxDirEntries {
		entries = entries[:maxDirEntries]
		truncated = true
	}

	var sb strings.Builder
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		sb.WriteString(name + "\n")
	}
	if truncated {
		sb.WriteString(fmt.Sprintf("\n[showing first %d entries]", maxDirEntries))
	}

	return ToolResult{Content: sb.String(), IsError: false}
}
