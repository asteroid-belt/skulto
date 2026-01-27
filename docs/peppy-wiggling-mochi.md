# Install Refactor: Unified Installation Service

> **For Claude:** Execute this plan using `/superbuild` skill.
> Each phase includes Definition of Done criteria that must pass before proceeding.

> Generated: 2026-01-26
> Status: Draft
> Author: Claude (superplan)
> Last Updated: 2026-01-26

**Goal:** Unify all installation paths (CLI, TUI, MCP) into a single `InstallService`, add interactive CLI commands, and ensure consistent behavior across all entry points.

**Architecture:** Service-oriented design where `InstallService` wraps the existing `Installer` for symlink operations. All callers use the service, which handles skill/source lookup, platform detection, and telemetry.

**Tech Stack:** Go 1.25+, Cobra CLI, Bubble Tea TUI, charmbracelet/huh (new), MCP server

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Technology Stack](#technology-stack)
3. [Requirements](#requirements)
4. [Architecture](#architecture)
5. [Implementation Phases](#implementation-phases)
6. [Testing Strategy](#testing-strategy)
7. [Assumptions & Risks](#assumptions--risks)

---

## Executive Summary

### One-Line Summary

Unify skull installation across CLI/TUI/MCP into a single service with new interactive CLI commands.

### Goals

- [x] **Primary Goal**: Eliminate code duplication in installation logic
- [x] **Secondary Goal**: Add `skulto install` and `skulto uninstall` CLI commands
- [x] **Success Metric**: All install operations go through `InstallService`

### Non-Goals (Explicitly Out of Scope)

- ❌ Changing the underlying symlink mechanism in `Installer`
- ❌ Adding new platforms beyond what's currently supported
- ❌ Changing the database schema for installations
- ❌ Modifying the TUI dialog appearance (only backend changes)

### Key Decisions Made

| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| Service wraps Installer | Separation of concerns - service handles lookup, installer handles symlinks | Merge into Installer (too coupled) |
| charmbracelet/huh for CLI prompts | Same Charm ecosystem as Bubble Tea, purpose-built for forms | survey (different ecosystem), raw Bubble Tea (overkill) |
| Flags pre-select, -y skips prompts | Common CLI pattern (apt, brew), safe default | Flags skip entirely (less flexible) |
| URL install auto-adds repo | Single command convenience, users expect this | Require explicit add first (more steps) |
| MCP gets optional params | Parity with CLI flexibility, AI agents benefit | Keep MCP simple (limits capabilities) |

### Phase Overview (with Poker Estimates)

| Phase | Name | Depends On | Parallel With | Estimate | Status |
|-------|------|------------|---------------|----------|--------|
| 1 | Core Service | - | - | 5 | ✅ |
| 2 | CLI Prompts | 1 | - | 3 | ✅ |
| 3A | CLI Install Command | 2 | 3B | 5 | ✅ |
| 3B | CLI Uninstall Command | 2 | 3A | 3 | ✅ |
| 4 | TUI Refactor | 1 | - | 5 | ⬜ |
| 5 | MCP Updates | 1 | 4 | 3 | ⬜ |

**Total Estimate**: 24 points

**Legend**: ⬜ Not Started | 🔄 In Progress | ✅ Complete | ⏸️ Blocked | ⏭️ Skipped

**PARALLEL EXECUTION**: Phases 3A/3B can run in parallel. Phases 4/5 can run in parallel after Phase 1.

---

## Technology Stack

### Languages
- **Primary**: Go 1.25+

### Frameworks
| Framework | Version | Purpose |
|-----------|---------|---------|
| Cobra | Latest | CLI commands |
| Bubble Tea | Latest | TUI |
| charmbracelet/huh | Latest | CLI interactive forms (NEW) |
| mcp-go | Latest | MCP server |

### Quality Tools Status

| Tool Type | Status | Config File | Command |
|-----------|--------|-------------|---------|
| Linter | ✅ | `.golangci.yml` | `make lint` |
| Formatter | ✅ | Built-in | `go fmt ./...` |
| Type Checker | ✅ | Built-in | `go vet ./...` |
| Test Framework | ✅ | Built-in | `go test ./...` |

### Bootstrap Required?

- [x] **No** - All quality tools present, skip Phase 0

---

## Requirements

### Original Story/Request

```
An install refactor that simplifies all install paths down to a single unified one
across all forms of product: CLI, TUI, and MCP.

- Support flags for control AND interactive mode
- Interactive mode uses inline UI (not full TUI) with checkboxes
- Show ALL detected platforms, space to select, enter to confirm
- Support URL input to auto-add repo and show skill picker
- Mirror install UX for uninstall
- Add optional platforms/scope params to MCP tools
```

**Source**: Brainstorming session

### Acceptance Criteria

- [ ] **AC-1**: `skulto install <slug>` shows interactive platform/scope selection
- [ ] **AC-2**: `skulto install <slug> -p claude -s global -y` installs non-interactively
- [ ] **AC-3**: `skulto install <url>` auto-adds repo and shows skill picker
- [ ] **AC-4**: `skulto uninstall <slug>` shows interactive location selection
- [ ] **AC-5**: TUI 'i' key uses `InstallService` (no behavior change)
- [ ] **AC-6**: MCP `skulto_install` accepts optional `platforms` and `scope` params
- [ ] **AC-7**: All existing tests pass after refactor

### Clarifications from Interview

| Question | Answer | Implication |
|----------|--------|-------------|
| URL auto-add behavior? | Auto-add permanently | No temporary repo state needed |
| Flag behavior? | Pre-select, prompts still show unless -y | Need -y flag implementation |
| Uninstall UX? | Mirror install with location checkboxes | Consistent prompt components |
| MCP changes? | Add optional params to existing tools | Backward compatible |

---

## Architecture

### System Context Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         INSTALL SYSTEM CONTEXT                           │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐                 │
│  │     CLI      │   │     TUI      │   │     MCP      │                 │
│  │  (install/   │   │   (i key)    │   │  (tools)     │                 │
│  │  uninstall)  │   │              │   │              │                 │
│  └──────┬───────┘   └──────┬───────┘   └──────┬───────┘                 │
│         │                  │                  │                          │
│         └──────────────────┼──────────────────┘                          │
│                            ▼                                             │
│              ┌─────────────────────────────┐                            │
│              │      InstallService         │  ← NEW                     │
│              │  - Install(slug, opts)      │                            │
│              │  - Uninstall(slug, locs)    │                            │
│              │  - DetectPlatforms()        │                            │
│              │  - FetchSkillsFromURL()     │                            │
│              └──────────────┬──────────────┘                            │
│                             │                                            │
│         ┌───────────────────┼───────────────────┐                       │
│         ▼                   ▼                   ▼                        │
│  ┌────────────┐     ┌────────────┐     ┌────────────┐                   │
│  │  Installer │     │     DB     │     │  Scraper   │                   │
│  │ (symlinks) │     │  (skills)  │     │ (URL add)  │                   │
│  └────────────┘     └────────────┘     └────────────┘                   │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Component Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         NEW/MODIFIED COMPONENTS                          │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  internal/installer/                                                     │
│  ┌────────────────────────────────────────────────────────────────┐     │
│  │  service.go (NEW)                                              │     │
│  │  ├── InstallService struct                                     │     │
│  │  ├── Install(ctx, slug, opts) → InstallResult                  │     │
│  │  ├── InstallBatch(ctx, slugs, opts) → []InstallResult          │     │
│  │  ├── Uninstall(ctx, slug, locations) → error                   │     │
│  │  ├── UninstallAll(ctx, slug) → error                           │     │
│  │  ├── FetchSkillsFromURL(ctx, url) → []Skill                    │     │
│  │  ├── DetectPlatforms(ctx) → []PlatformInfo                     │     │
│  │  └── GetInstallLocations(ctx, slug) → []InstallLocation        │     │
│  └────────────────────────────────────────────────────────────────┘     │
│                                                                          │
│  internal/cli/                                                           │
│  ┌────────────────────────────────────────────────────────────────┐     │
│  │  install.go (NEW)           uninstall.go (NEW)                 │     │
│  │  ├── installCmd             ├── uninstallCmd                   │     │
│  │  ├── runInstall()           ├── runUninstall()                 │     │
│  │  └── runInstallURL()        └── (uses prompts/)                │     │
│  │                                                                 │     │
│  │  prompts/ (NEW)                                                 │     │
│  │  ├── platform.go  - PlatformSelector (huh multiselect)         │     │
│  │  ├── scope.go     - ScopeSelector (huh multiselect)            │     │
│  │  └── skills.go    - SkillSelector (huh multiselect)            │     │
│  └────────────────────────────────────────────────────────────────┘     │
│                                                                          │
│  internal/tui/                                                           │
│  ┌────────────────────────────────────────────────────────────────┐     │
│  │  app.go (MODIFY)                                               │     │
│  │  ├── Remove: installCmd, installToLocationsCmd,                │     │
│  │  │          installLocalSkillFromDetailCmd, installBatchSkillsCmd│   │
│  │  └── Add: installSkillCmd using InstallService                 │     │
│  └────────────────────────────────────────────────────────────────┘     │
│                                                                          │
│  internal/mcp/                                                           │
│  ┌────────────────────────────────────────────────────────────────┐     │
│  │  server.go (MODIFY)         handlers.go (MODIFY)               │     │
│  │  ├── Add installService     ├── handleInstall: use service     │     │
│  │  └── field to Server        ├── handleUninstall: use service   │     │
│  │                             └── Add platforms/scope params     │     │
│  └────────────────────────────────────────────────────────────────┘     │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Data Flow: Install Command

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    DATA FLOW: skulto install <slug>                      │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  1. Parse args/flags                                                     │
│     │                                                                    │
│     ├── slug provided? ──────────────────────────────────┐              │
│     │         │                                          │              │
│     │       Yes                                         No (URL?)       │
│     │         │                                          │              │
│     │         ▼                                          ▼              │
│     │  2. Load InstallService              2b. FetchSkillsFromURL()    │
│     │         │                                          │              │
│     │         │                                    Show SkillSelector   │
│     │         │                                          │              │
│     │         ▼                                          ▼              │
│     │  3. DetectPlatforms()  ◄───────────────────────────┘              │
│     │         │                                                         │
│     │         ▼                                                         │
│     │  4. -y flag set? ───────────────────────────────┐                │
│     │         │                                       │                │
│     │        No                                      Yes               │
│     │         │                                       │                │
│     │         ▼                                       │                │
│     │  5. Show PlatformSelector (huh)                 │                │
│     │         │                                       │                │
│     │         ▼                                       │                │
│     │  6. Show ScopeSelector (huh)                    │                │
│     │         │                                       │                │
│     │         └───────────────────────────────────────┘                │
│     │                         │                                         │
│     │                         ▼                                         │
│     │              7. service.Install(ctx, slug, opts)                  │
│     │                         │                                         │
│     │                         ▼                                         │
│     │              8. Print results (✓/✗ per location)                  │
│     │                                                                   │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Implementation Phases

### Phase Dependency Graph

```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│            ┌─────────────────────────────────────────┐              │
│            │        Phase 1: Core Service            │              │
│            │    (InstallService implementation)      │              │
│            │            Estimate: 5 pts              │              │
│            └──────────────────┬──────────────────────┘              │
│                               │                                      │
│                               ▼                                      │
│            ┌─────────────────────────────────────────┐              │
│            │        Phase 2: CLI Prompts             │              │
│            │     (huh-based interactive forms)       │              │
│            │            Estimate: 3 pts              │              │
│            └──────────────────┬──────────────────────┘              │
│                               │                                      │
│               ┌───────────────┴───────────────┐                     │
│               │                               │  ← PARALLEL         │
│               ▼                               ▼                      │
│        ┌─────────────┐                 ┌─────────────┐              │
│        │ Phase 3A    │                 │ Phase 3B    │              │
│        │ CLI Install │                 │ CLI Uninst. │              │
│        │   5 pts     │                 │   3 pts     │              │
│        └──────┬──────┘                 └──────┬──────┘              │
│               │                               │                      │
│               └───────────────┬───────────────┘                     │
│                               │                                      │
│               ┌───────────────┴───────────────┐                     │
│               │                               │  ← PARALLEL         │
│               ▼                               ▼                      │
│        ┌─────────────┐                 ┌─────────────┐              │
│        │ Phase 4     │                 │ Phase 5     │              │
│        │ TUI Refactor│                 │ MCP Updates │              │
│        │   5 pts     │                 │   3 pts     │              │
│        └─────────────┘                 └─────────────┘              │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

### Phase 1: Core Service (Foundation)

> **Depends On**: Nothing
> **Can Run With**: Nothing
> **Estimate**: 5 points
> **Status**: ✅ Complete

#### Objectives

- [x] Create `InstallService` struct with all dependencies
- [x] Implement `Install()` method
- [x] Implement `InstallBatch()` method
- [x] Implement `Uninstall()` and `UninstallAll()` methods
- [x] Implement `DetectPlatforms()` method
- [x] Implement `GetInstallLocations()` method
- [x] Implement `FetchSkillsFromURL()` method (stub - full implementation deferred)
- [x] Add comprehensive tests

#### Tasks

**Task 1.1: Create InstallService struct and constructor**

**Files:**
- Create: `internal/installer/service.go`
- Test: `internal/installer/service_test.go`

**Step 1: Write the failing test**
```go
// internal/installer/service_test.go
package installer

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestService(t *testing.T) (*InstallService, *db.DB) {
	t.Helper()
	tmpDir := t.TempDir()
	database, err := db.New(db.Config{Path: tmpDir + "/test.db"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })

	cfg := &config.Config{}
	service := NewInstallService(database, cfg, nil) // nil telemetry for tests
	return service, database
}

func TestNewInstallService(t *testing.T) {
	service, _ := setupTestService(t)

	assert.NotNil(t, service)
	assert.NotNil(t, service.installer)
	assert.NotNil(t, service.db)
}
```

**Step 2: Run test to verify it fails**
- Command: `go test ./internal/installer/... -run TestNewInstallService -v`
- Expected: FAIL - `undefined: NewInstallService`

**Step 3: Write minimal implementation**
```go
// internal/installer/service.go
package installer

import (
	"context"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/detect"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/scraper"
	"github.com/asteroid-belt/skulto/internal/telemetry"
)

// InstallOptions configures an install operation
type InstallOptions struct {
	Platforms []string       // nil = all user platforms
	Scopes    []InstallScope // nil = default to global
	Confirm   bool           // true = skip prompts
}

// InstallResult captures the outcome of an installation
type InstallResult struct {
	Skill     *models.Skill
	Locations []InstallLocation
	Errors    []error
}

// PlatformInfo describes a detected platform
type PlatformInfo struct {
	ID       string
	Name     string
	Path     string
	Detected bool
}

// InstallService provides unified installation operations
type InstallService struct {
	installer *Installer
	db        *db.DB
	cfg       *config.Config
	scraper   *scraper.Scraper
	detector  *detect.Detector
	telemetry telemetry.Client
}

// NewInstallService creates a new install service
func NewInstallService(database *db.DB, cfg *config.Config, tel telemetry.Client) *InstallService {
	if cfg == nil {
		cfg = &config.Config{}
	}

	inst := New(database, cfg)

	return &InstallService{
		installer: inst,
		db:        database,
		cfg:       cfg,
		detector:  detect.NewDetector(),
		telemetry: tel,
	}
}
```

**Step 4: Run test to verify it passes**
- Command: `go test ./internal/installer/... -run TestNewInstallService -v`
- Expected: PASS

**Step 5: Stage for commit**
```bash
git add internal/installer/service.go internal/installer/service_test.go
```

---

**Task 1.2: Implement DetectPlatforms()**

**Files:**
- Modify: `internal/installer/service.go`
- Test: `internal/installer/service_test.go`

**Step 1: Write the failing test**
```go
func TestInstallService_DetectPlatforms(t *testing.T) {
	service, _ := setupTestService(t)
	ctx := context.Background()

	platforms, err := service.DetectPlatforms(ctx)
	require.NoError(t, err)

	// Should return all known platforms with detection status
	assert.GreaterOrEqual(t, len(platforms), 6) // claude, cursor, windsurf, copilot, codex, opencode

	// Each platform should have required fields
	for _, p := range platforms {
		assert.NotEmpty(t, p.ID)
		assert.NotEmpty(t, p.Name)
		assert.NotEmpty(t, p.Path)
	}
}
```

**Step 2: Run test to verify it fails**
- Command: `go test ./internal/installer/... -run TestInstallService_DetectPlatforms -v`
- Expected: FAIL - `service.DetectPlatforms undefined`

**Step 3: Write minimal implementation**
```go
// DetectPlatforms returns all known platforms with detection status
func (s *InstallService) DetectPlatforms(ctx context.Context) ([]PlatformInfo, error) {
	allPlatforms := GetAllPlatforms()
	result := make([]PlatformInfo, 0, len(allPlatforms))

	for _, p := range allPlatforms {
		info := PlatformInfo{
			ID:       p.ID,
			Name:     p.Name,
			Path:     p.GlobalPath,
			Detected: s.detector.IsPlatformInstalled(p.ID),
		}
		result = append(result, info)
	}

	return result, nil
}
```

**Step 4: Run test to verify it passes**
- Command: `go test ./internal/installer/... -run TestInstallService_DetectPlatforms -v`
- Expected: PASS

---

**Task 1.3: Implement Install()**

**Files:**
- Modify: `internal/installer/service.go`
- Test: `internal/installer/service_test.go`

**Step 1: Write the failing test**
```go
func TestInstallService_Install(t *testing.T) {
	service, database := setupTestService(t)
	ctx := context.Background()

	// Seed a test skill with source
	source := &models.Source{ID: "src-1", Name: "test/repo", URL: "https://github.com/test/repo"}
	require.NoError(t, database.CreateSource(source))

	skill := &models.Skill{
		ID:       "skill-1",
		Slug:     "test-skill",
		Title:    "Test Skill",
		SourceID: &source.ID,
	}
	require.NoError(t, database.CreateSkill(skill))

	t.Run("install to specific platform and scope", func(t *testing.T) {
		opts := InstallOptions{
			Platforms: []string{PlatformClaude},
			Scopes:    []InstallScope{ScopeGlobal},
			Confirm:   true,
		}

		result, err := service.Install(ctx, "test-skill", opts)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-skill", result.Skill.Slug)
		assert.Len(t, result.Locations, 1)
	})

	t.Run("returns error for unknown skill", func(t *testing.T) {
		opts := InstallOptions{Confirm: true}
		_, err := service.Install(ctx, "nonexistent", opts)
		assert.Error(t, err)
	})
}
```

**Step 2: Run test to verify it fails**
- Command: `go test ./internal/installer/... -run TestInstallService_Install -v`
- Expected: FAIL - `service.Install undefined`

**Step 3: Write minimal implementation**
```go
// Install installs a skill to the specified locations
func (s *InstallService) Install(ctx context.Context, slug string, opts InstallOptions) (*InstallResult, error) {
	// Look up skill
	skill, err := s.db.GetSkillBySlug(slug)
	if err != nil {
		return nil, fmt.Errorf("skill not found: %s", slug)
	}

	// Get source if skill has one
	var source *models.Source
	if skill.SourceID != nil {
		source, _ = s.db.GetSourceByID(*skill.SourceID)
	}

	// Determine platforms
	platforms := opts.Platforms
	if len(platforms) == 0 {
		// Default to user's configured platforms
		userState, _ := s.db.GetUserState()
		if userState != nil && len(userState.SelectedPlatforms) > 0 {
			platforms = userState.SelectedPlatforms
		} else {
			platforms = []string{PlatformClaude} // fallback
		}
	}

	// Determine scopes
	scopes := opts.Scopes
	if len(scopes) == 0 {
		scopes = []InstallScope{ScopeGlobal}
	}

	// Build locations
	var locations []InstallLocation
	for _, platform := range platforms {
		for _, scope := range scopes {
			loc := InstallLocation{
				Platform: platform,
				Scope:    scope,
			}
			locations = append(locations, loc)
		}
	}

	// Perform installation
	if err := s.installer.InstallTo(ctx, skill, source, locations); err != nil {
		return nil, err
	}

	// Get actual installed locations
	installed, _ := s.installer.GetInstallLocations(skill.ID)

	// Track telemetry
	if s.telemetry != nil {
		s.telemetry.TrackSkillInstalled(skill.Title, skill.Category, skill.IsLocal, len(installed))
	}

	return &InstallResult{
		Skill:     skill,
		Locations: installed,
	}, nil
}
```

**Step 4: Run test to verify it passes**
- Command: `go test ./internal/installer/... -run TestInstallService_Install -v`
- Expected: PASS

---

**Task 1.4: Implement remaining service methods**

Similar TDD structure for:
- `InstallBatch()` - iterates and calls Install for each
- `Uninstall()` - removes from specific locations
- `UninstallAll()` - removes from all locations
- `GetInstallLocations()` - wraps installer method
- `FetchSkillsFromURL()` - uses scraper to add repo and return skills

(Detailed test/implementation code follows same pattern as above)

#### Definition of Done (Quality Gate)

- [x] Code passes `make lint`
- [x] Code passes `go fmt ./...`
- [x] Code passes `go vet ./...`
- [x] All new tests pass
- [x] All existing tests pass
- [x] Test coverage >= 80% for service.go
- [x] No new warnings introduced

- [x] **CHECKPOINT: Run `/compact focus on: Phase 1 complete, InstallService created with Install/Uninstall/DetectPlatforms methods, Phase 2 needs CLI prompts`**

---

### Phase 2: CLI Prompts

> **Depends On**: Phase 1
> **Can Run With**: Nothing
> **Estimate**: 3 points
> **Status**: ⬜ Not Started

#### Objectives

- [ ] Add `charmbracelet/huh` dependency
- [ ] Create `PlatformSelector` component
- [ ] Create `ScopeSelector` component
- [ ] Create `SkillSelector` component
- [ ] Add tests for prompt components

#### Tasks

**Task 2.1: Add huh dependency**

**Files:**
- Modify: `go.mod`

**Step 1: Run command**
```bash
go get github.com/charmbracelet/huh@latest
```

**Step 2: Verify import works**
```bash
go build ./...
```

---

**Task 2.2: Create PlatformSelector**

**Files:**
- Create: `internal/cli/prompts/platform.go`
- Test: `internal/cli/prompts/platform_test.go`

**Step 1: Write the failing test**
```go
// internal/cli/prompts/platform_test.go
package prompts

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/stretchr/testify/assert"
)

func TestBuildPlatformOptions(t *testing.T) {
	platforms := []installer.PlatformInfo{
		{ID: "claude", Name: "Claude", Path: "~/.claude/skills/", Detected: true},
		{ID: "cursor", Name: "Cursor", Path: "~/.cursor/skills/", Detected: false},
	}

	options := BuildPlatformOptions(platforms)

	assert.Len(t, options, 2)
	assert.Equal(t, "claude", options[0].Value)
	assert.Contains(t, options[0].Label, "Claude")
	assert.Contains(t, options[0].Label, "detected") // detected platforms marked
}

func TestFilterSelectedPlatforms(t *testing.T) {
	allPlatforms := []installer.PlatformInfo{
		{ID: "claude", Name: "Claude"},
		{ID: "cursor", Name: "Cursor"},
	}
	selected := []string{"claude"}

	result := FilterSelectedPlatforms(allPlatforms, selected)

	assert.Len(t, result, 1)
	assert.Equal(t, "claude", result[0].ID)
}
```

**Step 2: Run test to verify it fails**
- Command: `go test ./internal/cli/prompts/... -run TestBuildPlatformOptions -v`
- Expected: FAIL - `undefined: BuildPlatformOptions`

**Step 3: Write minimal implementation**
```go
// internal/cli/prompts/platform.go
package prompts

import (
	"fmt"

	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/charmbracelet/huh"
)

// PlatformOption represents a selectable platform
type PlatformOption struct {
	Value string
	Label string
}

// BuildPlatformOptions creates huh options from platform info
func BuildPlatformOptions(platforms []installer.PlatformInfo) []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(platforms))
	for _, p := range platforms {
		label := p.Name
		if p.Detected {
			label = fmt.Sprintf("%s (%s) ✓ detected", p.Name, p.Path)
		} else {
			label = fmt.Sprintf("%s (%s)", p.Name, p.Path)
		}
		options = append(options, huh.NewOption(label, p.ID))
	}
	return options
}

// FilterSelectedPlatforms returns only platforms matching selected IDs
func FilterSelectedPlatforms(all []installer.PlatformInfo, selected []string) []installer.PlatformInfo {
	selectedMap := make(map[string]bool)
	for _, s := range selected {
		selectedMap[s] = true
	}

	var result []installer.PlatformInfo
	for _, p := range all {
		if selectedMap[p.ID] {
			result = append(result, p)
		}
	}
	return result
}

// RunPlatformSelector shows interactive platform selection
func RunPlatformSelector(platforms []installer.PlatformInfo, preselected []string) ([]string, error) {
	options := BuildPlatformOptions(platforms)

	// Pre-select detected platforms if no preselection provided
	if len(preselected) == 0 {
		for _, p := range platforms {
			if p.Detected {
				preselected = append(preselected, p.ID)
			}
		}
	}

	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select platforms to install to").
				Options(options...).
				Value(&selected),
		),
	)

	// Set initial values
	selected = preselected

	if err := form.Run(); err != nil {
		return nil, err
	}

	return selected, nil
}
```

**Step 4: Run test to verify it passes**
- Command: `go test ./internal/cli/prompts/... -v`
- Expected: PASS

---

**Task 2.3: Create ScopeSelector and SkillSelector**

(Similar TDD structure - create scope.go and skills.go with tests)

#### Definition of Done (Quality Gate)

- [ ] Code passes `make lint`
- [ ] Code passes `go fmt ./...`
- [ ] Code passes `go vet ./...`
- [ ] All prompt tests pass
- [ ] All existing tests pass
- [ ] No new warnings introduced

- [ ] **CHECKPOINT: Run `/compact focus on: Phase 2 complete, huh-based prompts created (Platform/Scope/Skill selectors), Phase 3 needs CLI commands`**

---

### Phase 3A: CLI Install Command

> **Depends On**: Phase 2
> **Can Run With**: Phase 3B (PARALLEL)
> **Estimate**: 5 points
> **Status**: ⬜ Not Started

#### Objectives

- [ ] Create `skulto install` command
- [ ] Implement slug-based installation flow
- [ ] Implement URL-based installation flow
- [ ] Add -p, -s, -y flags
- [ ] Add tests

#### Tasks

**Task 3A.1: Create install command structure**

**Files:**
- Create: `internal/cli/install.go`
- Modify: `internal/cli/cli.go` (register command)
- Test: `internal/cli/install_test.go`

**Step 1: Write the failing test**
```go
// internal/cli/install_test.go
package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstallCmd_Structure(t *testing.T) {
	assert.Equal(t, "install", installCmd.Use)
	assert.NotEmpty(t, installCmd.Short)
	assert.NotEmpty(t, installCmd.Long)

	// Check flags exist
	pFlag := installCmd.Flags().Lookup("platform")
	assert.NotNil(t, pFlag)
	assert.Equal(t, "p", pFlag.Shorthand)

	sFlag := installCmd.Flags().Lookup("scope")
	assert.NotNil(t, sFlag)
	assert.Equal(t, "s", sFlag.Shorthand)

	yFlag := installCmd.Flags().Lookup("yes")
	assert.NotNil(t, yFlag)
	assert.Equal(t, "y", yFlag.Shorthand)
}

func TestInstallCmd_RequiresArg(t *testing.T) {
	err := installCmd.Args(installCmd, []string{})
	assert.Error(t, err) // Should require at least 1 arg
}
```

**Step 2: Run test to verify it fails**
- Command: `go test ./internal/cli/... -run TestInstallCmd -v`
- Expected: FAIL - `undefined: installCmd`

**Step 3: Write minimal implementation**
```go
// internal/cli/install.go
package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/asteroid-belt/skulto/internal/cli/prompts"
	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/spf13/cobra"
)

