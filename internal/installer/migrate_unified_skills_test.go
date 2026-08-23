package installer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// migrationEnv carries the common objects a migration test needs.
type migrationEnv struct {
	tmp      string
	baseDir  string
	cfg      *config.Config
	database *db.DB
}

// newMigrationEnv prepares a temporary home-like environment with a fresh DB.
func newMigrationEnv(t *testing.T) *migrationEnv {
	t.Helper()
	tmp := t.TempDir()
	baseDir := filepath.Join(tmp, ".agents", "skulto")
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "skills"), 0755))
	cfg := &config.Config{BaseDir: baseDir}
	database, err := db.New(db.Config{Path: filepath.Join(baseDir, "skulto.db")})
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	return &migrationEnv{
		tmp:      tmp,
		baseDir:  baseDir,
		cfg:      cfg,
		database: database,
	}
}

// seedSkill creates a skills row directly via the DB.
func seedSkill(t *testing.T, database *db.DB, id, slug, filePath string, isLocal bool) {
	t.Helper()
	s := &models.Skill{
		ID:       id,
		Slug:     slug,
		Title:    slug,
		IsLocal:  isLocal,
		FilePath: filePath,
	}
	require.NoError(t, database.CreateSkill(s))
}

// seedInstallation creates a skill_installations row directly via the DB.
func seedInstallation(t *testing.T, database *db.DB, skillID, platform, scope, basePath, symlinkPath string) {
	t.Helper()
	inst := &models.SkillInstallation{
		SkillID:     skillID,
		Platform:    platform,
		Scope:       scope,
		BasePath:    basePath,
		SymlinkPath: symlinkPath,
		InstalledAt: time.Now(),
	}
	require.NoError(t, database.AddInstallation(inst))
}

// countSkillTagRows returns the number of skill_tags rows for the given skill.
func countSkillTagRows(t *testing.T, database *db.DB, skillID string) int {
	t.Helper()
	var count int64
	err := database.Raw("SELECT COUNT(*) FROM skill_tags WHERE skill_id = ?", skillID).Scan(&count).Error
	require.NoError(t, err)
	return int(count)
}

// markerPath returns the unified-skills migration marker path under cfg.BaseDir.
func markerPath(cfg *config.Config) string {
	return filepath.Join(cfg.BaseDir, ".migrations", unifiedSkillsMigrationMarker)
}

// writeSKILL creates a skill directory with a SKILL.md file inside.
func writeSKILL(t *testing.T, dir, contents string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(contents), 0644))
}

// evalSymlinkTarget follows a single-hop symlink and returns the EvalSymlinks-resolved path.
func evalSymlinkTarget(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	require.NoError(t, err)
	return resolved
}

// resolvedAbs returns an absolute, symlink-resolved form of path. Matches
// how the migration normalizes BaseDir and candidate paths so tests don't
// fail when /var ↔ /private/var-style aliasing is in play.
func resolvedAbs(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	require.NoError(t, err)
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}

