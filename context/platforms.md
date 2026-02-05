# Platform Registry Deep Dive

> This document details the platform support system in Skulto.

## Overview

Skulto supports 33 AI coding platforms through a data-driven registry defined in `internal/installer/platform.go`.

## Platform Type

```go
type Platform string

const (
    PlatformClaude      Platform = "claude"
    PlatformCursor      Platform = "cursor"
    PlatformCopilot     Platform = "copilot"
    PlatformCodex       Platform = "codex"
    PlatformOpenCode    Platform = "opencode"
    PlatformWindsurf    Platform = "windsurf"
    // ... 27 more platforms
)
```

## Platform Info Structure

Each platform defines metadata for detection and installation:

```go
type PlatformInfo struct {
    Name       string   // Display name (e.g., "Claude Code")
    SkillsPath string   // Relative skills directory (e.g., ".claude/skills")

    // Detection fields
    Command               string   // CLI command to check in PATH
    ProjectDir            string   // Project-level directory to check
    GlobalDir             string   // Global/home directory path
    Aliases               []string // Alternate platform IDs
    PlatformSpecificPaths []string // OS-specific paths (e.g., /Applications/Cursor.app)
}
```

## Complete Platform Registry

### Original 6 Platforms

| Platform | Name | Skills Path | Command | Global Dir |
|----------|------|-------------|---------|------------|
| `claude` | Claude Code | `.claude/skills` | `claude` | `~/.claude/skills/` |
| `cursor` | Cursor | `.cursor/skills` | `cursor` | `~/.cursor/skills/` |
| `copilot` | GitHub Copilot | `.github/skills` | - | `~/.copilot/skills/` |
| `codex` | OpenAI Codex | `.codex/skills` | `codex` | `~/.codex/skills/` |
| `opencode` | OpenCode | `.opencode/skills` | `opencode` | `~/.config/opencode/skills/` |
| `windsurf` | Windsurf | `.windsurf/skills` | `windsurf` | `~/.codeium/windsurf/skills/` |

### Extended Platforms

| Platform | Name | Skills Path | Command | Global Dir |
|----------|------|-------------|---------|------------|
| `amp` | Amp | `.agents/skills` | `amp` | `~/.config/agents/skills/` |
| `kimi-cli` | Kimi Code CLI | `.agents/skills` | `kimi-cli` | `~/.config/agents/skills/` |
| `antigravity` | Antigravity | `.agent/skills` | `antigravity` | `~/.gemini/antigravity/global_skills/` |
| `moltbot` | Moltbot | `skills` | `moltbot` | `~/.moltbot/skills/` |
| `cline` | Cline | `.cline/skills` | `cline` | `~/.cline/skills/` |
| `codebuddy` | CodeBuddy | `.codebuddy/skills` | `codebuddy` | `~/.codebuddy/skills/` |
| `command-code` | Command Code | `.commandcode/skills` | `command-code` | `~/.commandcode/skills/` |
| `continue` | Continue | `.continue/skills` | `continue` | `~/.continue/skills/` |
| `crush` | Crush | `.crush/skills` | `crush` | `~/.config/crush/skills/` |
| `droid` | Droid | `.factory/skills` | `droid` | `~/.factory/skills/` |
| `gemini-cli` | Gemini CLI | `.gemini/skills` | `gemini` | `~/.gemini/skills/` |
| `goose` | Goose | `.goose/skills` | `goose` | `~/.config/goose/skills/` |
| `junie` | Junie | `.junie/skills` | `junie` | `~/.junie/skills/` |
| `kilo` | Kilo Code | `.kilocode/skills` | `kilo` | `~/.kilocode/skills/` |
| `kiro-cli` | Kiro CLI | `.kiro/skills` | `kiro-cli` | `~/.kiro/skills/` |
| `kode` | Kode | `.kode/skills` | `kode` | `~/.kode/skills/` |
| `mcpjam` | MCPJam | `.mcpjam/skills` | `mcpjam` | `~/.mcpjam/skills/` |
| `mux` | Mux | `.mux/skills` | `mux` | `~/.mux/skills/` |
| `openhands` | OpenHands | `.openhands/skills` | `openhands` | `~/.openhands/skills/` |
| `pi` | Pi | `.pi/skills` | `pi` | `~/.pi/agent/skills/` |
| `qoder` | Qoder | `.qoder/skills` | `qoder` | `~/.qoder/skills/` |
| `qwen-code` | Qwen Code | `.qwen/skills` | `qwen-code` | `~/.qwen/skills/` |
| `roo` | Roo Code | `.roo/skills` | `roo` | `~/.roo/skills/` |
| `trae` | Trae | `.trae/skills` | `trae` | `~/.trae/skills/` |
| `zencoder` | Zencoder | `.zencoder/skills` | `zencoder` | `~/.zencoder/skills/` |
| `neovate` | Neovate | `.neovate/skills` | `neovate` | `~/.neovate/skills/` |
| `pochi` | Pochi | `.pochi/skills` | `pochi` | `~/.pochi/skills/` |

