package service

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// SkillFrontmatter holds the parsed YAML frontmatter from SKILL.md.
type SkillFrontmatter struct {
	Name          string            `yaml:"name"`
	Description   string            `yaml:"description"`
	License       string            `yaml:"license,omitempty"`
	Compatibility string            `yaml:"compatibility,omitempty"`
	AllowedTools  string            `yaml:"allowed-tools,omitempty"`
	Metadata      map[string]string `yaml:"metadata,omitempty"`
}

// ParsedSkill is the result of parsing a SKILL.md file.
type ParsedSkill struct {
	Frontmatter SkillFrontmatter
	Body        string // markdown content after frontmatter
}

// ParsedZip is the result of extracting a skill zip archive.
type ParsedZip struct {
	Skill ParsedSkill
	Files map[string]string // relative path -> content
}

var nameRegexp = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ParseSkillMD splits a SKILL.md into frontmatter and body.
func ParseSkillMD(raw string) (ParsedSkill, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "---") {
		return ParsedSkill{}, fmt.Errorf("SKILL.md must start with YAML frontmatter (---)")
	}

	rest := raw[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return ParsedSkill{}, fmt.Errorf("SKILL.md frontmatter is missing closing ---")
	}

	yamlBlock := rest[:idx]
	body := strings.TrimSpace(rest[idx+4:]) // skip \n---

	var fm SkillFrontmatter
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return ParsedSkill{}, fmt.Errorf("invalid YAML frontmatter: %w", err)
	}

	return ParsedSkill{Frontmatter: fm, Body: body}, nil
}

// ValidateFrontmatter checks that the frontmatter fields satisfy the Agent Skills specification.
func ValidateFrontmatter(fm SkillFrontmatter) error {
	// name: required, 1-64 chars, lowercase alphanumeric + hyphens
	if fm.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(fm.Name) > 64 {
		return fmt.Errorf("name must be at most 64 characters")
	}
	if !nameRegexp.MatchString(fm.Name) {
		return fmt.Errorf("name must contain only lowercase letters, numbers, and hyphens; must not start/end with a hyphen or contain consecutive hyphens")
	}

	// description: required, 1-1024 chars
	if fm.Description == "" {
		return fmt.Errorf("description is required")
	}
	if len(fm.Description) > 1024 {
		return fmt.Errorf("description must be at most 1024 characters")
	}

	// compatibility: optional, max 500 chars
	if len(fm.Compatibility) > 500 {
		return fmt.Errorf("compatibility must be at most 500 characters")
	}

	return nil
}

const maxZipSize = 10 * 1024 * 1024  // 10 MB
const maxFileSize = 1 * 1024 * 1024   // 1 MB per file

// ParseSkillZip extracts a zip archive and parses the SKILL.md inside.
// The zip must contain a SKILL.md either at root or inside a single top-level directory.
func ParseSkillZip(data []byte) (ParsedZip, error) {
	if len(data) > maxZipSize {
		return ParsedZip{}, fmt.Errorf("zip file exceeds maximum size of %d bytes", maxZipSize)
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return ParsedZip{}, fmt.Errorf("failed to read zip: %w", err)
	}

	files := map[string]string{}
	var skillMDContent string
	var skillMDPath string

	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if f.UncompressedSize64 > uint64(maxFileSize) {
			return ParsedZip{}, fmt.Errorf("file %s exceeds maximum size of %d bytes", f.Name, maxFileSize)
		}

		rc, err := f.Open()
		if err != nil {
			return ParsedZip{}, fmt.Errorf("failed to open %s: %w", f.Name, err)
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return ParsedZip{}, fmt.Errorf("failed to read %s: %w", f.Name, err)
		}

		// Normalize path: strip single top-level directory if present
		name := filepath.ToSlash(f.Name)
		files[name] = string(content)

		base := filepath.Base(name)
		if base == "SKILL.md" {
			skillMDContent = string(content)
			skillMDPath = name
		}
	}

	if skillMDContent == "" {
		return ParsedZip{}, fmt.Errorf("zip must contain a SKILL.md file")
	}

	parsed, err := ParseSkillMD(skillMDContent)
	if err != nil {
		return ParsedZip{}, fmt.Errorf("invalid SKILL.md: %w", err)
	}

	if err := ValidateFrontmatter(parsed.Frontmatter); err != nil {
		return ParsedZip{}, fmt.Errorf("invalid SKILL.md frontmatter: %w", err)
	}

	// Normalize file paths: strip the directory prefix that contains SKILL.md
	dir := filepath.Dir(skillMDPath)
	normalized := map[string]string{}
	for path, content := range files {
		rel := path
		if dir != "." {
			rel = strings.TrimPrefix(path, dir+"/")
		}
		// Skip SKILL.md itself from the files map
		if rel == "SKILL.md" {
			continue
		}
		normalized[rel] = content
	}

	return ParsedZip{
		Skill: parsed,
		Files: normalized,
	}, nil
}

// FilesToJSON converts a file map to json.RawMessage.
func FilesToJSON(files map[string]string) (json.RawMessage, error) {
	if len(files) == 0 {
		return json.RawMessage("{}"), nil
	}
	data, err := json.Marshal(files)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal files: %w", err)
	}
	return data, nil
}
