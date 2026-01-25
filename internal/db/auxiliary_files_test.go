package db

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- UpsertAuxiliaryFile Tests ---

func TestUpsertAuxiliaryFile_Create(t *testing.T) {
	db := testDB(t)

	// Create a skill first
	skill := &models.Skill{ID: "aux-skill-1", Slug: "aux-skill-1", Title: "Test Skill"}
	require.NoError(t, db.CreateSkill(skill))

	// Create auxiliary file
	file := &models.AuxiliaryFile{
		SkillID:  "aux-skill-1",
		DirType:  models.AuxDirScripts,
		FilePath: "scripts/setup.sh",
		FileName: "setup.sh",
		FileSize: 1024,
	}

	err := db.UpsertAuxiliaryFile(file)
	require.NoError(t, err)
	assert.NotEmpty(t, file.ID) // ID should be generated
}

func TestUpsertAuxiliaryFile_Update(t *testing.T) {
	db := testDB(t)

	// Create a skill first
	skill := &models.Skill{ID: "aux-skill-2", Slug: "aux-skill-2", Title: "Test Skill"}
	require.NoError(t, db.CreateSkill(skill))

	// Create auxiliary file
	file := &models.AuxiliaryFile{
		SkillID:  "aux-skill-2",
		DirType:  models.AuxDirScripts,
		FilePath: "scripts/run.sh",
		FileName: "run.sh",
		FileSize: 512,
	}
	require.NoError(t, db.UpsertAuxiliaryFile(file))
	originalID := file.ID

	// Update the file (same skill/dirtype/filepath = same ID)
	file.FileSize = 1024
	file.ContentHash = "abc123"
	err := db.UpsertAuxiliaryFile(file)
	require.NoError(t, err)

	// Should have same ID (upsert, not insert)
	assert.Equal(t, originalID, file.ID)

	// Verify update
	files, err := db.GetAuxiliaryFilesForSkill("aux-skill-2")
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, int64(1024), files[0].FileSize)
	assert.Equal(t, "abc123", files[0].ContentHash)
}

// --- GetAuxiliaryFilesForSkill Tests ---

func TestGetAuxiliaryFilesForSkill(t *testing.T) {
	db := testDB(t)

	// Create a skill
	skill := &models.Skill{ID: "aux-skill-3", Slug: "aux-skill-3", Title: "Test Skill"}
	require.NoError(t, db.CreateSkill(skill))

	// Create multiple auxiliary files
	files := []models.AuxiliaryFile{
		{SkillID: "aux-skill-3", DirType: models.AuxDirScripts, FilePath: "scripts/a.sh", FileName: "a.sh"},
		{SkillID: "aux-skill-3", DirType: models.AuxDirReferences, FilePath: "references/doc.md", FileName: "doc.md"},
		{SkillID: "aux-skill-3", DirType: models.AuxDirScripts, FilePath: "scripts/b.sh", FileName: "b.sh"},
	}
	for i := range files {
		require.NoError(t, db.UpsertAuxiliaryFile(&files[i]))
	}

	// Get all files for skill
	result, err := db.GetAuxiliaryFilesForSkill("aux-skill-3")
	require.NoError(t, err)
	assert.Len(t, result, 3)
}

func TestGetAuxiliaryFilesForSkill_Empty(t *testing.T) {
	db := testDB(t)

	// Create a skill with no files
	skill := &models.Skill{ID: "aux-skill-empty", Slug: "aux-skill-empty", Title: "Test Skill"}
	require.NoError(t, db.CreateSkill(skill))

	// Get files (should be empty)
	result, err := db.GetAuxiliaryFilesForSkill("aux-skill-empty")
	require.NoError(t, err)
	assert.Len(t, result, 0)
}

// --- GetAuxiliaryFilesByDirType Tests ---

