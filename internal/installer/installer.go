package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/log"
	"github.com/asteroid-belt/skulto/internal/models"
)

// Installer handles skill installation and uninstallation.
// Skills are installed by creating symlinks from repository skill directories
// to the platform skill directories (e.g., ~/.claude/skills/).
type Installer struct {
	db    *db.DB
	cfg   *config.Config
	paths *PathResolver
}

// New creates a new installer.
func New(database *db.DB, conf *config.Config) *Installer {
	return &Installer{
		db:    database,
		cfg:   conf,
		paths: NewPathResolver(conf),
	}
}

// Install installs a skill to all configured AI tools by creating symlinks.
// The skill source directory must exist in the cloned repository.
// This is a convenience wrapper around InstallTo that uses the user's configured platforms.
func (i *Installer) Install(ctx context.Context, skill *models.Skill, source *models.Source) error {
	if skill == nil {
		return fmt.Errorf("skill cannot be nil")
	}

	if skill.Slug == "" {
		return ErrInvalidSkill
	}

	if source == nil {
		return fmt.Errorf("source cannot be nil for symlink-based installation")
	}

	// Get user's selected AI tools
	userState, err := i.db.GetUserState()
	if err != nil {
		return fmt.Errorf("failed to get user state: %w", err)
	}

	platforms := parsePlatforms(userState)
	if len(platforms) == 0 {
		return ErrNoToolsSelected
	}

	// Build install locations from platforms (global scope by default)
	locations := make([]InstallLocation, 0, len(platforms))
	for _, platform := range platforms {
		loc, err := NewInstallLocation(platform, ScopeGlobal)
		if err != nil {
			continue
		}
		locations = append(locations, loc)
	}

	if len(locations) == 0 {
		return ErrNoToolsSelected
	}

	// Delegate to InstallTo which creates proper skill_installations records
	return i.InstallTo(ctx, skill, source, locations)
}

// Uninstall removes skill symlinks from all platforms.
// This is a convenience wrapper that delegates to UninstallAll.
func (i *Installer) Uninstall(ctx context.Context, skill *models.Skill) error {
	return i.UninstallAll(ctx, skill)
}

// ReInstall reinstalls a skill by uninstalling and reinstalling.
func (i *Installer) ReInstall(ctx context.Context, skill *models.Skill, source *models.Source) error {
	if err := i.Uninstall(ctx, skill); err != nil {
		return fmt.Errorf("uninstall failed: %w", err)
	}
	return i.Install(ctx, skill, source)
}

// IsInstalled checks if a skill is installed.
// Uses skill_installations as the single source of truth.
func (i *Installer) IsInstalled(skillID string) (bool, error) {
	// First check if skill exists
	skill, err := i.db.GetSkill(skillID)
	if err != nil {
		return false, err
	}
	if skill == nil {
		return false, ErrSkillNotFound
	}
	// Use skill_installations table as the source of truth
	return i.db.HasInstallations(skillID)
}

