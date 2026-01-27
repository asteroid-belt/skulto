package installer

import (
	"context"
	"testing"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestService(t *testing.T) (*InstallService, *db.DB) {
	t.Helper()
	tmpDir := t.TempDir()
	database, err := db.New(db.Config{Path: tmpDir + "/test.db"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })

	cfg := &config.Config{}
	service := NewInstallService(database, cfg, nil) // nil telemetry for tests
	return service, database
}

func TestNewInstallService(t *testing.T) {
	service, _ := setupTestService(t)

	assert.NotNil(t, service)
	assert.NotNil(t, service.installer)
	assert.NotNil(t, service.db)
}

func TestInstallService_DetectPlatforms(t *testing.T) {
	service, _ := setupTestService(t)
	ctx := context.Background()

	platforms, err := service.DetectPlatforms(ctx)
	require.NoError(t, err)

	// Should return all known platforms with detection status
	assert.GreaterOrEqual(t, len(platforms), 6) // claude, cursor, windsurf, copilot, codex, opencode

	// Each platform should have required fields (DetectedPlatform type)
	for _, p := range platforms {
		assert.NotEmpty(t, p.ID, "Platform ID should not be empty")
		assert.NotEmpty(t, p.Name, "Platform Name should not be empty")
		assert.NotEmpty(t, p.Path, "Platform Path should not be empty")
	}
}

func TestInstallService_Install(t *testing.T) {
	service, database := setupTestService(t)
	ctx := context.Background()

	// Seed a test skill with source
	source := &models.Source{ID: "src-1", FullName: "test/repo", URL: "https://github.com/test/repo"}
	require.NoError(t, database.CreateSource(source))

	skill := &models.Skill{
		ID:       "skill-1",
		Slug:     "test-skill",
		Title:    "Test Skill",
		SourceID: &source.ID,
	}
	require.NoError(t, database.CreateSkill(skill))

	t.Run("returns error for unknown skill", func(t *testing.T) {
		opts := InstallOptions{Confirm: true}
		_, err := service.Install(ctx, "nonexistent", opts)
		assert.Error(t, err)
	})
}

func TestInstallService_GetInstallLocations(t *testing.T) {
	service, _ := setupTestService(t)
	ctx := context.Background()

	// For a skill that doesn't exist, should return empty slice
	locations, err := service.GetInstallLocations(ctx, "nonexistent-slug")
	require.NoError(t, err)
	assert.Empty(t, locations)
}

func TestInstallService_Uninstall(t *testing.T) {
	service, database := setupTestService(t)
	ctx := context.Background()

	// Seed a test skill
	skill := &models.Skill{
		ID:    "skill-1",
		Slug:  "test-skill",
		Title: "Test Skill",
	}
	require.NoError(t, database.CreateSkill(skill))

	t.Run("returns error for unknown skill", func(t *testing.T) {
		err := service.Uninstall(ctx, "nonexistent", nil)
		assert.Error(t, err)
	})

	t.Run("uninstall with no locations is no-op", func(t *testing.T) {
		err := service.UninstallAll(ctx, "test-skill")
		// Should not error even if nothing to uninstall
		assert.NoError(t, err)
	})
}

func TestInstallService_InstallBatch(t *testing.T) {
	service, database := setupTestService(t)
	ctx := context.Background()

	// Seed test skills
	for i := 1; i <= 3; i++ {
		skill := &models.Skill{
			ID:    "skill-" + string(rune('0'+i)),
			Slug:  "test-skill-" + string(rune('0'+i)),
			Title: "Test Skill " + string(rune('0'+i)),
		}
		require.NoError(t, database.CreateSkill(skill))
	}

	t.Run("handles mixed success and failure", func(t *testing.T) {
		slugs := []string{"test-skill-1", "nonexistent", "test-skill-2"}
		opts := InstallOptions{Confirm: true}

		results := service.InstallBatch(ctx, slugs, opts)
		assert.Len(t, results, 3)

		// First and third should have skill, second should have error
		assert.NotNil(t, results[0].Skill)
		assert.Nil(t, results[1].Skill)
		assert.NotEmpty(t, results[1].Errors)
		assert.NotNil(t, results[2].Skill)
	})
}

func TestInstallService_GetInstalledSkillsSummary(t *testing.T) {
	service, database := setupTestService(t)
	ctx := context.Background()

	t.Run("returns empty slice when no installations", func(t *testing.T) {
		summary, err := service.GetInstalledSkillsSummary(ctx)
		require.NoError(t, err)
		assert.Empty(t, summary)
	})

	t.Run("returns skills with their installation locations", func(t *testing.T) {
		// Seed test skills
		skill1 := &models.Skill{ID: "skill-1", Slug: "teach", Title: "Teach"}
		skill2 := &models.Skill{ID: "skill-2", Slug: "superplan", Title: "Superplan"}
		require.NoError(t, database.CreateSkill(skill1))
		require.NoError(t, database.CreateSkill(skill2))

		// Add installations
		require.NoError(t, database.AddInstallation(&models.SkillInstallation{
			SkillID:  "skill-1",
			Platform: "claude",
			Scope:    "global",
			BasePath: "/home/test",
		}))
		require.NoError(t, database.AddInstallation(&models.SkillInstallation{
			SkillID:  "skill-2",
			Platform: "claude",
			Scope:    "global",
			BasePath: "/home/test",
		}))
		require.NoError(t, database.AddInstallation(&models.SkillInstallation{
			SkillID:  "skill-2",
			Platform: "cursor",
			Scope:    "project",
			BasePath: "/projects/myapp",
		}))

		summary, err := service.GetInstalledSkillsSummary(ctx)
		require.NoError(t, err)
		assert.Len(t, summary, 2)

		// Results should be sorted by slug
		assert.Equal(t, "superplan", summary[0].Slug)
		assert.Equal(t, "Superplan", summary[0].Title)
		assert.Equal(t, "teach", summary[1].Slug)
		assert.Equal(t, "Teach", summary[1].Title)

		// Check superplan locations (claude global + cursor project)
		assert.Len(t, summary[0].Locations, 2)
		assert.Contains(t, summary[0].Locations, PlatformClaude)
		assert.Contains(t, summary[0].Locations, PlatformCursor)
		assert.Contains(t, summary[0].Locations[PlatformClaude], ScopeGlobal)
		assert.Contains(t, summary[0].Locations[PlatformCursor], ScopeProject)

		// Check teach locations (claude global only)
		assert.Len(t, summary[1].Locations, 1)
		assert.Contains(t, summary[1].Locations, PlatformClaude)
		assert.Contains(t, summary[1].Locations[PlatformClaude], ScopeGlobal)
	})

	t.Run("handles skill with both global and project on same platform", func(t *testing.T) {
		// Clear previous data by creating fresh service
		service2, database2 := setupTestService(t)

		skill := &models.Skill{ID: "skill-3", Slug: "docker-expert", Title: "Docker Expert"}
		require.NoError(t, database2.CreateSkill(skill))

		// Install to both global and project on same platform
		require.NoError(t, database2.AddInstallation(&models.SkillInstallation{
			SkillID:  "skill-3",
			Platform: "codex",
			Scope:    "global",
			BasePath: "/home/test",
		}))
		require.NoError(t, database2.AddInstallation(&models.SkillInstallation{
			SkillID:  "skill-3",
			Platform: "codex",
			Scope:    "project",
			BasePath: "/projects/myapp",
		}))

		summary, err := service2.GetInstalledSkillsSummary(ctx)
		require.NoError(t, err)
		require.Len(t, summary, 1)

		// Should have codex with both scopes
		assert.Equal(t, "docker-expert", summary[0].Slug)
		assert.Contains(t, summary[0].Locations, PlatformCodex)
		scopes := summary[0].Locations[PlatformCodex]
		assert.Len(t, scopes, 2)
		assert.Contains(t, scopes, ScopeGlobal)
		assert.Contains(t, scopes, ScopeProject)
	})
}
