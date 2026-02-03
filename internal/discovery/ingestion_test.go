// internal/discovery/ingestion_test.go
package discovery

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestionService_ValidateSkill_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "my-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.md"), []byte("# My Skill"), 0644))

	svc := NewIngestionService(nil, nil)
	err := svc.ValidateSkill(skillDir)

	assert.NoError(t, err)
}

func TestIngestionService_ValidateSkill_MissingSkillMd(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "my-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	// No skill.md file

	svc := NewIngestionService(nil, nil)
	err := svc.ValidateSkill(skillDir)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "skill.md")
}

func TestIngestionService_ValidateSkill_AcceptsSKILLMD(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "my-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# My Skill"), 0644))

	svc := NewIngestionService(nil, nil)
	err := svc.ValidateSkill(skillDir)

	assert.NoError(t, err)
}

func TestIngestionService_CheckNameConflict_NoConflict(t *testing.T) {
	tmpDir := t.TempDir()
	svc := &IngestionService{destDirOverride: tmpDir}

	conflict, err := svc.CheckNameConflict("nonexistent", "project")

	assert.NoError(t, err)
	assert.False(t, conflict)
}

func TestIngestionService_CheckNameConflict_HasConflict(t *testing.T) {
	tmpDir := t.TempDir()
	existingDir := filepath.Join(tmpDir, "existing-skill")
	require.NoError(t, os.MkdirAll(existingDir, 0755))

	svc := &IngestionService{destDirOverride: tmpDir}

	conflict, err := svc.CheckNameConflict("existing-skill", "project")

	assert.NoError(t, err)
	assert.True(t, conflict)
}

func TestIngestionService_IngestSkill(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source skill
	sourceDir := filepath.Join(tmpDir, "platforms", ".claude", "skills", "my-skill")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "skill.md"), []byte("# My Skill"), 0644))

	// Create skulto destination directory
	skultoDir := filepath.Join(tmpDir, ".skulto", "skills")
	require.NoError(t, os.MkdirAll(skultoDir, 0755))

	ds := models.DiscoveredSkill{
		Platform: "claude",
		Scope:    "project",
		Path:     sourceDir,
		Name:     "my-skill",
	}
	ds.ID = ds.GenerateID()

	svc := &IngestionService{destDirOverride: skultoDir}
	result, err := svc.IngestSkill(context.Background(), &ds)

	require.NoError(t, err)
	assert.Equal(t, "my-skill", result.Name)
	assert.Equal(t, filepath.Join(skultoDir, "my-skill"), result.DestPath)

	// Verify skill was copied
	_, err = os.Stat(filepath.Join(skultoDir, "my-skill", "skill.md"))
	assert.NoError(t, err)

	// Verify original is now a symlink
	lstat, err := os.Lstat(sourceDir)
	require.NoError(t, err)
	assert.True(t, lstat.Mode()&os.ModeSymlink != 0)
}

func TestIngestionService_IngestSkill_WithMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source skill with multiple files
	sourceDir := filepath.Join(tmpDir, "platforms", ".claude", "skills", "complex-skill")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "skill.md"), []byte("# Complex Skill"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "helper.py"), []byte("print('hello')"), 0644))

	// Create subdirectory with files
	subDir := filepath.Join(sourceDir, "templates")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "template.txt"), []byte("template content"), 0644))

	skultoDir := filepath.Join(tmpDir, ".skulto", "skills")
	require.NoError(t, os.MkdirAll(skultoDir, 0755))

	ds := models.DiscoveredSkill{
		Platform: "claude",
		Scope:    "project",
		Path:     sourceDir,
		Name:     "complex-skill",
	}
	ds.ID = ds.GenerateID()

	svc := &IngestionService{destDirOverride: skultoDir}
	result, err := svc.IngestSkill(context.Background(), &ds)

	require.NoError(t, err)
	assert.Equal(t, "complex-skill", result.Name)

	// Verify all files were copied
	_, err = os.Stat(filepath.Join(skultoDir, "complex-skill", "skill.md"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(skultoDir, "complex-skill", "helper.py"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(skultoDir, "complex-skill", "templates", "template.txt"))
	assert.NoError(t, err)
}

func TestIngestionService_IngestSkill_ValidationFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source skill without skill.md
	sourceDir := filepath.Join(tmpDir, "platforms", ".claude", "skills", "invalid-skill")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	// No skill.md file

	skultoDir := filepath.Join(tmpDir, ".skulto", "skills")
	require.NoError(t, os.MkdirAll(skultoDir, 0755))

	ds := models.DiscoveredSkill{
		Platform: "claude",
		Scope:    "project",
		Path:     sourceDir,
		Name:     "invalid-skill",
	}
	ds.ID = ds.GenerateID()

	svc := &IngestionService{destDirOverride: skultoDir}
	result, err := svc.IngestSkill(context.Background(), &ds)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "skill.md")
}

func TestIngestionService_IngestSkill_SymlinkPointsToDestination(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source skill
	sourceDir := filepath.Join(tmpDir, "platforms", ".claude", "skills", "my-skill")
	require.NoError(t, os.MkdirAll(sourceDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "skill.md"), []byte("# My Skill"), 0644))

	skultoDir := filepath.Join(tmpDir, ".skulto", "skills")
	require.NoError(t, os.MkdirAll(skultoDir, 0755))

	ds := models.DiscoveredSkill{
		Platform: "claude",
		Scope:    "project",
		Path:     sourceDir,
		Name:     "my-skill",
	}
	ds.ID = ds.GenerateID()

	svc := &IngestionService{destDirOverride: skultoDir}
	_, err := svc.IngestSkill(context.Background(), &ds)
	require.NoError(t, err)

	// Read symlink target
	target, err := os.Readlink(sourceDir)
	require.NoError(t, err)

	// Resolve relative target to absolute
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(sourceDir), target)
	}
	target = filepath.Clean(target)

	// Should point to skulto destination
	expectedDest := filepath.Join(skultoDir, "my-skill")
	assert.Equal(t, expectedDest, target)
}
