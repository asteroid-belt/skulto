package telemetry

import (
	"runtime"
	"strings"

	"github.com/asteroid-belt/skulto/pkg/version"
)

// Event names - CLI
const (
	EventAppStarted         = "app_started"
	EventAppExited          = "app_exited"
	EventCLICommandExecuted = "cli_command_executed"
	EventRepoAdded          = "repo_added"
	EventRepoRemoved        = "repo_removed"
	EventRepoSynced         = "repo_synced"
	EventRepoListed         = "repo_listed"
	EventSkillInfoViewed    = "skill_info_viewed"
	EventConfigChanged      = "config_changed"
	EventCLIErrorOccurred   = "cli_error_occurred"
	EventCLIHelpViewed      = "cli_help_viewed"
	EventFavoriteAdded      = "favorite_added"
	EventFavoriteRemoved    = "favorite_removed"
	EventFavoritesListed    = "favorites_listed"
)

// Event names - TUI
const (
	EventViewNavigated          = "view_navigated"
	EventSkillInstalled         = "skill_installed"
	EventSkillUninstalled       = "skill_uninstalled"
	EventNewSkillCreatedSuccess = "new_skill_created_successfully"
	EventNewSkillCreatedFailure = "new_skill_created_failure"
	EventSearchPerformed        = "search_performed"
	EventFilterApplied          = "filter_applied"
	EventSortChanged            = "sort_changed"
	EventSkillPreviewed         = "skill_previewed"
	EventSkillCopied            = "skill_copied"
	EventOnboardingCompleted    = "onboarding_completed"
	EventOnboardingSkipped      = "onboarding_skipped"
	EventSettingsChanged        = "settings_changed"
	EventKeyboardShortcut       = "keyboard_shortcut_used"
	EventHelpViewed             = "help_viewed"
	EventPaginationUsed         = "pagination_used"
	EventTagBrowsingEntered     = "tag_browsing_entered"
	EventTagSelected            = "tag_selected"
	EventErrorDisplayed         = "error_displayed"
	EventSourceSelected         = "source_selected"
	EventListRefreshed          = "list_refreshed"
)

// Event names - Session
const (
	EventSessionSummary = "session_summary"
)

// Version is set at compile time via ldflags.
var Version string

// baseProperties returns common properties for all events.
func baseProperties() map[string]interface{} {
	return map[string]interface{}{
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"version":    Version,
		"prerelease": version.IsPrerelease(),
		"dev_build":  version.IsDevBuild(),
	}
}

// --- CLI Tracking Methods ---

// TrackAppStarted tracks application startup.
func (c *posthogClient) TrackAppStarted(mode string, hasSources bool, sourceCount int) {
	props := baseProperties()
	props["mode"] = mode
	props["has_sources"] = hasSources
	props["source_count"] = sourceCount
	c.Track(EventAppStarted, props)
}

// TrackAppExited tracks application exit.
func (c *posthogClient) TrackAppExited(mode string, sessionDurationMs int64, commandsRun int) {
	props := baseProperties()
	props["mode"] = mode
	props["session_duration_ms"] = sessionDurationMs
	props["commands_run"] = commandsRun
	c.Track(EventAppExited, props)
}

// TrackCLICommandExecuted tracks CLI command execution.
func (c *posthogClient) TrackCLICommandExecuted(commandName string, hasFlags bool, durationMs int64) {
	props := baseProperties()
	props["command_name"] = commandName
	props["has_flags"] = hasFlags
	props["execution_duration_ms"] = durationMs
	c.Track(EventCLICommandExecuted, props)
}

// TrackRepoAdded tracks when a repository is added.
func (c *posthogClient) TrackRepoAdded(sourceID string, skillCount int) {
	props := baseProperties()
	props["source_id"] = sourceID
	props["skill_count"] = skillCount
	c.Track(EventRepoAdded, props)
}

// TrackRepoRemoved tracks when a repository is removed.
func (c *posthogClient) TrackRepoRemoved(sourceID string, skillCount int) {
	props := baseProperties()
	props["source_id"] = sourceID
	props["skill_count"] = skillCount
	c.Track(EventRepoRemoved, props)
}

