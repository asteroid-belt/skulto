package mcp

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/favorites"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/telemetry"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTelemetryClient is a mock telemetry client for testing.
type mockTelemetryClient struct {
	mu     sync.Mutex
	events []mockEvent
}

type mockEvent struct {
	name       string
	properties map[string]interface{}
}

func (m *mockTelemetryClient) Track(event string, properties map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, mockEvent{name: event, properties: properties})
}

func (m *mockTelemetryClient) Close()                {}
func (m *mockTelemetryClient) GetTrackingID() string { return "test-tracking-id" }

// Implement all Client interface methods as no-ops for mock
func (m *mockTelemetryClient) TrackCLICommandExecuted(commandName string, hasFlags bool, durationMs int64) {
}
func (m *mockTelemetryClient) TrackRepoAdded(sourceID string, skillCount int) {
	m.Track(telemetry.EventRepoAdded, map[string]interface{}{"source_id": sourceID, "skill_count": skillCount})
}
func (m *mockTelemetryClient) TrackRepoRemoved(sourceID string, skillCount int)             {}
func (m *mockTelemetryClient) TrackRepoSynced(sourceID string, added, removed, updated int) {}
func (m *mockTelemetryClient) TrackRepoListed(sourceCount, totalSkillCount int)             {}
func (m *mockTelemetryClient) TrackSkillInfoViewed(category string, isLocal bool)           {}
func (m *mockTelemetryClient) TrackConfigChanged(settingName string, isDefault bool)        {}
func (m *mockTelemetryClient) TrackCLIError(commandName, errorType string)                  {}
func (m *mockTelemetryClient) TrackCLIHelpViewed(commandName string, cliArgs []string)      {}
func (m *mockTelemetryClient) TrackFavoriteAdded(slug string) {
	m.Track(telemetry.EventFavoriteAdded, map[string]interface{}{"skill_slug": slug})
}
func (m *mockTelemetryClient) TrackFavoriteRemoved(slug string) {
	m.Track(telemetry.EventFavoriteRemoved, map[string]interface{}{"skill_slug": slug})
}
func (m *mockTelemetryClient) TrackFavoritesListed(count int) {
	m.Track(telemetry.EventFavoritesListed, map[string]interface{}{"favorites_count": count})
}
func (m *mockTelemetryClient) TrackViewNavigated(viewName, previousView string) {}
func (m *mockTelemetryClient) TrackSkillInstalled(skillName, category string, isLocal bool, platformCount int) {
	m.Track(telemetry.EventSkillInstalled, map[string]interface{}{"skill_name": skillName})
}
func (m *mockTelemetryClient) TrackSkillUninstalled(skillName, category string, isLocal bool) {
	m.Track(telemetry.EventSkillUninstalled, map[string]interface{}{"skill_name": skillName})
}
func (m *mockTelemetryClient) TrackNewSkillCreatedSuccess(skillName string)    {}
func (m *mockTelemetryClient) TrackNewSkillCreatedFailure(errorMessage string) {}
func (m *mockTelemetryClient) TrackSearchPerformed(query string, resultCount int, searchType string) {
	m.Track(telemetry.EventSearchPerformed, map[string]interface{}{"query": query, "result_count": resultCount, "search_type": searchType})
}
func (m *mockTelemetryClient) TrackFilterApplied(filterType string, filterCount int)                {}
func (m *mockTelemetryClient) TrackSortChanged(sortField, sortDirection string)                     {}
func (m *mockTelemetryClient) TrackSkillPreviewed(skillName, category string, platformCount int)    {}
func (m *mockTelemetryClient) TrackSkillCopied(skillName string)                                    {}
func (m *mockTelemetryClient) TrackOnboardingCompleted(stepsViewed int, skipped bool)               {}
func (m *mockTelemetryClient) TrackOnboardingSkipped(stepName string)                               {}
func (m *mockTelemetryClient) TrackSettingsChanged(settingName string)                              {}
func (m *mockTelemetryClient) TrackKeyboardShortcut(shortcutKey, contextView string)                {}
func (m *mockTelemetryClient) TrackHelpViewed(contextView string)                                   {}
func (m *mockTelemetryClient) TrackPaginationUsed(direction string, pageNumber int)                 {}
func (m *mockTelemetryClient) TrackTagBrowsingEntered(tagCount int)                                 {}
func (m *mockTelemetryClient) TrackTagSelected(tagName string)                                      {}
func (m *mockTelemetryClient) TrackErrorDisplayed(errorType, contextView string)                    {}
func (m *mockTelemetryClient) TrackSourceSelected(sourceIndex, skillCount int)                      {}
func (m *mockTelemetryClient) TrackListRefreshed(trigger string, skillCount int)                    {}
func (m *mockTelemetryClient) TrackAppStarted(mode string, hasSources bool, sourceCount int)        {}
func (m *mockTelemetryClient) TrackAppExited(mode string, sessionDurationMs int64, commandsRun int) {}
func (m *mockTelemetryClient) TrackSessionSummary(durationMs int64, viewsVisited, searchesPerformed, skillsInstalled, skillsUninstalled, reposAdded, reposRemoved int) {
}

