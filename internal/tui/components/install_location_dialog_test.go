package components

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/detect"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergePreferencesWithDetected_Ordering(t *testing.T) {
	allPlatforms := []installer.Platform{
		installer.PlatformClaude,
		installer.PlatformCursor,
		installer.PlatformCline,
		installer.PlatformRooCode,
		installer.PlatformGoose,
	}

	saved := []string{"claude", "cline"}
	detected := []detect.DetectionResult{
		{Platform: installer.PlatformCursor, Detected: true},
		{Platform: installer.PlatformGoose, Detected: true},
	}

	result := mergePreferencesWithDetected(allPlatforms, saved, detected)
	require.Len(t, result, 5)

	// Saved first
	assert.Equal(t, installer.PlatformClaude, result[0])
	assert.Equal(t, installer.PlatformCline, result[1])
	// Detected (not in saved) next
	assert.Equal(t, installer.PlatformCursor, result[2])
	assert.Equal(t, installer.PlatformGoose, result[3])
	// Others last
	assert.Equal(t, installer.PlatformRooCode, result[4])
}

func TestMergePreferencesWithDetected_EmptySaved(t *testing.T) {
	allPlatforms := []installer.Platform{
		installer.PlatformClaude,
		installer.PlatformCursor,
	}

	detected := []detect.DetectionResult{
		{Platform: installer.PlatformClaude, Detected: true},
	}

	result := mergePreferencesWithDetected(allPlatforms, nil, detected)
	require.Len(t, result, 2)

	// Detected first since no saved
	assert.Equal(t, installer.PlatformClaude, result[0])
	assert.Equal(t, installer.PlatformCursor, result[1])
}

func TestMergePreferencesWithDetected_NoDetection(t *testing.T) {
	allPlatforms := []installer.Platform{
		installer.PlatformClaude,
		installer.PlatformCursor,
	}

	saved := []string{"cursor"}
	var detected []detect.DetectionResult

	result := mergePreferencesWithDetected(allPlatforms, saved, detected)
	require.Len(t, result, 2)

	// Saved first
	assert.Equal(t, installer.PlatformCursor, result[0])
	assert.Equal(t, installer.PlatformClaude, result[1])
}

func TestNewInstallLocationDialogWithPrefs_PreSelection(t *testing.T) {
	platforms := []installer.Platform{
		installer.PlatformClaude,
		installer.PlatformCline,
		installer.PlatformRooCode,
	}

	saved := map[string]string{"claude": "global", "cline": "global"}
	detected := []detect.DetectionResult{
		{Platform: installer.PlatformRooCode, Detected: true},
	}

	dialog := NewInstallLocationDialogWithPrefs(platforms, saved, detected)
	require.NotNil(t, dialog)

	// Should have options for all platforms (global + project each)
	locations := dialog.GetSelectedLocations()
	assert.NotEmpty(t, locations, "should have pre-selected locations")
}

func TestNewInstallLocationDialogWithPrefs_SavedAndDetectedPreSelected(t *testing.T) {
	platforms := []installer.Platform{
		installer.PlatformClaude,
		installer.PlatformCline,
	}

	saved := map[string]string{"claude": "global"}
	detected := []detect.DetectionResult{
		{Platform: installer.PlatformCline, Detected: true},
	}

	dialog := NewInstallLocationDialogWithPrefs(platforms, saved, detected)
	require.NotNil(t, dialog)

	// Both saved and detected platforms should have global selected
	selected := dialog.GetSelectedLocations()
	var selectedPlatforms []string
	for _, loc := range selected {
		selectedPlatforms = append(selectedPlatforms, string(loc.Platform))
	}
	assert.Contains(t, selectedPlatforms, "claude", "saved platform should be pre-selected")
	assert.Contains(t, selectedPlatforms, "cline", "detected platform should be pre-selected")
}

func TestNewInstallLocationDialogWithPrefs_ScopeAware(t *testing.T) {
	platforms := []installer.Platform{
		installer.PlatformClaude,
		installer.PlatformCursor,
	}

	// Claude saved with project scope, Cursor with global
	saved := map[string]string{"claude": "project", "cursor": "global"}
	var detected []detect.DetectionResult

	dialog := NewInstallLocationDialogWithPrefs(platforms, saved, detected)
	require.NotNil(t, dialog)

	selected := dialog.GetSelectedLocations()

	// Should have exactly 2 selections
	require.Len(t, selected, 2)

	// Build a map for easy checking
	selMap := make(map[string]string)
	for _, loc := range selected {
		selMap[string(loc.Platform)] = string(loc.Scope)
	}

	assert.Equal(t, "project", selMap["claude"], "claude should be pre-selected with project scope")
	assert.Equal(t, "global", selMap["cursor"], "cursor should be pre-selected with global scope")
}

