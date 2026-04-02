# Source-Mismatch Detection Across CLI Commands

**Date:** 2026-03-31
**Status:** Approved

## Problem

When a skill's backing repository is swapped (e.g., a supply-chain attack replaces `asteroid-belt/skills` with `evil-fork/skills`), skulto should warn the user rather than silently installing the imposter. Today, `skulto sync` has a basic inline check for this, but `skulto install <slug>` and `skulto update` do not. Additionally, the existing sync check offers no resolution path — it warns and skips, with no way to accept the new source or override.

## Solution

Extract the source-mismatch check into a shared helper, wire it into all three commands (`sync`, `install`, `update`), and offer three resolution options: Accept, Skip, Install anyway.

## Design

### New file: `internal/cli/source_check.go`

A shared helper with three functions:

```go
type SourceMismatchAction int

const (
    SourceMismatchSkip          SourceMismatchAction = iota // Don't install
    SourceMismatchAccept                                     // Update manifest to new source
    SourceMismatchInstallAnyway                              // Install without updating manifest
)

type SourceMismatch struct {
    Slug           string
    ExpectedSource string
    ActualSource   string
}
```

**`CheckSourceMismatch(skill *models.Skill, expectedSource string) *SourceMismatch`**
Compares the skill's `Source.FullName` against the expected source string. Returns nil if no mismatch (including when source is nil). Returns a `*SourceMismatch` describing the discrepancy otherwise.

**`PromptSourceMismatch(mismatch *SourceMismatch, reader *bufio.Reader) SourceMismatchAction`**
Prints a styled warning and prompts the user:

```
WARN Skill 'superplan' found but from different source (evil-fork/skills, expected asteroid-belt/skills)
  [a]ccept new source  [s]kip  [i]nstall anyway
  Choice [s]:
```

Returns the chosen action. Non-interactive mode (no TTY) returns `SourceMismatchSkip`.

**`ApplySourceMismatchAccept(dir string, slug string, newSource string) error`**
Reads `skulto.json` from `dir` via `manifest.Read`, updates the entry for `slug` to point to `newSource` (adding it if not present), and writes it back via `manifest.Write`. No-ops if no manifest file exists. Errors from `manifest.Write` are propagated.

### Wiring: `skulto sync`

Replace the inline mismatch check in `syncResolveSkills()` with calls to the shared helper.

`syncResolveSkills` gains two new parameters: `reader *bufio.Reader` and `cwd string` (both already available in `runSync`).

Resolution behavior:
- **Skip**: increment `skippedSkills`, continue (same as today)
- **Accept**: call `ApplySourceMismatchAccept`, add skill to install list
- **Install anyway**: add skill to install list without updating manifest

### Wiring: `skulto install <slug>`

The check goes into `runInstallBySlug()`, after skill resolution but before platform selection. The function obtains `cwd` via `os.Getwd()` and creates a `bufio.NewReader(os.Stdin)` for prompting.

Flow:
1. Resolve the skill via a new `service.ResolveSkill(slug)` method (thin wrapper around `db.GetSkillBySlug`). Note: `service.Install` will look the skill up again internally — this double lookup is acceptable for simplicity.
2. Check if `skulto.json` exists in cwd — if so, compare against the manifest's expected source
3. If no manifest, skip the mismatch check — there is no expected source to compare against
4. If mismatch detected, prompt with the three options
5. **Accept**: update `skulto.json`
6. **Skip**: return early
7. **Install anyway**: proceed normally

When the `-y` flag is passed, source mismatches default to **Skip** (same as non-interactive mode). This is the safe default — auto-accepting a changed source would defeat the purpose of the check.

### Wiring: `skulto update`

The check goes into `runUpdatePull()`, after repos are pulled and skills re-scraped, before the scan phase.

Flow:
1. After pull completes, check if `skulto.json` exists in cwd
2. If no manifest, skip — update without a manifest is just "pull latest"
3. If manifest exists, iterate entries and call `CheckSourceMismatch` for each skill
4. Present mismatches one at a time with the same prompt
5. Batch-collect **Accept** actions and apply them in a single `manifest.Write` call at the end
6. The check is advisory — it does not block the pull or scan, it surfaces drift

### New method: `InstallService.ResolveSkill`

Add to `internal/installer/service.go`:

```go
func (s *InstallService) ResolveSkill(slug string) (*models.Skill, error) {
    skill, err := s.db.GetSkillBySlug(slug)
    if err != nil {
        return nil, fmt.Errorf("failed to look up skill: %w", err)
    }
    return skill, nil
}
```

This keeps `runInstallBySlug` from reaching into the DB directly.

## Files changed

| File | Change |
|---|---|
| `internal/cli/source_check.go` | **New** — shared mismatch detection, prompting, and manifest update |
| `internal/cli/source_check_test.go` | **New** — unit tests |
| `internal/cli/sync.go` | **Modify** — replace inline check with shared helper |
| `internal/cli/install.go` | **Modify** — add mismatch check in `runInstallBySlug` |
| `internal/cli/update.go` | **Modify** — add post-pull mismatch check |
| `internal/installer/service.go` | **Modify** — add `ResolveSkill` method |

No model changes. No manifest schema changes. No database migrations.

## Testing

**Unit tests (`internal/cli/source_check_test.go`):**
- `TestCheckSourceMismatch`: nil source, matching source, mismatching source, nil expected source
- `TestApplySourceMismatchAccept`: updates existing entry, creates entry if missing, no-ops without manifest
- `TestPromptSourceMismatch`: inject `*bufio.Reader` backed by `strings.Reader` to test each choice (a/s/i) and default behavior

**Existing test integration:**
- Verify sync respects each action
- Verify install detects mismatch against manifest and prior DB source
- Verify update detects post-pull drift and batches manifest updates
