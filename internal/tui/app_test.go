package tui

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/tui/views"
	"github.com/stretchr/testify/assert"
)

// setupTestDB creates an in-memory database for app-level tests.
func setupTestDB(t *testing.T) *db.DB {
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

// TestApp_currentViewIsAcceptingTextInput_defaultViewsReturnFalse verifies
// that views without text inputs return false from the helper.
func TestApp_currentViewIsAcceptingTextInput_defaultViewsReturnFalse(t *testing.T) {
	m := &Model{}

	nonInputViews := []ViewType{
		ViewHome, ViewReset, ViewSkillDetail,
		ViewOnboardingIntro, ViewOnboardingSkillsIntro,
		ViewOnboardingSetup, ViewOnboardingTools, ViewOnboardingSkills,
		ViewHelp, ViewSettings,
	}
	for _, v := range nonInputViews {
		m.currentView = v
		assert.False(t, m.currentViewIsAcceptingTextInput(),
			"view %v should return false (no text input)", v)
	}
}

// TestApp_currentViewIsAcceptingTextInput_searchViewAlwaysTrue verifies that
// the helper returns true whenever currentView == ViewSearch, mirroring
// SearchView.IsAcceptingTextInput which is unconditionally true.
func TestApp_currentViewIsAcceptingTextInput_searchViewAlwaysTrue(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	m := &Model{
		currentView: ViewSearch,
		searchView:  views.NewSearchView(database, &config.Config{}, nil),
	}

	assert.True(t, m.currentViewIsAcceptingTextInput())
}

// TestApp_currentViewIsAcceptingTextInput_tagViewFollowsSearchActive verifies
// the helper returns the tag view's searchActive flag.
func TestApp_currentViewIsAcceptingTextInput_tagViewFollowsSearchActive(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()

	tv := views.NewTagView(database, &config.Config{})
	m := &Model{
		currentView: ViewTag,
		tagView:     tv,
	}

	// Default: search not active
	assert.False(t, m.currentViewIsAcceptingTextInput())

	// Activate search via "/" (the same way the user does it).
	tv.Update("/")
	assert.True(t, m.currentViewIsAcceptingTextInput())
}

// TestApp_currentViewIsAcceptingTextInput_addSourceFollowsInputFocus verifies
// the helper returns the add-source view's input focus state.
func TestApp_currentViewIsAcceptingTextInput_addSourceFollowsInputFocus(t *testing.T) {
	asv := views.NewAddSourceView(nil, &config.Config{})

	m := &Model{
		currentView:   ViewAddSource,
		addSourceView: asv,
	}

	// Pre-Init: input not focused
	assert.False(t, m.currentViewIsAcceptingTextInput())

	// After Init: input focused
	asv.Init()
	assert.True(t, m.currentViewIsAcceptingTextInput())
}