// TrackRepoSynced tracks repository sync operations.
func (c *posthogClient) TrackRepoSynced(sourceID string, added, removed, updated int) {
	props := baseProperties()
	props["source_id"] = sourceID
	props["skills_added"] = added
	props["skills_removed"] = removed
	props["skills_updated"] = updated
	c.Track(EventRepoSynced, props)
}

// TrackRepoListed tracks repo list command.
func (c *posthogClient) TrackRepoListed(sourceCount, totalSkillCount int) {
	props := baseProperties()
	props["source_count"] = sourceCount
	props["total_skill_count"] = totalSkillCount
	c.Track(EventRepoListed, props)
}

// TrackSkillInfoViewed tracks skill info viewing.
func (c *posthogClient) TrackSkillInfoViewed(category string, isLocal bool) {
	props := baseProperties()
	props["skill_category"] = category
	props["is_local"] = isLocal
	c.Track(EventSkillInfoViewed, props)
}

// TrackConfigChanged tracks config changes.
func (c *posthogClient) TrackConfigChanged(settingName string, isDefault bool) {
	props := baseProperties()
	props["setting_name"] = settingName
	props["is_default"] = isDefault
	c.Track(EventConfigChanged, props)
}

// TrackCLIError tracks CLI errors.
func (c *posthogClient) TrackCLIError(commandName, errorType string) {
	props := baseProperties()
	props["command_name"] = commandName
	props["error_type"] = errorType
	c.Track(EventCLIErrorOccurred, props)
}

// TrackCLIHelpViewed tracks help command usage.
func (c *posthogClient) TrackCLIHelpViewed(commandName string, cliArgs []string) {
	props := baseProperties()
	props["command_name"] = commandName
	props["cli_args"] = strings.Join(cliArgs, " ")
	c.Track(EventCLIHelpViewed, props)
}

// TrackFavoriteAdded tracks when a skill is added to favorites.
func (c *posthogClient) TrackFavoriteAdded(slug string) {
	props := baseProperties()
	props["skill_slug"] = slug
	c.Track(EventFavoriteAdded, props)
}

// TrackFavoriteRemoved tracks when a skill is removed from favorites.
func (c *posthogClient) TrackFavoriteRemoved(slug string) {
	props := baseProperties()
	props["skill_slug"] = slug
	c.Track(EventFavoriteRemoved, props)
}

// TrackFavoritesListed tracks when favorites are listed.
func (c *posthogClient) TrackFavoritesListed(count int) {
	props := baseProperties()
	props["favorites_count"] = count
	c.Track(EventFavoritesListed, props)
}

// --- TUI Tracking Methods ---

// TrackViewNavigated tracks view navigation.
func (c *posthogClient) TrackViewNavigated(viewName, previousView string) {
	props := baseProperties()
	props["view_name"] = viewName
	props["previous_view"] = previousView
	c.Track(EventViewNavigated, props)
}

// TrackSkillInstalled tracks skill installation.
func (c *posthogClient) TrackSkillInstalled(skillName, category string, isLocal bool, platformCount int) {
	props := baseProperties()
	props["skill_name"] = skillName
	props["skill_category"] = category
	props["is_local"] = isLocal
	props["platform_count"] = platformCount
	c.Track(EventSkillInstalled, props)
}

// TrackSkillUninstalled tracks skill uninstallation.
func (c *posthogClient) TrackSkillUninstalled(skillName, category string, isLocal bool) {
	props := baseProperties()
	props["skill_name"] = skillName
	props["skill_category"] = category
	props["is_local"] = isLocal
	c.Track(EventSkillUninstalled, props)
}

// TrackNewSkillCreatedSuccess tracks successful skill creation.
func (c *posthogClient) TrackNewSkillCreatedSuccess(skillName string) {
	props := baseProperties()
	props["skill_name"] = skillName
	c.Track(EventNewSkillCreatedSuccess, props)
}

// TrackNewSkillCreatedFailure tracks unsuccessful skill creation.
func (c *posthogClient) TrackNewSkillCreatedFailure(errorMessage string) {
	props := baseProperties()
	props["reason"] = errorMessage
	c.Track(EventNewSkillCreatedFailure, props)
}

