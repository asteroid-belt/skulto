package installer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSymlinkVerify tests verifying symlink targets.
func TestSymlinkVerify(t *testing.T) {
	cfg := setupTestConfig(t)
	resolver := NewPathResolver(cfg)
	sm := NewSymlinkManager(resolver)

	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.txt")
	targetLink := filepath.Join(tempDir, "target.txt")

	// Create source file
	err := os.WriteFile(sourceFile, []byte("content"), 0644)
	require.NoError(t, err)

	// Create symlink
	err = sm.Create(sourceFile, targetLink, false)
	require.NoError(t, err)

	// Verify correct target
	valid, err := sm.Verify(targetLink, sourceFile)
	require.NoError(t, err)
	assert.True(t, valid)

	// Verify wrong target
	valid, err = sm.Verify(targetLink, "/wrong/path")
	require.NoError(t, err)
	assert.False(t, valid)
}

// TestSymlinkReadLink tests reading symlink targets.
func TestSymlinkReadLink(t *testing.T) {
	cfg := setupTestConfig(t)
	resolver := NewPathResolver(cfg)
	sm := NewSymlinkManager(resolver)

	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.txt")
	targetLink := filepath.Join(tempDir, "target.txt")

	// Create source file
	err := os.WriteFile(sourceFile, []byte("content"), 0644)
	require.NoError(t, err)

	// Create symlink
	err = sm.Create(sourceFile, targetLink, false)
	require.NoError(t, err)

	// Read link
	target, err := sm.ReadLink(targetLink)
	require.NoError(t, err)
	assert.Equal(t, sourceFile, target)
}

