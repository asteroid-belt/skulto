package prompts

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/stretchr/testify/assert"
)

func TestBuildPlatformOptions(t *testing.T) {
	platforms := []installer.DetectedPlatform{
		{ID: "claude", Name: "Claude Code", Path: "~/.claude/skills/", Detected: true},
		{ID: "cursor", Name: "Cursor", Path: "~/.cursor/skills/", Detected: false},
	}

	options := BuildPlatformOptions(platforms)

	assert.Len(t, options, 2)
	assert.Equal(t, "claude", options[0].Value)
	assert.Contains(t, options[0].Key, "Claude Code")
	assert.Contains(t, options[0].Key, "detected") // detected platforms marked
	assert.Equal(t, "cursor", options[1].Value)
	assert.NotContains(t, options[1].Key, "detected")
}

func TestFilterSelectedPlatforms(t *testing.T) {
	allPlatforms := []installer.DetectedPlatform{
		{ID: "claude", Name: "Claude Code"},
		{ID: "cursor", Name: "Cursor"},
		{ID: "windsurf", Name: "Windsurf"},
	}
	selected := []string{"claude", "windsurf"}

	result := FilterSelectedPlatforms(allPlatforms, selected)

	assert.Len(t, result, 2)
	assert.Equal(t, "claude", result[0].ID)
	assert.Equal(t, "windsurf", result[1].ID)
}

func TestGetDefaultSelectedPlatforms(t *testing.T) {
	platforms := []installer.DetectedPlatform{
		{ID: "claude", Name: "Claude Code", Detected: true},
		{ID: "cursor", Name: "Cursor", Detected: false},
		{ID: "windsurf", Name: "Windsurf", Detected: true},
	}

	defaults := GetDefaultSelectedPlatforms(platforms)

	assert.Len(t, defaults, 2)
	assert.Contains(t, defaults, "claude")
	assert.Contains(t, defaults, "windsurf")
	assert.NotContains(t, defaults, "cursor")
}
