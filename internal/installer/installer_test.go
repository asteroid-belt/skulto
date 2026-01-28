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

// setupTestDB creates a test database instance.
func setupTestDB(t *testing.T) *db.DB {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	dbConfig := db.Config{Path: dbPath}
	database, err := db.New(dbConfig)
	require.NoError(t, err)
	return database
}

// setupTestConfig creates a test config with temporary directories.
func setupTestConfig(t *testing.T) *config.Config {
	tmpDir := t.TempDir()
	baseDir := filepath.Join(tmpDir, ".skulto")
	return &config.Config{
		BaseDir: baseDir,
	}
}

// setupTestSkillDir creates a mock skill directory in the test repository.
func setupTestSkillDir(t *testing.T, cfg *config.Config, owner, repo, skillSlug string) string {
	skillDir := filepath.Join(cfg.BaseDir, "repositories", owner, repo, "skills", skillSlug)
	err := os.MkdirAll(skillDir, 0755)
	require.NoError(t, err)

	// Create a SKILL.md file
	skillFile := filepath.Join(skillDir, "SKILL.md")
	err = os.WriteFile(skillFile, []byte("# Test Skill\n\nThis is a test skill."), 0644)
	require.NoError(t, err)

	return skillDir
}

// TestPlatformRegistry tests the platform registry.
func TestPlatformRegistry(t *testing.T) {
	platforms := AllPlatforms()
	assert.Len(t, platforms, 33)
	assert.Contains(t, platforms, PlatformClaude)
	assert.Contains(t, platforms, PlatformCursor)
	assert.Contains(t, platforms, PlatformCopilot)
	assert.Contains(t, platforms, PlatformCodex)
	assert.Contains(t, platforms, PlatformOpenCode)
	assert.Contains(t, platforms, PlatformWindsurf)
}

// TestPlatformIsValid tests platform validation.
func TestPlatformIsValid(t *testing.T) {
	assert.True(t, PlatformClaude.IsValid())
	assert.True(t, PlatformCursor.IsValid())
	assert.False(t, Platform("invalid").IsValid())
	assert.False(t, Platform("").IsValid())
}

// TestPlatformGetSkillPath tests skill path generation.
func TestPlatformGetSkillPath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name     string
		platform Platform
		slug     string
		expected string
	}{
		{
			name:     "Claude",
			platform: PlatformClaude,
			slug:     "test-skill",
			expected: filepath.Join(home, ".claude", "skills", "test-skill"),
		},
		{
			name:     "Cursor",
			platform: PlatformCursor,
			slug:     "test-skill",
			expected: filepath.Join(home, ".cursor", "skills", "test-skill"),
		},
		{
			name:     "OpenCode",
			platform: PlatformOpenCode,
			slug:     "test-skill",
			expected: filepath.Join(home, ".opencode", "skills", "test-skill"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := tt.platform.GetSkillPath(tt.slug)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, path)
		})
	}
}

// TestPlatformFromString tests platform string conversion.
func TestPlatformFromString(t *testing.T) {
	assert.Equal(t, PlatformClaude, PlatformFromString("claude"))
	assert.Equal(t, PlatformCursor, PlatformFromString("cursor"))
	assert.Equal(t, Platform(""), PlatformFromString("invalid"))
	assert.Equal(t, Platform(""), PlatformFromString(""))
}

