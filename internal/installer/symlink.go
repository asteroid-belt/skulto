package installer

import (
	"fmt"
	"os"
	"path/filepath"
)

// SymlinkManager handles symlink creation and removal operations.
type SymlinkManager struct {
	resolver *PathResolver
}

// NewSymlinkManager creates a new symlink manager.
func NewSymlinkManager(resolver *PathResolver) *SymlinkManager {
	return &SymlinkManager{resolver: resolver}
}

// Create creates a symlink from target to source.
// If the target already exists and backupExisting is true, it will be backed up.
func (sm *SymlinkManager) Create(source, target string, backupExisting bool) error {
	// Ensure target directory exists
	targetDir := filepath.Dir(target)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
	}

	// Handle existing file at target
	if sm.Exists(target) {
		if sm.IsSymlink(target) {
			// Remove existing symlink
			if err := os.Remove(target); err != nil {
				return fmt.Errorf("failed to remove existing symlink: %w", err)
			}
		} else {
			// Backup existing regular file
			if backupExisting {
				backupPath := target + ".backup"
				if err := os.Rename(target, backupPath); err != nil {
					return fmt.Errorf("failed to backup existing file: %w", err)
				}
			} else {
				return fmt.Errorf("file exists at %s (enable backup_existing to overwrite)", target)
			}
		}
	}

	// Create symlink
	if err := os.Symlink(source, target); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// Remove removes a symlink.
func (sm *SymlinkManager) Remove(target string) error {
	if !sm.Exists(target) {
		return nil // Already gone
	}

	if !sm.IsSymlink(target) {
		return fmt.Errorf("target is not a symlink: %s", target)
	}

	if err := os.Remove(target); err != nil {
		return fmt.Errorf("failed to remove symlink: %w", err)
	}

	return nil
}

// Exists checks if a file or symlink exists at the given path.
func (sm *SymlinkManager) Exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// IsSymlink checks if a path is a symlink.
func (sm *SymlinkManager) IsSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

// ReadLink returns the target of a symlink.
func (sm *SymlinkManager) ReadLink(path string) (string, error) {
	if !sm.IsSymlink(path) {
		return "", fmt.Errorf("not a symlink: %s", path)
	}

	target, err := os.Readlink(path)
	if err != nil {
		return "", fmt.Errorf("failed to read symlink: %w", err)
	}

	return target, nil
}

// Verify checks that a symlink points to the expected target.
func (sm *SymlinkManager) Verify(symlink, expectedTarget string) (bool, error) {
	if !sm.IsSymlink(symlink) {
		return false, nil
	}

	target, err := sm.ReadLink(symlink)
	if err != nil {
		return false, err
	}

	return target == expectedTarget, nil
}

// CreateBackup creates a backup of a file.
func (sm *SymlinkManager) CreateBackup(path string) error {
	if !sm.Exists(path) || sm.IsSymlink(path) {
		return nil // No need to backup symlinks or non-existent files
	}

	backupPath := path + ".backup"

	// If backup already exists, don't overwrite it
	if sm.Exists(backupPath) {
		return nil
	}

	if err := os.Rename(path, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	return nil
}

// RestoreBackup restores a file from backup.
func (sm *SymlinkManager) RestoreBackup(path string) error {
	backupPath := path + ".backup"

	if !sm.Exists(backupPath) {
		return fmt.Errorf("backup file not found: %s", backupPath)
	}

	// Remove the current file if it exists
	if sm.Exists(path) {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove current file: %w", err)
		}
	}

	if err := os.Rename(backupPath, path); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	return nil
}

// CleanupBackups removes all backup files for a path.
func (sm *SymlinkManager) CleanupBackups(path string) error {
	backupPath := path + ".backup"
	if sm.Exists(backupPath) {
		if err := os.Remove(backupPath); err != nil {
			return fmt.Errorf("failed to remove backup: %w", err)
		}
	}
	return nil
}
