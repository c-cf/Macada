package sandbox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c-cf/macada/internal/domain"
	svcctx "github.com/c-cf/macada/internal/runtime/context"
)

func TestDeploy_BasicAgent(t *testing.T) {
	dir := t.TempDir()
	d := NewDeployer()

	manifest := DeployManifest{
		Agent: AgentConfigFile{
			ID:               "agent_01ABC",
			Version:          2,
			Name:             "test-agent",
			SessionID:        "sesn_01XYZ",
			ControlPlaneURL:  "http://backend:8080",
			ControlPlaneToken: "tok_secret",
			AnthropicAPIKey:  "sk-ant-test",
		},
		SystemPrompt: "You are a helpful assistant.",
		Model:        domain.ModelConfig{ID: "claude-sonnet-4-6"},
		Tools:        json.RawMessage(`[{"name":"bash","description":"Run commands"}]`),
		MCPServers:   json.RawMessage(`{}`),
		Packages:     domain.Packages{Apt: []string{"curl", "git"}},
	}

	if err := d.Deploy(dir, manifest); err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Verify CLAUDE.md
	content := readTestFile(t, dir, "CLAUDE.md")
	if content != "You are a helpful assistant." {
		t.Errorf("CLAUDE.md = %q", content)
	}

	// Verify agent.json
	var agent AgentConfigFile
	readTestJSON(t, dir, ".claude/agent.json", &agent)
	if agent.ID != "agent_01ABC" {
		t.Errorf("agent.id = %q", agent.ID)
	}
	if agent.AnthropicAPIKey != "sk-ant-test" {
		t.Errorf("agent.anthropic_api_key = %q", agent.AnthropicAPIKey)
	}

	// Verify settings.json
	var settings SettingsFile
	readTestJSON(t, dir, ".claude/settings.json", &settings)
	if settings.Model.ID != "claude-sonnet-4-6" {
		t.Errorf("settings.model.id = %q", settings.Model.ID)
	}

	// Verify tools.json
	toolsContent := readTestFile(t, dir, ".claude/tools.json")
	if !strings.Contains(toolsContent, "bash") {
		t.Errorf("tools.json missing bash tool")
	}

	// Verify packages.json
	var pkgs PackagesFile
	readTestJSON(t, dir, ".claude/packages.json", &pkgs)
	if len(pkgs.Apt) != 2 || pkgs.Apt[0] != "curl" {
		t.Errorf("packages.apt = %v", pkgs.Apt)
	}
}

func TestDeploy_WithSkills(t *testing.T) {
	dir := t.TempDir()
	d := NewDeployer()

	manifest := DeployManifest{
		Agent: AgentConfigFile{
			ID:               "agent_01ABC",
			SessionID:        "sesn_01XYZ",
			ControlPlaneURL:  "http://backend:8080",
			AnthropicAPIKey:  "sk-ant-test",
		},
		Model: domain.ModelConfig{ID: "claude-sonnet-4-6"},
		Skills: []SkillManifest{
			{
				Name:        "code-review",
				Description: "Review code for quality.",
				Content:     "## Instructions\n\nCheck for bugs.",
				Files: map[string]string{
					"scripts/lint.sh":         "#!/bin/bash\neslint .",
					"references/REFERENCE.md": "# Linting rules\n...",
				},
			},
		},
	}

	if err := d.Deploy(dir, manifest); err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Verify SKILL.md
	skillMD := readTestFile(t, dir, ".claude/skills/code-review/SKILL.md")
	if !strings.Contains(skillMD, "name: code-review") {
		t.Error("SKILL.md missing name frontmatter")
	}
	if !strings.Contains(skillMD, "Check for bugs") {
		t.Error("SKILL.md missing body content")
	}

	// Verify script file
	script := readTestFile(t, dir, ".claude/skills/code-review/scripts/lint.sh")
	if !strings.Contains(script, "eslint") {
		t.Error("script missing content")
	}

	// Verify script is executable (skip on Windows — no Unix permission bits)
	if os.Getenv("OS") != "Windows_NT" {
		info, err := os.Stat(filepath.Join(dir, ".claude/skills/code-review/scripts/lint.sh"))
		if err != nil {
			t.Fatalf("stat script: %v", err)
		}
		if info.Mode()&0o111 == 0 {
			t.Error("script should be executable")
		}
	}

	// Verify reference file
	ref := readTestFile(t, dir, ".claude/skills/code-review/references/REFERENCE.md")
	if !strings.Contains(ref, "Linting rules") {
		t.Error("reference missing content")
	}
}

func TestDeploy_WithSessionMemory(t *testing.T) {
	dir := t.TempDir()
	d := NewDeployer()

	manifest := DeployManifest{
		Agent: AgentConfigFile{
			ID:              "agent_01",
			SessionID:       "sesn_01",
			ControlPlaneURL: "http://backend:8080",
			AnthropicAPIKey: "sk-ant-test",
		},
		Model: domain.ModelConfig{ID: "claude-sonnet-4-6"},
		SessionMemory: &svcctx.SessionMemory{
			Summary:   "User discussed Go patterns.",
			TurnCount: 5,
		},
	}

	if err := d.Deploy(dir, manifest); err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	var state SessionStateFile
	readTestJSON(t, dir, ".claude/session.json", &state)
	if state.Memory == nil {
		t.Fatal("session.json memory is nil")
	}
	if state.Memory.Summary != "User discussed Go patterns." {
		t.Errorf("memory.summary = %q", state.Memory.Summary)
	}
}

func TestDeploy_EmptySkillsAndTools(t *testing.T) {
	dir := t.TempDir()
	d := NewDeployer()

	manifest := DeployManifest{
		Agent: AgentConfigFile{
			ID:              "agent_01",
			SessionID:       "sesn_01",
			ControlPlaneURL: "http://backend:8080",
			AnthropicAPIKey: "sk-ant-test",
		},
		Model: domain.ModelConfig{ID: "claude-sonnet-4-6"},
	}

	if err := d.Deploy(dir, manifest); err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// tools.json should default to empty array
	tools := readTestFile(t, dir, ".claude/tools.json")
	if strings.TrimSpace(tools) != "[]" {
		t.Errorf("tools.json = %q, want []", tools)
	}

	// skills directory should exist but be empty
	entries, err := os.ReadDir(filepath.Join(dir, ".claude/skills"))
	if err != nil {
		t.Fatalf("read skills dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 skills, got %d", len(entries))
	}
}

// --- test helpers ---

func readTestFile(t *testing.T, base, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(base, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

func readTestJSON(t *testing.T, base, rel string, v interface{}) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(base, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("parse %s: %v", rel, err)
	}
}