// installToLocationsInternal is the shared implementation for installing skills via symlinks.
// Both InstallTo and InstallLocalSkillTo delegate to this method after resolving the source path.
func (i *Installer) installToLocationsInternal(skill *models.Skill, sourcePath string, locations []InstallLocation) error {
	// Create symlinks for each location
	var createdInstalls []models.SkillInstallation
	var lastErr error

	// Track directories created by MkdirAll for cleanup on total failure
	createdDirs := make(map[string]bool)

	for _, loc := range locations {
		targetPath := loc.GetSkillPath(skill.Slug)
		if targetPath == "" {
			continue
		}

		// Ensure target parent directory exists
		targetDir := filepath.Dir(targetPath)
		dirExisted := exists(targetDir)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			log.Printf("skulto: mkdir failed for %s: %v", targetDir, err)
			lastErr = err
			continue
		}
		if !dirExisted {
			createdDirs[targetDir] = true
		}

		// Remove existing symlink or directory
		if exists(targetPath) {
			if err := os.RemoveAll(targetPath); err != nil {
				lastErr = err
				continue
			}
		}

		// Create symlink: targetPath -> sourcePath
		if err := os.Symlink(sourcePath, targetPath); err != nil {
			log.Printf("skulto: symlink failed %s → %s: %v", sourcePath, targetPath, err)
			lastErr = err
			continue
		}

		// Record installation
		install := models.SkillInstallation{
			SkillID:     skill.ID,
			Platform:    string(loc.Platform),
			Scope:       string(loc.Scope),
			BasePath:    loc.BasePath,
			SymlinkPath: targetPath,
		}
		if err := i.db.AddInstallation(&install); err != nil {
			// Rollback symlink
			log.Printf("skulto: AddInstallation failed for %s, rolling back symlink: %v", targetPath, err)
			_ = os.Remove(targetPath)
			lastErr = err
			continue
		}

		createdInstalls = append(createdInstalls, install)
	}

	// If no locations succeeded, clean up empty directories and return error
	if len(createdInstalls) == 0 {
		for dir := range createdDirs {
			_ = os.Remove(dir) // only removes if empty
		}
		if lastErr != nil {
			return fmt.Errorf("failed to install to any location: %w", lastErr)
		}
		return ErrSymlinkFailed
	}

	// Update legacy IsInstalled flag for backward compatibility
	if err := i.db.SetInstalled(skill.ID, true); err != nil {
		// Rollback: remove created symlinks and installations
		for _, inst := range createdInstalls {
			_ = os.Remove(inst.SymlinkPath)
			_ = i.db.RemoveInstallation(inst.SkillID, inst.Platform, inst.Scope, inst.BasePath)
		}
		return fmt.Errorf("database update failed: %w", err)
	}

	return nil
}

// InstallTo installs a skill to specific locations by creating symlinks.
// This is the new location-aware installation method.
func (i *Installer) InstallTo(ctx context.Context, skill *models.Skill, source *models.Source, locations []InstallLocation) error {
	if skill == nil {
		return fmt.Errorf("skill cannot be nil")
	}
	if skill.Slug == "" {
		return ErrInvalidSkill
	}
	if source == nil {
		return fmt.Errorf("source cannot be nil for symlink-based installation")
	}
	if len(locations) == 0 {
		return fmt.Errorf("no installation locations specified")
	}

	// Get source skill path in repository using the skill's actual FilePath
	sourcePath := i.paths.GetSourcePath(source.Owner, source.Repo, skill.FilePath)

	// Verify source path exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("skill directory not found: %s", sourcePath)
	}

	return i.installToLocationsInternal(skill, sourcePath, locations)
}

// InstallLocalSkillTo installs a local skill (from ~/.skulto/skills) to specific locations.
// Unlike InstallTo, this doesn't require a Source object since local skills are self-contained.
func (i *Installer) InstallLocalSkillTo(ctx context.Context, skill *models.Skill, sourcePath string, locations []InstallLocation) error {
	if skill == nil {
		return fmt.Errorf("skill cannot be nil")
	}
	if skill.Slug == "" {
		return ErrInvalidSkill
	}
	if sourcePath == "" {
		return fmt.Errorf("source path cannot be empty")
	}
	if len(locations) == 0 {
		return fmt.Errorf("no installation locations specified")
	}

	// Verify source path exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("skill directory not found: %s", sourcePath)
	}

	return i.installToLocationsInternal(skill, sourcePath, locations)
}

// UninstallFrom removes a skill from specific locations.
func (i *Installer) UninstallFrom(ctx context.Context, skill *models.Skill, locations []InstallLocation) error {
	if skill == nil {
		return fmt.Errorf("skill cannot be nil")
	}
	if skill.Slug == "" {
		return ErrInvalidSkill
	}

	var errors []error
	for _, loc := range locations {
		targetPath := loc.GetSkillPath(skill.Slug)
		if targetPath == "" {
			continue
		}

		if exists(targetPath) && isSymlink(targetPath) {
			if err := os.Remove(targetPath); err != nil {
				errors = append(errors, fmt.Errorf("%s: %w", loc.ID(), err))
				continue
			}
		}

		// Remove installation record
		if err := i.db.RemoveInstallation(skill.ID, string(loc.Platform), string(loc.Scope), loc.BasePath); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("uninstall errors: %v", errors)
	}

	// Check if any installations remain
	remaining, err := i.db.GetInstallations(skill.ID)
	if err == nil && len(remaining) == 0 {
		// No installations remain, update legacy flag
		_ = i.db.SetInstalled(skill.ID, false)
	}

	return nil
}