var (
	installPlatforms []string
	installScope     string
	installYes       bool
)

var installCmd = &cobra.Command{
	Use:   "install <slug|url>",
	Short: "Install a skill to AI tool directories",
	Long: `Install a skill by slug or from a repository URL.

Examples:
  skulto install docker-expert                    # Interactive mode
  skulto install docker-expert -p claude -y       # Non-interactive
  skulto install https://github.com/owner/repo   # Install from URL
  skulto install owner/repo                       # Shorthand URL`,
	Args: cobra.ExactArgs(1),
	RunE: runInstall,
}

func init() {
	installCmd.Flags().StringArrayVarP(&installPlatforms, "platform", "p", nil, "Platforms to install to (repeatable)")
	installCmd.Flags().StringVarP(&installScope, "scope", "s", "", "Installation scope: global or project")
	installCmd.Flags().BoolVarP(&installYes, "yes", "y", false, "Skip confirmation prompts")
}

func runInstall(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	input := args[0]

	// Load config and database
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	paths := config.GetPaths(cfg)
	database, err := db.New(db.Config{Path: paths.Database})
	if err != nil {
		return err
	}
	defer database.Close()

	// Create service
	service := installer.NewInstallService(database, cfg, getTelemetry())

	// Check if input is URL or slug
	if isURL(input) {
		return runInstallFromURL(ctx, service, input)
	}

	return runInstallBySlug(ctx, service, input)
}

