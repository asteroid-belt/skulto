# OpenCode XDG Global Skills Path Design

**Date:** 2026-04-18  
**Status:** Approved (brainstorming)  
**Scope:** Installer path resolution for OpenCode only

## Summary

Update OpenCode installation path behavior so global installs use the XDG path
`~/.config/opencode/skills/<slug>`, while project installs remain
`./.opencode/skills/<slug>`.

This addresses user feedback that `~/.opencode/skills/` is deprecated for global
skills and aligns Skulto with current OpenCode docs.

## Goals

- Use XDG global path for OpenCode global installs.
- Preserve OpenCode project-local behavior.
- Keep all non-OpenCode platforms unchanged.
- Keep implementation small and centralized.

## Non-Goals

- No platform registry schema refactor.
- No database schema changes.
- No migration for existing legacy OpenCode install directories.

## Design

### Path Resolution Rule

Centralize install target path resolution in installer code:

- If `platform == opencode` and `scope == global`:
  - target path = `<home>/.config/opencode/skills/<slug>`
- Otherwise:
  - target path = `<basePath>/<platform.SkillsPath>/<slug>`

### Integration Points

Use the shared resolver in:

- `InstallLocation.GetSkillPath`
- `InstallLocation.GetBaseSkillsPath`
- `Platform.GetSkillPathForScope`

This ensures install, uninstall, and path expectations all use one rule.

### Why This Shape

- Minimal blast radius.
- Preserves current data model and API shape.
- Fixes real behavior mismatch without broad refactoring.

## Data Flow

1. Caller requests an install path for `(platform, scope, slug)`.
2. Scope resolves to base path (`home` or `cwd`).
3. Resolver applies OpenCode-global override when applicable.
4. Final target path is used by installer operations.

## Error Handling

- No new error types.
- Existing invalid scope behavior remains unchanged.
- Existing behavior for unknown/empty skills path remains unchanged.

## Testing Plan

- Update `internal/installer/installer_test.go`:
  - `TestPlatformGetSkillPath` OpenCode global expected path:
    - from `~/.opencode/skills/test-skill`
    - to `~/.config/opencode/skills/test-skill`
- Add/extend installer path tests to validate OpenCode split behavior:
  - Global scope uses `.config/opencode/skills`
  - Project scope uses `.opencode/skills`
- Run targeted installer tests, then full suite (`make test`).

## Documentation Plan

Update `context/platforms.md` to clarify OpenCode by scope:

- Project skills path: `.opencode/skills`
- Global skills path: `~/.config/opencode/skills/`

## Conventional Commit Summary

`fix(installer): use XDG global skills path for OpenCode`