// TestPathResolver tests path resolution.
func TestPathResolver(t *testing.T) {
	cfg := setupTestConfig(t)
	resolver := NewPathResolver(cfg)

	// Test GetSourcePath - now takes the skill's FilePath and derives the directory
	sourcePath := resolver.GetSourcePath("owner", "repo", "skills/skill-slug/SKILL.md")
	expected := filepath.Join(cfg.BaseDir, "repositories", "owner", "repo", "skills", "skill-slug")
	assert.Equal(t, expected, sourcePath)

	// Test GetSourcePath with different directory structure
	sourcePath2 := resolver.GetSourcePath("owner", "repo", "prompts/my-skill/SKILL.md")
	expected2 := filepath.Join(cfg.BaseDir, "repositories", "owner", "repo", "prompts", "my-skill")
	assert.Equal(t, expected2, sourcePath2)

	// Test GetRepositoriesDir
	repoDir := resolver.GetRepositoriesDir()
	assert.Equal(t, filepath.Join(cfg.BaseDir, "repositories"), repoDir)

	// Test GetGlobalPath
	home, _ := os.UserHomeDir()
	globalPath, err := resolver.GetGlobalPath(PlatformClaude, "test-skill")
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".claude", "skills", "test-skill"), globalPath)
}

// TestInstallerWithSinglePlatform tests install/uninstall with a single platform.
func TestInstallerWithSinglePlatform(t *testing.T) {
	database := setupTestDB(t)
	cfg := setupTestConfig(t)

	// Set up user state with single platform
	err := database.UpdateAITools(string(PlatformClaude))
	require.NoError(t, err)

	// Create installer
	inst := New(database, cfg)

	// Create test skill and source
	skill := &models.Skill{
		ID:      "test-skill-1",
		Slug:    "test-skill",
		Title:   "Test Skill",
		Content: "# Test Skill\n\nThis is a test.",
		Version: "1.0.0",
		Author:  "Test Author",
	}
	source := &models.Source{
		Owner: "test-owner",
		Repo:  "test-repo",
	}

	// Create skill directory in mock repository
	setupTestSkillDir(t, cfg, source.Owner, source.Repo, skill.Slug)

	// Create skill in database first
	err = database.CreateSkill(skill)
	require.NoError(t, err)

	// Cleanup after test
	t.Cleanup(func() {
		targetPath, _ := PlatformClaude.GetSkillPath(skill.Slug)
		_ = os.RemoveAll(targetPath)
	})

	// Install skill
	err = inst.Install(context.Background(), skill, source)
	require.NoError(t, err)

	// Verify symlink exists
	targetPath, err := PlatformClaude.GetSkillPath(skill.Slug)
	require.NoError(t, err)

	// Verify it's a symlink
	info, err := os.Lstat(targetPath)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "Expected symlink")

	// Verify database is updated
	installed, err := inst.IsInstalled(skill.ID)
	require.NoError(t, err)
	assert.True(t, installed)

	// Uninstall skill
	err = inst.Uninstall(context.Background(), skill)
	require.NoError(t, err)

	// Verify symlink is removed
	_, err = os.Lstat(targetPath)
	assert.True(t, os.IsNotExist(err))

	// Verify database is updated
	installed, err = inst.IsInstalled(skill.ID)
	require.NoError(t, err)
	assert.False(t, installed)
}

// TestInstallerWithMultiplePlatforms tests install with multiple platforms.
func TestInstallerWithMultiplePlatforms(t *testing.T) {
	database := setupTestDB(t)
	cfg := setupTestConfig(t)

	// Set up user state with multiple platforms
	aiTools := string(PlatformClaude) + "," + string(PlatformCursor)
	err := database.UpdateAITools(aiTools)
	require.NoError(t, err)

	// Create installer
	inst := New(database, cfg)

	// Create test skill and source
	skill := &models.Skill{
		ID:      "test-skill-2",
		Slug:    "multi-platform-skill",
		Title:   "Multi Platform Skill",
		Content: "# Multi Platform\n\nTest on multiple platforms.",
		Version: "1.0.0",
		Author:  "Test Author",
	}
	source := &models.Source{
		Owner: "test-owner",
		Repo:  "test-repo",
	}

	// Create skill directory in mock repository
	setupTestSkillDir(t, cfg, source.Owner, source.Repo, skill.Slug)

	// Create skill in database first
	err = database.CreateSkill(skill)
	require.NoError(t, err)

	// Cleanup after test
	t.Cleanup(func() {
		claudePath, _ := PlatformClaude.GetSkillPath(skill.Slug)
		cursorPath, _ := PlatformCursor.GetSkillPath(skill.Slug)
		_ = os.RemoveAll(claudePath)
		_ = os.RemoveAll(cursorPath)
	})

	// Install skill
	err = inst.Install(context.Background(), skill, source)
	require.NoError(t, err)

	// Verify symlinks exist for both platforms
	claudePath, _ := PlatformClaude.GetSkillPath(skill.Slug)
	cursorPath, _ := PlatformCursor.GetSkillPath(skill.Slug)

	info1, err := os.Lstat(claudePath)
	require.NoError(t, err)
	assert.True(t, info1.Mode()&os.ModeSymlink != 0)

	info2, err := os.Lstat(cursorPath)
	require.NoError(t, err)
	assert.True(t, info2.Mode()&os.ModeSymlink != 0)

	// Uninstall
	err = inst.Uninstall(context.Background(), skill)
	require.NoError(t, err)

	// Verify symlinks are removed
	_, err = os.Lstat(claudePath)
	assert.True(t, os.IsNotExist(err))
	_, err = os.Lstat(cursorPath)
	assert.True(t, os.IsNotExist(err))
}

