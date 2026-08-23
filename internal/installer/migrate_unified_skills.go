package installer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/log"
)

// unifiedSkillsMigrationMarker is the filename (inside BaseDir/.migrations) that
// records completion of the one-shot D2+D3-normalize+D5 migration pass.
const unifiedSkillsMigrationMarker = "unified-local-skills.done"

// UnifiedSkillsMigrationReport summarizes what a migration pass did.
type UnifiedSkillsMigrationReport struct {
	CwdPurged           int // D2: number of cwd-* skill rows deleted
	NormalizedFilePaths int // D3: number of local-* rows whose file_path was rewritten from .../skill.md to the containing dir
	Migrated            int // D5: number of project-scoped local-* skills copied into BaseDir/skills (no collision)
	Collisions          int // D5: number of project-scoped local-* skills where BaseDir/skills/<slug> already existed
}

// MigrateToUnifiedLocalSkills runs the one-shot D2+D3-normalize+D5 migration.
// It is safe to call on every startup; a marker file at
// <cfg.BaseDir>/.migrations/unified-local-skills.done ensures the body runs at
// most once per install. Returns (nil, nil) when the marker already exists.
//
// Per-row errors (invalid file_path, missing source dir, unsafe-looking
// newDir, per-skill symlink rewrite failure) are logged and skipped — they
// do NOT prevent the marker from being written. This is an explicit design
// trade-off from spec D5: skipped rows become permanently stranded in
// their pre-migration state after the marker lands. The alternative
// (retry-forever on every launch) would risk chaining damage against
// filesystems that are already partially inconsistent, and users whose
// rows skip here can always recover by running `skulto ingest <slug>` to
// carry the skill over manually.
//
// Only pass-level failures (missing config, can't create the marker dir,
// DB read failure, or writing the marker file itself) abort the pass; in
// those cases the marker is not written and the next launch retries.
func MigrateToUnifiedLocalSkills(database *db.DB, cfg *config.Config) (*UnifiedSkillsMigrationReport, error) {
	if cfg == nil || cfg.BaseDir == "" {
		return nil, fmt.Errorf("migrate unified skills: invalid config (nil or empty BaseDir)")
	}
	if database == nil {
		return nil, fmt.Errorf("migrate unified skills: database is nil")
	}

	markerDir := filepath.Join(cfg.BaseDir, ".migrations")
	markerFile := filepath.Join(markerDir, unifiedSkillsMigrationMarker)

	if _, err := os.Stat(markerFile); err == nil {
		// Marker present → already migrated. Return nil report per spec.
		return nil, nil
	}

	// Acquire a process-level lock before mutating. Prevents a second skulto
	// invocation (second CLI run, MCP server starting in parallel) from racing
	// this migration. We use syscall.Flock with LOCK_EX|LOCK_NB — the kernel
	// releases the lock automatically when the process exits (crash, SIGKILL,
	// or clean exit), so a stale lock file on disk never blocks future runs.
	// If another process holds the lock, we skip this pass; the marker gate
	// keeps us consistent on the next clean launch.
	if err := os.MkdirAll(markerDir, 0755); err != nil {
		return nil, fmt.Errorf("create migrations dir: %w", err)
	}
	lockFile := filepath.Join(markerDir, unifiedSkillsMigrationMarker+".lock")
	lockFH, err := os.OpenFile(lockFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open migration lock file: %w", err)
	}
	if err := syscall.Flock(int(lockFH.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		// Another process holds the lock. Skip this pass; the marker gate
		// ensures correctness on the next launch after that process finishes.
		_ = lockFH.Close()
		return nil, nil
	}
	defer func() {
		// Kernel releases the flock on close; the on-disk sentinel is best-effort.
		_ = syscall.Flock(int(lockFH.Fd()), syscall.LOCK_UN)
		_ = lockFH.Close()
		_ = os.Remove(lockFile)
	}()

	report := &UnifiedSkillsMigrationReport{}
	absBase, err := filepath.Abs(cfg.BaseDir)
	if err != nil {
		return nil, fmt.Errorf("abs BaseDir: %w", err)
	}
	// Resolve BaseDir through EvalSymlinks so containment checks behave
	// correctly when the base path is itself accessed through a symlink
	// alias (e.g., /var → /private/var on macOS, or user-configured aliases).
	// EvalSymlinks requires the path to exist; on a fresh install BaseDir
	// should already exist because config.Load() creates it, but we fall
	// back to the un-resolved absBase if resolution fails.
	if resolved, rerr := filepath.EvalSymlinks(absBase); rerr == nil {
		absBase = resolved
	}

	// D2: purge cwd-* rows + their installations + on-disk symlinks + skill_tags.
	if err := purgeCwdSkills(database, absBase, report); err != nil {
		return nil, fmt.Errorf("purge cwd skills: %w", err)
	}

	// D3-normalize: rewrite file_path from <dir>/skill.md to <dir> when the dir
	// exists under BaseDir. Outside-BaseDir rows are left to D5.
	if err := normalizeLocalFilePaths(database, absBase, report); err != nil {
		return nil, fmt.Errorf("normalize file paths: %w", err)
	}

	// D5: migrate project-scoped local-* rows into BaseDir/skills.
	if err := migrateProjectScopedLocalSkills(database, absBase, report); err != nil {
		return nil, fmt.Errorf("migrate project-scoped skills: %w", err)
	}

	// Marker — only written on pass-level success.
	if err := os.WriteFile(markerFile, []byte("done\n"), 0644); err != nil {
		return report, fmt.Errorf("write marker: %w", err)
	}

	return report, nil
}

// RunStartupMigrations is the convenience wrapper binary entry points call.
// It invokes MigrateToUnifiedLocalSkills and, when the migration actually did
// something, prints a short user-facing notice to stderr summarizing the
// changes. Errors are returned so the caller can decide whether to log and
// continue (the expected behavior: migration errors are non-fatal).
func RunStartupMigrations(database *db.DB, cfg *config.Config) (*UnifiedSkillsMigrationReport, error) {
	report, err := MigrateToUnifiedLocalSkills(database, cfg)
	if err != nil {
		return report, err
	}
	if report == nil {
		return nil, nil
	}

	// Only print when something happened; suppress a stderr line on first launch
	// for users who had no broken data.
	var parts []string
	if report.CwdPurged > 0 {
		parts = append(parts, fmt.Sprintf("Removed %d stale project-local skill entries", report.CwdPurged))
	}
	if report.Migrated > 0 {
		parts = append(parts, fmt.Sprintf("migrated %d project-scoped skills into ~/.agents/skulto/skills/", report.Migrated))
	}
	if report.Collisions > 0 {
		parts = append(parts, fmt.Sprintf("%d collisions resolved in favor of global version", report.Collisions))
	}
	if len(parts) > 0 {
		fmt.Fprintf(os.Stderr, "%s.\n", strings.Join(parts, "; "))
	}
	return report, nil
}

// purgeCwdSkills deletes every skill row whose ID starts with "cwd-", along
// with its skill_installations rows, its skill_tags associations, and any
// AI-tool symlink on disk whose target resolves OUTSIDE absBase (per spec D2).
// Links whose targets resolve into ~/.agents/skulto/ are left alone — they
// may be legitimate global-skill symlinks the user wants to keep.
func purgeCwdSkills(database *db.DB, absBase string, report *UnifiedSkillsMigrationReport) error {
	skills, err := database.GetAllSkills()
	if err != nil {
		return err
	}
	for i := range skills {
		skill := skills[i]
		if !strings.HasPrefix(skill.ID, "cwd-") {
			continue
		}

		// Best-effort: remove on-disk symlinks whose target resolves outside
		// BaseDir. Errors here are logged and do not prevent DB cleanup.
		if installs, ierr := database.GetInstallations(skill.ID); ierr == nil {
			for _, inst := range installs {
				if !exists(inst.SymlinkPath) || !isSymlink(inst.SymlinkPath) {
					continue
				}
				target, rerr := os.Readlink(inst.SymlinkPath)
				if rerr != nil {
					continue
				}
				if !filepath.IsAbs(target) {
					target = filepath.Join(filepath.Dir(inst.SymlinkPath), target)
				}
				target = filepath.Clean(target)
				// Spec D2: only remove when the target resolves OUTSIDE BaseDir.
				// Broken/dangling symlinks are left alone — we cannot verify
				// they violate the invariant, and leaving them is strictly
				// safer than removing something a user may want to preserve.
				resolvedTarget, evalErr := filepath.EvalSymlinks(target)
				if evalErr != nil {
					continue
				}
				if !isUnder(absBase, resolvedTarget) {
					if err := os.Remove(inst.SymlinkPath); err != nil {
						log.Errorf("purge cwd symlink %s: %v", inst.SymlinkPath, err)
					}
				}
			}
		}

		// Clean up skill_tags rows explicitly: HardDeleteSkill does not cascade.
		if err := database.Exec("DELETE FROM skill_tags WHERE skill_id = ?", skill.ID).Error; err != nil {
			log.Errorf("delete skill_tags for %s: %v", skill.ID, err)
		}

		// Remove installation records.
		if err := database.RemoveAllInstallations(skill.ID); err != nil {
			log.Errorf("remove installations for %s: %v", skill.ID, err)
		}

		// Hard-delete the skill row.
		if err := database.HardDeleteSkill(skill.ID); err != nil {
			log.Errorf("hard-delete skill %s: %v", skill.ID, err)
			continue
		}
		report.CwdPurged++
	}
	return nil
}

// normalizeLocalFilePaths rewrites file_path for local-* rows where:
//   - file_path ends in /skill.md (case-insensitive), AND
//   - the containing directory exists, AND
//   - that containing directory is inside cfg.BaseDir.
//
// Outside-BaseDir rows are left for D5 so it has a clear invariant (the row's
// oldDir is the thing to migrate; D5 handles the skill.md → dir step itself).
func normalizeLocalFilePaths(database *db.DB, absBase string, report *UnifiedSkillsMigrationReport) error {
	skills, err := database.GetAllSkills()
	if err != nil {
		return err
	}
	for i := range skills {
		skill := skills[i]
		if !skill.IsLocal || !strings.HasPrefix(skill.ID, "local-") {
			continue
		}
		if !strings.EqualFold(filepath.Base(skill.FilePath), "skill.md") {
			continue
		}
		candidateDir := filepath.Dir(skill.FilePath)
		absDir, err := filepath.Abs(candidateDir)
		if err != nil {
			log.Errorf("normalize: abs %s: %v", candidateDir, err)
			continue
		}
		if !isUnder(absBase, resolveSymlinksLenient(absDir)) {
			continue // D5 handles outside-BaseDir
		}
		info, err := os.Stat(absDir)
		if err != nil || !info.IsDir() {
			continue
		}
		skill.FilePath = absDir
		if err := database.UpdateSkill(&skill); err != nil {
			log.Errorf("normalize file_path for %s: %v", skill.ID, err)
			continue
		}
		report.NormalizedFilePaths++
	}
	return nil
}

// migrateProjectScopedLocalSkills implements D5. For each local-* row whose
// file_path (or its parent, if file_path ends in skill.md) resolves to a
// directory outside cfg.BaseDir:
//
//   - If <BaseDir>/skills/<slug> already exists (collision): rewrite DB
//     file_path to the global dir, rewrite installation symlinks that
//     resolved into the project dir, leave both directories untouched on disk.
//   - Else: copy oldDir → newDir, rename oldDir → oldDir+".skulto-backup",
//     create a back-symlink from oldDir → newDir, delete the backup on
//     success. On any failure during the swap, roll back and skip the row.
func migrateProjectScopedLocalSkills(database *db.DB, absBase string, report *UnifiedSkillsMigrationReport) error {
	skills, err := database.GetAllSkills()
	if err != nil {
		return err
	}
	globalSkillsDir := filepath.Join(absBase, "skills")
	if err := os.MkdirAll(globalSkillsDir, 0755); err != nil {
		return fmt.Errorf("mkdir global skills dir: %w", err)
	}

	for i := range skills {
		skill := skills[i]
		if !skill.IsLocal || !strings.HasPrefix(skill.ID, "local-") {
			continue
		}
		if skill.Slug == "" {
			continue
		}

		// Resolve oldDir: if file_path ends in skill.md, take the containing dir.
		oldDir := skill.FilePath
		if strings.EqualFold(filepath.Base(oldDir), "skill.md") {
			oldDir = filepath.Dir(oldDir)
		}
		absOld, err := filepath.Abs(oldDir)
		if err != nil {
			log.Errorf("migrate: abs %s: %v", oldDir, err)
			continue
		}

		// Only operate on rows whose source is OUTSIDE BaseDir. Compare via
		// EvalSymlinks so an absBase/child pair that differ only by an
		// intermediate symlink alias are treated as equal.
		if isUnder(absBase, resolveSymlinksLenient(absOld)) {
			continue
		}

		// Source must exist and be a directory to migrate.
		srcInfo, err := os.Stat(absOld)
		if err != nil || !srcInfo.IsDir() {
			log.Errorf("migrate: source missing or not a directory (%s): %v", absOld, err)
			continue
		}

		newDir := filepath.Join(globalSkillsDir, skill.Slug)

		// Collision is "newDir is a real managed directory under BaseDir".
		// A random file, a symlink pointing outside BaseDir, or a broken
		// path here is NOT a collision — treat it as "not a collision, no
		// safe place to write" and skip with a log.
		collision, unsafeNew := isManagedDir(newDir, absBase)
		if unsafeNew {
			log.Errorf("migration for %s: %s exists but is not a managed directory under BaseDir; skipping to avoid data loss",
				skill.ID, newDir)
			continue
		}

		if collision {
			// Collision → keep-global. Update DB first; symlink rewrite is a
			// best-effort cosmetic follow-up. Neither directory is touched
			// on disk. If UpdateSkill fails, the row is left untouched and
			// logged — per spec D5, per-row failures do not block the
			// marker from being written (the alternative would retry on
			// every launch forever against potentially-damaged FS state).
			// Users whose rows are stranded here can recover manually by
			// running `skulto ingest <slug>`.
			originalFilePath := skill.FilePath
			skill.FilePath = newDir
			if uerr := database.UpdateSkill(&skill); uerr != nil {
				log.Errorf("update file_path for %s: %v", skill.ID, uerr)
				// Restore in-memory value so nothing downstream leaks it.
				skill.FilePath = originalFilePath
				continue
			}
			// Rewrite installation symlinks after the DB is consistent. Per-row
			// failures here are logged and swallowed because the authoritative
			// source-of-truth (DB) is already correct; the worst case is a
			// stale symlink the user can re-install to refresh.
			if rerr := rewriteInstallationSymlinks(database, skill.ID, absOld, newDir); rerr != nil {
				log.Errorf("rewrite symlinks for %s: %v", skill.ID, rerr)
			}
			log.Printf("%s: kept global version at %s; project copy at %s is no longer tracked by skulto\n",
				skill.Slug, newDir, absOld)
			report.Collisions++
			continue
		}

		// No collision → copy + back-symlink (two-phase swap).
		if err := copyDirForMigration(absOld, newDir); err != nil {
			log.Errorf("migration copy for %s: %v", skill.ID, err)
			_ = os.RemoveAll(newDir)
			continue
		}
		backupPath := absOld + ".skulto-backup"
		// If a stale backup already exists (e.g., a prior crashed run), refuse
		// to clobber it — better to leave the row for manual resolution.
		if _, err := os.Lstat(backupPath); err == nil {
			log.Errorf("migration for %s: backup path %s already exists; skipping to avoid clobber", skill.ID, backupPath)
			_ = os.RemoveAll(newDir)
			continue
		}
		if err := os.Rename(absOld, backupPath); err != nil {
			log.Errorf("migration rename for %s: %v", skill.ID, err)
			_ = os.RemoveAll(newDir)
			continue
		}
		if err := os.Symlink(newDir, absOld); err != nil {
			log.Errorf("migration symlink for %s: %v", skill.ID, err)
			// Roll back: restore original dir from backup, drop the new copy.
			if rerr := os.Rename(backupPath, absOld); rerr != nil {
				log.Errorf("migration rollback rename for %s: %v", skill.ID, rerr)
			}
			_ = os.RemoveAll(newDir)
			continue
		}

		// Update DB row BEFORE touching installation symlinks. If the DB
		// update fails, we still have the backup and can roll the whole
		// filesystem swap back — leaving the row untouched per spec D5.
		originalFilePath := skill.FilePath
		skill.FilePath = newDir
		if err := database.UpdateSkill(&skill); err != nil {
			log.Errorf("update file_path for %s: %v", skill.ID, err)
			skill.FilePath = originalFilePath
			// Roll back FS swap: drop the new symlink, drop the new copy,
			// restore the original directory from backup. If any rollback
			// step fails, log — but the backup stays on disk so the next
			// launch can recover.
			if rerr := os.Remove(absOld); rerr != nil {
				log.Errorf("rollback remove symlink for %s: %v", skill.ID, rerr)
			}
			if rerr := os.Rename(backupPath, absOld); rerr != nil {
				log.Errorf("rollback restore dir for %s: %v", skill.ID, rerr)
			}
			_ = os.RemoveAll(newDir)
			continue
		}

		// DB is now authoritative. Drop the backup and rewrite installation
		// symlinks as a best-effort follow-up.
		if err := os.RemoveAll(backupPath); err != nil {
			log.Errorf("migration cleanup backup for %s: %v", skill.ID, err)
			// Non-fatal; the back-symlink and DB are consistent.
		}
		if rerr := rewriteInstallationSymlinks(database, skill.ID, absOld, newDir); rerr != nil {
			log.Errorf("rewrite symlinks for %s: %v", skill.ID, rerr)
		}
		report.Migrated++
	}
	return nil
}

// rewriteInstallationSymlinks walks every skill_installations row for the
// given skill and, for each one whose on-disk symlink resolves to oldDir (or
// anywhere inside oldDir), replaces it with a symlink pointing at the
// corresponding location under newDir. Symlinks that resolve elsewhere are
// left alone — the user may have pointed them somewhere intentionally.
func rewriteInstallationSymlinks(database *db.DB, skillID, oldDir, newDir string) error {
	installs, err := database.GetInstallations(skillID)
	if err != nil {
		return err
	}
	absOld := filepath.Clean(oldDir)
	absNew := filepath.Clean(newDir)
	for _, inst := range installs {
		if !exists(inst.SymlinkPath) || !isSymlink(inst.SymlinkPath) {
			continue
		}
		target, err := os.Readlink(inst.SymlinkPath)
		if err != nil {
			continue
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(inst.SymlinkPath), target)
		}
		target = filepath.Clean(target)
		// Decide whether target is inside absOld.
		rel, err := filepath.Rel(absOld, target)
		if err != nil {
			continue
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		// target is absOld itself (rel=".") or a path inside it.
		var newTarget string
		if rel == "." {
			newTarget = absNew
		} else {
			newTarget = filepath.Join(absNew, rel)
		}
		// Atomic swap: create the replacement symlink at a temp path next to
		// the existing one, then rename it over. os.Rename on POSIX replaces
		// symlinks atomically, so the installation location is never missing.
		// If the temp create fails or the rename fails, the original symlink
		// remains in place (possibly pointing at the now-defunct oldDir, but
		// still present — the user can re-install to refresh).
		tmpPath := inst.SymlinkPath + ".skulto-tmp"
		// Clean up any leftover temp from a prior crashed run.
		_ = os.Remove(tmpPath)
		if err := os.Symlink(newTarget, tmpPath); err != nil {
			log.Errorf("rewrite symlinks: create tmp %s -> %s: %v", tmpPath, newTarget, err)
			continue
		}
		if err := os.Rename(tmpPath, inst.SymlinkPath); err != nil {
			log.Errorf("rewrite symlinks: rename %s -> %s: %v", tmpPath, inst.SymlinkPath, err)
			_ = os.Remove(tmpPath)
			continue
		}
	}
	return nil
}

// copyDirForMigration copies every file/dir under src into dst, preserving
// file modes. Kept local to avoid importing the discovery package.
func copyDirForMigration(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}
		dstPath := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		// Treat symlinks as copies of their current target content; the common
		// case for a .skulto/skills/<slug>/ folder is regular files.
		if info.Mode()&os.ModeSymlink != 0 {
			target, rerr := os.Readlink(path)
			if rerr != nil {
				return rerr
			}
			return os.Symlink(target, dstPath)
		}
		return copyFileForMigration(path, dstPath)
	})
}

