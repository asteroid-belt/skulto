package installer

import (
	"context"
	"os"
	"path/filepath"

	"github.com/asteroid-belt/skulto/internal/log"
)

// PathMigrationResult summarizes deprecated-path migration outcomes.
type PathMigrationResult struct {
	Migrated                int
	SkippedConflicts        int
	SkippedBrokenSymlinks   int
	SkippedPlainDirectories int
}

// migrateDeprecatedPathsForLocation migrates deprecated skill directories for a
// specific platform/scope/basePath to the canonical policy path.
//
// Safety rules:
// - Only symlinks are migrated
// - Broken symlinks are skipped
// - Plain directories are skipped
// - Conflicting destinations are skipped
func (i *Installer) migrateDeprecatedPathsForLocation(platform Platform, scope InstallScope, basePath string) PathMigrationResult {
	result := PathMigrationResult{}

	canonicalBase := resolveCanonicalSkillsBasePath(platform, scope, basePath)
	if canonicalBase == "" {
		return result
	}

	deprecatedBases := resolveDeprecatedSkillsBasePaths(platform, scope, basePath)
	for _, deprecatedBase := range deprecatedBases {
		entries, err := os.ReadDir(deprecatedBase)
		if err != nil {
			continue // deprecated base missing/unreadable: nothing to migrate
		}

		for _, entry := range entries {
			sourcePath := filepath.Join(deprecatedBase, entry.Name())
			destPath := filepath.Join(canonicalBase, entry.Name())

			info, err := os.Lstat(sourcePath)
			if err != nil {
				continue
			}
			if info.Mode()&os.ModeSymlink == 0 {
				result.SkippedPlainDirectories++
				continue
			}

			resolvedTarget, err := filepath.EvalSymlinks(sourcePath)
			if err != nil {
				result.SkippedBrokenSymlinks++
				continue
			}

			if exists(destPath) {
				if i.sameSymlinkTarget(destPath, resolvedTarget) {
					if err := os.Remove(sourcePath); err != nil {
						result.SkippedConflicts++
						continue
					}
					i.updateInstallationSymlinkPath(string(platform), string(scope), basePath, sourcePath, destPath)
					result.Migrated++
					continue
				}
				result.SkippedConflicts++
				continue
			}

			if err := os.MkdirAll(canonicalBase, 0o755); err != nil {
				result.SkippedConflicts++
				continue
			}
			if err := os.Symlink(resolvedTarget, destPath); err != nil {
				result.SkippedConflicts++
				continue
			}
			if err := os.Remove(sourcePath); err != nil {
				_ = os.Remove(destPath)
				result.SkippedConflicts++
				continue
			}

			i.updateInstallationSymlinkPath(string(platform), string(scope), basePath, sourcePath, destPath)
			result.Migrated++
		}
	}

	return result
}

// EnsurePathPolicy enforces registered path policies for global and project scopes.
func (i *Installer) EnsurePathPolicy(ctx context.Context, cwd string) (PathMigrationResult, error) {
	_ = ctx // reserved for future cancellation support

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return PathMigrationResult{}, err
	}
	if cwd == "" {
		cwd, err = os.Getwd()
		if err != nil {
			return PathMigrationResult{}, err
		}
	}

	return i.ensurePathPolicyForBasePaths(map[InstallScope]string{
		ScopeGlobal:  homeDir,
		ScopeProject: cwd,
	}), nil
}

func (i *Installer) ensurePathPolicyForBasePaths(basePathByScope map[InstallScope]string) PathMigrationResult {
	var summary PathMigrationResult

	for key := range pathPolicies {
		basePath := basePathByScope[key.scope]
		if basePath == "" {
			continue
		}
		result := i.migrateDeprecatedPathsForLocation(key.platform, key.scope, basePath)
		summary.Migrated += result.Migrated
		summary.SkippedConflicts += result.SkippedConflicts
		summary.SkippedBrokenSymlinks += result.SkippedBrokenSymlinks
		summary.SkippedPlainDirectories += result.SkippedPlainDirectories
	}

	return summary
}

func (i *Installer) sameSymlinkTarget(path string, expectedTarget string) bool {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	return resolved == expectedTarget
}

func (i *Installer) updateInstallationSymlinkPath(platform, scope, basePath, oldPath, newPath string) {
	if i == nil || i.db == nil {
		return
	}
	if err := i.db.UpdateInstallationSymlinkPath(platform, scope, basePath, oldPath, newPath); err != nil {
		log.DebugLog("path_migrator", "failed to update installation symlink path from %s to %s: %v", oldPath, newPath, err)
	}
}
