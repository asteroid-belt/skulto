package installer

import (
	"path/filepath"

	"github.com/asteroid-belt/skulto/internal/config"
)

// PathResolver resolves paths for skill installation.
// Uses the Platform registry as the source of truth for installation paths.
type PathResolver struct {
	cfg *config.Config
}

// NewPathResolver creates a new path resolver.
func NewPathResolver(cfg *config.Config) *PathResolver {
	return &PathResolver{cfg: cfg}
}

// GetGlobalPath returns the target path for a skill in a platform's directory.
// Example: ~/.claude/skills/my-skill/
func (pr *PathResolver) GetGlobalPath(platform Platform, skillSlug string) (string, error) {
	return platform.GetSkillPath(skillSlug)
}

// GetSourcePath returns the source path for a skill in its cloned repository.
// Uses the skill's FilePath to determine the actual directory structure.
// Example: ~/.skulto/repositories/owner/repo/skills/skill-slug/
func (pr *PathResolver) GetSourcePath(owner, repo, skillFilePath string) string {
	// skillFilePath is the path to SKILL.md (e.g., "skills/my-skill/SKILL.md")
	// We need the parent directory (e.g., "skills/my-skill")
	skillDir := filepath.Dir(skillFilePath)

	return filepath.Join(
		pr.cfg.BaseDir,
		"repositories",
		owner,
		repo,
		skillDir,
	)
}

// GetRepositoriesDir returns the base repositories directory.
func (pr *PathResolver) GetRepositoriesDir() string {
	return filepath.Join(pr.cfg.BaseDir, "repositories")
}

// GetSkillsDir returns the skills directory (for backward compatibility).
// This now returns the repositories directory since skills are stored there.
func (pr *PathResolver) GetSkillsDir() string {
	return pr.GetRepositoriesDir()
}

// GetBasePath returns the base skills directory for a platform (without skill slug).
// Example: ~/.claude/skills/
func (pr *PathResolver) GetBasePath(platform Platform) (string, error) {
	// Get any skill path and extract the parent directory
	skillPath, err := platform.GetSkillPath("placeholder")
	if err != nil {
		return "", err
	}
	// Return the parent directory (remove the placeholder skill directory)
	return filepath.Dir(skillPath), nil
}
