# Skulto Architecture Deep Dive

> This document provides detailed technical context for the Skulto architecture.

## System Overview

Skulto is a cross-platform AI skills management tool with three user interfaces sharing common services.

```
                              ┌─────────────────────────────────────┐
                              │           User Interfaces           │
                              ├─────────────┬─────────────┬─────────┤
                              │   CLI       │    TUI      │   MCP   │
                              │  (Cobra)    │ (Bubble Tea)│(mcp-go) │
                              └──────┬──────┴──────┬──────┴────┬────┘
                                     │             │           │
                              ┌──────▼─────────────▼───────────▼────┐
                              │         Service Layer               │
                              ├─────────────────────────────────────┤
                              │  InstallService  │  SearchService   │
                              │  Scraper         │  SecurityScanner │
                              │  FavoritesStore  │  Telemetry       │
                              └──────────────────┬──────────────────┘
                                                 │
                              ┌──────────────────▼──────────────────┐
                              │         Data Layer                  │
                              ├─────────────────────────────────────┤
                              │  SQLite + GORM  │  FTS5 Search      │
                              │  Vector Store   │  File System      │
                              └─────────────────────────────────────┘
```

## Entry Points

### CLI/TUI Entry (`cmd/skulto/main.go`)

```go
func main() {
    // 1. Create context with signal handling
    ctx, cancel := context.WithCancel(context.Background())

    // 2. Load config and open database
    cfg, _ := config.Load()
    paths := config.GetPaths(cfg)
    database, _ := db.New(db.DefaultConfig(paths.Database))

    // 3. Initialize telemetry with persistent tracking ID
    telemetryClient := telemetry.New(database)

    // 4. Execute CLI (may launch TUI if no subcommand)
    cli.Execute(ctx, telemetryClient)
}
```

The CLI uses Cobra with Fang enhancements. If no subcommand is provided, `runTUI` launches the Bubble Tea application.

### MCP Server Entry (`cmd/skulto-mcp/main.go`)

```go
func main() {
    // 1. Load config and database
    cfg, _ := config.Load()
    database, _ := db.New(db.DefaultConfig(paths.Database))

    // 2. Initialize favorites and telemetry
    favStore := favorites.NewStore(paths.Favorites)
    tc := telemetry.New(database)

    // 3. Create and serve MCP server
    server := mcp.NewServer(database, cfg, favStore, tc)
    server.Serve(ctx)  // JSON-RPC 2.0 over stdio
}
```

## Core Services

### InstallService (`internal/installer/service.go`)

The `InstallService` is the unified entry point for skill installation across all interfaces.

**Key responsibilities:**
- Platform detection (command in PATH, directory existence)
- Skill lookup by slug
- Symlink creation/removal
- Telemetry tracking

**Important methods:**
```go
type InstallService struct {
    installer *Installer      // Low-level symlink operations
    db        *db.DB
    cfg       *config.Config
    telemetry telemetry.Client
}

// Install a skill to specified locations
func (s *InstallService) Install(ctx, slug string, opts InstallOptions) (*InstallResult, error)

// Uninstall from specific locations
func (s *InstallService) Uninstall(ctx, slug string, locations []InstallLocation) error

// Get all locations where a skill is installed
func (s *InstallService) GetInstallLocations(ctx, slug string) ([]InstallLocation, error)

// Get summary of all installed skills
func (s *InstallService) GetInstalledSkillsSummary(ctx) ([]InstalledSkillSummary, error)
```

**Installation flow:**
1. Look up skill by slug in database
2. Determine target platforms (from options or user preferences)
3. Build `InstallLocation` list (platform + scope combinations)
4. Call `Installer.InstallTo()` for symlink creation
5. Record installations in `skill_installations` table
6. Track telemetry event

### Scraper (`internal/scraper/`)

The scraper manages GitHub repositories using shallow git clones.

**Key components:**

| File | Purpose |
|------|---------|
| `repository.go` | `RepositoryManager` - git clone/fetch operations |
| `github.go` | `Scraper` - high-level scraping orchestration |
| `parser.go` | `SkillParser` - SKILL.md/CLAUDE.md parsing with frontmatter |

**Repository caching:**
- Repos cloned to `~/.skulto/repositories/{owner}/{repo}/`
- Shallow clones (depth=1) for efficiency
- `RecentUpdateTTL` (60s) prevents redundant fetches
- Per-repo mutex locks for concurrent access

