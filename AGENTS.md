# Skulto Agent Guide

> Scope: Root project (applies to all subdirectories unless overridden)

Skulto is an offline-first tool for syncing and managing agent skills. It scrapes GitHub for AI skill files, indexes them with SQLite FTS5, provides a Bubble Tea TUI, and translates skills between 6 AI platforms.

## Quick Facts

| Attribute | Value |
|-----------|-------|
| **Primary language** | Go 1.25+ |
| **Package manager** | Go modules |
| **Build system** | Make |
| **Database** | SQLite with FTS5 (GORM) |
| **TUI framework** | Bubble Tea + Lip Gloss |
| **CI/CD** | GitHub Actions |
| **Data location** | `~/.skulto/` (simplified from XDG) |

## Repository Tour

```
skulto/
├── cmd/skulto/           # Main CLI entry point
├── internal/             # Private packages (see below)
│   ├── cli/              # Cobra CLI commands (add, list, pull, etc.)
│   ├── config/           # Configuration (env vars only, no config file)
│   ├── db/               # GORM + SQLite + FTS5 database layer
│   ├── detect/           # AI tool detection on system
│   ├── embedding/        # Embedding provider abstraction
│   ├── installer/        # Skill installation via symlinks
│   ├── llm/              # LLM provider abstraction (Anthropic, OpenAI, OpenRouter)
│   ├── log/              # Structured logging
│   ├── migration/        # Database migrations
│   ├── models/           # Data structures (Skill, Tag, Source, etc.)
│   ├── scraper/          # GitHub scraping (git clone + REST API)
│   ├── search/           # Search service + background indexer
│   ├── security/         # Security scanner for skills
│   ├── skillgen/         # Skill generation utilities
│   ├── telemetry/        # PostHog analytics (opt-in)
│   ├── testutil/         # Test utilities (SkipAITests helper)
│   ├── tui/              # Bubble Tea TUI
│   │   ├── components/   # Reusable UI components
│   │   └── views/        # Screen views (home, search, detail, etc.)
│   └── vector/           # Vector store (chromem-go)
├── pkg/version/          # Version info (set via ldflags)
├── scripts/              # Build and release scripts
├── docs/                 # Planning documents
│   ├── completed/        # Finished implementation plans
│   └── in-review/        # Plans under review
└── .github/workflows/    # CI/CD pipelines
```

## Tooling & Setup

### Prerequisites

- **Go 1.25+** (check with `go version`)
- **Make** (for build automation)
- **asdf** (optional, uses `.tool-versions`)

### Environment Variables

Configuration is done entirely via environment variables (no config file).

| Variable | Purpose | Required |
|----------|---------|----------|
| `GITHUB_TOKEN` | Higher GitHub API rate limits | Optional |
| `OPENAI_API_KEY` | Embeddings for semantic search | Optional |
| `ANTHROPIC_API_KEY` | LLM provider | Optional |
| `OPENROUTER_API_KEY` | Alternative LLM provider | Optional |
| `RUN_AI_TESTS` | Enable AI-dependent tests | Test only |
| `SKULTO_TELEMETRY_TRACKING_ENABLED` | Set to `false` to disable telemetry (enabled by default) | Optional |

### Initial Setup

```bash
# Install dependencies + golangci-lint
make deps

# Build the binary
make build

# Run the TUI
./build/skulto
```

## Common Tasks

| Task | Command | Description |
|------|---------|-------------|
| Build | `make build` | Production build to `./build/skulto` |
| Dev build | `make dev` | Build with race detector (requires CGO) |
| Run | `make run` | Build and run TUI |
| Test | `make test` | Run all tests |
| Test + race | `make test-race` | Tests with race detector |
| Coverage | `make test-coverage` | Generate `coverage.html` |
| Lint | `make lint` | Run golangci-lint |
| Format | `make format` | Run `go fmt` |
| Clean | `make clean` | Remove build artifacts |
| Ship | `make ship_it` | Build, lint, test, then push |

### Running Specific Tests

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/db/...

# Run AI-dependent tests (requires API keys)
RUN_AI_TESTS=1 go test ./...

