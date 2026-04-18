package installer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateDeprecatedPathsForLocation_SuccessAndDBSync(t *testing.T) {
	database := setupTestDB(t)
	cfg := setupTestConfig(t)
	inst := New(database, cfg)

	basePath := t.TempDir()
	deprecatedBase := filepath.Join(basePath, ".opencode", "skills")
	canonicalBase := filepath.Join(basePath, ".config", "opencode", "skills")
	require.NoError(t, os.MkdirAll(deprecatedBase, 0o755))

	sourceTarget := t.TempDir()
	sourceLink := filepath.Join(deprecatedBase, "test-skill")
	require.NoError(t, os.Symlink(sourceTarget, sourceLink))

	skill := &models.Skill{
		ID:      "migration-skill-1",
		Slug:    "test-skill",
		Title:   "Migration Skill",
		Content: "# Skill",
	}
	require.NoError(t, database.CreateSkill(skill))
	require.NoError(t, database.AddInstallation(&models.SkillInstallation{
		SkillID:     skill.ID,
		Platform:    string(PlatformOpenCode),
		Scope:       string(ScopeGlobal),
		BasePath:    basePath,
		SymlinkPath: sourceLink,
	}))

	result := inst.migrateDeprecatedPathsForLocation(PlatformOpenCode, ScopeGlobal, basePath)
	assert.Equal(t, 1, result.Migrated)
	assert.Equal(t, 0, result.SkippedConflicts)
	assert.Equal(t, 0, result.SkippedBrokenSymlinks)
	assert.Equal(t, 0, result.SkippedPlainDirectories)

	destLink := filepath.Join(canonicalBase, "test-skill")
	assert.NoFileExists(t, sourceLink)
	assert.FileExists(t, destLink)

	resolvedTarget, err := filepath.EvalSymlinks(destLink)
	require.NoError(t, err)
	expectedTarget, err := filepath.EvalSymlinks(sourceTarget)
	require.NoError(t, err)
	assert.Equal(t, expectedTarget, resolvedTarget)

	installations, err := database.GetInstallations(skill.ID)
	require.NoError(t, err)
	require.Len(t, installations, 1)
	assert.Equal(t, destLink, installations[0].SymlinkPath)
}

func TestMigrateDeprecatedPathsForLocation_Idempotent(t *testing.T) {
	database := setupTestDB(t)
	cfg := setupTestConfig(t)
	inst := New(database, cfg)

	basePath := t.TempDir()
	deprecatedBase := filepath.Join(basePath, ".opencode", "skills")
	require.NoError(t, os.MkdirAll(deprecatedBase, 0o755))
	sourceTarget := t.TempDir()
	require.NoError(t, os.Symlink(sourceTarget, filepath.Join(deprecatedBase, "test-skill")))

	first := inst.migrateDeprecatedPathsForLocation(PlatformOpenCode, ScopeGlobal, basePath)
	second := inst.migrateDeprecatedPathsForLocation(PlatformOpenCode, ScopeGlobal, basePath)

	assert.Equal(t, 1, first.Migrated)
	assert.Equal(t, 0, second.Migrated)
	assert.Equal(t, 0, second.SkippedConflicts)
	assert.Equal(t, 0, second.SkippedBrokenSymlinks)
	assert.Equal(t, 0, second.SkippedPlainDirectories)
}

func TestMigrateDeprecatedPathsForLocation_ConflictSkipped(t *testing.T) {
	database := setupTestDB(t)
	cfg := setupTestConfig(t)
	inst := New(database, cfg)

	basePath := t.TempDir()
	deprecatedBase := filepath.Join(basePath, ".opencode", "skills")
	canonicalBase := filepath.Join(basePath, ".config", "opencode", "skills")
	require.NoError(t, os.MkdirAll(deprecatedBase, 0o755))
	require.NoError(t, os.MkdirAll(canonicalBase, 0o755))

	legacyTarget := t.TempDir()
	newTarget := t.TempDir()
	legacyLink := filepath.Join(deprecatedBase, "test-skill")
	destLink := filepath.Join(canonicalBase, "test-skill")
	require.NoError(t, os.Symlink(legacyTarget, legacyLink))
	require.NoError(t, os.Symlink(newTarget, destLink))

	result := inst.migrateDeprecatedPathsForLocation(PlatformOpenCode, ScopeGlobal, basePath)
	assert.Equal(t, 0, result.Migrated)
	assert.Equal(t, 1, result.SkippedConflicts)

	assert.FileExists(t, legacyLink)
	assert.FileExists(t, destLink)
}

func TestMigrateDeprecatedPathsForLocation_BrokenSymlinkSkipped(t *testing.T) {
	database := setupTestDB(t)
	cfg := setupTestConfig(t)
	inst := New(database, cfg)

	basePath := t.TempDir()
	deprecatedBase := filepath.Join(basePath, ".opencode", "skills")
	require.NoError(t, os.MkdirAll(deprecatedBase, 0o755))
	require.NoError(t, os.Symlink(filepath.Join(basePath, "does-not-exist"), filepath.Join(deprecatedBase, "broken-skill")))

	result := inst.migrateDeprecatedPathsForLocation(PlatformOpenCode, ScopeGlobal, basePath)
	assert.Equal(t, 0, result.Migrated)
	assert.Equal(t, 1, result.SkippedBrokenSymlinks)
}

func TestMigrateDeprecatedPathsForLocation_PlainDirectorySkipped(t *testing.T) {
	database := setupTestDB(t)
	cfg := setupTestConfig(t)
	inst := New(database, cfg)

	basePath := t.TempDir()
	deprecatedBase := filepath.Join(basePath, ".opencode", "skills")
	plainDir := filepath.Join(deprecatedBase, "plain-skill")
	require.NoError(t, os.MkdirAll(plainDir, 0o755))

	result := inst.migrateDeprecatedPathsForLocation(PlatformOpenCode, ScopeGlobal, basePath)
	assert.Equal(t, 0, result.Migrated)
	assert.Equal(t, 1, result.SkippedPlainDirectories)
	assert.DirExists(t, plainDir)
}

func TestEnsurePathPolicyForBasePaths_MigratesOpenCodeGlobal(t *testing.T) {
	database := setupTestDB(t)
	cfg := setupTestConfig(t)
	inst := New(database, cfg)

	globalBase := t.TempDir()
	projectBase := t.TempDir()
	deprecatedBase := filepath.Join(globalBase, ".opencode", "skills")
	require.NoError(t, os.MkdirAll(deprecatedBase, 0o755))
	target := t.TempDir()
	require.NoError(t, os.Symlink(target, filepath.Join(deprecatedBase, "policy-skill")))

	summary := inst.ensurePathPolicyForBasePaths(map[InstallScope]string{
		ScopeGlobal:  globalBase,
		ScopeProject: projectBase,
	})
	assert.Equal(t, 1, summary.Migrated)
	assert.Equal(t, 0, summary.SkippedConflicts)

	assert.FileExists(t, filepath.Join(globalBase, ".config", "opencode", "skills", "policy-skill"))
	assert.NoFileExists(t, filepath.Join(globalBase, ".opencode", "skills", "policy-skill"))
}
