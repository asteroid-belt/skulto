package installer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// CHARACTERIZATION TESTS
// =============================================================================
// These tests capture CURRENT behavior, not desired behavior.
// If these tests fail after refactoring, behavior changed (possibly incorrectly).
// DO NOT MODIFY these tests without understanding why existing behavior changed.
// =============================================================================

// setupCharTestDB creates a test database instance for characterization tests.
func setupCharTestDB(t *testing.T, tmpDir string) *db.DB {
	t.Helper()
	dbPath := filepath.Join(tmpDir, "test.db")
	database, err := db.New(db.DefaultConfig(dbPath))
	require.NoError(t, err)
	return database
}

// TestCharacterization_Install_NilSkill captures behavior when skill is nil.
func TestCharacterization_Install_NilSkill(t *testing.T) {
	tmpDir := t.TempDir()
	database := setupCharTestDB(t, tmpDir)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{BaseDir: tmpDir}
	inst := New(database, cfg)

	err := inst.Install(context.Background(), nil, &models.Source{})

	// Current behavior: returns error with message "skill cannot be nil"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "skill cannot be nil")
}

// TestCharacterization_Install_EmptySlug captures behavior when skill slug is empty.
func TestCharacterization_Install_EmptySlug(t *testing.T) {
	tmpDir := t.TempDir()
	database := setupCharTestDB(t, tmpDir)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{BaseDir: tmpDir}
	inst := New(database, cfg)

	skill := &models.Skill{ID: "test-id", Slug: ""}

	err := inst.Install(context.Background(), skill, &models.Source{})

	// Current behavior: returns ErrInvalidSkill
	assert.ErrorIs(t, err, ErrInvalidSkill)
}

// TestCharacterization_Install_NilSource captures behavior when source is nil.
func TestCharacterization_Install_NilSource(t *testing.T) {
	tmpDir := t.TempDir()
	database := setupCharTestDB(t, tmpDir)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{BaseDir: tmpDir}
	inst := New(database, cfg)

	skill := &models.Skill{ID: "test-id", Slug: "test-skill"}

	err := inst.Install(context.Background(), skill, nil)

	// Current behavior: returns error with message about nil source
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source cannot be nil")
}

// TestCharacterization_Install_NoToolsSelected captures behavior when no AI tools are selected.
func TestCharacterization_Install_NoToolsSelected(t *testing.T) {
	tmpDir := t.TempDir()
	database := setupCharTestDB(t, tmpDir)
	defer func() { _ = database.Close() }()

	// Create skill source directory (uses BaseDir/repositories)
	sourceDir := filepath.Join(tmpDir, "repositories", "test-owner", "test-repo", "skills", "test-skill")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))

	cfg := &config.Config{
		BaseDir: tmpDir,
	}
	inst := New(database, cfg)

	skill := &models.Skill{ID: "test-id", Slug: "test-skill", FilePath: "skills/test-skill"}
	source := &models.Source{Owner: "test-owner", Repo: "test-repo"}

	// User state has no tools selected (default state)
	err := inst.Install(context.Background(), skill, source)

	// Current behavior: returns ErrNoToolsSelected
	assert.ErrorIs(t, err, ErrNoToolsSelected)
}

// TestCharacterization_Install_SourcePathNotFound captures behavior when skill directory doesn't exist.
func TestCharacterization_Install_SourcePathNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	database := setupCharTestDB(t, tmpDir)
	defer func() { _ = database.Close() }()

	// Set up user state with AI tool selected
	err := database.UpdateAITools("claude")
	require.NoError(t, err)

	cfg := &config.Config{
		BaseDir: tmpDir,
	}
	inst := New(database, cfg)

	skill := &models.Skill{ID: "test-id", Slug: "test-skill", FilePath: "skills/nonexistent"}
	source := &models.Source{Owner: "test-owner", Repo: "test-repo"}

	err = inst.Install(context.Background(), skill, source)

	// Current behavior: returns error with "skill directory not found"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "skill directory not found")
}

// TestCharacterization_Uninstall_NilSkill captures behavior when skill is nil.
func TestCharacterization_Uninstall_NilSkill(t *testing.T) {
	tmpDir := t.TempDir()
	database := setupCharTestDB(t, tmpDir)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{BaseDir: tmpDir}
	inst := New(database, cfg)

	err := inst.Uninstall(context.Background(), nil)

	// Current behavior: returns error with message "skill cannot be nil"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "skill cannot be nil")
}

