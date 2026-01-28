package detect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandHomePath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	t.Run("expands tilde prefix", func(t *testing.T) {
		result := expandHomePath("~/some/path")
		assert.Equal(t, filepath.Join(home, "some/path"), result)
	})

	t.Run("returns absolute path unchanged", func(t *testing.T) {
		result := expandHomePath("/usr/local/bin")
		assert.Equal(t, "/usr/local/bin", result)
	})

	t.Run("returns relative path unchanged", func(t *testing.T) {
		result := expandHomePath("relative/path")
		assert.Equal(t, "relative/path", result)
	})

	t.Run("handles bare tilde", func(t *testing.T) {
		result := expandHomePath("~")
		assert.Equal(t, "~", result) // Only ~/... is expanded
	})
}

func TestDetectPlatform_NonExistent(t *testing.T) {
	// A platform with a command that doesn't exist and directories that don't exist
	// should return Detected=false
	result := DetectPlatform(installer.PlatformNeovate) // Unlikely to be installed
	// We can't guarantee it's not detected (CI might have weird things),
	// but at minimum the result should have the right platform
	assert.Equal(t, installer.PlatformNeovate, result.Platform)
}

func TestDetectPlatform_WithProjectDir(t *testing.T) {
	// Create a temporary directory to simulate a project directory
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	// Create a .pochi directory (unlikely to clash with real installs)
	require.NoError(t, os.Mkdir(".pochi", 0o755))

	result := DetectPlatform(installer.PlatformPochi)
	assert.True(t, result.Detected, "Should detect platform when project directory exists")
	assert.Equal(t, installer.PlatformPochi, result.Platform)
	assert.Equal(t, ".pochi", result.ProjectDir)
}

func TestDetectPlatform_WithGlobalDir(t *testing.T) {
	// Create a temporary home directory with a global path
	tmpDir := t.TempDir()
	globalPath := filepath.Join(tmpDir, ".neovate", "skills")
	require.NoError(t, os.MkdirAll(globalPath, 0o755))

	// We need to test expandHomePath with a known path.
	// Instead of mocking home, we test detectWithPaths directly.
	result := detectWithPaths("neovate", "", ".neovate", globalPath, nil)
	assert.True(t, result.Detected, "Should detect platform when global directory exists")
	assert.Equal(t, globalPath, result.GlobalDir)
}

func TestDetectPlatform_WithCommand(t *testing.T) {
	// "go" should exist in PATH on any Go test environment
	result := detectWithPaths("go", "", ".nonexistent-dir-12345", "/nonexistent-path-12345", nil)
	assert.True(t, result.Detected, "Should detect platform when command exists in PATH")
	assert.NotEmpty(t, result.CommandPath)
}

func TestDetectPlatform_NoCommand(t *testing.T) {
	// Empty command should skip command check without erroring
	result := detectWithPaths("", "", "/nonexistent-dir-12345", "/nonexistent-path-12345", nil)
	assert.False(t, result.Detected)
	assert.Empty(t, result.CommandPath)
}

func TestDetectPlatform_PlatformSpecificPaths(t *testing.T) {
	tmpDir := t.TempDir()
	specificPath := filepath.Join(tmpDir, "SpecialApp.app")
	require.NoError(t, os.Mkdir(specificPath, 0o755))

	result := detectWithPaths("nonexistent-cmd-12345", "", "/nonexistent-dir-12345", "/nonexistent-path-12345", []string{specificPath})
	assert.True(t, result.Detected, "Should detect platform when platform-specific path exists")
}

func TestDetectAll_ReturnsAllPlatforms(t *testing.T) {
	results := DetectAll()
	allPlatforms := installer.AllPlatforms()
	assert.Equal(t, len(allPlatforms), len(results),
		"DetectAll should return results for all registered platforms")

	// Verify each result has a valid platform
	platformSet := make(map[installer.Platform]bool)
	for _, r := range results {
		assert.True(t, r.Platform.IsValid(), "Platform %s should be valid", r.Platform)
		platformSet[r.Platform] = true
	}
	// Every platform from AllPlatforms should be in the results
	for _, p := range allPlatforms {
		assert.True(t, platformSet[p], "Platform %s should be in DetectAll results", p)
	}
}

func TestDetectAll_NoDuplicates(t *testing.T) {
	results := DetectAll()
	seen := make(map[installer.Platform]bool)
	for _, r := range results {
		assert.False(t, seen[r.Platform], "Platform %s appears more than once in DetectAll", r.Platform)
		seen[r.Platform] = true
	}
}
