package installer

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllPlatforms_ReturnsAllRegistered(t *testing.T) {
	all := AllPlatforms()
	// Ensure AllPlatforms returns at least 33 platforms (6 original + 27 new)
	assert.GreaterOrEqual(t, len(all), 33, "expected at least 33 platforms")

	// Every platform in AllPlatforms should be valid
	for _, p := range all {
		assert.True(t, p.IsValid(), "platform %q should be valid", p)
	}

	// Every platform in the registry should be in AllPlatforms
	for p := range platformRegistry {
		assert.True(t, slices.Contains(all, p), "registry platform %q should be in AllPlatforms()", p)
	}
}

func TestPlatform_IsValid(t *testing.T) {
	tests := []struct {
		platform Platform
		valid    bool
	}{
		{PlatformClaude, true},
		{PlatformCursor, true},
		{PlatformCopilot, true},
		{PlatformCodex, true},
		{PlatformOpenCode, true},
		{PlatformWindsurf, true},
		// New platforms
		{PlatformCline, true},
		{PlatformRooCode, true},
		{PlatformAmp, true},
		{PlatformKimiCLI, true},
		{PlatformGeminiCLI, true},
		{PlatformGoose, true},
		{PlatformPochi, true},
		{PlatformNeovate, true},
		// Invalid
		{Platform("nonexistent"), false},
		{Platform(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.platform), func(t *testing.T) {
			assert.Equal(t, tt.valid, tt.platform.IsValid())
		})
	}
}

func TestPlatform_Info_HasRequiredFields(t *testing.T) {
	for p, info := range platformRegistry {
		t.Run(string(p), func(t *testing.T) {
			assert.NotEmpty(t, info.Name, "platform %q should have a Name", p)
			assert.NotEmpty(t, info.SkillsPath, "platform %q should have a SkillsPath", p)
			assert.NotEmpty(t, info.ProjectDir, "platform %q should have a ProjectDir", p)
			assert.NotEmpty(t, info.GlobalDir, "platform %q should have a GlobalDir", p)
			// Command can be empty (e.g., Copilot has no standalone CLI)
		})
	}
}

func TestPlatform_Info_DetectionFields(t *testing.T) {
	// Verify specific platforms have correct detection config
	tests := []struct {
		platform   Platform
		command    string
		projectDir string
		globalDir  string
	}{
		{PlatformClaude, "claude", ".claude", "~/.claude/skills/"},
		{PlatformCline, "cline", ".cline", "~/.cline/skills/"},
		{PlatformRooCode, "roo", ".roo", "~/.roo/skills/"},
		{PlatformAmp, "amp", ".agents", "~/.config/agents/skills/"},
		{PlatformKimiCLI, "kimi-cli", ".agents", "~/.config/agents/skills/"},
		{PlatformDroid, "droid", ".factory", "~/.factory/skills/"},
		{PlatformPi, "pi", ".pi", "~/.pi/agent/skills/"},
	}

	for _, tt := range tests {
		t.Run(string(tt.platform), func(t *testing.T) {
			info := tt.platform.Info()
			assert.Equal(t, tt.command, info.Command)
			assert.Equal(t, tt.projectDir, info.ProjectDir)
			assert.Equal(t, tt.globalDir, info.GlobalDir)
		})
	}
}

func TestPlatformFromString_AllAgents(t *testing.T) {
	tests := []struct {
		input    string
		expected Platform
	}{
		{"claude", PlatformClaude},
		{"cursor", PlatformCursor},
		{"cline", PlatformCline},
		{"roo", PlatformRooCode},
		{"amp", PlatformAmp},
		{"kimi-cli", PlatformKimiCLI},
		{"gemini-cli", PlatformGeminiCLI},
		{"nonexistent", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := PlatformFromString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPlatformFromStringOrAlias(t *testing.T) {
	// Direct match should work
	assert.Equal(t, PlatformClaude, PlatformFromStringOrAlias("claude"))
	assert.Equal(t, PlatformCline, PlatformFromStringOrAlias("cline"))

	// Unknown should return empty
	assert.Equal(t, Platform(""), PlatformFromStringOrAlias("nonexistent"))
}

func TestIsValidAlias(t *testing.T) {
	// Currently no aliases are defined in the registry entries.
	// This tests the mechanism works (returns false for non-aliases).
	assert.False(t, IsValidAlias("nonexistent"))
	assert.False(t, IsValidAlias(""))
}

func TestPlatformCursor_HasPlatformSpecificPaths(t *testing.T) {
	info := PlatformCursor.Info()
	require.NotEmpty(t, info.PlatformSpecificPaths)
	assert.Contains(t, info.PlatformSpecificPaths, "/Applications/Cursor.app")
}

func TestAllPlatforms_NoDuplicates(t *testing.T) {
	all := AllPlatforms()
	seen := make(map[Platform]bool)
	for _, p := range all {
		assert.False(t, seen[p], "duplicate platform in AllPlatforms: %q", p)
		seen[p] = true
	}
}