## Platform Detection

Detection is performed by `isPlatformDetected()` in `internal/installer/service.go`:

```go
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
        if _, err := os.Stat(path); err == nil {
            return true
        }
    }

    return false
}
```

### Detection Priority

1. **Command in PATH** - Most reliable indicator the tool is installed
2. **Project-level directory** - Indicates tool was used in current project
3. **Global directory** - Indicates user has used the tool previously
4. **Platform-specific paths** - OS-specific app locations (e.g., `/Applications/Cursor.app`)

## Installation Scopes

Skills can be installed at two scopes:

```go
type InstallScope string

const (
    ScopeGlobal  InstallScope = "global"   // User-wide: ~/
    ScopeProject InstallScope = "project"  // Project-local: ./
)
```

### Path Resolution

```go
func resolveBasePath(scope InstallScope) (string, error) {
    switch scope {
    case ScopeGlobal:
        return os.UserHomeDir()
    case ScopeProject:
        return os.Getwd()
    default:
        return "", fmt.Errorf("invalid scope: %s", scope)
    }
}
```

### Example Installation Paths

For skill `superplan` on Claude:

| Scope | Full Path |
|-------|-----------|
| Global | `~/.claude/skills/superplan` -> `~/.skulto/repositories/asteroid-belt/skills/skills/superplan/` |
| Project | `./.claude/skills/superplan` -> `~/.skulto/repositories/asteroid-belt/skills/skills/superplan/` |

## InstallLocation

An `InstallLocation` combines platform + scope + resolved path:

```go
type InstallLocation struct {
    Platform Platform
    Scope    InstallScope
    Path     string  // Resolved full path
}

func NewInstallLocation(platform Platform, scope InstallScope) (InstallLocation, error) {
    basePath, err := resolveBasePath(scope)
    if err != nil {
        return InstallLocation{}, err
    }

    skillsPath := platform.Info().SkillsPath
    fullPath := filepath.Join(basePath, skillsPath)

    return InstallLocation{
        Platform: platform,
        Scope:    scope,
        Path:     fullPath,
    }, nil
}
```

## Symlink Installation

Skills are installed as directory symlinks:

```go
func (i *Installer) createSymlink(skillDir, targetPath string) error {
    // Ensure parent directory exists
    if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
        return err
    }

    // Create symlink: targetPath -> skillDir
    return os.Symlink(skillDir, targetPath)
}
```

### Symlink Source Resolution

For repository skills:
```go
sourceDir = filepath.Join(
    cfg.BaseDir,                    // ~/.skulto
    "repositories",
    source.Owner,                   // asteroid-belt
    source.Repo,                    // skills
    filepath.Dir(skill.FilePath),   // skills/superplan
)
```

For local skills:
```go
sourceDir = filepath.Dir(skill.FilePath)  // ~/.skulto/skills/my-skill
```

## Database Tracking

Installations are tracked in `skill_installations` table:

```sql
CREATE TABLE skill_installations (
    id         TEXT PRIMARY KEY,
    skill_id   TEXT NOT NULL,
    platform   TEXT NOT NULL,
    scope      TEXT NOT NULL,
    path       TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (skill_id) REFERENCES skills(id)
);
```

### Recording Installations

```go
func (db *DB) RecordInstallation(skillID string, loc InstallLocation) error {
    inst := &models.SkillInstallation{
        ID:       generateInstallID(skillID, loc),
        SkillID:  skillID,
        Platform: string(loc.Platform),
        Scope:    string(loc.Scope),
        Path:     loc.Path,
    }
    return db.Create(inst).Error
}
```

## Adding New Platforms

To add a new platform:

1. **Add constant** in `platform.go`:
```go
PlatformNewTool Platform = "newtool"
```

2. **Add registry entry**:
```go
PlatformNewTool: {
    Name:       "New Tool",
    SkillsPath: ".newtool/skills",
    Command:    "newtool",
    ProjectDir: ".newtool",
    GlobalDir:  "~/.newtool/skills/",
},
```

3. **Add to AllPlatforms()**:
```go
func AllPlatforms() []Platform {
    return []Platform{
        // ... existing platforms
        PlatformNewTool,
    }
}
```

4. **Update README.md** platform count and table

5. **Add tests** in `platform_test.go`

## Platform Aliases

Some platforms share skill paths (e.g., Amp and Kimi CLI both use `.agents/skills`):

```go
PlatformAmp: {
    Name:       "Amp",
    SkillsPath: ".agents/skills",
    Aliases:    []string{"kimi-cli"},  // Optional: for URL parsing
},
```

Aliases are resolved by `PlatformFromStringOrAlias()`:

```go
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
```
