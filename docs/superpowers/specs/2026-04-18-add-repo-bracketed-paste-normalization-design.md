# Add Repo Bracketed Paste Normalization Design

Date: 2026-04-18
Status: Draft (approved in-session)
Scope: TUI Add Repo dialog (`ViewAddSource`)

## Problem

The Add Repo dialog currently receives a lossy `key string` abstraction and converts it back into `tea.KeyMsg` for `textinput`. This drops richer key metadata and makes bracketed paste handling unreliable for multi-character clipboard content.

We need paste support that:
- uses terminal bracketed paste content
- normalizes pasted URLs to `https://{repo_host}/{repo_org}/{skill_repo_name}`
- strips query parameters and fragments
- truncates extra path segments beyond `{repo_org}/{skill_repo_name}`

## Goals

- Support bracketed paste in Add Repo dialog.
- Normalize pasted URLs into canonical repo root format.
- Preserve existing typing/navigation behavior in Add Repo.
- Keep submit-time repository validation as final guardrail.

## Non-Goals

- No explicit clipboard hotkey integration (`Ctrl+V`/`Cmd+V`) in this change.
- No cross-view refactor for all text-input views.
- No change to CLI add-source behavior.

## Approved Approach

Approach 2 (approved): pass raw `tea.KeyMsg` to Add Source view and handle bracketed paste explicitly inside the view.

## Architecture

1. Update `internal/tui/app.go`:
- In `ViewAddSource` branch, pass raw `tea.KeyMsg` into Add Source view update method.

2. Update `internal/tui/views/add_source.go`:
- Introduce `UpdateKey(msg tea.KeyMsg) (shouldGoBack bool, wasSuccessful bool)`.
- Keep key handling for `esc` and `enter`.
- For paste events, run a dedicated paste sanitizer/normalizer.
- For non-paste keys, call `textinput.Update(msg)` directly.

3. Keep existing integration:
- `GetRepositoryURL()` remains the source of submitted value.
- Existing submit path continues to validate with `scraper.ParseRepositoryURL`.

## Paste Normalization Rules

For bracketed paste content:

1. Trim leading/trailing whitespace and newlines.
2. Parse as URL.
3. Force scheme to `https`.
4. Remove query string and fragment.
5. Extract path segments and require at least two segments: `org/repo`.
6. Rebuild canonical form exactly as:
   `https://{host}/{org}/{repo}`
7. Ignore any additional segments after repo (e.g. `/tree/main`).

## Error Handling and UX

- If pasted value cannot be parsed into valid `host/org/repo` URL, leave existing input unchanged and show inline error.
- If normalization succeeds:
  - set input value to normalized URL
  - clear previous error
- Submit-time validation remains in place as a second check.

## Testing Plan

Add tests in `internal/tui/views/add_source_test.go`:

1. Bracketed paste with query/fragment normalizes correctly.
2. Bracketed paste with extra path segments truncates to repo root.
3. Invalid pasted URL keeps previous input and sets error.
4. Successful paste clears previous error.
5. Existing non-paste key input behavior remains intact.

Add/adjust `internal/tui/app.go` tests (if applicable in current suite) to ensure Add Source receives raw `tea.KeyMsg` flow.

## Risks and Mitigations

- Risk: terminal/platform differences in paste key events.
  - Mitigation: gate behavior on Bubble Tea paste key typing and keep non-paste path unchanged.
- Risk: over-normalization could hide user mistakes.
  - Mitigation: only normalize structural URL parts and keep strict submit validation.

## Rollout

- Internal behavior change only; no migration needed.
- No config flags required.

## Acceptance Criteria

- Pasting `https://github.com/org/repo?ref=main` yields `https://github.com/org/repo`.
- Pasting `https://github.com/org/repo/tree/main` yields `https://github.com/org/repo`.
- Invalid paste does not clobber existing input.
- Enter submit behavior still validates through `scraper.ParseRepositoryURL`.
- Existing Add Repo typing and navigation remain functional.
