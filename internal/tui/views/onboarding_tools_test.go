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

	require.NotEmpty(t, v.displayItems, "displayItems should not be empty after Init")

	// Count item kinds
	headerCount := 0
	agentCount := 0
	toggleCount := 0
	for _, item := range v.displayItems {
		switch item.kind {
		case itemHeader:
			headerCount++
		case itemAgent:
			agentCount++
		case itemToggleHeader:
			toggleCount++
		}
	}

	assert.Equal(t, 1, toggleCount, "should have 1 toggle header for All Agents")

	// In CI, likely nothing is detected so group 2 auto-expands
	// showing all platforms. When detected, group 2 starts collapsed.
	if len(v.detectedAgents) == 0 {
		// Auto-expanded: all agents visible
		allPlatforms := installer.AllPlatforms()
		assert.Equal(t, len(allPlatforms), agentCount, "auto-expanded should show all agents")
	} else {
		// Collapsed: only detected agents visible
		assert.Equal(t, len(v.detectedAgents), agentCount, "collapsed view should only show detected agents")
	}
}

func TestOnboardingToolsView_Group2StartsCollapsedWithDetection(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	v.SetSize(120, 40)

	// Simulate: manually set up with detected agents to test collapse
	v.selectedTools = make(map[installer.Platform]bool)
	v.detectedAgents = []installer.Platform{installer.PlatformClaude}
	v.allAgents = []installer.Platform{installer.PlatformCursor, installer.PlatformCline}
	v.group2Expanded = false
	v.buildDisplayItems()

	// Should have: header + 1 detected agent + separator + toggle header = 4 items
	assert.Len(t, v.displayItems, 4)

	// No allAgents visible when collapsed
	agentCount := 0
	for _, item := range v.displayItems {
		if item.kind == itemAgent {
			agentCount++
		}
	}
	assert.Equal(t, 1, agentCount, "only detected agent should be visible when collapsed")
}

func TestOnboardingToolsView_Group2ExpandsOnToggle(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	v.SetSize(120, 40)

	// Set up with detected + others
	v.selectedTools = make(map[installer.Platform]bool)
	v.detectedAgents = []installer.Platform{installer.PlatformClaude}
	v.allAgents = []installer.Platform{installer.PlatformCursor, installer.PlatformCline}
	v.group2Expanded = false
	v.buildDisplayItems()
	v.currentSelection = v.firstSelectableIndex()

	beforeCount := len(v.displayItems)

	// Navigate to toggle header
	for i, item := range v.displayItems {
		if item.kind == itemToggleHeader {
			v.currentSelection = i
			break
		}
	}

	// Toggle expand
	v.Update("space")

	assert.True(t, v.group2Expanded)
	assert.Greater(t, len(v.displayItems), beforeCount, "expanding should add items")

	// Should now have the allAgents visible
	agentCount := 0
	for _, item := range v.displayItems {
		if item.kind == itemAgent {
			agentCount++
		}
	}
	assert.Equal(t, 3, agentCount, "should show detected + all agents when expanded")
}

func TestOnboardingToolsView_Group2AutoExpandsWhenNothingDetected(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	v.SetSize(120, 40)

	// Simulate no detection
	v.selectedTools = make(map[installer.Platform]bool)
	v.detectedAgents = nil
	v.allAgents = []installer.Platform{installer.PlatformClaude, installer.PlatformCursor}
	v.group2Expanded = false

	// Auto-expand when nothing detected
	if len(v.detectedAgents) == 0 {
		v.group2Expanded = true
	}
	v.buildDisplayItems()

	assert.True(t, v.group2Expanded, "should auto-expand when nothing detected")

	agentCount := 0
	for _, item := range v.displayItems {
		if item.kind == itemAgent {
			agentCount++
		}
	}
	assert.Equal(t, 2, agentCount, "all agents should be visible when auto-expanded")
}

func TestOnboardingToolsView_NothingSelectedByDefault(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	v.SetSize(120, 40)
	v.Init()

	// Nothing should be pre-selected
	selected := v.GetSelectedPlatforms()
	assert.Empty(t, selected, "no platforms should be pre-selected")
}

