package views

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
)

func setupTagTestDB(t *testing.T) *db.DB {
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

func TestTagView_IsAcceptingTextInput_followsSearchActive(t *testing.T) {
	database := setupTagTestDB(t)
	defer func() { _ = database.Close() }()

	tv := NewTagView(database, &config.Config{})

	// Default state: search not active
	assert.False(t, tv.IsAcceptingTextInput())

	// Activate search via "/" (mirrors how the user enters search mode)
	tv.searchActive = true
	assert.True(t, tv.IsAcceptingTextInput())

	// Deactivate
	tv.searchActive = false
	assert.False(t, tv.IsAcceptingTextInput())
}

func TestTagView_updateSearch_jkAreTypable(t *testing.T) {
	database := setupTagTestDB(t)
	defer func() { _ = database.Close() }()

	tv := NewTagView(database, &config.Config{})
	// Pre-populate filteredSkills so j/k WOULD navigate if intercepted —
	// makes the selectedIdx assertion load-bearing.
	tv.filteredSkills = make([]models.Skill, 5)
	tv.searchActive = true
	tv.searchBar.Focus()

	initialIdx := tv.selectedIdx
	tv.Update("j")
	tv.Update("q")
	tv.Update("k")

	assert.Equal(t, "jqk", tv.searchQuery,
		"j, q, k should all be appended to the search query")
	assert.Equal(t, initialIdx, tv.selectedIdx,
		"selectedIdx should not change while typing in active search")
}

func TestTagView_updateSearch_arrowsStillNavigate(t *testing.T) {
	database := setupTagTestDB(t)
	defer func() { _ = database.Close() }()

	tv := NewTagView(database, &config.Config{})
	tv.filteredSkills = make([]models.Skill, 5)
	tv.searchActive = true
	tv.searchBar.Focus()

	initialIdx := tv.selectedIdx
	tv.Update("down")

	assert.Equal(t, initialIdx+1, tv.selectedIdx,
		"down arrow should navigate filtered results when search is active")
}

func TestTagView_outerUpdate_jkStillNavigateWhenSearchInactive(t *testing.T) {
	database := setupTagTestDB(t)
	defer func() { _ = database.Close() }()

	tv := NewTagView(database, &config.Config{})
	tv.filteredSkills = make([]models.Skill, 5)
	tv.searchActive = false // critical: outer Update path

	initialIdx := tv.selectedIdx
	tv.Update("j")

	assert.Equal(t, initialIdx+1, tv.selectedIdx,
		"j should navigate when search is NOT active (regression check)")
}
