package toolset

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	grepTimeout    = 30 * time.Second
	defaultHeadLim = 250
)

// executeGrep runs ripgrep (rg) with the given parameters.
// Falls back to grep -rn if rg is not available.
func executeGrep(ctx context.Context, workDir string, input json.RawMessage) ToolResult {
	var params struct {
		Pattern         string `json:"pattern"`
		Path            string `json:"path,omitempty"`
		Glob            string `json:"glob,omitempty"`
		OutputMode      string `json:"output_mode,omitempty"`
		CaseInsensitive *bool  `json:"case_insensitive,omitempty"`
		ContextLines    *int   `json:"context_lines,omitempty"`
		HeadLimit       *int   `json:"head_limit,omitempty"`
	}
	if err := json.Unmarshal(input, &params); err != nil {
		return ToolResult{Content: fmt.Sprintf("invalid input: %v", err), IsError: true}
	}
	if params.Pattern == "" {
		return ToolResult{Content: "pattern is required", IsError: true}
	}

	searchPath := workDir
	if params.Path != "" {
		resolved, err := resolvePath(workDir, params.Path)
		if err != nil {
			return ToolResult{Content: err.Error(), IsError: true}
		}
		searchPath = resolved
	}

	outputMode := "files_with_matches"
	if params.OutputMode != "" {
		outputMode = params.OutputMode
	}

	headLimit := defaultHeadLim
	if params.HeadLimit != nil && *params.HeadLimit >= 0 {
		headLimit = *params.HeadLimit
	}

	caseInsensitive := params.CaseInsensitive != nil && *params.CaseInsensitive

	if rgPath, err := exec.LookPath("rg"); err == nil {
		return runRipgrep(ctx, rgPath, searchPath, params.Pattern, outputMode, params.Glob,
			caseInsensitive, params.ContextLines, headLimit)
	}
	return runGrepFallback(ctx, searchPath, params.Pattern, outputMode,
		caseInsensitive, params.ContextLines, headLimit)
}

func runRipgrep(ctx context.Context, rgPath, searchPath, pattern, outputMode, glob string,
	caseInsensitive bool, contextLines *int, headLimit int) ToolResult {

	ctx, cancel := context.WithTimeout(ctx, grepTimeout)
	defer cancel()

	args := []string{"--no-heading", "--color=never"}

	switch outputMode {
	case "files_with_matches":
		args = append(args, "--files-with-matches")
	case "count":
		args = append(args, "--count")
	default: // "content"
		args = append(args, "--line-number")
	}

	if caseInsensitive {
		args = append(args, "--ignore-case")
	}

	if contextLines != nil && *contextLines > 0 && outputMode == "content" {
		args = append(args, "--context", strconv.Itoa(*contextLines))
	}

	if glob != "" {
		args = append(args, "--glob", glob)
	}

	// Exclude common noisy directories
	args = append(args,
		"--glob", "!.git/",
		"--glob", "!node_modules/",
		"--glob", "!vendor/",
		"--glob", "!__pycache__/",
	)

	args = append(args, pattern, searchPath)

	cmd := exec.CommandContext(ctx, rgPath, args...)
	cmd.Dir = searchPath

	output, err := cmd.CombinedOutput()
	result := string(output)

	// Apply head limit by truncating lines (rg has no --max-total-count)
	result = applyHeadLimit(result, headLimit)
	result = truncateOutput(result)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return ToolResult{Content: "[search timed out]", IsError: true}
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return ToolResult{Content: "no matches found", IsError: false}
		}
		return ToolResult{Content: result, IsError: true}
	}

	if strings.TrimSpace(result) == "" {
		return ToolResult{Content: "no matches found", IsError: false}
	}

	return ToolResult{Content: result, IsError: false}
}

func runGrepFallback(ctx context.Context, searchPath, pattern, outputMode string,
	caseInsensitive bool, contextLines *int, headLimit int) ToolResult {

	ctx, cancel := context.WithTimeout(ctx, grepTimeout)
	defer cancel()

	args := []string{"-rn", "--color=never"}

	switch outputMode {
	case "files_with_matches":
		args = []string{"-rl", "--color=never"}
	case "count":
		args = []string{"-rc", "--color=never"}
	}

	if caseInsensitive {
		args = append(args, "-i")
	}

	if contextLines != nil && *contextLines > 0 && outputMode == "content" {
		args = append(args, fmt.Sprintf("-C%d", *contextLines))
	}

	// Exclude noisy directories
	args = append(args,
		"--exclude-dir=.git",
		"--exclude-dir=node_modules",
		"--exclude-dir=vendor",
		"--exclude-dir=__pycache__",
	)

	args = append(args, pattern, searchPath)

	cmd := exec.CommandContext(ctx, "grep", args...)

	output, err := cmd.CombinedOutput()
	result := string(output)

	result = applyHeadLimit(result, headLimit)
	result = truncateOutput(result)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return ToolResult{Content: "[search timed out]", IsError: true}
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return ToolResult{Content: "no matches found", IsError: false}
		}
		return ToolResult{Content: result, IsError: true}
	}

	if strings.TrimSpace(result) == "" {
		return ToolResult{Content: "no matches found", IsError: false}
	}

	return ToolResult{Content: result, IsError: false}
}

// applyHeadLimit truncates output to the first N lines.
func applyHeadLimit(s string, limit int) string {
	if limit <= 0 {
		return s
	}
	lines := strings.SplitN(s, "\n", limit+1)
	if len(lines) <= limit {
		return s
	}
	return strings.Join(lines[:limit], "\n") + "\n[results truncated]"
}
