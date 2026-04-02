---
name: skulto-release-cert
description: Certify a skulto build for Homebrew prod release. Runs three passes — unit/lint/cross-compile, clean-slate CLI walkthrough, and security audit — then produces a certification summary.
---

# Skulto Release Certification

Certify a build for Homebrew production release. Three passes, all must be green before shipping.

## When to Use

Before tagging a release or updating the Homebrew tap. Run from the skulto repo root.

## Pass 1: Unit Tests, Lint, and Cross-Compile

Build both binaries and verify all quality gates.

```bash
make build-all
make test
make lint
make format
```

Cross-compile all release targets:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /dev/null ./cmd/skulto
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o /dev/null ./cmd/skulto
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o /dev/null ./cmd/skulto
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o /dev/null ./cmd/skulto
```

All four must succeed.

## Pass 2: CLI Command Walkthrough

Test every CLI code path against the built binary (`./build/skulto`).

### 2a: Warm state (existing data)

Run each command and verify expected output:

| Command | Expected |
|---------|----------|
| `skulto --help` | Shows usage, subcommands |
| `skulto check` | Lists installed skills with platforms |
| `skulto list` | Lists source repositories |
| `skulto info <slug>` | Shows metadata, tags, install status |
| `skulto favorites list` | Shows favorites or empty state |
| `skulto favorites add <slug>` | Adds skill |
| `skulto favorites remove <slug>` | Removes skill |
| `skulto save` | Saves manifest or "No changes" |
| `skulto save` (again) | "No changes" (idempotent) |
| `skulto scan --pending` | Scans unscanned skills |
| `skulto scan --skill <slug>` | Scans by slug (not just ID) |
| `skulto pull` | Syncs all repos, reconciles |
| `skulto update` | Pull + scan + summary |
| `skulto discover` | Lists unmanaged skills |
| `skulto install --help` | Shows usage |
| `skulto install nonexistent -y` | "No platforms selected" (empty selection safety) |
| `skulto uninstall --help` | Shows usage |
| `skulto add --help` | Shows usage |
| `skulto remove --help` | Shows usage |
| `skulto ingest --help` | Shows usage |
| `skulto feedback` | Shows feedback URL |
| `skulto-mcp --help` | Shows MCP server usage |

### 2b: Clean-slate (fresh install)

Back up existing data, delete `~/.agents/skulto`, and test the full lifecycle:

```bash
cp -r ~/.agents/skulto ~/.agents/skulto.release-cert-backup
rm -rf ~/.agents/skulto
```

Run this sequence — each step must succeed:

| # | Command | Verify |
|---|---------|--------|
| 1 | `skulto check` | "No skills installed", `~/.agents/skulto/` created |
| 2 | `skulto add asteroid-belt/skills` | Clones, indexes skills |
| 3 | `skulto list` | Shows 1 source |
| 4 | `skulto info superplan` | Shows metadata |
| 5 | `skulto install superplan -p claude -y` | Creates symlink, "Installed to 1 location" |
| 6 | `skulto check` | Shows superplan with claude (global) |
| 7 | `skulto install teach -p claude -y` | Second install works |
| 8 | `skulto install supercharge -p claude -y` | Third install works |
| 9 | `skulto uninstall supercharge -y` | Removes symlink + DB record |
| 10 | `skulto check` | supercharge gone |
| 11 | `skulto save` | Writes skulto.json with version |
| 12 | `skulto save` (again) | "No changes" |
| 13 | `skulto favorites add teach` / `list` / `remove` | Full cycle |
| 14 | `skulto scan --skill teach` | Scans by slug |
| 15 | `skulto remove asteroid-belt/skills --force` | Cleans up everything |
| 16 | `skulto check` | Empty |
| 17 | `skulto list` | Empty |

Restore backup:

```bash
rm -rf ~/.agents/skulto
mv ~/.agents/skulto.release-cert-backup ~/.agents/skulto
```

Verify restore:

```bash
skulto check  # Should show original installed skills
```

### 2c: Migration (if applicable)

Only needed when the release includes migration changes. Test:

| # | Step | Verify |
|---|------|--------|
| 1 | Move `~/.agents/skulto` to `~/.skulto` | Simulates pre-migration state |
| 2 | Remove `~/.skulto/.migration-complete` if present | Forces re-migration |
| 3 | Run `skulto check` | Migration runs, data at `~/.agents/skulto` |
| 4 | Check `~/.agents/skulto/.migration-complete` exists | Marker written |
| 5 | Check `~/.skulto` is gone | Old dir removed |
| 6 | Check symlinks still resolve | `readlink` on installed skills |

### 2d: Reconciliation

Test with stale DB (skills on disk but not in DB):

| # | Step | Verify |
|---|------|--------|
| 1 | Delete DB, keep symlinks | `rm ~/.agents/skulto/skulto.db` then any command |
| 2 | `skulto check` | Reconciles project skills, shows them |
| 3 | `skulto save` | Reconciles then saves |
| 4 | Plain dirs in project | Silently skipped, no ingestion prompts |

### 2e: Stale skill cleanup

Test that `skulto pull` removes DB records for skills no longer in upstream repos:

| # | Step | Verify |
|---|------|--------|
| 1 | Insert fake stale skill into DB | `sqlite3 ~/.agents/skulto/skulto.db "INSERT OR IGNORE INTO skills (id, slug, title, source_id, file_path) VALUES ('cert-stale-id', 'cert-stale-skill', 'Cert Stale', 'asteroid-belt/skills', 'skills/cert-stale-skill/SKILL.md');"` |
| 2 | Verify in DB | `sqlite3 ~/.agents/skulto/skulto.db "SELECT slug FROM skills WHERE slug = 'cert-stale-skill';"` returns `cert-stale-skill` |
| 3 | `skulto pull` | Output includes `Removed stale skill: cert-stale-skill` |
| 4 | Verify gone from DB | Same query returns empty |

This simulates a skill that was indexed then removed upstream. The pull detects the mismatch and cleans up.

## Pass 3: Security Audit

Scan the codebase for vulnerabilities. Each category must be clean.

### Automated scans

```bash
# Hardcoded secrets
grep -rn 'sk-\|ghp_\|phc_\|api_key.*=.*['"'"'"]' internal/ cmd/ --include="*.go" | grep -v _test.go | grep -v Getenv | grep -v '//'

