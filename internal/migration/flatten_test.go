package migration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlattenFilesystem_AlreadyFlat(t *testing.T) {
	skillsDir := t.TempDir()
	result := &MigrationResult{}

	// Create already-flat skill with SKILL.md
	flatSkill := filepath.Join(skillsDir, "my-skill")
	require.NoError(t, os.MkdirAll(flatSkill, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(flatSkill, "SKILL.md"),
		[]byte("# Test"), 0644))

	err := flattenFilesystem(skillsDir, result)
	require.NoError(t, err)

	// Should be skipped, not moved
	assert.Equal(t, 1, result.SkillsSkipped)
	assert.Equal(t, 0, result.SkillsMoved)
	assert.FileExists(t, filepath.Join(flatSkill, "SKILL.md"))
}

func TestFlattenFilesystem_LowercaseSkillMd(t *testing.T) {
	skillsDir := t.TempDir()
	result := &MigrationResult{}

	// Create skill with lowercase skill.md (like teleport/)
	flatSkill := filepath.Join(skillsDir, "teleport")
	require.NoError(t, os.MkdirAll(flatSkill, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(flatSkill, "skill.md"), // lowercase
		[]byte("# Test"), 0644))

	err := flattenFilesystem(skillsDir, result)
	require.NoError(t, err)

	// Should be recognized as flat and skipped
	assert.Equal(t, 1, result.SkillsSkipped)
	assert.Equal(t, 0, result.SkillsMoved)
}

func TestFlattenFilesystem_NestedSkill(t *testing.T) {
	skillsDir := t.TempDir()
	result := &MigrationResult{}

	// Create nested skill: category/slug/SKILL.md
	nestedSkill := filepath.Join(skillsDir, "jokes", "dadjoke")
	require.NoError(t, os.MkdirAll(nestedSkill, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(nestedSkill, "SKILL.md"),
		[]byte("# Dad Joke"), 0644))

	err := flattenFilesystem(skillsDir, result)
	require.NoError(t, err)

	// Should be moved to flat location
	assert.Equal(t, 1, result.SkillsMoved)
	assert.FileExists(t, filepath.Join(skillsDir, "dadjoke", "SKILL.md"))
	assert.NoDirExists(t, filepath.Join(skillsDir, "jokes")) // Category removed
}

func TestFlattenFilesystem_CategoryWithSpace(t *testing.T) {
	skillsDir := t.TempDir()
	result := &MigrationResult{}

	// Create skill with space in category name
	nestedSkill := filepath.Join(skillsDir, "software engineering", "code-review")
	require.NoError(t, os.MkdirAll(nestedSkill, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(nestedSkill, "SKILL.md"),
		[]byte("# Code Review"), 0644))

	err := flattenFilesystem(skillsDir, result)
	require.NoError(t, err)

	// Should be moved to flat location
	assert.Equal(t, 1, result.SkillsMoved)
	assert.FileExists(t, filepath.Join(skillsDir, "code-review", "SKILL.md"))
}

func TestFlattenFilesystem_NameCollision(t *testing.T) {
	skillsDir := t.TempDir()
	result := &MigrationResult{}

	// Create flat skill first
	flatSkill := filepath.Join(skillsDir, "my-skill")
	require.NoError(t, os.MkdirAll(flatSkill, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(flatSkill, "SKILL.md"),
		[]byte("# Flat Skill"), 0644))

	// Create nested skill with same name in different category
	nestedSkill := filepath.Join(skillsDir, "python", "my-skill")
	require.NoError(t, os.MkdirAll(nestedSkill, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(nestedSkill, "SKILL.md"),
		[]byte("# Python Skill"), 0644))

	err := flattenFilesystem(skillsDir, result)
	require.NoError(t, err)

	// Original flat skill preserved
	assert.FileExists(t, filepath.Join(flatSkill, "SKILL.md"))
	// Nested skill moved with category suffix
	assert.FileExists(t, filepath.Join(skillsDir, "my-skill-python", "SKILL.md"))
	assert.Equal(t, 1, result.SkillsSkipped) // flat one
	assert.Equal(t, 1, result.SkillsMoved)   // nested one with suffix
}

func TestFlattenFilesystem_DualNesting(t *testing.T) {
	skillsDir := t.TempDir()
	result := &MigrationResult{}

	// Create dual nesting like superplan/SKILL.md + superplan/superplan/SKILL.md
	rootSkill := filepath.Join(skillsDir, "superplan")
	nestedSkill := filepath.Join(skillsDir, "superplan", "superplan")
	require.NoError(t, os.MkdirAll(nestedSkill, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(rootSkill, "SKILL.md"),
		[]byte("# Root Superplan"), 0644))
	require.NoError(t, os.WriteFile(
		filepath.Join(nestedSkill, "SKILL.md"),
		[]byte("# Nested Superplan"), 0644))

	err := flattenFilesystem(skillsDir, result)
	require.NoError(t, err)

	// Root skill preserved (already flat)
	assert.FileExists(t, filepath.Join(rootSkill, "SKILL.md"))
	// Nested skill moved with suffix to avoid collision
	assert.FileExists(t, filepath.Join(skillsDir, "superplan-superplan", "SKILL.md"))
}