// TestInstallerNoToolsSelected tests install with no tools selected.
func TestInstallerNoToolsSelected(t *testing.T) {
	database := setupTestDB(t)
	cfg := setupTestConfig(t)

	// Don't set user state - no tools selected
	inst := New(database, cfg)

	skill := &models.Skill{
		ID:      "test-skill-3",
		Slug:    "test-skill",
		Title:   "Test Skill",
		Content: "# Test",
		Version: "1.0.0",
	}
	source := &models.Source{
		Owner: "test-owner",
		Repo:  "test-repo",
	}

	err := inst.Install(context.Background(), skill, source)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoToolsSelected)
}

// TestInstallerInvalidSkill tests install with invalid skill.
func TestInstallerInvalidSkill(t *testing.T) {
	database := setupTestDB(t)
	cfg := setupTestConfig(t)

	// Set up user state
	_ = database.UpdateAITools(string(PlatformClaude))

	inst := New(database, cfg)

	// Skill with empty slug
	skill := &models.Skill{
		ID:      "test-skill-4",
		Slug:    "", // Empty slug - invalid
		Title:   "Test",
		Content: "# Test",
	}
	source := &models.Source{
		Owner: "test-owner",
		Repo:  "test-repo",
	}

	err := inst.Install(context.Background(), skill, source)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidSkill)
}

// TestInstallerNilSource tests install with nil source.
func TestInstallerNilSource(t *testing.T) {
	database := setupTestDB(t)
	cfg := setupTestConfig(t)

	_ = database.UpdateAITools(string(PlatformClaude))

	inst := New(database, cfg)

	skill := &models.Skill{
		ID:      "test-skill-5",
		Slug:    "test-skill",
		Title:   "Test",
		Content: "# Test",
	}

	err := inst.Install(context.Background(), skill, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source cannot be nil")
}

// TestInstallerSourceNotFound tests install with non-existent source directory.
func TestInstallerSourceNotFound(t *testing.T) {
	database := setupTestDB(t)
	cfg := setupTestConfig(t)

	_ = database.UpdateAITools(string(PlatformClaude))

	inst := New(database, cfg)

	skill := &models.Skill{
		ID:      "test-skill-6",
		Slug:    "nonexistent-skill",
		Title:   "Test",
		Content: "# Test",
	}
	source := &models.Source{
		Owner: "nonexistent-owner",
		Repo:  "nonexistent-repo",
	}

	err := inst.Install(context.Background(), skill, source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "skill directory not found")
}

// TestReInstall tests reinstalling a skill.
func TestReInstall(t *testing.T) {
	database := setupTestDB(t)
	cfg := setupTestConfig(t)

	_ = database.UpdateAITools(string(PlatformClaude))

	inst := New(database, cfg)

	skill := &models.Skill{
		ID:      "test-skill-7",
		Slug:    "reinstall-skill",
		Title:   "Reinstall Test",
		Content: "# Original Content",
		Version: "1.0.0",
	}
	source := &models.Source{
		Owner: "test-owner",
		Repo:  "test-repo",
	}

	// Create skill directory in mock repository
	setupTestSkillDir(t, cfg, source.Owner, source.Repo, skill.Slug)

	// Create skill in database first
	err := database.CreateSkill(skill)
	require.NoError(t, err)

	// Cleanup after test
	t.Cleanup(func() {
		targetPath, _ := PlatformClaude.GetSkillPath(skill.Slug)
		_ = os.RemoveAll(targetPath)
	})

	// Initial install
	err = inst.Install(context.Background(), skill, source)
	require.NoError(t, err)

	// Update content and reinstall
	skill.Content = "# Updated Content"
	skill.Version = "2.0.0"

	err = inst.ReInstall(context.Background(), skill, source)
	require.NoError(t, err)

	// Verify symlink still exists
	targetPath, _ := PlatformClaude.GetSkillPath(skill.Slug)
	info, err := os.Lstat(targetPath)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0)

	// Verify database is still marked as installed
	installed, err := inst.IsInstalled(skill.ID)
	require.NoError(t, err)
	assert.True(t, installed)
}

