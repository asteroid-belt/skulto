# Skulto Agent Guide

> Scope: Root project (applies to all subdirectories unless overridden)

## Quick Facts

| Item | Value |
|------|-------|
| **Primary language** | Go 1.25+ |
| **Package manager** | Go modules (`go mod`) |
| **Build tool** | Make (`make build`, `make build-mcp`) |
| **CLI framework** | [Cobra](https://github.com/spf13/cobra) + [Fang](https://github.com/charmbracelet/fang) |
| **TUI framework** | [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| **MCP framework** | [mcp-go](https://github.com/mark3labs/mcp-go) |
| **Database** | SQLite with GORM + FTS5 for full-text search |
| **CI/CD** | GitHub Actions (`.github/workflows/ci.yml`) |
| **Analytics** | PostHog (opt-out via `SKULTO_TELEMETRY_TRACKING_ENABLED=false`) |
| **Primary binaries** | `skulto` (CLI/TUI), `skulto-mcp` (MCP server) |

## Repository Tour

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
│   ├── telemetry/           # PostHog analytics (opt-out)
│   ├── testutil/            # Test utilities
│   ├── tui/                 # Bubble Tea TUI
│   │   ├── components/      # Reusable UI components (dialogs, selectors)
│   │   └── views/           # Screen views (home, search, detail, etc.)
│   └── vector/              # Vector store
├── pkg/version/             # Version info (set via ldflags)
├── scripts/                 # Build and release scripts
├── docs/                    # Documentation and plan files
│   └── plans/               # Implementation plans
├── assets/                  # Demo GIFs for README
└── .github/workflows/       # CI/CD pipelines
```

### Key Directories

| Path | Owner/Purpose |
|------|---------------|
| `internal/cli/` | CLI command implementations - each file is a Cobra command |
| `internal/tui/views/` | TUI screens - Bubble Tea models for each view |
| `internal/mcp/` | MCP server - handlers expose CLI/TUI functionality to AI agents |
| `internal/installer/` | Cross-platform skill installation - symlink management for 33 AI tools |
| `internal/telemetry/` | PostHog event tracking - events defined in `events.go`, client in `client.go` |
| `internal/db/` | Database operations - GORM models and FTS5 search |

## Tooling & Setup

### Requirements

- Go 1.25 or higher
- Make
- (Optional) `GITHUB_TOKEN` for higher GitHub API rate limits
- (Optional) `OPENAI_API_KEY` for embeddings in semantic search

### Install Dependencies

```bash
make deps
```

This downloads Go modules and installs golangci-lint to `./bin/`.

### Environment Variables

| Variable | Purpose |
|----------|---------|
| `GITHUB_TOKEN` | Higher GitHub API rate limits (optional) |
| `OPENAI_API_KEY` | Embeddings for semantic search (optional) |
| `SKULTO_TELEMETRY_TRACKING_ENABLED` | Set to `false` to disable telemetry |
| `SKULTO_POSTHOG_API_KEY` | PostHog API key (set at build time) |

### Data Directory

Skulto stores data in `~/.skulto/`:

| Path | Purpose |
|------|---------|
| `~/.skulto/skulto.db` | SQLite database |
| `~/.skulto/skulto.log` | Logfile |
| `~/.skulto/repositories/` | Cloned git repositories |
| `~/.skulto/favorites.json` | Favorite skills (persists across DB resets) |

## Common Tasks

### Build

```bash
make build           # Build skulto binary → ./build/skulto
make build-mcp       # Build MCP server → ./build/skulto-mcp
make build-all       # Build both binaries
make dev             # Development build with race detector (requires CGO)
```

### Run

```bash
./build/skulto       # Launch TUI
./build/skulto <cmd> # Run CLI command (add, install, pull, etc.)
```

### Test

```bash
make test            # Run all tests with coverage → coverage.html
make test-race       # Run tests with race detector
```

### Lint & Format

```bash
make lint            # Run golangci-lint
make format          # Format code with gofmt
make vet             # Run go vet
```

### Clean

```bash
make clean           # Remove build artifacts and coverage files
```

### Ship (Push after checks)

```bash
make ship_it         # Build, lint, test, then push (via scripts/ship-it.sh)
```

## Testing & Quality Gates

### Running Tests

Tests are located alongside source files in `*_test.go` files:

```bash
make test                     # Full test suite with coverage
go test ./internal/cli/...    # Test specific package
go test -v -run TestSearch    # Run specific test
```

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
- Feature branches should be prefixed with `feature/` or similar
- PRs require passing CI checks

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(scraper): add support for new skill format
fix(tui): correct keybinding for search
docs(readme): update installation instructions
refactor(installer): extract symlink helper
test(db): add characterization tests
chore(deps): update go-openai to v1.41
```

### Code Style

- Follow standard Go conventions
- Run `make format` before committing
- Write tests for new functionality
- Document exported functions

## Architecture Notes

### Three Interfaces, One Codebase

Skulto exposes the same functionality through three interfaces:

1. **CLI** (`internal/cli/`) - Cobra commands for scripting
2. **TUI** (`internal/tui/`) - Bubble Tea interactive UI
3. **MCP** (`internal/mcp/`) - Model Context Protocol server for AI tools

All three use shared services:
- `internal/installer/InstallService` - Unified installation
- `internal/db/` - Database operations
- `internal/telemetry/` - Analytics tracking

### Telemetry Consistency

**Important for maintainers**: The telemetry system (`internal/telemetry/`) tracks user actions across all three interfaces. When adding new user-facing features:

1. Define the event in `internal/telemetry/events.go`
2. Add the tracking method to both `posthogClient` and `noopClient`
3. Add the method signature to the `Client` interface in `client.go`
4. Call the tracking method from CLI, TUI, AND MCP handlers

Current telemetry events are organized by surface area:
- **CLI events**: `EventCLICommandExecuted`, `EventRepoAdded`, etc.
- **TUI events**: `EventViewNavigated`, `EventSkillInstalled`, etc.
- **Session events**: `EventSessionSummary`, `EventAppStarted`, `EventAppExited`

### InstallService

The `InstallService` (`internal/installer/service.go`) is the unified entry point for skill installation. It:
- Handles platform detection
- Manages symlink creation
- Tracks telemetry for installations
- Is used by CLI, TUI, and MCP

### Platform Registry

Skulto supports 33 AI platforms. The platform registry is in `internal/installer/platforms.go`. Each platform defines:
- Detection method (command, directory)
- Global and project-level skill paths
- Display name and aliases

## Documentation Duties

- Update `README.md` when features, setup steps, or CLI commands change
- Document new CLI commands with examples
- Update this `AGENTS.md` when adding new packages or significant architecture changes
- Plan documents go in `docs/plans/`

## Finish the Task Checklist

Before completing any task, verify:

- [ ] Code passes linter (`make lint`)
- [ ] Code is formatted (`make format`)
- [ ] All new code has tests
- [ ] All tests pass (`make test`)
- [ ] No new warnings introduced
- [ ] Update relevant docs (& `README.md` if significant changes)
- [ ] Summarize changes in conventional commit format

### Commit Summary Template

```
<type>(<scope>): <short description>

<optional body explaining the "why">

<optional footer with breaking changes or issue refs>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `perf`
