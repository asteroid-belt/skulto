package db

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearch_ExcludesSoftDeleted(t *testing.T) {
	db := testDB(t)

	// Create two skills
	skill1 := &models.Skill{ID: "s1", Slug: "react-testing", Title: "React Testing", Content: "react content"}
	skill2 := &models.Skill{ID: "s2", Slug: "react-hooks", Title: "React Hooks", Content: "react hooks"}
	require.NoError(t, db.CreateSkill(skill1))
	require.NoError(t, db.CreateSkill(skill2))

	// Both should appear in search
	results, err := db.Search("react", 10)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Soft-delete one
	require.NoError(t, db.DeleteSkill("s2"))

	// Only one should appear now
	results, err = db.Search("react", 10)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "s1", results[0].ID)
}

func TestSearchByCategory_ExcludesSoftDeleted(t *testing.T) {
	db := testDB(t)

	// Create two skills in the same category
	skill1 := &models.Skill{ID: "cat1", Slug: "go-testing", Title: "Go Testing", Content: "golang test", Category: "backend"}
	skill2 := &models.Skill{ID: "cat2", Slug: "go-mocking", Title: "Go Mocking", Content: "golang mock", Category: "backend"}
	require.NoError(t, db.CreateSkill(skill1))
	require.NoError(t, db.CreateSkill(skill2))

	// Both should appear in search
	results, err := db.SearchByCategory("golang", "backend", 10)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Soft-delete one
	require.NoError(t, db.DeleteSkill("cat2"))

	// Only one should appear now
	results, err = db.SearchByCategory("golang", "backend", 10)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "cat1", results[0].ID)
}

func TestSearchByTag_ExcludesSoftDeleted(t *testing.T) {
	db := testDB(t)

	// Create tag first
	tag := &models.Tag{ID: "testing", Slug: "testing", Name: "Testing"}
	require.NoError(t, db.UpsertTag(tag))

	// Create two skills with the same tag
	skill1 := &models.Skill{ID: "tag1", Slug: "python-testing", Title: "Python Testing", Content: "python test"}
	skill2 := &models.Skill{ID: "tag2", Slug: "python-mocking", Title: "Python Mocking", Content: "python mock"}

	require.NoError(t, db.UpsertSkillWithTags(skill1, []models.Tag{*tag}))
	require.NoError(t, db.UpsertSkillWithTags(skill2, []models.Tag{*tag}))

	// Both should appear in search
	results, err := db.SearchByTag("python", "testing", 10)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Soft-delete one
	require.NoError(t, db.DeleteSkill("tag2"))

	// Only one should appear now
	results, err = db.SearchByTag("python", "testing", 10)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "tag1", results[0].ID)
}

// --- Phase 3C: Source and AuxiliaryFiles Loading Tests ---

func TestGetSkill_LoadsAuxiliaryFiles(t *testing.T) {
	db := testDB(t)

	// Create a skill
	skill := &models.Skill{
		ID:    "skill-with-aux",
		Slug:  "skill-with-aux",
		Title: "Skill with Auxiliary Files",
	}
	require.NoError(t, db.CreateSkill(skill))

	// Create auxiliary files for the skill
	auxFile1 := &models.AuxiliaryFile{
		SkillID:  "skill-with-aux",
		DirType:  models.AuxDirScripts,
		FilePath: "scripts/helper.sh",
		FileName: "helper.sh",
	}
	auxFile2 := &models.AuxiliaryFile{
		SkillID:  "skill-with-aux",
		DirType:  models.AuxDirReferences,
		FilePath: "references/demo.md",
		FileName: "demo.md",
	}
	require.NoError(t, db.Create(auxFile1).Error)
	require.NoError(t, db.Create(auxFile2).Error)

	// Retrieve skill - should have AuxiliaryFiles preloaded
	retrieved, err := db.GetSkill("skill-with-aux")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	// Verify auxiliary files are loaded
	assert.Len(t, retrieved.AuxiliaryFiles, 2, "GetSkill should preload AuxiliaryFiles")

	// Verify the files are correctly associated
	fileNames := make([]string, len(retrieved.AuxiliaryFiles))
	for i, f := range retrieved.AuxiliaryFiles {
		fileNames[i] = f.FileName
	}
	assert.Contains(t, fileNames, "helper.sh")
	assert.Contains(t, fileNames, "demo.md")
}

func TestSearchSkills_LoadsSource(t *testing.T) {
	db := testDB(t)

	// Create source first
	source := &models.Source{
		ID:       "search-source/repo",
		Owner:    "search-source",
		Repo:     "repo",
		FullName: "search-source/repo",
	}
	require.NoError(t, db.CreateSource(source))

	// Create skills with source reference
	sourceID := "search-source/repo"
	skill1 := &models.Skill{
		ID:       "search-skill-1",
		Slug:     "search-skill-1",
		Title:    "Search Skill One",
		Content:  "searchable content here",
		SourceID: &sourceID,
	}
	skill2 := &models.Skill{
		ID:       "search-skill-2",
		Slug:     "search-skill-2",
		Title:    "Search Skill Two",
		Content:  "searchable content here",
		SourceID: &sourceID,
	}
	require.NoError(t, db.CreateSkill(skill1))
	require.NoError(t, db.CreateSkill(skill2))

	// Search for skills
	skills, err := db.SearchSkills("searchable", 10)
	require.NoError(t, err)
	require.Len(t, skills, 2, "Should find 2 skills")

	// Verify Source is loaded for all results
	for _, s := range skills {
		require.NotNil(t, s.Source, "SearchSkills should load Source for skill %s", s.ID)
		assert.Equal(t, "search-source", s.Source.Owner, "Source.Owner should be loaded")
		assert.Equal(t, "repo", s.Source.Repo, "Source.Repo should be loaded")
	}
}

func TestListSkills_LoadsSource(t *testing.T) {
	db := testDB(t)

	// Create source first
	source := &models.Source{
		ID:       "list-source/repo",
		Owner:    "list-source",
		Repo:     "repo",
		FullName: "list-source/repo",
	}
	require.NoError(t, db.CreateSource(source))

	// Create skills with source reference
	sourceID := "list-source/repo"
	skill1 := &models.Skill{
		ID:       "list-skill-1",
		Slug:     "list-skill-1",
		Title:    "List Skill One",
		SourceID: &sourceID,
	}
	skill2 := &models.Skill{
		ID:       "list-skill-2",
		Slug:     "list-skill-2",
		Title:    "List Skill Two",
		SourceID: &sourceID,
	}
	require.NoError(t, db.CreateSkill(skill1))
	require.NoError(t, db.CreateSkill(skill2))

	// List skills
	skills, err := db.ListSkills(10, 0)
	require.NoError(t, err)
	require.Len(t, skills, 2, "Should list 2 skills")

	// Verify Source is loaded for all results
	for _, s := range skills {
		require.NotNil(t, s.Source, "ListSkills should load Source for skill %s", s.ID)
		assert.Equal(t, "list-source", s.Source.Owner, "Source.Owner should be loaded")
		assert.Equal(t, "repo", s.Source.Repo, "Source.Repo should be loaded")
	}
}
