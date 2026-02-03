package views

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHomeView_DiscoveryBadge_ShowsCountWhenPositive tests that the discovery badge
// appears in navigation when there are undismissed discovered skills.
func TestHomeView_DiscoveryBadge_ShowsCountWhenPositive(t *testing.T) {
	// Create a home view without database (we'll manually set the count)
	hv := &HomeView{
		width:  120,
		height: 40,
	}

	// Set a positive discovery count
	hv.SetDiscoveryCount(3)

	// Verify the count is set
	assert.Equal(t, int64(3), hv.GetDiscoveryCount())

	// Render the welcome section which contains the navigation
	rendered := hv.renderWelcome()

	// The badge should appear as "(3)" next to manage
	assert.Contains(t, rendered, "(3)")
	assert.Contains(t, rendered, "manage")
}

// TestHomeView_DiscoveryBadge_HiddenWhenZero tests that the discovery badge
// does not appear when there are no undismissed discovered skills.
func TestHomeView_DiscoveryBadge_HiddenWhenZero(t *testing.T) {
	// Create a home view without database
	hv := &HomeView{
		width:  120,
		height: 40,
	}

	// Set zero discovery count (default)
	hv.SetDiscoveryCount(0)

	// Verify the count is zero
	assert.Equal(t, int64(0), hv.GetDiscoveryCount())

	// Render the welcome section
	rendered := hv.renderWelcome()

	// The badge "(N)" should NOT appear - just plain "m (manage)"
	// The output should contain "m (manage)" without any badge
	assert.Contains(t, rendered, "m (manage)")
	// Count how many times "(manage" appears - should not have a count
	assert.NotContains(t, rendered, "(manage (")
}

// TestHomeView_DiscoveryBadge_DefaultIsZero tests that the default discovery count is zero.
func TestHomeView_DiscoveryBadge_DefaultIsZero(t *testing.T) {
	hv := &HomeView{
		width:  120,
		height: 40,
	}

	// Default should be zero
	assert.Equal(t, int64(0), hv.GetDiscoveryCount())

	// Render and verify no badge
	rendered := hv.renderWelcome()
	assert.Contains(t, rendered, "m (manage)")
	// Ensure there's no number badge
	assert.NotContains(t, rendered, "(manage (")
}

// TestHomeView_DiscoveryBadge_UpdatesWhenCountChanges tests that the badge updates
// when the discovery count changes.
func TestHomeView_DiscoveryBadge_UpdatesWhenCountChanges(t *testing.T) {
	hv := &HomeView{
		width:  120,
		height: 40,
	}

	// Start with no discoveries
	hv.SetDiscoveryCount(0)
	rendered := hv.renderWelcome()
	assert.NotContains(t, rendered, "(manage (")

	// Add discoveries
	hv.SetDiscoveryCount(5)
	rendered = hv.renderWelcome()
	assert.Contains(t, rendered, "(5)")

	// Remove discoveries
	hv.SetDiscoveryCount(0)
	rendered = hv.renderWelcome()
	assert.NotContains(t, rendered, "(5)")
}

// TestHomeView_DiscoveryBadge_LargeNumbers tests that large discovery counts display correctly.
func TestHomeView_DiscoveryBadge_LargeNumbers(t *testing.T) {
	hv := &HomeView{
		width:  120,
		height: 40,
	}

	// Test with a larger number
	hv.SetDiscoveryCount(99)
	rendered := hv.renderWelcome()
	assert.Contains(t, rendered, "(99)")

	// Test with even larger number
	hv.SetDiscoveryCount(123)
	rendered = hv.renderWelcome()
	assert.Contains(t, rendered, "(123)")
}

// TestHomeView_SetDiscoveryCount_SetsValue tests the SetDiscoveryCount method.
func TestHomeView_SetDiscoveryCount_SetsValue(t *testing.T) {
	hv := &HomeView{}

	testCases := []int64{0, 1, 5, 10, 100, 1000}

	for _, tc := range testCases {
		hv.SetDiscoveryCount(tc)
		require.Equal(t, tc, hv.discoveryCount, "SetDiscoveryCount(%d) should set discoveryCount to %d", tc, tc)
		require.Equal(t, tc, hv.GetDiscoveryCount(), "GetDiscoveryCount() should return %d after SetDiscoveryCount(%d)", tc, tc)
	}
}

// TestHomeView_RenderWelcome_ContainsManage tests that renderWelcome always contains manage navigation.
func TestHomeView_RenderWelcome_ContainsManage(t *testing.T) {
	hv := &HomeView{
		width:  120,
		height: 40,
	}

	// Test with and without badge
	for _, count := range []int64{0, 1, 10} {
		hv.SetDiscoveryCount(count)
		rendered := hv.renderWelcome()

		// Should always contain "manage" text
		assert.True(t, strings.Contains(rendered, "manage"),
			"renderWelcome() should contain 'manage' when discoveryCount is %d", count)

		// Should always contain "m (" prefix for the key binding
		assert.True(t, strings.Contains(rendered, "m ("),
			"renderWelcome() should contain 'm (' key binding when discoveryCount is %d", count)
	}
}
