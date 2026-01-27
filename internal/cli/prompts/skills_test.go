package prompts

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestBuildSkillOptions(t *testing.T) {
	skills := []*models.Skill{
		{Slug: "docker-expert", Title: "Docker Expert", Description: "Docker help"},
		{Slug: "react-patterns", Title: "React Patterns", Description: "React best practices"},
	}

	options := BuildSkillOptions(skills)

	assert.Len(t, options, 2)
	assert.Equal(t, "docker-expert", options[0].Value)
	assert.Contains(t, options[0].Key, "Docker Expert")
	assert.Equal(t, "react-patterns", options[1].Value)
	assert.Contains(t, options[1].Key, "React Patterns")
}

func TestFilterSelectedSkills(t *testing.T) {
	skills := []*models.Skill{
		{Slug: "docker-expert", Title: "Docker Expert"},
		{Slug: "react-patterns", Title: "React Patterns"},
		{Slug: "go-idioms", Title: "Go Idioms"},
	}
	selected := []string{"docker-expert", "go-idioms"}

	result := FilterSelectedSkills(skills, selected)

	assert.Len(t, result, 2)
	assert.Equal(t, "docker-expert", result[0].Slug)
	assert.Equal(t, "go-idioms", result[1].Slug)
}
