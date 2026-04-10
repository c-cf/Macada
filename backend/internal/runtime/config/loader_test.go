package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeTestFiles creates a minimal config tree for testing.
func writeTestFiles(t *testing.T, base string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		full := filepath.Join(base, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
}

func TestLoad_FullConfig(t *testing.T) {
	dir := t.TempDir()
	writeTestFiles(t, dir, map[string]string{
		"CLAUDE.md": "You are helpful.",
		".claude/agent.json": `{
			"id": "agent_01ABC",
			"version": 2,
			"name": "test",
			"session_id": "sesn_01XYZ",
			"control_plane_url": "http://backend:8080",
			"control_plane_token": "tok_123",
			"control_plane_token": "tok_secret"
		}`,
		".claude/settings.json": `{"model":{"id":"claude-sonnet-4-6"}}`,
		".claude/tools.json":    `[{"name":"bash","description":"Run commands"}]`,
		".claude/packages.json": `{"apt":["curl"]}`,
		".claude/session.json":  `{"memory":{"summary":"Previous context","turn_count":3},"status":"idle"}`,
		".claude/skills/code-review/SKILL.md": "---\nname: code-review\ndescription: Review code.\n---\n\nCheck for bugs.",
	})

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// agent
	if cfg.Agent.ID != "agent_01ABC" {
		t.Errorf("agent.id = %q", cfg.Agent.ID)
	}
	if cfg.Agent.SessionID != "sesn_01XYZ" {
		t.Errorf("agent.session_id = %q", cfg.Agent.SessionID)
	}

	// system prompt
	if cfg.SystemPrompt != "You are helpful." {
		t.Errorf("system_prompt = %q", cfg.SystemPrompt)
	}

	// settings
	if cfg.Settings.Model.ID != "claude-sonnet-4-6" {
		t.Errorf("model.id = %q", cfg.Settings.Model.ID)
	}

	// tools
	var tools []json.RawMessage
	_ = json.Unmarshal(cfg.Tools, &tools)
	if len(tools) != 1 {
		t.Errorf("tools count = %d, want 1", len(tools))
	}

	// packages
	if len(cfg.Packages.Apt) != 1 || cfg.Packages.Apt[0] != "curl" {
		t.Errorf("packages.apt = %v", cfg.Packages.Apt)
	}

	// session memory
	if cfg.Session.Memory == nil {
		t.Fatal("session memory is nil")
	}
	if cfg.Session.Memory.Summary != "Previous context" {
		t.Errorf("memory.summary = %q", cfg.Session.Memory.Summary)
	}

	// skills
	if len(cfg.Skills) != 1 {
		t.Fatalf("skills count = %d, want 1", len(cfg.Skills))
	}
	if cfg.Skills[0].Name != "code-review" {
		t.Errorf("skill.name = %q", cfg.Skills[0].Name)
	}
	if cfg.Skills[0].Content != "Check for bugs." {
		t.Errorf("skill.content = %q", cfg.Skills[0].Content)
	}
}

func TestLoad_MissingAgentJSON(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for missing agent.json")
	}
}

func TestLoad_AgentMissingRequiredFields(t *testing.T) {
	dir := t.TempDir()
	writeTestFiles(t, dir, map[string]string{
		".claude/agent.json": `{"id":"agent_01","session_id":"sesn_01"}`,
	})
	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for missing control_plane_url")
	}
}

func TestLoad_MinimalConfig(t *testing.T) {
	dir := t.TempDir()
	writeTestFiles(t, dir, map[string]string{
		".claude/agent.json": `{
			"id": "agent_01",
			"session_id": "sesn_01",
			"control_plane_url": "http://backend:8080",
			"control_plane_token": "tok_secret"
		}`,
	})

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.SystemPrompt != "" {
		t.Errorf("system_prompt should be empty, got %q", cfg.SystemPrompt)
	}
	if string(cfg.Tools) != "[]" {
		t.Errorf("tools should be [], got %s", cfg.Tools)
	}
	if len(cfg.Skills) != 0 {
		t.Errorf("skills should be empty, got %d", len(cfg.Skills))
	}
}

func TestLoad_SkillDiscovery(t *testing.T) {
	dir := t.TempDir()
	writeTestFiles(t, dir, map[string]string{
		".claude/agent.json": `{
			"id": "agent_01",
			"session_id": "sesn_01",
			"control_plane_url": "http://backend:8080",
			"control_plane_token": "tok_secret"
		}`,
		".claude/skills/skill-a/SKILL.md": "---\nname: skill-a\ndescription: First skill.\n---\n\nDo A.",
		".claude/skills/skill-b/SKILL.md": "---\nname: skill-b\ndescription: Second skill.\n---\n\nDo B.",
		".claude/skills/not-a-skill.txt":  "random file",
	})

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.Skills) != 2 {
		t.Fatalf("skills count = %d, want 2", len(cfg.Skills))
	}

	names := map[string]bool{}
	for _, s := range cfg.Skills {
		names[s.Name] = true
	}
	if !names["skill-a"] || !names["skill-b"] {
		t.Errorf("expected skill-a and skill-b, got %v", names)
	}
}
