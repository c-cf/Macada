package service

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestParseSkillMD_Valid(t *testing.T) {
	raw := `---
name: pdf-processing
description: Extract PDF text, fill forms, merge files. Use when handling PDFs.
license: Apache-2.0
metadata:
  author: example-org
  version: "1.0"
---

## Instructions

Process PDFs using the scripts below.
`

	parsed, err := ParseSkillMD(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.Frontmatter.Name != "pdf-processing" {
		t.Errorf("name = %q, want %q", parsed.Frontmatter.Name, "pdf-processing")
	}
	if parsed.Frontmatter.License != "Apache-2.0" {
		t.Errorf("license = %q, want %q", parsed.Frontmatter.License, "Apache-2.0")
	}
	if parsed.Frontmatter.Metadata["author"] != "example-org" {
		t.Errorf("metadata.author = %q, want %q", parsed.Frontmatter.Metadata["author"], "example-org")
	}
	if parsed.Body == "" {
		t.Error("body should not be empty")
	}
}

func TestParseSkillMD_MissingFrontmatter(t *testing.T) {
	raw := `# Just markdown, no frontmatter`
	_, err := ParseSkillMD(raw)
	if err == nil {
		t.Fatal("expected error for missing frontmatter")
	}
}

func TestParseSkillMD_UnclosedFrontmatter(t *testing.T) {
	raw := `---
name: broken
description: missing closing delimiter
`
	_, err := ParseSkillMD(raw)
	if err == nil {
		t.Fatal("expected error for unclosed frontmatter")
	}
}

func TestValidateFrontmatter_Valid(t *testing.T) {
	fm := SkillFrontmatter{
		Name:        "code-review",
		Description: "Review code for quality.",
	}
	if err := ValidateFrontmatter(fm); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFrontmatter_InvalidNames(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"", "name is required"},
		{"PDF-Processing", "lowercase"},
		{"-pdf", "must not start"},
		{"pdf-", "must not start"},
		{"pdf--processing", "consecutive"},
		{"a" + string(make([]byte, 64)), "at most 64"},
	}

	for _, tc := range cases {
		fm := SkillFrontmatter{Name: tc.name, Description: "valid desc"}
		err := ValidateFrontmatter(fm)
		if err == nil {
			t.Errorf("expected error for name %q", tc.name)
		}
	}
}

func TestValidateFrontmatter_MissingDescription(t *testing.T) {
	fm := SkillFrontmatter{Name: "valid-name", Description: ""}
	err := ValidateFrontmatter(fm)
	if err == nil {
		t.Fatal("expected error for missing description")
	}
}

func TestParseSkillZip_Valid(t *testing.T) {
	buf := createTestZip(t, map[string]string{
		"my-skill/SKILL.md":               "---\nname: my-skill\ndescription: A test skill.\n---\n\n## Usage\nDo things.",
		"my-skill/scripts/run.sh":         "#!/bin/bash\necho hello",
		"my-skill/references/REFERENCE.md": "# Reference\nDetails here.",
	})

	result, err := ParseSkillZip(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Skill.Frontmatter.Name != "my-skill" {
		t.Errorf("name = %q, want %q", result.Skill.Frontmatter.Name, "my-skill")
	}
	if _, ok := result.Files["scripts/run.sh"]; !ok {
		t.Error("expected scripts/run.sh in files")
	}
	if _, ok := result.Files["references/REFERENCE.md"]; !ok {
		t.Error("expected references/REFERENCE.md in files")
	}
	// SKILL.md should be excluded from files
	if _, ok := result.Files["SKILL.md"]; ok {
		t.Error("SKILL.md should not be in files map")
	}
}

func TestParseSkillZip_NoSkillMD(t *testing.T) {
	buf := createTestZip(t, map[string]string{
		"readme.txt": "just a readme",
	})

	_, err := ParseSkillZip(buf)
	if err == nil {
		t.Fatal("expected error for missing SKILL.md")
	}
}

func TestParseSkillZip_RootSkillMD(t *testing.T) {
	buf := createTestZip(t, map[string]string{
		"SKILL.md": "---\nname: root-skill\ndescription: Skill at root.\n---\n\nContent.",
	})

	result, err := ParseSkillZip(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Skill.Frontmatter.Name != "root-skill" {
		t.Errorf("name = %q, want %q", result.Skill.Frontmatter.Name, "root-skill")
	}
}

// createTestZip builds an in-memory zip with the given file contents.
func createTestZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("failed to create zip entry %s: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write zip entry %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close zip: %v", err)
	}
	return buf.Bytes()
}