**Skill file detection:**
```go
// IsSkillFilePath matches skill.md or claude.md at any level
func IsSkillFilePath(path string) bool {
    lowerPath := strings.ToLower(path)
    return strings.HasSuffix(lowerPath, "/skill.md") ||
           lowerPath == "skill.md" ||
           strings.HasSuffix(lowerPath, "/claude.md") ||
           lowerPath == "claude.md"
}
```

### Search Service (`internal/search/service.go`)

Hybrid search combining FTS5 and semantic (vector) search.

```go
type Service struct {
    db     *db.DB
    store  vector.VectorStore  // Optional, nil if semantic search disabled
    config Config
}

// Search performs hybrid search with graceful degradation
func (s *Service) Search(ctx, query string, opts SearchOptions) (*SearchResults, error) {
    // 1. Semantic search (if enabled and vector store available)
    if opts.IncludeSemantic && s.store != nil {
        hits, _ := s.semanticSearch(ctx, query, opts)
        s.categorizeResults(ctx, hits, results, query, seen)
    }

    // 2. FTS5 search
    if opts.IncludeFTS {
        ftsResults, _ := s.db.SearchSkills(query, opts.Limit)
        s.mergeWithFTS(ftsResults, results, query, seen)
    }

    return results, nil
}
```

**Search result categories:**
- `TitleMatches` - Skills where query matches title
- `ContentMatches` - Skills where query matches content only

### Security Scanner (`internal/security/`)

Pattern-based detection of prompt injection threats.

**Scanning flow:**
1. `Scanner.ScanAndClassify(skill)` - main entry point
2. Match content against `PromptInjectionPatterns`
3. Score matches using `Scorer.ScoreMatches()`
4. Apply context mitigation (educational content, etc.)
5. Classify threat level: NONE < LOW < MEDIUM < HIGH < CRITICAL

**Severity weights:**
```go
func SeverityWeight(s ThreatLevel) int {
    switch s {
    case ThreatLevelCritical: return 10
    case ThreatLevelHigh:     return 5
    case ThreatLevelMedium:   return 2
    case ThreatLevelLow:      return 1
    default:                  return 0
    }
}
```

## TUI Architecture (`internal/tui/`)

The TUI uses Bubble Tea's Elm-inspired architecture.

### Model Structure

```go
type Model struct {
    db             *db.DB
    cfg            *config.Config
    telemetry      telemetry.Client
    installer      *installer.Installer
    installService *installer.InstallService
    searchSvc      *search.Service
    favorites      *favorites.Store

    // Views
    currentView   ViewType
    previousView  ViewType
    homeView      *views.HomeView
    searchView    *views.SearchView
    detailView    *views.DetailView
    // ... more views

    // Dialogs (overlay components)
    locationDialog       *components.InstallLocationDialog
    showLocationDialog   bool
    newSkillDialog       *components.NewSkillDialog
    showingNewSkillDialog bool

    // Session tracking
    sessionStart      time.Time
    viewsVisited      int
    searchesPerformed int
}
```

### View Navigation

Views are identified by `ViewType` enum:

```go
const (
    ViewHome ViewType = iota
    ViewSearch
    ViewSkillDetail
    ViewTag
    ViewOnboardingIntro
    ViewOnboardingSetup
    ViewOnboardingTools
    ViewOnboardingSkills
    ViewAddSource
    ViewHelp
    ViewSettings
    ViewManage
)
```

Navigation is tracked for telemetry:
```go
func (m *Model) trackViewNavigation(toView ViewType) {
    m.telemetry.TrackViewNavigated(toView.String(), m.currentView.String())
    m.viewsVisited++
}
```

### Message Flow

Bubble Tea messages flow through `Update()`:

```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Handle keyboard input per view
    case tea.WindowSizeMsg:
        // Propagate size to all views
    case pullCompleteMsg:
        // Handle async sync completion
    case views.SkillInstalledMsg:
        // Handle install/uninstall completion
    }
}
```

### Async Operations Pattern

Long-running operations return `tea.Cmd`:

```go
// syncCmd returns a command that syncs repositories
func (m *Model) syncCmd(githubToken string) tea.Cmd {
    return func() tea.Msg {
        // Do async work...
        return pullCompleteMsg{
            skillsFound: count,
            err:         nil,
        }
    }
}
```

## MCP Server Architecture (`internal/mcp/`)

The MCP server exposes Skulto functionality via JSON-RPC 2.0 over stdio.

### Server Structure

