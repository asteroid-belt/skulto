package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
)

// MigrationResult tracks what was done during migration.
type MigrationResult struct {
	SkillsUpdated     int
	CategoriesCleared int
	SkillsMoved       int
	SkillsSkipped     int
	DirsRemoved       int
	Errors            []string
}

// FlattenSkills migrates nested skills to flat structure.
// This operation is idempotent - safe to run multiple times.
func FlattenSkills(database *db.DB, skillsDir string) (*MigrationResult, error) {
	result := &MigrationResult{}

	// 1. Set all existing skills to PENDING (only if not already set)
	res := database.Model(&models.Skill{}).
		Where("security_status = '' OR security_status IS NULL").
		Update("security_status", models.SecurityStatusPending)
	if res.Error != nil {
		return result, fmt.Errorf("set pending status: %w", res.Error)
	}
	result.SkillsUpdated = int(res.RowsAffected)

	// 2. Clear category field on all skills
	res = database.Model(&models.Skill{}).
		Where("category != ''").
		Update("category", "")
	if res.Error != nil {
		return result, fmt.Errorf("clear categories: %w", res.Error)
	}
	result.CategoriesCleared = int(res.RowsAffected)

	// 3. Flatten filesystem structure
	if err := flattenFilesystem(skillsDir, result); err != nil {
		return result, fmt.Errorf("flatten filesystem: %w", err)
	}

	return result, nil
}

// flattenFilesystem moves nested skills to top level.
// Handles edge cases: already flat, lowercase filenames, spaces in paths,
// dual nesting (superplan case), and name collisions.
func flattenFilesystem(skillsDir string, result *MigrationResult) error {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No skills dir yet
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirPath := filepath.Join(skillsDir, entry.Name())

		// Check if this directory is already a flat skill
		// Support both SKILL.md (uppercase) and skill.md (lowercase)
		if isSkillDir(dirPath) {
			result.SkillsSkipped++

			// Even if it's flat, check for nested skills inside (dual nesting case)
			// e.g., superplan/SKILL.md + superplan/superplan/SKILL.md
			if err := processCategory(skillsDir, dirPath, entry.Name(), result); err != nil {
				result.Errors = append(result.Errors, err.Error())
			}
			continue
		}

		// This might be a category directory - check for nested skills
		if err := processCategory(skillsDir, dirPath, entry.Name(), result); err != nil {
			result.Errors = append(result.Errors, err.Error())
			// Continue processing other categories
		}

		// Remove empty category directory (handles spaces in names)
		if isEmpty(dirPath) {
			if err := os.Remove(dirPath); err == nil {
				result.DirsRemoved++
			}
		}
	}

	return nil
}

// isSkillDir checks if a directory contains a skill file (case-insensitive).
func isSkillDir(dirPath string) bool {
	// Check uppercase first (preferred)
	if _, err := os.Stat(filepath.Join(dirPath, "SKILL.md")); err == nil {
		return true
	}
	// Check lowercase
	if _, err := os.Stat(filepath.Join(dirPath, "skill.md")); err == nil {
		return true
	}
	return false
}

// processCategory handles a potential category directory.
func processCategory(skillsDir, categoryDir, categoryName string, result *MigrationResult) error {
	subEntries, err := os.ReadDir(categoryDir)
	if err != nil {
		return fmt.Errorf("read category %s: %w", categoryName, err)
	}

	for _, subEntry := range subEntries {
		if !subEntry.IsDir() {
			continue
		}

		nestedSkillDir := filepath.Join(categoryDir, subEntry.Name())

		// Check if this subdirectory is a skill
		if !isSkillDir(nestedSkillDir) {
			continue
		}

		// Determine target flat location
		flatSkillDir := filepath.Join(skillsDir, subEntry.Name())

		// Handle name collision (e.g., superplan/superplan case)
		if _, err := os.Stat(flatSkillDir); err == nil {
			// Target exists - check if it's the same as source (dual nesting)
			if filepath.Clean(flatSkillDir) == filepath.Clean(nestedSkillDir) {
				result.SkillsSkipped++
				continue
			}

			// True collision - append sanitized category as suffix
			sanitizedCategory := sanitizeForPath(categoryName)
			flatSkillDir = filepath.Join(skillsDir,
				fmt.Sprintf("%s-%s", subEntry.Name(), sanitizedCategory))

			// If still collides, add numeric suffix
			for i := 2; ; i++ {
				if _, err := os.Stat(flatSkillDir); os.IsNotExist(err) {
					break
				}
				flatSkillDir = filepath.Join(skillsDir,
					fmt.Sprintf("%s-%s-%d", subEntry.Name(), sanitizedCategory, i))
			}

		}

		if err := os.Rename(nestedSkillDir, flatSkillDir); err != nil {
			return fmt.Errorf("move %s: %w", nestedSkillDir, err)
		}
		result.SkillsMoved++
	}

	return nil
}

// sanitizeForPath removes/replaces characters that are problematic in paths.
func sanitizeForPath(s string) string {
	if s == "" {
		return ""
	}

	// Replace spaces with hyphens
	s = strings.ReplaceAll(s, " ", "-")
	// Remove other problematic characters
	s = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' ||
			r == '"' || r == '<' || r == '>' || r == '|' {
			return '-'
		}
		return r
	}, s)
	// Collapse multiple hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return strings.Trim(s, "-")
}

// isEmpty checks if a directory is empty.
func isEmpty(dir string) bool {
	entries, err := os.ReadDir(dir)
	return err == nil && len(entries) == 0
}
