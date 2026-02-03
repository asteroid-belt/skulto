// internal/discovery/integration_test.go
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

// setupTestDB creates a temporary test database for integration tests.
func setupTestDB(t *testing.T) *db.DB {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "integration_test.db")

	database, err := db.New(db.Config{
		Path:        dbPath,
		Debug:       false,
		MaxIdleConn: 1,
		MaxOpenConn: 1,
	})
	require.NoError(t, err, "failed to create test database")

	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Logf("Failed to close test database: %v", err)
		}
	})

	return database
}

// TestIntegration_ScanToIngest tests the full workflow:
// 1. Create unmanaged skill directory
// 2. Scan to discover it
// 3. Store in database
// 4. Ingest to create symlink
// 5. Verify final state
func TestIntegration_ScanToIngest(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup: Create platform skill directory with skill.md
	skillDir := filepath.Join(tmpDir, ".claude", "skills", "my-test-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(skillDir, "skill.md"),
		[]byte("# My Test Skill\n\nTest content"),
		0644,
	))

	// Setup: Create skulto destination
	skultoDir := filepath.Join(tmpDir, ".skulto", "skills")
	require.NoError(t, os.MkdirAll(skultoDir, 0755))

	// Step 1: Scan for unmanaged skills
	scanner := NewScannerService()
	discoveries, err := scanner.ScanDirectory(
		filepath.Join(tmpDir, ".claude", "skills"),
		"claude",
		"project",
	)
	require.NoError(t, err)
	require.Len(t, discoveries, 1)
	assert.Equal(t, "my-test-skill", discoveries[0].Name)

	// Step 2: Store in database
	database := setupTestDB(t)
	for _, d := range discoveries {
		require.NoError(t, database.UpsertDiscoveredSkill(&d))
	}

	// Verify stored
	stored, err := database.ListDiscoveredSkills()
	require.NoError(t, err)
	require.Len(t, stored, 1)

	// Step 3: Ingest the skill
	ingestionSvc := &IngestionService{destDirOverride: skultoDir}
	result, err := ingestionSvc.IngestSkill(context.Background(), &stored[0])
	require.NoError(t, err)
	assert.Equal(t, "my-test-skill", result.Name)

	// Step 4: Verify final state
	// - Skill is copied to skulto dir
	_, err = os.Stat(filepath.Join(skultoDir, "my-test-skill", "skill.md"))
	assert.NoError(t, err, "skill.md should exist in destination")

	// - Original location is now a symlink
	lstat, err := os.Lstat(skillDir)
	require.NoError(t, err)
	assert.True(t, lstat.Mode()&os.ModeSymlink != 0, "original should be symlink")

	// - Symlink points to correct location
	target, err := os.Readlink(skillDir)
	require.NoError(t, err)

	// Resolve relative target to absolute
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(skillDir), target)
	}
	target = filepath.Clean(target)
	assert.Equal(t, filepath.Join(skultoDir, "my-test-skill"), target)
}