func TestNewInstallLocationDialogWithPrefs_DefaultsToGlobal(t *testing.T) {
	platforms := []installer.Platform{
		installer.PlatformClaude,
	}

	// Saved with empty scope string defaults to global matching
	saved := map[string]string{"claude": "global"}
	var detected []detect.DetectionResult

	dialog := NewInstallLocationDialogWithPrefs(platforms, saved, detected)
	require.NotNil(t, dialog)

	selected := dialog.GetSelectedLocations()
	require.Len(t, selected, 1)
	assert.Equal(t, installer.ScopeGlobal, selected[0].Scope, "should default to global scope")
}

func TestInstallLocationDialog_CollapsibleGroups(t *testing.T) {
	platforms := []installer.Platform{
		installer.PlatformClaude,
		installer.PlatformCline,
		installer.PlatformRooCode,
	}

	saved := map[string]string{"claude": "global"}
	detected := []detect.DetectionResult{
		{Platform: installer.PlatformCline, Detected: true},
	}

	dialog := NewInstallLocationDialogWithPrefs(platforms, saved, detected)
	require.NotNil(t, dialog)

	// Group 2 should start collapsed
	assert.False(t, dialog.group2Expanded, "group 2 should start collapsed")

	// preferredCount should cover claude + cline options (global + project each = 4)
	assert.Equal(t, 4, dialog.preferredCount, "preferred count should be 4 (2 platforms x 2 scopes)")

	// Display items should have: header + 4 preferred options + separator + toggle + remember = 8
	assert.Len(t, dialog.displayItems, 8, "collapsed display should have 8 items")

	// Find and verify toggle header exists
	hasToggle := false
	for _, item := range dialog.displayItems {
		if item.kind == dkToggle {
			hasToggle = true
			break
		}
	}
	assert.True(t, hasToggle, "should have a toggle header for group 2")
}

func TestInstallLocationDialog_ToggleExpandsGroup2(t *testing.T) {
	platforms := []installer.Platform{
		installer.PlatformClaude,
		installer.PlatformCline,
		installer.PlatformRooCode,
	}

	saved := map[string]string{"claude": "global"}
	detected := []detect.DetectionResult{}

	dialog := NewInstallLocationDialogWithPrefs(platforms, saved, detected)
	require.NotNil(t, dialog)

	collapsedCount := len(dialog.displayItems)

	// Navigate to toggle header
	for i, item := range dialog.displayItems {
		if item.kind == dkToggle {
			dialog.currentIndex = i
			break
		}
	}

	// Toggle expand
	dialog.handleToggle()

	assert.True(t, dialog.group2Expanded, "group 2 should be expanded after toggle")
	assert.Greater(t, len(dialog.displayItems), collapsedCount, "expanding should add items")

	// Toggle collapse
	// Find toggle again (index may have changed)
	for i, item := range dialog.displayItems {
		if item.kind == dkToggle {
			dialog.currentIndex = i
			break
		}
	}
	dialog.handleToggle()

	assert.False(t, dialog.group2Expanded, "group 2 should be collapsed after second toggle")
	assert.Equal(t, collapsedCount, len(dialog.displayItems), "collapsing should remove items")
}

func TestInstallLocationDialog_NoGroupsWhenAllPreferred(t *testing.T) {
	// When NewInstallLocationDialog is used (no prefs), all are in group 1
	platforms := []installer.Platform{
		installer.PlatformClaude,
		installer.PlatformCursor,
	}

	dialog := NewInstallLocationDialog(platforms)
	require.NotNil(t, dialog)

	// Should have no toggle header (all in single group)
	hasToggle := false
	for _, item := range dialog.displayItems {
		if item.kind == dkToggle {
			hasToggle = true
			break
		}
	}
	assert.False(t, hasToggle, "should not have toggle header when all platforms are in group 1")
}

func TestInstallLocationDialog_NavigationWrapsAround(t *testing.T) {
	platforms := []installer.Platform{
		installer.PlatformClaude,
	}

	dialog := NewInstallLocationDialog(platforms)
	require.NotNil(t, dialog)

	// Navigate up from first item should wrap to last
	dialog.currentIndex = 0
	// Find first interactive item
	for i, item := range dialog.displayItems {
		if isInteractiveDialogItem(item.kind) {
			dialog.currentIndex = i
			break
		}
	}

	firstIdx := dialog.currentIndex
	dialog.moveCursor(-1)
	// Should have wrapped to the last interactive item (remember)
	assert.NotEqual(t, firstIdx, dialog.currentIndex, "should wrap around when moving up from first")
}

func TestInstallLocationDialog_ResetCollapsesGroup2(t *testing.T) {
	platforms := []installer.Platform{
		installer.PlatformClaude,
		installer.PlatformCline,
	}

	saved := map[string]string{"claude": "global"}
	dialog := NewInstallLocationDialogWithPrefs(platforms, saved, nil)

	// Expand group 2
	dialog.group2Expanded = true
	dialog.buildDisplayItems()
	expandedCount := len(dialog.displayItems)

	// Reset
	dialog.Reset()

	assert.False(t, dialog.group2Expanded, "reset should collapse group 2")
	assert.Less(t, len(dialog.displayItems), expandedCount, "reset should reduce display items")
}
