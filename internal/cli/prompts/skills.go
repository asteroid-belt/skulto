package prompts

import (
	"fmt"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/charmbracelet/huh"
)

// BuildSkillOptions creates huh options from skills.
func BuildSkillOptions(skills []*models.Skill) []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(skills))
	for _, s := range skills {
		label := s.Title
		if s.Description != "" {
			// Truncate long descriptions
			desc := s.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			label = fmt.Sprintf("%s - %s", s.Title, desc)
		}
		options = append(options, huh.NewOption(label, s.Slug))
	}
	return options
}

// FilterSelectedSkills returns only skills matching selected slugs.
func FilterSelectedSkills(all []*models.Skill, selected []string) []*models.Skill {
	selectedMap := make(map[string]bool, len(selected))
	for _, s := range selected {
		selectedMap[s] = true
	}

	result := make([]*models.Skill, 0, len(selected))
	for _, s := range all {
		if selectedMap[s.Slug] {
			result = append(result, s)
		}
	}
	return result
}

// RunSkillSelector shows interactive skill selection.
// Returns selected skill slugs.
func RunSkillSelector(skills []*models.Skill, preselected []string) ([]string, error) {
	options := BuildSkillOptions(skills)

	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select skills to install").
				Description("Space to toggle, Enter to confirm").
				Options(options...).
				Value(&selected),
		),
	)

	// Set initial selection
	selected = preselected

	if err := form.Run(); err != nil {
		return nil, err
	}

	return selected, nil
}
