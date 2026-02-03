// internal/discovery/scanner.go
package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/asteroid-belt/skulto/internal/models"
)

// ScannerService discovers unmanaged skills in platform directories.
type ScannerService struct{}

// PlatformConfig contains the configuration for a detected platform.
type PlatformConfig struct {
	ID         string
	SkillsPath string
}

// DiscoveredSkillWithSource wraps a DiscoveredSkill with its platform name for display.
type DiscoveredSkillWithSource struct {
	models.DiscoveredSkill
	Platform string
}

// NewScannerService creates a new scanner service.
func NewScannerService() *ScannerService {
	return &ScannerService{}
}

// ScanDirectory scans a platform's skills directory for unmanaged skill directories.
// It returns only non-symlinked directories.
func (s *ScannerService) ScanDirectory(dir, platform, scope string) ([]models.DiscoveredSkill, error) {
	// Check if directory exists
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var discovered []models.DiscoveredSkill
	for _, entry := range entries {
		// Skip files
		if !entry.IsDir() {
			continue
		}

		entryPath := filepath.Join(dir, entry.Name())

		// Check if symlink using Lstat
		lstat, err := os.Lstat(entryPath)
		if err != nil {
			continue
		}
		if lstat.Mode()&os.ModeSymlink != 0 {
			// It's a symlink, skip it
			continue
		}

		// Found an unmanaged skill directory
		ds := models.DiscoveredSkill{
			Platform:     platform,
			Scope:        scope,
			Path:         entryPath,
			Name:         entry.Name(),
			DiscoveredAt: time.Now(),
		}
		ds.ID = ds.GenerateID()
		discovered = append(discovered, ds)
	}

	return discovered, nil
}

// CategorizeSymlink determines who manages a symlinked skill.
func (s *ScannerService) CategorizeSymlink(path string) models.ManagementSource {
	// Check if it's actually a symlink
	lstat, err := os.Lstat(path)
	if err != nil {
		return models.ManagementNone
	}
	if lstat.Mode()&os.ModeSymlink == 0 {
		return models.ManagementNone
	}

	// Read the symlink target
	target, err := os.Readlink(path)
	if err != nil {
		return models.ManagementExternal
	}

	// Resolve to absolute path if relative
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(path), target)
	}
	target = filepath.Clean(target)

	return s.categorizeByTarget(target)
}

// categorizeByTarget categorizes based on the symlink target path.
func (s *ScannerService) categorizeByTarget(target string) models.ManagementSource {
	// Check for skulto management (.skulto/skills/ or .skulto/repositories/)
	if strings.Contains(target, ".skulto"+string(filepath.Separator)+"skills") ||
		strings.Contains(target, ".skulto"+string(filepath.Separator)+"repositories") {
		return models.ManagementSkulto
	}

	// Check for Vercel management (~/.agents/skills/)
	if strings.Contains(target, ".agents"+string(filepath.Separator)+"skills") {
		return models.ManagementVercel
	}

	// Everything else is external
	return models.ManagementExternal
}

// ScanPlatforms scans multiple platform directories and aggregates results.
func (s *ScannerService) ScanPlatforms(platforms []PlatformConfig, scope string) ([]models.DiscoveredSkill, error) {
	var allDiscovered []models.DiscoveredSkill

	for _, p := range platforms {
		discovered, err := s.ScanDirectory(p.SkillsPath, p.ID, scope)
		if err != nil {
			// Log but continue with other platforms
			continue
		}
		allDiscovered = append(allDiscovered, discovered...)
	}

	return allDiscovered, nil
}
