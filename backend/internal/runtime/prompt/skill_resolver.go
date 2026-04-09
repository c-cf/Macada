package prompt

import (
	"context"

	"github.com/c-cf/macada/internal/domain"
	"github.com/rs/zerolog/log"
)

// SkillResolver resolves skill names to their full content.
type SkillResolver struct {
	skillRepo domain.SkillRepository
}

// NewSkillResolver creates a new SkillResolver.
func NewSkillResolver(skillRepo domain.SkillRepository) *SkillResolver {
	return &SkillResolver{skillRepo: skillRepo}
}

// Resolve looks up each skill name and returns resolved skills.
// Missing skills are logged as warnings and skipped.
func (r *SkillResolver) Resolve(ctx context.Context, workspaceID string, skillNames []string) ([]ResolvedSkill, error) {
	if len(skillNames) == 0 {
		return nil, nil
	}

	resolved := make([]ResolvedSkill, 0, len(skillNames))
	for _, name := range skillNames {
		skill, err := r.skillRepo.GetByName(ctx, workspaceID, name)
		if err != nil {
			log.Warn().Str("skill", name).Msg("skill not found, skipping")
			continue
		}
		resolved = append(resolved, ResolvedSkill{
			Name:    skill.Name,
			Content: skill.Content,
		})
	}
	return resolved, nil
}
