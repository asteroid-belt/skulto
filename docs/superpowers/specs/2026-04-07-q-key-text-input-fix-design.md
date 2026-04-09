# Fix `q`/`p`/`j`/`k` Eaten in Text Inputs

**Date:** 2026-04-07
**Status:** Approved, ready for implementation plan

## Problem

Pressing the `q` character in the search bar shows the quit-confirmation dialog instead of typing `q` into the input. The user cannot type queries like `bigquery`, `quack`, or `queue`. The same class of bug affects `j` and `k` in the tag view's `/` search mode (intercepted as result-navigation), and to a lesser degree `p` (latent — currently safe by accident).

## Root cause

Two distinct code paths create the same user-visible symptom:

**Class A — Globals in `app.go` swallow keys before any view sees them.**
At `internal/tui/app.go:697`, the `q` handler runs before view dispatch:

```go
if key == "q" {
    m.showingQuitConfirm = true
    return m, nil
}
```

A focused `textinput` never gets a chance to receive the keystroke. The same shape applies to `p` at line 720, which is currently gated to `ViewHome` (where there is no text input today, so it does not yet trigger user-visible bug reports).

**Class B — A view's own handler intercepts vim-letters for navigation while its text input is focused.**
At `internal/tui/views/tag.go:163-194`, `TagView.updateSearch()` handles `j`/`k` as result navigation in the same code branch that otherwise delegates printable characters to the search bar. The same anti-pattern exists in two branches of `internal/tui/views/search.go`:
- `updateTagGrid` (lines 195, 205) — when the tag grid is focused, the `default:` branch refocuses the search bar and types the character; but `j`/`k` are intercepted before the default branch.
- `updateSearchMode` unfocused branch (lines 267, 271) — same pattern.

## Goals

1. Typing `q`, `p`, `j`, or `k` into any focused text input should land that character in the input.
2. Typing a string that contains these characters (e.g., `bigquery`, `jq`, `kafka`, `kubectl`, `quack`) must work end-to-end.
3. When the user is inside a view whose default behavior is to route printable characters to a text input — even when the input is not the currently-focused widget — the global `q` and `p` shortcuts must not pre-empt that routing.
4. No regressions for views or dialogs that have no text inputs (`HomeView`, `DetailView`, `SettingsView`, `ManageView`, onboarding views, all dialog components). Vim-style `j`/`k` navigation continues to work in those locations.
5. `ctrl+c` continues to quit immediately from anywhere. `?` and `ctrl+r` continue to be handled globally.

## Non-goals

- Removing `j`/`k` navigation in views/dialogs that have no text inputs. This would be a UX-wide change unrelated to the typing-conflict bug.
- Fixing `h`/`l` interception in `SearchView.updateTagGrid` (lines 187, 191). Same bug class as `j`/`k` but deliberately out of scope; can be a one-line follow-up if needed.
- Fixing `?` and `ctrl+r` global interception. Same bug class but deliberately out of scope.
- Reordering view dispatch so that views handle keys before globals run. Too invasive for the size of this fix.

## Design

### Class A: gate `q` and `p` on a per-view "is accepting text input" predicate

Each view that contains a text input (or routes printable characters to one) exposes a method `IsAcceptingTextInput() bool`. The app model owns a helper `currentViewIsAcceptingTextInput()` that switches on `m.currentView` and asks the active view. The global `q` and `p` handlers in `app.go` are gated on this helper.

The naming `IsAcceptingTextInput` is deliberately broader than "has focused text input." It captures both the literal-focus case (TagView, AddSourceView) and the "this view treats every printable character as text input regardless of which sub-component currently holds focus" case (SearchView). The helper at the call site is the same in both cases.

### Class B: stop intercepting `j`/`k` for navigation in code branches where a text input is the typing target

In `TagView.updateSearch()`, `SearchView.updateTagGrid()`, and `SearchView.updateSearchMode()` unfocused branch, the `case "up", "k":` and `case "down", "j":` arms are reduced to `case "up":` and `case "down":`. The vim letters fall through to the `default:` branch, which delegates to (or refocuses) the search bar.

While search/tag-grid/results-list interaction is "active" in these branches, **navigation is via arrow keys only**. This matches standard convention: vim-style letter navigation is suspended while typing.

The `j`/`k` cases in `TagView.Update()` outer switch (lines 119, 126), used when search is **not** active, are untouched. Likewise all `j`/`k` cases in views and dialogs without text inputs are untouched.

### Why SearchView is `IsAcceptingTextInput() == true` always

After the Class B fix, every code branch in SearchView routes printable characters to the search bar:

| Branch | Search bar focused? | Printable char routing |
|---|---|---|
| `updateSearchBarInTagMode` | yes | `default:` → `searchBar.HandleKey` |
| `updateTagGrid` | no (tag grid focused) | `default:` → refocus + `searchBar.HandleKey` |
| `updateSearchMode` (focused) | yes | `default:` → `searchBar.HandleKey` |
| `updateSearchMode` (unfocused) | no (results focused) | `default:` → refocus + `searchBar.HandleKey` |

