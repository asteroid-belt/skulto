# Favorites Feature Implementation Plan

## Overview

Implement a persistent favorites feature that allows users to save their favorite skills. Favorites persist across database resets and install state changes by storing data in a separate JSON file (`~/.skulto/favorites.json`).

## Requirements

| Requirement | Description |
|-------------|-------------|
| R1 | Users can add/remove skills to/from favorites |
| R2 | Favorites persist across database resets |
| R3 | Favorites are independent from installation state |
| R4 | TUI: `f` key toggles favorite on skill detail view |
| R5 | CLI: `skulto favorites add/remove/list` commands |
| R6 | MCP: `skulto_bookmark` and `skulto_get_bookmarks` tools work correctly |

## Architecture

### Data Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         FAVORITES DATA FLOW                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ~/.skulto/favorites.json                                                │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  {                                                                │   │
│  │    "version": 1,                                                  │   │
│  │    "favorites": [                                                 │   │
│  │      { "slug": "docker-expert", "added_at": "2026-01-26T..." },  │   │
│  │      { "slug": "react-best-practices", "added_at": "..." }       │   │
│  │    ]                                                              │   │
│  │  }                                                                │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                              │                                           │
│                              ▼                                           │
│         ┌─────────────────────────────────────────┐                     │
│         │       internal/favorites/favorites.go   │                     │
│         │                                         │                     │
│         │  - Load() → []Favorite                  │                     │
│         │  - Save([]Favorite)                     │                     │
│         │  - Add(slug string) error               │                     │
│         │  - Remove(slug string) error            │                     │
│         │  - IsFavorite(slug string) bool         │                     │
│         │  - List() []Favorite                    │                     │
│         └─────────────────────────────────────────┘                     │
│                              │                                           │
│              ┌───────────────┼───────────────┐                          │
│              │               │               │                          │
│              ▼               ▼               ▼                          │
│         ┌─────────┐    ┌─────────┐    ┌─────────────┐                   │
│         │   TUI   │    │   CLI   │    │ MCP Server  │                   │
│         │  (f key)│    │favorites│    │ (bookmark)  │                   │
│         └─────────┘    └─────────┘    └─────────────┘                   │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Component Tree

```
internal/
├── favorites/           # NEW PACKAGE
│   ├── favorites.go     # Core logic: Load, Save, Add, Remove, IsFavorite
│   └── favorites_test.go
├── cli/
│   ├── cli.go           # MODIFY: Register favoritesCmd
│   └── favorites.go     # NEW: add/remove/list subcommands
├── tui/
│   └── views/
│       └── detail.go    # MODIFY: Add 'f' key handler
├── mcp/
│   ├── handlers.go      # MODIFY: Fix bookmark handlers
│   └── server.go        # MODIFY: Pass favorites store
└── config/
    └── paths.go         # MODIFY: Add Favorites path
```

## Phases

### Phase 0: Foundation (Estimate: 2)

Set up the core favorites package with file-based persistence.

**Tasks:**

- [x] Task 0.1: Add `Favorites` path to `config/paths.go`
  - File: `internal/config/paths.go:9-15` (MODIFY)
  - Add `Favorites string` field to `Paths` struct
  - Update `GetPaths()` to set `Favorites: filepath.Join(cfg.BaseDir, "favorites.json")`

- [x] Task 0.2: Create `internal/favorites/favorites.go`
  - File: `internal/favorites/favorites.go` (CREATE)
  - Define `Favorite` struct: `{Slug string, AddedAt time.Time}`
  - Define `Store` struct: `{path string, mu sync.RWMutex, cache *fileData}`
  - Define `fileData` struct: `{Version int, Favorites []Favorite}`
  - Implement `NewStore(path string) *Store`
  - Implement `Load() error` - reads JSON, initializes if missing
  - Implement `Save() error` - writes JSON atomically
  - Implement `Add(slug string) error` - adds if not present
  - Implement `Remove(slug string) error` - removes if present
  - Implement `IsFavorite(slug string) bool`
  - Implement `List() []Favorite`
  - Implement `Count() int`

- [x] Task 0.3: Create `internal/favorites/favorites_test.go`
  - File: `internal/favorites/favorites_test.go` (CREATE)
  - Test: `TestStore_AddAndRemove` - verify add/remove operations
  - Test: `TestStore_IsFavorite` - verify lookup works
  - Test: `TestStore_Persistence` - verify JSON file is created and read correctly
  - Test: `TestStore_EmptyFile` - verify graceful handling of missing/empty file
  - Test: `TestStore_Concurrent` - verify thread safety with parallel operations

- [x] **CHECKPOINT: Run `/compact focus on: Phase 0 complete, favorites package created with file persistence, Phase 1 needs CLI commands`**