// TestSymlinkCreateWithBackup tests creating symlink with backup of existing file.
func TestSymlinkCreateWithBackup(t *testing.T) {
	cfg := setupTestConfig(t)
	resolver := NewPathResolver(cfg)
	sm := NewSymlinkManager(resolver)

	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.txt")
	targetLink := filepath.Join(tempDir, "target.txt")

	// Create source and existing files
	err := os.WriteFile(sourceFile, []byte("source content"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(targetLink, []byte("existing content"), 0644)
	require.NoError(t, err)

	// Create symlink with backup
	err = sm.Create(sourceFile, targetLink, true)
	require.NoError(t, err)

	// Verify symlink created
	assert.True(t, sm.IsSymlink(targetLink))

	// Verify backup exists
	backupPath := targetLink + ".backup"
	assert.FileExists(t, backupPath)

	// Verify backup content
	backup, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, "existing content", string(backup))
}

// TestSymlinkCreateWithoutBackup tests creating symlink fails if file exists and backup disabled.
func TestSymlinkCreateWithoutBackup(t *testing.T) {
	cfg := setupTestConfig(t)
	resolver := NewPathResolver(cfg)
	sm := NewSymlinkManager(resolver)

	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.txt")
	targetLink := filepath.Join(tempDir, "target.txt")

	// Create source and existing files
	err := os.WriteFile(sourceFile, []byte("source content"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(targetLink, []byte("existing content"), 0644)
	require.NoError(t, err)

	// Create symlink without backup should fail
	err = sm.Create(sourceFile, targetLink, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exists")

	// Verify no symlink created
	assert.False(t, sm.IsSymlink(targetLink))
}

// TestSymlinkRemoveNonSymlink tests removing a non-symlink fails.
func TestSymlinkRemoveNonSymlink(t *testing.T) {
	cfg := setupTestConfig(t)
	resolver := NewPathResolver(cfg)
	sm := NewSymlinkManager(resolver)

	tempDir := t.TempDir()
	regularFile := filepath.Join(tempDir, "regular.txt")
	err := os.WriteFile(regularFile, []byte("content"), 0644)
	require.NoError(t, err)

	// Remove non-symlink should fail
	err = sm.Remove(regularFile)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a symlink")

	// File should still exist
	assert.FileExists(t, regularFile)
}

// TestSymlinkCreateBackup tests creating a backup of a file.
func TestSymlinkCreateBackup(t *testing.T) {
	cfg := setupTestConfig(t)
	resolver := NewPathResolver(cfg)
	sm := NewSymlinkManager(resolver)

	tempDir := t.TempDir()
	regularFile := filepath.Join(tempDir, "regular.txt")
	err := os.WriteFile(regularFile, []byte("original content"), 0644)
	require.NoError(t, err)

	// Create backup (moves file to backup location)
	err = sm.CreateBackup(regularFile)
	require.NoError(t, err)

	// Verify backup exists
	backupPath := regularFile + ".backup"
	assert.FileExists(t, backupPath)

	// Verify content
	backup, err := os.ReadFile(backupPath)
	require.NoError(t, err)
	assert.Equal(t, "original content", string(backup))

	// Original should no longer exist (it was moved to backup)
	assert.NoFileExists(t, regularFile)
}

// TestSymlinkRestoreBackup tests restoring a file from backup.
func TestSymlinkRestoreBackup(t *testing.T) {
	cfg := setupTestConfig(t)
	resolver := NewPathResolver(cfg)
	sm := NewSymlinkManager(resolver)

	tempDir := t.TempDir()
	originalFile := filepath.Join(tempDir, "original.txt")
	backupPath := originalFile + ".backup"

	// Create backup file
	err := os.WriteFile(backupPath, []byte("backup content"), 0644)
	require.NoError(t, err)

	// Create a current file (simulating it being replaced)
	err = os.WriteFile(originalFile, []byte("new content"), 0644)
	require.NoError(t, err)

	// Restore from backup
	err = sm.RestoreBackup(originalFile)
	require.NoError(t, err)

	// Verify backup is gone
	assert.NoFileExists(t, backupPath)

	// Verify content is restored
	content, err := os.ReadFile(originalFile)
	require.NoError(t, err)
	assert.Equal(t, "backup content", string(content))
}

// TestSymlinkCleanupBackups tests removing backup files.
func TestSymlinkCleanupBackups(t *testing.T) {
	cfg := setupTestConfig(t)
	resolver := NewPathResolver(cfg)
	sm := NewSymlinkManager(resolver)

	tempDir := t.TempDir()
	originalFile := filepath.Join(tempDir, "original.txt")
	backupPath := originalFile + ".backup"

	// Create backup file
	err := os.WriteFile(backupPath, []byte("backup content"), 0644)
	require.NoError(t, err)

	// Cleanup backups
	err = sm.CleanupBackups(originalFile)
	require.NoError(t, err)

	// Verify backup is gone
	assert.NoFileExists(t, backupPath)
}

// TestSymlinkExists tests checking if a path exists.
func TestSymlinkExists(t *testing.T) {
	cfg := setupTestConfig(t)
	resolver := NewPathResolver(cfg)
	sm := NewSymlinkManager(resolver)

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.txt")

	// Should not exist initially
	assert.False(t, sm.Exists(filePath))

	// Create file
	err := os.WriteFile(filePath, []byte("content"), 0644)
	require.NoError(t, err)

	// Should exist now
	assert.True(t, sm.Exists(filePath))
}

// TestSymlinkReplaceExistingSymlink tests replacing an existing symlink.
func TestSymlinkReplaceExistingSymlink(t *testing.T) {
	cfg := setupTestConfig(t)
	resolver := NewPathResolver(cfg)
	sm := NewSymlinkManager(resolver)

	tempDir := t.TempDir()
	source1 := filepath.Join(tempDir, "source1.txt")
	source2 := filepath.Join(tempDir, "source2.txt")
	target := filepath.Join(tempDir, "target.txt")

	// Create source files
	err := os.WriteFile(source1, []byte("source1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(source2, []byte("source2"), 0644)
	require.NoError(t, err)

	// Create symlink to source1
	err = sm.Create(source1, target, false)
	require.NoError(t, err)

	// Verify points to source1
	linkedTo, err := sm.ReadLink(target)
	require.NoError(t, err)
	assert.Equal(t, source1, linkedTo)

	// Replace with symlink to source2
	err = sm.Create(source2, target, false)
	require.NoError(t, err)

	// Verify points to source2 now
	linkedTo, err = sm.ReadLink(target)
	require.NoError(t, err)
	assert.Equal(t, source2, linkedTo)
}