// copyFileForMigration copies one file, preserving its mode.
func copyFileForMigration(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	fi, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fi.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	_, err = io.Copy(out, in)
	return err
}

// isManagedDir reports whether newDir qualifies as a "pre-existing global
// skill directory" for the D5 collision check. It returns (true, false) when
// newDir is a directory (possibly via a symlink) that resolves to a path
// under absBase. It returns (false, false) when newDir simply does not exist.
// It returns (false, true) when newDir exists but is unsafe to treat as a
// collision (a regular file, or a symlink that resolves outside BaseDir or
// is broken). Callers should skip the row in the unsafe case.
func isManagedDir(newDir, absBase string) (collision, unsafe bool) {
	lstat, err := os.Lstat(newDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, false
		}
		// Permission or other error → treat as unsafe; we don't know.
		return false, true
	}
	// Reject non-directory regular files outright.
	if lstat.Mode().IsRegular() {
		return false, true
	}
	// Follow symlinks / verify it really is a directory that resolves under BaseDir.
	resolved, err := filepath.EvalSymlinks(newDir)
	if err != nil {
		return false, true // broken symlink
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return false, true
	}
	// Resolve absBase as well so the comparison is symlink-aware (on macOS,
	// /var is a symlink to /private/var — a direct prefix comparison would
	// spuriously reject legitimate targets).
	resolvedBase, err := filepath.EvalSymlinks(absBase)
	if err != nil {
		// If BaseDir itself can't be resolved we can't make a safety call.
		return false, true
	}
	if !isUnder(resolvedBase, resolved) {
		return false, true
	}
	return true, false
}

// resolveSymlinksLenient returns filepath.EvalSymlinks(p) if it succeeds,
// otherwise the input unchanged. Used to normalize candidate paths before
// containment checks without rejecting non-existent paths (which the callers
// want to classify as "outside BaseDir" without erroring).
func resolveSymlinksLenient(p string) string {
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved
	}
	return p
}

// isUnder reports whether child is the same path as parent or is nested
// inside parent. Both arguments should be absolute and cleaned. Using
// filepath.Rel keeps the comparison OS-aware (path separators).
func isUnder(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}
