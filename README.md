# Skulto

> Offline-first tool for syncing and managing agent skills

[![CI](https://github.com/asteroid-belt/skulto/actions/workflows/ci.yml/badge.svg)](https://github.com/asteroid-belt/skulto/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

```
   ╔════════════════════════════════════════════════════════╗
   ║    ███████╗██╗  ██╗██╗   ██╗██╗  ████████╗ ██████╗     ║
   ║    ██╔════╝██║ ██╔╝██║   ██║██║  ╚══██╔══╝██╔═══██╗    ║
   ║    ███████╗█████╔╝ ██║   ██║██║     ██║   ██║   ██║    ║
   ║    ╚════██║██╔═██╗ ██║   ██║██║     ██║   ██║   ██║    ║
   ║    ███████║██║  ██╗╚██████╔╝███████╗██║   ╚██████╔╝    ║
   ║    ╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝    ╚═════╝     ║
   ╠════════════════════════════════════════════════════════╣
   ║            CROSS-PLATFORM AI SKILLS MANAGEMENT         ║
   ╚════════════════════════════════════════════════════════╝
```

![Skulto Demo](assets/demo.gif)

## What is Skulto?

Skulto is a cross-platform CLI tool for managing AI coding assistant skills across 30+ platforms. It provides:

1. **Multi-platform installation** - Install skills to Claude Code, Cursor, Windsurf, Copilot, Codex, Cline, Roo Code, Gemini CLI, Kiro CLI, and 25+ more
2. **Repository management** - Add, sync, and remove skill repositories
3. **Full-text search** - SQLite FTS5-powered search across all indexed skills
4. **Security scanning** - Detect prompt injection and dangerous code patterns
5. **Platform detection** - Automatically detects which AI tools are installed on your system
6. **Interactive TUI** - Bubble Tea-powered terminal interface with collapsible groups, multi-select, and keyboard navigation
7. **URL-based install** - Install directly from GitHub repositories via `skulto install owner/repo`

## Features

- **30+ platform support** - Claude Code, Cursor, Windsurf, GitHub Copilot, OpenAI Codex, OpenCode, Cline, Roo Code, Gemini CLI, Kiro CLI, Amp, Continue, Goose, Junie, Qwen Code, Trae, and more
- **Platform detection** - Detects installed AI tools and surfaces them in platform choosers
- **Offline-first** - Works without internet after initial sync
- **Fast search** - FTS5-powered full-text search with BM25 ranking (~50ms latency)
- **Git-based sync** - Clone and pull repositories for reliable updates
- **Security scanner** - Detects prompt injection in frontmatter, references, scripts and dangerous patterns with threat levels
- **Smart multi-skill install** - Install multiple skills from a repository URL with per-skill conflict resolution (skip already-installed, add new locations, or skip all)
- **Scope selection** - Install skills globally (`~/`) or per-project (`./`) with separate control per platform
- **Collapsible platform groups** - Detected/preferred platforms at top, all others in a collapsed group across all choosers
- **Install location memory** - Optionally remember your platform/scope choices for future installs
- **Favorites** - Save favorite skills that persist across database resets
- **Recently viewed** - Tracks and displays skills you've recently viewed
- **MCP Server** - Model Context Protocol server for AI tool integration (search, install, manage skills programmatically)
- **Telemetry** - Anonymous usage stats (opt-out with env var in Settings)

### Supported Platforms

Skulto detects and installs skills to 33 AI coding tools:

| | | | |
|---|---|---|---|
| Claude Code | Cursor | Windsurf | GitHub Copilot |
| OpenAI Codex | OpenCode | Cline | Roo Code |
| Gemini CLI | Kiro CLI | Amp | Continue |
| Goose | Junie | Kilo Code | Trae |
| Qwen Code | Kimi Code CLI | CodeBuddy | Command Code |
| Crush | Droid | Kode | MCPJam |
| Mux | OpenHands | Pi | Qoder |
| Zencoder | Neovate | Pochi | Antigravity |
| Moltbot | | | |

## Installation

### Homebrew

```bash
brew install asteroid-belt/tap/skulto
```

To upgrade:

```bash
brew upgrade asteroid-belt/tap/skulto
```

### From Source

```bash
# Clone the repository
git clone https://github.com/asteroid-belt/skulto.git
cd skulto

# Install dependencies
make deps

# Build (outputs to ./build/)
make build-all

# Run
./build/skulto
```

### Requirements

- Go 1.25+
- (Optional) `GITHUB_TOKEN` for higher API rate limits

## Quick Start

```bash
# Launch the TUI (guided onboarding on first run)
skulto

# Or install skills directly from a repository URL
skulto install asteroid-belt/skills
```

On first launch, Skulto walks you through onboarding:

1. **Platform selection** - Detected AI tools appear at top; select which ones to sync skills to
2. **Skill selection** - Curated starter skills from Asteroid Belt (superplan, superbuild, teach, agentsmd-generator, and more)
3. **Location chooser** - Pick global or project scope per platform, with your previous selections pre-filled

## Usage

### TUI Mode (Default)

```bash
skulto
```

**Key Bindings:**

| Key | Action |
| --- | --- |
| `/` | Open search |
| `j` / `k` | Navigate down / up |
| `h` / `l` | Navigate left / right (between columns) |
| `↑` / `↓` | Navigate results |
| `Enter` | Select / confirm |
| `Space` | Toggle selection (in choosers) |
| `f` | Toggle favorite / bookmark |
| `i` | Install / manage skill locations |
| `c` | Copy skill content to clipboard |
| `p` | Pull/sync repositories |
| `Esc` | Back / cancel |
| `q` | Quit |

### Home Dashboard

The home view displays three columns:

1. **Installed Skills** - Your installed skills (scrollable, shows up to 5 at a time)
2. **Recently Viewed Skills** - Skills you've recently viewed
3. **Top Tags** - Popular skill categories

### Skill Details

When you select a skill, you'll see:

- **Install / Manage** - Install to new platforms or manage existing locations
- **Metadata** - Author, category, source repository
- **Tags** - Categorized skill tags
- **Security status** - Threat level from security scan
- **Full markdown content** - Rendered with syntax highlighting and scrolling
- **Copy to clipboard** - Press `c` to copy the full skill content

### Install Location Dialog

When installing a skill, you choose where to install it:

- **Platform headers** - Each AI tool listed with nested scope options
- **Global vs Project** - Install to `~/.claude/skills/` (user-wide) or `./.claude/skills/` (project-local)
- **Collapsible groups** - Preferred/detected platforms at top, others collapsed below
- **Remember locations** - Optionally save choices for future installs
- **Quick keys** - `a` all, `n` none, `g` global only, `p` project only

### Manage View

Press `i` on an installed skill to manage its locations:

- **Installed platforms** shown at top with checkboxes pre-selected
- **Other platforms** collapsed below in an expandable group
- **Add/remove** locations across any combination of platforms and scopes

### CLI Commands

Skulto provides CLI subcommands for scripting and automation:

| Command | Purpose |
| --- | --- |
| `skulto` | Launch the interactive TUI |
| `skulto install <slug or repo>` | Install skills by slug or from a repository URL |
| `skulto add <repo>` | Add a skill repository and sync its skills |
| `skulto list` | List all configured source repositories |
| `skulto pull` | Pull/sync all repositories and reconcile installed skills |
| `skulto remove [repo]` | Remove a repository (interactive selection if no repo specified) |
| `skulto scan` | Scan skills for security threats |
| `skulto update` | Pull + scan with change reporting |
| `skulto info <slug>` | Show detailed information about a skill |
| `skulto favorites add <slug>` | Add a skill to favorites |
| `skulto favorites remove <slug>` | Remove a skill from favorites |
| `skulto favorites list` | List all favorited skills |
| `skulto feedback` | Open the feedback/bug report page |

#### `skulto install`

Install skills by slug or directly from a GitHub repository:

```bash
# Install a single skill by slug
skulto install superplan

# Install from a repository (auto-detects all skills)
skulto install asteroid-belt/skills

# Install from a full GitHub URL
skulto install https://github.com/asteroid-belt/skills

# Non-interactive mode (accept defaults)
skulto install asteroid-belt/skills -y
```

When installing from a repository URL:
1. Skulto syncs the repository and presents all available skills
2. Select which skills to install with an interactive checklist
3. Choose target platforms with a collapsible platform chooser (detected platforms at top)
4. **Smart skip** for already-installed skills: prompted with `y` (add locations), `N` (skip, default), or `s` (skip all remaining)
5. Final summary shows installed, skipped, and failed counts

#### `skulto add <repo>`

Add a skill repository to Skulto:

```bash
# Short format
skulto add asteroid-belt/skills

# Full URL
skulto add https://github.com/asteroid-belt/skills

# Skip initial sync
skulto add asteroid-belt/skills --no-sync
```

#### `skulto pull`

Sync all registered repositories:

```bash
skulto pull
```

This clones/updates all repositories and reconciles installed skill state with the filesystem.

#### `skulto remove`

Remove a repository and all its skills:

```bash
# Interactive selection
skulto remove

# Specify repository
skulto remove asteroid-belt/skills

# Skip confirmation
skulto remove asteroid-belt/skills --force
```

#### `skulto scan`

Scan skills for security threats:

```bash
# Scan all skills
skulto scan --all

# Scan specific skill
skulto scan --skill abc123

# Scan skills from a source
skulto scan --source asteroid-belt/skills

# Scan only unscanned skills
skulto scan --pending
```

Reports threat levels: CRITICAL, HIGH, MEDIUM, LOW

#### `skulto update`

Combined pull + scan with reporting:

```bash
# Update and scan new/updated skills
skulto update

# Update and scan ALL skills
skulto update --scan-all
```

#### `skulto favorites`

Manage your favorite skills. Favorites persist across database resets and are stored separately in `~/.skulto/favorites.json`.

```bash
# Add a skill to favorites
skulto favorites add docker-expert

# Remove a skill from favorites
skulto favorites remove docker-expert

# List all favorited skills
skulto favorites list
```

You can also toggle favorites in the TUI by pressing `f` on any skill detail view.

### MCP Server (`skulto-mcp`)

Skulto includes an MCP (Model Context Protocol) server that exposes skills to Claude Code and other MCP-compatible clients. This enables AI assistants to search, browse, install, and manage skills and repositories programmatically.

Add to your Claude Code settings (`.claude.json`):

```json
{
  "mcpServers": {
    "skulto": {
      "command": "/opt/homebrew/bin/skulto-mcp",
      "type": "stdio"
    }
  }
}
```

#### Available Tools

| Tool | Description |
| --- | --- |
| `skulto_search` | Search skills using full-text search with BM25 ranking |
| `skulto_get_skill` | Get detailed information about a skill including full content and tags |
| `skulto_list_skills` | List all skills with pagination |
| `skulto_browse_tags` | List available tags by category (language, framework, tool, concept, domain) |
| `skulto_get_stats` | Get database statistics (total skills, tags, sources) |
| `skulto_get_recent` | Get recently viewed skills |
| `skulto_install` | Install a skill to any supported platform (30+ platforms, global or project scope) |
| `skulto_uninstall` | Uninstall a skill from specified platforms |
| `skulto_favorite` | Add or remove a skill from favorites |
| `skulto_get_favorites` | Get favorite skills |
| `skulto_check` | List all installed skills and their installation locations |
| `skulto_add` | Add a skill repository and sync its skills |

#### Resources

The MCP server also exposes resources for direct skill access:

| Resource URI | Description |
| --- | --- |
| `skulto://skill/{slug}` | Full markdown content of a skill |
| `skulto://skill/{slug}/metadata` | JSON metadata including tags, source, and stats |

### Database Location

Skulto stores data in `~/.skulto/`:

| Path | Purpose |
| --- | --- |
| `~/.skulto/skulto.db` | SQLite database |
| `~/.skulto/skulto.log` | Logfile |
| `~/.skulto/repositories/` | Cloned git repositories |
| `~/.skulto/favorites.json` | Favorite skills (persists across DB resets) |

## Development

```bash
# Build
make build           # Production build
make dev             # Development build with race detector

# Test
make test            # Run all tests with coverage
make test-race       # Run with race detector

# Lint
make lint            # Run golangci-lint
make format          # Format code

# Clean
make clean           # Remove build artifacts
```

## Architecture

```
skulto/
├── cmd/skulto/              # Main CLI entry point
├── cmd/skulto-mcp/          # MCP server binary
├── internal/
│   ├── cli/                 # Cobra CLI commands (add, install, pull, etc.)
│   │   └── prompts/         # Interactive CLI prompts (platform selector)
│   ├── config/              # Configuration (env vars only)
│   ├── db/                  # GORM + SQLite + FTS5 database layer
│   ├── detect/              # AI tool detection on system
│   ├── embedding/           # Embedding provider abstraction
│   ├── favorites/           # File-based favorites persistence
│   ├── installer/           # Skill installation via symlinks (33 platforms)
│   ├── llm/                 # LLM provider abstraction
│   ├── log/                 # Structured logging
│   ├── mcp/                 # MCP server implementation
│   ├── migration/           # Database migrations
│   ├── models/              # Data structures (Skill, Tag, Source, etc.)
│   ├── scraper/             # GitHub scraping (git clone based)
│   ├── search/              # Search service
│   ├── security/            # Security scanner for skills
│   ├── telemetry/           # PostHog analytics (opt-in)
│   ├── testutil/            # Test utilities
│   ├── tui/                 # Bubble Tea TUI
│   │   ├── components/      # Reusable UI components (dialogs, selectors)
│   │   └── views/           # Screen views (home, search, detail, onboarding, manage)
│   └── vector/              # Vector store
├── pkg/version/             # Version info (set via ldflags)
└── scripts/                 # Build and release scripts
```

## Configuration

Skulto is configured entirely via environment variables (no config file):

| Variable | Purpose |
| --- | --- |
| `GITHUB_TOKEN` | Higher GitHub API rate limits (optional) |
| `OPENAI_API_KEY` | Embeddings for semantic search (optional) |
| `SKULTO_TELEMETRY_TRACKING_ENABLED` | Set to `false` to disable telemetry |

## Telemetry

Skulto collects anonymous usage stats (command frequency, error rates) to improve the tool. **Telemetry is enabled by default.**

To opt-out:

```bash
export SKULTO_TELEMETRY_TRACKING_ENABLED=false
```

No personal data, no IP addresses are collected. See more in [events](./internal/telemetry/events.go).

## Contributing

Contributions welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) first.

## License

MIT License - see [LICENSE](LICENSE) for details.

---

Made with ❤️ by [Asteroid Belt](https://github.com/asteroid-belt)
