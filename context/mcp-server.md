# MCP Server Deep Dive

> This document details the Model Context Protocol (MCP) server implementation in Skulto.

## Overview

The MCP server (`skulto-mcp`) exposes Skulto functionality to Claude Code and other MCP-compatible clients via JSON-RPC 2.0 over stdio.

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                    MCP Client (Claude Code)                       │
└───────────────────────────┬──────────────────────────────────────┘
                            │ JSON-RPC 2.0 over stdio
┌───────────────────────────▼──────────────────────────────────────┐
│                       skulto-mcp                                  │
├──────────────────────────────────────────────────────────────────┤
│  ┌────────────────┐  ┌─────────────────┐  ┌──────────────────┐  │
│  │  Tool Handlers │  │ Resource Handlers│  │   Telemetry      │  │
│  │                │  │                  │  │                  │  │
│  │ - handleSearch │  │ - skill content  │  │ TrackMCPToolCalled│
│  │ - handleInstall│  │ - skill metadata │  │                  │  │
│  │ - handleAdd    │  │                  │  │                  │  │
│  └───────┬────────┘  └────────┬─────────┘  └──────────────────┘  │
│          │                    │                                   │
│  ┌───────▼────────────────────▼───────────────────────────────┐  │
│  │                    Shared Services                          │  │
│  │  InstallService  │  Database  │  Scraper  │  FavoritesStore │  │
│  └─────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

## Configuration

Add to Claude Code settings (`~/.claude.json` or `.mcp.json`):

```json
{
  "mcpServers": {
    "skulto": {
      "type": "stdio",
      "command": "/opt/homebrew/bin/skulto-mcp"
    }
  }
}
```

## Server Structure

```go
type Server struct {
    db             *db.DB
    cfg            *config.Config
    installer      *installer.Installer      // Low-level symlink operations
    installService *installer.InstallService // Unified install with telemetry
    favorites      *favorites.Store
    server         *server.MCPServer         // mcp-go server instance
    telemetry      telemetry.Client
}
```

## Tools

### Core Tools (Phase 1A)

#### `skulto_search`

Full-text search with BM25 ranking.

```go
func searchTool() mcp.Tool {
    return mcp.NewTool("skulto_search",
        mcp.WithDescription("Search skills using full-text search with BM25 ranking"),
        mcp.WithString("query",
            mcp.Required(),
            mcp.Description("Search query - supports partial matching"),
        ),
        mcp.WithNumber("limit",
            mcp.Description("Max results (default 20, max 100)"),
        ),
    )
}

func (s *Server) handleSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    start := time.Now()

    query := request.Params.Arguments["query"].(string)
    limit := 20
    if l, ok := request.Params.Arguments["limit"].(float64); ok {
        limit = min(int(l), 100)
    }

    skills, err := s.db.SearchSkills(query, limit)
    if err != nil {
        s.telemetry.TrackMCPToolCalled("skulto_search", time.Since(start).Milliseconds(), false)
        return mcp.NewToolResultError(err.Error()), nil
    }

    // Track telemetry
    s.telemetry.TrackMCPToolCalled("skulto_search", time.Since(start).Milliseconds(), true)
    s.telemetry.TrackSkillsListed(len(skills), "mcp")

    // Format results as JSON
    result := formatSearchResults(skills)
    return mcp.NewToolResultText(result), nil
}
```

#### `skulto_get_skill`

Get detailed skill information.

```go
func getSkillTool() mcp.Tool {
    return mcp.NewTool("skulto_get_skill",
        mcp.WithDescription("Get detailed skill information including full content and tags"),
        mcp.WithString("slug",
            mcp.Required(),
            mcp.Description("The skill's unique slug identifier"),
        ),
    )
}

func (s *Server) handleGetSkill(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    start := time.Now()

    slug := request.Params.Arguments["slug"].(string)

    skill, err := s.db.GetSkillBySlugWithPriority(slug)
    if err != nil || skill == nil {
        s.telemetry.TrackMCPToolCalled("skulto_get_skill", time.Since(start).Milliseconds(), false)
        return mcp.NewToolResultError("skill not found"), nil
    }

    // Record view
    _ = s.db.RecordSkillView(skill.ID)

    // Track telemetry
    s.telemetry.TrackMCPToolCalled("skulto_get_skill", time.Since(start).Milliseconds(), true)
    s.telemetry.TrackSkillViewed(skill.Slug, skill.Category, skill.IsLocal)

    result := formatSkillDetail(skill)
    return mcp.NewToolResultText(result), nil
}
```

#### `skulto_list_skills`

List all skills with pagination.

```go
func listSkillsTool() mcp.Tool {
    return mcp.NewTool("skulto_list_skills",
        mcp.WithDescription("List all skills with pagination"),
        mcp.WithNumber("limit", mcp.Description("Max results (default 20, max 100)")),
        mcp.WithNumber("offset", mcp.Description("Offset for pagination")),
    )
}
```

### Browse Tools (Phase 1B)

#### `skulto_browse_tags`

List available tags by category.