// TestCharacterization_Uninstall_EmptySlug captures behavior when skill slug is empty.
func TestCharacterization_Uninstall_EmptySlug(t *testing.T) {
	tmpDir := t.TempDir()
	database := setupCharTestDB(t, tmpDir)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{BaseDir: tmpDir}
	inst := New(database, cfg)

	skill := &models.Skill{ID: "test-id", Slug: ""}

	err := inst.Uninstall(context.Background(), skill)

	// Current behavior: returns ErrInvalidSkill
	assert.ErrorIs(t, err, ErrInvalidSkill)
}

// TestCharacterization_Uninstall_NoSymlinksExist captures behavior when no symlinks exist.
func TestCharacterization_Uninstall_NoSymlinksExist(t *testing.T) {
	tmpDir := t.TempDir()
	database := setupCharTestDB(t, tmpDir)
	defer func() { _ = database.Close() }()

	// Add a source first (to satisfy foreign key constraint)
	source := &models.Source{ID: "test-source", Owner: "test-owner", Repo: "test-repo"}
	err := database.CreateSource(source)
	require.NoError(t, err)

	// Add a skill to the database
	skill := &models.Skill{ID: "test-id", Slug: "test-skill", SourceID: &source.ID}
	err = database.UpsertSkill(skill)
	require.NoError(t, err)

	cfg := &config.Config{BaseDir: tmpDir}
	inst := New(database, cfg)

	err = inst.Uninstall(context.Background(), skill)

	// Current behavior: succeeds even if no symlinks exist (idempotent)
	assert.NoError(t, err)
}

// TestCharacterization_InstallTo_EmptyLocations captures behavior when no locations specified.
func TestCharacterization_InstallTo_EmptyLocations(t *testing.T) {
	tmpDir := t.TempDir()
	database := setupCharTestDB(t, tmpDir)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{BaseDir: tmpDir}
	inst := New(database, cfg)

	skill := &models.Skill{ID: "test-id", Slug: "test-skill"}
	source := &models.Source{Owner: "test-owner", Repo: "test-repo"}

	err := inst.InstallTo(context.Background(), skill, source, []InstallLocation{})

	// Current behavior: returns error with "no installation locations specified"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no installation locations specified")
}

// TestCharacterization_InstallLocalSkillTo_EmptySourcePath captures behavior when source path is empty.
func TestCharacterization_InstallLocalSkillTo_EmptySourcePath(t *testing.T) {
	tmpDir := t.TempDir()
	database := setupCharTestDB(t, tmpDir)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{BaseDir: tmpDir}
	inst := New(database, cfg)

	skill := &models.Skill{ID: "test-id", Slug: "test-skill"}
	locations := []InstallLocation{{Platform: PlatformClaude, Scope: ScopeGlobal}}

	err := inst.InstallLocalSkillTo(context.Background(), skill, "", locations)

	// Current behavior: returns error with "source path cannot be empty"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source path cannot be empty")
}

// TestCharacterization_UninstallAll_NilSkill captures behavior when skill is nil.
func TestCharacterization_UninstallAll_NilSkill(t *testing.T) {
	tmpDir := t.TempDir()
	database := setupCharTestDB(t, tmpDir)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{BaseDir: tmpDir}
	inst := New(database, cfg)

	err := inst.UninstallAll(context.Background(), nil)

	// Current behavior: returns error with message "skill cannot be nil"
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "skill cannot be nil")
}

// TestCharacterization_UninstallAll_EmptySlug captures behavior when skill slug is empty.
func TestCharacterization_UninstallAll_EmptySlug(t *testing.T) {
	tmpDir := t.TempDir()
	database := setupCharTestDB(t, tmpDir)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{BaseDir: tmpDir}
	inst := New(database, cfg)

	skill := &models.Skill{ID: "test-id", Slug: ""}

	err := inst.UninstallAll(context.Background(), skill)

	// Current behavior: returns ErrInvalidSkill
	assert.ErrorIs(t, err, ErrInvalidSkill)
}

// TestCharacterization_IsInstalled_SkillNotFound captures behavior when skill doesn't exist.
func TestCharacterization_IsInstalled_SkillNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	database := setupCharTestDB(t, tmpDir)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{BaseDir: tmpDir}
	inst := New(database, cfg)

	installed, err := inst.IsInstalled("nonexistent-id")

	// Current behavior: returns ErrSkillNotFound
	assert.ErrorIs(t, err, ErrSkillNotFound)
	assert.False(t, installed)
}

