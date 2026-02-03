package db

import (
	"testing"
	"time"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDB_UpsertDiscoveredSkill(t *testing.T) {
	db := testDB(t)

	ds := models.DiscoveredSkill{
		Platform:     "claude",
		Scope:        "project",
		Path:         "/test/.claude/skills/my-skill",
		Name:         "my-skill",
		DiscoveredAt: time.Now(),
	}
	ds.ID = ds.GenerateID()

	err := db.UpsertDiscoveredSkill(&ds)
	require.NoError(t, err)

	// Verify it was saved
	var saved models.DiscoveredSkill
	err = db.First(&saved, "id = ?", ds.ID).Error
	require.NoError(t, err)
	assert.Equal(t, ds.Name, saved.Name)
}

func TestDB_UpsertDiscoveredSkill_UpdatesExisting(t *testing.T) {
	db := testDB(t)

	ds := models.DiscoveredSkill{
		Platform:     "claude",
		Scope:        "project",
		Path:         "/test/.claude/skills/my-skill",
		Name:         "my-skill",
		DiscoveredAt: time.Now(),
	}
	ds.ID = ds.GenerateID()

	// First insert
	require.NoError(t, db.UpsertDiscoveredSkill(&ds))

	// Update with same path (should update, not create new)
	require.NoError(t, db.UpsertDiscoveredSkill(&ds))

	// Should only have one record
	var count int64
	db.Model(&models.DiscoveredSkill{}).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestDB_ListDiscoveredSkills(t *testing.T) {
	db := testDB(t)

	// Insert some discoveries
	for i, name := range []string{"skill-a", "skill-b", "skill-c"} {
		ds := models.DiscoveredSkill{
			Platform:     "claude",
			Scope:        "project",
			Path:         "/test/.claude/skills/" + name,
			Name:         name,
			DiscoveredAt: time.Now().Add(time.Duration(i) * time.Hour),
		}
		ds.ID = ds.GenerateID()
		require.NoError(t, db.UpsertDiscoveredSkill(&ds))
	}

	skills, err := db.ListDiscoveredSkills()
	require.NoError(t, err)
	assert.Len(t, skills, 3)
}

func TestDB_ListUnnotifiedDiscoveredSkills(t *testing.T) {
	db := testDB(t)
	now := time.Now()

	// Insert notified and unnotified discoveries
	notified := models.DiscoveredSkill{
		Platform:     "claude",
		Scope:        "project",
		Path:         "/test/.claude/skills/notified",
		Name:         "notified",
		DiscoveredAt: now,
		NotifiedAt:   &now,
	}
	notified.ID = notified.GenerateID()
	require.NoError(t, db.UpsertDiscoveredSkill(&notified))

	unnotified := models.DiscoveredSkill{
		Platform:     "claude",
		Scope:        "project",
		Path:         "/test/.claude/skills/unnotified",
		Name:         "unnotified",
		DiscoveredAt: now,
		NotifiedAt:   nil,
	}
	unnotified.ID = unnotified.GenerateID()
	require.NoError(t, db.UpsertDiscoveredSkill(&unnotified))

	skills, err := db.ListUnnotifiedDiscoveredSkills()
	require.NoError(t, err)
	assert.Len(t, skills, 1)
	assert.Equal(t, "unnotified", skills[0].Name)
}

func TestDB_ListDiscoveredSkillsByScope(t *testing.T) {
	db := testDB(t)

	// Insert discoveries with different scopes
	projectSkill := models.DiscoveredSkill{
		Platform:     "claude",
		Scope:        "project",
		Path:         "/test/.claude/skills/project-skill",
		Name:         "project-skill",
		DiscoveredAt: time.Now(),
	}
	projectSkill.ID = projectSkill.GenerateID()
	require.NoError(t, db.UpsertDiscoveredSkill(&projectSkill))

	globalSkill := models.DiscoveredSkill{
		Platform:     "claude",
		Scope:        "global",
		Path:         "/home/.claude/skills/global-skill",
		Name:         "global-skill",
		DiscoveredAt: time.Now(),
	}
	globalSkill.ID = globalSkill.GenerateID()
	require.NoError(t, db.UpsertDiscoveredSkill(&globalSkill))

	// List by project scope
	projectSkills, err := db.ListDiscoveredSkillsByScope("project")
	require.NoError(t, err)
	assert.Len(t, projectSkills, 1)
	assert.Equal(t, "project-skill", projectSkills[0].Name)

	// List by global scope
	globalSkills, err := db.ListDiscoveredSkillsByScope("global")
	require.NoError(t, err)
	assert.Len(t, globalSkills, 1)
	assert.Equal(t, "global-skill", globalSkills[0].Name)
}

func TestDB_MarkDiscoveredSkillsNotified(t *testing.T) {
	db := testDB(t)

	ds := models.DiscoveredSkill{
		Platform:     "claude",
		Scope:        "project",
		Path:         "/test/.claude/skills/my-skill",
		Name:         "my-skill",
		DiscoveredAt: time.Now(),
	}
	ds.ID = ds.GenerateID()
	require.NoError(t, db.UpsertDiscoveredSkill(&ds))

	err := db.MarkDiscoveredSkillsNotified([]string{ds.ID})
	require.NoError(t, err)

	// Verify it was marked
	var saved models.DiscoveredSkill
	err = db.First(&saved, "id = ?", ds.ID).Error
	require.NoError(t, err)
	assert.NotNil(t, saved.NotifiedAt)
}

func TestDB_DeleteDiscoveredSkill(t *testing.T) {
	db := testDB(t)

	ds := models.DiscoveredSkill{
		Platform:     "claude",
		Scope:        "project",
		Path:         "/test/.claude/skills/my-skill",
		Name:         "my-skill",
		DiscoveredAt: time.Now(),
	}
	ds.ID = ds.GenerateID()
	require.NoError(t, db.UpsertDiscoveredSkill(&ds))

	err := db.DeleteDiscoveredSkill(ds.ID)
	require.NoError(t, err)

	// Verify it was deleted
	var count int64
	db.Model(&models.DiscoveredSkill{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestDB_GetDiscoveredSkillByPath(t *testing.T) {
	db := testDB(t)

	ds := models.DiscoveredSkill{
		Platform:     "claude",
		Scope:        "project",
		Path:         "/test/.claude/skills/my-skill",
		Name:         "my-skill",
		DiscoveredAt: time.Now(),
	}
	ds.ID = ds.GenerateID()
	require.NoError(t, db.UpsertDiscoveredSkill(&ds))

	found, err := db.GetDiscoveredSkillByPath(ds.Path)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, ds.Name, found.Name)
}

func TestDB_GetDiscoveredSkillByPath_NotFound(t *testing.T) {
	db := testDB(t)

	found, err := db.GetDiscoveredSkillByPath("/nonexistent/path")
	require.Error(t, err)
	assert.Nil(t, found)
}

func TestDB_GetDiscoveredSkillByName(t *testing.T) {
	db := testDB(t)

	ds := models.DiscoveredSkill{
		Platform:     "claude",
		Scope:        "project",
		Path:         "/test/.claude/skills/my-skill",
		Name:         "my-skill",
		DiscoveredAt: time.Now(),
	}
	ds.ID = ds.GenerateID()
	require.NoError(t, db.UpsertDiscoveredSkill(&ds))

	found, err := db.GetDiscoveredSkillByName("my-skill", "project")
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, ds.Path, found.Path)
}

func TestDB_GetDiscoveredSkillByName_NotFound(t *testing.T) {
	db := testDB(t)

	found, err := db.GetDiscoveredSkillByName("nonexistent", "project")
	require.Error(t, err)
	assert.Nil(t, found)
}

func TestDB_CountDiscoveredSkills(t *testing.T) {
	db := testDB(t)

	for _, name := range []string{"skill-a", "skill-b"} {
		ds := models.DiscoveredSkill{
			Platform:     "claude",
			Scope:        "project",
			Path:         "/test/.claude/skills/" + name,
			Name:         name,
			DiscoveredAt: time.Now(),
		}
		ds.ID = ds.GenerateID()
		require.NoError(t, db.UpsertDiscoveredSkill(&ds))
	}

	count, err := db.CountDiscoveredSkills()
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestDB_CleanupStaleDiscoveries(t *testing.T) {
	db := testDB(t)

	// Insert discoveries
	valid := models.DiscoveredSkill{
		Platform:     "claude",
		Scope:        "project",
		Path:         "/valid/path",
		Name:         "valid-skill",
		DiscoveredAt: time.Now(),
	}
	valid.ID = valid.GenerateID()
	require.NoError(t, db.UpsertDiscoveredSkill(&valid))

	stale := models.DiscoveredSkill{
		Platform:     "claude",
		Scope:        "project",
		Path:         "/stale/path",
		Name:         "stale-skill",
		DiscoveredAt: time.Now(),
	}
	stale.ID = stale.GenerateID()
	require.NoError(t, db.UpsertDiscoveredSkill(&stale))

	// Path checker that marks /stale/path as invalid
	pathChecker := func(path string) bool {
		return path == "/valid/path"
	}

	err := db.CleanupStaleDiscoveries(pathChecker)
	require.NoError(t, err)

	// Should only have the valid skill left
	skills, err := db.ListDiscoveredSkills()
	require.NoError(t, err)
	assert.Len(t, skills, 1)
	assert.Equal(t, "valid-skill", skills[0].Name)
}
