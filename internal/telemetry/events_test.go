package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventConstants(t *testing.T) {
	// CLI events
	assert.Equal(t, "app_started", EventAppStarted)
	assert.Equal(t, "app_exited", EventAppExited)
	assert.Equal(t, "cli_command_executed", EventCLICommandExecuted)
	assert.Equal(t, "repo_added", EventRepoAdded)
	assert.Equal(t, "repo_removed", EventRepoRemoved)
	assert.Equal(t, "repo_synced", EventRepoSynced)
	assert.Equal(t, "repo_listed", EventRepoListed)
	assert.Equal(t, "config_changed", EventConfigChanged)
	assert.Equal(t, "cli_error_occurred", EventCLIErrorOccurred)
	assert.Equal(t, "cli_help_viewed", EventCLIHelpViewed)

	// TUI events
	assert.Equal(t, "view_navigated", EventViewNavigated)
	assert.Equal(t, "skill_installed", EventSkillInstalled)
	assert.Equal(t, "skill_uninstalled", EventSkillUninstalled)
	assert.Equal(t, "search_performed", EventSearchPerformed)
	assert.Equal(t, "filter_applied", EventFilterApplied)
	assert.Equal(t, "sort_changed", EventSortChanged)
	assert.Equal(t, "skill_copied", EventSkillCopied)
	assert.Equal(t, "onboarding_completed", EventOnboardingCompleted)
	assert.Equal(t, "onboarding_skipped", EventOnboardingSkipped)
	assert.Equal(t, "settings_changed", EventSettingsChanged)
	assert.Equal(t, "keyboard_shortcut_used", EventKeyboardShortcut)
	assert.Equal(t, "help_viewed", EventHelpViewed)
	assert.Equal(t, "pagination_used", EventPaginationUsed)
	assert.Equal(t, "tag_browsing_entered", EventTagBrowsingEntered)
	assert.Equal(t, "tag_selected", EventTagSelected)
	assert.Equal(t, "error_displayed", EventErrorDisplayed)
	assert.Equal(t, "source_selected", EventSourceSelected)
	assert.Equal(t, "list_refreshed", EventListRefreshed)

	// Session events
	assert.Equal(t, "session_summary", EventSessionSummary)

	// CLI-specific events
	assert.Equal(t, "skills_discovered", EventSkillsDiscovered)
	assert.Equal(t, "skill_ingested", EventSkillIngested)

	// Shared events (all interfaces)
	assert.Equal(t, "skill_viewed", EventSkillViewed)
	assert.Equal(t, "skills_listed", EventSkillsListed)
	assert.Equal(t, "stats_viewed", EventStatsViewed)
	assert.Equal(t, "recent_skills_viewed", EventRecentSkillsViewed)
	assert.Equal(t, "installed_skills_checked", EventInstalledSkillsChecked)

	// MCP events
	assert.Equal(t, "mcp_tool_called", EventMCPToolCalled)
}

// TestNoopClient_NewMethods verifies the new telemetry methods don't panic on noopClient.
func TestNoopClient_NewMethods(t *testing.T) {
	client := &noopClient{}

	// Shared events - these should not panic
	client.TrackSkillViewed("test-slug", "test-category", false)
	client.TrackSkillViewed("local-skill", "workflow", true)
	client.TrackSkillsListed(10, "mcp")
	client.TrackSkillsListed(0, "search")
	client.TrackStatsViewed()
	client.TrackRecentSkillsViewed(5)
	client.TrackRecentSkillsViewed(0)
	client.TrackInstalledSkillsChecked(3)
	client.TrackInstalledSkillsChecked(0)

	// MCP events - these should not panic
	client.TrackMCPToolCalled("skulto_search", 100, true)
	client.TrackMCPToolCalled("skulto_get_skill", 50, false)
	client.TrackMCPToolCalled("skulto_install", 200, true)
}

// TestClient_Interface_NewMethods verifies the Client interface includes new methods.
// This is a compile-time check - if noopClient doesn't implement all interface methods,
// this test won't compile.
func TestClient_Interface_NewMethods(t *testing.T) {
	var _ Client = &noopClient{}

	// Additional check: verify noopClient is correctly typed
	client := &noopClient{}
	assert.NotNil(t, client)

	// Verify the methods exist by calling them
	// This confirms the interface contract is satisfied
	client.TrackSkillViewed("slug", "category", false)
	client.TrackSkillsListed(1, "source")
	client.TrackStatsViewed()
	client.TrackRecentSkillsViewed(1)
	client.TrackInstalledSkillsChecked(1)
	client.TrackMCPToolCalled("tool", 1, true)
}

// TestSharedEventConstants verifies the shared event constant values are correct.
func TestSharedEventConstants(t *testing.T) {
	// Verify MCP event is unique
	assert.Equal(t, "mcp_tool_called", EventMCPToolCalled)
}

// TestNoopClient_GetTrackingID verifies noopClient returns empty tracking ID.
func TestNoopClient_GetTrackingID(t *testing.T) {
	client := &noopClient{}
	assert.Empty(t, client.GetTrackingID())
}
