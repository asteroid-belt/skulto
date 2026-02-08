# Architecture

> How Skulto is structured and why.

## System Overview

Skulto exposes identical functionality through three interfaces that share a common service layer:

```text
┌─────────────────────────────────────────────────────────────────┐
│                        User Interfaces                          │
├─────────────────┬─────────────────┬─────────────────────────────┤
│   CLI (Cobra)   │   TUI (Bubble)  │   MCP (mcp-go over stdio)   │
│   cmd/skulto    │   internal/tui  │   cmd/skulto-mcp            │
├─────────────────┴─────────────────┴─────────────────────────────┤
│                      Shared Services                            │
├─────────────────────────────────────────────────────────────────┤
│  InstallService   │  SearchService   │  SecurityScanner         │
│  (installer/)     │  (search/)       │  (security/)             │
│  Scraper          │  Telemetry       │  Favorites               │
│  (scraper/)       │  (telemetry/)    │  (favorites/)            │
├─────────────────────────────────────────────────────────────────┤
│                      Data Layer                                 │
├─────────────────────────────────────────────────────────────────┤
│  Database (db/)       │  Vector Store (vector/)                  │
│  SQLite + GORM + FTS5 │  chromem-go + OpenAI embeddings          │
└─────────────────────────────────────────────────────────────────┘
```

Both binaries (`skulto` and `skulto-mcp`) follow the same initialization pattern: load config from environment variables, open the SQLite database, initialize telemetry, then hand off to their respective interface layer.

## Components

### CLI (`cmd/skulto/`, `internal/cli/`)

| Attribute | Value |
|-----------|-------|
| **Location** | `cmd/skulto/main.go`, `internal/cli/` |
| **Responsibility** | Parse commands and flags, execute operations, launch TUI when no subcommand given |
| **Dependencies** | config, db, telemetry, installer, scraper, security, discovery |
| **Depends On It** | Nothing (entry point) |

The CLI is built with Cobra and Fang. Each subcommand lives in its own file (e.g., `internal/cli/install.go`). When `skulto` is invoked without a subcommand, `internal/cli/tui.go` launches the Bubble Tea TUI. Each command creates its own database and service instances, following a pattern of: load config, open DB, create service, execute, close.

### TUI (`internal/tui/`)

| Attribute | Value |
|-----------|-------|
| **Location** | `internal/tui/` |
| **Responsibility** | Interactive terminal interface with views for home, search, detail, settings, onboarding, and management |
| **Dependencies** | db, installer, scraper, search, security, telemetry, favorites |
| **Depends On It** | CLI (launches it) |

The TUI uses the Elm architecture via Bubble Tea: a central `App` model dispatches messages to the active view, and each view returns an updated model and optional commands. Views are in `internal/tui/views/`, reusable UI components (dialogs, selectors, tag grids) are in `internal/tui/components/`, and design constants (colors, ASCII art) are in `internal/tui/design/`.

### MCP Server (`cmd/skulto-mcp/`, `internal/mcp/`)

| Attribute | Value |
|-----------|-------|
| **Location** | `cmd/skulto-mcp/main.go`, `internal/mcp/` |
| **Responsibility** | Expose Skulto operations to AI assistants via JSON-RPC 2.0 over stdio |
| **Dependencies** | db, config, favorites, telemetry, installer |
| **Depends On It** | Claude Code and other MCP clients |

The MCP server is a separate binary that communicates via stdin/stdout. It registers tools (search, install, get_skill, etc.) and resources (`skulto://skill/{slug}`) using the mcp-go library. Handlers in `internal/mcp/handlers.go` wrap the same service layer the CLI and TUI use.

### Installer (`internal/installer/`)

| Attribute | Value |
|-----------|-------|
| **Location** | `internal/installer/` |
| **Responsibility** | Create/remove symlinks between cloned repositories and platform skill directories, track installations |
| **Dependencies** | db, config, models |
| **Depends On It** | CLI, TUI, MCP |

The installer creates symlinks from skill directories in cloned repositories to each platform's skills directory. `InstallService` provides a higher-level API with platform detection and telemetry. The platform registry (`platform.go`) is a data-driven map defining 33 platforms with their commands, paths, and detection heuristics. Installation state is tracked in the `skill_installations` table.

### Scraper (`internal/scraper/`)

| Attribute | Value |
|-----------|-------|
| **Location** | `internal/scraper/` |
| **Responsibility** | Clone GitHub repositories, parse skill markdown files, extract tags, detect licenses |
| **Dependencies** | db, go-git, go-github |
| **Depends On It** | CLI (add/pull/update commands), TUI (onboarding) |

The scraper uses shallow git clones (via go-git) to fetch repositories locally. `SkillParser` reads markdown files, extracts YAML frontmatter, and generates skill records. `ExtractTagsWithContext` infers tags from content with implied tag relationships (e.g., "react" implies "javascript"). A response cache with TTL prevents redundant API calls.

### Security Scanner (`internal/security/`)

| Attribute | Value |
|-----------|-------|
| **Location** | `internal/security/` |
| **Responsibility** | Detect prompt injection patterns in skill content and auxiliary files |
| **Dependencies** | models |
| **Depends On It** | CLI (scan command), TUI (detail view), MCP |

The scanner applies ~40 regex patterns across categories: instruction override, jailbreak, data exfiltration, and dangerous commands. Each pattern has a severity weight (CRITICAL=10, HIGH=5, MEDIUM=2, LOW=1). A context analyzer calculates mitigation scores for educational content or safe contexts. Final score = base - mitigation; skills scoring >= 3 are quarantined.

