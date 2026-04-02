# Source-Mismatch Detection Implementation Plan

> **Spec**: `docs/superpowers/specs/2026-03-31-source-mismatch-detection-design.md`
> **Date**: 2026-04-01
> **Total Estimate**: 18 points

## Requirements

- Extract inline source-mismatch check from `skulto sync` into a shared helper
- Wire mismatch detection into `skulto install <slug>` and `skulto update`
- Offer three resolution options: Accept (update manifest), Skip, Install anyway
- Non-interactive / `-y` mode defaults to Skip
- Add `ResolveSkill` method to `InstallService`

## Architecture

```
                        source_check.go (NEW)
                    ┌─────────────────────────┐
                    │ CheckSourceMismatch()    │
                    │ PromptSourceMismatch()   │
                    │ ApplySourceMismatchAccept│
                    └────────┬────────────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
         sync.go        install.go     update.go
      (replace inline)  (add check)   (add post-pull)
              │              │
              └──────┬───────┘
                     │
              manifest.Read/Write
```

## Phase Dependency Diagram

```
Phase 1A ─────────┐
(source_check.go) │
                  ├──→ Phase 2A ─────────┐
Phase 1B ─────────┤    (sync.go)         │
(ResolveSkill)    │                      ├──→ Phase 3
                  ├──→ Phase 2B ─────────┤    (integration tests)
                  │    (install.go)      │
                  │                      │
                  └──→ Phase 2C ─────────┘
                       (update.go)
```

| Phase | Name | Depends On | Parallel With | Estimate | Status |
|-------|------|------------|---------------|----------|--------|
| 1A | Shared helper + unit tests | — | 1B | 5 | pending |
| 1B | `ResolveSkill` method | — | 1A | 1 | pending |
| 2A | Wire into sync | 1A | 2B, 2C | 3 | pending |
| 2B | Wire into install | 1A, 1B | 2A, 2C | 5 | pending |
| 2C | Wire into update | 1A | 2A, 2B | 3 | pending |
| 3 | Integration tests | 2A, 2B, 2C | — | 1 | pending |

---

## Phase 1A: Shared helper + unit tests (5 pts)

### Definition of Done
- [ ] `source_check.go` created with all three functions
- [ ] `source_check_test.go` with full coverage
- [ ] Code passes linter (`make lint`)
- [ ] Code passes formatter (`make format`)
- [ ] All tests pass (`make test`)

### Code Deltas

#### `internal/cli/source_check.go` (CREATE)

```go
package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/asteroid-belt/skulto/internal/manifest"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/charmbracelet/lipgloss"
)

// SourceMismatchAction is the user's chosen resolution for a source mismatch.
type SourceMismatchAction int

const (
	// SourceMismatchSkip skips the skill (do not install).
	SourceMismatchSkip SourceMismatchAction = iota
	// SourceMismatchAccept updates the manifest to the new source and installs.
	SourceMismatchAccept
	// SourceMismatchInstallAnyway installs without updating the manifest.
	SourceMismatchInstallAnyway
)

// SourceMismatch describes a detected source mismatch.
type SourceMismatch struct {
	Slug           string
	ExpectedSource string
	ActualSource   string
}

// CheckSourceMismatch compares a skill's actual source against an expected source.
// Returns nil if no mismatch (including when skill.Source is nil).
func CheckSourceMismatch(skill *models.Skill, expectedSource string) *SourceMismatch {
	if skill.Source == nil {
		return nil
	}
	if skill.Source.FullName == expectedSource {
		return nil
	}
	return &SourceMismatch{
		Slug:           skill.Slug,
		ExpectedSource: expectedSource,
		ActualSource:   skill.Source.FullName,
	}
}

// PromptSourceMismatch prints a warning and prompts the user for resolution.
// In non-interactive mode, returns SourceMismatchSkip.
func PromptSourceMismatch(mismatch *SourceMismatch, reader *bufio.Reader) SourceMismatchAction {
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	fmt.Printf("  %s Skill '%s' found but from different source (%s, expected %s)\n",
		warnStyle.Render("WARN"), mismatch.Slug, mismatch.ActualSource, mismatch.ExpectedSource)

	if !isInteractive() {
		return SourceMismatchSkip
	}

	fmt.Print("  [a]ccept new source  [s]kip  [i]nstall anyway\n")
	fmt.Print("  Choice [s]: ")

	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	switch answer {
	case "a", "accept":
		return SourceMismatchAccept
	case "i", "install":
		return SourceMismatchInstallAnyway
	default:
		return SourceMismatchSkip
	}
}

// ApplySourceMismatchAccept updates skulto.json to point the slug at the new source.
// No-ops if no manifest file exists. If the slug is not in the manifest, it is added.
// Uses manifest.Read and manifest.Write for atomic file operations.
func ApplySourceMismatchAccept(dir string, slug string, newSource string) error {
	mf, err := manifest.Read(dir)
	if err != nil {
		return fmt.Errorf("read manifest for source update: %w", err)
	}
	if mf == nil {
		return nil // No manifest, nothing to update
	}

	mf.Skills[slug] = newSource

	if err := manifest.Write(dir, mf); err != nil {
		return fmt.Errorf("write manifest after source update: %w", err)
	}

	fmt.Printf("  Updated skulto.json: %s -> %s\n", slug, newSource)
	return nil
}
```

