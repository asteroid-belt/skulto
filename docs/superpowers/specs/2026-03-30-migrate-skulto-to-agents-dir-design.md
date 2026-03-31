# Design: Migrate ~/.skulto to ~/.agents/skulto

**Date:** 2026-03-30
**Status:** Draft
**Scope:** Global home directory only. Project-local `./.skulto/skills` is unchanged.

## Motivation

Move Skulto's data directory from `~/.skulto` to `~/.agents/skulto` to coexist under the `~/.agents/` namespace alongside other agent tooling (e.g., vercel-labs/skills uses `~/.agents/skills/`). This aligns Skulto with the emerging convention for agent data directories.

## Design

### 1. Change DefaultBaseDir

`internal/config/paths.go:DefaultBaseDir()` changes from `~/.skulto` to `~/.agents/skulto`. This is the single source of truth — all other paths derive from `cfg.BaseDir`.

### 2. Auto-Migration on Launch

A new `migrateBaseDir()` function runs in `config.Load()` before `ensureDirectories()`. It performs a deterministic sequence of filesystem checks and actions.

#### Migration Flow

```
migrateBaseDir()
  1. Does ~/.skulto exist?
     → no: done (fresh install)
     → is symlink: skip
     → yes, is real dir:
  2. Does ~/.agents/ exist?
     → no: create it (os.MkdirAll, 0755)
  3. Does ~/.agents/skulto/ exist?
     → no: create it (os.MkdirAll, 0755)
  4. Does ~/.agents/skulto/ have contents?
     → yes: remove ~/.skulto, done
     → no:
        a. copy all contents from ~/.skulto → ~/.agents/skulto
        b. verify integrity (file count + total sizes match)
        c. remove ~/.skulto
        d. log.Info("Migrated ~/.skulto → ~/.agents/skulto")
```

**Why copy instead of rename?** `os.Rename` is attempted first (atomic, instant on same filesystem). If it fails (cross-device), fall back to recursive copy with integrity verification.

#### Error Handling

| Step | Failure | Behavior |
|------|---------|----------|
| 2/3 (mkdir) | Permission denied | Return error, fall back to `~/.skulto` for this session. Retries next launch. |
| 4a (copy) | Partial copy | Leave both dirs. Next launch sees contents in `~/.agents/skulto` → removes `~/.skulto`. |
| 4b (verify) | Mismatch | Don't remove `~/.skulto`. Log error. Retries next launch. |
| 4c (remove) | Permission denied | Non-fatal warning. `~/.agents/skulto` is already active. Stale `~/.skulto` is harmless. |

### 3. Database Path Migration

The SQLite database stores absolute paths for local skills that reference `~/.skulto`:

```
skills.file_path: /Users/x/.skulto/skills/my-skill/skill.md
```

After the filesystem move, run:

```sql
UPDATE skills
SET file_path = REPLACE(file_path, '/.skulto/', '/.agents/skulto/')
WHERE file_path LIKE '%/.skulto/%';
```

This runs as part of `migrateBaseDir()`, after the filesystem move succeeds and before returning. The DB file has already been moved as part of the directory, so it's opened from its new location.

**Scope:** Only the `skills.file_path` column contains `~/.skulto` references. All other path columns (`skill_installations.symlink_path`, `discovered_skills.path`, etc.) store paths to platform directories (`~/.claude/skills/`, `~/.cursor/skills/`) which are unaffected.

### 4. Hardcoded Path Updates

Six Go files bypass `cfg.BaseDir` and hardcode `.skulto` in path construction. Only the **global home directory** references change — project-local `./.skulto/skills` paths are unchanged.

| File | Line(s) | What Changes |
|------|---------|--------------|
| `internal/config/paths.go` | 34, 36 | `DefaultBaseDir()`: `.skulto` → `.agents/skulto` |
| `internal/vector/chromem.go` | 32 | Fallback: `".skulto", "vectors"` → `".agents", "skulto", "vectors"` |
| `internal/skillgen/executor.go` | 444 | Home skills dir: `".skulto", "skills"` → `".agents", "skulto", "skills"` |
| `internal/discovery/scanner.go` | 116-117 | Symlink target matching: add `.agents/skulto` patterns alongside existing `.skulto` patterns |
| `internal/discovery/ingestion.go` | 138, 149 | Fallback paths (these are cwd-relative, **no change** — they use `.skulto/skills` for local project context) |
| `internal/cli/ingest.go` | 168, 249 | These are cwd-relative (`.skulto/skills`), **no change** |

