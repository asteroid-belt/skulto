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
	assert.Equal(t, "skill_info_viewed", EventSkillInfoViewed)
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
	assert.Equal(t, "skill_previewed", EventSkillPreviewed)
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
}
