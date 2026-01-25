package installer

import (
	"os"
	"path/filepath"
)

// InstallScope represents where skills are installed relative to the filesystem.
type InstallScope string

const (
	// ScopeGlobal installs to the user's home directory (e.g., ~/.claude/skills/)
	ScopeGlobal InstallScope = "global"
	// ScopeProject installs to the current working directory (e.g., ./.claude/skills/)
	ScopeProject InstallScope = "project"
)

// AllScopes returns all valid installation scopes.
func AllScopes() []InstallScope {
	return []InstallScope{ScopeGlobal, ScopeProject}
}

// IsValid checks if the scope is a valid installation scope.
func (s InstallScope) IsValid() bool {
	return s == ScopeGlobal || s == ScopeProject
}

// DisplayName returns a human-readable name for the scope.
func (s InstallScope) DisplayName() string {
	switch s {
	case ScopeGlobal:
		return "Global (Home)"
	case ScopeProject:
		return "Project (CWD)"
	default:
		return string(s)
	}
}

// InstallLocation represents a specific installation target combining platform and scope.
type InstallLocation struct {
	Platform Platform     // The AI platform (claude, cursor, etc.)
	Scope    InstallScope // Global or project scope
	BasePath string       // The resolved base path (home dir or cwd)
}

// NewInstallLocation creates a new install location with resolved base path.
func NewInstallLocation(platform Platform, scope InstallScope) (InstallLocation, error) {
	basePath, err := resolveBasePath(scope)
	if err != nil {
		return InstallLocation{}, err
	}
	return InstallLocation{
		Platform: platform,
		Scope:    scope,
		BasePath: basePath,
	}, nil
}

// GetSkillPath returns the full path to a skill directory for this location.
// Example: ~/.claude/skills/{slug}/ or ./.claude/skills/{slug}/
func (loc InstallLocation) GetSkillPath(skillSlug string) string {
	info := loc.Platform.Info()
	if info.SkillsPath == "" {
		return ""
	}
	return filepath.Join(loc.BasePath, info.SkillsPath, skillSlug)
}

// GetBaseSkillsPath returns the skills directory without the skill slug.
// Example: ~/.claude/skills/ or ./.claude/skills/
func (loc InstallLocation) GetBaseSkillsPath() string {
	info := loc.Platform.Info()
	if info.SkillsPath == "" {
		return ""
	}
	return filepath.Join(loc.BasePath, info.SkillsPath)
}

// String returns a string representation for display.
func (loc InstallLocation) String() string {
	return loc.Platform.Info().Name + " - " + loc.Scope.DisplayName()
}

// ID returns a unique identifier for this location (for map keys).
func (loc InstallLocation) ID() string {
	return string(loc.Platform) + ":" + string(loc.Scope)
}

// resolveBasePath returns the base path for the given scope.
func resolveBasePath(scope InstallScope) (string, error) {
	switch scope {
	case ScopeGlobal:
		return os.UserHomeDir()
	case ScopeProject:
		return os.Getwd()
	default:
		return "", ErrInvalidScope
	}
}
