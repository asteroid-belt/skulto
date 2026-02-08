# ADR-0003: Three Interfaces with Shared Services

**Status:** Accepted
**Date:** 2026-01-01

## Context

Skulto needs to serve three distinct use cases: scripted automation (CI, shell scripts), interactive browsing (developers exploring skills), and programmatic access from AI assistants. Each use case has different interaction patterns, but all operate on the same data and business logic.

## Decision

Skulto exposes three interfaces - CLI, TUI, and MCP server - that share a common service layer. The CLI and TUI are compiled into a single binary (`skulto`); the MCP server is a separate binary (`skulto-mcp`). Both binaries initialize the same way (config, database, telemetry) and use the same internal packages.

- **CLI** (Cobra): subcommand-based interface for scripting and automation
- **TUI** (Bubble Tea): interactive terminal UI launched when no subcommand is given
- **MCP** (mcp-go): JSON-RPC 2.0 over stdio for AI tool integration

## Consequences

### Benefits

- Feature parity: every operation available in one interface is available in all three
- Code reuse: business logic is implemented once in service packages
- Consistent data: all interfaces use the same database and models
- Telemetry coverage: events are tracked uniformly across interfaces

### Trade-offs

- Two separate binaries to build and distribute
- MCP server runs as a long-lived process while CLI commands are one-shot
- TUI state management adds complexity that CLI and MCP do not need

### Alternatives Considered

| Alternative | Why Not Chosen |
|-------------|---------------|
| CLI only | No interactive browsing; poor discoverability for new users |
| Web UI | Requires a server process; heavier dependency footprint |
| Single binary for all three | MCP server needs to run as a daemon; mixing with CLI/TUI adds complexity |

## Sources

> Evidence used to reconstruct this decision.

| Source Type | Reference |
|-------------|-----------|
| Code | `cmd/skulto/main.go` - CLI/TUI entry point initializes config, db, telemetry |
| Code | `cmd/skulto-mcp/main.go` - MCP entry point follows same initialization pattern |
| Code | `AGENTS.md:79-96` - Architecture diagram showing three interfaces sharing services |
| Git commit | `f5b762b` - "feat(mcp): add MCP server for Claude Code integration" |

## Related

- [Architecture](../architecture.md) - System overview diagram