#### `internal/cli/source_check_test.go` (CREATE)

```go
package cli

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/asteroid-belt/skulto/internal/manifest"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckSourceMismatch_NilSource(t *testing.T) {
	skill := &models.Skill{Slug: "test-skill"}
	result := CheckSourceMismatch(skill, "owner/repo")
	assert.Nil(t, result)
}

func TestCheckSourceMismatch_MatchingSource(t *testing.T) {
	skill := &models.Skill{
		Slug:   "test-skill",
		Source: &models.Source{FullName: "owner/repo"},
	}
	result := CheckSourceMismatch(skill, "owner/repo")
	assert.Nil(t, result)
}

func TestCheckSourceMismatch_MismatchDetected(t *testing.T) {
	skill := &models.Skill{
		Slug:   "test-skill",
		Source: &models.Source{FullName: "evil-fork/repo"},
	}
	result := CheckSourceMismatch(skill, "owner/repo")

	require.NotNil(t, result)
	assert.Equal(t, "test-skill", result.Slug)
	assert.Equal(t, "owner/repo", result.ExpectedSource)
	assert.Equal(t, "evil-fork/repo", result.ActualSource)
}

func TestPromptSourceMismatch_AcceptChoice(t *testing.T) {
	mismatch := &SourceMismatch{
		Slug:           "superplan",
		ExpectedSource: "asteroid-belt/skills",
		ActualSource:   "evil-fork/skills",
	}
	reader := bufio.NewReader(strings.NewReader("a\n"))
	action := PromptSourceMismatch(mismatch, reader)
	assert.Equal(t, SourceMismatchAccept, action)
}

func TestPromptSourceMismatch_SkipChoice(t *testing.T) {
	mismatch := &SourceMismatch{
		Slug:           "superplan",
		ExpectedSource: "asteroid-belt/skills",
		ActualSource:   "evil-fork/skills",
	}
	reader := bufio.NewReader(strings.NewReader("s\n"))
	action := PromptSourceMismatch(mismatch, reader)
	assert.Equal(t, SourceMismatchSkip, action)
}

func TestPromptSourceMismatch_InstallAnywayChoice(t *testing.T) {
	mismatch := &SourceMismatch{
		Slug:           "superplan",
		ExpectedSource: "asteroid-belt/skills",
		ActualSource:   "evil-fork/skills",
	}
	reader := bufio.NewReader(strings.NewReader("i\n"))
	action := PromptSourceMismatch(mismatch, reader)
	assert.Equal(t, SourceMismatchInstallAnyway, action)
}

func TestPromptSourceMismatch_DefaultIsSkip(t *testing.T) {
	mismatch := &SourceMismatch{
		Slug:           "superplan",
		ExpectedSource: "asteroid-belt/skills",
		ActualSource:   "evil-fork/skills",
	}
	reader := bufio.NewReader(strings.NewReader("\n"))
	action := PromptSourceMismatch(mismatch, reader)
	assert.Equal(t, SourceMismatchSkip, action)
}

func TestApplySourceMismatchAccept_UpdatesExistingEntry(t *testing.T) {
	dir := t.TempDir()
	mf := manifest.New()
	mf.Skills["superplan"] = "old-owner/skills"
	require.NoError(t, manifest.Write(dir, mf))

	err := ApplySourceMismatchAccept(dir, "superplan", "new-owner/skills")
	require.NoError(t, err)

	updated, err := manifest.Read(dir)
	require.NoError(t, err)
	assert.Equal(t, "new-owner/skills", updated.Skills["superplan"])
}

func TestApplySourceMismatchAccept_AddsNewEntry(t *testing.T) {
	dir := t.TempDir()
	mf := manifest.New()
	mf.Skills["existing"] = "owner/repo"
	require.NoError(t, manifest.Write(dir, mf))

	err := ApplySourceMismatchAccept(dir, "new-skill", "owner/repo")
	require.NoError(t, err)

	updated, err := manifest.Read(dir)
	require.NoError(t, err)
	assert.Equal(t, "owner/repo", updated.Skills["new-skill"])
	assert.Equal(t, "owner/repo", updated.Skills["existing"])
}

func TestApplySourceMismatchAccept_NoManifest(t *testing.T) {
	dir := t.TempDir()

	err := ApplySourceMismatchAccept(dir, "superplan", "new-owner/skills")
	assert.NoError(t, err)

	// Verify no manifest was created
	_, err = os.Stat(filepath.Join(dir, "skulto.json"))
	assert.True(t, os.IsNotExist(err))
}
```