### Phase 1: CLI Commands (Estimate: 3)

Implement the `skulto favorites` CLI with add/remove/list subcommands.

**Tasks:**

- [x] Task 1.1: Create `internal/cli/favorites.go`
  - File: `internal/cli/favorites.go` (CREATE)
  - Define `favoritesCmd` - parent command with Long description
  - Define `favoritesAddCmd` - `add <slug>`, validates skill exists in DB
  - Define `favoritesRemoveCmd` - `remove <slug>`
  - Define `favoritesListCmd` - `list`, shows table of favorites with skill titles
  - Use pattern: `config.Load()` → `config.GetPaths()` → `favorites.NewStore()`
  - Add telemetry tracking: `TrackFavoriteAdded`, `TrackFavoriteRemoved`, `TrackFavoritesList`

- [x] Task 1.2: Register favorites command in `cli.go`
  - File: `internal/cli/cli.go:61-69` (MODIFY)
  - Add `rootCmd.AddCommand(favoritesCmd)` in `init()`

- [x] Task 1.3: Add telemetry methods for favorites
  - File: `internal/telemetry/client.go` (MODIFY)
  - Add `TrackFavoriteAdded(slug string)`
  - Add `TrackFavoriteRemoved(slug string)`
  - Add `TrackFavoritesListed(count int)`
  - File: `internal/telemetry/noop.go` (MODIFY)
  - Add no-op implementations

- [x] Task 1.4: Create `internal/cli/favorites_test.go`
  - File: `internal/cli/favorites_test.go` (CREATE)
  - Test: `TestFavoritesCmd_Structure` - verify command structure
  - Test: `TestFavoritesAddCmd_ArgsValidation` - verify requires 1 arg
  - Test: `TestFavoritesRemoveCmd_ArgsValidation` - verify requires 1 arg
  - Test: `TestFavoritesListCmd_NoArgs` - verify requires 0 args

- [x] **CHECKPOINT: Run `/compact focus on: Phase 1 complete, CLI favorites add/remove/list working, Phase 2 needs TUI integration`**

### Phase 2: TUI Integration (Estimate: 3)

Add `f` key binding to toggle favorites in skill detail view.

**Tasks:**

- [x] Task 2.1: Add favorites store to TUI app
  - File: `internal/tui/app.go` (MODIFY)
  - Add `favorites *favorites.Store` field to `Model` struct
  - Initialize in `New()` or `Init()` using `config.GetPaths()`

- [x] Task 2.2: Add favorite state to skill display
  - File: `internal/tui/views/detail.go` (MODIFY)
  - Add `favorites *favorites.Store` field to `DetailView` struct
  - Add `isFavorite bool` field (cached state)
  - Update `NewDetailView()` to accept favorites store
  - Update `HandleSkillLoaded()` to check `favorites.IsFavorite(skill.Slug)`

- [x] Task 2.3: Implement `f` key handler
  - File: `internal/tui/views/detail.go:264-306` (MODIFY)
  - Add case `"f"` in `Update()` switch
  - Toggle favorite: `if isFavorite { favorites.Remove(slug) } else { favorites.Add(slug) }`
  - Update `isFavorite` state
  - Track telemetry

- [x] Task 2.4: Update detail view rendering
  - File: `internal/tui/views/detail.go` (MODIFY)
  - Add `renderFavoriteIndicator()` method
  - Show `[★ Favorite]` or `[☆ Add to favorites (f)]` in metadata section
  - Update `renderScrollIndicator()` to show `f (favorite)` in keybindings

- [x] Task 2.5: Update keyboard commands help
  - File: `internal/tui/views/detail.go:813-835` (MODIFY)
  - Add `{Key: "f", Description: "Toggle favorite"}` to `GetKeyboardCommands()`

- [x] Task 2.6: Add favorites view/filter to search (optional enhancement) - DEFERRED to v2
  - File: `internal/tui/views/search.go` (MODIFY)
  - Consider: Add filter mode to show only favorites
  - This can be deferred to v2

- [x] **CHECKPOINT: Run `/compact focus on: Phase 2 complete, TUI f-key toggles favorites, Phase 3 needs MCP fix`**

### Phase 3: MCP Server Fix (Estimate: 2)

Fix the MCP bookmark tools to use the favorites store instead of installed state.

**Tasks:**

- [x] Task 3.1: Add favorites store to MCP server
  - File: `internal/mcp/server.go` (MODIFY)
  - Add `favorites *favorites.Store` field to `Server` struct
  - Update `NewServer()` to accept and store favorites
  - Update constructor call sites (cmd/skulto-mcp/main.go)

