package toolset

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxGlobResults = 200

// executeGlob finds files matching a glob pattern using filepath.Walk.
// Supports "**" for recursive directory matching via doublestar semantics.
func executeGlob(_ context.Context, workDir string, input json.RawMessage) ToolResult {
	var params struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path,omitempty"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return ToolResult{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}
	}
	if params.Pattern == "" {
		return ToolResult{Content: "pattern is required", IsError: true}
	}

	searchDir := workDir
	if params.Path != "" {
		resolved, err := resolvePath(workDir, params.Path)
		if err != nil {
			return ToolResult{Content: err.Error(), IsError: true}
		}
		searchDir = resolved
	}

	type fileEntry struct {
		path    string
		modTime int64
	}

	var matches []fileEntry

	err := filepath.Walk(searchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable paths
		}

		// Skip hidden and noisy directories
		name := info.Name()
		if info.IsDir() && (name == ".git" || name == "node_modules" || name == "vendor" || name == "__pycache__") {
			return filepath.SkipDir
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(searchDir, path)
		if err != nil {
			return nil
		}

		if matchDoublestar(params.Pattern, filepath.ToSlash(relPath)) {
			matches = append(matches, fileEntry{
				path:    relPath,
				modTime: info.ModTime().Unix(),
			})
		}

		return nil
	})
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("walk error: %v", err), IsError: true}
	}

	// Sort by modification time (newest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].modTime > matches[j].modTime
	})

	if len(matches) == 0 {
		return ToolResult{Content: "no files matched the pattern", IsError: false}
	}

	truncated := false
	if len(matches) > maxGlobResults {
		matches = matches[:maxGlobResults]
		truncated = true
	}

	var sb strings.Builder
	for _, m := range matches {
		sb.WriteString(m.path + "\n")
	}
	if truncated {
		fmt.Fprintf(&sb, "\n[showing first %d results]", maxGlobResults)
	}

	return ToolResult{Content: sb.String(), IsError: false}
}

// matchDoublestar matches a slash-separated path against a pattern
// that may contain "**" (match zero or more path segments) and
// standard glob wildcards (*, ?).
//
// Both pattern and path must use "/" as separator.
func matchDoublestar(pattern, path string) bool {
	patParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")
	return matchParts(patParts, pathParts)
}

// matchParts recursively matches pattern segments against path segments.
func matchParts(patParts, pathParts []string) bool {
	for len(patParts) > 0 {
		seg := patParts[0]

		if seg == "**" {
			// "**" at the end matches everything
			if len(patParts) == 1 {
				return true
			}

			// Try matching the rest of the pattern against every suffix of the path
			rest := patParts[1:]
			for i := 0; i <= len(pathParts); i++ {
				if matchParts(rest, pathParts[i:]) {
					return true
				}
			}
			return false
		}

		// No more path segments but pattern still has non-** segments
		if len(pathParts) == 0 {
			return false
		}

		// Match current segment with filepath.Match
		matched, _ := filepath.Match(seg, pathParts[0])
		if !matched {
			return false
		}

		patParts = patParts[1:]
		pathParts = pathParts[1:]
	}

	// Pattern exhausted; path must also be exhausted
	return len(pathParts) == 0
}
