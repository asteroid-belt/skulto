package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_DisabledByEnvVar(t *testing.T) {
	t.Setenv("SKULTO_TELEMETRY_TRACKING_ENABLED", "false")

	client := New(nil)
	_, ok := client.(*noopClient)
	assert.True(t, ok, "Should return noopClient when disabled")
}

func TestNew_DisabledWithoutAPIKey(t *testing.T) {
	originalKey := PostHogAPIKey
	PostHogAPIKey = ""
	defer func() { PostHogAPIKey = originalKey }()

	client := New(nil)
	_, ok := client.(*noopClient)
	assert.True(t, ok, "Should return noopClient without API key")
}

func TestNoopClient_DoesNotPanic(t *testing.T) {
	client := &noopClient{}

	// Should not panic - CLI events
	client.Track("test_event", map[string]interface{}{"key": "value"})
	client.TrackAppStarted("cli", true, 2)
	client.TrackAppExited("cli", 5000, 3)
	client.TrackCLICommandExecuted("add", true, 100)
	client.TrackRepoAdded("owner/repo", 10)
	client.TrackRepoRemoved("owner/repo", 10)
	client.TrackRepoSynced("owner/repo", 5, 2, 3)
	client.TrackRepoListed(2, 25)
	client.TrackConfigChanged("theme", false)
	client.TrackSkillsDiscovered(5, true, true)
	client.TrackSkillIngested("my-skill", "project")
	client.TrackCLIError("add", "network_error")
	client.TrackCLIHelpViewed("root", []string{"--help"})

	// TUI events
	client.TrackViewNavigated("search", "home")
	client.TrackSkillInstalled("My Skill", "workflow", false, 2)
	client.TrackSkillUninstalled("My Skill", "workflow", false)
	client.TrackSearchPerformed("test query", 5, "fts")
	client.TrackFilterApplied("tag", 3)
	client.TrackSortChanged("name", "asc")
	client.TrackSkillCopied("My Skill")
	client.TrackOnboardingCompleted(3, false)
	client.TrackOnboardingSkipped("setup")
	client.TrackSettingsChanged("theme")
	client.TrackKeyboardShortcut("/", "home")
	client.TrackHelpViewed("search")
	client.TrackPaginationUsed("next", 2)
	client.TrackTagBrowsingEntered(15)
	client.TrackTagSelected("python")
	client.TrackErrorDisplayed("network", "search")
	client.TrackSourceSelected(0, 10)
	client.TrackListRefreshed("manual", 25)

	// Session events
	client.TrackSessionSummary(60000, 5, 3, 2, 1, 1, 0)

	// Shared events (all interfaces)
	client.TrackSkillViewed("test-slug", "workflow", false)
	client.TrackSkillViewed("local-skill", "workflow", true)
	client.TrackSkillsListed(10, "mcp")
	client.TrackSkillsListed(5, "search")
	client.TrackStatsViewed()
	client.TrackRecentSkillsViewed(5)
	client.TrackInstalledSkillsChecked(3)

	// MCP events
	client.TrackMCPToolCalled("skulto_search", 100, true)
	client.TrackMCPToolCalled("skulto_get_skill", 50, false)
	client.TrackMCPToolCalled("skulto_install", 200, true)

	client.Close()
}

func TestBaseProperties(t *testing.T) {
	props := baseProperties()

	assert.Contains(t, props, "os")
	assert.Contains(t, props, "arch")
	assert.Contains(t, props, "version")
}