// TrackSearchPerformed tracks search operations.
func (c *posthogClient) TrackSearchPerformed(query string, resultCount int, searchType string) {
	props := baseProperties()
	props["query"] = query
	props["query_length"] = len(query)
	props["result_count"] = resultCount
	props["search_type"] = searchType
	c.Track(EventSearchPerformed, props)
}

// TrackFilterApplied tracks filter application.
func (c *posthogClient) TrackFilterApplied(filterType string, filterCount int) {
	props := baseProperties()
	props["filter_type"] = filterType
	props["filter_count"] = filterCount
	c.Track(EventFilterApplied, props)
}

// TrackSortChanged tracks sort changes.
func (c *posthogClient) TrackSortChanged(sortField, sortDirection string) {
	props := baseProperties()
	props["sort_field"] = sortField
	props["sort_direction"] = sortDirection
	c.Track(EventSortChanged, props)
}

// TrackSkillPreviewed tracks skill preview.
func (c *posthogClient) TrackSkillPreviewed(skillName, category string, platformCount int) {
	props := baseProperties()
	props["skill_name"] = skillName
	props["skill_category"] = category
	props["platform_count"] = platformCount
	c.Track(EventSkillPreviewed, props)
}

// TrackSkillCopied tracks clipboard copy.
func (c *posthogClient) TrackSkillCopied(skillName string) {
	props := baseProperties()
	props["skill_name"] = skillName
	c.Track(EventSkillCopied, props)
}

// TrackOnboardingCompleted tracks onboarding completion.
func (c *posthogClient) TrackOnboardingCompleted(stepsViewed int, skipped bool) {
	props := baseProperties()
	props["steps_viewed"] = stepsViewed
	props["skipped"] = skipped
	c.Track(EventOnboardingCompleted, props)
}

// TrackOnboardingSkipped tracks onboarding skip.
func (c *posthogClient) TrackOnboardingSkipped(stepName string) {
	props := baseProperties()
	props["step_name"] = stepName
	c.Track(EventOnboardingSkipped, props)
}

// TrackSettingsChanged tracks settings changes.
func (c *posthogClient) TrackSettingsChanged(settingName string) {
	props := baseProperties()
	props["setting_name"] = settingName
	c.Track(EventSettingsChanged, props)
}

// TrackKeyboardShortcut tracks keyboard shortcut usage.
func (c *posthogClient) TrackKeyboardShortcut(shortcutKey, contextView string) {
	props := baseProperties()
	props["shortcut_key"] = shortcutKey
	props["context_view"] = contextView
	c.Track(EventKeyboardShortcut, props)
}

// TrackHelpViewed tracks help modal views.
func (c *posthogClient) TrackHelpViewed(contextView string) {
	props := baseProperties()
	props["context_view"] = contextView
	c.Track(EventHelpViewed, props)
}

// TrackPaginationUsed tracks pagination navigation.
func (c *posthogClient) TrackPaginationUsed(direction string, pageNumber int) {
	props := baseProperties()
	props["direction"] = direction
	props["page_number"] = pageNumber
	c.Track(EventPaginationUsed, props)
}

// TrackTagBrowsingEntered tracks tag browsing mode entry.
func (c *posthogClient) TrackTagBrowsingEntered(tagCount int) {
	props := baseProperties()
	props["tag_count"] = tagCount
	c.Track(EventTagBrowsingEntered, props)
}

// TrackTagSelected tracks tag selection.
func (c *posthogClient) TrackTagSelected(tagName string) {
	props := baseProperties()
	props["tag_name"] = tagName
	c.Track(EventTagSelected, props)
}

// TrackErrorDisplayed tracks error display.
func (c *posthogClient) TrackErrorDisplayed(errorType, contextView string) {
	props := baseProperties()
	props["error_type"] = errorType
	props["context_view"] = contextView
	c.Track(EventErrorDisplayed, props)
}

// TrackSourceSelected tracks source selection.
func (c *posthogClient) TrackSourceSelected(sourceIndex, skillCount int) {
	props := baseProperties()
	props["source_index"] = sourceIndex
	props["skill_count"] = skillCount
	c.Track(EventSourceSelected, props)
}

// TrackListRefreshed tracks list refresh.
func (c *posthogClient) TrackListRefreshed(trigger string, skillCount int) {
	props := baseProperties()
	props["trigger"] = trigger
	props["skill_count"] = skillCount
	c.Track(EventListRefreshed, props)
}

