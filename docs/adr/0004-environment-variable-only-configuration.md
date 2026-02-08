# ADR-0004: Environment-Variable-Only Configuration

**Status:** Accepted
**Date:** 2026-01-01

## Context

Skulto needs configuration for API keys (GitHub, OpenAI, Anthropic), telemetry opt-out, and potentially other settings. Configuration could be managed via a config file (YAML/TOML/JSON), environment variables, or a combination.

The application is a CLI tool that should be simple to set up and not require users to manage yet another config file alongside their many AI tool configurations.

## Decision

All configuration is read from environment variables. There is no config file. The `config.Load()` function reads `os.Getenv()` for each setting and applies defaults.

Key environment variables:
- `GITHUB_TOKEN` - GitHub API access
- `OPENAI_API_KEY` - Embeddings for semantic search
- `ANTHROPIC_API_KEY` - LLM provider for skill generation
- `SKULTO_TELEMETRY_TRACKING_ENABLED` - Telemetry opt-out

The base data directory (`~/.skulto/`) is hardcoded rather than configurable.

## Consequences

### Benefits

- Zero-config setup: `brew install` and run, no config file creation needed
- Familiar pattern: developers already use env vars for API keys
- No file management: no config file to create, find, parse, or keep in sync
- CI/CD friendly: environment variables are the standard configuration mechanism

### Trade-offs

- No way to persist complex settings without environment variable management tools
- The `Paths` struct in `config/paths.go` still references a `Config` file path field (unused)
- Users managing many settings must set them in their shell profile

### Alternatives Considered

| Alternative | Why Not Chosen |
|-------------|---------------|
| YAML/TOML config file | Additional file for users to manage; overkill for a few API keys |
| Mixed (env vars + config file) | Complexity of precedence rules; confusing when both are set |
| XDG-based config | More complex path resolution for minimal benefit |

## Sources

> Evidence used to reconstruct this decision.

| Source Type | Reference |
|-------------|-----------|
| Code | `internal/config/config.go:82` - `Load()` reads exclusively from `os.Getenv()` |
| Code comment | `AGENTS.md:33` - "Configuration (env vars only, no config file)" |
| Code | `internal/config/paths.go` - hardcoded `~/.skulto` base directory |

## Related

- [Getting Started](../getting-started.md) - Configuration section
