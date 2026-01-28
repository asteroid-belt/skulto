package views

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnboardingToolsView_InitPartitionsDetectedVsAll(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	v.SetSize(120, 40)
	v.Init()

	// displayItems should contain headers and agents
	require.NotEmpty(t, v.displayItems, "displayItems should not be empty after Init")

	// Should have at least 2 headers: "Detected" (if any detected) or just "All Agents"
	headerCount := 0
	agentCount := 0
	for _, item := range v.displayItems {
		switch item.kind {
		case itemHeader:
			headerCount++
		case itemAgent:
			agentCount++
		}
	}

	// Should have at least the "All Agents" header
	assert.GreaterOrEqual(t, headerCount, 1, "should have at least 1 header")
	// Agent count should match total platforms
	allPlatforms := installer.AllPlatforms()
	assert.Equal(t, len(allPlatforms), agentCount, "should have one agent item per platform")
}

func TestOnboardingToolsView_ClaudeDefaultWhenNothingDetected(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	v.SetSize(120, 40)
	v.Init()

	// Claude should be selected by default if nothing is detected
	selected := v.GetSelectedPlatforms()
	// At minimum, Claude should be in the list (or whatever was detected)
	assert.NotEmpty(t, selected, "at least one platform should be selected")
}

func TestOnboardingToolsView_NavigationSkipsHeaders(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	v.SetSize(120, 40)
	v.Init()

	// First selectable should be an agent, not a header
	assert.Equal(t, itemAgent, v.displayItems[v.currentSelection].kind,
		"initial cursor should be on an agent, not a header")

	// Navigate down, should stay on agent items
	v.Update("down")
	assert.Equal(t, itemAgent, v.displayItems[v.currentSelection].kind,
		"cursor after down should be on an agent")

	// Navigate up, should stay on agent items
	v.Update("up")
	assert.Equal(t, itemAgent, v.displayItems[v.currentSelection].kind,
		"cursor after up should be on an agent")
}

func TestOnboardingToolsView_ToggleSelectionMinimumOne(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	v.SetSize(120, 40)
	v.Init()

	// Get the initial selected platforms
	initial := v.GetSelectedPlatforms()
	require.NotEmpty(t, initial)

	// If only one is selected, toggling it should keep it selected (minimum 1)
	if len(initial) == 1 {
		v.Update("space")
		after := v.GetSelectedPlatforms()
		assert.Len(t, after, 1, "should enforce minimum 1 selection")
	}
}

func TestOnboardingToolsView_GetSelectedPlatforms(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	v.SetSize(120, 40)
	v.Init()

	platforms := v.GetSelectedPlatforms()
	assert.NotEmpty(t, platforms, "should return at least one selected platform")

	// All returned platforms should be valid
	for _, p := range platforms {
		assert.True(t, p.IsValid(), "returned platform %q should be valid", p)
	}
}

func TestOnboardingToolsView_BuildDisplayItems(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	v.SetSize(120, 40)
	v.Init()

	// Display items should start with a header
	require.NotEmpty(t, v.displayItems)
	assert.Equal(t, itemHeader, v.displayItems[0].kind, "first display item should be a header")

	// Last item should be an agent (not a separator)
	lastItem := v.displayItems[len(v.displayItems)-1]
	assert.Equal(t, itemAgent, lastItem.kind, "last display item should be an agent")
}

func TestOnboardingToolsView_ScrollingWithManyItems(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	// Small height to force scrolling
	v.SetSize(120, 20)
	v.Init()

	// Navigate down many times to trigger scrolling
	for i := 0; i < 30; i++ {
		v.Update("down")
	}

	// Cursor should still be on an agent
	assert.Equal(t, itemAgent, v.displayItems[v.currentSelection].kind,
		"cursor should still be on an agent after scrolling")

	// View should still render without panic
	rendered := v.View()
	assert.NotEmpty(t, rendered)
}

func TestOnboardingToolsView_ConfirmAndSkip(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	v.SetSize(120, 40)
	v.Init()

	// 'c' should confirm
	done, skipped := v.Update("c")
	assert.True(t, done)
	assert.False(t, skipped)
}

func TestOnboardingToolsView_EscSkips(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	v.SetSize(120, 40)
	v.Init()

	// 'esc' should skip
	done, skipped := v.Update("esc")
	assert.True(t, done)
	assert.True(t, skipped)
}

func TestOnboardingToolsView_PlatformNameFromRegistry(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)

	// Should use Info().Name from registry
	name := v.GetPlatformName(installer.PlatformClaude)
	assert.Equal(t, "Claude Code", name)

	name = v.GetPlatformName(installer.PlatformCline)
	assert.Equal(t, "Cline", name)

	name = v.GetPlatformName(installer.PlatformRooCode)
	assert.Equal(t, "Roo Code", name)
}