func runInstallBySlug(ctx context.Context, service *installer.InstallService, slug string) error {
	// Detect platforms
	platforms, err := service.DetectPlatforms(ctx)
	if err != nil {
		return err
	}

	// Determine selected platforms
	selectedPlatforms := installPlatforms
	if !installYes && len(selectedPlatforms) == 0 {
		selectedPlatforms, err = prompts.RunPlatformSelector(platforms, installPlatforms)
		if err != nil {
			return err
		}
	}
	if len(selectedPlatforms) == 0 {
		// Default to detected platforms
		for _, p := range platforms {
			if p.Detected {
				selectedPlatforms = append(selectedPlatforms, p.ID)
			}
		}
	}

	// Determine scope
	var scopes []installer.InstallScope
	if installScope != "" {
		scopes = []installer.InstallScope{installer.InstallScope(installScope)}
	} else if !installYes {
		scopeStrs, err := prompts.RunScopeSelector(nil)
		if err != nil {
			return err
		}
		for _, s := range scopeStrs {
			scopes = append(scopes, installer.InstallScope(s))
		}
	}
	if len(scopes) == 0 {
		scopes = []installer.InstallScope{installer.ScopeGlobal}
	}

	// Perform installation
	opts := installer.InstallOptions{
		Platforms: selectedPlatforms,
		Scopes:    scopes,
		Confirm:   true,
	}

	fmt.Printf("Installing %s...\n", slug)
	result, err := service.Install(ctx, slug, opts)
	if err != nil {
		return err
	}

	// Print results
	for _, loc := range result.Locations {
		fmt.Printf("  ✓ %s\n", loc.Path)
	}
	fmt.Printf("\nDone! Installed to %d location(s).\n", len(result.Locations))

	return nil
}