// UninstallAll removes a skill from all installed locations.
// It first checks the skill_installations table for recorded installations,
// then falls back to checking legacy global paths for backward compatibility.
func (i *Installer) UninstallAll(ctx context.Context, skill *models.Skill) error {
	if skill == nil {
		return fmt.Errorf("skill cannot be nil")
	}

	if skill.Slug == "" {
		return ErrInvalidSkill
	}

	var errors []error

	// First, remove symlinks from recorded installations (new system)
	installations, err := i.db.GetInstallations(skill.ID)
	if err != nil {
		errors = append(errors, err)
	}

	for _, inst := range installations {
		if exists(inst.SymlinkPath) && isSymlink(inst.SymlinkPath) {
			if err := os.Remove(inst.SymlinkPath); err != nil {
				errors = append(errors, err)
			}
		}
	}

	// Remove all installation records
	if err := i.db.RemoveAllInstallations(skill.ID); err != nil {
		errors = append(errors, err)
	}

	// Also check legacy global paths for backward compatibility
	// This handles skills installed before the new location tracking system
	for _, platform := range AllPlatforms() {
		targetPath, err := platform.GetSkillPath(skill.Slug)
		if err != nil {
			continue
		}

		if exists(targetPath) && isSymlink(targetPath) {
			if err := os.Remove(targetPath); err != nil {
				errors = append(errors, fmt.Errorf("%s: %w", platform, err))
			}
		}
	}

	// Update legacy flag
	_ = i.db.SetInstalled(skill.ID, false)

	if len(errors) > 0 {
		return fmt.Errorf("uninstall errors: %v", errors)
	}

	return nil
}

// GetInstallLocations returns all locations where a skill is installed.
func (i *Installer) GetInstallLocations(skillID string) ([]InstallLocation, error) {
	installations, err := i.db.GetInstallations(skillID)
	if err != nil {
		return nil, err
	}

	locations := make([]InstallLocation, 0, len(installations))
	for _, inst := range installations {
		locations = append(locations, InstallLocation{
			Platform: PlatformFromString(inst.Platform),
			Scope:    InstallScope(inst.Scope),
			BasePath: inst.BasePath,
		})
	}
	return locations, nil
}

// Helper functions

// parsePlatforms converts UserState.AITools to []Platform.
func parsePlatforms(state *models.UserState) []Platform {
	if state == nil || len(state.GetAITools()) == 0 {
		return nil
	}

	toolNames := state.GetAITools()
	platforms := make([]Platform, 0, len(toolNames))

	for _, name := range toolNames {
		name = strings.TrimSpace(name)
		if p := PlatformFromString(name); p != "" {
			platforms = append(platforms, p)
		}
	}

	return platforms
}

// exists checks if a file or directory exists.
func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// isSymlink checks if a path is a symbolic link.
func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeSymlink) != 0
}