// TestCharacterization_GetInstallLocations_SkillNotInstalled captures behavior for uninstalled skill.
func TestCharacterization_GetInstallLocations_SkillNotInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	database := setupCharTestDB(t, tmpDir)
	defer func() { _ = database.Close() }()

	cfg := &config.Config{BaseDir: tmpDir}
	inst := New(database, cfg)

	locations, err := inst.GetInstallLocations("nonexistent-id")

	// Current behavior: returns empty slice without error
	assert.NoError(t, err)
	assert.Empty(t, locations)
}

// TestCharacterization_parsePlatforms_NilState captures behavior when user state is nil.
func TestCharacterization_parsePlatforms_NilState(t *testing.T) {
	platforms := parsePlatforms(nil)

	// Current behavior: returns nil slice
	assert.Nil(t, platforms)
}

// TestCharacterization_parsePlatforms_EmptyTools captures behavior when no tools are set.
func TestCharacterization_parsePlatforms_EmptyTools(t *testing.T) {
	state := &models.UserState{AITools: ""}
	platforms := parsePlatforms(state)

	// Current behavior: returns nil slice
	assert.Nil(t, platforms)
}

// TestCharacterization_parsePlatforms_ValidTools captures behavior with valid tool names.
func TestCharacterization_parsePlatforms_ValidTools(t *testing.T) {
	state := &models.UserState{AITools: "claude,cursor,copilot"}
	platforms := parsePlatforms(state)

	// Current behavior: returns slice of matching platforms
	assert.Len(t, platforms, 3)
	assert.Contains(t, platforms, PlatformClaude)
	assert.Contains(t, platforms, PlatformCursor)
	assert.Contains(t, platforms, PlatformCopilot)
}

// TestCharacterization_parsePlatforms_WithWhitespace captures behavior with whitespace in tool names.
func TestCharacterization_parsePlatforms_WithWhitespace(t *testing.T) {
	state := &models.UserState{AITools: "claude , cursor , copilot"}
	platforms := parsePlatforms(state)

	// Current behavior: trims whitespace and returns matching platforms
	assert.Len(t, platforms, 3)
}

// TestCharacterization_parsePlatforms_UnknownTool captures behavior with unknown tool names.
func TestCharacterization_parsePlatforms_UnknownTool(t *testing.T) {
	state := &models.UserState{AITools: "claude,unknown-tool,cursor"}
	platforms := parsePlatforms(state)

	// Current behavior: skips unknown tools, returns only valid ones
	assert.Len(t, platforms, 2)
	assert.Contains(t, platforms, PlatformClaude)
	assert.Contains(t, platforms, PlatformCursor)
}

// TestCharacterization_exists_NonExistentPath captures behavior for non-existent path.
func TestCharacterization_exists_NonExistentPath(t *testing.T) {
	result := exists("/nonexistent/path/12345")

	// Current behavior: returns false
	assert.False(t, result)
}

// TestCharacterization_exists_ExistingPath captures behavior for existing path.
func TestCharacterization_exists_ExistingPath(t *testing.T) {
	tmpDir := t.TempDir()
	result := exists(tmpDir)

	// Current behavior: returns true
	assert.True(t, result)
}

// TestCharacterization_isSymlink_RegularFile captures behavior for regular file.
func TestCharacterization_isSymlink_RegularFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "regular-file")
	require.NoError(t, os.WriteFile(filePath, []byte("content"), 0644))

	result := isSymlink(filePath)

	// Current behavior: returns false for regular file
	assert.False(t, result)
}

// TestCharacterization_isSymlink_Directory captures behavior for directory.
func TestCharacterization_isSymlink_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	result := isSymlink(tmpDir)

	// Current behavior: returns false for directory
	assert.False(t, result)
}

// TestCharacterization_isSymlink_ActualSymlink captures behavior for actual symlink.
func TestCharacterization_isSymlink_ActualSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target")
	require.NoError(t, os.WriteFile(target, []byte("content"), 0644))

	link := filepath.Join(tmpDir, "link")
	require.NoError(t, os.Symlink(target, link))

	result := isSymlink(link)

	// Current behavior: returns true for symlink
	assert.True(t, result)
}

// TestCharacterization_isSymlink_NonExistentPath captures behavior for non-existent path.
func TestCharacterization_isSymlink_NonExistentPath(t *testing.T) {
	result := isSymlink("/nonexistent/path/12345")

	// Current behavior: returns false (Lstat fails)
	assert.False(t, result)
}
