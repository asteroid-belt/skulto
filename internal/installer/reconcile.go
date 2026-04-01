package installer

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/asteroid-belt/skulto/internal/log"
	"github.com/asteroid-belt/skulto/internal/models"
)

// ReconcileResult holds the output of project skill reconciliation.
type ReconcileResult struct {
	Reconciled []ReconciledEntry // skills added to skill_installations
	Unmanaged  []UnmanagedEntry  // entries not skulto-managed
}

// ReconciledEntry records a reconciled skill with its platform.
type ReconciledEntry struct {
	Slug     string
	Platform Platform
}

// UnmanagedEntry records an unmanaged skill with its location.
type UnmanagedEntry struct {
	Name     string
	Platform Platform
	Path     string // full path to the entry
}

// ReconcileProjectSkills scans project platform dirs for skills on disk
// that lack skill_installations DB records. For skulto-managed symlinks,
// it creates the missing records. Returns unmanaged entries for the caller.
func (i *Installer) ReconcileProjectSkills(cwd string) (*ReconcileResult, error) {
	result := &ReconcileResult{}

	for _, platform := range AllPlatforms() {
		info := platform.Info()
		if info.SkillsPath == "" {
			continue
		}

		skillsDir := filepath.Join(cwd, info.SkillsPath)
		entries, err := os.ReadDir(skillsDir)
		if err != nil {
			continue // dir doesn't exist, skip
		}

		for _, entry := range entries {
			entryPath := filepath.Join(skillsDir, entry.Name())

			if !isSymlink(entryPath) {
				// Plain directory — skip silently (committed to repo, not skulto-managed)
				continue
			}

			// Resolve symlink target (handles relative symlinks)
			resolvedTarget, err := filepath.EvalSymlinks(entryPath)
			if err != nil {
				log.DebugLog("reconcile", "broken symlink %s: %v", entryPath, err)
				continue // broken symlink, skip
			}

			// Classify: is this a skulto-managed symlink?
			skill, isSkultoManaged := i.resolveSkultoSkill(entry.Name(), resolvedTarget)
			if !isSkultoManaged {
				// Not skulto-managed — unmanaged
				result.Unmanaged = append(result.Unmanaged, UnmanagedEntry{
					Name: entry.Name(), Platform: platform, Path: entryPath,
				})
				continue
			}
			if skill == nil {
				// Skulto-managed but no matching skill in DB — skip silently
				continue
			}

			// Check if already tracked in DB
			installed, err := i.db.IsInstalledAt(skill.ID, string(platform), "project", cwd)
			if err != nil {
				log.DebugLog("reconcile", "DB check failed for %s: %v", entry.Name(), err)
				continue
			}
			if installed {
				continue // already tracked
			}

			// Create missing installation record
			install := models.SkillInstallation{
				SkillID:     skill.ID,
				Platform:    string(platform),
				Scope:       string(ScopeProject),
				BasePath:    cwd,
				SymlinkPath: entryPath,
			}
			if err := i.db.AddInstallation(&install); err != nil {
				log.DebugLog("reconcile", "failed to add installation for %s: %v", entry.Name(), err)
				continue
			}

			result.Reconciled = append(result.Reconciled, ReconciledEntry{
				Slug: entry.Name(), Platform: platform,
			})
		}
	}

	return result, nil
}

// resolveSkultoSkill attempts to match a symlink target to a known skill in the DB.
// Returns (skill, true) if matched, (nil, true) if skulto-managed but no DB match,
// or (nil, false) if not skulto-managed at all.
func (i *Installer) resolveSkultoSkill(slug, resolvedTarget string) (*models.Skill, bool) {
	// Check for repo-backed skill: target contains /repositories/{owner}/{repo}/
	if idx := strings.Index(resolvedTarget, "/repositories/"); idx != -1 {
		afterRepos := resolvedTarget[idx+len("/repositories/"):]
		parts := strings.SplitN(afterRepos, "/", 3)
		if len(parts) >= 2 {
			sourceFullName := parts[0] + "/" + parts[1]
			if skill, err := i.db.GetSkillBySlugAndSource(slug, sourceFullName); err == nil && skill != nil {
				return skill, true
			}
		}
		return nil, true // skulto-managed but no DB match
	}

	// Check for local skill: target contains /.agents/skulto/skills/ or /.skulto/skills/
	if strings.Contains(resolvedTarget, "/.agents/skulto/skills/") ||
		strings.Contains(resolvedTarget, "/.skulto/skills/") {
		if skill := i.findSkillBySlug(slug); skill != nil && skill.IsLocal {
			return skill, true
		}
		return nil, true // skulto-managed but no DB match
	}

	return nil, false // not skulto-managed
}