// TestInstallLocalSkillTo tests installation of local skills to specific locations.
func TestInstallLocalSkillTo(t *testing.T) {
	// Setup temp directories
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source", "my-skill")
	destDir := filepath.Join(tempDir, "dest")

	// Create source skill
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "skill.md"), []byte("# Test Skill\n\nTest content."), 0644))

	// Create mock DB and config
	cfg := setupTestConfig(t)
	database := setupTestDB(t)

	// Add skill to database
	skill := &models.Skill{
		ID:      "local-my-skill",
		Slug:    "my-skill",
		Title:   "My Skill",
		Content: "# Test Skill",
		IsLocal: true,
	}
	require.NoError(t, database.CreateSkill(skill))

	// Create installer
	inst := New(database, cfg)

	// Create test location pointing to temp dest
	// InstallLocation.GetSkillPath expects the basePath to be the home-like root
	// For Claude, it would be ~/.claude/skills/<slug>
	// We'll test with a custom basePath
	loc := InstallLocation{
		Platform: PlatformClaude,
		Scope:    ScopeGlobal,
		BasePath: destDir,
	}

	// Install
	err := inst.InstallLocalSkillTo(context.Background(), skill, sourceDir, []InstallLocation{loc})
	require.NoError(t, err)

	// Verify symlink created
	// The actual path depends on how GetSkillPath works with BasePath
	expectedPath := loc.GetSkillPath(skill.Slug)
	info, err := os.Lstat(expectedPath)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0, "should be symlink")

	// Verify symlink target
	target, err := os.Readlink(expectedPath)
	require.NoError(t, err)
	assert.Equal(t, sourceDir, target)

	// Verify installation recorded
	installations, err := database.GetInstallations(skill.ID)
	require.NoError(t, err)
	assert.Len(t, installations, 1)

	// Verify legacy flag updated
	installed, err := inst.IsInstalled(skill.ID)
	require.NoError(t, err)
	assert.True(t, installed)
}

