package installer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReconcile_RepoSymlink_CreatesRecord(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()
	cfg := setupTestConfig(t)
	inst := New(database, cfg)
	cwd := t.TempDir()

	source := &models.Source{ID: "org/repo", Owner: "org", Repo: "repo", FullName: "org/repo"}
	require.NoError(t, database.Save(source).Error)
	skill := &models.Skill{ID: "s1", Slug: "teach", SourceID: &source.ID}
	require.NoError(t, database.UpsertSkill(skill))

	repoDir := filepath.Join(cfg.BaseDir, "repositories", "org", "repo", "skills", "teach")
	require.NoError(t, os.MkdirAll(repoDir, 0755))

	claudeSkills := filepath.Join(cwd, ".claude", "skills")
	require.NoError(t, os.MkdirAll(claudeSkills, 0755))
	require.NoError(t, os.Symlink(repoDir, filepath.Join(claudeSkills, "teach")))

	result, err := inst.ReconcileProjectSkills(cwd)
	require.NoError(t, err)

	assert.Len(t, result.Reconciled, 1)
	assert.Equal(t, "teach", result.Reconciled[0].Slug)
	assert.Equal(t, PlatformClaude, result.Reconciled[0].Platform)
	assert.Empty(t, result.Unmanaged)

	installed, err := database.IsInstalledAt(skill.ID, "claude", "project", cwd)
	require.NoError(t, err)
	assert.True(t, installed)
}

func TestReconcile_NonSkultoSymlink_Unmanaged(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()
	cfg := setupTestConfig(t)
	inst := New(database, cfg)
	cwd := t.TempDir()

	externalDir := t.TempDir()
	claudeSkills := filepath.Join(cwd, ".claude", "skills")
	require.NoError(t, os.MkdirAll(claudeSkills, 0755))
	require.NoError(t, os.Symlink(externalDir, filepath.Join(claudeSkills, "external")))

	result, err := inst.ReconcileProjectSkills(cwd)
	require.NoError(t, err)

	assert.Empty(t, result.Reconciled)
	assert.Len(t, result.Unmanaged, 1)
	assert.Equal(t, "external", result.Unmanaged[0].Name)
	assert.Equal(t, PlatformClaude, result.Unmanaged[0].Platform)
}

func TestReconcile_PlainDir_Skipped(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()
	cfg := setupTestConfig(t)
	inst := New(database, cfg)
	cwd := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(cwd, ".claude", "skills", "manual"), 0755))

	result, err := inst.ReconcileProjectSkills(cwd)
	require.NoError(t, err)

	assert.Empty(t, result.Reconciled)
	assert.Empty(t, result.Unmanaged, "plain directories should be silently skipped")
}

func TestReconcile_AlreadyTracked_NoChange(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()
	cfg := setupTestConfig(t)
	inst := New(database, cfg)
	cwd := t.TempDir()

	source := &models.Source{ID: "org/repo", Owner: "org", Repo: "repo", FullName: "org/repo"}
	require.NoError(t, database.Save(source).Error)
	skill := &models.Skill{ID: "s1", Slug: "teach", SourceID: &source.ID}
	require.NoError(t, database.UpsertSkill(skill))

	symlinkPath := filepath.Join(cwd, ".claude", "skills", "teach")
	existing := &models.SkillInstallation{
		SkillID: skill.ID, Platform: "claude", Scope: "project",
		BasePath: cwd, SymlinkPath: symlinkPath,
	}
	require.NoError(t, database.AddInstallation(existing))

	repoDir := filepath.Join(cfg.BaseDir, "repositories", "org", "repo", "skills", "teach")
	require.NoError(t, os.MkdirAll(repoDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Dir(symlinkPath), 0755))
	require.NoError(t, os.Symlink(repoDir, symlinkPath))

	result, err := inst.ReconcileProjectSkills(cwd)
	require.NoError(t, err)
	assert.Empty(t, result.Reconciled)
}

func TestReconcile_UnknownSlug_Skipped(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()
	cfg := setupTestConfig(t)
	inst := New(database, cfg)
	cwd := t.TempDir()

	repoDir := filepath.Join(cfg.BaseDir, "repositories", "org", "repo", "skills", "unknown")
	require.NoError(t, os.MkdirAll(repoDir, 0755))
	claudeSkills := filepath.Join(cwd, ".claude", "skills")
	require.NoError(t, os.MkdirAll(claudeSkills, 0755))
	require.NoError(t, os.Symlink(repoDir, filepath.Join(claudeSkills, "unknown")))

	result, err := inst.ReconcileProjectSkills(cwd)
	require.NoError(t, err)
	assert.Empty(t, result.Reconciled)
	assert.Empty(t, result.Unmanaged)
}

func TestReconcile_Idempotent(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()
	cfg := setupTestConfig(t)
	inst := New(database, cfg)
	cwd := t.TempDir()

	source := &models.Source{ID: "org/repo", Owner: "org", Repo: "repo", FullName: "org/repo"}
	require.NoError(t, database.Save(source).Error)
	skill := &models.Skill{ID: "s1", Slug: "teach", SourceID: &source.ID}
	require.NoError(t, database.UpsertSkill(skill))

	repoDir := filepath.Join(cfg.BaseDir, "repositories", "org", "repo", "skills", "teach")
	require.NoError(t, os.MkdirAll(repoDir, 0755))
	claudeSkills := filepath.Join(cwd, ".claude", "skills")
	require.NoError(t, os.MkdirAll(claudeSkills, 0755))
	require.NoError(t, os.Symlink(repoDir, filepath.Join(claudeSkills, "teach")))

	r1, _ := inst.ReconcileProjectSkills(cwd)
	assert.Len(t, r1.Reconciled, 1)

	r2, _ := inst.ReconcileProjectSkills(cwd)
	assert.Empty(t, r2.Reconciled)
}

func TestReconcile_NoPlatformDir_Empty(t *testing.T) {
	database := setupTestDB(t)
	defer func() { _ = database.Close() }()
	cfg := setupTestConfig(t)
	inst := New(database, cfg)

	result, err := inst.ReconcileProjectSkills(t.TempDir())
	require.NoError(t, err)
	assert.Empty(t, result.Reconciled)
	assert.Empty(t, result.Unmanaged)
}