// TestIntegration_RescanAfterIngest verifies that ingested skills
// no longer appear as "discovered" (they're now symlinks)
func TestIntegration_RescanAfterIngest(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup skill and ingest it
	skillDir := filepath.Join(tmpDir, ".claude", "skills", "ingested-skill")
	skultoDir := filepath.Join(tmpDir, ".skulto", "skills")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	require.NoError(t, os.MkdirAll(skultoDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.md"), []byte("# Test"), 0644))

	// Initial scan
	scanner := NewScannerService()
	discoveries, err := scanner.ScanDirectory(filepath.Join(tmpDir, ".claude", "skills"), "claude", "project")
	require.NoError(t, err)
	require.Len(t, discoveries, 1)

	// Ingest
	svc := &IngestionService{destDirOverride: skultoDir}
	ds := discoveries[0]
	_, err = svc.IngestSkill(context.Background(), &ds)
	require.NoError(t, err)

	// Re-scan should find 0 discoveries (original is now a symlink)
	discoveries2, err := scanner.ScanDirectory(filepath.Join(tmpDir, ".claude", "skills"), "claude", "project")
	require.NoError(t, err)
	assert.Len(t, discoveries2, 0, "ingested skill should not appear as discovered")
}

// TestIntegration_MultiPlatformScan tests scanning multiple platforms
func TestIntegration_MultiPlatformScan(t *testing.T) {
	tmpDir := t.TempDir()

	// Create skills in multiple platform directories
	platforms := []struct {
		platform string
		name     string
	}{
		{"claude", "claude-skill"},
		{"cursor", "cursor-skill"},
	}

	for _, p := range platforms {
		skillDir := filepath.Join(tmpDir, "."+p.platform, "skills", p.name)
		require.NoError(t, os.MkdirAll(skillDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.md"), []byte("# "+p.name), 0644))
	}

	scanner := NewScannerService()

	// Scan all platforms
	configs := []PlatformConfig{
		{ID: "claude", SkillsPath: filepath.Join(tmpDir, ".claude", "skills")},
		{ID: "cursor", SkillsPath: filepath.Join(tmpDir, ".cursor", "skills")},
	}

	discoveries, err := scanner.ScanPlatforms(configs, "project")
	require.NoError(t, err)
	assert.Len(t, discoveries, 2)

	// Verify both platforms found
	names := map[string]bool{}
	for _, d := range discoveries {
		names[d.Name] = true
	}
	assert.True(t, names["claude-skill"])
	assert.True(t, names["cursor-skill"])
}

// TestIntegration_DatabasePersistence tests that discoveries persist across DB operations
func TestIntegration_DatabasePersistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create skill
	skillDir := filepath.Join(tmpDir, ".claude", "skills", "persistent-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.md"), []byte("# Persistent"), 0644))

	// Scan
	scanner := NewScannerService()
	discoveries, err := scanner.ScanDirectory(filepath.Join(tmpDir, ".claude", "skills"), "claude", "project")
	require.NoError(t, err)
	require.Len(t, discoveries, 1)

	// Store in database
	database := setupTestDB(t)
	for _, d := range discoveries {
		require.NoError(t, database.UpsertDiscoveredSkill(&d))
	}

	// Query by path
	found, err := database.GetDiscoveredSkillByPath(skillDir)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "persistent-skill", found.Name)

	// Query by name and scope
	found2, err := database.GetDiscoveredSkillByName("persistent-skill", "project")
	require.NoError(t, err)
	assert.NotNil(t, found2)
	assert.Equal(t, skillDir, found2.Path)

	// Count
	count, err := database.CountDiscoveredSkills()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

// TestIntegration_ValidationFailure tests that ingestion fails without skill.md
func TestIntegration_ValidationFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create skill without skill.md
	skillDir := filepath.Join(tmpDir, ".claude", "skills", "invalid-skill")
	skultoDir := filepath.Join(tmpDir, ".skulto", "skills")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	require.NoError(t, os.MkdirAll(skultoDir, 0755))
	// No skill.md file created

	// The scanner will find it (it doesn't validate skill.md)
	scanner := NewScannerService()
	discoveries, err := scanner.ScanDirectory(filepath.Join(tmpDir, ".claude", "skills"), "claude", "project")
	require.NoError(t, err)
	require.Len(t, discoveries, 1)

	// But ingestion should fail
	svc := &IngestionService{destDirOverride: skultoDir}
	ds := discoveries[0]
	result, err := svc.IngestSkill(context.Background(), &ds)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "skill.md")
}

// TestIntegration_NameConflictCheck tests conflict detection
func TestIntegration_NameConflictCheck(t *testing.T) {
	tmpDir := t.TempDir()

	// Create skulto destination with existing skill
	skultoDir := filepath.Join(tmpDir, ".skulto", "skills")
	existingSkillDir := filepath.Join(skultoDir, "existing-skill")
	require.NoError(t, os.MkdirAll(existingSkillDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(existingSkillDir, "skill.md"), []byte("# Existing"), 0644))

	svc := &IngestionService{destDirOverride: skultoDir}

	// Should detect conflict
	hasConflict, err := svc.CheckNameConflict("existing-skill", "project")
	require.NoError(t, err)
	assert.True(t, hasConflict)

	// Should not detect conflict for new name
	hasConflict, err = svc.CheckNameConflict("new-skill", "project")
	require.NoError(t, err)
	assert.False(t, hasConflict)
}