### Database (`internal/db/`)

| Attribute | Value |
|-----------|-------|
| **Location** | `internal/db/` |
| **Responsibility** | Data persistence, FTS5 search, CRUD operations for all entities |
| **Dependencies** | GORM, SQLite (pure-Go driver), models |
| **Depends On It** | Everything |

The database wraps GORM with Skulto-specific operations. It uses the pure-Go SQLite driver (glebarez/sqlite) which includes FTS5 support without CGO. On initialization, it auto-migrates schema, creates the `skills_fts` virtual table, and sets up triggers for automatic FTS index synchronization. Operations are organized by entity in separate files (skills.go, tags.go, sources.go, etc.).

### Search Service (`internal/search/`)

| Attribute | Value |
|-----------|-------|
| **Location** | `internal/search/` |
| **Responsibility** | Hybrid search combining FTS5 and optional vector similarity |
| **Dependencies** | db, vector |
| **Depends On It** | TUI (search view), MCP (skulto_search) |

The search service queries both FTS5 and the vector store (if enabled), deduplicates results, and categorizes matches as title/tag matches vs content matches. Snippet extraction provides context around matching terms. The vector store is gracefully degraded - search works with FTS5 alone when no OpenAI API key is configured.

### Telemetry (`internal/telemetry/`)

| Attribute | Value |
|-----------|-------|
| **Location** | `internal/telemetry/` |
| **Responsibility** | Anonymous usage tracking via PostHog |
| **Dependencies** | PostHog SDK, db (for tracking ID) |
| **Depends On It** | CLI, TUI, MCP |

Implements a `Client` interface with `posthogClient` (real) and `noopClient` (disabled) implementations. Events are defined in `events.go` and cover CLI commands, TUI navigation, MCP tool calls, and cross-interface actions (install, search, scan). The API key is injected at build time via ldflags.

## Data Flow

### Skill Installation

1. User selects a skill and target platforms (via CLI, TUI, or MCP)
2. `InstallService.DetectPlatforms()` checks which AI tools are installed on the system
3. For each selected platform+scope, `Installer.installToLocationsInternal()` resolves the source path in the cloned repository
4. A symlink is created from the repo skill directory to the platform's skills directory
5. A `SkillInstallation` record is persisted in the database
6. Telemetry tracks the installation event

### Skill Search

1. User enters a query (min 3 characters in TUI, any length in MCP/CLI)
2. `SearchService.Search()` dispatches to both FTS5 and vector store (if available)
3. FTS5 returns BM25-ranked results from the `skills_fts` virtual table
4. Vector store returns cosine-similarity results above a threshold (default 0.6)
5. Results are deduplicated, categorized (title vs content match), and sorted
6. Snippet extraction provides context around matching terms

### Repository Sync

1. `Scraper.ScrapeRepository()` clones or fetches the repository via go-git
2. `SkillParser` walks the repository tree, finding markdown files with skill content
3. Tags are extracted with contextual inference and implied tag relationships
4. Skills are upserted in the database with deduplication by slug+source
5. `SecurityScanner.ScanSkill()` runs on each skill, setting threat level and security status
6. FTS5 triggers automatically update the search index

## Key Interfaces

| Interface | Location | Purpose |
|-----------|----------|---------|
| `telemetry.Client` | `internal/telemetry/client.go` | Event tracking abstraction (posthog vs noop) |
| `vector.VectorStore` | `internal/vector/store.go` | Vector storage and similarity search |
| `scraper.Client` | `internal/scraper/scraper.go` | GitHub data access (git clone vs API) |
| `installer.Platform` | `internal/installer/platform.go` | Platform identity and metadata |

## External Dependencies

| Dependency | Purpose | Why This Choice |
|------------|---------|-----------------|
| Cobra + Fang | CLI framework | Industry-standard Go CLI framework with subcommand support and auto-generated help |
| Bubble Tea + Lip Gloss | TUI framework | Elm architecture for terminals, rich styling, active Charm ecosystem |
| GORM + glebarez/sqlite | Database ORM | Pure-Go SQLite driver (no CGO required) with FTS5 support, enabling cross-compilation |
| mcp-go | MCP protocol | Reference Go implementation of MCP for AI tool integration |
| go-git | Git operations | Pure-Go git implementation, enables shallow clones without git binary dependency |
| PostHog | Analytics | Product analytics with generous free tier, Go SDK, and GDPR-friendly opt-out |
| chromem-go | Vector store | Embedded vector database for semantic search, no external service required |

## Design Decisions

Key architectural choices and their rationale:

1. **Symlinks for installation** - Skills are installed as symlinks, not copies. See [ADR-0001](adr/0001-symlink-based-skill-installation.md).
2. **Pure-Go SQLite** - No CGO dependency enables easy cross-compilation. See [ADR-0002](adr/0002-pure-go-sqlite-with-fts5.md).
3. **Three interfaces, one codebase** - CLI, TUI, and MCP share services. See [ADR-0003](adr/0003-three-interfaces-shared-services.md).
4. **Environment-variable-only config** - No config file; all settings via env vars. See [ADR-0004](adr/0004-environment-variable-only-configuration.md).
5. **Git clone over GitHub API** - Repository sync uses git clone instead of the REST API. See [ADR-0005](adr/0005-git-clone-over-github-api.md).

## Related Documentation

- [Overview](overview.md) - Project purpose and scope
- [Glossary](glossary.md) - Domain terminology used here
- [Decisions](adr/README.md) - Architecture Decision Records
