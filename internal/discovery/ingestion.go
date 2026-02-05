// internal/discovery/ingestion.go
package discovery

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/scraper"
)

// IngestionResult contains the result of ingesting a skill.
type IngestionResult struct {
	Name     string
	OrigPath string
	DestPath string
	Skill    *models.Skill // The created skill record
}

// IngestionService handles copying discovered skills to skulto management.
type IngestionService struct {
	db              *db.DB
	cfg             *config.Config
	destDirOverride string // For testing
}

// NewIngestionService creates a new ingestion service.
func NewIngestionService(database *db.DB, cfg *config.Config) *IngestionService {
	return &IngestionService{
		db:  database,
		cfg: cfg,
	}
}

// ValidateSkill checks if a skill directory is valid (has skill.md or SKILL.md).
func (s *IngestionService) ValidateSkill(path string) error {
	// Check for skill.md
	skillMdPath := filepath.Join(path, "skill.md")
	if _, err := os.Stat(skillMdPath); err == nil {
		return nil
	}

	// Check for SKILL.md
	skillMdUpperPath := filepath.Join(path, "SKILL.md")
	if _, err := os.Stat(skillMdUpperPath); err == nil {
		return nil
	}

	return fmt.Errorf("cannot import: no skill.md found in %s", path)
}

// CheckNameConflict checks if a skill name already exists in skulto.
func (s *IngestionService) CheckNameConflict(name, scope string) (bool, error) {
	destDir := s.getDestDir(scope)
	destPath := filepath.Join(destDir, name)

	_, err := os.Stat(destPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// IngestSkill copies a discovered skill to skulto and creates a symlink.
func (s *IngestionService) IngestSkill(ctx context.Context, skill *models.DiscoveredSkill) (*IngestionResult, error) {
	// Validate skill
	if err := s.ValidateSkill(skill.Path); err != nil {
		return nil, err
	}

	destDir := s.getDestDir(skill.Scope)
	destPath := filepath.Join(destDir, skill.Name)

	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Copy skill directory to destination
	if err := copyDir(skill.Path, destPath); err != nil {
		return nil, fmt.Errorf("failed to copy skill: %w", err)
	}

	// Remove original directory
	if err := os.RemoveAll(skill.Path); err != nil {
		// Try to rollback
		_ = os.RemoveAll(destPath)
		return nil, fmt.Errorf("failed to remove original directory: %w", err)
	}

	// Create symlink from original location to new location
	relPath, err := filepath.Rel(filepath.Dir(skill.Path), destPath)
	if err != nil {
		relPath = destPath // Fall back to absolute path
	}

	if err := os.Symlink(relPath, skill.Path); err != nil {
		return nil, fmt.Errorf("failed to create symlink: %w", err)
	}

	// Remove from discovered_skills table
	if s.db != nil {
		_ = s.db.DeleteDiscoveredSkill(skill.ID)
	}

	// Parse skill.md and create database records
	var parsedSkill *models.Skill
	if s.db != nil {
		var err error
		parsedSkill, err = s.parseAndCreateSkillRecord(destPath, skill)
		if err != nil {
			return nil, fmt.Errorf("failed to create skill record: %w", err)
		}
	}

	return &IngestionResult{
		Name:     skill.Name,
		OrigPath: skill.Path,
		DestPath: destPath,
		Skill:    parsedSkill,
	}, nil
}

func (s *IngestionService) getDestDir(scope string) string {
	if s.destDirOverride != "" {
		return s.destDirOverride
	}

	if s.cfg == nil {
		return ".skulto/skills"
	}

	// Use config paths based on scope
	paths := config.GetPaths(s.cfg)
	if scope == "global" {
		return paths.Skills
	}
	// For project scope, use current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return ".skulto/skills"
	}
	return filepath.Join(cwd, ".skulto", "skills")
}

// copyDir recursively copies a directory.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

// copyFile copies a single file.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// parseAndCreateSkillRecord parses the skill.md file and creates database records.
func (s *IngestionService) parseAndCreateSkillRecord(destPath string, discoveredSkill *models.DiscoveredSkill) (*models.Skill, error) {
	// Read skill.md content (try both lowercase and uppercase)
	skillMdPath := filepath.Join(destPath, "skill.md")
	content, err := os.ReadFile(skillMdPath)
	if err != nil {
		// Try SKILL.md
		skillMdPath = filepath.Join(destPath, "SKILL.md")
		content, err = os.ReadFile(skillMdPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read skill.md: %w", err)
		}
	}

	// Parse skill using the scraper parser
	// Use "local-" + name format to match startup sync (syncLocalSkillsCmd)
	// This ensures skills aren't duplicated after DB reset
	parser := scraper.NewSkillParser()
	skillFile := &scraper.SkillFile{
		ID:   "local-" + discoveredSkill.Name,
		Path: destPath,
		// Don't set RepoName - local skills have no source
	}
	parsedSkill, err := parser.Parse(string(content), skillFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse skill: %w", err)
	}

	// Set local skill flags and clear SourceID (local skills have no source)
	parsedSkill.IsLocal = true
	parsedSkill.IsInstalled = true
	parsedSkill.FilePath = destPath
	parsedSkill.SourceID = nil // Local skills have no source

	// Extract tags from content
	tags := scraper.ExtractTagsWithContext(parsedSkill.Title, parsedSkill.Description, string(content))

	// Upsert skill with tags
	if err := s.db.UpsertSkillWithTags(parsedSkill, tags); err != nil {
		return nil, fmt.Errorf("failed to save skill: %w", err)
	}

	// Create SkillInstallation record
	installation := &models.SkillInstallation{
		SkillID:     parsedSkill.ID,
		Platform:    discoveredSkill.Platform,
		Scope:       discoveredSkill.Scope,
		BasePath:    destPath,
		SymlinkPath: discoveredSkill.Path, // Original path (now a symlink)
	}
	installation.ID = installation.GenerateID()

	if err := s.db.AddInstallation(installation); err != nil {
		return nil, fmt.Errorf("failed to create installation: %w", err)
	}

	// Add to installed table so it appears on home page
	if err := s.db.SetInstalled(parsedSkill.ID, true); err != nil {
		return nil, fmt.Errorf("failed to mark as installed: %w", err)
	}

	return parsedSkill, nil
}
