# Skulto Agent Guide

> Scope: Root project (applies to all subdirectories unless overridden)

## Quick Facts

| Item | Value |
|------|-------|
| **Primary language** | Go 1.25+ |
| **Package manager** | Go modules (`go mod`) |
| **Build tool** | Make (`make build`, `make build-mcp`, `make build-all`) |
| **CLI framework** | [Cobra](https://github.com/spf13/cobra) + [Fang](https://github.com/charmbracelet/fang) |
| **TUI framework** | [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| **MCP framework** | [mcp-go](https://github.com/mark3labs/mcp-go) v0.27+ |
| **Database** | SQLite + GORM + FTS5 full-text search |
| **Vector search** | chromem-go (optional, for semantic search) |
| **Analytics** | PostHog (opt-out via `SKULTO_TELEMETRY_TRACKING_ENABLED=false`) |
| **Primary binaries** | `skulto` (CLI/TUI), `skulto-mcp` (MCP server) |
| **CI/CD** | GitHub Actions (`.github/workflows/ci.yml`) |
| **Test command** | `make test` (coverage report: `coverage.html`) |
| **Lint command** | `make lint` (golangci-lint v2.7.2) |

## Repository Tour

```
skulto/
├── cmd/
│   ├── skulto/              # Main CLI/TUI entry point
│   └── skulto-mcp/          # MCP server binary (JSON-RPC 2.0 over stdio)
├── internal/
│   ├── cli/                 # Cobra CLI commands (add, install, pull, scan, etc.)
│   │   └── prompts/         # Interactive CLI prompts (platform selector)
│   ├── config/              # Configuration (env vars only, no config file)
│   ├── db/                  # GORM + SQLite + FTS5 database layer
│   ├── detect/              # AI tool detection on system (command/dir checks)
│   ├── discovery/           # Skill discovery and ingestion from local dirs
│   ├── embedding/           # OpenAI embedding provider abstraction
│   ├── favorites/           # File-based favorites persistence (~/.skulto/favorites.json)
│   ├── installer/           # Skill installation via symlinks (33 platforms)
│   ├── llm/                 # LLM provider abstraction (Anthropic, OpenAI, OpenRouter)
│   ├── log/                 # Structured logging
│   ├── mcp/                 # MCP server implementation (tools + resources)
│   ├── migration/           # Database migrations
│   ├── models/              # Data structures (Skill, Tag, Source, Security, etc.)
│   ├── scraper/             # GitHub scraping (git clone based, shallow clones)
│   ├── search/              # Hybrid search service (FTS5 + semantic)
│   ├── security/            # Security scanner (prompt injection detection)
│   ├── skillgen/            # Local skill scanning (~/.skulto/skills, ./.skulto/skills)
│   ├── telemetry/           # PostHog analytics (opt-in events defined in events.go)
│   ├── testutil/            # Test utilities and helpers
│   ├── tui/                 # Bubble Tea TUI application
│   │   ├── components/      # Reusable UI components (dialogs, selectors)
│   │   ├── design/          # Design constants and skull ASCII art
│   │   └── views/           # Screen views (home, search, detail, onboarding, manage)
│   └── vector/              # Vector store abstraction for semantic search
├── pkg/version/             # Version info (set via ldflags at build time)
├── scripts/                 # Build and release scripts (ship-it.sh, release.sh)
├── docs/                    # Project documentation (overview, architecture, ADRs, glossary)
├── context/                 # Deep-dive technical documentation (see below)
├── plans/                   # Implementation plans (agent-detection, mcp-plan)
├── assets/                  # Demo GIFs for README
└── .github/workflows/       # CI/CD pipelines
```

### Key Directories by Function

| Path | Owner/Purpose |
|------|---------------|
| `internal/cli/` | CLI command implementations - each file is a Cobra command |
| `internal/tui/views/` | TUI screens - Bubble Tea models for each view |
| `internal/mcp/` | MCP server - handlers expose CLI/TUI functionality to AI agents |
| `internal/installer/` | Cross-platform skill installation - symlink management for 33 AI tools |
| `internal/telemetry/` | PostHog event tracking - events in `events.go`, client in `client.go` |
| `internal/db/` | Database operations - GORM models and FTS5 search queries |
| `internal/security/` | Security scanner - regex patterns for prompt injection detection |
| `internal/scraper/` | Repository manager - git clone/fetch, skill file parsing |

## Architecture Overview

### Three Interfaces, One Codebase

Skulto exposes identical functionality through three interfaces that share services:

```
┌─────────────────────────────────────────────────────────────────┐
│                        User Interfaces                          │
├─────────────────┬─────────────────┬─────────────────────────────┤
│   CLI (Cobra)   │   TUI (Bubble)  │   MCP (mcp-go over stdio)   │
│   cmd/skulto    │   internal/tui  │   cmd/skulto-mcp            │
├─────────────────┴─────────────────┴─────────────────────────────┤
│                      Shared Services                            │
├─────────────────────────────────────────────────────────────────┤
│  InstallService (installer/)  │  SearchService (search/)        │
│  Scraper (scraper/)           │  SecurityScanner (security/)    │
│  Database (db/)               │  Telemetry (telemetry/)         │
└─────────────────────────────────────────────────────────────────┘
```

**See:** [context/architecture.md](context/architecture.md) for detailed component interactions.

### Platform Support (33 AI Coding Tools)

Skulto detects and installs skills to platforms defined in `internal/installer/platform.go`:

- **Original 6**: Claude Code, Cursor, Windsurf, GitHub Copilot, OpenAI Codex, OpenCode
- **Extended 27**: Amp, Kimi CLI, Antigravity, Moltbot, Cline, Roo Code, Gemini CLI, Kiro CLI, Continue, Goose, Junie, Kilo Code, MCPJam, Mux, OpenHands, Pi, Qoder, Qwen Code, Trae, Zencoder, Neovate, Pochi, CodeBuddy, Command Code, Crush, Droid, Kode

Each platform defines: command name, project dir, global dir, and platform-specific paths.

**See:** [context/platforms.md](context/platforms.md) for platform registry details.

### Security Scanner

The security scanner (`internal/security/`) detects prompt injection threats:

| Category | Severity | Examples |
|----------|----------|----------|
| Instruction Override | HIGH | "ignore previous instructions", "disregard rules" |
| Jailbreak | CRITICAL | DAN jailbreak, developer mode, unrestricted AI |
| Data Exfiltration | HIGH | Requests to leak system prompts or context |
| Dangerous Commands | MEDIUM-HIGH | Shell execution, file operations |

Scoring uses base severity weights with context mitigation (e.g., educational content reduces score).

**See:** [context/security.md](context/security.md) for pattern details.

### Telemetry System

Events are tracked consistently across all interfaces:

1. **Define event** in `internal/telemetry/events.go`
2. **Add method** to both `posthogClient` and `noopClient`
3. **Add signature** to `Client` interface in `client.go`
4. **Call from** CLI, TUI, AND MCP handlers

Current event categories: CLI, TUI, Session, Shared (cross-interface), MCP-specific.

**See:** [context/telemetry.md](context/telemetry.md) for event catalog.

## Tooling & Setup

### Requirements

- Go 1.25 or higher
- Make
- (Optional) `GITHUB_TOKEN` for higher GitHub API rate limits
- (Optional) `OPENAI_API_KEY` for semantic search embeddings

### Install Dependencies

```bash
make deps
```

Downloads Go modules, runs `go mod tidy`, and installs golangci-lint to `./bin/`.

### Environment Variables

| Variable | Purpose |
|----------|---------|
| `GITHUB_TOKEN` | Higher GitHub API rate limits |
| `OPENAI_API_KEY` | Embeddings for semantic search |
| `SKULTO_TELEMETRY_TRACKING_ENABLED` | Set to `false` to disable telemetry |
| `SKULTO_POSTHOG_API_KEY` | PostHog API key (set at build time via ldflags) |

### Data Directory

Skulto stores data in `~/.skulto/`:

| Path | Purpose |
|------|---------|
| `~/.skulto/skulto.db` | SQLite database |
| `~/.skulto/skulto.log` | Logfile |
| `~/.skulto/repositories/{owner}/{repo}/` | Cloned git repositories |
| `~/.skulto/favorites.json` | Favorite skills (persists across DB resets) |
| `~/.skulto/skills/` | User's local skills directory |

## Common Tasks

### Build

```bash
make build           # Build skulto binary → ./build/skulto
make build-mcp       # Build MCP server → ./build/skulto-mcp
make build-all       # Build both binaries
make dev             # Development build with race detector (requires CGO_ENABLED=1)
```

### Run

```bash
./build/skulto           # Launch TUI (default)
./build/skulto <cmd>     # Run CLI command
./build/skulto-mcp       # Run MCP server (stdio)
```

### Test

```bash
make test            # Run all tests with coverage → coverage.html
make test-race       # Run tests with race detector (CGO_ENABLED=1)
go test ./internal/cli/...    # Test specific package
go test -v -run TestSearch    # Run specific test
```

### Lint & Format

```bash
make lint            # Run golangci-lint (5m timeout)
make format          # Format code with gofmt
make vet             # Run go vet
```

### Clean

```bash
make clean           # Remove build artifacts, coverage files, and release dir
```

### Ship (Pre-push Validation)

```bash
make ship_it         # Build, lint, test, then push (via scripts/ship-it.sh)
```

## Testing & Quality Gates

### Test Organization

Tests are co-located with source files in `*_test.go` files:

- Unit tests: `foo_test.go` alongside `foo.go`
- Characterization tests: `*_characterization_test.go` for snapshot-style testing
- Integration tests: `*_integration_test.go` (may require network)

### Coverage

After running `make test`:
- `coverage.out` - Raw coverage data
- `coverage.html` - HTML report (open in browser)

### CI Expectations

CI (`.github/workflows/ci.yml`) runs on every push to `main` and all PRs:

1. **Lint** - `make lint` (golangci-lint)
2. **Test** - `make test` with coverage
3. **Build matrix** - Cross-compile for linux/darwin on amd64/arm64

All CI checks must pass before merging.

## Workflow Expectations

### Branching

- `main` is the primary branch
- Feature branches: `feature/description` or `topic/description`
- PRs require passing CI checks

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(scraper): add support for nested skill directories
fix(tui): correct keybinding conflict for search
docs(readme): update MCP server configuration
refactor(installer): extract platform detection to service
test(db): add FTS5 ranking characterization tests
chore(deps): update go-openai to v1.41
perf(search): cache FTS query preparation
```

**Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `perf`

**Scopes**: `cli`, `tui`, `mcp`, `db`, `installer`, `scraper`, `search`, `security`, `telemetry`, `deps`

### Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Run `make format` before committing
- Write tests for new functionality
- Document exported functions
- Keep functions focused and small

## Documentation

### Project Documentation (`docs/`)

| File | Topic |
|------|-------|
| [docs/overview.md](docs/overview.md) | Project identity, purpose, key features, status |
| [docs/architecture.md](docs/architecture.md) | System design, components, data flow, external dependencies |
| [docs/getting-started.md](docs/getting-started.md) | Prerequisites, installation, configuration, first run |
| [docs/development.md](docs/development.md) | Repository structure, workflow, testing, CI/CD, distribution |
| [docs/adr/README.md](docs/adr/README.md) | Architecture Decision Records index |
| [docs/glossary.md](docs/glossary.md) | Domain terminology and acronyms |

### Deep-Dive Documentation (`context/`)

The `context/` directory contains detailed technical documentation:

| File | Topic |
|------|-------|
| [context/architecture.md](context/architecture.md) | System architecture, data flow, component interactions |
| [context/platforms.md](context/platforms.md) | Platform registry, detection logic, installation paths |
| [context/security.md](context/security.md) | Security scanner patterns, scoring, threat levels |
| [context/telemetry.md](context/telemetry.md) | Event catalog, tracking implementation, opt-out |
| [context/database.md](context/database.md) | Schema, FTS5 setup, GORM models, migrations |
| [context/mcp-server.md](context/mcp-server.md) | MCP tools, resources, handler implementation |

## Documentation Duties

- Update `README.md` when features, setup steps, or CLI commands change
- Document new CLI commands with examples
- Update this `AGENTS.md` when adding new packages or significant architecture changes
- Project-level docs (overview, architecture, getting started, development) go in `docs/`
- Architecture Decision Records go in `docs/adr/`
- Plan documents go in `plans/`
- Deep-dive technical docs go in `context/`

## Finish the Task Checklist

Before completing any task, verify:

- [ ] Code passes linter (`make lint`)
- [ ] Code is formatted (`make format`)
- [ ] All new code has tests
- [ ] All tests pass (`make test`)
- [ ] No new warnings introduced
- [ ] Telemetry events added for new user-facing features (if applicable)
- [ ] Update relevant docs (& `README.md` if significant changes)
- [ ] Summarize changes in conventional commit format

### Commit Summary Template

```
<type>(<scope>): <short description>

<optional body explaining the "why">

<optional footer with breaking changes or issue refs>
```