// TestUnifiedSkillMigration_PurgesCwdRows asserts D2 semantics.
func TestUnifiedSkillMigration_PurgesCwdRows(t *testing.T) {
	env := newMigrationEnv(t)

	// Seed a fake project dir representing where the cwd skill lived.
	cwdSkillDir := filepath.Join(env.tmp, "someproject", ".skulto", "skills", "foo")
	writeSKILL(t, cwdSkillDir, "---\ntitle: Foo\n---\n")

	// Seed the cwd skill row — FilePath is a file path (legacy) pointing at skill.md.
	legacyFilePath := filepath.Join(cwdSkillDir, "skill.md")
	require.NoError(t, os.WriteFile(legacyFilePath, []byte("---\ntitle: Foo\n---\n"), 0644))
	seedSkill(t, env.database, "cwd-foo", "foo", legacyFilePath, true)

	// Attach a tag to the skill so we can assert skill_tags cleanup.
	tag := models.Tag{ID: "tool:cwd-fixture", Name: "cwd-fixture", Slug: "cwd-fixture", Category: "tool"}
	require.NoError(t, env.database.Create(&tag).Error)
	require.NoError(t, env.database.AddTagToSkill("cwd-foo", tag.ID))
	require.Equal(t, 1, countSkillTagRows(t, env.database, "cwd-foo"))

	// Seed a claude-style symlink on disk that points at the cwd skill file.
	claudeDir := filepath.Join(env.tmp, "fakeclaude", "skills")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))
	claudeSymlink := filepath.Join(claudeDir, "foo")
	require.NoError(t, os.Symlink(legacyFilePath, claudeSymlink))

	seedInstallation(t, env.database, "cwd-foo", "claude", "global", filepath.Dir(claudeDir), claudeSymlink)

	// Run migration.
	report, err := MigrateToUnifiedLocalSkills(env.database, env.cfg)
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, 1, report.CwdPurged)

	// skill row gone
	var skills []models.Skill
	require.NoError(t, env.database.Where("id = ?", "cwd-foo").Find(&skills).Error)
	assert.Empty(t, skills)

	// installations gone
	insts, err := env.database.GetInstallations("cwd-foo")
	require.NoError(t, err)
	assert.Empty(t, insts)

	// skill_tags row gone
	assert.Equal(t, 0, countSkillTagRows(t, env.database, "cwd-foo"))

	// on-disk symlink removed
	_, statErr := os.Lstat(claudeSymlink)
	assert.True(t, os.IsNotExist(statErr), "claude symlink should have been removed, got: %v", statErr)

	// marker written
	_, err = os.Stat(markerPath(env.cfg))
	assert.NoError(t, err)
}

// TestUnifiedSkillMigration_NormalizesFilePathEndingInSkillMd asserts D3 normalization.
func TestUnifiedSkillMigration_NormalizesFilePathEndingInSkillMd(t *testing.T) {
	env := newMigrationEnv(t)

	skillDir := filepath.Join(env.baseDir, "skills", "foo")
	writeSKILL(t, skillDir, "---\ntitle: Foo\n---\n")
	filePath := filepath.Join(skillDir, "skill.md")
	require.NoError(t, os.WriteFile(filePath, []byte("---\ntitle: Foo\n---\n"), 0644))

	seedSkill(t, env.database, "local-foo", "foo", filePath, true)

	report, err := MigrateToUnifiedLocalSkills(env.database, env.cfg)
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, 1, report.NormalizedFilePaths)
	assert.Equal(t, 0, report.Migrated)

	updated, err := env.database.GetSkill("local-foo")
	require.NoError(t, err)
	require.NotNil(t, updated)

	assert.Equal(t, resolvedAbs(t, skillDir), resolvedAbs(t, updated.FilePath))
}

// TestUnifiedSkillMigration_NormalizationSkipsPathsOutsideBaseDir asserts D3 only
// touches rows that live inside BaseDir — the outside-BaseDir case is D5's job.
func TestUnifiedSkillMigration_NormalizationSkipsPathsOutsideBaseDir(t *testing.T) {
	env := newMigrationEnv(t)

	// Project dir is OUTSIDE env.baseDir.
	projectSkillDir := filepath.Join(env.tmp, "project", ".skulto", "skills", "foo")
	writeSKILL(t, projectSkillDir, "---\ntitle: Foo\n---\n")
	filePath := filepath.Join(projectSkillDir, "skill.md")
	require.NoError(t, os.WriteFile(filePath, []byte("---\ntitle: Foo\n---\n"), 0644))

	seedSkill(t, env.database, "local-foo", "foo", filePath, true)

	report, err := MigrateToUnifiedLocalSkills(env.database, env.cfg)
	require.NoError(t, err)
	require.NotNil(t, report)

	// D3 did NOT normalize: it's outside BaseDir (D5 handles it).
	assert.Equal(t, 0, report.NormalizedFilePaths)
	// D5 DID migrate it (no collision), so report reflects that.
	assert.Equal(t, 1, report.Migrated)
	assert.Equal(t, 0, report.Collisions)

	// The row's file_path should now point at the global dir (set by D5),
	// not the legacy file path or the legacy project dir.
	updated, err := env.database.GetSkill("local-foo")
	require.NoError(t, err)
	assert.Equal(t,
		resolvedAbs(t, filepath.Join(env.baseDir, "skills", "foo")),
		resolvedAbs(t, updated.FilePath))
}

