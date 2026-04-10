package toolset

import (
	"encoding/json"
	"testing"
)

func TestResolve(t *testing.T) {
	ts := Resolve("agent_toolset_20260401", "/workspace")
	if ts == nil {
		t.Fatal("expected toolset for agent_toolset_20260401")
	}

	if ts := Resolve("unknown_type", "/workspace"); ts != nil {
		t.Errorf("expected nil for unknown type, got %v", ts)
	}

	if ts := Resolve("", "/workspace"); ts != nil {
		t.Errorf("expected nil for empty type, got %v", ts)
	}
}

func TestToolsetDefinitions(t *testing.T) {
	ts := Resolve("agent_toolset_20260401", "/workspace")
	defs := ts.Definitions()

	var tools []ToolDef
	if err := json.Unmarshal(defs, &tools); err != nil {
		t.Fatalf("failed to unmarshal definitions: %v", err)
	}

	expected := map[string]bool{
		"bash": true, "read_file": true, "write_file": true,
		"edit_file": true, "list_dir": true, "grep": true, "glob": true,
	}

	if len(tools) != len(expected) {
		t.Errorf("expected %d tools, got %d", len(expected), len(tools))
	}

	for _, tool := range tools {
		if !expected[tool.Name] {
			t.Errorf("unexpected tool: %s", tool.Name)
		}
		if tool.Description == "" {
			t.Errorf("tool %s has empty description", tool.Name)
		}
		if len(tool.InputSchema) == 0 {
			t.Errorf("tool %s has empty input schema", tool.Name)
		}
	}
}

func TestCanExecute(t *testing.T) {
	ts := Resolve("agent_toolset_20260401", "/workspace")

	for _, name := range []string{"bash", "read_file", "write_file", "edit_file", "list_dir", "grep", "glob"} {
		if !ts.CanExecute(name) {
			t.Errorf("expected CanExecute(%q) = true", name)
		}
	}

	if ts.CanExecute("unknown_tool") {
		t.Error("expected CanExecute(unknown_tool) = false")
	}
}

func TestMergeDefinitions(t *testing.T) {
	ts := Resolve("agent_toolset_20260401", "/workspace")

	// Merge with empty user tools
	result := ts.MergeDefinitions(json.RawMessage("[]"))
	var tools []json.RawMessage
	_ = json.Unmarshal(result, &tools)
	if len(tools) != 7 {
		t.Errorf("expected 7 tools with empty user tools, got %d", len(tools))
	}

	// Merge with user tool that doesn't conflict
	userTools := json.RawMessage(`[{"name":"custom_tool","description":"a custom tool","input_schema":{"type":"object"}}]`)
	result = ts.MergeDefinitions(userTools)
	_ = json.Unmarshal(result, &tools)
	if len(tools) != 8 {
		t.Errorf("expected 8 tools with one custom tool, got %d", len(tools))
	}

	// Merge with user tool that conflicts (same name as built-in)
	conflicting := json.RawMessage(`[{"name":"bash","description":"overridden bash","input_schema":{"type":"object"}}]`)
	result = ts.MergeDefinitions(conflicting)
	_ = json.Unmarshal(result, &tools)
	if len(tools) != 7 {
		t.Errorf("expected 7 tools (conflict deduped), got %d", len(tools))
	}
}

func TestMergeDefinitionsNilToolset(t *testing.T) {
	var ts *Toolset
	userTools := json.RawMessage(`[{"name":"bash"}]`)
	result := ts.MergeDefinitions(userTools)
	if string(result) != string(userTools) {
		t.Errorf("nil toolset should return user tools as-is")
	}
}