**Note on scanner.go:** The `categorizeByTarget()` function string-matches symlink targets to determine management source. It must recognize both old (`/.skulto/`) and new (`/.agents/skulto/`) paths to correctly categorize skills installed before and after migration.

### 5. UI/Display String Updates

| File | Line(s) | Change |
|------|---------|--------|
| `internal/tui/components/quick_skill_dialog.go` | 656, 744, 958 | `~/.skulto/skills/` → `~/.agents/skulto/skills/` |
| `internal/tui/components/save_options_dialog.go` | 57 | `.skulto/skills/` → `.agents/skulto/skills/` (this is a generic description, update to match new path) |

### 6. Documentation/Comment Updates

Update `~/.skulto` references in:

- `AGENTS.md` — data directory table
- `README.md` — favorites path, directory descriptions
- `context/database.md` — DB path reference
- `context/platforms.md` — source path examples
- `docs/getting-started.md` — setup instructions
- `docs/glossary.md` — if referenced
- `docs/adr/0001-symlink-based-skill-installation.md` — path examples
- `docs/adr/0004-environment-variable-only-configuration.md` — base dir reference
- `docs/adr/0005-git-clone-over-github-api.md` — clone dir reference
- Inline comments in `internal/tui/app.go`, `internal/installer/installer.go`, `internal/installer/paths.go`, `internal/scraper/scraper.go`, `internal/config/config.go`, `internal/tui/views/reset.go`

### 7. Test Updates

| File | Change |
|------|--------|
| `internal/config/migrate_test.go` | **New file.** Test all migration branches: fresh install, already migrated, successful move, cross-device fallback, partial failure recovery, DB path update. |
| `internal/installer/installer_test.go` | Update `.skulto` in test helper `testConfig()` |
| `internal/discovery/scanner_test.go` | Update `.skulto` in symlink target tests |
| `internal/discovery/ingestion_test.go` | Update `.skulto` in test paths (global refs only) |
| `internal/discovery/integration_test.go` | Update `.skulto` in test paths (global refs only) |
| `internal/skillgen/executor_test.go` | Update `.skulto` in test paths and assertions |
| `internal/tui/views/paths_test.go` | Update expected path: `~/.skulto/skills` → `~/.agents/skulto/skills` |

## What Does NOT Change

- **Project-local paths**: `./.skulto/skills` (relative to cwd) stays as-is
- **Platform installation paths**: `~/.claude/skills/`, `~/.cursor/skills/`, etc. are unaffected
- **Database schema**: No new columns or tables. Only a data-level UPDATE on `skills.file_path`
- **Config loading**: `config.Load()` still reads env vars the same way
- **`ensureDirectories()`**: Still creates `BaseDir` and `BaseDir/repositories` — just at the new path

## File Inventory

### Must Change (Code)
1. `internal/config/paths.go` — DefaultBaseDir
2. `internal/config/config.go` — call migrateBaseDir in Load
3. `internal/config/migrate.go` — new file, migration logic
4. `internal/vector/chromem.go` — fallback path
5. `internal/skillgen/executor.go` — home skills dir
6. `internal/discovery/scanner.go` — symlink target matching

### Must Change (UI Strings)
7. `internal/tui/components/quick_skill_dialog.go`
8. `internal/tui/components/save_options_dialog.go`

### Must Change (Tests)
9. `internal/config/migrate_test.go` — new file
10. `internal/installer/installer_test.go`
11. `internal/discovery/scanner_test.go`
12. `internal/discovery/ingestion_test.go`
13. `internal/discovery/integration_test.go`
14. `internal/skillgen/executor_test.go`
15. `internal/tui/views/paths_test.go`

### Must Change (Docs/Comments)
16. `AGENTS.md`
17. `README.md`
18. `context/database.md`
19. `context/platforms.md`
20. `docs/getting-started.md`
21. `docs/glossary.md`
22. `docs/adr/0001-symlink-based-skill-installation.md`
23. `docs/adr/0004-environment-variable-only-configuration.md`
24. `docs/adr/0005-git-clone-over-github-api.md`
25. Inline comments (~6 files listed in section 6)