// TestUnifiedSkillMigration_CopiesProjectScopedSkillsToBaseDir asserts the D5 no-collision branch.
func TestUnifiedSkillMigration_CopiesProjectScopedSkillsToBaseDir(t *testing.T) {
	env := newMigrationEnv(t)

	projectSkillDir := filepath.Join(env.tmp, "project", ".skulto", "skills", "foo")
	writeSKILL(t, projectSkillDir, "---\ntitle: Foo Project\n---\n")

	seedSkill(t, env.database, "local-foo", "foo", projectSkillDir, true)

	// Seed a pre-existing claude symlink pointing at the project dir.
	claudeDir := filepath.Join(env.tmp, "fakeclaude", "skills")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))
	claudeSymlink := filepath.Join(claudeDir, "foo")
	require.NoError(t, os.Symlink(projectSkillDir, claudeSymlink))
	seedInstallation(t, env.database, "local-foo", "claude", "global", filepath.Dir(claudeDir), claudeSymlink)

	report, err := MigrateToUnifiedLocalSkills(env.database, env.cfg)
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, 1, report.Migrated)
	assert.Equal(t, 0, report.Collisions)

	// New dir exists with SKILL.md
	newDir := filepath.Join(env.baseDir, "skills", "foo")
	_, err = os.Stat(filepath.Join(newDir, "SKILL.md"))
	require.NoError(t, err)

	// Original project location is now a symlink resolving to newDir.
	projectInfo, err := os.Lstat(projectSkillDir)
	require.NoError(t, err)
	assert.NotZero(t, projectInfo.Mode()&os.ModeSymlink, "project dir should now be a symlink")
	assert.Equal(t, evalSymlinkTarget(t, newDir), evalSymlinkTarget(t, projectSkillDir))

	// DB file_path rewritten to newDir.
	updated, err := env.database.GetSkill("local-foo")
	require.NoError(t, err)
	assert.Equal(t, resolvedAbs(t, newDir), resolvedAbs(t, updated.FilePath))

	// claude symlink now resolves to newDir — NOT the old project path, NOT the skills parent.
	assert.Equal(t, evalSymlinkTarget(t, newDir), evalSymlinkTarget(t, claudeSymlink),
		"claude symlink must resolve to the new skill dir precisely")
}

// TestUnifiedSkillMigration_CollisionKeepsGlobalAndRewritesRow asserts the D5 collision branch.
func TestUnifiedSkillMigration_CollisionKeepsGlobalAndRewritesRow(t *testing.T) {
	env := newMigrationEnv(t)

	// Pre-existing global skill (the "keep" side of keep-global).
	globalDir := filepath.Join(env.baseDir, "skills", "foo")
	writeSKILL(t, globalDir, "---\ntitle: Foo Global\n---\n")

	// Project copy with different content.
	projectSkillDir := filepath.Join(env.tmp, "project", ".skulto", "skills", "foo")
	writeSKILL(t, projectSkillDir, "---\ntitle: Foo Project\n---\n")

	seedSkill(t, env.database, "local-foo", "foo", projectSkillDir, true)

	claudeDir := filepath.Join(env.tmp, "fakeclaude", "skills")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))
	claudeSymlink := filepath.Join(claudeDir, "foo")
	require.NoError(t, os.Symlink(projectSkillDir, claudeSymlink))
	seedInstallation(t, env.database, "local-foo", "claude", "global", filepath.Dir(claudeDir), claudeSymlink)

	report, err := MigrateToUnifiedLocalSkills(env.database, env.cfg)
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, 0, report.Migrated)
	assert.Equal(t, 1, report.Collisions)

	// Neither dir modified on disk: the project copy is still a real directory (NOT a symlink).
	projInfo, err := os.Lstat(projectSkillDir)
	require.NoError(t, err)
	assert.Zero(t, projInfo.Mode()&os.ModeSymlink, "project dir must remain a real directory in the collision branch")
	assert.True(t, projInfo.IsDir())

	// Global dir still has its original content (prove it wasn't overwritten).
	globalContent, err := os.ReadFile(filepath.Join(globalDir, "SKILL.md"))
	require.NoError(t, err)
	assert.Contains(t, string(globalContent), "Foo Global")

	// DB file_path rewritten to globalDir (compare symlink-resolved to tolerate
	// /var ↔ /private/var aliasing on macOS).
	updated, err := env.database.GetSkill("local-foo")
	require.NoError(t, err)
	assert.Equal(t, resolvedAbs(t, globalDir), resolvedAbs(t, updated.FilePath))

	// claude symlink now resolves to globalDir precisely.
	assert.Equal(t, evalSymlinkTarget(t, globalDir), evalSymlinkTarget(t, claudeSymlink))
}

