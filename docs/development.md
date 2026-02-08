# Development Guide

> How to contribute to Skulto.

## Repository Structure

```text
skulto/
├── cmd/
│   ├── skulto/                 # Main CLI/TUI binary entry point
│   └── skulto-mcp/             # MCP server binary entry point
├── internal/
│   ├── cli/                    # Cobra CLI commands (one file per command)
│   │   └── prompts/            # Interactive CLI prompts (platform/scope selectors)
│   ├── config/                 # Configuration (env vars only, no config file)
│   ├── constants/              # Application-wide constants
│   ├── db/                     # GORM + SQLite + FTS5 database layer
│   ├── detect/                 # AI tool detection on system
│   ├── discovery/              # Skill discovery and ingestion from local dirs
│   ├── embedding/              # OpenAI embedding provider
│   ├── favorites/              # File-based favorites persistence
│   ├── hash/                   # Truncated SHA256 hash utility
│   ├── installer/              # Cross-platform skill installation (33 platforms)
│   ├── llm/                    # LLM provider abstraction (Anthropic, OpenAI, OpenRouter)
│   ├── log/                    # Structured logging
│   ├── mcp/                    # MCP server implementation (tools + resources)
│   ├── migration/              # Database migrations
│   ├── models/                 # Data structures (Skill, Tag, Source, Security, etc.)
│   ├── scraper/                # GitHub scraping (git clone, skill parsing, tag extraction)
│   ├── search/                 # Hybrid search (FTS5 + semantic)
│   ├── security/               # Security scanner (prompt injection detection)
│   ├── skillgen/               # Local skill scanning and AI CLI execution
│   ├── telemetry/              # PostHog analytics (opt-in events)
│   ├── testutil/               # Test utilities and helpers
│   ├── tui/                    # Bubble Tea TUI application
│   │   ├── components/         # Reusable UI components (dialogs, selectors)
│   │   ├── design/             # Design constants, theme, ASCII art
│   │   └── views/              # Screen views (home, search, detail, onboarding, etc.)
│   └── vector/                 # Vector store for semantic search (chromem-go)
├── pkg/version/                # Version info (set via ldflags at build time)
├── scripts/                    # Build and release scripts
├── context/                    # Deep-dive technical documentation
├── plans/                      # Implementation plans
├── assets/                     # Demo GIFs for README
└── .github/workflows/          # CI/CD pipelines
```

| Directory | Purpose |
|-----------|---------|
| `internal/cli/` | Each file is a Cobra command (add.go, install.go, pull.go, etc.) |
| `internal/tui/views/` | Each file is a Bubble Tea view (home.go, search.go, detail.go, etc.) |
| `internal/mcp/` | MCP server handlers expose CLI/TUI functionality to AI agents |
| `internal/installer/` | Platform registry and symlink management for 33 AI tools |
| `internal/db/` | GORM operations organized by entity (skills.go, tags.go, sources.go) |
| `internal/security/` | Regex patterns for prompt injection detection with scoring |
| `internal/scraper/` | Git clone, skill parsing, tag extraction, license detection |
| `context/` | Deep-dive technical docs (architecture, platforms, security, telemetry, database, MCP) |

## Development Workflow

### Making Changes

1. Branch from `main` using the convention `feature/description` or `topic/description`
2. Make changes and ensure tests pass
3. Commit using [Conventional Commits](https://www.conventionalcommits.org/)
4. Open a PR against `main` - all CI checks must pass

### Code Style

| Aspect | Convention | Enforced By |
|--------|-----------|-------------|
| Formatting | `gofmt` standard | `make format` |
| Linting | golangci-lint rules | `make lint` (v2.7.2, 5m timeout) |
| Vetting | `go vet` checks | `make vet` |
| Naming | Standard Go conventions | Manual review |

```bash
# Format code
make format

# Lint code
make lint

# Vet code
make vet
```

### Commit Messages

Follow Conventional Commits:

```text
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

## Testing

### Test Structure

| Type | Location | Command |
|------|----------|---------|
| Unit | `*_test.go` alongside source files | `make test` |
| Characterization | `*_characterization_test.go` | `make test` |
| Integration | `*_integration_test.go` (may require network) | `make test` |
| Race detection | All tests with `-race` flag | `make test-race` |

```bash
# Run all tests with coverage
make test

# Run with race detector (requires CGO_ENABLED=1)
make test-race

# Run specific package
go test ./internal/cli/...

# Run specific test
go test -v -run TestSearch ./internal/db/...
```

### Test Patterns

Tests are co-located with source files. Common patterns:

- **Test databases**: `testDB(t)` helper creates a temp SQLite DB that is cleaned up automatically
- **Table-driven tests**: Standard Go pattern used throughout
- **Test helpers**: Shared utilities in `internal/testutil/`
- **Cleanup**: Tests use `t.Cleanup()` for deferred resource release

### Coverage

After running `make test`:

- `coverage.out` - Raw coverage data
- `coverage.html` - HTML report (open in browser)

## Building

```bash
# Production build (CGO_ENABLED=0)
make build              # → ./build/skulto

# Build MCP server
make build-mcp          # → ./build/skulto-mcp

# Build both binaries
make build-all

# Development build with race detector
make dev                # Requires CGO_ENABLED=1

# Cross-compile for release
make release GOOS=linux GOARCH=amd64
```

Version, commit hash, and build date are injected via ldflags. The PostHog API key is also injected at build time.

## CI/CD

| Stage | Trigger | What It Does |
|-------|---------|-------------|
| **CI** | Push to `main`, all PRs | Lint (`make lint`), test with coverage (`make test`), cross-compile build matrix (linux/darwin, amd64/arm64) |
| **Release** | Version tag (`v*`) or manual dispatch | Validate semver, run tests, build all platforms, create GitHub Release with tarballs and checksums |

### CI Pipeline (`.github/workflows/ci.yml`)

1. Set up Go 1.25
2. Cache golangci-lint v2.7.2
3. Install dependencies (`make deps`)
4. Run linter (`make lint`)
5. Run tests with coverage (`make test`)
6. Generate coverage summary in GitHub step summary
7. Build matrix: linux/darwin x amd64/arm64

### Release Pipeline (`.github/workflows/release.yml`)

1. Validate semver format (supports pre-release and metadata)
2. Run full test suite
3. Build for all 4 platform targets
4. Create tarballs with both `skulto` and `skulto-mcp`
5. Generate SHA256 checksums
6. Create GitHub Release with release notes and assets

### Distribution

- **Homebrew**: `brew install asteroid-belt/tap/skulto`
- **GitHub Releases**: Tarballs for linux-amd64, linux-arm64, darwin-amd64, darwin-arm64

## Pre-Push Validation

```bash
# Build, lint, test, then push
make ship_it
```

This runs `scripts/ship-it.sh` which ensures all checks pass before pushing.

## Related Documentation

- [Getting Started](getting-started.md) - Initial setup
- [Architecture](architecture.md) - System design context