---

## Phase 1B: `ResolveSkill` method (1 pt)

### Definition of Done
- [ ] `ResolveSkill` added to `InstallService`
- [ ] Code passes linter (`make lint`)
- [ ] All tests pass (`make test`)

### Code Deltas

#### `internal/installer/service.go` (MODIFY)

Add after the existing `Install` method:

```diff
+// ResolveSkill looks up a skill by slug without installing it.
+// Returns nil, nil if the skill is not found.
+func (s *InstallService) ResolveSkill(slug string) (*models.Skill, error) {
+	skill, err := s.db.GetSkillBySlug(slug)
+	if err != nil {
+		return nil, fmt.Errorf("failed to look up skill: %w", err)
+	}
+	return skill, nil
+}
```

---

## Phase 2A: Wire into sync (3 pts)

### Definition of Done
- [ ] Inline mismatch check replaced with shared helper
- [ ] `syncResolveSkills` accepts `reader` and `cwd` parameters
- [ ] All three actions (Accept/Skip/Install anyway) work correctly
- [ ] Code passes linter (`make lint`)
- [ ] All tests pass (`make test`)

### Code Deltas

#### `internal/cli/sync.go` (MODIFY)

**Change 1**: Update `syncResolveSkills` call site in `runSync`:

```diff
-	skillsToInstall, skippedSkills := syncResolveSkills(mf, database, skippedSources)
+	skillsToInstall, skippedSkills := syncResolveSkills(mf, database, skippedSources, reader, cwd)
```

**Change 2**: Update `syncResolveSkills` signature and replace inline check:

```diff
 func syncResolveSkills(
 	mf *manifest.ManifestFile,
 	database *db.DB,
 	skippedSources map[string]bool,
+	reader *bufio.Reader,
+	cwd string,
 ) ([]*models.Skill, int) {
 	var skillsToInstall []*models.Skill
 	var skippedSkills int

-	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
-
 	for _, slug := range mf.SortedSlugs() {
 		sourceName := mf.Skills[slug]

 		if skippedSources[sourceName] {
 			skippedSkills++
 			continue
 		}

 		skill, err := database.GetSkillBySlug(slug)
 		if err != nil || skill == nil {
+			warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
 			fmt.Printf("  %s Skill '%s' not found in database\n", warnStyle.Render("WARN"), slug)
 			skippedSkills++
 			continue
 		}

-		if skill.Source != nil && skill.Source.FullName != sourceName {
-			fmt.Printf("  %s Skill '%s' found but from different source (%s, expected %s)\n",
-				warnStyle.Render("WARN"), slug, skill.Source.FullName, sourceName)
-			skippedSkills++
-			continue
+		if mismatch := CheckSourceMismatch(skill, sourceName); mismatch != nil {
+			action := PromptSourceMismatch(mismatch, reader)
+			switch action {
+			case SourceMismatchSkip:
+				skippedSkills++
+				continue
+			case SourceMismatchAccept:
+				if err := ApplySourceMismatchAccept(cwd, slug, mismatch.ActualSource); err != nil {
+					fmt.Printf("  Error updating manifest: %v\n", err)
+				}
+			case SourceMismatchInstallAnyway:
+				// proceed without updating manifest
+			}
 		}

 		skillsToInstall = append(skillsToInstall, skill)
 	}
```