// TestInstallLocalSkillToEmptySource tests that empty source path returns error.
func TestInstallLocalSkillToEmptySource(t *testing.T) {
	cfg := setupTestConfig(t)
	database := setupTestDB(t)

	skill := &models.Skill{
		ID:      "local-test",
		Slug:    "test",
		IsLocal: true,
	}
	require.NoError(t, database.CreateSkill(skill))

	inst := New(database, cfg)

	err := inst.InstallLocalSkillTo(context.Background(), skill, "", []InstallLocation{{Platform: PlatformClaude}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source path cannot be empty")
}

// TestInstallLocalSkillToNoLocations tests that empty locations returns error.
func TestInstallLocalSkillToNoLocations(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))

	cfg := setupTestConfig(t)
	database := setupTestDB(t)

	skill := &models.Skill{
		ID:      "local-test",
		Slug:    "test",
		IsLocal: true,
	}
	require.NoError(t, database.CreateSkill(skill))

	inst := New(database, cfg)

	err := inst.InstallLocalSkillTo(context.Background(), skill, sourceDir, []InstallLocation{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no installation locations specified")
}

// TestSyncInstallState tests that sync correctly reconciles database with symlinks.
func TestSyncInstallState(t *testing.T) {
	tempDir := t.TempDir()

	// Create a mock home directory structure
	homeDir := filepath.Join(tempDir, "home")
	skillsDir := filepath.Join(homeDir, ".claude", "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0755))

	// Create source skill directory
	sourceDir := filepath.Join(tempDir, "source", "my-skill")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "skill.md"), []byte("# Test"), 0644))

	// Create symlink
	symlinkPath := filepath.Join(skillsDir, "my-skill")
	require.NoError(t, os.Symlink(sourceDir, symlinkPath))

	cfg := setupTestConfig(t)
	database := setupTestDB(t)

	// Create skill in database with is_installed = false
	skill := &models.Skill{
		ID:          "local-my-skill",
		Slug:        "my-skill",
		IsLocal:     true,
		IsInstalled: false,
	}
	require.NoError(t, database.CreateSkill(skill))

	// Verify initial state
	assert.False(t, skill.IsInstalled)
	installs, _ := database.GetInstallations(skill.ID)
	assert.Empty(t, installs)

	// Override home directory for test
	t.Setenv("HOME", homeDir)

	inst := New(database, cfg)

	// Run sync
	err := inst.SyncInstallState(context.Background())
	require.NoError(t, err)

	// Verify skill is now marked as installed
	updatedSkill, err := database.GetSkill(skill.ID)
	require.NoError(t, err)
	assert.True(t, updatedSkill.IsInstalled)

	// Verify skill_installations record was created
	installs, err = database.GetInstallations(skill.ID)
	require.NoError(t, err)
	assert.Len(t, installs, 1)
	assert.Equal(t, "claude", installs[0].Platform)
	assert.Equal(t, "global", installs[0].Scope)
}

// TestSyncInstallStateRemovesOrphans tests that sync removes orphaned records.
func TestSyncInstallStateRemovesOrphans(t *testing.T) {
	tempDir := t.TempDir()

	// Create a mock home directory structure (empty skills dir)
	homeDir := filepath.Join(tempDir, "home")
	skillsDir := filepath.Join(homeDir, ".claude", "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0755))

	cfg := setupTestConfig(t)
	database := setupTestDB(t)

	// Create skill in database marked as installed
	skill := &models.Skill{
		ID:          "local-orphan-skill",
		Slug:        "orphan-skill",
		IsLocal:     true,
		IsInstalled: true,
	}
	require.NoError(t, database.CreateSkill(skill))

	// Create orphaned installation record (symlink doesn't exist)
	install := &models.SkillInstallation{
		SkillID:     skill.ID,
		Platform:    "claude",
		Scope:       "global",
		BasePath:    homeDir,
		SymlinkPath: filepath.Join(skillsDir, "orphan-skill"),
	}
	require.NoError(t, database.AddInstallation(install))

	// Override home directory for test
	t.Setenv("HOME", homeDir)

	inst := New(database, cfg)

	// Run sync
	err := inst.SyncInstallState(context.Background())
	require.NoError(t, err)

	// Verify skill is now marked as NOT installed
	updatedSkill, err := database.GetSkill(skill.ID)
	require.NoError(t, err)
	assert.False(t, updatedSkill.IsInstalled)

	// Verify orphaned installation record was removed
	installs, err := database.GetInstallations(skill.ID)
	require.NoError(t, err)
	assert.Empty(t, installs)
}
