// Package telemetry provides anonymous usage tracking via PostHog.
package telemetry

import (
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/posthog/posthog-go"
)

// PostHogAPIKey is set at compile time via ldflags.
var PostHogAPIKey string

// TrackingIDProvider is an interface for getting tracking IDs.
// This allows for testing without a real database.
type TrackingIDProvider interface {
	GetOrCreateTrackingID() string
}

// Client interface for telemetry operations.
type Client interface {
	Track(event string, properties map[string]interface{})
	Close()
	GetTrackingID() string

	// CLI events
	TrackCLICommandExecuted(commandName string, hasFlags bool, durationMs int64)
	TrackRepoAdded(sourceID string, skillCount int)
	TrackRepoRemoved(sourceID string, skillCount int)
	TrackRepoSynced(sourceID string, added, removed, updated int)
	TrackRepoListed(sourceCount, totalSkillCount int)
	TrackSkillInfoViewed(category string, isLocal bool)
	TrackConfigChanged(settingName string, isDefault bool)
	TrackCLIError(commandName, errorType string)
	TrackCLIHelpViewed(commandName string, cliArgs []string)

	// TUI events
	TrackViewNavigated(viewName, previousView string)
	TrackSkillInstalled(skillName, category string, isLocal bool, platformCount int)
	TrackSkillUninstalled(skillName, category string, isLocal bool)
	TrackNewSkillCreatedSuccess(skillName string)
	TrackNewSkillCreatedFailure(errorMessage string)
	TrackSearchPerformed(query string, resultCount int, searchType string)
	TrackFilterApplied(filterType string, filterCount int)
	TrackSortChanged(sortField, sortDirection string)
	TrackSkillPreviewed(skillName, category string, platformCount int)
	TrackSkillCopied(skillName string)
	TrackOnboardingCompleted(stepsViewed int, skipped bool)
	TrackOnboardingSkipped(stepName string)
	TrackSettingsChanged(settingName string)
	TrackKeyboardShortcut(shortcutKey, contextView string)
	TrackHelpViewed(contextView string)
	TrackPaginationUsed(direction string, pageNumber int)
	TrackTagBrowsingEntered(tagCount int)
	TrackTagSelected(tagName string)
	TrackErrorDisplayed(errorType, contextView string)
	TrackSourceSelected(sourceIndex, skillCount int)
	TrackListRefreshed(trigger string, skillCount int)

	// Used in CLI & TUI
	TrackAppStarted(mode string, hasSources bool, sourceCount int)
	TrackAppExited(mode string, sessionDurationMs int64, commandsRun int)

	// Session events
	TrackSessionSummary(durationMs int64, viewsVisited, searchesPerformed, skillsInstalled, skillsUninstalled, reposAdded, reposRemoved int)
}

// posthogClient wraps the PostHog SDK.
type posthogClient struct {
	client    posthog.Client
	sessionID string
	mu        sync.Mutex
}

// noopClient does nothing (for disabled telemetry).
type noopClient struct{}

// IsEnabled returns true if telemetry is enabled.
// Telemetry is opt-out: enabled by default unless SKULTO_TELEMETRY_TRACKING_ENABLED=false.
func IsEnabled() bool {
	return os.Getenv("SKULTO_TELEMETRY_TRACKING_ENABLED") != "false" && PostHogAPIKey != ""
}

// New creates a new telemetry client with a persistent tracking ID from the database.
// If provider is nil, a new UUID is generated per session (fallback behavior).
// Telemetry is opt-out: enabled by default unless SKULTO_TELEMETRY_TRACKING_ENABLED=false.
func New(provider TrackingIDProvider) Client {
	// Telemetry is opt-out - disabled only if explicitly set to "false"
	if !IsEnabled() {
		return &noopClient{}
	}

	client, err := posthog.NewWithConfig(PostHogAPIKey, posthog.Config{
		Endpoint:  "https://us.i.posthog.com",
		BatchSize: 250,
		Interval:  5 * time.Second,
	})
	if err != nil {
		return &noopClient{}
	}

	// Get or create persistent tracking ID
	var sessionID string
	if provider != nil {
		sessionID = provider.GetOrCreateTrackingID()
	} else {
		sessionID = uuid.New().String()
	}

	return &posthogClient{
		client:    client,
		sessionID: sessionID,
	}
}

// Track sends an event to PostHog.
func (c *posthogClient) Track(event string, properties map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	props := posthog.NewProperties()
	props.Set("$process_person_profile", true)
	props.Set("$geoip_disable", true)

	for k, v := range properties {
		props.Set(k, v)
	}

	_ = c.client.Enqueue(posthog.Capture{
		DistinctId: c.sessionID,
		Event:      event,
		Properties: props,
	})
}

// Close flushes remaining events and closes the client.
func (c *posthogClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.client.Close()
}

// Track is a no-op for disabled telemetry.
func (c *noopClient) Track(event string, properties map[string]interface{}) {}

// Close is a no-op for disabled telemetry.
func (c *noopClient) Close() {}

// GetTrackingID returns the anonymous tracking ID for the session.
func (c *posthogClient) GetTrackingID() string {
	return c.sessionID
}

// GetTrackingID returns empty string for disabled telemetry.
func (c *noopClient) GetTrackingID() string {
	return ""
}
