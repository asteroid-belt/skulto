package views

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/telemetry"
	"github.com/stretchr/testify/assert"
)

// noopTelemetry returns a noop telemetry client for testing.
func noopTelemetry() telemetry.Client {
	return telemetry.New(nil)
}

// setupSearchTestDB creates an in-memory test database with sample tags
// Note: The database auto-creates a "mine" tag on init with priority 100
func setupSearchTestDB(t *testing.T) *db.DB {
	t.Helper()

	database, err := db.New(db.Config{
		Path:        ":memory:",
		Debug:       false,
		MaxIdleConn: 1,
		MaxOpenConn: 1,
	})
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	return database
}

func TestSearchViewTagModeDisplay(t *testing.T) {
	database := setupSearchTestDB(t)
	defer func() { _ = database.Close() }()

	// Insert test tags (need unique slugs)
	tags := []models.Tag{
		{ID: "python", Name: "python", Slug: "python", Count: 10, Category: "language"},
		{ID: "react", Name: "react", Slug: "react", Count: 5, Category: "framework"},
	}
	for _, tag := range tags {
		err := database.CreateTag(&tag)
		assert.NoError(t, err)
	}

	sv := NewSearchView(database, &config.Config{}, nil)
	sv.SetSize(80, 24)
	sv.Init(noopTelemetry())

	// With empty query, should show tags
	view := sv.View()
	assert.Contains(t, view, "Browse All Tags")
	assert.Contains(t, view, "python")
	assert.Contains(t, view, "react")
}

func TestSearchViewTagModeFocusTransition(t *testing.T) {
	database := setupSearchTestDB(t)
	defer func() { _ = database.Close() }()

	// Don't need extra tags - "mine" tag exists from DB init
	sv := NewSearchView(database, &config.Config{}, nil)
	sv.SetSize(80, 24)
	sv.Init(noopTelemetry())

	// Initially focused on search bar
	assert.Equal(t, FocusSearchBar, sv.focusArea)

	// Tab should move to tag grid
	sv.Update("tab")
	assert.Equal(t, FocusTagGrid, sv.focusArea)

	// Up should move back to search bar (from first row)
	sv.Update("up")
	assert.Equal(t, FocusSearchBar, sv.focusArea)
}

func TestSearchViewTagModeDownKey(t *testing.T) {
	database := setupSearchTestDB(t)
	defer func() { _ = database.Close() }()

	// Don't need extra tags - "mine" tag exists from DB init
	sv := NewSearchView(database, &config.Config{}, nil)
	sv.SetSize(80, 24)
	sv.Init(noopTelemetry())

	// Down should also move to tag grid (like Tab)
	sv.Update("down")
	assert.Equal(t, FocusTagGrid, sv.focusArea)
}

func TestSearchViewTagModeToSearchMode(t *testing.T) {
	database := setupSearchTestDB(t)
	defer func() { _ = database.Close() }()

	sv := NewSearchView(database, &config.Config{}, nil)
	sv.SetSize(80, 24)
	sv.Init(noopTelemetry())

	// Type less than 3 chars - still in tag mode
	sv.Update("a")
	sv.Update("b")
	assert.Less(t, len(sv.query), MinSearchChars)

	view := sv.View()
	assert.Contains(t, view, "Browse All Tags")

	// Type 3rd char - switch to search mode
	sv.Update("c")
	assert.GreaterOrEqual(t, len(sv.query), MinSearchChars)

	view = sv.View()
	assert.NotContains(t, view, "Browse All Tags")
}

func TestSearchViewTagSelection(t *testing.T) {
	database := setupSearchTestDB(t)
	defer func() { _ = database.Close() }()

	// Add tags with slugs. Tags are ordered: mine (priority 100), then by count desc
	tags := []models.Tag{
		{ID: "python", Name: "python", Slug: "python", Count: 10},
		{ID: "react", Name: "react", Slug: "react", Count: 5},
	}
	for _, tag := range tags {
		err := database.CreateTag(&tag)
		assert.NoError(t, err)
	}

	sv := NewSearchView(database, &config.Config{}, nil)
	sv.SetSize(80, 24)
	sv.Init(noopTelemetry())

	// Move to tag grid - first tag is "mine" (priority 100)
	sv.Update("tab")

	// Navigate right twice to get to "react" (mine -> python -> react)
	sv.Update("right")
	sv.Update("right")

	// Press enter - should signal tag selected
	_, tagSelected, _ := sv.Update("enter")
	assert.True(t, tagSelected)

	// Should return the selected tag
	tag := sv.GetSelectedTag()
	assert.NotNil(t, tag)
	assert.Equal(t, "react", tag.Name)
}