func TestGetAuxiliaryFilesByDirType(t *testing.T) {
	db := testDB(t)

	// Create a skill
	skill := &models.Skill{ID: "aux-skill-4", Slug: "aux-skill-4", Title: "Test Skill"}
	require.NoError(t, db.CreateSkill(skill))

	// Create files in different directories
	files := []models.AuxiliaryFile{
		{SkillID: "aux-skill-4", DirType: models.AuxDirScripts, FilePath: "scripts/a.sh", FileName: "a.sh"},
		{SkillID: "aux-skill-4", DirType: models.AuxDirReferences, FilePath: "references/doc.md", FileName: "doc.md"},
		{SkillID: "aux-skill-4", DirType: models.AuxDirScripts, FilePath: "scripts/b.sh", FileName: "b.sh"},
	}
	for i := range files {
		require.NoError(t, db.UpsertAuxiliaryFile(&files[i]))
	}

	// Get only scripts
	scripts, err := db.GetAuxiliaryFilesByDirType("aux-skill-4", models.AuxDirScripts)
	require.NoError(t, err)
	assert.Len(t, scripts, 2)

	// Get only references
	refs, err := db.GetAuxiliaryFilesByDirType("aux-skill-4", models.AuxDirReferences)
	require.NoError(t, err)
	assert.Len(t, refs, 1)
}

// --- GetQuarantinedFiles Tests ---

func TestGetQuarantinedFiles(t *testing.T) {
	db := testDB(t)

	// Create a skill
	skill := &models.Skill{ID: "aux-skill-5", Slug: "aux-skill-5", Title: "Test Skill"}
	require.NoError(t, db.CreateSkill(skill))

	// Create files with various statuses
	files := []models.AuxiliaryFile{
		{SkillID: "aux-skill-5", DirType: models.AuxDirScripts, FilePath: "scripts/safe.sh", FileName: "safe.sh", SecurityStatus: models.SecurityStatusClean},
		{SkillID: "aux-skill-5", DirType: models.AuxDirScripts, FilePath: "scripts/bad.sh", FileName: "bad.sh", SecurityStatus: models.SecurityStatusQuarantined, ThreatLevel: models.ThreatLevelHigh},
		{SkillID: "aux-skill-5", DirType: models.AuxDirScripts, FilePath: "scripts/worse.sh", FileName: "worse.sh", SecurityStatus: models.SecurityStatusQuarantined, ThreatLevel: models.ThreatLevelCritical},
	}
	for i := range files {
		require.NoError(t, db.UpsertAuxiliaryFile(&files[i]))
	}

	// Get quarantined files
	quarantined, err := db.GetQuarantinedFiles()
	require.NoError(t, err)
	assert.Len(t, quarantined, 2)
	// Should be ordered by threat level DESC
	assert.Equal(t, models.ThreatLevelCritical, quarantined[0].ThreatLevel)
	assert.Equal(t, models.ThreatLevelHigh, quarantined[1].ThreatLevel)
}

// --- QuarantineFile Tests ---

func TestQuarantineFile(t *testing.T) {
	db := testDB(t)

	// Create skill and file
	skill := &models.Skill{ID: "aux-skill-6", Slug: "aux-skill-6", Title: "Test Skill"}
	require.NoError(t, db.CreateSkill(skill))

	file := &models.AuxiliaryFile{
		SkillID:        "aux-skill-6",
		DirType:        models.AuxDirScripts,
		FilePath:       "scripts/suspect.sh",
		FileName:       "suspect.sh",
		SecurityStatus: models.SecurityStatusPending,
	}
	require.NoError(t, db.UpsertAuxiliaryFile(file))

	// Quarantine the file
	err := db.QuarantineFile(file.ID, models.ThreatLevelHigh, "Malicious code detected")
	require.NoError(t, err)

	// Verify
	files, err := db.GetAuxiliaryFilesForSkill("aux-skill-6")
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, models.SecurityStatusQuarantined, files[0].SecurityStatus)
	assert.Equal(t, models.ThreatLevelHigh, files[0].ThreatLevel)
	assert.Equal(t, "Malicious code detected", files[0].ThreatSummary)
}

// --- ReleaseFile Tests ---

