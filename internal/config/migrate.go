package config

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/asteroid-belt/skulto/internal/log"
	_ "github.com/glebarez/go-sqlite"
)

const migrationMarker = ".migration-complete"

// migrateBaseDir moves ~/.skulto to ~/.agents/skulto if needed.
// This runs once on startup before ensureDirectories().
// Uses a .migration-complete marker file to track state safely.
func migrateBaseDir(homeDir string) error {
	oldDir := filepath.Join(homeDir, ".skulto")
	newDir := filepath.Join(homeDir, ".agents", "skulto")
	markerFile := filepath.Join(newDir, migrationMarker)

	// 1. Does ~/.skulto exist (real dir, not symlink)?
	info, err := os.Lstat(oldDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Fresh install
		}
		return fmt.Errorf("stat old dir: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil // Symlink or not a dir — skip
	}

	// 2. Was migration already completed in a prior run?
	if _, err := os.Stat(markerFile); err == nil {
		// Marker exists — prior migration succeeded.
		// Do NOT delete ~/.skulto here. It may have been intentionally restored
		// or contain new data. The old dir is harmless — it won't be used.
		return nil
	}

	// 3. Move data to new location
	if err := os.MkdirAll(filepath.Join(homeDir, ".agents"), 0755); err != nil {
		return fmt.Errorf("create .agents dir: %w", err)
	}

	newIsEmpty := true
	if contents, _ := dirHasContents(newDir); contents {
		newIsEmpty = false
	}

	if newIsEmpty {
		// Remove empty dir so os.Rename works
		_ = os.RemoveAll(newDir)
		if err := os.Rename(oldDir, newDir); err != nil {
			// Cross-device fallback: copy + verify
			if err := os.MkdirAll(newDir, 0755); err != nil {
				return fmt.Errorf("create new dir: %w", err)
			}
			if err := copyDirContents(oldDir, newDir); err != nil {
				return fmt.Errorf("copy migration: %w", err)
			}
			if err := verifyCriticalFiles(oldDir, newDir); err != nil {
				return fmt.Errorf("verify migration: %w", err)
			}
		}
	} else {
		// Partial migration from prior attempt — copy missing contents
		if err := copyDirContents(oldDir, newDir); err != nil {
			return fmt.Errorf("complete partial migration: %w", err)
		}
	}

	// 4. Post-move fixups (idempotent, safe to re-run)
	dbPath := filepath.Join(newDir, "skulto.db")
	if _, err := os.Stat(dbPath); err == nil {
		if err := migrateDBPaths(dbPath); err != nil {
			return fmt.Errorf("migrate DB paths: %w", err)
		}
	}

	if err := rewriteSymlinks(dbPath, oldDir, newDir); err != nil {
		return fmt.Errorf("rewrite symlinks: %w", err)
	}

	// 5. Write marker — signals migration completed successfully
	if err := os.WriteFile(markerFile, []byte("migrated"), 0644); err != nil {
		return fmt.Errorf("write migration marker: %w", err)
	}

	// 6. Remove old dir (safe — marker confirms success)
	if err := os.RemoveAll(oldDir); err != nil {
		log.Errorf("migrate: failed to remove old dir %s: %v\n", oldDir, err)
	}

	log.Printf("Migrated %s → %s\n", oldDir, newDir)
	return nil
}

// migrateDBPaths updates absolute paths in the DB from /.skulto/ to /.agents/skulto/.
// Returns nil if the DB can't be opened (e.g., corrupted) — non-fatal for migration.
func migrateDBPaths(dbPath string) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Errorf("migrate: could not open DB for path migration: %v\n", err)
		return nil
	}
	defer func() { _ = db.Close() }()

	// Verify it's actually a SQLite database before running queries
	if err := db.Ping(); err != nil {
		log.Errorf("migrate: DB ping failed, skipping path migration: %v\n", err)
		return nil
	}

	_, err = db.Exec(
		`UPDATE skills SET file_path = REPLACE(file_path, '/.skulto/', '/.agents/skulto/') WHERE file_path LIKE '%/.skulto/%'`,
	)
	if err != nil {
		log.Errorf("migrate: failed to update skills paths: %v\n", err)
		return nil
	}
	return nil
}