func TestSearchViewTagModeEscape(t *testing.T) {
	database := setupSearchTestDB(t)
	defer func() { _ = database.Close() }()

	sv := NewSearchView(database, &config.Config{}, nil)
	sv.SetSize(80, 24)
	sv.Init(noopTelemetry())

	// Escape from search bar should return true (go back)
	back, _, _ := sv.Update("esc")
	assert.True(t, back)
}

func TestSearchViewTagModeEscapeFromGrid(t *testing.T) {
	database := setupSearchTestDB(t)
	defer func() { _ = database.Close() }()

	sv := NewSearchView(database, &config.Config{}, nil)
	sv.SetSize(80, 24)
	sv.Init(noopTelemetry())

	// Move to tag grid
	sv.Update("tab")
	assert.Equal(t, FocusTagGrid, sv.focusArea)

	// Escape from tag grid should also return true (go back)
	back, _, _ := sv.Update("esc")
	assert.True(t, back)
}

func TestSearchViewTagModePrintableCharRefocusesSearch(t *testing.T) {
	database := setupSearchTestDB(t)
	defer func() { _ = database.Close() }()

	sv := NewSearchView(database, &config.Config{}, nil)
	sv.SetSize(80, 24)
	sv.Init(noopTelemetry())

	// Move to tag grid
	sv.Update("tab")
	assert.Equal(t, FocusTagGrid, sv.focusArea)

	// Typing a character should refocus search bar and add the character
	sv.Update("p")
	assert.Equal(t, FocusSearchBar, sv.focusArea)
	assert.Equal(t, "p", sv.query)
}

func TestSearchViewTagModeNavigation(t *testing.T) {
	database := setupSearchTestDB(t)
	defer func() { _ = database.Close() }()

	// Add tags with slugs. Order will be: python (10), react (5), docker (3)
	// Note: "mine" tag is filtered out from the UI
	tags := []models.Tag{
		{ID: "python", Name: "python", Slug: "python", Count: 10},
		{ID: "react", Name: "react", Slug: "react", Count: 5},
		{ID: "docker", Name: "docker", Slug: "docker", Count: 3},
	}
	for _, tag := range tags {
		err := database.CreateTag(&tag)
		assert.NoError(t, err)
	}

	sv := NewSearchView(database, &config.Config{}, nil)
	sv.SetSize(80, 24)
	sv.Init(noopTelemetry())

	// Move to tag grid - first tag is "python" (mine is filtered out)
	sv.Update("tab")

	tag := sv.GetSelectedTag()
	assert.NotNil(t, tag)
	assert.Equal(t, "python", tag.Name)

	// Navigate right -> react
	sv.Update("right")
	tag = sv.GetSelectedTag()
	assert.Equal(t, "react", tag.Name)

	// Navigate right -> docker
	sv.Update("right")
	tag = sv.GetSelectedTag()
	assert.Equal(t, "docker", tag.Name)

	// Navigate left -> react
	sv.Update("left")
	tag = sv.GetSelectedTag()
	assert.Equal(t, "react", tag.Name)

	// Navigate left -> python
	sv.Update("left")
	tag = sv.GetSelectedTag()
	assert.Equal(t, "python", tag.Name)
}

func TestSearchViewTagModeVimNavigation(t *testing.T) {
	database := setupSearchTestDB(t)
	defer func() { _ = database.Close() }()

	// Add tags. Note: "mine" tag is filtered out from the UI
	tags := []models.Tag{
		{ID: "python", Name: "python", Slug: "python", Count: 10},
		{ID: "react", Name: "react", Slug: "react", Count: 5},
	}
	for _, tag := range tags {
		err := database.CreateTag(&tag)
		assert.NoError(t, err)
	}

	sv := NewSearchView(database, &config.Config{}, nil)
	sv.SetSize(80, 24)
	sv.Init(noopTelemetry())

	// Move to tag grid - starts at "python" (mine is filtered out)
	sv.Update("tab")
	tag := sv.GetSelectedTag()
	assert.Equal(t, "python", tag.Name)

	// Navigate with 'l' (vim right) -> react
	sv.Update("l")
	tag = sv.GetSelectedTag()
	assert.Equal(t, "react", tag.Name)

	// Navigate with 'h' (vim left) -> python
	sv.Update("h")
	tag = sv.GetSelectedTag()
	assert.Equal(t, "python", tag.Name)
}