- [x] Task 3.2: Fix `handleBookmark` handler
  - File: `internal/mcp/handlers.go:383-429` (MODIFY)
  - Replace `s.db.AddInstalled(skill.ID)` with `s.favorites.Add(skill.Slug)`
  - Replace `s.db.RemoveInstalled(skill.ID)` with `s.favorites.Remove(skill.Slug)`
  - Update messages to use "favorites" terminology

- [x] Task 3.3: Fix `handleGetBookmarks` handler
  - File: `internal/mcp/handlers.go:431-462` (MODIFY)
  - Replace `s.db.GetInstalled()` with `s.favorites.List()`
  - Look up skill details from DB by slug for each favorite
  - Handle case where favorited skill no longer exists in DB

- [x] Task 3.4: Update MCP main to initialize favorites
  - File: `cmd/skulto-mcp/main.go` (MODIFY)
  - Initialize favorites store with correct path
  - Pass to `NewServer()`

- [x] Task 3.5: Update MCP tests
  - File: `internal/mcp/handlers_test.go` (MODIFY)
  - Update tests to use favorites store
  - Add test: bookmark persists to file
  - Add test: bookmarks survive server restart

- [x] **CHECKPOINT: Run `/compact focus on: Phase 3 complete, MCP bookmark tools use favorites.json, Phase 4 needs cleanup`**

### Phase 4: Cleanup & Documentation (Estimate: 1)

Clean up legacy bookmark/installed conflation and update docs.

**Tasks:**

- [x] Task 4.1: Update README with favorites documentation
  - File: `README.md` (MODIFY)
  - Add section on favorites feature
  - Document CLI commands
  - Document TUI keybinding
  - Note MCP tools

- [x] Task 4.2: Add favorites to MCP server README
  - File: `cmd/skulto-mcp/README.md` (MODIFY if exists, or inline in main README)
  - Document `skulto_bookmark` tool
  - Document `skulto_get_bookmarks` tool
  - Note: MCP tools documented in main README, no separate MCP README exists

- [x] Task 4.3: Verify installed vs favorites separation
  - Ensure `Installed` model is only used for actual installation tracking
  - Ensure favorites are only in `~/.skulto/favorites.json`
  - No conflation between the two concepts
  - Verified: "Installed" = skills symlinked to AI tool dirs (database-backed)
  - Verified: "Favorites" = user's saved skills (file-backed in ~/.skulto/favorites.json)

- [x] **CHECKPOINT: Run `/compact focus on: Phase 4 complete, documentation updated, feature ready for review`**

## Definition of Done (Per Phase)

- [x] Code passes `golangci-lint run`
- [x] Code passes `go fmt ./...`
- [x] Code passes `go vet ./...`
- [x] All new tests pass (`go test ./...`)
- [x] All existing tests pass
- [x] Test coverage >= 80% for new code
- [x] No new warnings introduced

## Test Plan

### Unit Tests

| Test | Package | Description |
|------|---------|-------------|
| `TestStore_AddAndRemove` | favorites | Add then remove a favorite |
| `TestStore_IsFavorite` | favorites | Check favorite lookup |
| `TestStore_Persistence` | favorites | Verify JSON file read/write |
| `TestStore_EmptyFile` | favorites | Handle missing/empty file |
| `TestStore_Concurrent` | favorites | Thread safety |
| `TestFavoritesCmd_Structure` | cli | CLI command structure |
| `TestFavoritesAddCmd` | cli | Add command args validation |
| `TestFavoritesRemoveCmd` | cli | Remove command args validation |

### Integration Tests

| Test | Description |
|------|-------------|
| CLI add → list shows favorite | End-to-end CLI flow |
| TUI f-key → favorites.json updated | TUI integration |
| MCP bookmark → favorites.json updated | MCP integration |
| DB reset → favorites preserved | Persistence across resets |

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| File corruption | Atomic writes (write to temp, then rename) |
| Concurrent access | sync.RWMutex in Store |
| Orphaned favorites (skill deleted) | Graceful handling - show warning, offer cleanup |

## Conventional Commit Messages

```
Phase 0:
feat(favorites): add file-based favorites persistence

Phase 1:
feat(cli): add favorites add/remove/list commands

Phase 2:
feat(tui): add f-key to toggle favorites in detail view

Phase 3:
fix(mcp): use favorites store for bookmark tools

Phase 4:
docs: add favorites feature documentation
```

## Execution Options

After approval, execute with:

```
Option 1: Execute Now (This Session)
  Run `/superbuild docs/favorites-feature-plan.md`

Option 2: Execute in Fresh Session
  Start new session and run `/superbuild docs/favorites-feature-plan.md`

Option 3: Review First
  Read through the plan, suggest modifications, then execute
```