```go
type Server struct {
    db             *db.DB
    cfg            *config.Config
    installer      *Installer        // Same installer used by TUI
    installService *InstallService   // Unified service
    favorites      *favorites.Store
    server         *server.MCPServer // mcp-go server
    telemetry      telemetry.Client
}
```

### Tool Registration

Tools are registered in `registerTools()`:

```go
func (s *Server) registerTools() {
    // Core tools
    s.server.AddTool(searchTool(), s.handleSearch)
    s.server.AddTool(getSkillTool(), s.handleGetSkill)
    s.server.AddTool(listSkillsTool(), s.handleListSkills)

    // Browse tools
    s.server.AddTool(browseTagsTool(), s.handleBrowseTags)
    s.server.AddTool(getStatsTool(), s.handleGetStats)

    // User state tools
    s.server.AddTool(installTool(), s.handleInstall)
    s.server.AddTool(uninstallTool(), s.handleUninstall)
    s.server.AddTool(favoriteTool(), s.handleFavorite)

    // Repository management
    s.server.AddTool(addTool(), s.handleAdd)
}
```

### Resource Templates

MCP resources provide direct skill access:

```go
func (s *Server) registerResources() {
    // Skill content as markdown
    s.server.AddResourceTemplate(
        mcp.NewResourceTemplate(
            "skulto://skill/{slug}",
            "Skill content",
            mcp.WithTemplateMIMEType("text/markdown"),
        ),
        s.handleSkillContentResource,
    )

    // Skill metadata as JSON
    s.server.AddResourceTemplate(
        mcp.NewResourceTemplate(
            "skulto://skill/{slug}/metadata",
            "Skill metadata",
            mcp.WithTemplateMIMEType("application/json"),
        ),
        s.handleSkillMetadataResource,
    )
}
```

## Data Flow Examples

### Installing a Skill (TUI)

```
1. User presses 'i' on skill detail view
2. DetailView.Update() sets IsInstalling=true
3. Model shows InstallLocationDialog
4. User selects platforms and scopes, confirms
5. Model calls installToLocationsCmd()
6. Installer.InstallTo() creates symlinks
7. DB records installations in skill_installations
8. Telemetry tracks skill_installed event
9. views.SkillInstalledMsg sent back to Update()
10. DetailView updates UI to show installed state
```

### Syncing Repositories (TUI)

```
1. User presses 'p' on home view
2. HomeView.SetPulling(true)
3. Model calls syncCmd() and watchPullProgressCmd()
4. Scraper.ScrapeSeedsWithOptions() runs in goroutine
5. Progress sent to pullProgressCh channel
6. pullProgressMsg updates HomeView progress bar
7. Security scanner runs on pending skills
8. scanProgressMsg updates progress
9. pullCompleteMsg triggers HomeView refresh
```

### MCP Tool Invocation

```
1. Claude Code sends JSON-RPC request over stdin
2. mcp-go server routes to handleInstall
3. Handler parses slug and options from request
4. InstallService.Install() called
5. Telemetry tracks mcp_tool_called event
6. Result marshaled and sent to stdout
```

## Key Design Decisions

### Symlink-Based Installation

Skills are installed as symlinks pointing to cloned repository files:

```
~/.claude/skills/superplan -> ~/.skulto/repositories/asteroid-belt/skills/skills/superplan/
```

**Benefits:**
- Skills automatically update when repository is pulled
- No file duplication
- Easy uninstall (remove symlink)

**Tradeoffs:**
- Requires working tree repos (not bare clones)
- Symlink may break if repo is deleted

### FTS5 for Full-Text Search

SQLite FTS5 with BM25 ranking provides fast, relevant search:

```sql
SELECT s.*, bm25(skills_fts, 10.0, 5.0, 1.0, 2.0, 3.0) as rank
FROM skills s
JOIN skills_fts fts ON s.rowid = fts.rowid
WHERE skills_fts MATCH ?
ORDER BY rank
```

BM25 weights: title(10), description(5), content(1), summary(2), tags(3)

### Telemetry Consistency

All interfaces track the same events through the `telemetry.Client` interface:

```go
type Client interface {
    // Shared events (all interfaces)
    TrackSkillViewed(slug, category string, isLocal bool)
    TrackSkillInstalled(skillName, category string, isLocal bool, platformCount int)

    // MCP-specific
    TrackMCPToolCalled(toolName string, durationMs int64, success bool)
}
```

This ensures analytics accurately reflect user behavior regardless of which interface they use.