// Shared events
func (m *mockTelemetryClient) TrackSkillViewed(slug, category string, isLocal bool) {
	m.Track(telemetry.EventSkillViewed, map[string]interface{}{"skill_slug": slug, "skill_category": category, "is_local": isLocal})
}
func (m *mockTelemetryClient) TrackSkillsListed(count int, source string) {
	m.Track(telemetry.EventSkillsListed, map[string]interface{}{"result_count": count, "source": source})
}
func (m *mockTelemetryClient) TrackStatsViewed() {
	m.Track(telemetry.EventStatsViewed, map[string]interface{}{})
}
func (m *mockTelemetryClient) TrackRecentSkillsViewed(count int) {
	m.Track(telemetry.EventRecentSkillsViewed, map[string]interface{}{"result_count": count})
}
func (m *mockTelemetryClient) TrackInstalledSkillsChecked(count int) {
	m.Track(telemetry.EventInstalledSkillsChecked, map[string]interface{}{"installed_count": count})
}

// MCP events
func (m *mockTelemetryClient) TrackMCPToolCalled(toolName string, durationMs int64, success bool) {
	m.Track(telemetry.EventMCPToolCalled, map[string]interface{}{"tool_name": toolName, "success": success})
}

func (m *mockTelemetryClient) getEvents() []mockEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	events := make([]mockEvent, len(m.events))
	copy(events, m.events)
	return events
}

func (m *mockTelemetryClient) hasEvent(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.events {
		if e.name == name {
			return true
		}
	}
	return false
}

// Verify mockTelemetryClient satisfies the telemetry.Client interface
var _ telemetry.Client = (*mockTelemetryClient)(nil)

func setupTestDB(t *testing.T) *db.DB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.New(db.Config{
		Path:        dbPath,
		Debug:       false,
		MaxIdleConn: 1,
		MaxOpenConn: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func setupTestFavorites(t *testing.T) *favorites.Store {
	t.Helper()
	tmpDir := t.TempDir()
	favPath := filepath.Join(tmpDir, "favorites.json")
	store := favorites.NewStore(favPath)
	require.NoError(t, store.Load())
	return store
}

func TestNewServer(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)

	server := NewServer(database, cfg, favStore, nil)

	assert.NotNil(t, server)
	assert.NotNil(t, server.server)
	assert.NotNil(t, server.db)
	assert.NotNil(t, server.installer)
	assert.NotNil(t, server.favorites)
}

func TestNewServer_WithNilConfig(t *testing.T) {
	database := setupTestDB(t)
	favStore := setupTestFavorites(t)

	// Should not panic with nil config
	server := NewServer(database, nil, favStore, nil)

	assert.NotNil(t, server)
	assert.NotNil(t, server.server)
}

func TestNewServer_WithNilFavorites(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}

	// Should not panic with nil favorites store
	server := NewServer(database, cfg, nil, nil)

	assert.NotNil(t, server)
	assert.NotNil(t, server.server)
	assert.Nil(t, server.favorites)
}