// TestIntegration_CleanupStaleDiscoveries tests cleanup of stale entries
func TestIntegration_CleanupStaleDiscoveries(t *testing.T) {
	tmpDir := t.TempDir()

	// Create and store two discoveries
	database := setupTestDB(t)

	// First skill exists
	existingSkillDir := filepath.Join(tmpDir, ".claude", "skills", "existing-skill")
	require.NoError(t, os.MkdirAll(existingSkillDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(existingSkillDir, "skill.md"), []byte("# Existing"), 0644))

	existingSkill := models.DiscoveredSkill{
		Platform: "claude",
		Scope:    "project",
		Path:     existingSkillDir,
		Name:     "existing-skill",
	}
	existingSkill.ID = existingSkill.GenerateID()
	require.NoError(t, database.UpsertDiscoveredSkill(&existingSkill))

	// Second skill path no longer exists
	staleSkill := models.DiscoveredSkill{
		Platform: "claude",
		Scope:    "project",
		Path:     filepath.Join(tmpDir, ".claude", "skills", "removed-skill"),
		Name:     "removed-skill",
	}
	staleSkill.ID = staleSkill.GenerateID()
	require.NoError(t, database.UpsertDiscoveredSkill(&staleSkill))

	// Verify we have 2 discoveries
	count, err := database.CountDiscoveredSkills()
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Cleanup with path checker that verifies paths still exist as non-symlinks
	pathChecker := func(path string) bool {
		info, err := os.Lstat(path)
		if err != nil {
			return false
		}
		// Must exist and NOT be a symlink
		return info.Mode()&os.ModeSymlink == 0
	}

	err = database.CleanupStaleDiscoveries(pathChecker)
	require.NoError(t, err)

	// Should only have 1 discovery left
	count, err = database.CountDiscoveredSkills()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Verify it's the existing one
	remaining, err := database.ListDiscoveredSkills()
	require.NoError(t, err)
	assert.Equal(t, "existing-skill", remaining[0].Name)
}

// TestIntegration_IngestWithDatabaseCleanup tests that ingestion removes from DB
func TestIntegration_IngestWithDatabaseCleanup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create skill
	skillDir := filepath.Join(tmpDir, ".claude", "skills", "db-cleanup-skill")
	skultoDir := filepath.Join(tmpDir, ".skulto", "skills")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	require.NoError(t, os.MkdirAll(skultoDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.md"), []byte("# DB Cleanup Test"), 0644))

	// Scan and store in database
	scanner := NewScannerService()
	discoveries, err := scanner.ScanDirectory(filepath.Join(tmpDir, ".claude", "skills"), "claude", "project")
	require.NoError(t, err)
	require.Len(t, discoveries, 1)

	database := setupTestDB(t)
	for _, d := range discoveries {
		require.NoError(t, database.UpsertDiscoveredSkill(&d))
	}

	// Verify in database
	count, err := database.CountDiscoveredSkills()
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Ingest WITH database connection
	svc := &IngestionService{
		db:              database,
		destDirOverride: skultoDir,
	}
	stored, _ := database.ListDiscoveredSkills()
	_, err = svc.IngestSkill(context.Background(), &stored[0])
	require.NoError(t, err)

	// Verify removed from database
	count, err = database.CountDiscoveredSkills()
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

// TestIntegration_MultipleFilesPreserved tests that ingestion preserves all files
func TestIntegration_MultipleFilesPreserved(t *testing.T) {
	tmpDir := t.TempDir()

	// Create skill with multiple files and subdirectories
	skillDir := filepath.Join(tmpDir, ".claude", "skills", "complex-skill")
	skultoDir := filepath.Join(tmpDir, ".skulto", "skills")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	require.NoError(t, os.MkdirAll(skultoDir, 0755))

	// Create main skill.md
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.md"), []byte("# Complex Skill"), 0644))

	// Create helper files
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "helper.py"), []byte("print('hello')"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "config.json"), []byte("{}"), 0644))

	// Create subdirectory with files
	subDir := filepath.Join(skillDir, "templates")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "template.txt"), []byte("template content"), 0644))

	// Scan and ingest
	scanner := NewScannerService()
	discoveries, err := scanner.ScanDirectory(filepath.Join(tmpDir, ".claude", "skills"), "claude", "project")
	require.NoError(t, err)
	require.Len(t, discoveries, 1)

	svc := &IngestionService{destDirOverride: skultoDir}
	ds := discoveries[0]
	_, err = svc.IngestSkill(context.Background(), &ds)
	require.NoError(t, err)

	// Verify all files were copied
	destDir := filepath.Join(skultoDir, "complex-skill")
	_, err = os.Stat(filepath.Join(destDir, "skill.md"))
	assert.NoError(t, err, "skill.md should exist")
	_, err = os.Stat(filepath.Join(destDir, "helper.py"))
	assert.NoError(t, err, "helper.py should exist")
	_, err = os.Stat(filepath.Join(destDir, "config.json"))
	assert.NoError(t, err, "config.json should exist")
	_, err = os.Stat(filepath.Join(destDir, "templates", "template.txt"))
	assert.NoError(t, err, "templates/template.txt should exist")
}