Therefore the global `q` and `p` shortcuts must always be suppressed inside SearchView; otherwise `j`/`k`/`a`/`b`/etc. would type while `q` would quit, in the same code branch.

**Behavioral consequence (accepted):** Once the user enters the search view, `q` no longer triggers the quit confirmation from anywhere within it. To quit, they must `esc` back to home, then `q`. This is consistent with how the existing `default:` branches treat all other letters and matches the user's mental model: "I might keep typing at any moment, so don't hijack letter keys."

## Code changes

### File 1: `internal/tui/views/search.go`

Add method:

```go
// IsAcceptingTextInput reports whether this view routes printable characters
// to a text input. Always true for SearchView: every navigation state
// (search bar focused, tag grid focused, results list focused) routes
// printable characters to the search bar via the existing default branches.
// The global q/p shortcuts must therefore be suppressed throughout the view.
func (sv *SearchView) IsAcceptingTextInput() bool {
    return true
}
```

Modify `updateTagGrid` (~lines 195, 205):

```go
case "up":   // was: "up", "k"
    atTop := sv.tagGrid.MoveUp()
    if atTop {
        sv.focusArea = FocusSearchBar
        sv.searchBar.Focus()
        sv.tagGrid.SetFocused(false)
    }
    return false, false, nil

case "down": // was: "down", "j"
    sv.tagGrid.MoveDown()
    return false, false, nil
```

Modify `updateSearchMode` unfocused branch (~lines 267, 271):

```go
case "up":   // was: "up", "k"
    sv.navigateUp()
    return false, false, nil

case "down": // was: "down", "j"
    sv.navigateDown()
    return false, false, nil
```

### File 2: `internal/tui/views/tag.go`

Add method:

```go
// IsAcceptingTextInput reports whether the search bar is currently accepting
// keystrokes. Returns true while searchActive even when navigating filtered
// results, because any printable key the user types is meant to extend the
// search query.
func (tv *TagView) IsAcceptingTextInput() bool {
    return tv.searchActive
}
```

Modify `updateSearch` (~lines 165, 172):

```go
case "up":   // was: "up", "k"
    if tv.selectedIdx > 0 {
        tv.selectedIdx--
        tv.adjustScroll()
    }
    return false, false

case "down": // was: "down", "j"
    if tv.selectedIdx < len(tv.filteredSkills)-1 {
        tv.selectedIdx++
        tv.adjustScroll()
    }
    return false, false
```

### File 3: `internal/tui/views/add_source.go`

Add method:

```go
// IsAcceptingTextInput reports whether the URL input is currently accepting
// keystrokes.
func (asv *AddSourceView) IsAcceptingTextInput() bool {
    return asv.input.Focused()
}
```

No other changes — `AddSourceView.Update()` already routes `j`/`k` to the input via its `default:` branch.

### File 4: `internal/tui/app.go`

Add helper near the existing view-dispatch code:

```go
// currentViewIsAcceptingTextInput reports whether the active view will route
// printable characters to a text input. Used to suppress single-character
// global shortcuts (q, p) that would otherwise be eaten before reaching the
// input.
func (m *Model) currentViewIsAcceptingTextInput() bool {
    switch m.currentView {
    case ViewSearch:
        return m.searchView.IsAcceptingTextInput()
    case ViewTag:
        return m.tagView.IsAcceptingTextInput()
    case ViewAddSource:
        return m.addSourceView.IsAcceptingTextInput()
    default:
        return false
    }
}
```

Modify the `q` global handler at line 697:

```go
if key == "q" && !m.currentViewIsAcceptingTextInput() {
    m.showingQuitConfirm = true
    return m, nil
}
```

Modify the `p` global handler at line 720:

```go
if key == "p" && m.currentView == ViewHome && !m.homeView.IsPulling() && !m.currentViewIsAcceptingTextInput() {
    // ... existing pull logic unchanged
}
```

The `p` gate is defensive — `p` is currently only fired on `ViewHome` and `HomeView` has no text input today, so the gate is a no-op in current state. Including it future-proofs `p` against `HomeView` ever gaining a text input and matches the user's stated scope decision.

## Test plan

### Unit tests on the new methods

- `TestSearchView_IsAcceptingTextInput_alwaysTrue` — sanity check that the method returns `true` regardless of focus state or query length.
- `TestTagView_IsAcceptingTextInput_followsSearchActive` — `searchActive=false → false`; `searchActive=true → true`; verify it stays `true` after `j`/`k` navigation in search mode.
- `TestAddSourceView_IsAcceptingTextInput_followsInputFocus` — focused → true; blurred → false.

### Behavioral tests on the Class A fix

