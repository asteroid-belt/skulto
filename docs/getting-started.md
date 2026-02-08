# Getting Started

> How to set up and run Skulto locally.

## Prerequisites

| Requirement | Version | Install |
|------------|---------|---------|
| Go | >= 1.25 | [go.dev/dl](https://go.dev/dl/) |
| Make | Any | Included with Xcode CLI tools (macOS) or `apt install make` (Linux) |
| Git | Any | Included with most OS installations |

**Optional:**

| Requirement | Purpose | How to Set |
|------------|---------|------------|
| `GITHUB_TOKEN` | Higher GitHub API rate limits when syncing repositories | `export GITHUB_TOKEN=ghp_...` |
| `OPENAI_API_KEY` | Enable semantic search via vector embeddings | `export OPENAI_API_KEY=sk-...` |

## Installation

### Homebrew (Recommended)

```bash
brew install asteroid-belt/tap/skulto
```

This installs both `skulto` (CLI/TUI) and `skulto-mcp` (MCP server).

### From Source

```bash
# Clone the repository
git clone https://github.com/asteroid-belt/skulto.git
cd skulto

# Install dependencies (Go modules + golangci-lint)
make deps

# Build both binaries
make build-all

# Binaries are in ./build/
./build/skulto
```

## Configuration

Skulto uses environment variables exclusively - there is no config file.

| Variable | Required | Description | Default |
|----------|----------|-------------|---------|
| `GITHUB_TOKEN` | No | GitHub API token for higher rate limits | None (public access) |
| `OPENAI_API_KEY` | No | Enables semantic search with OpenAI embeddings | None (FTS5 only) |
| `SKULTO_TELEMETRY_TRACKING_ENABLED` | No | Set to `false` to disable anonymous telemetry | `true` |

All data is stored in `~/.skulto/`:

| Path | Purpose |
|------|---------|
| `~/.skulto/skulto.db` | SQLite database |
| `~/.skulto/skulto.log` | Log file |
| `~/.skulto/repositories/` | Cloned git repositories |
| `~/.skulto/favorites.json` | Favorite skills (persists across DB resets) |
| `~/.skulto/skills/` | User's local skills directory |

## Running Locally

### TUI (Interactive Mode)

```bash
skulto
```

On first launch, Skulto runs an onboarding flow:

1. **Platform selection** - Detected AI tools appear at the top; select which ones to manage
2. **Skill selection** - Choose from curated starter skills
3. **Location chooser** - Pick global or project scope per platform

After onboarding, the home dashboard shows installed skills, recently viewed skills, and top tags.

### CLI (Scripted Mode)

```bash
# Add a skill repository
skulto add asteroid-belt/skills

# Search for skills
skulto install superplan

# Install from a repository URL
skulto install asteroid-belt/skills

# Pull/sync all repositories
skulto pull

# Scan skills for security threats
skulto scan --all
```

### MCP Server

To use Skulto with Claude Code or other MCP clients, add to your MCP config:

```json
{
  "mcpServers": {
    "skulto": {
      "command": "skulto-mcp",
      "type": "stdio"
    }
  }
}
```

## Verifying It Works

```bash
# Build and run tests
make test

# Or try a quick smoke test
./build/skulto --help
```

You should see a help message listing all available subcommands (add, install, pull, scan, etc.).

## Common Issues

| Problem | Cause | Solution |
|---------|-------|----------|
| `make build` fails with CGO errors | Some systems try to enable CGO | Skulto builds with `CGO_ENABLED=0` by default; ensure no override is set |
| `skulto add` times out | GitHub API rate limiting | Set `GITHUB_TOKEN` environment variable |
| Search returns no results | No repositories synced yet | Run `skulto add asteroid-belt/skills` to add a skill repository |
| MCP server not found by Claude Code | `skulto-mcp` not in PATH | Use the full path in MCP config (e.g., `/opt/homebrew/bin/skulto-mcp`) |

## Next Steps

- [Development](development.md) - Learn how to contribute
- [Architecture](architecture.md) - Understand the system design
