package installer

import (
	"path/filepath"
	"slices"
)

// Platform represents a supported AI platform for skill installation.
type Platform string

const (
	PlatformClaude      Platform = "claude"
	PlatformCursor      Platform = "cursor"
	PlatformCopilot     Platform = "copilot"
	PlatformCodex       Platform = "codex"
	PlatformOpenCode    Platform = "opencode"
	PlatformWindsurf    Platform = "windsurf"
	PlatformAmp         Platform = "amp"
	PlatformKimiCLI     Platform = "kimi-cli"
	PlatformAntigravity Platform = "antigravity"
	PlatformMoltbot     Platform = "moltbot"
	PlatformCline       Platform = "cline"
	PlatformCodeBuddy   Platform = "codebuddy"
	PlatformCommandCode Platform = "command-code"
	PlatformContinue    Platform = "continue"
	PlatformCrush       Platform = "crush"
	PlatformDroid       Platform = "droid"
	PlatformGeminiCLI   Platform = "gemini-cli"
	PlatformGoose       Platform = "goose"
	PlatformJunie       Platform = "junie"
	PlatformKiloCode    Platform = "kilo"
	PlatformKiroCLI     Platform = "kiro-cli"
	PlatformKode        Platform = "kode"
	PlatformMCPJam      Platform = "mcpjam"
	PlatformMux         Platform = "mux"
	PlatformOpenHands   Platform = "openhands"
	PlatformPi          Platform = "pi"
	PlatformQoder       Platform = "qoder"
	PlatformQwenCode    Platform = "qwen-code"
	PlatformRooCode     Platform = "roo"
	PlatformTrae        Platform = "trae"
	PlatformZencoder    Platform = "zencoder"
	PlatformNeovate     Platform = "neovate"
	PlatformPochi       Platform = "pochi"
)

// PlatformInfo contains information about a supported platform.
type PlatformInfo struct {
	Name       string // Display name (e.g., "Claude Code")
	SkillsPath string // Relative skills directory path (e.g., ".claude/skills")

	// Detection fields
	Command               string   // CLI command name to check in PATH (e.g., "claude")
	ProjectDir            string   // Project-level directory to check (e.g., ".claude")
	GlobalDir             string   // Global/home directory path for detection (e.g., "~/.claude/skills/")
	Aliases               []string // Alternate platform IDs (e.g., kimi-cli shares path with amp)
	PlatformSpecificPaths []string // OS-specific paths to check (e.g., /Applications/Cursor.app)
}