func TestNewServer_WithTelemetry(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	mockTC := &mockTelemetryClient{}

	server := NewServer(database, cfg, favStore, mockTC)

	assert.NotNil(t, server)
	assert.NotNil(t, server.telemetry)
}

// Helper to seed skills for telemetry tests
func seedSkillsForTelemetryTests(t *testing.T, database *db.DB) {
	t.Helper()
	skills := []models.Skill{
		{
			ID:          "telemetry-test-1",
			Slug:        "telemetry-test-skill",
			Title:       "Telemetry Test Skill",
			Description: "A skill for testing telemetry",
			Content:     "# Test Content",
			Category:    "testing",
		},
	}
	for _, skill := range skills {
		require.NoError(t, database.CreateSkill(&skill))
	}
}

func TestMCPTelemetry_SearchCallsTelemetry(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	mockTC := &mockTelemetryClient{}

	server := NewServer(database, cfg, favStore, mockTC)
	seedSkillsForTelemetryTests(t, database)

	ctx := context.Background()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"query": "telemetry",
	}

	result, err := server.handleSearch(ctx, req)
	require.NoError(t, err)
	require.False(t, result.IsError)

	// Verify telemetry events were tracked
	assert.True(t, mockTC.hasEvent(telemetry.EventSearchPerformed), "search_performed event should be tracked")
	assert.True(t, mockTC.hasEvent(telemetry.EventMCPToolCalled), "mcp_tool_called event should be tracked")
}

func TestMCPTelemetry_GetSkillCallsTelemetry(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	mockTC := &mockTelemetryClient{}

	server := NewServer(database, cfg, favStore, mockTC)
	seedSkillsForTelemetryTests(t, database)

	ctx := context.Background()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"slug": "telemetry-test-skill",
	}

	result, err := server.handleGetSkill(ctx, req)
	require.NoError(t, err)
	require.False(t, result.IsError)

	// Verify telemetry events were tracked
	assert.True(t, mockTC.hasEvent(telemetry.EventSkillViewed), "skill_viewed event should be tracked")
	assert.True(t, mockTC.hasEvent(telemetry.EventMCPToolCalled), "mcp_tool_called event should be tracked")
}

func TestMCPTelemetry_ListSkillsCallsTelemetry(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	mockTC := &mockTelemetryClient{}

	server := NewServer(database, cfg, favStore, mockTC)
	seedSkillsForTelemetryTests(t, database)

	ctx := context.Background()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"limit": float64(10),
	}

	result, err := server.handleListSkills(ctx, req)
	require.NoError(t, err)
	require.False(t, result.IsError)

	// Verify telemetry events were tracked
	assert.True(t, mockTC.hasEvent(telemetry.EventSkillsListed), "skills_listed event should be tracked")
	assert.True(t, mockTC.hasEvent(telemetry.EventMCPToolCalled), "mcp_tool_called event should be tracked")
}

func TestMCPTelemetry_GetStatsCallsTelemetry(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	mockTC := &mockTelemetryClient{}

	server := NewServer(database, cfg, favStore, mockTC)

	ctx := context.Background()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	result, err := server.handleGetStats(ctx, req)
	require.NoError(t, err)
	require.False(t, result.IsError)

	// Verify telemetry events were tracked
	assert.True(t, mockTC.hasEvent(telemetry.EventStatsViewed), "stats_viewed event should be tracked")
	assert.True(t, mockTC.hasEvent(telemetry.EventMCPToolCalled), "mcp_tool_called event should be tracked")
}

func TestMCPTelemetry_GetRecentCallsTelemetry(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	mockTC := &mockTelemetryClient{}

	server := NewServer(database, cfg, favStore, mockTC)

	ctx := context.Background()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"limit": float64(5),
	}

	result, err := server.handleGetRecent(ctx, req)
	require.NoError(t, err)
	require.False(t, result.IsError)

	// Verify telemetry events were tracked
	assert.True(t, mockTC.hasEvent(telemetry.EventRecentSkillsViewed), "recent_skills_viewed event should be tracked")
	assert.True(t, mockTC.hasEvent(telemetry.EventMCPToolCalled), "mcp_tool_called event should be tracked")
}

