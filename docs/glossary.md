# Glossary

> Project-specific terminology used in Skulto.

Terms are defined as they are used in this project. General programming terms
are omitted unless they have a project-specific meaning.

| Term | Definition | Where Used |
|------|-----------|------------|
| Skill | A markdown file (typically with YAML frontmatter) that provides instructions, prompts, or capabilities to an AI coding assistant. Skills are the atomic unit Skulto manages. | `internal/models/skill.go`, all views and commands |
| Source | A GitHub repository that contains one or more skills. Skulto clones sources locally and indexes their skills. | `internal/models/source.go`, `internal/scraper/` |
| Platform | An AI coding tool (e.g., Claude Code, Cursor, Windsurf) that can consume skills. Skulto supports 33 platforms. | `internal/installer/platform.go` |
| Slug | A URL-safe identifier for a skill, derived from its title. Unique within a source. Used as the primary lookup key in MCP tools and CLI commands. | `internal/models/skill.go` |
| Scope | Whether a skill is installed globally (`~/.platform/skills/`) or per-project (`./.platform/skills/`). Each platform+scope combination is a separate install location. | `internal/installer/scope.go` |
| Installation | A symlink from a cloned skill directory to a platform's skills directory. Tracked in the `skill_installations` table. | `internal/installer/installer.go`, `internal/models/skill_installation.go` |
| Threat Level | Security classification assigned by the scanner: NONE, LOW, MEDIUM, HIGH, or CRITICAL. Determines whether a skill is quarantined. | `internal/models/security.go`, `internal/security/` |
| Quarantine | A security status that blocks a skill from being installed. Applied when scanning detects prompt injection patterns above a confidence threshold. | `internal/security/scanner.go` |
| Discovered Skill | A non-symlinked skill directory found in a platform's skill folder that is not managed by Skulto. The discovery system notifies users about these. | `internal/models/discovered_skill.go`, `internal/discovery/` |
| Ingestion | The process of importing a discovered (unmanaged) skill into Skulto management by copying it to `~/.skulto/skills/` and replacing the original with a symlink. | `internal/discovery/ingestion.go` |
| Seed | A pre-configured source repository that ships with Skulto (e.g., `asteroid-belt/skills`). Seeds are scraped during onboarding. | `internal/scraper/seeds.go` |
| Onboarding | The first-run experience where users select AI platforms and starter skills. Tracked via `UserState.OnboardingStatus`. | `internal/tui/views/onboarding_*.go` |
| FTS5 | SQLite's Full-Text Search extension version 5, used for BM25-ranked skill search. Skulto maintains a virtual table (`skills_fts`) synchronized via triggers. | `internal/db/db.go`, `internal/db/skills.go` |
| BM25 | Best Matching 25 - the ranking algorithm used by FTS5 to score search results by relevance. Skulto configures column weights for title, description, and content. | `internal/db/skills.go` |
| Vector Store | An optional semantic search layer backed by chromem-go and OpenAI embeddings. Provides similarity-based search when an API key is configured. | `internal/vector/`, `internal/search/` |
| MCP | Model Context Protocol - a JSON-RPC 2.0 protocol over stdio that enables AI assistants like Claude Code to interact with external tools. Skulto's MCP server exposes search, install, and management capabilities. | `internal/mcp/`, `cmd/skulto-mcp/` |
| Auxiliary File | A non-markdown file (script, reference, asset) bundled with a skill. These are tracked separately and scanned independently for security threats. | `internal/models/skill.go` |
| Tag | A categorization label applied to skills. Tags have categories (language, framework, tool, concept, domain) and are used for browsing and filtering. | `internal/models/tag.go`, `internal/db/tags.go` |
| Favorites | User-bookmarked skills persisted in `~/.skulto/favorites.json`. Favorites survive database resets because they are stored outside the SQLite database. | `internal/favorites/favorites.go` |

## Acronyms

| Acronym | Expansion | Meaning |
|---------|-----------|---------|
| CLI | Command-Line Interface | Skulto's Cobra-based command interface (`skulto add`, `skulto install`, etc.) |
| TUI | Terminal User Interface | Skulto's interactive Bubble Tea-based visual interface, launched via `skulto` with no subcommand |
| MCP | Model Context Protocol | JSON-RPC 2.0 protocol for AI tool integration, served by `skulto-mcp` |
| FTS | Full-Text Search | SQLite FTS5 extension used for skill search |
| CWD | Current Working Directory | Used when resolving project-scope installations |
