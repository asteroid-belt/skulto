package db

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetRememberInstallLocations_TrueAndFalse(t *testing.T) {
	db := testDB(t)

	// Default should be false
	enabled, err := db.GetRememberInstallLocations()
	require.NoError(t, err)
	assert.False(t, enabled, "should default to false")

	// Set to true
	err = db.SetRememberInstallLocations(true)
	require.NoError(t, err)

	enabled, err = db.GetRememberInstallLocations()
	require.NoError(t, err)
	assert.True(t, enabled, "should be true after setting")

	// Set back to false
	err = db.SetRememberInstallLocations(false)
	require.NoError(t, err)

	enabled, err = db.GetRememberInstallLocations()
	require.NoError(t, err)
	assert.False(t, enabled, "should be false after clearing")
}

func TestGetRememberInstallLocations_DefaultFalse(t *testing.T) {
	db := testDB(t)

	// Fresh DB with no user state row should return false
	enabled, err := db.GetRememberInstallLocations()
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestRememberInstallLocations_UpsertDoesNotClobberOtherFields(t *testing.T) {
	db := testDB(t)

	// Set up initial state with onboarding and AI tools
	err := db.UpdateOnboardingStatus(models.OnboardingFinished)
	require.NoError(t, err)

	err = db.UpdateAITools("claude,cursor")
	require.NoError(t, err)

	// Now set remember install locations
	err = db.SetRememberInstallLocations(true)
	require.NoError(t, err)

	// Verify other fields are untouched
	state, err := db.GetUserState()
	require.NoError(t, err)
	assert.Equal(t, models.OnboardingFinished, state.OnboardingStatus, "onboarding status should be unchanged")
	assert.Equal(t, "claude,cursor", state.AITools, "AI tools should be unchanged")
	assert.True(t, state.RememberInstallLocations, "remember should be true")
}