```go
func browseTagsTool() mcp.Tool {
    return mcp.NewTool("skulto_browse_tags",
        mcp.WithDescription("List available tags for filtering"),
        mcp.WithString("category",
            mcp.Description("Filter by category: language, framework, tool, concept, domain"),
        ),
    )
}
```

#### `skulto_get_stats`

Get database statistics.

```go
func getStatsTool() mcp.Tool {
    return mcp.NewTool("skulto_get_stats",
        mcp.WithDescription("Get database statistics including total skills, tags, sources"),
    )
}
```

#### `skulto_get_recent`

Get recently viewed skills.

```go
func getRecentTool() mcp.Tool {
    return mcp.NewTool("skulto_get_recent",
        mcp.WithDescription("Get recently viewed skills ordered by view time"),
        mcp.WithNumber("limit", mcp.Description("Max results (default 10, max 50)")),
    )
}
```

### User State Tools (Phase 2)

#### `skulto_install`

Install a skill to specified platforms.

```go
func installTool() mcp.Tool {
    return mcp.NewTool("skulto_install",
        mcp.WithDescription("Install a skill to Claude Code"),
        mcp.WithString("slug",
            mcp.Required(),
            mcp.Description("The skill's unique slug identifier"),
        ),
        mcp.WithArray("platforms",
            mcp.Description("Platforms to install to (default: user's configured platforms)"),
        ),
        mcp.WithString("scope",
            mcp.Description("Installation scope: 'global' or 'project' (default: project)"),
        ),
    )
}

func (s *Server) handleInstall(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    start := time.Now()

    slug := request.Params.Arguments["slug"].(string)

    // Parse platforms
    var platforms []string
    if p, ok := request.Params.Arguments["platforms"].([]interface{}); ok {
        for _, v := range p {
            if str, ok := v.(string); ok {
                platforms = append(platforms, str)
            }
        }
    }

    // Parse scope
    scope := installer.ScopeProject
    if s, ok := request.Params.Arguments["scope"].(string); ok && s == "global" {
        scope = installer.ScopeGlobal
    }

    // Use InstallService for unified installation
    opts := installer.InstallOptions{
        Platforms: platforms,
        Scopes:    []installer.InstallScope{scope},
        Confirm:   true,
    }

    result, err := s.installService.Install(ctx, slug, opts)
    if err != nil {
        s.telemetry.TrackMCPToolCalled("skulto_install", time.Since(start).Milliseconds(), false)
        return mcp.NewToolResultError(err.Error()), nil
    }

    s.telemetry.TrackMCPToolCalled("skulto_install", time.Since(start).Milliseconds(), true)

    return mcp.NewToolResultText(formatInstallResult(result)), nil
}
```

#### `skulto_uninstall`

Uninstall a skill from specified platforms.

```go
func uninstallTool() mcp.Tool {
    return mcp.NewTool("skulto_uninstall",
        mcp.WithDescription("Uninstall a skill from specified platforms"),
        mcp.WithString("slug", mcp.Required()),
        mcp.WithArray("platforms"),
        mcp.WithString("scope"),
    )
}
```

#### `skulto_favorite`

Add or remove a skill from favorites.

```go
func favoriteTool() mcp.Tool {
    return mcp.NewTool("skulto_favorite",
        mcp.WithDescription("Add or remove a skill from favorites"),
        mcp.WithString("slug", mcp.Required()),
        mcp.WithString("action",
            mcp.Required(),
            mcp.Description("Action: 'add' or 'remove'"),
        ),
    )
}
```

#### `skulto_get_favorites`

Get favorite skills.

```go
func getFavoritesTool() mcp.Tool {
    return mcp.NewTool("skulto_get_favorites",
        mcp.WithDescription("Get your favorite skills"),
        mcp.WithNumber("limit"),
    )
}
```

### Management Tools

#### `skulto_check`

List all installed skills and their locations.

```go
func checkTool() mcp.Tool {
    return mcp.NewTool("skulto_check",
        mcp.WithDescription("List all installed skills and their installation locations"),
    )
}

func (s *Server) handleCheck(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    start := time.Now()

    // Sync install state first (same as TUI)
    _ = s.installer.SyncInstallState(ctx)

    // Get installed skills summary
    summaries, err := s.installService.GetInstalledSkillsSummary(ctx)
    if err != nil {
        s.telemetry.TrackMCPToolCalled("skulto_check", time.Since(start).Milliseconds(), false)
        return mcp.NewToolResultError(err.Error()), nil
    }

    // Track telemetry
    s.telemetry.TrackMCPToolCalled("skulto_check", time.Since(start).Milliseconds(), true)
    s.telemetry.TrackInstalledSkillsChecked(len(summaries))

    return mcp.NewToolResultText(formatCheckResult(summaries)), nil
}
```

#### `skulto_add`

Add a skill repository.

```go
func addTool() mcp.Tool {
    return mcp.NewTool("skulto_add",
        mcp.WithDescription("Add a skill repository and sync its skills"),
        mcp.WithString("repository",
            mcp.Required(),
            mcp.Description("Repository URL or owner/repo format"),
        ),
    )
}
```

