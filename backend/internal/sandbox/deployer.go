package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	claudeDir  = ".claude"
	skillsDir  = ".claude/skills"
	claudeMD   = "CLAUDE.md"
)

// Deployer materializes DB entities into sandbox filesystem config files.
type Deployer struct{}

// NewDeployer creates a new Deployer.
func NewDeployer() *Deployer {
	return &Deployer{}
}

// Deploy writes all config files into the given base directory.
// Directory structure:
//
//	{basePath}/
//	├── CLAUDE.md
//	├── .claude/
//	│   ├── agent.json
//	│   ├── settings.json
//	│   ├── tools.json
//	│   ├── packages.json
//	│   ├── session.json
//	│   └── skills/
//	│       └── {name}/
//	│           ├── SKILL.md
//	│           └── scripts/*, references/*, ...
func (d *Deployer) Deploy(basePath string, manifest DeployManifest) error {
	// Create directory structure
	dirs := []string{
		filepath.Join(basePath, claudeDir),
		filepath.Join(basePath, skillsDir),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}

	// 1. CLAUDE.md — system prompt
	if err := writeFile(basePath, claudeMD, manifest.SystemPrompt); err != nil {
		return fmt.Errorf("write CLAUDE.md: %w", err)
	}

	// 2. agent.json — agent metadata + connection info
	if err := writeJSON(basePath, filepath.Join(claudeDir, "agent.json"), manifest.Agent); err != nil {
		return fmt.Errorf("write agent.json: %w", err)
	}

	// 3. settings.json — model + MCP servers
	settings := SettingsFile{
		Model:      manifest.Model,
		MCPServers: manifest.MCPServers,
	}
	if err := writeJSON(basePath, filepath.Join(claudeDir, "settings.json"), settings); err != nil {
		return fmt.Errorf("write settings.json: %w", err)
	}

	// 4. tools.json — tool definitions
	tools := manifest.Tools
	if len(tools) == 0 {
		tools = json.RawMessage("[]")
	}
	if err := writeFile(basePath, filepath.Join(claudeDir, "tools.json"), string(tools)); err != nil {
		return fmt.Errorf("write tools.json: %w", err)
	}

	// 5. packages.json — environment packages
	pkgs := PackagesFile{
		Apt:   manifest.Packages.Apt,
		Cargo: manifest.Packages.Cargo,
		Gem:   manifest.Packages.Gem,
		Go:    manifest.Packages.Go,
		Npm:   manifest.Packages.Npm,
		Pip:   manifest.Packages.Pip,
	}
	if err := writeJSON(basePath, filepath.Join(claudeDir, "packages.json"), pkgs); err != nil {
		return fmt.Errorf("write packages.json: %w", err)
	}

	// 6. session.json — session memory state
	sessionState := SessionStateFile{
		Memory: manifest.SessionMemory,
		Status: "idle",
	}
	if err := writeJSON(basePath, filepath.Join(claudeDir, "session.json"), sessionState); err != nil {
		return fmt.Errorf("write session.json: %w", err)
	}

	// 7. Skills — each skill as a directory with SKILL.md + files
	for _, skill := range manifest.Skills {
		if err := d.deploySkill(basePath, skill); err != nil {
			return fmt.Errorf("deploy skill %s: %w", skill.Name, err)
		}
	}

	return nil
}

// deploySkill writes a single skill to the skills directory.
func (d *Deployer) deploySkill(basePath string, skill SkillManifest) error {
	skillDir := filepath.Join(basePath, skillsDir, skill.Name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return fmt.Errorf("create skill dir: %w", err)
	}

	// Reconstruct SKILL.md with frontmatter
	skillMD := reconstructSkillMD(skill)
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0o644); err != nil {
		return fmt.Errorf("write SKILL.md: %w", err)
	}

	// Write additional files (scripts/, references/, assets/)
	for relPath, content := range skill.Files {
		fullPath := filepath.Join(skillDir, relPath)

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return fmt.Errorf("create dir for %s: %w", relPath, err)
		}

		perm := os.FileMode(0o644)
		if strings.HasPrefix(relPath, "scripts/") {
			perm = 0o755 // scripts are executable
		}

		if err := os.WriteFile(fullPath, []byte(content), perm); err != nil {
			return fmt.Errorf("write %s: %w", relPath, err)
		}
	}

	return nil
}

// reconstructSkillMD rebuilds a SKILL.md file with YAML frontmatter from DB fields.
func reconstructSkillMD(skill SkillManifest) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "name: %s\n", skill.Name)
	fmt.Fprintf(&sb, "description: %s\n", yamlQuote(skill.Description))
	if skill.License != "" {
		fmt.Fprintf(&sb, "license: %s\n", skill.License)
	}
	sb.WriteString("---\n\n")
	sb.WriteString(skill.Content)
	return sb.String()
}

// yamlQuote wraps a string in double quotes if it contains special characters.
func yamlQuote(s string) string {
	if strings.ContainsAny(s, ":#{}[]&*?|-<>=!%@`\"'\n") {
		escaped := strings.ReplaceAll(s, `"`, `\"`)
		return `"` + escaped + `"`
	}
	return s
}

func writeFile(basePath, relPath, content string) error {
	fullPath := filepath.Join(basePath, relPath)
	return os.WriteFile(fullPath, []byte(content), 0o644)
}

func writeJSON(basePath, relPath string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return writeFile(basePath, relPath, string(data))
}
