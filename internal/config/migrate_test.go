package config

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/glebarez/go-sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateBaseDir_FreshInstall(t *testing.T) {
	home := t.TempDir()

	err := migrateBaseDir(home)
	require.NoError(t, err)

	// Neither dir should exist (ensureDirectories handles creation later)
	_, err = os.Stat(filepath.Join(home, ".skulto"))
	assert.True(t, os.IsNotExist(err))
}

func TestMigrateBaseDir_AlreadyMigrated(t *testing.T) {
	home := t.TempDir()

	// ~/.agents/skulto has marker file from prior migration
	newDir := filepath.Join(home, ".agents", "skulto")
	require.NoError(t, os.MkdirAll(newDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(newDir, "skulto.db"), []byte("data"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(newDir, migrationMarker), []byte("migrated"), 0644))

	// ~/.skulto also exists (could be restored by user or leftover)
	oldDir := filepath.Join(home, ".skulto")
	require.NoError(t, os.MkdirAll(oldDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(oldDir, "skulto.db"), []byte("data"), 0644))

	err := migrateBaseDir(home)
	require.NoError(t, err)

	// Old dir should NOT be removed — marker means migration is done,
	// but we don't delete what might be user-restored data
	_, err = os.Stat(oldDir)
	assert.NoError(t, err)

	// New dir still intact
	_, err = os.Stat(filepath.Join(newDir, "skulto.db"))
	assert.NoError(t, err)
}

func TestMigrateBaseDir_MovesData(t *testing.T) {
	home := t.TempDir()

	oldDir := filepath.Join(home, ".skulto")
	require.NoError(t, os.MkdirAll(filepath.Join(oldDir, "repositories", "owner", "repo"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(oldDir, "skulto.db"), []byte("database"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(oldDir, "favorites.json"), []byte("{}"), 0644))

	err := migrateBaseDir(home)
	require.NoError(t, err)

	newDir := filepath.Join(home, ".agents", "skulto")

	// Old dir removed
	_, err = os.Stat(oldDir)
	assert.True(t, os.IsNotExist(err))

	// Marker exists
	_, err = os.Stat(filepath.Join(newDir, migrationMarker))
	assert.NoError(t, err)

	// Data present
	data, err := os.ReadFile(filepath.Join(newDir, "skulto.db"))
	require.NoError(t, err)
	assert.Equal(t, "database", string(data))

	data, err = os.ReadFile(filepath.Join(newDir, "favorites.json"))
	require.NoError(t, err)
	assert.Equal(t, "{}", string(data))

	// Nested dirs preserved
	_, err = os.Stat(filepath.Join(newDir, "repositories", "owner", "repo"))
	assert.NoError(t, err)
}

func TestMigrateBaseDir_SkipsSymlink(t *testing.T) {
	home := t.TempDir()

	realDir := filepath.Join(home, "real-skulto")
	require.NoError(t, os.MkdirAll(realDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(realDir, "skulto.db"), []byte("data"), 0644))

	oldDir := filepath.Join(home, ".skulto")
	require.NoError(t, os.Symlink(realDir, oldDir))

	err := migrateBaseDir(home)
	require.NoError(t, err)

	// Symlink should still exist (not removed or followed)
	info, err := os.Lstat(oldDir)
	require.NoError(t, err)
	assert.True(t, info.Mode()&os.ModeSymlink != 0)
}

func TestMigrateBaseDir_PartialMigration(t *testing.T) {
	home := t.TempDir()

	// ~/.skulto has data
	oldDir := filepath.Join(home, ".skulto")
	require.NoError(t, os.MkdirAll(oldDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(oldDir, "skulto.db"), []byte("database"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(oldDir, "favorites.json"), []byte("{}"), 0644))

	// ~/.agents/skulto exists with partial contents (no marker)
	newDir := filepath.Join(home, ".agents", "skulto")
	require.NoError(t, os.MkdirAll(newDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(newDir, "skulto.db"), []byte("database"), 0644))

	err := migrateBaseDir(home)
	require.NoError(t, err)

	// Old dir removed
	_, err = os.Stat(oldDir)
	assert.True(t, os.IsNotExist(err))

	// Missing file was copied
	data, err := os.ReadFile(filepath.Join(newDir, "favorites.json"))
	require.NoError(t, err)
	assert.Equal(t, "{}", string(data))

	// Marker written
	_, err = os.Stat(filepath.Join(newDir, migrationMarker))
	assert.NoError(t, err)
}

func TestMigrateBaseDir_SymlinksRewritten(t *testing.T) {
	home := t.TempDir()

	// Create ~/.skulto with repo and DB
	oldDir := filepath.Join(home, ".skulto")
	repoDir := filepath.Join(oldDir, "repositories", "owner", "repo", "skills", "test-skill")
	require.NoError(t, os.MkdirAll(repoDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "SKILL.md"), []byte("# test"), 0644))

	// Create DB with skill_installations record
	dbPath := filepath.Join(oldDir, "skulto.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE skills (id TEXT, file_path TEXT)`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE skill_installations (id TEXT, skill_id TEXT NOT NULL, platform TEXT NOT NULL, scope TEXT NOT NULL, base_path TEXT NOT NULL, symlink_path TEXT, installed_at DATETIME)`)
	require.NoError(t, err)
	symlinkPath := filepath.Join(home, ".claude", "skills", "test-skill")
	_, err = db.Exec(`INSERT INTO skill_installations (id, skill_id, platform, scope, base_path, symlink_path) VALUES (?, ?, ?, ?, ?, ?)`,
		"test-id", "test-skill", "claude", "global", home, symlinkPath)
	require.NoError(t, err)
	_ = db.Close()

	// Create platform symlink pointing into old dir
	require.NoError(t, os.MkdirAll(filepath.Dir(symlinkPath), 0755))
	require.NoError(t, os.Symlink(repoDir, symlinkPath))

	err = migrateBaseDir(home)
	require.NoError(t, err)

	// Old dir removed
	_, err = os.Stat(oldDir)
	assert.True(t, os.IsNotExist(err))

	// Symlink now points to new location
	target, err := os.Readlink(symlinkPath)
	require.NoError(t, err)
	assert.Contains(t, target, filepath.Join(".agents", "skulto"))
	assert.NotContains(t, target, filepath.Join(home, ".skulto"))

	// Symlink still resolves
	_, err = os.Stat(filepath.Join(symlinkPath, "SKILL.md"))
	assert.NoError(t, err)
}

func TestMigrateBaseDir_AllInstalledSkillsSurvive(t *testing.T) {
	home := t.TempDir()
	oldDir := filepath.Join(home, ".skulto")

	// Create 3 repos with skill content
	repos := []struct {
		owner, repo, skill string
	}{
		{"asteroid-belt", "skills", "teach"},
		{"asteroid-belt", "skills", "superbuild"},
		{"other-org", "other-repo", "my-skill"},
	}
	for _, r := range repos {
		dir := filepath.Join(oldDir, "repositories", r.owner, r.repo, "skills", r.skill)
		require.NoError(t, os.MkdirAll(dir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# "+r.skill), 0644))
	}

	// Create DB
	dbPath := filepath.Join(oldDir, "skulto.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE skills (id TEXT, file_path TEXT)`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE skill_installations (id TEXT, skill_id TEXT NOT NULL, platform TEXT NOT NULL, scope TEXT NOT NULL, base_path TEXT NOT NULL, symlink_path TEXT, installed_at DATETIME)`)
	require.NoError(t, err)

	// Install skills across multiple platforms (global + project scope)
	type install struct {
		id, skillID, platform, scope, basePath string
		repoOwner, repoName, skillSlug         string
	}
	projectDir := filepath.Join(home, "myproject")
	installs := []install{
		{"i1", "s1", "claude", "global", home, "asteroid-belt", "skills", "teach"},
		{"i2", "s1", "codex", "global", home, "asteroid-belt", "skills", "teach"},
		{"i3", "s2", "claude", "global", home, "asteroid-belt", "skills", "superbuild"},
		{"i4", "s2", "roo", "global", home, "asteroid-belt", "skills", "superbuild"},
		{"i5", "s3", "claude", "project", projectDir, "other-org", "other-repo", "my-skill"},
	}

	var symlinkPaths []string
	for _, inst := range installs {
		var skillsRel string
		switch inst.platform {
		case "claude":
			skillsRel = ".claude/skills"
		case "codex":
			skillsRel = ".codex/skills"
		case "roo":
			skillsRel = ".roo/skills"
		}
		symlinkPath := filepath.Join(inst.basePath, skillsRel, inst.skillSlug)
		symlinkPaths = append(symlinkPaths, symlinkPath)
		_, err = db.Exec(`INSERT INTO skill_installations (id, skill_id, platform, scope, base_path, symlink_path) VALUES (?, ?, ?, ?, ?, ?)`,
			inst.id, inst.skillID, inst.platform, inst.scope, inst.basePath, symlinkPath)
		require.NoError(t, err)

		// Create the actual symlink
		repoDir := filepath.Join(oldDir, "repositories", inst.repoOwner, inst.repoName, "skills", inst.skillSlug)
		require.NoError(t, os.MkdirAll(filepath.Dir(symlinkPath), 0755))
		require.NoError(t, os.Symlink(repoDir, symlinkPath))
	}
	_ = db.Close()

	// Verify pre-migration: all symlinks work
	for _, sp := range symlinkPaths {
		_, err := os.Stat(filepath.Join(sp, "SKILL.md"))
		require.NoError(t, err, "pre-migration: symlink %s should resolve", sp)
	}

	// Run migration
	err = migrateBaseDir(home)
	require.NoError(t, err)

	// Post-migration: every symlink still resolves
	for _, sp := range symlinkPaths {
		_, err := os.Stat(filepath.Join(sp, "SKILL.md"))
		assert.NoError(t, err, "post-migration: symlink %s should still resolve", sp)
	}

	// Every symlink target points to new dir, not old
	for _, sp := range symlinkPaths {
		target, err := os.Readlink(sp)
		require.NoError(t, err)
		assert.Contains(t, target, filepath.Join(".agents", "skulto"),
			"symlink %s should point to new dir", sp)
		assert.NotContains(t, target, filepath.Join(home, ".skulto"),
			"symlink %s should not point to old dir", sp)
	}

	// DB still has all installation records
	newDB, err := sql.Open("sqlite", filepath.Join(home, ".agents", "skulto", "skulto.db"))
	require.NoError(t, err)
	defer func() { _ = newDB.Close() }()
	var count int
	err = newDB.QueryRow("SELECT COUNT(*) FROM skill_installations").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, len(installs), count, "all installation records should survive migration")
}

func TestMigrateDBPaths(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "skulto.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	_, err = db.Exec(`CREATE TABLE skills (id TEXT, file_path TEXT)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO skills (id, file_path) VALUES (?, ?)`,
		"local-test", "/Users/test/.skulto/skills/test/skill.md")
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO skills (id, file_path) VALUES (?, ?)`,
		"remote-test", "skills/remote/SKILL.md")
	require.NoError(t, err)
	// Path containing project name "skulto" — must NOT be rewritten
	_, err = db.Exec(`INSERT INTO skills (id, file_path) VALUES (?, ?)`,
		"project-test", "/Users/test/codes/skulto/.claude/skills/foo/SKILL.md")
	require.NoError(t, err)

	err = migrateDBPaths(dbPath)
	require.NoError(t, err)

	// Local path updated
	var filePath string
	err = db.QueryRow(`SELECT file_path FROM skills WHERE id = ?`, "local-test").Scan(&filePath)
	require.NoError(t, err)
	assert.Equal(t, "/Users/test/.agents/skulto/skills/test/skill.md", filePath)

	// Remote path NOT changed
	err = db.QueryRow(`SELECT file_path FROM skills WHERE id = ?`, "remote-test").Scan(&filePath)
	require.NoError(t, err)
	assert.Equal(t, "skills/remote/SKILL.md", filePath)

	// Project path NOT changed (contains "skulto" but not "/.skulto/")
	err = db.QueryRow(`SELECT file_path FROM skills WHERE id = ?`, "project-test").Scan(&filePath)
	require.NoError(t, err)
	assert.Equal(t, "/Users/test/codes/skulto/.claude/skills/foo/SKILL.md", filePath)
}

func TestRewriteSymlinks(t *testing.T) {
	home := t.TempDir()
	oldDir := filepath.Join(home, ".skulto")
	newDir := filepath.Join(home, ".agents", "skulto")

	// Create target dirs
	oldRepoDir := filepath.Join(oldDir, "repositories", "owner", "repo")
	newRepoDir := filepath.Join(newDir, "repositories", "owner", "repo")
	require.NoError(t, os.MkdirAll(oldRepoDir, 0755))
	require.NoError(t, os.MkdirAll(newRepoDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(newRepoDir, "SKILL.md"), []byte("# skill"), 0644))

	// Create symlink
	claudeDir := filepath.Join(home, ".claude", "skills")
	require.NoError(t, os.MkdirAll(claudeDir, 0755))
	linkPath := filepath.Join(claudeDir, "my-skill")
	require.NoError(t, os.Symlink(oldRepoDir, linkPath))

	// Create DB with installation record
	dbPath := filepath.Join(newDir, "skulto.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE skill_installations (id TEXT, skill_id TEXT NOT NULL, platform TEXT NOT NULL, scope TEXT NOT NULL, base_path TEXT NOT NULL, symlink_path TEXT, installed_at DATETIME)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO skill_installations (id, skill_id, platform, scope, base_path, symlink_path) VALUES (?, ?, ?, ?, ?, ?)`,
		"id1", "skill1", "claude", "global", home, linkPath)
	require.NoError(t, err)
	_ = db.Close()

	err = rewriteSymlinks(dbPath, oldDir, newDir)
	require.NoError(t, err)

	// Symlink rewritten
	target, err := os.Readlink(linkPath)
	require.NoError(t, err)
	assert.Contains(t, target, filepath.Join(".agents", "skulto"))
	assert.NotContains(t, target, filepath.Join(home, ".skulto"))
}

func TestDefaultBaseDir_UsesAgentsDir(t *testing.T) {
	dir := DefaultBaseDir()
	assert.Contains(t, dir, filepath.Join(".agents", "skulto"))
}