// platformRegistry maps platforms to their metadata.
var platformRegistry = map[Platform]PlatformInfo{
	// --- Original 6 platforms ---
	PlatformClaude: {
		Name:       "Claude Code",
		SkillsPath: ".claude/skills",
		Command:    "claude",
		ProjectDir: ".claude",
		GlobalDir:  "~/.claude/skills/",
	},
	PlatformCursor: {
		Name:       "Cursor",
		SkillsPath: ".cursor/skills",
		Command:    "cursor",
		ProjectDir: ".cursor",
		GlobalDir:  "~/.cursor/skills/",
		PlatformSpecificPaths: []string{
			"/Applications/Cursor.app",
		},
	},
	PlatformCopilot: {
		Name:       "GitHub Copilot",
		SkillsPath: ".github/skills",
		Command:    "",
		ProjectDir: ".github",
		GlobalDir:  "~/.copilot/skills/",
	},
	PlatformCodex: {
		Name:       "OpenAI Codex",
		SkillsPath: ".codex/skills",
		Command:    "codex",
		ProjectDir: ".codex",
		GlobalDir:  "~/.codex/skills/",
	},
	PlatformOpenCode: {
		Name:       "OpenCode",
		SkillsPath: ".opencode/skills",
		Command:    "opencode",
		ProjectDir: ".opencode",
		GlobalDir:  "~/.config/opencode/skills/",
	},
	PlatformWindsurf: {
		Name:       "Windsurf",
		SkillsPath: ".windsurf/skills",
		Command:    "windsurf",
		ProjectDir: ".windsurf",
		GlobalDir:  "~/.codeium/windsurf/skills/",
	},

	// --- New agents ---
	PlatformAmp: {
		Name:       "Amp",
		SkillsPath: ".agents/skills",
		Command:    "amp",
		ProjectDir: ".agents",
		GlobalDir:  "~/.config/agents/skills/",
	},
	PlatformKimiCLI: {
		Name:       "Kimi Code CLI",
		SkillsPath: ".agents/skills",
		Command:    "kimi-cli",
		ProjectDir: ".agents",
		GlobalDir:  "~/.config/agents/skills/",
	},
	PlatformAntigravity: {
		Name:       "Antigravity",
		SkillsPath: ".agent/skills",
		Command:    "antigravity",
		ProjectDir: ".agent",
		GlobalDir:  "~/.gemini/antigravity/global_skills/",
	},
	PlatformMoltbot: {
		Name:       "Moltbot",
		SkillsPath: "skills",
		Command:    "moltbot",
		ProjectDir: "skills",
		GlobalDir:  "~/.moltbot/skills/",
	},
	PlatformCline: {
		Name:       "Cline",
		SkillsPath: ".cline/skills",
		Command:    "cline",
		ProjectDir: ".cline",
		GlobalDir:  "~/.cline/skills/",
	},
	PlatformCodeBuddy: {
		Name:       "CodeBuddy",
		SkillsPath: ".codebuddy/skills",
		Command:    "codebuddy",
		ProjectDir: ".codebuddy",
		GlobalDir:  "~/.codebuddy/skills/",
	},
	PlatformCommandCode: {
		Name:       "Command Code",
		SkillsPath: ".commandcode/skills",
		Command:    "command-code",
		ProjectDir: ".commandcode",
		GlobalDir:  "~/.commandcode/skills/",
	},
	PlatformContinue: {
		Name:       "Continue",
		SkillsPath: ".continue/skills",
		Command:    "continue",
		ProjectDir: ".continue",
		GlobalDir:  "~/.continue/skills/",
	},
	PlatformCrush: {
		Name:       "Crush",
		SkillsPath: ".crush/skills",
		Command:    "crush",
		ProjectDir: ".crush",
		GlobalDir:  "~/.config/crush/skills/",
	},
	PlatformDroid: {
		Name:       "Droid",
		SkillsPath: ".factory/skills",
		Command:    "droid",
		ProjectDir: ".factory",
		GlobalDir:  "~/.factory/skills/",
	},
	PlatformGeminiCLI: {
		Name:       "Gemini CLI",
		SkillsPath: ".gemini/skills",
		Command:    "gemini",
		ProjectDir: ".gemini",
		GlobalDir:  "~/.gemini/skills/",
	},
	PlatformGoose: {
		Name:       "Goose",
		SkillsPath: ".goose/skills",
		Command:    "goose",
		ProjectDir: ".goose",
		GlobalDir:  "~/.config/goose/skills/",
	},
	PlatformJunie: {
		Name:       "Junie",
		SkillsPath: ".junie/skills",
		Command:    "junie",
		ProjectDir: ".junie",
		GlobalDir:  "~/.junie/skills/",
	},
	PlatformKiloCode: {
		Name:       "Kilo Code",
		SkillsPath: ".kilocode/skills",
		Command:    "kilo",
		ProjectDir: ".kilocode",
		GlobalDir:  "~/.kilocode/skills/",
	},
	PlatformKiroCLI: {
		Name:       "Kiro CLI",
		SkillsPath: ".kiro/skills",
		Command:    "kiro-cli",
		ProjectDir: ".kiro",
		GlobalDir:  "~/.kiro/skills/",
	},
	PlatformKode: {
		Name:       "Kode",
		SkillsPath: ".kode/skills",
		Command:    "kode",
		ProjectDir: ".kode",
		GlobalDir:  "~/.kode/skills/",
	},
	PlatformMCPJam: {
		Name:       "MCPJam",
		SkillsPath: ".mcpjam/skills",
		Command:    "mcpjam",
		ProjectDir: ".mcpjam",
		GlobalDir:  "~/.mcpjam/skills/",
	},
	PlatformMux: {
		Name:       "Mux",
		SkillsPath: ".mux/skills",
		Command:    "mux",
		ProjectDir: ".mux",
		GlobalDir:  "~/.mux/skills/",
	},
	PlatformOpenHands: {
		Name:       "OpenHands",
		SkillsPath: ".openhands/skills",
		Command:    "openhands",
		ProjectDir: ".openhands",
		GlobalDir:  "~/.openhands/skills/",
	},
	PlatformPi: {
		Name:       "Pi",
		SkillsPath: ".pi/skills",
		Command:    "pi",
		ProjectDir: ".pi",
		GlobalDir:  "~/.pi/agent/skills/",
	},
	PlatformQoder: {
		Name:       "Qoder",
		SkillsPath: ".qoder/skills",
		Command:    "qoder",
		ProjectDir: ".qoder",
		GlobalDir:  "~/.qoder/skills/",
	},
	PlatformQwenCode: {
		Name:       "Qwen Code",
		SkillsPath: ".qwen/skills",
		Command:    "qwen-code",
		ProjectDir: ".qwen",
		GlobalDir:  "~/.qwen/skills/",
	},
	PlatformRooCode: {
		Name:       "Roo Code",
		SkillsPath: ".roo/skills",
		Command:    "roo",
		ProjectDir: ".roo",
		GlobalDir:  "~/.roo/skills/",
	},
	PlatformTrae: {
		Name:       "Trae",
		SkillsPath: ".trae/skills",
		Command:    "trae",
		ProjectDir: ".trae",
		GlobalDir:  "~/.trae/skills/",
	},
	PlatformZencoder: {
		Name:       "Zencoder",
		SkillsPath: ".zencoder/skills",
		Command:    "zencoder",
		ProjectDir: ".zencoder",
		GlobalDir:  "~/.zencoder/skills/",
	},
	PlatformNeovate: {
		Name:       "Neovate",
		SkillsPath: ".neovate/skills",
		Command:    "neovate",
		ProjectDir: ".neovate",
		GlobalDir:  "~/.neovate/skills/",
	},
	PlatformPochi: {
		Name:       "Pochi",
		SkillsPath: ".pochi/skills",
		Command:    "pochi",
		ProjectDir: ".pochi",
		GlobalDir:  "~/.pochi/skills/",
	},
}

