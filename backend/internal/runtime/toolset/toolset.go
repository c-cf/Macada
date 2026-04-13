package toolset

import (
	"context"
	"encoding/json"
)

// ToolResult is the output of a tool execution.
type ToolResult struct {
	Content string
	IsError bool
}

// Toolset provides tool definitions (for the LLM) and execution logic.
type Toolset struct {
	tools     []ToolDef
	executors map[string]ExecutorFunc
	workDir   string
}

// ToolDef is the Anthropic API tool schema sent to the model.
// Type must be "custom" for user-defined tools; the Anthropic API uses it as a discriminator.
type ToolDef struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// ExecutorFunc executes a tool and returns its result.
type ExecutorFunc func(ctx context.Context, workDir string, input json.RawMessage) ToolResult

// Resolve returns the Toolset for a given agent type, or nil if unknown.
func Resolve(agentType string, workDir string) *Toolset {
	switch agentType {
	case "agent_toolset_20260401":
		return newV20260401(workDir)
	default:
		return nil
	}
}

// Definitions returns the tool definitions as JSON for the Anthropic API.
func (ts *Toolset) Definitions() json.RawMessage {
	data, _ := json.Marshal(ts.tools)
	return data
}

// CanExecute reports whether this toolset handles the named tool.
func (ts *Toolset) CanExecute(name string) bool {
	_, ok := ts.executors[name]
	return ok
}

// Execute runs the named tool. Caller must check CanExecute first.
func (ts *Toolset) Execute(ctx context.Context, name string, input json.RawMessage) ToolResult {
	fn, ok := ts.executors[name]
	if !ok {
		return ToolResult{Content: "unknown tool: " + name, IsError: true}
	}
	return fn(ctx, ts.workDir, input)
}

// MergeDefinitions merges toolset definitions with user-provided tools JSON.
// Toolset definitions come first; user tools are appended (no dedup).
func (ts *Toolset) MergeDefinitions(userTools json.RawMessage) json.RawMessage {
	if ts == nil {
		return userTools
	}

	tsDefs := ts.Definitions()

	// Parse both arrays
	var tsArr []json.RawMessage
	var userArr []json.RawMessage

	_ = json.Unmarshal(tsDefs, &tsArr)
	if len(userTools) > 0 && string(userTools) != "[]" && string(userTools) != "null" {
		_ = json.Unmarshal(userTools, &userArr)
	}

	// Build name set from toolset to avoid duplicates
	nameSet := make(map[string]struct{}, len(tsArr))
	for _, raw := range tsArr {
		var t struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(raw, &t)
		nameSet[t.Name] = struct{}{}
	}

	// Append user tools that don't conflict
	for _, raw := range userArr {
		var t struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(raw, &t)
		if _, exists := nameSet[t.Name]; !exists {
			tsArr = append(tsArr, raw)
		}
	}

	merged, _ := json.Marshal(tsArr)
	return merged
}
