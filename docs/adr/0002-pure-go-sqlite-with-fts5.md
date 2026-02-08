# ADR-0002: Pure-Go SQLite with FTS5

**Status:** Accepted
**Date:** 2026-01-01

## Context

Skulto needs a local database for skill metadata, installation tracking, tags, and full-text search. The database must work without external services (offline-first), support full-text search with relevance ranking, and enable easy cross-compilation to multiple OS/architecture targets.

Standard SQLite Go drivers (mattn/go-sqlite3) require CGO, which complicates cross-compilation and CI builds. The project builds for 4 targets: linux-amd64, linux-arm64, darwin-amd64, darwin-arm64.

## Decision

Use the pure-Go SQLite driver (`glebarez/sqlite`) with GORM as the ORM. This driver includes FTS5 support without requiring CGO. The Makefile sets `CGO_ENABLED=0` for production builds.

FTS5 is used for full-text search with BM25 ranking. A virtual table (`skills_fts`) is synchronized via database triggers on insert/update/delete operations on the `skills` table.

## Consequences

### Benefits

- Cross-compilation works with a single `go build` command, no C compiler needed
- CI pipeline is simpler: no CGO toolchain setup required
- FTS5 provides fast, relevance-ranked search (~50ms latency) with built-in BM25 scoring
- Automatic index synchronization via triggers eliminates manual index management
- Single-file database simplifies deployment and data management

### Trade-offs

- Pure-Go SQLite is slower than the C implementation for some operations
- GORM adds abstraction overhead compared to raw SQL
- Development builds with race detection (`make dev`) still require `CGO_ENABLED=1`

### Alternatives Considered

| Alternative | Why Not Chosen |
|-------------|---------------|
| mattn/go-sqlite3 (CGO) | Requires C compiler, complicates cross-compilation and CI |
| PostgreSQL / MySQL | Requires external service, contradicts offline-first design |
| bbolt / BadgerDB | No built-in full-text search capability |
| Bleve / Tantivy | Separate search index to maintain alongside primary data store |

## Sources

> Evidence used to reconstruct this decision.

| Source Type | Reference |
|-------------|-----------|
| Code comment | `internal/db/db.go:2` - "It uses the pure-Go SQLite driver with FTS5 support" |
| Code | `internal/db/db.go` - `setupFTS()` creates `skills_fts` virtual table with FTS5 |
| Config file | `Makefile:14` - `CGO_ENABLED=0` |
| Dependency | `go.mod` - `github.com/glebarez/sqlite v1.11.0` |

## Related

- [Architecture](../architecture.md) - Database component
- [Glossary](../glossary.md) - FTS5, BM25 definitions