// SyncInstallState scans all AI tool skill directories and reconciles the database
// with the actual state of symlinks on disk. This ensures is_installed flags and
// skill_installations records match reality.
func (i *Installer) SyncInstallState(ctx context.Context) error {
	// Track all found installations: skillID -> list of locations
	foundInstalls := make(map[string][]InstallLocation)

	// Scan all platforms and scopes
	for _, platform := range AllPlatforms() {
		for _, scope := range AllScopes() {
			if err := i.scanPlatformScope(ctx, platform, scope, foundInstalls); err != nil {
				// Log but continue - don't fail the whole sync for one platform
				continue
			}
		}
	}

	// Get all skills from database
	skills, err := i.db.GetAllSkills()
	if err != nil {
		return fmt.Errorf("failed to get skills: %w", err)
	}

	// Reconcile each skill
	for _, skill := range skills {
		found := foundInstalls[skill.ID]

		// Get current database installations
		dbInstalls, err := i.db.GetInstallations(skill.ID)
		if err != nil {
			continue
		}

		// Build set of current DB install keys for comparison
		dbInstallKeys := make(map[string]bool)
		for _, inst := range dbInstalls {
			key := fmt.Sprintf("%s:%s:%s", inst.Platform, inst.Scope, inst.BasePath)
			dbInstallKeys[key] = true
		}

		// Add missing installations to DB
		for _, loc := range found {
			key := fmt.Sprintf("%s:%s:%s", loc.Platform, loc.Scope, loc.BasePath)
			if !dbInstallKeys[key] {
				install := models.SkillInstallation{
					SkillID:     skill.ID,
					Platform:    string(loc.Platform),
					Scope:       string(loc.Scope),
					BasePath:    loc.BasePath,
					SymlinkPath: loc.GetSkillPath(skill.Slug),
				}
				_ = i.db.AddInstallation(&install)
			}
		}

		// Build set of found install keys
		foundKeys := make(map[string]bool)
		for _, loc := range found {
			key := fmt.Sprintf("%s:%s:%s", loc.Platform, loc.Scope, loc.BasePath)
			foundKeys[key] = true
		}

		// Remove orphaned installations from DB (symlink no longer exists)
		for _, inst := range dbInstalls {
			key := fmt.Sprintf("%s:%s:%s", inst.Platform, inst.Scope, inst.BasePath)
			if !foundKeys[key] {
				_ = i.db.RemoveInstallation(inst.SkillID, inst.Platform, inst.Scope, inst.BasePath)
			}
		}

		// Update is_installed flag
		hasInstalls := len(found) > 0
		if skill.IsInstalled != hasInstalls {
			_ = i.db.SetInstalled(skill.ID, hasInstalls)
		}
	}

	return nil
}

// scanPlatformScope scans a specific platform/scope directory for skill symlinks.
func (i *Installer) scanPlatformScope(ctx context.Context, platform Platform, scope InstallScope, found map[string][]InstallLocation) error {
	info := platform.Info()
	if info.SkillsPath == "" {
		return nil
	}

	basePath, err := resolveBasePath(scope)
	if err != nil {
		return err
	}

	skillsDir := filepath.Join(basePath, info.SkillsPath)

	// Check if directory exists
	if !exists(skillsDir) {
		return nil
	}

	// Read directory entries
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		// Skip hidden files
		if entry.Name()[0] == '.' {
			continue
		}

		entryPath := filepath.Join(skillsDir, entry.Name())

		// Check if it's a symlink
		if !isSymlink(entryPath) {
			continue
		}

		// Get the slug from the directory name
		slug := entry.Name()

		// Try to find skill by slug (check both local and remote patterns)
		skill := i.findSkillBySlug(slug)
		if skill == nil {
			continue
		}

		// Record this installation
		loc := InstallLocation{
			Platform: platform,
			Scope:    scope,
			BasePath: basePath,
		}
		found[skill.ID] = append(found[skill.ID], loc)
	}

	return nil
}

// findSkillBySlug finds a skill by its slug, checking various ID patterns.
func (i *Installer) findSkillBySlug(slug string) *models.Skill {
	// Try direct slug lookup first
	if skill, err := i.db.GetSkillBySlug(slug); err == nil && skill != nil {
		return skill
	}

	// Try common ID patterns
	patterns := []string{
		"local-" + slug,
		"cwd-" + slug,
		slug, // some skills might use slug as ID directly
	}

	for _, pattern := range patterns {
		if skill, err := i.db.GetSkill(pattern); err == nil && skill != nil {
			return skill
		}
	}

	return nil
}
