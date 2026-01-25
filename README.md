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

Skulto is a cross-platform CLI tool for managing AI coding assistant skills. It provides:

1. **Repository management** - Add, sync, and remove skill repositories from GitHub
2. **Full-text search** - SQLite FTS5-powered search across all indexed skills
3. **Security scanning** - Detect prompt injection and dangerous code patterns
4. **Skill installation** - Install skills to AI tool directories via symlinks
5. **Interactive TUI** - Bubble Tea-powered terminal interface with vim keybindings

## Features

- **Offline-first** - Works without internet after initial sync
- **Fast search** - FTS5-powered full-text search with BM25 ranking (~50ms latency)
- **Git-based sync** - Clone and pull repositories for reliable updates
- **Security scanner** - Detects prompt injection in frontmatter, references, scripts and dangerous patterns with threat levels
- **Skill installation** - Installs skills to global AI tool directories via symlinks
- **Recently viewed** - Tracks and displays skills you've recently viewed
- **Telemetry** - Anonymous usage stats (opt-out with env var in Settings)

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

# Build
make build

# Run
./build/skulto
```

### Requirements

- Go 1.25+
- (Optional) `GITHUB_TOKEN` for higher API rate limits

## Quick Start

```bash
# 1. Set your GitHub token for faster syncing (optional but recommended)
export GITHUB_TOKEN=ghp_your_token_here
make build

# 2. Add a skill repository
./build/skulto add asteroid-belt/skills

# 3. Launch the TUI
./build/skulto
```

## Usage

### TUI Mode (Default)

```bash
./build/skulto
```

**Key Bindings:**

| Key | Action |
| --- | --- |
| `/` | Open search |
| `j` / `k` | Navigate down / up |
| `h` / `l` | Navigate left / right (between columns) |
| `↑` / `↓` | Navigate results |
| `Enter` | Select skill / tag |
| `f` | Toggle favorite / bookmark |
| `i` | Install / uninstall skill |
| `c` | Copy skill content to clipboard |
| `p` | Pull/sync repositories |
| `Esc` | Back to previous view |
| `q` | Quit |

### Home Dashboard

The home view displays three columns:

1. **Installed Skills** - Your installed/bookmarked skills
2. **Recently Viewed Skills** - Skills you've recently viewed
3. **Top Tags** - Popular skill categories

### Skill Details View

When you select a skill, you'll see:

- **Metadata** - Author, category, source repository
- **Tags** - Categorized skill tags
- **Security status** - Threat level from security scan
- **Full markdown content** - Rendered with syntax highlighting and scrolling
- **Favorite toggle** - Press `f` to bookmark a skill
- **Copy to clipboard** - Press `c` to copy the full skill content

### CLI Commands

Skulto provides CLI subcommands for scripting and automation:

| Command | Purpose |
| --- | --- |
| `skulto` | Launch the interactive TUI |
| `skulto add <repo>` | Add a skill repository and sync its skills |
| `skulto list` | List all configured source repositories |
| `skulto pull` | Pull/sync all repositories and reconcile installed skills |
| `skulto remove [repo]` | Remove a repository (interactive selection if no repo specified) |
| `skulto scan` | Scan skills for security threats |
| `skulto update` | Pull + scan with change reporting |
| `skulto info <slug>` | Show detailed information about a skill |

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

### Database Location

Skulto stores data in `~/.skulto/`:

| Path | Purpose |
| --- | --- |
| `~/.skulto/skulto.db` | SQLite database |
| `~/.skulto/repositories/` | Cloned git repositories |

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
├── internal/
│   ├── cli/                 # Cobra CLI commands (add, list, pull, etc.)
│   ├── config/              # Configuration (env vars only)
│   ├── db/                  # GORM + SQLite + FTS5 database layer
│   ├── detect/              # AI tool detection on system
│   ├── embedding/           # Embedding provider abstraction
│   ├── installer/           # Skill installation via symlinks
│   ├── llm/                 # LLM provider abstraction
│   ├── log/                 # Structured logging
│   ├── migration/           # Database migrations
│   ├── models/              # Data structures (Skill, Tag, Source, etc.)
│   ├── scraper/             # GitHub scraping (git clone based)
│   ├── search/              # Search service
│   ├── security/            # Security scanner for skills
│   ├── telemetry/           # PostHog analytics (opt-in)
│   ├── testutil/            # Test utilities
│   ├── tui/                 # Bubble Tea TUI
│   │   ├── components/      # Reusable UI components
│   │   └── views/           # Screen views (home, search, detail, etc.)
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

No personal data, no skill content, no IP addresses are collected.

## Contributing

Contributions welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) first.

## License

MIT License - see [LICENSE](LICENSE) for details.

---

Made with ❤️ by [Asteroid Belt](https://github.com/asteroid-belt)
