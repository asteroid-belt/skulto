# ADR-0001: Symlink-Based Skill Installation

**Status:** Accepted
**Date:** 2026-01-01

## Context

Skulto needs to install skills from cloned repositories to multiple AI platform directories (e.g., `~/.claude/skills/`, `~/.cursor/skills/`). The same skill may be installed to many platforms simultaneously. Skills are updated when repositories are synced, and the installed copy must reflect the latest version.

Two approaches were considered: copying skill files to each target directory, or creating symbolic links from the target directories back to the cloned repository.

## Decision

Skills are installed as symlinks. The installer creates a symbolic link from the platform's skills directory to the skill's directory inside the cloned repository at `~/.skulto/repositories/{owner}/{repo}/`.

## Consequences

### Benefits

- Updates propagate automatically: when `skulto pull` syncs a repository, all installed locations immediately reflect the new content
- Disk usage is minimal: one copy of each skill regardless of how many platforms it is installed to
- Uninstallation is clean: remove the symlink, no orphaned files
- Installation state can be verified by checking if the symlink target exists

### Trade-offs

- Requires the cloned repository to remain intact; removing a repository breaks all installations from it
- Symlinks may not work identically across all operating systems (though Skulto targets macOS and Linux where symlink behavior is consistent)
- Some AI tools may not follow symlinks (not observed in practice with supported platforms)

### Alternatives Considered

| Alternative | Why Not Chosen |
|-------------|---------------|
| File copy | Updates would require re-copying on every sync; disk usage grows linearly with platform count |
| Hard links | Cannot span filesystem boundaries; harder to audit |

## Sources

> Evidence used to reconstruct this decision.

| Source Type | Reference |
|-------------|-----------|
| Code comment | `internal/installer/installer.go` - "Skills are installed by creating symlinks from repository skill directories to the platform skill directories" |
| Code | `internal/installer/symlink.go` - symlink creation and removal implementation |
| Config file | `internal/installer/platform.go` - platform registry with SkillsPath definitions |
| Code | `internal/installer/service.go:40` - "InstallService...wraps the underlying Installer for symlink operations" |

## Related

- [Architecture](../architecture.md) - Installer component
