// internal/discovery/ingestion_test.go
package discovery

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestionService_ValidateSkill_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "my-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.md"), []byte("# My Skill"), 0644))

	svc := NewIngestionService(nil, nil)
	err := svc.ValidateSkill(skillDir)

	assert.NoError(t, err)
}

func TestIngestionService_ValidateSkill_MissingSkillMd(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "my-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	// No skill.md file

	svc := NewIngestionService(nil, nil)
	err := svc.ValidateSkill(skillDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "skill.md")
}

func TestIngestionService_ValidateSkill_AcceptsSKILLMD(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "my-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# My Skill"), 0644))

	svc := NewIngestionService(nil, nil)
	err := svc.ValidateSkill(skillDir)

	assert.NoError(t, err)
}

func TestIngestionService_CheckNameConflict_NoConflict(t *testing.T) {
	tmpDir := t.TempDir()
	svc := &IngestionService{destDirOverride: tmpDir}

	conflict, err := svc.CheckNameConflict("nonexistent", "project")

	assert.NoError(t, err)
	assert.False(t, conflict)
}

func TestIngestionService_CheckNameConflict_HasConflict(t *testing.T) {
	tmpDir := t.TempDir()
	existingDir := filepath.Join(tmpDir, "existing-skill")
	require.NoError(t, os.MkdirAll(existingDir, 0755))

	svc := &IngestionService{destDirOverride: tmpDir}

	conflict, err := svc.CheckNameConflict("existing-skill", "project")

	assert.NoError(t, err)
	assert.True(t, conflict)
}

func TestIngestionService_IngestSkill(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source skill
	sourceDir := filepath.Join(tmpDir, "platforms", ".claude", "skills", "my-skill")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "skill.md"), []byte("# My Skill"), 0644))

	// Create skulto destination directory
	skultoDir := filepath.Join(tmpDir, ".skulto", "skills")
	require.NoError(t, os.MkdirAll(skultoDir, 0755))

	ds := models.DiscoveredSkill{
		Platform: "claude",
		Scope:    "project",
		Path:     sourceDir,
		Name:     "my-skill",
	}
	ds.ID = ds.GenerateID()

	svc := &IngestionService{destDirOverride: skultoDir}
	result, err := svc.IngestSkill(context.Background(), &ds)

	require.NoError(t, err)
	assert.Equal(t, "my-skill", result.Name)
	assert.Equal(t, filepath.Join(skultoDir, "my-skill"), result.DestPath)

	// Verify skill was copied
	_, err = os.Stat(filepath.Join(skultoDir, "my-skill", "skill.md"))
	assert.NoError(t, err)

	// Verify original is now a symlink
	lstat, err := os.Lstat(sourceDir)
	require.NoError(t, err)
	assert.True(t, lstat.Mode()&os.ModeSymlink != 0)
}

func TestIngestionService_IngestSkill_WithMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source skill with multiple files
	sourceDir := filepath.Join(tmpDir, "platforms", ".claude", "skills", "complex-skill")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "skill.md"), []byte("# Complex Skill"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "helper.py"), []byte("print('hello')"), 0644))

	// Create subdirectory with files
	subDir := filepath.Join(sourceDir, "templates")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "template.txt"), []byte("template content"), 0644))

	skultoDir := filepath.Join(tmpDir, ".skulto", "skills")
	require.NoError(t, os.MkdirAll(skultoDir, 0755))

	ds := models.DiscoveredSkill{
		Platform: "claude",
		Scope:    "project",
		Path:     sourceDir,
		Name:     "complex-skill",
	}
	ds.ID = ds.GenerateID()

	svc := &IngestionService{destDirOverride: skultoDir}
	result, err := svc.IngestSkill(context.Background(), &ds)

	require.NoError(t, err)
	assert.Equal(t, "complex-skill", result.Name)

	// Verify all files were copied
	_, err = os.Stat(filepath.Join(skultoDir, "complex-skill", "skill.md"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(skultoDir, "complex-skill", "helper.py"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(skultoDir, "complex-skill", "templates", "template.txt"))
	assert.NoError(t, err)
}

func TestIngestionService_IngestSkill_ValidationFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source skill without skill.md
	sourceDir := filepath.Join(tmpDir, "platforms", ".claude", "skills", "invalid-skill")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	// No skill.md file

	skultoDir := filepath.Join(tmpDir, ".skulto", "skills")
	require.NoError(t, os.MkdirAll(skultoDir, 0755))

	ds := models.DiscoveredSkill{
		Platform: "claude",
		Scope:    "project",
		Path:     sourceDir,
		Name:     "invalid-skill",
	}
	ds.ID = ds.GenerateID()

	svc := &IngestionService{destDirOverride: skultoDir}
	result, err := svc.IngestSkill(context.Background(), &ds)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "skill.md")
}

func TestIngestionService_IngestSkill_SymlinkPointsToDestination(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source skill
	sourceDir := filepath.Join(tmpDir, "platforms", ".claude", "skills", "my-skill")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "skill.md"), []byte("# My Skill"), 0644))

	skultoDir := filepath.Join(tmpDir, ".skulto", "skills")
	require.NoError(t, os.MkdirAll(skultoDir, 0755))

	ds := models.DiscoveredSkill{
		Platform: "claude",
		Scope:    "project",
		Path:     sourceDir,
		Name:     "my-skill",
	}
	ds.ID = ds.GenerateID()

	svc := &IngestionService{destDirOverride: skultoDir}
	_, err := svc.IngestSkill(context.Background(), &ds)
	require.NoError(t, err)

	// Read symlink target
	target, err := os.Readlink(sourceDir)
	require.NoError(t, err)

	// Resolve relative target to absolute
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(sourceDir), target)
	}
	target = filepath.Clean(target)

	// Should point to skulto destination
	expectedDest := filepath.Join(skultoDir, "my-skill")
	assert.Equal(t, expectedDest, target)
}

