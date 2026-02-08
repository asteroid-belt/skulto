# Skulto

> Cross-platform CLI tool for managing AI coding assistant skills across 33 platforms, with built-in security scanning for prompt injection.

## What Is This?

AI coding assistants like Claude Code, Cursor, and GitHub Copilot support "skills" - markdown instruction files that customize their behavior. As the ecosystem grows, developers accumulate skills across multiple tools, but there is no unified way to discover, install, update, or audit them. Skills can also contain prompt injection attacks, and installing one without review can compromise your AI assistant's behavior.

Skulto solves this by providing a single tool that manages the full skill lifecycle: discovering skills from GitHub repositories, indexing them with full-text search, scanning them for security threats, and installing them as symlinks to any combination of 33 AI platforms. It works offline after initial sync and tracks installations across global and project scopes.

Skulto is built for developers who use multiple AI coding tools and want a central hub to manage their skill libraries safely. It exposes the same functionality through three interfaces - a CLI for scripting, a TUI for interactive browsing, and an MCP server for programmatic access from AI assistants themselves.

## Key Features

| Feature | Description |
|---------|-------------|
| Multi-platform installation | Install skills to 33 AI tools via symlinks, with per-platform scope control (global vs project) |
| Security scanning | Regex-based prompt injection detection with severity scoring, context-aware mitigation, and quarantine system |
| Full-text search | SQLite FTS5 with BM25 ranking (~50ms latency), optional semantic search via OpenAI embeddings |
| Repository management | Git-clone-based sync with shallow clones, automatic skill parsing from markdown with YAML frontmatter |
| Interactive TUI | Bubble Tea terminal interface with home dashboard, search, skill details, onboarding flow, and platform choosers |
| MCP server | JSON-RPC 2.0 over stdio for AI assistants to search, install, and manage skills programmatically |
| Platform detection | Automatic detection of installed AI tools via PATH commands, directory checks, and OS-specific paths |
| Offline-first | Full functionality after initial repository sync, no internet required for search or installation |
| Favorites | Skill bookmarks persisted outside the database, surviving resets |
| Skill discovery | Detects unmanaged skills in platform directories and offers to import them into Skulto management |

## Project Status

| Attribute | Value |
|-----------|-------|
| **Stage** | Production |
| **License** | MIT |
| **Primary Language** | Go 1.25+ |
| **CLI Framework** | Cobra + Fang |
| **TUI Framework** | Bubble Tea + Lip Gloss |
| **Database** | SQLite (GORM) with FTS5 |
| **Distribution** | Homebrew (`asteroid-belt/tap/skulto`) and GitHub Releases |

## Quick Links

- [Getting Started](getting-started.md) - Set up and run the project
- [Architecture](architecture.md) - How the system is designed
- [Development](development.md) - Contributing and testing
- [Decisions](adr/README.md) - Architecture Decision Records
- [Glossary](glossary.md) - Project terminology
