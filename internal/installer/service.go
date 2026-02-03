package installer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/telemetry"
)

// InstallOptions configures an install operation.
type InstallOptions struct {
	Platforms []string       // nil = all user platforms
	Scopes    []InstallScope // nil = default to global
	Confirm   bool           // true = skip prompts (for non-interactive mode)
}

// InstallResult captures the outcome of an installation.
type InstallResult struct {
	Skill     *models.Skill
	Locations []InstallLocation
	Errors    []error
}

// DetectedPlatform describes a platform with its installation status.
// This extends the basic PlatformInfo with detection and path details.
type DetectedPlatform struct {
	ID       string // Platform ID (e.g., "claude")
	Name     string // Display name (e.g., "Claude Code")
	Path     string // Installation path (e.g., "~/.claude/skills")
	Detected bool   // Whether the platform was detected on the system
}

// InstallService provides unified installation operations across CLI, TUI, and MCP.
// It wraps the underlying Installer for symlink operations and handles skill/source
// lookup, platform detection, and telemetry.
type InstallService struct {
	installer *Installer
	db        *db.DB
	cfg       *config.Config
	telemetry telemetry.Client
}

// NewInstallService creates a new install service.
func NewInstallService(database *db.DB, cfg *config.Config, tel telemetry.Client) *InstallService {
	if cfg == nil {
		cfg = &config.Config{}
	}

	inst := New(database, cfg)

	return &InstallService{
		installer: inst,
		db:        database,
		cfg:       cfg,
		telemetry: tel,
	}
}

// DetectPlatforms returns all known platforms with their detection status.
// Detection is done by checking if platform commands exist in PATH or
// if platform directories exist.
func (s *InstallService) DetectPlatforms(ctx context.Context) ([]DetectedPlatform, error) {
	// Build result with all platforms
	allPlatforms := AllPlatforms()
	result := make([]DetectedPlatform, 0, len(allPlatforms))

	for _, p := range allPlatforms {
		info := p.Info()

		platformInfo := DetectedPlatform{
			ID:       string(p),
			Name:     info.Name,
			Path:     info.SkillsPath,
			Detected: isPlatformDetected(p),
		}
		result = append(result, platformInfo)
	}

	return result, nil
}

// isPlatformDetected checks if a platform is installed on the system.
// Uses the platform registry config for data-driven detection:
// command in PATH, project directory, and global directory.
func isPlatformDetected(p Platform) bool {
	info := p.Info()

	// 1. Check command in PATH
	if info.Command != "" {
		if _, err := exec.LookPath(info.Command); err == nil {
			return true
		}
	}

	// 2. Check project-level directory
	if info.ProjectDir != "" {
		if _, err := os.Stat(info.ProjectDir); err == nil {
			return true
		}
	}

	// 3. Check global/home directory
	if info.GlobalDir != "" {
		globalPath := expandHomePath(info.GlobalDir)
		if _, err := os.Stat(globalPath); err == nil {
			return true
		}
	}

	// 4. Check platform-specific paths
	for _, path := range info.PlatformSpecificPaths {
		expanded := os.ExpandEnv(path)
		if _, err := os.Stat(expanded); err == nil {
			return true
		}
	}

	return false
}