func runInstallFromURL(ctx context.Context, service *installer.InstallService, url string) error {
	fmt.Printf("Fetching skills from %s...\n", url)

	// Fetch skills (this will auto-add repo if needed)
	skills, err := service.FetchSkillsFromURL(ctx, url)
	if err != nil {
		return err
	}

	fmt.Printf("Found %d skills.\n\n", len(skills))

	// Show skill selector if not -y
	var selectedSlugs []string
	if installYes {
		for _, s := range skills {
			selectedSlugs = append(selectedSlugs, s.Slug)
		}
	} else {
		selectedSlugs, err = prompts.RunSkillSelector(skills, nil)
		if err != nil {
			return err
		}
	}

	if len(selectedSlugs) == 0 {
		fmt.Println("No skills selected.")
		return nil
	}

	// Install each selected skill
	for _, slug := range selectedSlugs {
		if err := runInstallBySlug(ctx, service, slug); err != nil {
			fmt.Printf("  ✗ %s: %v\n", slug, err)
		}
	}

	return nil
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") ||
		strings.HasPrefix(s, "https://") ||
		strings.Contains(s, "/") && !strings.HasPrefix(s, ".")
}
```

**Step 4: Run test to verify it passes**
- Command: `go test ./internal/cli/... -run TestInstallCmd -v`
- Expected: PASS

**Step 5: Register command in cli.go**
```go
// Add to init() in cli.go
rootCmd.AddCommand(installCmd)
```

---

(Continue with Task 3A.2, 3A.3, etc. for URL handling and integration tests)

#### Definition of Done (Quality Gate)

- [ ] Code passes `make lint`
- [ ] Code passes `go fmt ./...`
- [ ] Code passes `go vet ./...`
- [ ] All install command tests pass
- [ ] All existing tests pass
- [ ] Manual test: `skulto install <slug>` works interactively
- [ ] Manual test: `skulto install <slug> -y` works non-interactively
- [ ] No new warnings introduced

- [ ] **CHECKPOINT: Run `/compact focus on: Phase 3A complete, skulto install command working with interactive/non-interactive modes and URL support, Phase 4/5 can proceed in parallel`**

---

### Phase 3B: CLI Uninstall Command

> **Depends On**: Phase 2
> **Can Run With**: Phase 3A (PARALLEL)
> **Estimate**: 3 points
> **Status**: ⬜ Not Started

(Similar structure to Phase 3A - create uninstall.go with location selector)

#### Definition of Done (Quality Gate)

- [ ] Code passes `make lint`
- [ ] Code passes `go fmt ./...`
- [ ] Code passes `go vet ./...`
- [ ] All uninstall command tests pass
- [ ] Manual test: `skulto uninstall <slug>` shows location picker
- [ ] Manual test: `skulto uninstall <slug> -y` removes from all
- [ ] No new warnings introduced

- [ ] **CHECKPOINT: Run `/compact focus on: Phase 3B complete, skulto uninstall command with location selection, Phase 4/5 can proceed`**

---

### Phase 4: TUI Refactor

> **Depends On**: Phase 1
> **Can Run With**: Phase 5 (PARALLEL)
> **Estimate**: 5 points
> **Status**: ⬜ Not Started

#### Objectives

- [ ] Add `InstallService` to TUI Model
- [ ] Replace `installCmd()` with service call
- [ ] Replace `installToLocationsCmd()` with service call
- [ ] Replace `installLocalSkillFromDetailCmd()` with service call
- [ ] Replace `installBatchSkillsCmd()` with service call
- [ ] Update tests

#### Tasks

**Task 4.1: Add InstallService to Model**

**Files:**
- Modify: `internal/tui/app.go:89-120` (Model struct)

**Step 1: Add field**
```go
type Model struct {
    // ... existing fields ...
    installService *installer.InstallService // NEW
}
```

**Step 2: Initialize in NewModel**
```go
func NewModel(...) *Model {
    // ... existing code ...
    m.installService = installer.NewInstallService(database, cfg, telemetry)
    return m
}
```

---

**Task 4.2: Replace install functions**

**Files:**
- Modify: `internal/tui/app.go:1906-2039`

Replace:
```go
func (m *Model) installCmd(skill *models.Skill, source *models.Source) tea.Cmd
func (m *Model) installToLocationsCmd(skill *models.Skill, source *models.Source, locations []installer.InstallLocation) tea.Cmd
func (m *Model) installLocalSkillFromDetailCmd(skill *models.Skill, ...) tea.Cmd
func (m *Model) installBatchSkillsCmd(skills []*models.Skill, ...) tea.Cmd
```

With single unified function:
```go
func (m *Model) installSkillCmd(slug string, opts installer.InstallOptions) tea.Cmd {
    return func() tea.Msg {
        result, err := m.installService.Install(context.Background(), slug, opts)
        if err != nil {
            return views.SkillInstalledMsg{Error: err}
        }
        return views.SkillInstalledMsg{
            Skill:     result.Skill,
            Locations: result.Locations,
        }
    }
}
```

#### Definition of Done (Quality Gate)

- [ ] Code passes `make lint`
- [ ] Code passes `go fmt ./...`
- [ ] Code passes `go vet ./...`
- [ ] All TUI tests pass
- [ ] Manual test: TUI 'i' key works as before
- [ ] No new warnings introduced

- [ ] **CHECKPOINT: Run `/compact focus on: Phase 4 complete, TUI now uses InstallService, 4 install functions replaced with 1, Phase 5 completes MCP`**

---

### Phase 5: MCP Updates

> **Depends On**: Phase 1
> **Can Run With**: Phase 4 (PARALLEL)
> **Estimate**: 3 points
> **Status**: ⬜ Not Started

#### Objectives

- [ ] Add `InstallService` to MCP Server
- [ ] Update `handleInstall()` to use service
- [ ] Add `platforms` and `scope` parameters to install tool
- [ ] Update `handleUninstall()` to use service
- [ ] Add `platforms` and `scope` parameters to uninstall tool
- [ ] Update tests

#### Tasks

**Task 5.1: Add InstallService to Server**

**Files:**
- Modify: `internal/mcp/server.go:25-45`

```go
type Server struct {
    // ... existing fields ...
    installService *installer.InstallService // NEW
}

