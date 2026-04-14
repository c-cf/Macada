package toolset

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const maxUploadBytes = 100 * 1024 * 1024 // 100 MB

// newUploadExecutor returns an ExecutorFunc that captures the given uploader.
func newUploadExecutor(uploader FileUploader) ExecutorFunc {
	return func(ctx context.Context, workDir string, input json.RawMessage) ToolResult {
		var params struct {
			FilePath string `json:"file_path"`
			Filename string `json:"filename"`
		}
		if err := json.Unmarshal(input, &params); err != nil {
			return ToolResult{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}
		}
		if params.FilePath == "" {
			return ToolResult{Content: "file_path is required", IsError: true}
		}

		absPath, err := resolvePath(workDir, params.FilePath)
		if err != nil {
			return ToolResult{Content: err.Error(), IsError: true}
		}

		info, err := os.Stat(absPath)
		if err != nil {
			return ToolResult{Content: fmt.Sprintf("file not found: %v", err), IsError: true}
		}
		if info.IsDir() {
			return ToolResult{Content: "cannot upload a directory", IsError: true}
		}
		if info.Size() > maxUploadBytes {
			return ToolResult{Content: fmt.Sprintf("file too large: %d bytes exceeds %d byte limit", info.Size(), maxUploadBytes), IsError: true}
		}

		filename := params.Filename
		if filename == "" {
			filename = filepath.Base(absPath)
		}

		if err := uploader.UploadFile(ctx, absPath, filename); err != nil {
			return ToolResult{Content: fmt.Sprintf("upload failed: %v", err), IsError: true}
		}

		return ToolResult{
			Content: fmt.Sprintf("uploaded %s (%d bytes)", filename, info.Size()),
			IsError: false,
		}
	}
}
