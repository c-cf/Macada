package toolset

import "encoding/json"

// newV20260401 creates the agent_toolset_20260401 toolset.
func newV20260401(workDir string, uploader FileUploader) *Toolset {
	ts := &Toolset{
		workDir:  workDir,
		uploader: uploader,
		executors: map[string]ExecutorFunc{
			"bash":       executeBash,
			"read_file":  executeReadFile,
			"write_file": executeWriteFile,
			"edit_file":  executeEditFile,
			"list_dir":   executeListDir,
			"grep":       executeGrep,
			"glob":       executeGlob,
		},
	}

	if uploader != nil {
		ts.executors["upload_file"] = newUploadExecutor(uploader)
	}

	ts.tools = []ToolDef{
		{
			Type:        "custom",
			Name:        "bash",
			Description: "Execute a bash command in the sandbox. The working directory is /workspace. Use this for system commands, git operations, running tests, installing packages, etc.",
			InputSchema: mustJSON(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "The bash command to execute.",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "A short description of what this command does (for logging).",
					},
					"timeout": map[string]interface{}{
						"type":        "integer",
						"description": "Optional timeout in milliseconds (max 600000). Default: 30000.",
					},
				},
				"required": []string{"command"},
			}),
		},
		{
			Type:        "custom",
			Name:        "read_file",
			Description: "Read a file from the filesystem. Returns content with line numbers. Supports offset/limit for large files.",
			InputSchema: mustJSON(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "The path to the file to read (absolute or relative to /workspace).",
					},
					"offset": map[string]interface{}{
						"type":        "integer",
						"description": "Line number to start reading from (0-based). Default: 0.",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of lines to read. Default: 2000.",
					},
				},
				"required": []string{"file_path"},
			}),
		},
		{
			Type:        "custom",
			Name:        "write_file",
			Description: "Write content to a file. Creates parent directories if needed. Overwrites existing files.",
			InputSchema: mustJSON(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "The path to write to (absolute or relative to /workspace).",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The content to write to the file.",
					},
				},
				"required": []string{"file_path", "content"},
			}),
		},
		{
			Type:        "custom",
			Name:        "edit_file",
			Description: "Edit a file by replacing an exact string match. The old_string must appear exactly once in the file (unless replace_all is true). Use this instead of rewriting entire files.",
			InputSchema: mustJSON(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "The path to the file to edit.",
					},
					"old_string": map[string]interface{}{
						"type":        "string",
						"description": "The exact string to find and replace. Must be unique in the file unless replace_all is true.",
					},
					"new_string": map[string]interface{}{
						"type":        "string",
						"description": "The replacement string.",
					},
					"replace_all": map[string]interface{}{
						"type":        "boolean",
						"description": "If true, replace all occurrences. Default: false.",
					},
				},
				"required": []string{"file_path", "old_string", "new_string"},
			}),
		},
		{
			Type:        "custom",
			Name:        "list_dir",
			Description: "List the contents of a directory. Returns entries with '/' suffix for directories.",
			InputSchema: mustJSON(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "The directory path (absolute or relative to /workspace). Default: current working directory.",
					},
				},
				"required": []string{},
			}),
		},
		{
			Type:        "custom",
			Name:        "grep",
			Description: "Search file contents using regex patterns (powered by ripgrep). Supports context lines, pagination, and multiple output modes.",
			InputSchema: mustJSON(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "The regex pattern to search for.",
					},
					"path": map[string]interface{}{
						"type":        "string",
						"description": "File or directory to search in. Default: /workspace.",
					},
					"glob": map[string]interface{}{
						"type":        "string",
						"description": "Glob pattern to filter files (e.g. \"*.go\", \"*.{ts,tsx}\").",
					},
					"output_mode": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"content", "files_with_matches", "count"},
						"description": "Output mode: 'content' shows matching lines, 'files_with_matches' shows file paths (default), 'count' shows match counts.",
					},
					"case_insensitive": map[string]interface{}{
						"type":        "boolean",
						"description": "Case insensitive search. Default: false.",
					},
					"context_lines": map[string]interface{}{
						"type":        "integer",
						"description": "Number of context lines to show before and after each match (only for 'content' mode).",
					},
					"head_limit": map[string]interface{}{
						"type":        "integer",
						"description": "Limit output to first N results. Default: 250.",
					},
				},
				"required": []string{"pattern"},
			}),
		},
		{
			Type:        "custom",
			Name:        "glob",
			Description: "Find files by glob pattern. Returns matching file paths sorted by modification time. Use this to discover files in the codebase.",
			InputSchema: mustJSON(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "The glob pattern to match (e.g. \"**/*.go\", \"src/**/*.ts\").",
					},
					"path": map[string]interface{}{
						"type":        "string",
						"description": "The directory to search in. Default: /workspace.",
					},
				},
				"required": []string{"pattern"},
			}),
		},
	}

	if uploader != nil {
		ts.tools = append(ts.tools, ToolDef{
			Type:        "custom",
			Name:        "upload_file",
			Description: "Upload a file from the sandbox to the workspace so the user can download it. Use this for files the user explicitly requested (e.g. generated charts, reports, exported data, build artifacts). Do NOT upload source code, logs, or intermediate files unless the user asks.",
			InputSchema: mustJSON(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to upload (absolute or relative to /workspace).",
					},
					"filename": map[string]interface{}{
						"type":        "string",
						"description": "Optional display name for the uploaded file. Defaults to the file's basename.",
					},
				},
				"required": []string{"file_path"},
			}),
		})
	}

	return ts
}

func mustJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
