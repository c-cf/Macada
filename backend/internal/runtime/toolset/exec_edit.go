package toolset

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func executeEditFile(_ context.Context, workDir string, input json.RawMessage) ToolResult {
	var params struct {
		Path       string `json:"file_path"`
		OldString  string `json:"old_string"`
		NewString  string `json:"new_string"`
		ReplaceAll *bool  `json:"replace_all,omitempty"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return ToolResult{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}
	}
	if params.Path == "" {
		return ToolResult{Content: "file_path is required", IsError: true}
	}
	if params.OldString == params.NewString {
		return ToolResult{Content: "old_string and new_string must be different", IsError: true}
	}

	path, err := resolvePath(workDir, params.Path)
	if err != nil {
		return ToolResult{Content: err.Error(), IsError: true}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("read error: %v", err), IsError: true}
	}

	content := string(data)
	replaceAll := params.ReplaceAll != nil && *params.ReplaceAll

	count := strings.Count(content, params.OldString)
	if count == 0 {
		return ToolResult{
			Content: "old_string not found in file. Make sure the string matches exactly, including whitespace and indentation.",
			IsError: true,
		}
	}

	if !replaceAll && count > 1 {
		return ToolResult{
			Content: fmt.Sprintf("old_string appears %d times in the file. Provide more surrounding context to make it unique, or set replace_all to true.", count),
			IsError: true,
		}
	}

	var newContent string
	if replaceAll {
		newContent = strings.ReplaceAll(content, params.OldString, params.NewString)
	} else {
		newContent = strings.Replace(content, params.OldString, params.NewString, 1)
	}

	if len(newContent) > maxWriteBytes {
		return ToolResult{Content: fmt.Sprintf("resulting file too large: %d bytes exceeds %d byte limit", len(newContent), maxWriteBytes), IsError: true}
	}

	if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
		return ToolResult{Content: fmt.Sprintf("write error: %v", err), IsError: true}
	}

	if replaceAll {
		return ToolResult{
			Content: fmt.Sprintf("replaced %d occurrences in %s", count, params.Path),
			IsError: false,
		}
	}
	return ToolResult{
		Content: fmt.Sprintf("edited %s (1 replacement)", params.Path),
		IsError: false,
	}
}
