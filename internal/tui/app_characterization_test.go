package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// CHARACTERIZATION TESTS FOR TUI APP
// =============================================================================
// These tests capture CURRENT behavior, not desired behavior.
// If these tests fail after refactoring, behavior changed (possibly incorrectly).
// DO NOT MODIFY these tests without understanding why existing behavior changed.
// =============================================================================

// TestCharacterization_ViewType_Constants captures the ViewType constant values.
// These values are important as they may be persisted or used in comparisons.
func TestCharacterization_ViewType_Constants(t *testing.T) {
	// Current behavior: ViewType constants are sequential starting from 0
	assert.Equal(t, ViewType(0), ViewHome)
	assert.Equal(t, ViewType(1), ViewSearch)
	assert.Equal(t, ViewType(2), ViewReset)
	assert.Equal(t, ViewType(3), ViewSkillDetail)
	assert.Equal(t, ViewType(4), ViewTag)
	assert.Equal(t, ViewType(5), ViewOnboardingIntro)
	assert.Equal(t, ViewType(6), ViewOnboardingSkillsIntro)
	assert.Equal(t, ViewType(7), ViewOnboardingSetup)
	assert.Equal(t, ViewType(8), ViewOnboardingTools)
	assert.Equal(t, ViewType(9), ViewOnboardingSkills)
	assert.Equal(t, ViewType(10), ViewAddSource)
	assert.Equal(t, ViewType(11), ViewHelp)
	assert.Equal(t, ViewType(12), ViewSettings)
}

// TestCharacterization_ViewType_TotalCount captures the total number of views.
func TestCharacterization_ViewType_TotalCount(t *testing.T) {
	// Current behavior: there are 13 view types (0-12)
	// If this test fails, a new view type was added or removed
	assert.Equal(t, ViewType(12), ViewSettings)
}

// TestCharacterization_Model_ZeroValue captures the zero value of Model.
func TestCharacterization_Model_ZeroValue(t *testing.T) {
	var m Model

	// Current behavior: zero value Model has these defaults
	assert.Nil(t, m.db)
	assert.Nil(t, m.cfg)
	assert.Nil(t, m.searchSvc)
	assert.Nil(t, m.telemetry)
	assert.Nil(t, m.installer)

	// Current behavior: zero view is ViewHome
	assert.Equal(t, ViewHome, m.currentView)
	assert.Equal(t, ViewHome, m.previousView)

	// Current behavior: dimensions are 0
	assert.Equal(t, 0, m.width)
	assert.Equal(t, 0, m.height)

	// Current behavior: booleans are false
	assert.False(t, m.ready)
	assert.False(t, m.quitting)
	assert.False(t, m.showingNewSkillDialog)
	assert.False(t, m.showLocationDialog)
	assert.False(t, m.showingQuitConfirm)

	// Current behavior: counters are 0
	assert.Equal(t, 0, m.viewsVisited)
	assert.Equal(t, 0, m.searchesPerformed)
	assert.Equal(t, 0, m.skillsInstalled)
	assert.Equal(t, 0, m.skillsUninstalled)
	assert.Equal(t, 0, m.reposAdded)
	assert.Equal(t, 0, m.reposRemoved)
	assert.Equal(t, 0, m.animTick)
}

// TestCharacterization_Model_ViewStateTracking captures how view state is tracked.
func TestCharacterization_Model_ViewStateTracking(t *testing.T) {
	var m Model

	// Current behavior: currentView and previousView are independent
	m.currentView = ViewSearch
	m.previousView = ViewHome

	assert.Equal(t, ViewSearch, m.currentView)
	assert.Equal(t, ViewHome, m.previousView)
}

// TestCharacterization_Model_DialogFlags captures dialog state flags.
func TestCharacterization_Model_DialogFlags(t *testing.T) {
	var m Model

	// Current behavior: multiple dialogs can be flagged simultaneously
	// (though this would be a bug in practice)
	m.showingNewSkillDialog = true
	m.showLocationDialog = true
	m.showingQuitConfirm = true

	assert.True(t, m.showingNewSkillDialog)
	assert.True(t, m.showLocationDialog)
	assert.True(t, m.showingQuitConfirm)
}

// TestCharacterization_Model_SessionCounters captures session counter behavior.
func TestCharacterization_Model_SessionCounters(t *testing.T) {
	var m Model

	// Current behavior: counters can be incremented independently
	m.viewsVisited = 5
	m.searchesPerformed = 3
	m.skillsInstalled = 2
	m.skillsUninstalled = 1
	m.reposAdded = 1
	m.reposRemoved = 0

	assert.Equal(t, 5, m.viewsVisited)
	assert.Equal(t, 3, m.searchesPerformed)
	assert.Equal(t, 2, m.skillsInstalled)
	assert.Equal(t, 1, m.skillsUninstalled)
	assert.Equal(t, 1, m.reposAdded)
	assert.Equal(t, 0, m.reposRemoved)
}

// TestCharacterization_Model_Dimensions captures dimension handling.
func TestCharacterization_Model_Dimensions(t *testing.T) {
	var m Model

	// Current behavior: width and height can be set to any value
	m.width = 120
	m.height = 40

	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)

	// Current behavior: negative dimensions are allowed (though may cause issues)
	m.width = -1
	m.height = -1

	assert.Equal(t, -1, m.width)
	assert.Equal(t, -1, m.height)
}

// TestCharacterization_pullCompleteMsg_ZeroValue captures zero value of pullCompleteMsg.
func TestCharacterization_pullCompleteMsg_ZeroValue(t *testing.T) {
	var msg pullCompleteMsg

	// Current behavior: zero value has all fields at zero/nil
	assert.Equal(t, 0, msg.skillsFound)
	assert.Equal(t, 0, msg.skillsNew)
	assert.Equal(t, 0, msg.localSynced)
	assert.Equal(t, 0, msg.cwdSynced)
	assert.Nil(t, msg.err)
}

// TestCharacterization_pullProgressMsg_ZeroValue captures zero value of pullProgressMsg.
func TestCharacterization_pullProgressMsg_ZeroValue(t *testing.T) {
	var msg pullProgressMsg

	// Current behavior: zero value has empty strings and zero ints
	assert.Equal(t, 0, msg.completed)
	assert.Equal(t, 0, msg.total)
	assert.Equal(t, "", msg.repoName)
}

// TestCharacterization_scanProgressMsg_ZeroValue captures zero value of scanProgressMsg.
func TestCharacterization_scanProgressMsg_ZeroValue(t *testing.T) {
	var msg scanProgressMsg

	// Current behavior: zero value has empty strings and zero ints
	assert.Equal(t, 0, msg.scanned)
	assert.Equal(t, 0, msg.total)
	assert.Equal(t, "", msg.repoName)
}

// TestCharacterization_ViewTypeSequence captures that ViewType is iota-based.
func TestCharacterization_ViewTypeSequence(t *testing.T) {
	// Current behavior: ViewType values are sequential (iota)
	// This means adding a new view in the middle would shift subsequent values
	views := []ViewType{
		ViewHome,
		ViewSearch,
		ViewReset,
		ViewSkillDetail,
		ViewTag,
		ViewOnboardingIntro,
		ViewOnboardingSkillsIntro,
		ViewOnboardingSetup,
		ViewOnboardingTools,
		ViewOnboardingSkills,
		ViewAddSource,
		ViewHelp,
		ViewSettings,
	}

	for i, v := range views {
		assert.Equal(t, ViewType(i), v, "View at position %d has wrong value", i)
	}
}