Note: The `lipgloss` import may become unused if the only usage was the `warnStyle` in the removed block. Move the `warnStyle` declaration into the "not found" branch (shown above) or keep the import if other styled output remains.

---

## Phase 2B: Wire into install (5 pts)

### Definition of Done
- [ ] Mismatch check added to `runInstallBySlug`
- [ ] Respects `-y` flag (defaults to Skip)
- [ ] Only checks when `skulto.json` exists
- [ ] All three actions work correctly
- [ ] Code passes linter (`make lint`)
- [ ] All tests pass (`make test`)

### Code Deltas

#### `internal/cli/install.go` (MODIFY)

Replace `runInstallBySlug`:

```diff
 func runInstallBySlug(ctx context.Context, service *installer.InstallService, slug string) error {
+	// Check for source mismatch against manifest
+	if !installYes {
+		cwd, err := os.Getwd()
+		if err == nil {
+			mf, _ := manifest.Read(cwd)
+			if mf != nil {
+				if expectedSource, ok := mf.Skills[slug]; ok {
+					skill, err := service.ResolveSkill(slug)
+					if err == nil && skill != nil {
+						if mismatch := CheckSourceMismatch(skill, expectedSource); mismatch != nil {
+							reader := bufio.NewReader(os.Stdin)
+							action := PromptSourceMismatch(mismatch, reader)
+							switch action {
+							case SourceMismatchSkip:
+								fmt.Println("  Skipped.")
+								return nil
+							case SourceMismatchAccept:
+								if err := ApplySourceMismatchAccept(cwd, slug, mismatch.ActualSource); err != nil {
+									fmt.Printf("  Error updating manifest: %v\n", err)
+								}
+							case SourceMismatchInstallAnyway:
+								// proceed
+							}
+						}
+					}
+				}
+			}
+		}
+	}
+
 	opts, err := selectPlatformsAndScope(service, ctx, slug)
 	if err != nil {
 		return trackCLIError("install", err)
```

Add imports at the top of the file:

```diff
 import (
 	"bufio"
 	"context"
 	"fmt"
 	"os"
 	"strings"

 	"github.com/asteroid-belt/skulto/internal/cli/prompts"
 	"github.com/asteroid-belt/skulto/internal/config"
 	"github.com/asteroid-belt/skulto/internal/db"
 	"github.com/asteroid-belt/skulto/internal/installer"
+	"github.com/asteroid-belt/skulto/internal/manifest"
 	"github.com/asteroid-belt/skulto/internal/models"
 	"github.com/asteroid-belt/skulto/internal/scraper"
```

Note: `bufio` is already imported. `manifest` is the new import.

---

## Phase 2C: Wire into update (3 pts)

### Definition of Done
- [ ] Post-pull mismatch check added when manifest exists
- [ ] Mismatches prompted one at a time
- [ ] Accept actions batch-applied in single `manifest.Write`
- [ ] Code passes linter (`make lint`)
- [ ] All tests pass (`make test`)

### Code Deltas

#### `internal/cli/update.go` (MODIFY)

**Change 1**: Add source mismatch check between pull and scan phases in `runUpdate`:

```diff
 	if err := runUpdatePull(ctx, cfg, database, result); err != nil {
 		return err
 	}

+	// Check for source mismatches against manifest
+	cwd, _ := os.Getwd()
+	if cwd != "" {
+		runUpdateSourceCheck(cwd, database)
+	}
+
 	// Phase 2: Security scan
```

**Change 2**: Add the new function:

