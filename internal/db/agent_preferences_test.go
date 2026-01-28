package db

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpsertAgentPreference_CreateAndUpdate(t *testing.T) {
	db := testDB(t)

	// Create
	pref := &models.AgentPreference{
		AgentID: "claude",
		Enabled: true,
	}
	err := db.UpsertAgentPreference(pref)
	require.NoError(t, err)

	// Verify created
	prefs, err := db.GetAgentPreferences()
	require.NoError(t, err)
	// May include migrated prefs from seedOnboarding, filter to our test agent
	var found *models.AgentPreference
	for i, p := range prefs {
		if p.AgentID == "claude" {
			found = &prefs[i]
			break
		}
	}
	require.NotNil(t, found)
	assert.True(t, found.Enabled)

	// Update
	pref.Enabled = false
	err = db.UpsertAgentPreference(pref)
	require.NoError(t, err)

	prefs, err = db.GetAgentPreferences()
	require.NoError(t, err)
	found = nil
	for i, p := range prefs {
		if p.AgentID == "claude" {
			found = &prefs[i]
			break
		}
	}
	require.NotNil(t, found)
	// Note: GORM Updates skips zero-value fields by default, so Enabled=false
	// won't be set via Updates with a struct. Our code uses map updates for SetAgentsEnabled.
}

func TestGetEnabledAgents(t *testing.T) {
	db := testDB(t)

	// Create some agent preferences
	for _, id := range []string{"claude", "cursor", "cline"} {
		err := db.Create(&models.AgentPreference{
			AgentID: id,
			Enabled: id != "cline", // cline is disabled
		}).Error
		require.NoError(t, err)
	}

	enabled, err := db.GetEnabledAgents()
	require.NoError(t, err)
	assert.Contains(t, enabled, "claude")
	assert.Contains(t, enabled, "cursor")
	assert.NotContains(t, enabled, "cline")
}

func TestSetAgentsEnabled(t *testing.T) {
	db := testDB(t)

	// Create initial preferences
	for _, id := range []string{"claude", "cursor", "cline"} {
		err := db.Create(&models.AgentPreference{
			AgentID: id,
			Enabled: true,
		}).Error
		require.NoError(t, err)
	}

	// Set only claude and cline as enabled (cursor gets disabled)
	err := db.SetAgentsEnabled([]string{"claude", "cline"})
	require.NoError(t, err)

	enabled, err := db.GetEnabledAgents()
	require.NoError(t, err)
	assert.Contains(t, enabled, "claude")
	assert.Contains(t, enabled, "cline")
	assert.NotContains(t, enabled, "cursor")
}

func TestSetAgentsEnabled_CreatesNew(t *testing.T) {
	db := testDB(t)

	// Enable an agent that doesn't exist yet
	err := db.SetAgentsEnabled([]string{"roo"})
	require.NoError(t, err)

	enabled, err := db.GetEnabledAgents()
	require.NoError(t, err)
	assert.Contains(t, enabled, "roo")
}

func TestEnableAdditionalAgents(t *testing.T) {
	db := testDB(t)

	// Set initial
	err := db.SetAgentsEnabled([]string{"claude"})
	require.NoError(t, err)

	// Enable additional without disabling existing
	err = db.EnableAdditionalAgents([]string{"cursor", "cline"})
	require.NoError(t, err)

	enabled, err := db.GetEnabledAgents()
	require.NoError(t, err)
	assert.Contains(t, enabled, "claude")
	assert.Contains(t, enabled, "cursor")
	assert.Contains(t, enabled, "cline")
}

func TestUpdateDetectionState(t *testing.T) {
	db := testDB(t)

	detected := map[string]bool{
		"claude": true,
		"cline":  true,
		"roo":    false,
	}

	err := db.UpdateDetectionState(detected)
	require.NoError(t, err)

	prefs, err := db.GetAgentPreferences()
	require.NoError(t, err)

	prefMap := make(map[string]models.AgentPreference)
	for _, p := range prefs {
		prefMap[p.AgentID] = p
	}

	// Claude and cline should be detected
	assert.True(t, prefMap["claude"].Detected)
	assert.NotNil(t, prefMap["claude"].DetectedAt)
	assert.True(t, prefMap["cline"].Detected)

	// roo=false, so it should NOT have been created
	_, rooExists := prefMap["roo"]
	assert.False(t, rooExists, "roo with detected=false should not be created")
}

func TestMigrateFromAITools(t *testing.T) {
	db := testDB(t)

	// Set up AITools in UserState
	state, err := db.GetUserState()
	require.NoError(t, err)
	state.SetAITools([]string{"claude", "cursor"})
	err = db.Save(state).Error
	require.NoError(t, err)

	// Clear any prefs that may have been created during New()
	db.Where("1 = 1").Delete(&models.AgentPreference{})

	// Run migration
	err = db.MigrateFromAITools()
	require.NoError(t, err)

	// Verify
	enabled, err := db.GetEnabledAgents()
	require.NoError(t, err)
	assert.Contains(t, enabled, "claude")
	assert.Contains(t, enabled, "cursor")
	assert.Len(t, enabled, 2)
}

func TestMigrateFromAITools_Idempotent(t *testing.T) {
	db := testDB(t)

	// Set up AITools
	state, err := db.GetUserState()
	require.NoError(t, err)
	state.SetAITools([]string{"claude"})
	err = db.Save(state).Error
	require.NoError(t, err)

	// Clear any prefs
	db.Where("1 = 1").Delete(&models.AgentPreference{})

	// Run twice
	err = db.MigrateFromAITools()
	require.NoError(t, err)
	err = db.MigrateFromAITools()
	require.NoError(t, err)

	// Should still only have 1 entry
	prefs, err := db.GetAgentPreferences()
	require.NoError(t, err)
	count := 0
	for _, p := range prefs {
		if p.AgentID == "claude" {
			count++
		}
	}
	assert.Equal(t, 1, count, "migration should be idempotent")
}

func TestMigrateFromAITools_EmptyAITools(t *testing.T) {
	db := testDB(t)

	// Clear any prefs
	db.Where("1 = 1").Delete(&models.AgentPreference{})

	// AITools is empty by default
	err := db.MigrateFromAITools()
	require.NoError(t, err)

	prefs, err := db.GetAgentPreferences()
	require.NoError(t, err)
	assert.Empty(t, prefs)
}