// --- Session Tracking Methods ---

// TrackSessionSummary tracks session summary on exit.
func (c *posthogClient) TrackSessionSummary(durationMs int64, viewsVisited, searchesPerformed, skillsInstalled, skillsUninstalled, reposAdded, reposRemoved int) {
	props := baseProperties()
	props["duration_ms"] = durationMs
	props["views_visited"] = viewsVisited
	props["searches_performed"] = searchesPerformed
	props["skills_installed"] = skillsInstalled
	props["skills_uninstalled"] = skillsUninstalled
	props["repos_added"] = reposAdded
	props["repos_removed"] = reposRemoved
	c.Track(EventSessionSummary, props)
}

// --- noopClient implementations (no-ops) ---

func (c *noopClient) TrackAppStarted(mode string, hasSources bool, sourceCount int)               {}
func (c *noopClient) TrackAppExited(mode string, sessionDurationMs int64, commandsRun int)        {}
func (c *noopClient) TrackCLICommandExecuted(commandName string, hasFlags bool, durationMs int64) {}
func (c *noopClient) TrackRepoAdded(sourceID string, skillCount int)                              {}
func (c *noopClient) TrackRepoRemoved(sourceID string, skillCount int)                            {}
func (c *noopClient) TrackRepoSynced(sourceID string, added, removed, updated int)                {}
func (c *noopClient) TrackRepoListed(sourceCount, totalSkillCount int)                            {}
func (c *noopClient) TrackSkillInfoViewed(category string, isLocal bool)                          {}
func (c *noopClient) TrackConfigChanged(settingName string, isDefault bool)                       {}
func (c *noopClient) TrackCLIError(commandName, errorType string)                                 {}
func (c *noopClient) TrackCLIHelpViewed(commandName string, cliArgs []string)                     {}
func (c *noopClient) TrackFavoriteAdded(slug string)                                              {}
func (c *noopClient) TrackFavoriteRemoved(slug string)                                            {}
func (c *noopClient) TrackFavoritesListed(count int)                                              {}
func (c *noopClient) TrackViewNavigated(viewName, previousView string)                            {}
func (c *noopClient) TrackSkillInstalled(skillName, category string, isLocal bool, platformCount int) {
}
func (c *noopClient) TrackSkillUninstalled(skillName, category string, isLocal bool)        {}
func (c *noopClient) TrackNewSkillCreatedSuccess(skillName string)                          {}
func (c *noopClient) TrackNewSkillCreatedFailure(errorMessage string)                       {}
func (c *noopClient) TrackSearchPerformed(query string, resultCount int, searchType string) {}
func (c *noopClient) TrackFilterApplied(filterType string, filterCount int)                 {}
func (c *noopClient) TrackSortChanged(sortField, sortDirection string)                      {}
func (c *noopClient) TrackSkillPreviewed(skillName, category string, platformCount int)     {}
func (c *noopClient) TrackSkillCopied(skillName string)                                     {}
func (c *noopClient) TrackOnboardingCompleted(stepsViewed int, skipped bool)                {}
func (c *noopClient) TrackOnboardingSkipped(stepName string)                                {}
func (c *noopClient) TrackSettingsChanged(settingName string)                               {}
func (c *noopClient) TrackKeyboardShortcut(shortcutKey, contextView string)                 {}
func (c *noopClient) TrackHelpViewed(contextView string)                                    {}
func (c *noopClient) TrackPaginationUsed(direction string, pageNumber int)                  {}
func (c *noopClient) TrackTagBrowsingEntered(tagCount int)                                  {}
func (c *noopClient) TrackTagSelected(tagName string)                                       {}
func (c *noopClient) TrackErrorDisplayed(errorType, contextView string)                     {}
func (c *noopClient) TrackSourceSelected(sourceIndex, skillCount int)                       {}
func (c *noopClient) TrackListRefreshed(trigger string, skillCount int)                     {}
func (c *noopClient) TrackSessionSummary(durationMs int64, viewsVisited, searchesPerformed, skillsInstalled, skillsUninstalled, reposAdded, reposRemoved int) {
}