```go
// runUpdateSourceCheck checks for source mismatches against skulto.json after pulling.
// This is advisory — it does not block the update, it surfaces drift.
func runUpdateSourceCheck(cwd string, database *db.DB) {
	mf, err := manifest.Read(cwd)
	if err != nil || mf == nil {
		return // No manifest, nothing to check
	}

	reader := bufio.NewReader(os.Stdin)
	var acceptedUpdates []struct {
		slug      string
		newSource string
	}

	for _, slug := range mf.SortedSlugs() {
		expectedSource := mf.Skills[slug]

		skill, err := database.GetSkillBySlug(slug)
		if err != nil || skill == nil {
			continue // Skill not in DB, skip
		}

		mismatch := CheckSourceMismatch(skill, expectedSource)
		if mismatch == nil {
			continue
		}

		action := PromptSourceMismatch(mismatch, reader)
		switch action {
		case SourceMismatchAccept:
			acceptedUpdates = append(acceptedUpdates, struct {
				slug      string
				newSource string
			}{slug, mismatch.ActualSource})
		case SourceMismatchSkip, SourceMismatchInstallAnyway:
			// Skip and Install anyway both do nothing to the manifest during update
		}
	}

	// Batch-apply accepted source updates
	if len(acceptedUpdates) > 0 {
		mf, err := manifest.Read(cwd)
		if err != nil || mf == nil {
			return
		}
		for _, u := range acceptedUpdates {
			mf.Skills[u.slug] = u.newSource
		}
		if err := manifest.Write(cwd, mf); err != nil {
			fmt.Printf("  Error updating manifest: %v\n", err)
		} else {
			fmt.Printf("  Updated skulto.json with %d source change(s)\n", len(acceptedUpdates))
		}
	}
}
```

**Change 3**: Add imports:

```diff
 import (
+	"bufio"
 	"context"
 	"fmt"
+	"os"
 	"time"

 	"github.com/asteroid-belt/skulto/internal/config"
 	"github.com/asteroid-belt/skulto/internal/db"
 	"github.com/asteroid-belt/skulto/internal/installer"
+	"github.com/asteroid-belt/skulto/internal/manifest"
 	"github.com/asteroid-belt/skulto/internal/models"
```

---

## Phase 3: Integration tests (1 pt)

### Definition of Done
- [ ] Sync test covers mismatch detection with shared helper
- [ ] Install test covers manifest-based mismatch detection
- [ ] All tests pass (`make test`)
- [ ] `make lint` passes

### Code Deltas

#### `internal/cli/source_check_test.go` (MODIFY — add integration tests)

Append to the existing test file:

```go
func TestCheckSourceMismatch_Integration_SyncScenario(t *testing.T) {
	// Simulates sync resolving a skill from the wrong source
	sourceID := "evil-fork/skills"
	skill := &models.Skill{
		Slug:     "superplan",
		SourceID: &sourceID,
		Source:   &models.Source{FullName: "evil-fork/skills"},
	}

	mismatch := CheckSourceMismatch(skill, "asteroid-belt/skills")
	require.NotNil(t, mismatch)
	assert.Equal(t, "superplan", mismatch.Slug)
	assert.Equal(t, "asteroid-belt/skills", mismatch.ExpectedSource)
	assert.Equal(t, "evil-fork/skills", mismatch.ActualSource)
}

func TestApplySourceMismatchAccept_Integration_ManifestRoundTrip(t *testing.T) {
	dir := t.TempDir()

	// Create a manifest with multiple skills
	mf := manifest.New()
	mf.Skills["superplan"] = "asteroid-belt/skills"
	mf.Skills["teach"] = "asteroid-belt/skills"
	mf.Skills["other"] = "other-owner/repo"
	require.NoError(t, manifest.Write(dir, mf))

	// Accept new source for superplan
	err := ApplySourceMismatchAccept(dir, "superplan", "new-owner/skills")
	require.NoError(t, err)

	// Verify only superplan changed
	updated, err := manifest.Read(dir)
	require.NoError(t, err)
	assert.Equal(t, "new-owner/skills", updated.Skills["superplan"])
	assert.Equal(t, "asteroid-belt/skills", updated.Skills["teach"])
	assert.Equal(t, "other-owner/repo", updated.Skills["other"])
}
```

---

## Testing Strategy

| Type | Count | Location |
|------|-------|----------|
| Unit (CheckSourceMismatch) | 3 | source_check_test.go |
| Unit (PromptSourceMismatch) | 4 | source_check_test.go |
| Unit (ApplySourceMismatchAccept) | 3 | source_check_test.go |
| Integration | 2 | source_check_test.go |
| **Total** | **12** | |

Tests follow existing patterns: `testify/assert` + `testify/require`, `t.TempDir()` for isolation, `strings.NewReader` for prompt injection.

## Assumptions

- `isInteractive()` from `install.go` is accessible within the `cli` package (same package)
- `manifest.Read` returns `nil, nil` for missing files (confirmed in source)
- The `lipgloss` warn style (color 214) matches existing usage in sync.go
- No telemetry events needed for mismatch detection itself (the existing sync/install telemetry covers outcomes)