func TestSearchViewGetKeyboardCommandsTagMode(t *testing.T) {
	database := setupSearchTestDB(t)
	defer func() { _ = database.Close() }()

	sv := NewSearchView(database, &config.Config{}, nil)
	sv.SetSize(80, 24)
	sv.Init(noopTelemetry())

	// In tag mode (query < 3)
	cmds := sv.GetKeyboardCommands()
	assert.Equal(t, "Search (Tag Browse)", cmds.ViewName)
}

func TestSearchViewGetKeyboardCommandsSearchMode(t *testing.T) {
	database := setupSearchTestDB(t)
	defer func() { _ = database.Close() }()

	sv := NewSearchView(database, &config.Config{}, nil)
	sv.SetSize(80, 24)
	sv.Init(noopTelemetry())

	// Enter search mode
	sv.Update("a")
	sv.Update("b")
	sv.Update("c")

	cmds := sv.GetKeyboardCommands()
	assert.Equal(t, "Search", cmds.ViewName)
}

func TestSearchViewMineTagFiltered(t *testing.T) {
	database := setupSearchTestDB(t)
	defer func() { _ = database.Close() }()

	// The database auto-creates a "mine" tag on init.
	// Add a regular tag - the "mine" tag should be filtered out from the UI.
	tags := []models.Tag{
		{ID: "python", Name: "python", Slug: "python", Count: 1000},
	}
	for _, tag := range tags {
		err := database.CreateTag(&tag)
		assert.NoError(t, err)
	}

	sv := NewSearchView(database, &config.Config{}, nil)
	sv.SetSize(80, 24)
	sv.Init(noopTelemetry())

	// Move to tag grid - first tag should be "python" (mine is filtered out)
	sv.Update("tab")

	tag := sv.GetSelectedTag()
	assert.NotNil(t, tag)
	assert.Equal(t, "python", tag.Name)

	// Verify mine tag is not in the list
	for _, tag := range sv.allTags {
		assert.NotEqual(t, "mine", tag.ID)
	}
}

func TestSearchViewTagEnterFromSearchBar(t *testing.T) {
	database := setupSearchTestDB(t)
	defer func() { _ = database.Close() }()

	sv := NewSearchView(database, &config.Config{}, nil)
	sv.SetSize(80, 24)
	sv.Init(noopTelemetry())

	// Press enter while search bar is focused (query < 3)
	// Should not select a tag (need to be in tag grid)
	_, tagSelected, _ := sv.Update("enter")
	assert.False(t, tagSelected)
}

func TestSearchView_ShowsLocalBadge(t *testing.T) {
	database := setupSearchTestDB(t)
	defer func() { _ = database.Close() }()

	// Create a local skill with is_local=true
	localSkill := &models.Skill{
		ID:          "local-skill-id",
		Title:       "My Local Skill",
		Description: "A locally ingested skill",
		Slug:        "my-local-skill",
		IsLocal:     true,
	}
	err := database.CreateSkill(localSkill)
	assert.NoError(t, err)

	// Create a remote skill with is_local=false
	remoteSkill := &models.Skill{
		ID:          "remote-skill-id",
		Title:       "Remote Skill",
		Description: "A skill from the registry",
		Slug:        "remote-skill",
		IsLocal:     false,
	}
	err = database.CreateSkill(remoteSkill)
	assert.NoError(t, err)

	sv := NewSearchView(database, &config.Config{}, nil)
	sv.SetSize(80, 24)
	sv.Init(noopTelemetry())

	// Simulate search results using legacy mode (no search service)
	sv.legacyResults = []models.Skill{*localSkill, *remoteSkill}
	sv.query = "skill"

	// Render the view
	view := sv.View()

	// Local skill should show [local] badge
	assert.Contains(t, view, "[local]", "Local skill should show [local] badge")
	// The local skill title should be present
	assert.Contains(t, view, "My Local Skill")
	// The remote skill should also be present but not show [local]
	assert.Contains(t, view, "Remote Skill")
}
