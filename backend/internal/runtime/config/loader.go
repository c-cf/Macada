package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	claudeMDFile   = "CLAUDE.md"
	agentFile      = ".claude/agent.json"
	settingsFile   = ".claude/settings.json"
	toolsFile      = ".claude/tools.json"
	packagesFile   = ".claude/packages.json"
	sessionFile    = ".claude/session.json"
	skillsDir      = ".claude/skills"
)

// Load reads all config files from the given base directory and returns a RuntimeConfig.
// Required files: agent.json. All others are optional with sensible defaults.
func Load(basePath string) (*RuntimeConfig, error) {
	cfg := &RuntimeConfig{}

	// 1. agent.json (required)
	if err := readJSON(basePath, agentFile, &cfg.Agent); err != nil {
		return nil, fmt.Errorf("load agent.json (required): %w", err)
	}
	if cfg.Agent.ID == "" || cfg.Agent.SessionID == "" {
		return nil, fmt.Errorf("agent.json must contain id and session_id")
	}
	if cfg.Agent.ControlPlaneURL == "" {
		return nil, fmt.Errorf("agent.json must contain control_plane_url")
	}
	if cfg.Agent.AnthropicAPIKey == "" {
		return nil, fmt.Errorf("agent.json must contain anthropic_api_key")
	}

	// 2. CLAUDE.md (optional)
	cfg.SystemPrompt = readFileOrEmpty(basePath, claudeMDFile)

	// 3. settings.json (optional)
	readJSON(basePath, settingsFile, &cfg.Settings)

	// 4. tools.json (optional)
	toolsData := readFileOrEmpty(basePath, toolsFile)
	if toolsData != "" {
		cfg.Tools = json.RawMessage(toolsData)
	} else {
		cfg.Tools = json.RawMessage("[]")
	}

	// 5. packages.json (optional)
	readJSON(basePath, packagesFile, &cfg.Packages)

	// 6. session.json (optional)
	readJSON(basePath, sessionFile, &cfg.Session)

	// 7. Discover skills
	skills, err := discoverSkills(filepath.Join(basePath, skillsDir))
	if err != nil {
		// Non-fatal: skills directory might not exist
		skills = nil
	}
	cfg.Skills = skills

	return cfg, nil
}

// discoverSkills walks the skills directory and loads each skill's SKILL.md.
func discoverSkills(skillsPath string) ([]LoadedSkill, error) {
	entries, err := os.ReadDir(skillsPath)
	if err != nil {
		return nil, err
	}

	var skills []LoadedSkill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillDir := filepath.Join(skillsPath, entry.Name())
		skillMDPath := filepath.Join(skillDir, "SKILL.md")

		content, err := os.ReadFile(skillMDPath)
		if err != nil {
			continue // skip skills without SKILL.md
		}

		name, description, body := parseFrontmatter(string(content))
		if name == "" {
			name = entry.Name() // fallback to directory name
		}

		skills = append(skills, LoadedSkill{
			Name:        name,
			Description: description,
			Content:     body,
			Dir:         skillDir,
		})
	}

	return skills, nil
}

// parseFrontmatter extracts name, description, and body from a SKILL.md.
func parseFrontmatter(raw string) (name, description, body string) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "---") {
		return "", "", raw
	}

	rest := raw[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return "", "", raw
	}

	fmBlock := rest[:idx]
	body = strings.TrimSpace(rest[idx+4:])

	// Simple YAML key extraction (no full parser needed here)
	for _, line := range strings.Split(fmBlock, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			name = strings.Trim(name, `"'`)
		}
		if strings.HasPrefix(line, "description:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			description = strings.Trim(description, `"'`)
		}
	}

	return name, description, body
}

func readJSON(basePath, relPath string, v interface{}) error {
	data, err := os.ReadFile(filepath.Join(basePath, relPath))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func readFileOrEmpty(basePath, relPath string) string {
	data, err := os.ReadFile(filepath.Join(basePath, relPath))
	if err != nil {
		return ""
	}
	return string(data)
}