func TestFlattenFilesystem_Idempotent(t *testing.T) {
	skillsDir := t.TempDir()

	// Create nested skill
	nestedSkill := filepath.Join(skillsDir, "category", "skill")
	require.NoError(t, os.MkdirAll(nestedSkill, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(nestedSkill, "SKILL.md"),
		[]byte("# Test"), 0644))

	// Run migration twice
	result1 := &MigrationResult{}
	err := flattenFilesystem(skillsDir, result1)
	require.NoError(t, err)
	assert.Equal(t, 1, result1.SkillsMoved)

	result2 := &MigrationResult{}
	err = flattenFilesystem(skillsDir, result2)
	require.NoError(t, err)

	// Second run should skip (already flat)
	assert.Equal(t, 0, result2.SkillsMoved)
	assert.Equal(t, 1, result2.SkillsSkipped)
}

func TestFlattenFilesystem_NonexistentDir(t *testing.T) {
	result := &MigrationResult{}
	err := flattenFilesystem("/nonexistent/path", result)
	require.NoError(t, err) // Should handle gracefully
	assert.Equal(t, 0, result.SkillsMoved)
	assert.Equal(t, 0, result.SkillsSkipped)
}

func TestFlattenFilesystem_PreservesAuxiliaryDirs(t *testing.T) {
	skillsDir := t.TempDir()
	result := &MigrationResult{}

	// Create nested skill with auxiliary directories
	nestedSkill := filepath.Join(skillsDir, "category", "my-skill")
	require.NoError(t, os.MkdirAll(filepath.Join(nestedSkill, "references"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(nestedSkill, "scripts"), 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(nestedSkill, "SKILL.md"),
		[]byte("# Skill with aux"), 0644))
	require.NoError(t, os.WriteFile(
		filepath.Join(nestedSkill, "references", "guide.md"),
		[]byte("# Guide"), 0644))
	require.NoError(t, os.WriteFile(
		filepath.Join(nestedSkill, "scripts", "setup.sh"),
		[]byte("#!/bin/bash"), 0644))

	err := flattenFilesystem(skillsDir, result)
	require.NoError(t, err)

	// Skill moved with all auxiliary dirs preserved
	assert.FileExists(t, filepath.Join(skillsDir, "my-skill", "SKILL.md"))
	assert.FileExists(t, filepath.Join(skillsDir, "my-skill", "references", "guide.md"))
	assert.FileExists(t, filepath.Join(skillsDir, "my-skill", "scripts", "setup.sh"))
}

func TestSanitizeForPath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"software engineering", "software-engineering"},
		{"test/slash", "test-slash"},
		{"multi  space", "multi-space"},
		{"clean", "clean"},
		{"-leading-trailing-", "leading-trailing"},
		{"colon:test", "colon-test"},
		{"question?mark", "question-mark"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, sanitizeForPath(tt.input))
		})
	}
}

func TestIsSkillDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Test uppercase SKILL.md
	upperDir := filepath.Join(tmpDir, "upper")
	require.NoError(t, os.MkdirAll(upperDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(upperDir, "SKILL.md"), []byte("test"), 0644))
	assert.True(t, isSkillDir(upperDir))

	// Test lowercase skill.md
	lowerDir := filepath.Join(tmpDir, "lower")
	require.NoError(t, os.MkdirAll(lowerDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(lowerDir, "skill.md"), []byte("test"), 0644))
	assert.True(t, isSkillDir(lowerDir))

	// Test no skill file
	emptyDir := filepath.Join(tmpDir, "empty")
	require.NoError(t, os.MkdirAll(emptyDir, 0755))
	assert.False(t, isSkillDir(emptyDir))
}

func TestIsEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	// Test empty dir
	emptyDir := filepath.Join(tmpDir, "empty")
	require.NoError(t, os.MkdirAll(emptyDir, 0755))
	assert.True(t, isEmpty(emptyDir))

	// Test non-empty dir
	nonEmptyDir := filepath.Join(tmpDir, "nonempty")
	require.NoError(t, os.MkdirAll(nonEmptyDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nonEmptyDir, "file.txt"), []byte("test"), 0644))
	assert.False(t, isEmpty(nonEmptyDir))
}