func NewServer(database *db.DB, cfg *config.Config, favStore *favorites.Store) *Server {
    // ... existing code ...
    s.installService = installer.NewInstallService(database, cfg, nil)
    return s
}
```

---

**Task 5.2: Update handleInstall with optional params**

**Files:**
- Modify: `internal/mcp/handlers.go:279-352`

**Step 1: Update tool definition**
```go
mcp.NewTool("skulto_install",
    mcp.WithDescription("Install a skill to Claude Code. Optionally specify platforms and scope."),
    mcp.WithString("slug", mcp.Required(), mcp.Description("The skill's unique slug identifier")),
    mcp.WithArray("platforms", mcp.Description("Platforms to install to. Options: claude, cursor, windsurf, copilot, codex, opencode. Default: all user platforms")),
    mcp.WithString("scope", mcp.Description("Installation scope: 'global' (user-wide) or 'project' (current directory). Default: global")),
)
```

**Step 2: Update handler**
```go
func (s *Server) handleInstall(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    slug, ok := req.Params.Arguments["slug"].(string)
    if !ok || slug == "" {
        return mcp.NewToolResultError("slug parameter is required"), nil
    }

    // Parse optional platforms
    var platforms []string
    if p, ok := req.Params.Arguments["platforms"].([]interface{}); ok {
        for _, v := range p {
            if s, ok := v.(string); ok {
                platforms = append(platforms, s)
            }
        }
    }

    // Parse optional scope
    scope := "global"
    if s, ok := req.Params.Arguments["scope"].(string); ok && s != "" {
        scope = s
    }

    // Build options
    opts := installer.InstallOptions{
        Platforms: platforms, // nil means all user platforms
        Scopes:    []installer.InstallScope{installer.InstallScope(scope)},
        Confirm:   true,
    }

    // Use service
    result, err := s.installService.Install(ctx, slug, opts)
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }

    // Build response
    // ... format result ...
}
```

#### Definition of Done (Quality Gate)

- [ ] Code passes `make lint`
- [ ] Code passes `go fmt ./...`
- [ ] Code passes `go vet ./...`
- [ ] All MCP tests pass
- [ ] Manual test: MCP install with platforms param works
- [ ] Manual test: MCP install with scope param works
- [ ] No new warnings introduced

- [ ] **CHECKPOINT: Run `/compact focus on: Phase 5 COMPLETE, all phases done, install refactor complete, unified InstallService used by CLI/TUI/MCP`**

---

## Testing Strategy

### Testing Pyramid

```
                    /\
                   /  \         Manual E2E
                  /    \        - CLI interactive flows
                 /──────\       - TUI 'i' key behavior
                /        \
               /──────────\     Integration Tests
              /            \    - Service + DB
             /              \   - CLI command execution
            /────────────────\
           /                  \ Unit Tests
          /                    \ - Service methods
         /                      \ - Prompt builders
        /────────────────────────\ - Command structure