- `TestApp_currentViewIsAcceptingTextInput` — for each `ViewXxx`, set `currentView` and assert the helper returns the expected value. This is the regression-prevention test: a future view with a text input that fails to wire up `IsAcceptingTextInput()` will fail this case.
- `TestApp_qSuppressedInSearchView` — set `currentView = ViewSearch`, dispatch `q`, assert `showingQuitConfirm == false` and the key reached `searchView.Update`.
- `TestApp_qSuppressedInTagViewWhenSearchActive` — `currentView = ViewTag`, `tv.searchActive = true`, dispatch `q`, assert `showingQuitConfirm == false`.
- `TestApp_qStillQuitsInTagViewWhenSearchInactive` — `currentView = ViewTag`, `tv.searchActive = false`, dispatch `q`, assert `showingQuitConfirm == true` (regression check).
- `TestApp_qStillQuitsOnHome` — regression check.
- `TestApp_qStillQuitsOnDetailView` — regression check.

### Behavioral tests on the Class B fix

- `TestTagView_updateSearch_jkAreTypable` — `searchActive=true`, dispatch `j`, `q`, `k`; assert query is `"jqk"` and `selectedIdx == 0` (did not navigate).
- `TestTagView_updateSearch_arrowsStillNavigate` — `searchActive=true` with multiple filtered results; dispatch `down`; assert `selectedIdx` advanced.
- `TestTagView_outerUpdate_jkStillNavigateWhenSearchInactive` — `searchActive=false`; dispatch `j`; assert `selectedIdx` advanced (regression check).
- `TestSearchView_updateTagGrid_jkAreTypable` — focus tag grid; dispatch `j`; assert search bar is focused, tag grid is blurred, query contains `"j"`.
- `TestSearchView_updateTagGrid_arrowsStillNavigate` — focus tag grid; dispatch `down`; assert grid moved, search bar still blurred.
- `TestSearchView_updateSearchMode_jkAreTypable` — query at 3+ chars, blur search bar onto results, dispatch `j`; assert `"j"` is appended to query and search bar refocused.
- `TestSearchView_updateSearchMode_arrowsStillNavigate` — same setup; dispatch `down`; assert results-list selection moved.

### End-to-end smoke

If the project has end-to-end TUI tests, simulate typing `bigquery` into the SearchView search bar by dispatching the eight keystrokes through the existing `searchView.Update(key)` path and assert:
1. The final query equals `"bigquery"`.
2. `model.showingQuitConfirm` is `false` throughout.
3. The search bar remains focused throughout (or becomes focused on the first keystroke).

Repeat for `kubectl`, `jq`, `kafka` in TagView's `/` search mode.

## Edge cases

| # | Scenario | Expected |
|---|---|---|
| 1 | Search bar focused, type `q` | `q` extends query |
| 2 | Search bar focused, type `bigquery` | full string lands in query |
| 3 | Search bar focused, type `p` | `p` extends query |
| 4 | Search view, results focused, type `q` | `q` refocuses bar and extends query (not quit) |
| 5 | Search view, tag grid focused, type `q` | `q` refocuses bar and extends query |
| 6 | Search view, results focused, type `j` | `j` refocuses bar and extends query (was: navigated) |
| 7 | Search view, results focused, press `down` arrow | results-list selection moves (regression check) |
| 8 | Tag view, `/` active, type `j` | `j` extends query (was: navigated) |
| 9 | Tag view, `/` active, type `kafka` | full string lands in query |
| 10 | Tag view, `/` active, press `down` arrow | filtered-results selection moves |
| 11 | Tag view, `/` NOT active, press `j` | navigates skill list (regression check) |
| 12 | Tag view, `/` NOT active, press `q` | quit dialog appears (regression check) |
| 13 | Add source view, type a URL with `q`/`p`/`j`/`k` (e.g., `github.com/foo/quack-jk`) | full URL lands in input |
| 14 | Home view, press `q` | quit dialog appears (regression check) |
| 15 | Detail view, press `q` | quit dialog appears (regression check) |
| 16 | Detail view, press `j`/`k` | scrolls (regression check) |
| 17 | Settings view, press `q` | quit dialog appears (regression check) |
| 18 | Settings view, press `j`/`k` | scrolls (regression check) |
| 19 | Any view, press `ctrl+c` | quits immediately (untouched) |
| 20 | Any view, press `?` | help opens (untouched, deliberately out of scope) |
| 21 | Any view, press `ctrl+r` | reset opens (untouched, deliberately out of scope) |

## Known limitations / follow-ups

- **`h`/`l` in `SearchView.updateTagGrid` (lines 187, 191)** still intercept those letters for left/right navigation. So a user cannot type "hello", "lambda", or "html" while the tag grid is focused without first refocusing the bar another way. Same bug class, deliberately out of scope per the user's stated preference for a surgical fix. One-line removal if/when desired.
- **`?` and `ctrl+r` globals** still hijack those keys regardless of input focus. Same `currentViewIsAcceptingTextInput()` helper makes each a one-line follow-up.
- **`SettingsView` defines its own local `q` handler** (line 191) that is currently unreachable because the global handler at `app.go:697` returns first. After this fix, the global still wins for SettingsView (it has no text input, so the helper returns `false`), so settings's local `q` remains dead code. Not in scope to clean up, but worth knowing.

## Surface area

Approximately 30 lines added, 6 lines modified, across 4 files. No deletions, no signature changes, no behavior change for any view or dialog without a text input.
