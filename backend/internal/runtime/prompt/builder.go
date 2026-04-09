package prompt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Builder composes system prompts with cache-friendly ordering.
type Builder struct {
	skillResolver *SkillResolver
}

// NewBuilder creates a new prompt Builder.
func NewBuilder(skillResolver *SkillResolver) *Builder {
	return &Builder{skillResolver: skillResolver}
}

// Build composes the system prompt as an array of SystemBlocks.
//
// Section ordering (static first, dynamic last):
//  1. Base instructions (Agent.System)          — static, cacheable
//  2. Skill instructions (resolved content)     — static, cacheable
//  3. Tool descriptions (Agent.Tools)           — static, cacheable
//  4. Session context (memory/summary)          — dynamic, not cached
//
// The cache_control marker is placed on the LAST static block.
// Anthropic caches everything up to and including that block.
func (b *Builder) Build(ctx context.Context, input PromptInput) ([]SystemBlock, error) {
	// Resolve skills if not already done
	skills := input.ResolvedSkills
	if skills == nil && len(input.Agent.Skills) > 0 && b.skillResolver != nil {
		var err error
		skills, err = b.skillResolver.Resolve(ctx, input.WorkspaceID, input.Agent.Skills)
		if err != nil {
			return nil, fmt.Errorf("resolve skills: %w", err)
		}
	}

	// Build static sections
	var staticBlocks []SystemBlock

	// Section 1: Base instructions
	if input.Agent.System != "" {
		staticBlocks = append(staticBlocks, SystemBlock{
			Type: "text",
			Text: input.Agent.System,
		})
	}

	// Section 2: Skill instructions
	if len(skills) > 0 {
		skillText := buildSkillSection(skills)
		staticBlocks = append(staticBlocks, SystemBlock{
			Type: "text",
			Text: skillText,
		})
	}

	// Section 3: Tool descriptions
	toolText := buildToolSection(input.Agent.Tools)
	if toolText != "" {
		staticBlocks = append(staticBlocks, SystemBlock{
			Type: "text",
			Text: toolText,
		})
	}

	// Mark the last static block with cache_control
	if len(staticBlocks) > 0 {
		staticBlocks[len(staticBlocks)-1].CacheControl = EphemeralCache
	}

	// Section 4: Session context (dynamic)
	var dynamicBlocks []SystemBlock

	if input.SessionMemory != nil && input.SessionMemory.Summary != "" {
		dynamicBlocks = append(dynamicBlocks, SystemBlock{
			Type: "text",
			Text: fmt.Sprintf("<session-memory>\n%s\n</session-memory>", input.SessionMemory.Summary),
		})
	}

	result := make([]SystemBlock, 0, len(staticBlocks)+len(dynamicBlocks))
	result = append(result, staticBlocks...)
	result = append(result, dynamicBlocks...)

	return result, nil
}

// buildSkillSection formats resolved skills into a single text block.
func buildSkillSection(skills []ResolvedSkill) string {
	var sb strings.Builder
	sb.WriteString("# Available Skills\n\n")
	for i, skill := range skills {
		if i > 0 {
			sb.WriteString("\n---\n\n")
		}
		fmt.Fprintf(&sb, "## %s\n\n%s", skill.Name, skill.Content)
	}
	return sb.String()
}

// buildToolSection formats the agent's tools JSON into a readable text block.
func buildToolSection(toolsRaw json.RawMessage) string {
	if len(toolsRaw) == 0 || string(toolsRaw) == "[]" || string(toolsRaw) == "null" {
		return ""
	}

	var tools []json.RawMessage
	if err := json.Unmarshal(toolsRaw, &tools); err != nil {
		return ""
	}
	if len(tools) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("# Available Tools\n\n")
	for _, tool := range tools {
		var t struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if json.Unmarshal(tool, &t) == nil && t.Name != "" {
			fmt.Fprintf(&sb, "- **%s**: %s\n", t.Name, t.Description)
		}
	}
	return sb.String()
}