// expandHomePath replaces ~ with the home directory.
func expandHomePath(path string) string {
	if len(path) >= 2 && path[0] == '~' && path[1] == '/' {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// Install installs a skill to the specified locations.
func (s *InstallService) Install(ctx context.Context, slug string, opts InstallOptions) (*InstallResult, error) {
	// Look up skill
	skill, err := s.db.GetSkillBySlug(slug)
	if err != nil {
		return nil, fmt.Errorf("failed to look up skill: %w", err)
	}
	if skill == nil {
		return nil, fmt.Errorf("skill not found: %s", slug)
	}

	// Get source if skill has one
	var source *models.Source
	if skill.SourceID != nil {
		source, _ = s.db.GetSource(*skill.SourceID)
	}

	// Determine platforms
	platforms := opts.Platforms
	if len(platforms) == 0 {
		// Default to user's configured platforms
		userState, _ := s.db.GetUserState()
		if userState != nil {
			userPlatforms := userState.GetAITools()
			if len(userPlatforms) > 0 {
				platforms = userPlatforms
			}
		}
		if len(platforms) == 0 {
			platforms = []string{string(PlatformClaude)} // fallback
		}
	}

	// Determine scopes
	scopes := opts.Scopes
	if len(scopes) == 0 {
		scopes = []InstallScope{ScopeGlobal}
	}

	// Build locations
	var locations []InstallLocation
	for _, platformStr := range platforms {
		platform := PlatformFromString(platformStr)
		if platform == "" {
			continue // Skip invalid platforms
		}
		for _, scope := range scopes {
			loc, err := NewInstallLocation(platform, scope)
			if err != nil {
				continue // Skip locations that can't be resolved
			}
			locations = append(locations, loc)
		}
	}

	// Perform installation
	if skill.IsLocal {
		// Local skill - use InstallLocalSkillTo
		if err := s.installer.InstallLocalSkillTo(ctx, skill, skill.FilePath, locations); err != nil {
			return &InstallResult{Skill: skill, Errors: []error{err}}, err
		}
	} else if source != nil {
		// Remote skill with source - use InstallTo
		if err := s.installer.InstallTo(ctx, skill, source, locations); err != nil {
			return &InstallResult{Skill: skill, Errors: []error{err}}, err
		}
	} else {
		return nil, fmt.Errorf("cannot install skill without source: %s", slug)
	}

	// Get actual installed locations
	installed, _ := s.installer.GetInstallLocations(skill.ID)

	// Track telemetry
	if s.telemetry != nil {
		s.telemetry.TrackSkillInstalled(skill.Title, skill.Category, skill.IsLocal, len(installed))
	}

	return &InstallResult{
		Skill:     skill,
		Locations: installed,
	}, nil
}

// InstallBatch installs multiple skills with the same options.
// Returns results for each skill, including errors.
func (s *InstallService) InstallBatch(ctx context.Context, slugs []string, opts InstallOptions) []InstallResult {
	results := make([]InstallResult, len(slugs))

	for i, slug := range slugs {
		result, err := s.Install(ctx, slug, opts)
		if err != nil {
			// Still look up the skill for the result if possible
			skill, _ := s.db.GetSkillBySlug(slug)
			results[i] = InstallResult{
				Skill:  skill,
				Errors: []error{err},
			}
		} else {
			results[i] = *result
		}
	}

	return results
}

// Uninstall removes a skill from the specified locations.
// If locations is nil or empty, does nothing (use UninstallAll for that).
func (s *InstallService) Uninstall(ctx context.Context, slug string, locations []InstallLocation) error {
	// Look up skill
	skill, err := s.db.GetSkillBySlug(slug)
	if err != nil {
		return fmt.Errorf("failed to look up skill: %w", err)
	}
	if skill == nil {
		return fmt.Errorf("skill not found: %s", slug)
	}

	if len(locations) == 0 {
		return nil // Nothing to uninstall
	}

	// Uninstall from specified locations
	if err := s.installer.UninstallFrom(ctx, skill, locations); err != nil {
		return err
	}

	// Track telemetry
	if s.telemetry != nil {
		s.telemetry.TrackSkillUninstalled(skill.Title, skill.Category, skill.IsLocal)
	}

	return nil
}

// UninstallAll removes a skill from all installed locations.
func (s *InstallService) UninstallAll(ctx context.Context, slug string) error {
	// Look up skill
	skill, err := s.db.GetSkillBySlug(slug)
	if err != nil {
		return fmt.Errorf("failed to look up skill: %w", err)
	}
	if skill == nil {
		// If skill doesn't exist, nothing to uninstall
		return nil
	}

	// Uninstall from all locations
	if err := s.installer.UninstallAll(ctx, skill); err != nil {
		return err
	}

	// Track telemetry
	if s.telemetry != nil {
		s.telemetry.TrackSkillUninstalled(skill.Title, skill.Category, skill.IsLocal)
	}

	return nil
}

// GetInstallLocations returns all locations where a skill is currently installed.
func (s *InstallService) GetInstallLocations(ctx context.Context, slug string) ([]InstallLocation, error) {
	// Look up skill
	skill, err := s.db.GetSkillBySlug(slug)
	if err != nil {
		return nil, err
	}
	if skill == nil {
		// Return empty slice for not found, not an error
		return []InstallLocation{}, nil
	}

	locations, err := s.installer.GetInstallLocations(skill.ID)
	if err != nil {
		return nil, err
	}

	return locations, nil
}

// FetchSkillsFromURL fetches skills from a repository URL.
// It auto-adds the repository if not already present.
// This method is a placeholder - actual implementation requires scraper integration.
func (s *InstallService) FetchSkillsFromURL(ctx context.Context, url string) ([]*models.Skill, error) {
	// This is a stub - full implementation would:
	// 1. Parse URL to get owner/repo
	// 2. Check if source already exists in DB
	// 3. If not, use scraper to add and index the repo
	// 4. Return all skills from that source

	// For now, return error indicating this needs implementation
	return nil, fmt.Errorf("FetchSkillsFromURL not yet implemented for URL: %s", url)
}

// InstalledSkillSummary represents a skill and where it's installed.
type InstalledSkillSummary struct {
	Slug      string                      // Skill slug (e.g., "teach", "superplan")
	Title     string                      // Skill display name
	Locations map[Platform][]InstallScope // Platform -> scopes installed
	IsLocal   bool                        // True for locally ingested skills
}

// GetInstalledSkillsSummary returns all installed skills with their installation locations.
// Only returns skills that have at least one installation.
// Results are sorted by skill slug for consistent output.
func (s *InstallService) GetInstalledSkillsSummary(ctx context.Context) ([]InstalledSkillSummary, error) {
	// 1. Get all installations from database
	installations, err := s.db.GetAllInstallations()
	if err != nil {
		return nil, fmt.Errorf("failed to get installations: %w", err)
	}

	if len(installations) == 0 {
		return []InstalledSkillSummary{}, nil
	}

	// 2. Group by skill ID and collect platforms/scopes
	type skillData struct {
		slug      string
		title     string
		isLocal   bool
		locations map[Platform][]InstallScope
	}
	skillMap := make(map[string]*skillData)

	for _, inst := range installations {
		data, exists := skillMap[inst.SkillID]
		if !exists {
			// Look up skill details
			skill, err := s.db.GetSkill(inst.SkillID)
			if err != nil || skill == nil {
				continue // Skip installations for unknown skills
			}
			data = &skillData{
				slug:      skill.Slug,
				title:     skill.Title,
				isLocal:   skill.IsLocal,
				locations: make(map[Platform][]InstallScope),
			}
			skillMap[inst.SkillID] = data
		}

		// Add this location
		platform := Platform(inst.Platform)
		scope := InstallScope(inst.Scope)

		// Avoid duplicate scopes for same platform
		scopes := data.locations[platform]
		found := false
		for _, s := range scopes {
			if s == scope {
				found = true
				break
			}
		}
		if !found {
			data.locations[platform] = append(scopes, scope)
		}
	}

	// 3. Convert to slice and sort by slug
	result := make([]InstalledSkillSummary, 0, len(skillMap))
	for _, data := range skillMap {
		result = append(result, InstalledSkillSummary{
			Slug:      data.slug,
			Title:     data.title,
			IsLocal:   data.isLocal,
			Locations: data.locations,
		})
	}

	// Sort by slug
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Slug > result[j].Slug {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}