## Resources

MCP resources provide direct skill content access.

### Skill Content Resource

```go
// Template: skulto://skill/{slug}
func (s *Server) handleSkillContentResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
    // Extract slug from URI
    slug := extractSlugFromURI(request.Params.URI)

    skill, err := s.db.GetSkillBySlugWithPriority(slug)
    if err != nil || skill == nil {
        return nil, fmt.Errorf("skill not found: %s", slug)
    }

    // Record view
    _ = s.db.RecordSkillView(skill.ID)

    return []mcp.ResourceContents{
        mcp.NewTextResourceContents(
            request.Params.URI,
            skill.Content,
            "text/markdown",
        ),
    }, nil
}
```

### Skill Metadata Resource

```go
// Template: skulto://skill/{slug}/metadata
func (s *Server) handleSkillMetadataResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
    slug := extractSlugFromURI(request.Params.URI)

    skill, err := s.db.GetSkillBySlugWithPriority(slug)
    if err != nil || skill == nil {
        return nil, fmt.Errorf("skill not found: %s", slug)
    }

    // Build metadata JSON
    metadata := map[string]interface{}{
        "slug":        skill.Slug,
        "title":       skill.Title,
        "description": skill.Description,
        "author":      skill.Author,
        "category":    skill.Category,
        "difficulty":  skill.Difficulty,
        "tags":        extractTagNames(skill.Tags),
        "is_local":    skill.IsLocal,
        "stars":       skill.Stars,
        "forks":       skill.Forks,
        "source_id":   skill.SourceID,
        "threat_level": skill.ThreatLevel,
    }

    jsonBytes, _ := json.MarshalIndent(metadata, "", "  ")

    return []mcp.ResourceContents{
        mcp.NewTextResourceContents(
            request.Params.URI,
            string(jsonBytes),
            "application/json",
        ),
    }, nil
}
```

## Startup Sequence

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle signals
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigCh
        cancel()
    }()

    // 1. Load config
    cfg, _ := config.Load()
    paths := config.GetPaths(cfg)

    // 2. Open database
    database, _ := db.New(db.DefaultConfig(paths.Database))
    defer database.Close()

    // 3. Initialize favorites store
    favStore := favorites.NewStore(paths.Favorites)
    _ = favStore.Load()

    // 4. Initialize telemetry
    tc := telemetry.New(database)
    defer tc.Close()

    // 5. Create and serve MCP server
    server := mcp.NewServer(database, cfg, favStore, tc)
    server.Serve(ctx)  // Blocks until context cancelled
}
```

## Server Initialization

```go
func NewServer(database *db.DB, cfg *config.Config, favStore *favorites.Store, tc telemetry.Client) *Server {
    s := &Server{
        db:             database,
        cfg:            cfg,
        installer:      installer.New(database, cfg),
        installService: installer.NewInstallService(database, cfg, tc),
        favorites:      favStore,
        telemetry:      tc,
    }

    // Create mcp-go server with capabilities
    s.server = server.NewMCPServer(
        "skulto",
        version.Version,
        server.WithToolCapabilities(true),
        server.WithResourceCapabilities(true, false),  // subscribe=false
    )

    // Register handlers
    s.registerTools()
    s.registerResources()

    return s
}
```

## Serve Loop

```go
func (s *Server) Serve(ctx context.Context) error {
    // Sync install state on startup (same as TUI)
    _ = s.installer.SyncInstallState(ctx)

    // Start JSON-RPC server over stdio
    return server.ServeStdio(s.server)
}
```

## Response Formatting

All tool responses are formatted as human-readable text with JSON where appropriate.

### Search Results Format

```
Found 5 skills matching "test":

1. tdd-master (by asteroid-belt)
   Test-driven development workflow for Go projects
   Tags: testing, go, tdd
   Installed: Yes (claude global)

2. test-automation (by skills-hub)
   Automated testing patterns
   Tags: testing, automation
   Installed: No
```

### Install Result Format

```
Successfully installed "superplan" to:
- claude (global): ~/.claude/skills/superplan
- cursor (project): ./.cursor/skills/superplan
```

### Check Result Format

```
Installed skills (3):

superplan
  - claude (global): ~/.claude/skills/superplan
  - cursor (project): ./.cursor/skills/superplan

teach
  - claude (global): ~/.claude/skills/teach

agentsmd-generator
  - claude (project): ./.claude/skills/agentsmd-generator
```

## Error Handling

Errors are returned via `mcp.NewToolResultError()`:

```go
func (s *Server) handleInstall(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // ...
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("installation failed: %v", err)), nil
    }
    // ...
}
```

## Testing

```go
func TestHandleSearch(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()

    // Create server with noop telemetry
    tc := telemetry.New(nil)
    server := NewServer(db, &config.Config{}, nil, tc)

    // Create request
    request := mcp.CallToolRequest{
        Params: mcp.CallToolParams{
            Arguments: map[string]interface{}{
                "query": "test",
                "limit": float64(10),
            },
        },
    }

    // Call handler
    result, err := server.handleSearch(context.Background(), request)

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```
