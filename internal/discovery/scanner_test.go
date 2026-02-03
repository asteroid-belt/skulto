// internal/discovery/scanner_test.go
package discovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScannerService_ScanDirectory_FindsUnmanagedSkills(t *testing.T) {
	// Setup: create temp directory with skills
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".claude", "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0755))

	// Create an unmanaged skill directory
	skillDir := filepath.Join(skillsDir, "my-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "skill.md"), []byte("# My Skill"), 0644))

	scanner := NewScannerService()
	results, err := scanner.ScanDirectory(skillsDir, "claude", "project")

	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "my-skill", results[0].Name)
	assert.Equal(t, "claude", results[0].Platform)
	assert.Equal(t, "project", results[0].Scope)
	assert.Equal(t, skillDir, results[0].Path)
}

func TestScannerService_ScanDirectory_SkipsSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".claude", "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0755))

	// Create a symlinked skill
	targetDir := filepath.Join(tmpDir, "skulto", "skills", "linked-skill")
	require.NoError(t, os.MkdirAll(targetDir, 0755))

	symlinkPath := filepath.Join(skillsDir, "linked-skill")
	require.NoError(t, os.Symlink(targetDir, symlinkPath))

	scanner := NewScannerService()
	results, err := scanner.ScanDirectory(skillsDir, "claude", "project")

	require.NoError(t, err)
	assert.Len(t, results, 0) // Symlinks should be skipped
}

func TestScannerService_ScanDirectory_SkipsFiles(t *testing.T) {
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, ".claude", "skills")
	require.NoError(t, os.MkdirAll(skillsDir, 0755))

	// Create a file (not a directory)
	require.NoError(t, os.WriteFile(filepath.Join(skillsDir, "not-a-skill.txt"), []byte("test"), 0644))

	scanner := NewScannerService()
	results, err := scanner.ScanDirectory(skillsDir, "claude", "project")

	require.NoError(t, err)
	assert.Len(t, results, 0) // Files should be skipped
}

func TestScannerService_ScanDirectory_NonexistentDir(t *testing.T) {
	scanner := NewScannerService()
	results, err := scanner.ScanDirectory("/nonexistent/path", "claude", "project")

	require.NoError(t, err) // Should not error, just return empty
	assert.Len(t, results, 0)
}

func TestScannerService_CategorizeSymlink_Skulto(t *testing.T) {
	tmpDir := t.TempDir()

	// Create skulto-managed symlink target
	targetDir := filepath.Join(tmpDir, ".skulto", "skills", "my-skill")
	require.NoError(t, os.MkdirAll(targetDir, 0755))

	// Create symlink
	symlinkPath := filepath.Join(tmpDir, "link")
	require.NoError(t, os.Symlink(targetDir, symlinkPath))

	scanner := NewScannerService()
	source := scanner.CategorizeSymlink(symlinkPath)

	assert.Equal(t, models.ManagementSkulto, source)
}

func TestScannerService_CategorizeSymlink_Vercel(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir, _ := os.UserHomeDir()

	// For this test, we'll use a path pattern that matches Vercel
	// In real use, symlinks would point to ~/.agents/skills/
	scanner := NewScannerService()

	// Test with mock symlink target containing .agents
	targetPath := filepath.Join(homeDir, ".agents", "skills", "some-skill")

	// We can't actually create this, so test the path-matching logic
	source := scanner.categorizeByTarget(targetPath)
	assert.Equal(t, models.ManagementVercel, source)

	_ = tmpDir // Avoid unused variable warning
}

func TestScannerService_CategorizeSymlink_External(t *testing.T) {
	tmpDir := t.TempDir()

	// Create external symlink target
	targetDir := filepath.Join(tmpDir, "some", "other", "path")
	require.NoError(t, os.MkdirAll(targetDir, 0755))

	// Create symlink
	symlinkPath := filepath.Join(tmpDir, "link")
	require.NoError(t, os.Symlink(targetDir, symlinkPath))

	scanner := NewScannerService()
	source := scanner.CategorizeSymlink(symlinkPath)

	assert.Equal(t, models.ManagementExternal, source)
}

func TestScannerService_CategorizeSymlink_NotSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a regular directory
	dirPath := filepath.Join(tmpDir, "regular-dir")
	require.NoError(t, os.MkdirAll(dirPath, 0755))

	scanner := NewScannerService()
	source := scanner.CategorizeSymlink(dirPath)

	assert.Equal(t, models.ManagementNone, source)
}

func TestScannerService_ScanPlatforms(t *testing.T) {
	tmpDir := t.TempDir()

	// Create skills in multiple platforms
	claudeSkillsDir := filepath.Join(tmpDir, ".claude", "skills")
	require.NoError(t, os.MkdirAll(filepath.Join(claudeSkillsDir, "claude-skill"), 0755))

	cursorSkillsDir := filepath.Join(tmpDir, ".cursor", "skills")
	require.NoError(t, os.MkdirAll(filepath.Join(cursorSkillsDir, "cursor-skill"), 0755))

	platforms := []PlatformConfig{
		{ID: "claude", SkillsPath: claudeSkillsDir},
		{ID: "cursor", SkillsPath: cursorSkillsDir},
	}

	scanner := NewScannerService()
	results, err := scanner.ScanPlatforms(platforms, "project")

	require.NoError(t, err)
	assert.Len(t, results, 2)

	// Check we found both skills
	names := make(map[string]bool)
	for _, r := range results {
		names[r.Name] = true
	}
	assert.True(t, names["claude-skill"])
	assert.True(t, names["cursor-skill"])
}