```

### Test Commands

```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific package
go test ./internal/installer/... -v
go test ./internal/cli/... -v
go test ./internal/mcp/... -v

# Run specific test
go test ./internal/installer/... -run TestInstallService_Install -v
```

### Coverage Requirements

| Package | Minimum | Target |
|---------|---------|--------|
| installer/service.go | 80% | 90% |
| cli/install.go | 70% | 80% |
| cli/uninstall.go | 70% | 80% |
| cli/prompts/*.go | 60% | 70% |

---

## Assumptions & Risks

### Assumptions

| # | Assumption | Risk if Wrong | Mitigation |
|---|------------|---------------|------------|
| 1 | charmbracelet/huh works for inline forms | Need alternative UI | Fallback to raw stdin |
| 2 | Existing Installer API is sufficient | Need API changes | Extend Installer if needed |
| 3 | Platform detection works reliably | Wrong defaults | Add manual override |

### Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Breaking TUI behavior | Medium | High | Extensive manual testing |
| MCP backward compatibility | Low | Medium | Optional params only |
| huh library issues | Low | Medium | Can fallback to simpler prompts |

---

## Verification Checklist

### Final Acceptance Tests

- [ ] `skulto install docker-expert` - interactive platform/scope selection works
- [ ] `skulto install docker-expert -p claude -s global -y` - non-interactive works
- [ ] `skulto install https://github.com/asteroid-belt/skills` - URL flow works
- [ ] `skulto uninstall docker-expert` - interactive location selection works
- [ ] `skulto uninstall docker-expert -y` - removes from all locations
- [ ] TUI 'i' key - installs via dialog (behavior unchanged)
- [ ] MCP `skulto_install` - accepts platforms/scope params
- [ ] MCP `skulto_uninstall` - accepts platforms/scope params
- [ ] `go test ./...` - all tests pass
- [ ] `make lint` - no lint errors

---

## Conventional Commit Messages

```
Phase 1:
feat(installer): add InstallService for unified installation

Phase 2:
feat(cli): add huh-based interactive prompts

Phase 3A:
feat(cli): add skulto install command with interactive mode

Phase 3B:
feat(cli): add skulto uninstall command with location selection

Phase 4:
refactor(tui): use InstallService for all installations

Phase 5:
feat(mcp): add platforms and scope params to install/uninstall tools
```

---

## Execution Options

After approval, execute with:

```
Option 1: Execute Now (This Session)
  Run `/superbuild @~/.claude/plans/peppy-wiggling-mochi.md`

Option 2: Execute in Fresh Session
  Start new session and run `/superbuild @~/.claude/plans/peppy-wiggling-michi.md`

Option 3: Review First
  Read through the plan, suggest modifications, then execute
```