func TestReleaseFile(t *testing.T) {
	db := testDB(t)

	// Create skill and quarantined file
	skill := &models.Skill{ID: "aux-skill-7", Slug: "aux-skill-7", Title: "Test Skill"}
	require.NoError(t, db.CreateSkill(skill))

	file := &models.AuxiliaryFile{
		SkillID:        "aux-skill-7",
		DirType:        models.AuxDirScripts,
		FilePath:       "scripts/approved.sh",
		FileName:       "approved.sh",
		SecurityStatus: models.SecurityStatusQuarantined,
	}
	require.NoError(t, db.UpsertAuxiliaryFile(file))

	// Release the file
	err := db.ReleaseFile(file.ID)
	require.NoError(t, err)

	// Verify
	files, err := db.GetAuxiliaryFilesForSkill("aux-skill-7")
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, models.SecurityStatusReleased, files[0].SecurityStatus)
	assert.NotNil(t, files[0].ReleasedAt)
}

// --- MarkFileClean Tests ---

func TestMarkFileClean(t *testing.T) {
	db := testDB(t)

	// Create skill and pending file
	skill := &models.Skill{ID: "aux-skill-8", Slug: "aux-skill-8", Title: "Test Skill"}
	require.NoError(t, db.CreateSkill(skill))

	file := &models.AuxiliaryFile{
		SkillID:        "aux-skill-8",
		DirType:        models.AuxDirScripts,
		FilePath:       "scripts/clean.sh",
		FileName:       "clean.sh",
		SecurityStatus: models.SecurityStatusPending,
	}
	require.NoError(t, db.UpsertAuxiliaryFile(file))

	// Mark as clean
	err := db.MarkFileClean(file.ID)
	require.NoError(t, err)

	// Verify
	files, err := db.GetAuxiliaryFilesForSkill("aux-skill-8")
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, models.SecurityStatusClean, files[0].SecurityStatus)
	assert.Equal(t, models.ThreatLevelNone, files[0].ThreatLevel)
}

// --- DeleteAuxiliaryFilesForSkill Tests ---

func TestDeleteAuxiliaryFilesForSkill(t *testing.T) {
	db := testDB(t)

	// Create skill and files
	skill := &models.Skill{ID: "aux-skill-9", Slug: "aux-skill-9", Title: "Test Skill"}
	require.NoError(t, db.CreateSkill(skill))

	for i := 0; i < 3; i++ {
		file := &models.AuxiliaryFile{
			SkillID:  "aux-skill-9",
			DirType:  models.AuxDirScripts,
			FilePath: "scripts/file" + string(rune('a'+i)) + ".sh",
			FileName: "file" + string(rune('a'+i)) + ".sh",
		}
		require.NoError(t, db.UpsertAuxiliaryFile(file))
	}

	// Verify files exist
	files, _ := db.GetAuxiliaryFilesForSkill("aux-skill-9")
	assert.Len(t, files, 3)

	// Soft delete all files
	err := db.DeleteAuxiliaryFilesForSkill("aux-skill-9")
	require.NoError(t, err)

	// Verify files are gone (soft deleted)
	files, _ = db.GetAuxiliaryFilesForSkill("aux-skill-9")
	assert.Len(t, files, 0)
}

// --- HardDeleteAuxiliaryFilesForSkill Tests ---

func TestHardDeleteAuxiliaryFilesForSkill(t *testing.T) {
	db := testDB(t)

	// Create skill and files
	skill := &models.Skill{ID: "aux-skill-10", Slug: "aux-skill-10", Title: "Test Skill"}
	require.NoError(t, db.CreateSkill(skill))

	file := &models.AuxiliaryFile{
		SkillID:  "aux-skill-10",
		DirType:  models.AuxDirScripts,
		FilePath: "scripts/temp.sh",
		FileName: "temp.sh",
	}
	require.NoError(t, db.UpsertAuxiliaryFile(file))

	// Hard delete
	err := db.HardDeleteAuxiliaryFilesForSkill("aux-skill-10")
	require.NoError(t, err)

	// Verify completely gone (even with unscoped)
	var count int64
	db.Unscoped().Model(&models.AuxiliaryFile{}).Where("skill_id = ?", "aux-skill-10").Count(&count)
	assert.Equal(t, int64(0), count)
}