// AllPlatforms returns a slice of all supported platforms in display order.
func AllPlatforms() []Platform {
	return []Platform{
		PlatformClaude,
		PlatformCursor,
		PlatformCopilot,
		PlatformCodex,
		PlatformOpenCode,
		PlatformWindsurf,
		PlatformAmp,
		PlatformKimiCLI,
		PlatformAntigravity,
		PlatformMoltbot,
		PlatformCline,
		PlatformCodeBuddy,
		PlatformCommandCode,
		PlatformContinue,
		PlatformCrush,
		PlatformDroid,
		PlatformGeminiCLI,
		PlatformGoose,
		PlatformJunie,
		PlatformKiloCode,
		PlatformKiroCLI,
		PlatformKode,
		PlatformMCPJam,
		PlatformMux,
		PlatformOpenHands,
		PlatformPi,
		PlatformQoder,
		PlatformQwenCode,
		PlatformRooCode,
		PlatformTrae,
		PlatformZencoder,
		PlatformNeovate,
		PlatformPochi,
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

// IsValidAlias checks if a string is a valid platform alias.
func IsValidAlias(s string) bool {
	for _, info := range platformRegistry {
		if slices.Contains(info.Aliases, s) {
			return true
		}
	}
	return false
}

// PlatformFromStringOrAlias converts a string or alias to a Platform.
// Checks direct platform ID first, then aliases.
func PlatformFromStringOrAlias(s string) Platform {
	// Direct match first
	if p := PlatformFromString(s); p != "" {
		return p
	}
	// Check aliases
	for platform, info := range platformRegistry {
		if slices.Contains(info.Aliases, s) {
			return platform
		}
	}
	return ""
}
