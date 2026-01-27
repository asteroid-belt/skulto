package installer

import (
	"path/filepath"
)

// Platform represents a supported AI platform for skill installation.
type Platform string

const (
	PlatformClaude   Platform = "claude"
	PlatformCursor   Platform = "cursor"
	PlatformCopilot  Platform = "copilot"
	PlatformCodex    Platform = "codex"
	PlatformOpenCode Platform = "opencode"
	PlatformWindsurf Platform = "windsurf"
)

// PlatformInfo contains information about a supported platform.
type PlatformInfo struct {
	Name       string // Display name (e.g., "Claude Code")
	SkillsPath string // Relative skills directory path (e.g., ".claude/skills")
}

// platformRegistry maps platforms to their metadata.
var platformRegistry = map[Platform]PlatformInfo{
	PlatformClaude: {
		Name:       "Claude Code",
		SkillsPath: ".claude/skills",
	},
	PlatformCursor: {
		Name:       "Cursor",
		SkillsPath: ".cursor/skills",
	},
	PlatformCopilot: {
		Name:       "GitHub Copilot",
		SkillsPath: ".github/skills",
	},
	PlatformCodex: {
		Name:       "OpenAI Codex",
		SkillsPath: ".codex/skills",
	},
	PlatformOpenCode: {
		Name:       "OpenCode",
		SkillsPath: ".opencode/skills",
	},
	PlatformWindsurf: {
		Name:       "Windsurf",
		SkillsPath: ".windsurf/skills",
	},
}

// AllPlatforms returns a slice of all supported platforms.
func AllPlatforms() []Platform {
	return []Platform{
		PlatformClaude,
		PlatformCursor,
		PlatformCopilot,
		PlatformCodex,
		PlatformOpenCode,
		PlatformWindsurf,
	}
}

// IsValid checks if the platform is a supported platform.
func (p Platform) IsValid() bool {
	_, exists := platformRegistry[p]
	return exists
}

// Info returns the platform info for this platform.
// Returns an empty PlatformInfo if the platform is not valid.
func (p Platform) Info() PlatformInfo {
	if info, exists := platformRegistry[p]; exists {
		return info
	}
	return PlatformInfo{}
}

// GetSkillPath returns the full path to a skill directory for this platform.
// Example: ~/.claude/skills/{slug}/ (uses global scope by default)
// Deprecated: Use GetSkillPathForScope instead for explicit scope control.
func (p Platform) GetSkillPath(skillSlug string) (string, error) {
	return p.GetSkillPathForScope(skillSlug, ScopeGlobal)
}

// GetSkillPathForScope returns the full path to a skill directory for this platform and scope.
// Example: ~/.claude/skills/{slug}/ for global, ./.claude/skills/{slug}/ for project
func (p Platform) GetSkillPathForScope(skillSlug string, scope InstallScope) (string, error) {
	info := p.Info()
	if info.SkillsPath == "" {
		return "", nil
	}

	basePath, err := resolveBasePath(scope)
	if err != nil {
		return "", err
	}

	return filepath.Join(basePath, info.SkillsPath, skillSlug), nil
}

// PlatformFromString converts a string to a Platform.
// Returns an empty Platform if the string is not a valid platform.
func PlatformFromString(s string) Platform {
	p := Platform(s)
	if p.IsValid() {
		return p
	}
	return ""
}