// testDB creates a temporary test database for ingestion tests.
func testDB(t *testing.T) *db.DB {
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

	t.Cleanup(func() {
		_ = database.Close()
	})

	return database
}

func TestIngestionService_IngestSkill_CreatesSkillRecord(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source skill with proper skill.md content
	sourceDir := filepath.Join(tmpDir, "platforms", ".claude", "skills", "test-skill")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))

	skillContent := `---
name: Test Skill
description: A test skill for Python development
---

# Test Skill

This skill helps with Python development and testing.
`
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "skill.md"), []byte(skillContent), 0644))

	skultoDir := filepath.Join(tmpDir, ".skulto", "skills")
	require.NoError(t, os.MkdirAll(skultoDir, 0755))

	// Create test database
	database := testDB(t)

	ds := models.DiscoveredSkill{
		Platform: "claude",
		Scope:    "project",
		Path:     sourceDir,
		Name:     "test-skill",
	}
	ds.ID = ds.GenerateID()

	svc := &IngestionService{
		db:              database,
		destDirOverride: skultoDir,
	}
	result, err := svc.IngestSkill(context.Background(), &ds)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Assert: Skill record was returned in result
	assert.NotNil(t, result.Skill, "IngestionResult should include Skill")
	assert.True(t, result.Skill.IsLocal, "Skill should be marked as local")
	assert.True(t, result.Skill.IsInstalled, "Skill should be marked as installed")
	assert.Equal(t, "Test Skill", result.Skill.Title)
	assert.Contains(t, result.Skill.Description, "test skill")

	// Assert: Skill record exists in database
	skill, err := database.GetSkill(result.Skill.ID)
	require.NoError(t, err)
	require.NotNil(t, skill, "Skill should exist in database")
	assert.True(t, skill.IsLocal)
	assert.Equal(t, "Test Skill", skill.Title)
}

func TestIngestionService_IngestSkill_CreatesInstallationRecord(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source skill
	sourceDir := filepath.Join(tmpDir, "platforms", ".claude", "skills", "install-test")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))

	skillContent := `# Installation Test Skill

This skill tests installation record creation.
`
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "skill.md"), []byte(skillContent), 0644))

	skultoDir := filepath.Join(tmpDir, ".skulto", "skills")
	require.NoError(t, os.MkdirAll(skultoDir, 0755))

	// Create test database
	database := testDB(t)

	ds := models.DiscoveredSkill{
		Platform: "claude",
		Scope:    "project",
		Path:     sourceDir,
		Name:     "install-test",
	}
	ds.ID = ds.GenerateID()

	svc := &IngestionService{
		db:              database,
		destDirOverride: skultoDir,
	}
	result, err := svc.IngestSkill(context.Background(), &ds)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Skill)

	// Assert: SkillInstallation record exists
	installations, err := database.GetInstallations(result.Skill.ID)
	require.NoError(t, err)
	require.Len(t, installations, 1, "Should have exactly one installation record")

	installation := installations[0]
	assert.Equal(t, "claude", installation.Platform)
	assert.Equal(t, "project", installation.Scope)
	assert.Equal(t, sourceDir, installation.SymlinkPath, "SymlinkPath should be original discovered path")
	assert.Equal(t, filepath.Join(skultoDir, "install-test"), installation.BasePath)
}

func TestIngestionService_IngestSkill_ExtractsTags(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source skill with content that should extract tags
	sourceDir := filepath.Join(tmpDir, "platforms", ".cursor", "skills", "python-skill")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))

	skillContent := `# Python Testing Skill

This skill helps with Python testing using pytest.
Great for test-driven development workflows.
`
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "skill.md"), []byte(skillContent), 0644))

	skultoDir := filepath.Join(tmpDir, ".skulto", "skills")
	require.NoError(t, os.MkdirAll(skultoDir, 0755))

	database := testDB(t)

	ds := models.DiscoveredSkill{
		Platform: "cursor",
		Scope:    "global",
		Path:     sourceDir,
		Name:     "python-skill",
	}
	ds.ID = ds.GenerateID()

	svc := &IngestionService{
		db:              database,
		destDirOverride: skultoDir,
	}
	result, err := svc.IngestSkill(context.Background(), &ds)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Skill)

	// Assert: Skill has tags (Python should be detected)
	skill, err := database.GetSkill(result.Skill.ID)
	require.NoError(t, err)
	require.NotNil(t, skill)

	// Check that at least Python tag was extracted
	tagNames := make([]string, len(skill.Tags))
	for i, tag := range skill.Tags {
		tagNames[i] = tag.Name
	}
	assert.Contains(t, tagNames, "python", "Should extract python tag from content")
}

func TestGenerateLocalSkillID(t *testing.T) {
	// Test that generateLocalSkillID produces consistent, deterministic IDs
	path1 := "/home/user/.skulto/skills/my-skill"
	path2 := "/home/user/.skulto/skills/other-skill"

	id1a := generateLocalSkillID(path1)
	id1b := generateLocalSkillID(path1)
	id2 := generateLocalSkillID(path2)

	// Same path should produce same ID
	assert.Equal(t, id1a, id1b, "Same path should produce same ID")

	// Different paths should produce different IDs
	assert.NotEqual(t, id1a, id2, "Different paths should produce different IDs")

	// ID should have reasonable length (hex-encoded hash prefix)
	assert.Len(t, id1a, 16, "ID should be 16 characters (truncated hash)")
}