# Run with verbose output
go test -v ./internal/config/...
```

## CLI Subcommands

Skulto provides both an interactive TUI and CLI subcommands for scripting.

| Command | Purpose |
|---------|---------|
| `skulto` | Launch the interactive Bubble Tea TUI |
| `skulto add <repo>` | Add a skill repository and sync its skills |
| `skulto list` | List all configured source repositories with skill counts |
| `skulto pull` | Pull/sync all skill repositories and reconcile installed skills |
| `skulto remove [repo]` | Remove a skill repository (uninstalls skills, deletes from DB and disk) |
| `skulto scan` | Scan skills for security threats (prompt injection, dangerous patterns) |
| `skulto update` | Combined `pull` + `scan` with enhanced change reporting |
| `skulto info <slug>` | Show detailed information about a specific skill |

### Command Details

**`skulto add <repo>`** - Add a skill repository
- Parses GitHub URLs: `owner/repo`, `https://github.com/owner/repo`, `git@github.com:owner/repo.git`
- Clones repository and scans for skills
- Flag: `--no-sync` to defer cloning

**`skulto pull`** - Sync all repositories
- Clones/updates all registered repositories
- Scans AI tool directories to detect installed skills
- Reconciles database state with filesystem

**`skulto remove [repo]`** - Remove a repository
- Interactive selection if no repo specified
- Uninstalls all skills (removes symlinks)
- Deletes skill records, source record, and git clone
- Flag: `--force` to skip confirmation

**`skulto scan`** - Security scanning
- Scans for prompt injection and dangerous code patterns
- Flags: `--all`, `--skill <id>`, `--source <owner/repo>`, `--pending`
- Reports threat levels: CRITICAL, HIGH, MEDIUM, LOW

**`skulto update`** - Pull + scan with reporting
- Phase 1: Pull repositories
- Phase 2: Security scan on new/updated skills
- Phase 3: Summary report with change details
- Flag: `--scan-all` to scan all skills

## Testing & Quality Gates

### Test Categories

1. **Unit tests** - Fast, no external dependencies
2. **Integration tests** - May require network (git clone tests)
3. **AI tests** - Require API keys, gated by `RUN_AI_TESTS=1`

### AI Test Guard

Tests requiring LLM/embedding API keys use `testutil.SkipAITests(t)`:

```go
func TestSomethingWithAI(t *testing.T) {
    testutil.SkipAITests(t)  // Skips unless RUN_AI_TESTS=1

    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        t.Skip("OPENAI_API_KEY not set")
    }
    // ...
}
```

### CI Pipeline

The CI workflow (`.github/workflows/ci.yml`) runs:
1. `make deps` - Install dependencies
2. `make lint` - Linting with golangci-lint
3. `make test` - All tests
4. Cross-platform builds (linux/darwin, amd64/arm64)

### Coverage Expectations

- Aim for meaningful coverage on business logic
- Characterization tests exist for installer behavior
- Integration tests cover scraper and search pipelines

## Workflow Expectations

### Branching

- `main` is the primary branch
- Feature branches: `feature/<name>`
- Bug fixes: `fix/<name>`

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(scraper): add support for new skill format
fix(tui): correct keybinding for search
docs(readme): update installation instructions
refactor(installer): extract symlink helper
test(db): add characterization tests
chore(deps): update dependencies
```

### Pre-Push Checklist

1. `make format` - Format code
2. `make lint` - Pass linting
3. `make test` - Pass all tests
4. `make build` - Verify build succeeds

Or use: `make ship_it` (runs build, lint, test, then pushes)

## Code Conventions

### Package Structure

- `internal/` packages are private to the module
- Each package has a primary file matching the package name (e.g., `db/db.go`)
- Tests are co-located with source files (`*_test.go`)

### Error Handling

- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
- Return early on errors
- Use `require.NoError(t, err)` in tests

### Configuration

- All config flows through `internal/config`
- Configuration is read from environment variables only (no config file)
- `Config.BaseDir` is the root for all data (`~/.skulto/`)
- Repositories are cloned to `~/.skulto/repositories/`
- Use `config.GetPaths(cfg)` for derived paths

### Database

- GORM with SQLite backend
- FTS5 for full-text search on skills
- Models in `internal/models/`
- Database operations in `internal/db/`

## Documentation Duties

### When to Update README.md

- New user-facing features
- Changed CLI commands or flags
- Updated installation steps
- Modified keybindings

## Finish the Task Checklist

After completing any task:

- [ ] Run `make format` and `make lint`
- [ ] Run `make test` to verify no regressions
- [ ] Update `README.md` and `AGENTS.md` if user-facing/project-changes changes were made
- [ ] Update relevant docs if architecture changed
- [ ] Commit with conventional commit format

### Commit Template

```
<type>(<scope>): <description>

[optional body]
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`
