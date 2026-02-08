# Architecture Decision Records

> Significant technical decisions made in Skulto, recorded as individual ADRs.

ADRs capture the **context**, **decision**, and **consequences** of choices that
shape the project's architecture. They are mined from git history, plan files,
code comments, and project structure.

## ADR Index

| # | Title | Status | Date |
|---|-------|--------|------|
| [0001](0001-symlink-based-skill-installation.md) | Symlink-based skill installation | Accepted | 2026-01-01 |
| [0002](0002-pure-go-sqlite-with-fts5.md) | Pure-Go SQLite with FTS5 | Accepted | 2026-01-01 |
| [0003](0003-three-interfaces-shared-services.md) | Three interfaces with shared services | Accepted | 2026-01-01 |
| [0004](0004-environment-variable-only-configuration.md) | Environment-variable-only configuration | Accepted | 2026-01-01 |
| [0005](0005-git-clone-over-github-api.md) | Git clone over GitHub API for repository sync | Accepted | 2026-01-01 |
| [0006](0006-posthog-for-telemetry.md) | PostHog for anonymous telemetry | Accepted | 2026-01-01 |
| [0007](0007-skill-installations-table-over-boolean-flag.md) | Skill installations table over boolean flag | Accepted | 2026-01-01 |

## Status Definitions

| Status | Meaning |
|--------|---------|
| **Proposed** | Under discussion, not yet adopted |
| **Accepted** | Active and in effect |
| **Superseded** | Replaced by a newer ADR (link to replacement) |
| **Deprecated** | No longer relevant |

## Related Documentation

- [Architecture](../architecture.md) - System design context
- [Development](../development.md) - Contributing workflow
- [Overview](../overview.md) - Project purpose and scope