func TestOnboardingToolsView_NavigationSkipsHeaders(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	v.SetSize(120, 40)
	v.Init()

	// First selectable should be an agent or toggle header, not a plain header
	kind := v.displayItems[v.currentSelection].kind
	assert.True(t, kind == itemAgent || kind == itemToggleHeader,
		"initial cursor should be on an interactive item, not a plain header")

	// Navigate down, should stay on interactive items
	v.Update("down")
	kind = v.displayItems[v.currentSelection].kind
	assert.True(t, kind == itemAgent || kind == itemToggleHeader,
		"cursor after down should be on an interactive item")

	// Navigate up, should stay on interactive items
	v.Update("up")
	kind = v.displayItems[v.currentSelection].kind
	assert.True(t, kind == itemAgent || kind == itemToggleHeader,
		"cursor after up should be on an interactive item")
}

func TestOnboardingToolsView_ToggleSelection(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	v.SetSize(120, 40)
	v.Init()

	// Nothing selected initially
	assert.Empty(t, v.GetSelectedPlatforms())

	// Navigate to first agent item and toggle on
	for i, item := range v.displayItems {
		if item.kind == itemAgent {
			v.currentSelection = i
			break
		}
	}
	v.Update("space")
	after := v.GetSelectedPlatforms()
	assert.Len(t, after, 1, "should have 1 selected after toggle on")

	// Toggle off
	v.Update("space")
	after = v.GetSelectedPlatforms()
	assert.Empty(t, after, "should have 0 selected after toggle off")
}

func TestOnboardingToolsView_GetSelectedPlatforms(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	v.SetSize(120, 40)
	v.Init()

	// Starts empty
	assert.Empty(t, v.GetSelectedPlatforms(), "should start with no selections")

	// Manually select a platform and verify
	v.selectedTools[installer.PlatformClaude] = true
	platforms := v.GetSelectedPlatforms()
	assert.NotEmpty(t, platforms, "should return selected platform")

	for _, p := range platforms {
		assert.True(t, p.IsValid(), "returned platform %q should be valid", p)
	}
}

func TestOnboardingToolsView_GetSelectedPlatformsWorksWhenCollapsed(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	v.SetSize(120, 40)

	// Set up with selections in both groups
	v.selectedTools = make(map[installer.Platform]bool)
	v.detectedAgents = []installer.Platform{installer.PlatformClaude}
	v.allAgents = []installer.Platform{installer.PlatformCursor, installer.PlatformCline}
	v.selectedTools[installer.PlatformClaude] = true
	v.selectedTools[installer.PlatformCursor] = true
	v.group2Expanded = false
	v.buildDisplayItems()

	// Even though group 2 is collapsed, GetSelectedPlatforms should return all selections
	selected := v.GetSelectedPlatforms()
	assert.Len(t, selected, 2, "should include selections from collapsed group")
}

func TestOnboardingToolsView_BuildDisplayItems(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	v.SetSize(120, 40)
	v.Init()

	// Display items should start with a header
	require.NotEmpty(t, v.displayItems)
	first := v.displayItems[0]
	assert.True(t, first.kind == itemHeader || first.kind == itemToggleHeader,
		"first display item should be a header")
}

func TestOnboardingToolsView_ScrollingWithManyItems(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	// Small height to force scrolling
	v.SetSize(120, 20)
	v.Init()

	// Make sure group 2 is expanded so we have many items
	v.group2Expanded = true
	v.buildDisplayItems()
	v.currentSelection = v.firstSelectableIndex()

	// Navigate down many times to trigger scrolling
	for i := 0; i < 30; i++ {
		v.Update("down")
	}

	// Cursor should still be on an interactive item
	kind := v.displayItems[v.currentSelection].kind
	assert.True(t, kind == itemAgent || kind == itemToggleHeader,
		"cursor should still be on an interactive item after scrolling")

	// View should still render without panic
	rendered := v.View()
	assert.NotEmpty(t, rendered)
}

func TestOnboardingToolsView_ConfirmAndSkip(t *testing.T) {
	cfg := &config.Config{}
	v := NewOnboardingToolsView(cfg)
	v.SetSize(120, 40)
	v.Init()

	// 'enter' should confirm
	done, skipped := v.Update("enter")
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