// TestUnifiedSkillMigration_IdempotentAfterMarker asserts the marker gates the pass.
func TestUnifiedSkillMigration_IdempotentAfterMarker(t *testing.T) {
	env := newMigrationEnv(t)

	// First run: seed cwd-foo.
	cwdSkillDir := filepath.Join(env.tmp, "projectA", ".skulto", "skills", "foo")
	writeSKILL(t, cwdSkillDir, "---\ntitle: Foo\n---\n")
	seedSkill(t, env.database, "cwd-foo", "foo", filepath.Join(cwdSkillDir, "skill.md"), true)

	report, err := MigrateToUnifiedLocalSkills(env.database, env.cfg)
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, 1, report.CwdPurged)

	// Confirm the cwd-foo row is gone.
	var after []models.Skill
	require.NoError(t, env.database.Where("id = ?", "cwd-foo").Find(&after).Error)
	assert.Empty(t, after)

	// Simulate the user somehow ending up with a new cwd-bar row post-migration.
	seedSkill(t, env.database, "cwd-bar", "bar", "/nonexistent/skill.md", true)

	// Second run: should be a no-op (nil report) because the marker exists.
	report2, err := MigrateToUnifiedLocalSkills(env.database, env.cfg)
	require.NoError(t, err)
	assert.Nil(t, report2)

	// cwd-bar still present (not touched).
	bar, err := env.database.GetSkill("cwd-bar")
	require.NoError(t, err)
	require.NotNil(t, bar)
	assert.Equal(t, "cwd-bar", bar.ID)
}

// TestUnifiedSkillMigration_MarkerNotWrittenOnError asserts on pass-level failure
// the marker is absent so the next launch retries.
func TestUnifiedSkillMigration_MarkerNotWrittenOnError(t *testing.T) {
	// Invalid config (empty BaseDir) forces a pass-level error.
	badCfg := &config.Config{BaseDir: ""}
	// We don't need a real DB for this error branch — the function should fail before
	// touching it. Pass nil and make sure the function returns without panicking.
	report, err := MigrateToUnifiedLocalSkills(nil, badCfg)
	require.Error(t, err)
	assert.Nil(t, report)

	// Now with a working setup the marker appears.
	env := newMigrationEnv(t)
	_, err = os.Stat(markerPath(env.cfg))
	assert.True(t, os.IsNotExist(err), "marker should not exist before first run")

	report2, err := MigrateToUnifiedLocalSkills(env.database, env.cfg)
	require.NoError(t, err)
	require.NotNil(t, report2)

	_, err = os.Stat(markerPath(env.cfg))
	assert.NoError(t, err, "marker must be written after a successful run")
}

// TestUnifiedSkillMigration_StaleLockFileDoesNotBlock asserts that a lingering
// lock-file sentinel (e.g. a prior crashed run left the file on disk but the
// kernel has already released the flock) does NOT prevent the migration from
// running. This is the crash-recovery case flagged by codex round 2.
func TestUnifiedSkillMigration_StaleLockFileDoesNotBlock(t *testing.T) {
	env := newMigrationEnv(t)

	// Create the lock file on disk as if a prior run had crashed.
	markerDir := filepath.Join(env.cfg.BaseDir, ".migrations")
	require.NoError(t, os.MkdirAll(markerDir, 0755))
	lockFile := filepath.Join(markerDir, unifiedSkillsMigrationMarker+".lock")
	require.NoError(t, os.WriteFile(lockFile, []byte{}, 0644))

	// Seed something so the migration has visible work to do.
	seedSkill(t, env.database, "cwd-stale", "stale", "/nonexistent/skill.md", true)

	report, err := MigrateToUnifiedLocalSkills(env.database, env.cfg)
	require.NoError(t, err)
	require.NotNil(t, report, "stale lock sentinel must not block the migration")
	assert.Equal(t, 1, report.CwdPurged)

	// Marker written.
	_, err = os.Stat(markerPath(env.cfg))
	assert.NoError(t, err)
}