// TestIntegration_SymlinkCategorization tests that symlinks are categorized correctly
func TestIntegration_SymlinkCategorization(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup: Create skulto-managed skill via ingestion
	skillDir := filepath.Join(tmpDir, ".claude", "skills", "categorize-skill")
	skultoDir := filepath.Join(tmpDir, ".skulto", "skills")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	require.NoError(t, os.MkdirAll(skultoDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.md"), []byte("# Categorize"), 0644))

	// Scan and ingest
	scanner := NewScannerService()
	discoveries, err := scanner.ScanDirectory(filepath.Join(tmpDir, ".claude", "skills"), "claude", "project")
	require.NoError(t, err)
	require.Len(t, discoveries, 1)

	svc := &IngestionService{destDirOverride: skultoDir}
	ds := discoveries[0]
	_, err = svc.IngestSkill(context.Background(), &ds)
	require.NoError(t, err)

	// Now categorize the symlink - should be skulto-managed
	source := scanner.CategorizeSymlink(skillDir)
	assert.Equal(t, models.ManagementSkulto, source)
}

// TestIntegration_ScopeIsolation tests that project and global scopes are isolated
func TestIntegration_ScopeIsolation(t *testing.T) {
	database := setupTestDB(t)

	// Create discoveries with different scopes
	projectSkill := models.DiscoveredSkill{
		Platform: "claude",
		Scope:    "project",
		Path:     "/project/.claude/skills/my-skill",
		Name:     "my-skill",
	}
	projectSkill.ID = projectSkill.GenerateID()
	require.NoError(t, database.UpsertDiscoveredSkill(&projectSkill))

	globalSkill := models.DiscoveredSkill{
		Platform: "claude",
		Scope:    "global",
		Path:     "/home/user/.claude/skills/my-skill",
		Name:     "my-skill", // Same name, different scope
	}
	globalSkill.ID = globalSkill.GenerateID()
	require.NoError(t, database.UpsertDiscoveredSkill(&globalSkill))

	// Should have 2 total discoveries
	count, err := database.CountDiscoveredSkills()
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	// Filter by scope should return only matching
	projectSkills, err := database.ListDiscoveredSkillsByScope("project")
	require.NoError(t, err)
	assert.Len(t, projectSkills, 1)
	assert.Equal(t, "project", projectSkills[0].Scope)

	globalSkills, err := database.ListDiscoveredSkillsByScope("global")
	require.NoError(t, err)
	assert.Len(t, globalSkills, 1)
	assert.Equal(t, "global", globalSkills[0].Scope)
}

// TestIntegration_NotificationTracking tests notification flow
func TestIntegration_NotificationTracking(t *testing.T) {
	database := setupTestDB(t)

	// Create discovery
	ds := models.DiscoveredSkill{
		Platform: "claude",
		Scope:    "project",
		Path:     "/test/.claude/skills/notify-skill",
		Name:     "notify-skill",
	}
	ds.ID = ds.GenerateID()
	require.NoError(t, database.UpsertDiscoveredSkill(&ds))

	// Should be in unnotified list
	unnotified, err := database.ListUnnotifiedDiscoveredSkills()
	require.NoError(t, err)
	assert.Len(t, unnotified, 1)

	// Mark as notified
	err = database.MarkDiscoveredSkillsNotified([]string{ds.ID})
	require.NoError(t, err)

	// Should no longer be in unnotified list
	unnotified, err = database.ListUnnotifiedDiscoveredSkills()
	require.NoError(t, err)
	assert.Len(t, unnotified, 0)

	// But still in total list
	all, err := database.ListDiscoveredSkills()
	require.NoError(t, err)
	assert.Len(t, all, 1)
	assert.NotNil(t, all[0].NotifiedAt)
}

// TestIntegration_EndToEndUserJourney tests a complete user journey
func TestIntegration_EndToEndUserJourney(t *testing.T) {
	tmpDir := t.TempDir()
	database := setupTestDB(t)

	// Step 1: User has manually created a skill
	skillDir := filepath.Join(tmpDir, ".claude", "skills", "user-skill")
	skultoDir := filepath.Join(tmpDir, ".skulto", "skills")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	require.NoError(t, os.MkdirAll(skultoDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.md"), []byte("# User Created Skill\n\nMy awesome skill."), 0644))

	// Step 2: Skulto scans and discovers it
	scanner := NewScannerService()
	discoveries, err := scanner.ScanDirectory(filepath.Join(tmpDir, ".claude", "skills"), "claude", "project")
	require.NoError(t, err)
	require.Len(t, discoveries, 1, "Scanner should find the skill")

	// Step 3: Store discovery in database
	for _, d := range discoveries {
		require.NoError(t, database.UpsertDiscoveredSkill(&d))
	}

	// Step 4: Get unnotified discoveries (for startup notification)
	unnotified, err := database.ListUnnotifiedDiscoveredSkills()
	require.NoError(t, err)
	assert.Len(t, unnotified, 1, "Should have 1 unnotified discovery")

	// Step 5: Mark as notified (user saw notification)
	ids := make([]string, len(unnotified))
	for i, u := range unnotified {
		ids[i] = u.ID
	}
	require.NoError(t, database.MarkDiscoveredSkillsNotified(ids))

	// Step 6: User decides to ingest
	stored, err := database.ListDiscoveredSkills()
	require.NoError(t, err)
	require.Len(t, stored, 1)

	svc := &IngestionService{
		db:              database,
		destDirOverride: skultoDir,
	}
	result, err := svc.IngestSkill(context.Background(), &stored[0])
	require.NoError(t, err)
	assert.Equal(t, "user-skill", result.Name)

	// Step 7: Verify final state
	// - Original is now a symlink
	lstat, err := os.Lstat(skillDir)
	require.NoError(t, err)
	assert.True(t, lstat.Mode()&os.ModeSymlink != 0, "Original should be symlink")

	// - Skill is in skulto directory
	_, err = os.Stat(filepath.Join(skultoDir, "user-skill", "skill.md"))
	assert.NoError(t, err, "Skill should exist in skulto dir")

	// - Discovery removed from database
	count, err := database.CountDiscoveredSkills()
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "Discovery should be removed from DB")

	// - Re-scan should not find it (it's now a symlink)
	rediscoveries, err := scanner.ScanDirectory(filepath.Join(tmpDir, ".claude", "skills"), "claude", "project")
	require.NoError(t, err)
	assert.Len(t, rediscoveries, 0, "Re-scan should not find ingested skill")

	// - Symlink is categorized as skulto-managed
	source := scanner.CategorizeSymlink(skillDir)
	assert.Equal(t, models.ManagementSkulto, source, "Symlink should be skulto-managed")
}