// rewriteSymlinks finds all symlinks on disk whose targets point into oldDir
// and rewrites them to point into newDir. Uses the skill_installations DB table
// to discover symlink locations across all platforms and scopes.
func rewriteSymlinks(dbPath, oldDir, newDir string) error {
	symlinkPaths, err := getInstalledSymlinkPaths(dbPath)
	if err != nil {
		log.Errorf("migrate: could not read skill_installations: %v\n", err)
		return nil // Non-fatal — symlinks will be stale but not catastrophic
	}

	rewritten := 0
	for _, linkPath := range symlinkPaths {
		if rewriteOneSymlink(linkPath, oldDir, newDir) {
			rewritten++
		}
	}

	if rewritten > 0 {
		log.Printf("Rewrote %d symlinks\n", rewritten)
	}
	return nil
}

// getInstalledSymlinkPaths reads all symlink_path values from skill_installations.
func getInstalledSymlinkPaths(dbPath string) ([]string, error) {
	if _, err := os.Stat(dbPath); err != nil {
		return nil, nil // No DB yet
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()

	rows, err := db.Query("SELECT symlink_path FROM skill_installations WHERE symlink_path IS NOT NULL AND symlink_path != ''")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			continue
		}
		paths = append(paths, p)
	}
	return paths, rows.Err()
}

// rewriteOneSymlink rewrites a single symlink if its target points into oldDir.
func rewriteOneSymlink(linkPath, oldDir, newDir string) bool {
	target, err := os.Readlink(linkPath)
	if err != nil {
		return false
	}
	if !strings.Contains(target, oldDir) {
		return false
	}
	newTarget := strings.Replace(target, oldDir, newDir, 1)
	if _, err := os.Stat(newTarget); err != nil {
		return false // New target doesn't exist
	}
	if err := os.Remove(linkPath); err != nil {
		log.Errorf("migrate: failed to remove symlink %s: %v\n", linkPath, err)
		return false
	}
	if err := os.Symlink(newTarget, linkPath); err != nil {
		log.Errorf("migrate: failed to create symlink %s → %s: %v\n", linkPath, newTarget, err)
		return false
	}
	return true
}

// verifyCriticalFiles checks that user-data files were copied successfully.
func verifyCriticalFiles(src, dst string) error {
	if _, err := os.Stat(filepath.Join(src, "skulto.db")); err == nil {
		if _, err := os.Stat(filepath.Join(dst, "skulto.db")); err != nil {
			return fmt.Errorf("skulto.db missing from destination")
		}
	}
	if _, err := os.Stat(filepath.Join(src, "favorites.json")); err == nil {
		if _, err := os.Stat(filepath.Join(dst, "favorites.json")); err != nil {
			return fmt.Errorf("favorites.json missing from destination")
		}
	}
	srcSkills := filepath.Join(src, "skills")
	dstSkills := filepath.Join(dst, "skills")
	if srcEntries, err := os.ReadDir(srcSkills); err == nil {
		dstEntries, err := os.ReadDir(dstSkills)
		if err != nil {
			return fmt.Errorf("skills/ dir missing from destination")
		}
		if len(srcEntries) != len(dstEntries) {
			return fmt.Errorf("skills/ entry count mismatch: src=%d dst=%d", len(srcEntries), len(dstEntries))
		}
	}
	return nil
}

// dirHasContents returns true if a directory has any entries.
func dirHasContents(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}

// copyDirContents recursively copies all contents from src to dst.
// Skips files that already exist at the destination (safe for partial recovery).
func copyDirContents(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Skip files that already exist
		if _, err := os.Stat(dstPath); err == nil {
			return nil
		}

		// Handle symlinks within the directory tree
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(target, dstPath)
		}

		return copyFile(path, dstPath, info.Mode())
	})
}

// copyFile copies a single file.
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, in)
	return err
}