// TestUnifiedSkillMigration_CollisionRejectsUnsafeNewDir asserts the D5
// collision predicate: if <BaseDir>/skills/<slug> exists but is NOT a real
// managed directory under BaseDir (e.g. a regular file, a symlink to outside
// BaseDir), the migration refuses to treat it as a collision and skips the
// row rather than rewriting the DB to an invalid target.
func TestUnifiedSkillMigration_CollisionRejectsUnsafeNewDir(t *testing.T) {
	env := newMigrationEnv(t)

	// Project-scoped source.
	projectSkillDir := filepath.Join(env.tmp, "project", ".skulto", "skills", "foo")
	writeSKILL(t, projectSkillDir, "---\ntitle: Foo\n---\n")
	seedSkill(t, env.database, "local-foo", "foo", projectSkillDir, true)

	// BaseDir/skills/foo is a regular file instead of a directory.
	newPath := filepath.Join(env.baseDir, "skills", "foo")
	require.NoError(t, os.MkdirAll(filepath.Dir(newPath), 0755))
	require.NoError(t, os.WriteFile(newPath, []byte("bogus file"), 0644))

	report, err := MigrateToUnifiedLocalSkills(env.database, env.cfg)
	require.NoError(t, err)
	require.NotNil(t, report)

	// Neither collision nor migrated — the row was skipped.
	assert.Equal(t, 0, report.Collisions)
	assert.Equal(t, 0, report.Migrated)

	// DB row left untouched.
	unchanged, err := env.database.GetSkill("local-foo")
	require.NoError(t, err)
	require.NotNil(t, unchanged)
	assert.Equal(t, resolvedAbs(t, projectSkillDir), resolvedAbs(t, unchanged.FilePath))

	// The bogus file is untouched.
	content, err := os.ReadFile(newPath)
	require.NoError(t, err)
	assert.Equal(t, "bogus file", string(content))

	// Marker still written (per-row skips don't block the pass).
	_, err = os.Stat(markerPath(env.cfg))
	assert.NoError(t, err)
}

// TestUnifiedSkillMigration_BaseDirAccessedViaSymlinkAlias asserts that when
// cfg.BaseDir is accessed through a symlink alias (the /var ↔ /private/var
// class of problem), D3 normalization and D5 classification still work
// because absBase is EvalSymlinks-resolved at entry.
func TestUnifiedSkillMigration_BaseDirAccessedViaSymlinkAlias(t *testing.T) {
	tmp := t.TempDir()
	// Real BaseDir on disk.
	realBase := filepath.Join(tmp, "real", ".agents", "skulto")
	require.NoError(t, os.MkdirAll(filepath.Join(realBase, "skills"), 0755))

	// Symlink alias that points at the real base.
	aliasParent := filepath.Join(tmp, "alias")
	require.NoError(t, os.MkdirAll(aliasParent, 0755))
	aliasBase := filepath.Join(aliasParent, "skulto-alias")
	require.NoError(t, os.Symlink(realBase, aliasBase))

	// cfg.BaseDir uses the alias.
	cfg := &config.Config{BaseDir: aliasBase}
	database, err := db.New(db.Config{Path: filepath.Join(realBase, "skulto.db")})
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })

	// Seed a row whose file_path also reaches the real base via the alias,
	// and ends in /skill.md so D3 normalization will try to run.
	skillDirAlias := filepath.Join(aliasBase, "skills", "foo")
	require.NoError(t, os.MkdirAll(skillDirAlias, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDirAlias, "skill.md"),
		[]byte("---\ntitle: Foo\n---\n"), 0644))
	seedSkill(t, database, "local-foo", "foo",
		filepath.Join(skillDirAlias, "skill.md"), true)

	// D5-candidate row: lives outside BaseDir (in real terms).
	outside := filepath.Join(tmp, "outside-project", ".skulto", "skills", "bar")
	writeSKILL(t, outside, "---\ntitle: Bar\n---\n")
	seedSkill(t, database, "local-bar", "bar", outside, true)

	report, err := MigrateToUnifiedLocalSkills(database, cfg)
	require.NoError(t, err)
	require.NotNil(t, report)

	// D3 should have normalized local-foo because it DOES resolve under BaseDir.
	assert.Equal(t, 1, report.NormalizedFilePaths)
	// D5 should have migrated local-bar because it resolves outside.
	assert.Equal(t, 1, report.Migrated)

	// local-foo file_path now points at the skill directory.
	fooUpdated, err := database.GetSkill("local-foo")
	require.NoError(t, err)
	assert.Equal(t, resolvedAbs(t, skillDirAlias), resolvedAbs(t, fooUpdated.FilePath))

	// local-bar migrated into <realBase>/skills/bar.
	barUpdated, err := database.GetSkill("local-bar")
	require.NoError(t, err)
	assert.Equal(t, resolvedAbs(t, filepath.Join(realBase, "skills", "bar")),
		resolvedAbs(t, barUpdated.FilePath))
}

