package db

import (
	"testing"
	"time"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- QuarantineSkill Tests ---

func TestQuarantineSkill(t *testing.T) {
	db := testDB(t)

	// Create a skill with PENDING status
	skill := &models.Skill{
		ID:             "quarantine-test-skill",
		Slug:           "quarantine-test",
		Title:          "Quarantine Test Skill",
		Content:        "test content",
		SecurityStatus: models.SecurityStatusPending,
	}
	require.NoError(t, db.CreateSkill(skill))

	// Quarantine the skill
	err := db.QuarantineSkill("quarantine-test-skill", models.ThreatLevelHigh, "Prompt injection detected")
	require.NoError(t, err)

	// Verify the skill was quarantined
	updated, err := db.GetSkill("quarantine-test-skill")
	require.NoError(t, err)
	assert.Equal(t, models.SecurityStatusQuarantined, updated.SecurityStatus)
	assert.Equal(t, models.ThreatLevelHigh, updated.ThreatLevel)
	assert.Equal(t, "Prompt injection detected", updated.ThreatSummary)
	assert.NotNil(t, updated.ScannedAt)
}

func TestQuarantineSkill_NonExistent(t *testing.T) {
	db := testDB(t)

	// Attempt to quarantine non-existent skill
	err := db.QuarantineSkill("non-existent", models.ThreatLevelHigh, "test")
	// Should not error - just updates 0 rows
	assert.NoError(t, err)
}

// --- ReleaseSkill Tests ---

func TestReleaseSkill(t *testing.T) {
	db := testDB(t)

	// Create a quarantined skill
	skill := &models.Skill{
		ID:             "release-test-skill",
		Slug:           "release-test",
		Title:          "Release Test Skill",
		Content:        "test content for hashing",
		SecurityStatus: models.SecurityStatusQuarantined,
		ThreatLevel:    models.ThreatLevelHigh,
		ThreatSummary:  "Previous threat",
	}
	require.NoError(t, db.CreateSkill(skill))

	// Release the skill
	err := db.ReleaseSkill("release-test-skill")
	require.NoError(t, err)

	// Verify the skill was released
	updated, err := db.GetSkill("release-test-skill")
	require.NoError(t, err)
	assert.Equal(t, models.SecurityStatusReleased, updated.SecurityStatus)
	assert.NotNil(t, updated.ReleasedAt)
	assert.NotEmpty(t, updated.ContentHash)
}

func TestReleaseSkill_NonExistent(t *testing.T) {
	db := testDB(t)

	// Attempt to release non-existent skill
	err := db.ReleaseSkill("non-existent")
	assert.Error(t, err) // Should error because First() fails
}

// --- MarkSkillClean Tests ---

func TestMarkSkillClean(t *testing.T) {
	db := testDB(t)

	// Create a pending skill
	skill := &models.Skill{
		ID:             "clean-test-skill",
		Slug:           "clean-test",
		Title:          "Clean Test Skill",
		Content:        "safe content",
		SecurityStatus: models.SecurityStatusPending,
	}
	require.NoError(t, db.CreateSkill(skill))

	// Mark as clean
	err := db.MarkSkillClean("clean-test-skill")
	require.NoError(t, err)

	// Verify the skill was marked clean
	updated, err := db.GetSkill("clean-test-skill")
	require.NoError(t, err)
	assert.Equal(t, models.SecurityStatusClean, updated.SecurityStatus)
	assert.Equal(t, models.ThreatLevelNone, updated.ThreatLevel)
	assert.Empty(t, updated.ThreatSummary)
	assert.NotNil(t, updated.ScannedAt)
	assert.NotEmpty(t, updated.ContentHash)
}

// --- GetQuarantinedSkills Tests ---

func TestGetQuarantinedSkills(t *testing.T) {
	db := testDB(t)

	// Create skills with various statuses
	skills := []models.Skill{
		{ID: "q1", Slug: "q1", Title: "Quarantined 1", SecurityStatus: models.SecurityStatusQuarantined, ThreatLevel: models.ThreatLevelHigh},
		{ID: "q2", Slug: "q2", Title: "Quarantined 2", SecurityStatus: models.SecurityStatusQuarantined, ThreatLevel: models.ThreatLevelCritical},
		{ID: "c1", Slug: "c1", Title: "Clean 1", SecurityStatus: models.SecurityStatusClean},
		{ID: "p1", Slug: "p1", Title: "Pending 1", SecurityStatus: models.SecurityStatusPending},
	}
	for i := range skills {
		require.NoError(t, db.CreateSkill(&skills[i]))
	}

	// Get quarantined skills
	quarantined, err := db.GetQuarantinedSkills()
	require.NoError(t, err)

	// Should only return quarantined skills, ordered by threat level DESC
	assert.Len(t, quarantined, 2)
	assert.Equal(t, "q2", quarantined[0].ID) // CRITICAL first
	assert.Equal(t, "q1", quarantined[1].ID) // HIGH second
}

// --- GetPendingSkills Tests ---

func TestGetPendingSkills(t *testing.T) {
	db := testDB(t)

	// Create skills with various statuses
	skills := []models.Skill{
		{ID: "pend1", Slug: "pend1", Title: "Pending 1", SecurityStatus: models.SecurityStatusPending},
		{ID: "pend2", Slug: "pend2", Title: "Pending 2", SecurityStatus: models.SecurityStatusPending},
		{ID: "clean1", Slug: "clean1", Title: "Clean 1", SecurityStatus: models.SecurityStatusClean},
	}
	for i := range skills {
		require.NoError(t, db.CreateSkill(&skills[i]))
	}

	// Get pending skills
	pending, err := db.GetPendingSkills()
	require.NoError(t, err)

	assert.Len(t, pending, 2)
}

// --- GetCleanSkills Tests ---

func TestGetCleanSkills(t *testing.T) {
	db := testDB(t)

	// Create skills with various statuses
	skills := []models.Skill{
		{ID: "cl1", Slug: "cl1", Title: "Clean 1", SecurityStatus: models.SecurityStatusClean},
		{ID: "cl2", Slug: "cl2", Title: "Clean 2", SecurityStatus: models.SecurityStatusClean},
		{ID: "pn1", Slug: "pn1", Title: "Pending 1", SecurityStatus: models.SecurityStatusPending},
	}
	for i := range skills {
		require.NoError(t, db.CreateSkill(&skills[i]))
	}

	// Get clean skills
	clean, err := db.GetCleanSkills()
	require.NoError(t, err)

	assert.Len(t, clean, 2)
}

// --- GetUsableSkills Tests ---

func TestGetUsableSkills(t *testing.T) {
	db := testDB(t)

	// Create skills with various statuses
	skills := []models.Skill{
		{ID: "us1", Slug: "us1", Title: "Clean", SecurityStatus: models.SecurityStatusClean},
		{ID: "us2", Slug: "us2", Title: "Released", SecurityStatus: models.SecurityStatusReleased},
		{ID: "us3", Slug: "us3", Title: "Pending", SecurityStatus: models.SecurityStatusPending},
		{ID: "us4", Slug: "us4", Title: "Quarantined", SecurityStatus: models.SecurityStatusQuarantined},
	}
	for i := range skills {
		require.NoError(t, db.CreateSkill(&skills[i]))
	}

	// Get usable skills (CLEAN or RELEASED only)
	usable, err := db.GetUsableSkills()
	require.NoError(t, err)

	assert.Len(t, usable, 2)
	for _, s := range usable {
		assert.True(t, s.SecurityStatus == models.SecurityStatusClean || s.SecurityStatus == models.SecurityStatusReleased)
	}
}

// --- CountBySecurityStatus Tests ---

func TestCountBySecurityStatus(t *testing.T) {
	db := testDB(t)

	// Create skills with various statuses
	skills := []models.Skill{
		{ID: "cnt1", Slug: "cnt1", SecurityStatus: models.SecurityStatusClean},
		{ID: "cnt2", Slug: "cnt2", SecurityStatus: models.SecurityStatusClean},
		{ID: "cnt3", Slug: "cnt3", SecurityStatus: models.SecurityStatusPending},
		{ID: "cnt4", Slug: "cnt4", SecurityStatus: models.SecurityStatusQuarantined},
	}
	for i := range skills {
		require.NoError(t, db.CreateSkill(&skills[i]))
	}

	// Get counts
	counts, err := db.CountBySecurityStatus()
	require.NoError(t, err)

	assert.Equal(t, int64(2), counts[models.SecurityStatusClean])
	assert.Equal(t, int64(1), counts[models.SecurityStatusPending])
	assert.Equal(t, int64(1), counts[models.SecurityStatusQuarantined])
}

// --- SecurityScan Audit Tests ---

func TestCreateSecurityScan(t *testing.T) {
	db := testDB(t)

	scan := &models.SecurityScan{
		ScanType:  models.ScanTypeFull,
		StartedAt: time.Now(),
	}

	err := db.CreateSecurityScan(scan)
	require.NoError(t, err)
	assert.NotZero(t, scan.ID)
}

func TestCompleteSecurityScan(t *testing.T) {
	db := testDB(t)

	// Create a scan
	scan := &models.SecurityScan{
		ScanType:  models.ScanTypeFull,
		StartedAt: time.Now(),
	}
	require.NoError(t, db.CreateSecurityScan(scan))

	// Complete the scan
	stats := models.SecurityScan{
		SkillsScanned:    10,
		FilesScanned:     25,
		ThreatsFound:     2,
		QuarantinedCount: 1,
		ScanSummary:      `{"details": "test"}`,
	}
	err := db.CompleteSecurityScan(scan.ID, stats)
	require.NoError(t, err)

	// Verify completion
	scans, err := db.GetRecentScans(1)
	require.NoError(t, err)
	require.Len(t, scans, 1)
	assert.NotNil(t, scans[0].CompletedAt)
	assert.Equal(t, 10, scans[0].SkillsScanned)
	assert.Equal(t, 25, scans[0].FilesScanned)
	assert.Equal(t, 2, scans[0].ThreatsFound)
}

func TestGetRecentScans(t *testing.T) {
	db := testDB(t)

	// Create multiple scans
	for i := 0; i < 5; i++ {
		scan := &models.SecurityScan{
			ScanType:  models.ScanTypeDelta,
			StartedAt: time.Now(),
		}
		require.NoError(t, db.CreateSecurityScan(scan))
	}

	// Get recent scans with limit
	scans, err := db.GetRecentScans(3)
	require.NoError(t, err)
	assert.Len(t, scans, 3)
}

// --- UpdateSkillSecurity Tests ---

func TestUpdateSkillSecurity(t *testing.T) {
	db := testDB(t)

	// Create a skill with PENDING status
	skill := &models.Skill{
		ID:             "update-security-test",
		Slug:           "update-security-test",
		Title:          "Update Security Test",
		Content:        "test content",
		SecurityStatus: models.SecurityStatusPending,
		ThreatLevel:    models.ThreatLevelNone,
	}
	require.NoError(t, db.CreateSkill(skill))

	// Update security fields
	now := time.Now()
	skill.SecurityStatus = models.SecurityStatusClean
	skill.ThreatLevel = models.ThreatLevelMedium
	skill.ThreatSummary = "Found potential issues"
	skill.ScannedAt = &now
	skill.ContentHash = skill.ComputeContentHash()

	err := db.UpdateSkillSecurity(skill)
	require.NoError(t, err)

	// Verify the update
	updated, err := db.GetSkill("update-security-test")
	require.NoError(t, err)
	assert.Equal(t, models.SecurityStatusClean, updated.SecurityStatus)
	assert.Equal(t, models.ThreatLevelMedium, updated.ThreatLevel)
	assert.Equal(t, "Found potential issues", updated.ThreatSummary)
	assert.NotNil(t, updated.ScannedAt)
	assert.NotEmpty(t, updated.ContentHash)
}

func TestUpdateSkillSecurity_ClearThreat(t *testing.T) {
	db := testDB(t)

	// Create a skill with existing threat
	skill := &models.Skill{
		ID:             "clear-threat-test",
		Slug:           "clear-threat-test",
		Title:          "Clear Threat Test",
		Content:        "safe content now",
		SecurityStatus: models.SecurityStatusClean,
		ThreatLevel:    models.ThreatLevelHigh,
		ThreatSummary:  "Previous threat",
	}
	require.NoError(t, db.CreateSkill(skill))

	// Update to clear threat
	now := time.Now()
	skill.ThreatLevel = models.ThreatLevelNone
	skill.ThreatSummary = ""
	skill.ScannedAt = &now
	skill.ContentHash = skill.ComputeContentHash()

	err := db.UpdateSkillSecurity(skill)
	require.NoError(t, err)

	// Verify the threat was cleared
	updated, err := db.GetSkill("clear-threat-test")
	require.NoError(t, err)
	assert.Equal(t, models.ThreatLevelNone, updated.ThreatLevel)
	assert.Empty(t, updated.ThreatSummary)
}

// --- CountSkillsWithWarnings Tests ---

func TestCountSkillsWithWarnings(t *testing.T) {
	db := testDB(t)

	// Create skills with various threat levels
	skills := []models.Skill{
		{ID: "warn1", Slug: "warn1", Title: "Warning 1", SecurityStatus: models.SecurityStatusClean, ThreatLevel: models.ThreatLevelLow},
		{ID: "warn2", Slug: "warn2", Title: "Warning 2", SecurityStatus: models.SecurityStatusClean, ThreatLevel: models.ThreatLevelMedium},
		{ID: "warn3", Slug: "warn3", Title: "Warning 3", SecurityStatus: models.SecurityStatusClean, ThreatLevel: models.ThreatLevelHigh},
		{ID: "clean1", Slug: "nwarn1", Title: "Clean 1", SecurityStatus: models.SecurityStatusClean, ThreatLevel: models.ThreatLevelNone},
		{ID: "clean2", Slug: "nwarn2", Title: "Clean 2", SecurityStatus: models.SecurityStatusClean, ThreatLevel: models.ThreatLevelNone},
	}
	for i := range skills {
		require.NoError(t, db.CreateSkill(&skills[i]))
	}

	// Count skills with warnings
	count, err := db.CountSkillsWithWarnings()
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestCountSkillsWithWarnings_Empty(t *testing.T) {
	db := testDB(t)

	// Create only clean skills
	skills := []models.Skill{
		{ID: "allclean1", Slug: "allclean1", SecurityStatus: models.SecurityStatusClean, ThreatLevel: models.ThreatLevelNone},
		{ID: "allclean2", Slug: "allclean2", SecurityStatus: models.SecurityStatusClean, ThreatLevel: models.ThreatLevelNone},
	}
	for i := range skills {
		require.NoError(t, db.CreateSkill(&skills[i]))
	}

	// Count should be 0
	count, err := db.CountSkillsWithWarnings()
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}