# SQL injection (raw string interpolation in queries)
grep -rn 'fmt.Sprintf.*SELECT\|fmt.Sprintf.*INSERT\|fmt.Sprintf.*UPDATE\|fmt.Sprintf.*DELETE' internal/ --include="*.go" | grep -v _test.go

# Command injection
grep -rn 'exec.Command' internal/ --include="*.go" | grep -v _test.go

# Credentials in repo
find . -name ".env" -o -name "credentials*" -o -name "*.pem" -o -name "*.key" | grep -v .git

# os.RemoveAll on user paths
grep -rn 'os.RemoveAll' internal/ --include="*.go" | grep -v _test.go
```

### Manual review checklist

- [ ] No hardcoded API keys or tokens in source
- [ ] All SQL uses parameterized queries (GORM) — no string interpolation
- [ ] `exec.Command` uses argument arrays, not shell strings
- [ ] `os.RemoveAll` only on config-derived paths, never user input
- [ ] `os.Remove` (not `RemoveAll`) in installer symlink cleanup
- [ ] Empty slug guard in `installToLocationsInternal`
- [ ] No `.env` or credential files committed
- [ ] PostHog key injected via ldflags, not in source
- [ ] JSON deserialization only on local user-owned files
- [ ] Git clone uses `go-git` library (no shell execution)

## Certification Output

After all three passes, produce a summary table:

```
SKULTO RELEASE CERTIFICATION
━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Version: <version from build>
Date: <date>
Certifier: <agent/human>

PASS 1: Unit Tests + Quality
  make test:      PASS / FAIL
  make lint:      PASS / FAIL
  make format:    PASS / FAIL
  linux/amd64:    PASS / FAIL
  linux/arm64:    PASS / FAIL
  darwin/amd64:   PASS / FAIL
  darwin/arm64:   PASS / FAIL

PASS 2: CLI Walkthrough
  Warm state:     PASS / FAIL (N/N commands)
  Clean slate:    PASS / FAIL (N/N steps)
  Migration:      PASS / FAIL / SKIPPED
  Reconciliation: PASS / FAIL
  Stale cleanup:  PASS / FAIL

PASS 3: Security Audit
  Secrets scan:   CLEAN / FOUND
  SQL injection:  CLEAN / FOUND
  Cmd injection:  CLEAN / FOUND
  Credentials:    CLEAN / FOUND
  Manual review:  PASS / FAIL

VERDICT: CERTIFIED FOR RELEASE / BLOCKED
```

If any check is FAIL/FOUND/BLOCKED, list the specific failures and do NOT certify.

## Pre-existing Issues

Track known pre-existing issues that are NOT blockers:

| Issue | Notes |
|-------|-------|
| `skulto sync -y` not recognized | Flag is `--yes`, not `-y` |
| `skulto install repo -y` silent skip | Already-installed skills silently skipped in non-interactive |

These do not block release but should be tracked for future fixes.