// TestUnifiedSkillMigration_D2LeavesDanglingSymlinksAlone asserts that D2's
// symlink purge follows the spec strictly: only symlinks whose target
// RESOLVES outside BaseDir are removed. Dangling (broken) symlinks are left
// alone — we cannot verify they violate the invariant.
func TestUnifiedSkillMigration_D2LeavesDanglingSymlinksAlone(t *testing.T) {
	env := newMigrationEnv(t)

	// Seed cwd-foo row; file_path points at a file that no longer exists.
	missingPath := filepath.Join(env.tmp, "gone", "skill.md")
	seedSkill(t, env.database, "cwd-foo", "foo", missingPath, true)

	// Dangling symlink on disk at a claude-like location.
	claudeDir := filepath.Join(env.tmp, "fakeclaude", "skills")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))
	dangling := filepath.Join(claudeDir, "foo")
	require.NoError(t, os.Symlink(missingPath, dangling))

	seedInstallation(t, env.database, "cwd-foo", "claude", "global", filepath.Dir(claudeDir), dangling)

	report, err := MigrateToUnifiedLocalSkills(env.database, env.cfg)
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, 1, report.CwdPurged)

	// The dangling symlink is STILL present — per spec D2, only resolves-outside-BaseDir links are removed.
	info, err := os.Lstat(dangling)
	require.NoError(t, err, "dangling symlink must NOT be removed by D2")
	assert.NotZero(t, info.Mode()&os.ModeSymlink)
}

// TestUnifiedSkillMigration_PerRowErrorsDoNotAbortPass asserts per-row failures
// are logged-and-skipped and do NOT prevent the marker from being written.
func TestUnifiedSkillMigration_PerRowErrorsDoNotAbortPass(t *testing.T) {
	env := newMigrationEnv(t)

	// Valid row: project dir with SKILL.md that should be migrated.
	projectSkillDir := filepath.Join(env.tmp, "project", ".skulto", "skills", "goodone")
	writeSKILL(t, projectSkillDir, "---\ntitle: Good\n---\n")
	seedSkill(t, env.database, "local-goodone", "goodone", projectSkillDir, true)

	// Invalid row: points at a non-existent directory.
	seedSkill(t, env.database, "local-missing", "missing",
		filepath.Join(env.tmp, "does-not-exist", "skill.md"), true)

	report, err := MigrateToUnifiedLocalSkills(env.database, env.cfg)
	require.NoError(t, err)
	require.NotNil(t, report)

	// Valid row: migrated.
	assert.Equal(t, 1, report.Migrated)
	updated, err := env.database.GetSkill("local-goodone")
	require.NoError(t, err)
	assert.Equal(t, resolvedAbs(t, filepath.Join(env.baseDir, "skills", "goodone")),
		resolvedAbs(t, updated.FilePath))

	// Invalid row: row still present, file_path unchanged.
	stillMissing, err := env.database.GetSkill("local-missing")
	require.NoError(t, err)
	require.NotNil(t, stillMissing)
	// The row is left alone — not deleted, not rewritten.
	assert.Contains(t, stillMissing.FilePath, "does-not-exist")

	// Marker is written despite the per-row skip.
	_, err = os.Stat(markerPath(env.cfg))
	assert.NoError(t, err)
}
