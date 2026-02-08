# ADR-0007: Skill Installations Table Over Boolean Flag

**Status:** Accepted
**Date:** 2026-01-01

## Context

Originally, skill installation was tracked by a boolean `is_installed` field on the `skills` table. This approach could not answer: "Where is this skill installed?" (which platforms, which scopes, which paths). With 33 platforms and 2 scopes per platform, the boolean was insufficient for the manage/uninstall workflows.

## Decision

Introduce a `skill_installations` table with composite tracking: `(skill_id, platform, scope, base_path)`. Each row represents one installed location with its symlink path and timestamp. The `is_installed` boolean on the skills table is maintained as a derived field but `skill_installations` is the source of truth.

`InstallService.SyncInstallState()` reconciles the database with the filesystem on startup, handling both new-style installations (tracked in the table) and legacy installations (tracked only by the boolean flag).

## Consequences

### Benefits

- Complete visibility into where each skill is installed (platform, scope, exact path)
- Supports partial uninstallation (remove from one platform while keeping others)
- Enables the TUI "manage" view showing per-platform installation status
- Migration is non-breaking: legacy boolean is preserved for backward compatibility

### Trade-offs

- Additional table to maintain and query
- Sync logic must handle both old and new installation tracking
- More complex install/uninstall code paths

### Alternatives Considered

| Alternative | Why Not Chosen |
|-------------|---------------|
| JSON column storing locations | Harder to query; no referential integrity |
| Keep boolean only | Cannot support per-platform management or partial uninstall |

## Sources

> Evidence used to reconstruct this decision.

| Source Type | Reference |
|-------------|-----------|
| Git commit | `73643f8` - "refactor(installer): use skill_installations as single source of truth" |
| Code | `internal/models/skill_installation.go` - `SkillInstallation` model definition |
| Code | `internal/db/skill_installations.go` - CRUD operations for installations |
| Code comment | `internal/installer/service.go:40` - "InstallService provides unified installation operations" |

## Related

- [ADR-0001](0001-symlink-based-skill-installation.md) - Symlink installation mechanism
- [Architecture](../architecture.md) - Installer component