func TestMCPTelemetry_FavoriteCallsTelemetry(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	mockTC := &mockTelemetryClient{}

	server := NewServer(database, cfg, favStore, mockTC)
	seedSkillsForTelemetryTests(t, database)

	ctx := context.Background()

	// Test adding favorite
	addReq := mcp.CallToolRequest{}
	addReq.Params.Arguments = map[string]any{
		"slug":   "telemetry-test-skill",
		"action": "add",
	}

	result, err := server.handleFavorite(ctx, addReq)
	require.NoError(t, err)
	require.False(t, result.IsError)

	assert.True(t, mockTC.hasEvent(telemetry.EventFavoriteAdded), "favorite_added event should be tracked")
	assert.True(t, mockTC.hasEvent(telemetry.EventMCPToolCalled), "mcp_tool_called event should be tracked")

	// Test removing favorite
	mockTC.events = nil // Reset events
	removeReq := mcp.CallToolRequest{}
	removeReq.Params.Arguments = map[string]any{
		"slug":   "telemetry-test-skill",
		"action": "remove",
	}

	result, err = server.handleFavorite(ctx, removeReq)
	require.NoError(t, err)
	require.False(t, result.IsError)

	assert.True(t, mockTC.hasEvent(telemetry.EventFavoriteRemoved), "favorite_removed event should be tracked")
}

func TestMCPTelemetry_GetFavoritesCallsTelemetry(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	mockTC := &mockTelemetryClient{}

	server := NewServer(database, cfg, favStore, mockTC)

	ctx := context.Background()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	result, err := server.handleGetFavorites(ctx, req)
	require.NoError(t, err)
	require.False(t, result.IsError)

	// Verify telemetry events were tracked
	assert.True(t, mockTC.hasEvent(telemetry.EventFavoritesListed), "favorites_listed event should be tracked")
	assert.True(t, mockTC.hasEvent(telemetry.EventMCPToolCalled), "mcp_tool_called event should be tracked")
}

func TestMCPTelemetry_CheckCallsTelemetry(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	mockTC := &mockTelemetryClient{}

	server := NewServer(database, cfg, favStore, mockTC)

	ctx := context.Background()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	result, err := server.handleCheck(ctx, req)
	require.NoError(t, err)
	require.False(t, result.IsError)

	// Verify telemetry events were tracked
	assert.True(t, mockTC.hasEvent(telemetry.EventInstalledSkillsChecked), "installed_skills_checked event should be tracked")
	assert.True(t, mockTC.hasEvent(telemetry.EventMCPToolCalled), "mcp_tool_called event should be tracked")
}

func TestMCPTelemetry_BrowseTagsCallsTelemetry(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	mockTC := &mockTelemetryClient{}

	server := NewServer(database, cfg, favStore, mockTC)

	ctx := context.Background()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}

	result, err := server.handleBrowseTags(ctx, req)
	require.NoError(t, err)
	require.False(t, result.IsError)

	// Verify mcp_tool_called is tracked (browse_tags only tracks tool call, no specific event)
	assert.True(t, mockTC.hasEvent(telemetry.EventMCPToolCalled), "mcp_tool_called event should be tracked")
}

func TestMCPTelemetry_FailedOperationTracksFalseSuccess(t *testing.T) {
	database := setupTestDB(t)
	cfg := &config.Config{}
	favStore := setupTestFavorites(t)
	mockTC := &mockTelemetryClient{}

	server := NewServer(database, cfg, favStore, mockTC)

	ctx := context.Background()

	// Try to get a non-existent skill
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"slug": "non-existent-skill",
	}

	result, err := server.handleGetSkill(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.IsError)

	// Verify mcp_tool_called was tracked with success=false
	events := mockTC.getEvents()
	var foundToolCall bool
	for _, e := range events {
		if e.name == telemetry.EventMCPToolCalled {
			foundToolCall = true
			success, ok := e.properties["success"].(bool)
			assert.True(t, ok, "success property should be a bool")
			assert.False(t, success, "failed operation should track success=false")
		}
	}
	assert.True(t, foundToolCall, "mcp_tool_called event should be tracked even on failure")
}
