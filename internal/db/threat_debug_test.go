package db

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSkill_LoadsThreatLevel(t *testing.T) {
	db := testDB(t)

	// Create a skill with CRITICAL threat level
	skill := &models.Skill{
		ID:            "threat-test-001",
		Slug:          "threat-test",
		Title:         "Threat Test Skill",
		ThreatLevel:   models.ThreatLevelCritical,
		ThreatSummary: "Test threat summary",
	}
	require.NoError(t, db.CreateSkill(skill))

	// Retrieve and verify
	retrieved, err := db.GetSkill("threat-test-001")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	t.Logf("ThreatLevel: '%s'", retrieved.ThreatLevel)
	t.Logf("ThreatLevel == CRITICAL: %v", retrieved.ThreatLevel == models.ThreatLevelCritical)
	t.Logf("ThreatLevel == NONE: %v", retrieved.ThreatLevel == models.ThreatLevelNone)

	assert.Equal(t, models.ThreatLevelCritical, retrieved.ThreatLevel)
	assert.Equal(t, "Test threat summary", retrieved.ThreatSummary)
}
