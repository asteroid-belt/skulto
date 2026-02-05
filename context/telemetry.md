# Telemetry System Deep Dive

> This document details the telemetry system in Skulto.

## Overview

Skulto uses PostHog for anonymous usage analytics. Telemetry is **opt-out** (enabled by default) and never collects personal information, IP addresses, or custom/local data.

## Opt-Out

```bash
export SKULTO_TELEMETRY_TRACKING_ENABLED=false
```

Or set this environment variable persistently in your shell profile.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Telemetry Client                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────┐         ┌──────────────────────────┐  │
│  │  posthogClient  │         │      noopClient          │  │
│  │  (when enabled) │         │  (when disabled)         │  │
│  │                 │         │                          │  │
│  │  - PostHog SDK  │         │  - All methods no-op     │  │
│  │  - Async batch  │         │  - Zero overhead         │  │
│  │  - 5s interval  │         │                          │  │
│  └─────────────────┘         └──────────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Client Interface

All telemetry goes through the `telemetry.Client` interface:

```go
type Client interface {
    Track(event string, properties map[string]interface{})
    Close()
    GetTrackingID() string

    // CLI events
    TrackCLICommandExecuted(commandName string, hasFlags bool, durationMs int64)
    TrackRepoAdded(sourceID string, skillCount int)
    TrackRepoRemoved(sourceID string, skillCount int)
    TrackRepoSynced(sourceID string, added, removed, updated int)
    TrackRepoListed(sourceCount, totalSkillCount int)
    TrackConfigChanged(settingName string, isDefault bool)
    TrackSkillsDiscovered(count int, scopeGlobal, scopeProject bool)
    TrackSkillIngested(skillName, scope string)
    TrackCLIError(commandName, errorType string)
    TrackCLIHelpViewed(commandName string, cliArgs []string)
    TrackFavoriteAdded(slug string)
    TrackFavoriteRemoved(slug string)
    TrackFavoritesListed(count int)

    // TUI events
    TrackViewNavigated(viewName, previousView string)
    TrackSkillInstalled(skillName, category string, isLocal bool, platformCount int)
    TrackSkillUninstalled(skillName, category string, isLocal bool)
    TrackNewSkillCreatedSuccess(skillName string)
    TrackNewSkillCreatedFailure(errorMessage string)
    TrackSearchPerformed(query string, resultCount int, searchType string)
    TrackFilterApplied(filterType string, filterCount int)
    TrackSortChanged(sortField, sortDirection string)
    TrackSkillCopied(skillName string)
    TrackOnboardingCompleted(stepsViewed int, skipped bool)
    TrackOnboardingSkipped(stepName string)
    TrackSettingsChanged(settingName string)
    TrackKeyboardShortcut(shortcutKey, contextView string)
    TrackHelpViewed(contextView string)
    TrackPaginationUsed(direction string, pageNumber int)
    TrackTagBrowsingEntered(tagCount int)
    TrackTagSelected(tagName string)
    TrackErrorDisplayed(errorType, contextView string)
    TrackSourceSelected(sourceIndex, skillCount int)
    TrackListRefreshed(trigger string, skillCount int)

    // Shared events (CLI & TUI)
    TrackAppStarted(mode string, hasSources bool, sourceCount int)
    TrackAppExited(mode string, sessionDurationMs int64, commandsRun int)

    // Session events
    TrackSessionSummary(durationMs int64, viewsVisited, searchesPerformed, skillsInstalled, skillsUninstalled, reposAdded, reposRemoved int)

    // Shared events (all interfaces)
    TrackSkillViewed(slug, category string, isLocal bool)
    TrackSkillsListed(count int, source string)
    TrackStatsViewed()
    TrackRecentSkillsViewed(count int)
    TrackInstalledSkillsChecked(count int)

    // MCP events
    TrackMCPToolCalled(toolName string, durationMs int64, success bool)
}
```

## Event Catalog

### CLI Events

| Event Name | Description | Properties |
|------------|-------------|------------|
| `cli_command_executed` | CLI command run | `command_name`, `has_flags`, `execution_duration_ms` |
| `repo_added` | Repository added | `source_id`, `skill_count` |
| `repo_removed` | Repository removed | `source_id`, `skill_count` |
| `repo_synced` | Repository synced | `source_id`, `skills_added`, `skills_removed`, `skills_updated` |
| `repo_listed` | List command run | `source_count`, `total_skill_count` |
| `config_changed` | Config changed | `setting_name`, `is_default` |
| `skills_discovered` | Unmanaged skills discovered | `discovered_count`, `scope_global`, `scope_project` |
| `skill_ingested` | Discovered skill imported | `skill_name`, `scope` |
| `cli_error_occurred` | CLI error | `command_name`, `error_type` |
| `cli_help_viewed` | Help viewed | `command_name`, `cli_args` |
| `favorite_added` | Favorite added | `skill_slug` |
| `favorite_removed` | Favorite removed | `skill_slug` |
| `favorites_listed` | Favorites listed | `favorites_count` |

### TUI Events

| Event Name | Description | Properties |
|------------|-------------|------------|
| `view_navigated` | View changed | `view_name`, `previous_view` |
| `skill_installed` | Skill installed | `skill_name`, `skill_category`, `is_local`, `platform_count` |
| `skill_uninstalled` | Skill uninstalled | `skill_name`, `skill_category`, `is_local` |
| `new_skill_created_successfully` | Skill created | `skill_name` |
| `new_skill_created_failure` | Skill creation failed | `reason` |
| `search_performed` | Search executed | `query`, `query_length`, `result_count`, `search_type` |
| `filter_applied` | Filter applied | `filter_type`, `filter_count` |
| `sort_changed` | Sort changed | `sort_field`, `sort_direction` |
| `skill_copied` | Skill copied | `skill_name` |
| `onboarding_completed` | Onboarding done | `steps_viewed`, `skipped` |
| `onboarding_skipped` | Onboarding skipped | `step_name` |
| `settings_changed` | Settings changed | `setting_name` |
| `keyboard_shortcut_used` | Shortcut used | `shortcut_key`, `context_view` |
| `help_viewed` | Help viewed | `context_view` |
| `pagination_used` | Pagination | `direction`, `page_number` |
| `tag_browsing_entered` | Tag browse | `tag_count` |
| `tag_selected` | Tag selected | `tag_name` |
| `error_displayed` | Error shown | `error_type`, `context_view` |
| `source_selected` | Source selected | `source_index`, `skill_count` |
| `list_refreshed` | List refreshed | `trigger`, `skill_count` |

### Session Events

| Event Name | Description | Properties |
|------------|-------------|------------|
| `app_started` | App launched | `mode`, `has_sources`, `source_count` |
| `app_exited` | App exited | `mode`, `session_duration_ms`, `commands_run` |
| `session_summary` | Session stats | `duration_ms`, `views_visited`, `searches_performed`, `skills_installed`, `skills_uninstalled`, `repos_added`, `repos_removed` |

### Shared Events (All Interfaces)

| Event Name | Description | Properties |
|------------|-------------|------------|
| `skill_viewed` | Skill details viewed | `skill_slug`, `skill_category`, `is_local` |
| `skills_listed` | Skills listed | `result_count`, `source` |
| `stats_viewed` | Stats viewed | - |
| `recent_skills_viewed` | Recent skills viewed | `result_count` |
| `installed_skills_checked` | Installed check | `installed_count` |

### MCP Events

| Event Name | Description | Properties |
|------------|-------------|------------|
| `mcp_tool_called` | MCP tool invoked | `tool_name`, `duration_ms`, `success` |

## Base Properties

All events include these common properties:

```go
func baseProperties() map[string]interface{} {
    return map[string]interface{}{
        "os":         runtime.GOOS,      // darwin, linux, windows
        "arch":       runtime.GOARCH,    // amd64, arm64
        "version":    Version,           // e.g., "1.2.3"
        "prerelease": version.IsPrerelease(),
        "dev_build":  version.IsDevBuild(),
    }
}
```

## Tracking ID

The tracking ID is persistent across sessions, stored in the database:

```go
func New(provider TrackingIDProvider) Client {
    var sessionID string
    if provider != nil {
        sessionID = provider.GetOrCreateTrackingID()  // From DB
    } else {
        sessionID = uuid.New().String()  // Fallback
    }
    // ...
}
```

The database stores this in the `user_state` table:

```sql
CREATE TABLE user_state (
    id           INTEGER PRIMARY KEY,
    tracking_id  TEXT,
    -- other fields
);
```

## PostHog Configuration

```go
client, _ := posthog.NewWithConfig(PostHogAPIKey, posthog.Config{
    Endpoint:  "https://us.i.posthog.com",
    BatchSize: 250,
    Interval:  5 * time.Second,
})
```

Events are batched and sent every 5 seconds or when batch size reaches 250.

## Adding New Events

To add a new telemetry event:

### 1. Define Event Constant

In `internal/telemetry/events.go`:

```go
const (
    // ... existing events
    EventMyNewEvent = "my_new_event"
)
```

### 2. Add Tracking Method to posthogClient

```go
func (c *posthogClient) TrackMyNewEvent(param1 string, param2 int) {
    props := baseProperties()
    props["param1"] = param1
    props["param2"] = param2
    c.Track(EventMyNewEvent, props)
}
```

### 3. Add No-op Implementation

```go
func (c *noopClient) TrackMyNewEvent(param1 string, param2 int) {}
```

### 4. Add to Client Interface

```go
type Client interface {
    // ... existing methods
    TrackMyNewEvent(param1 string, param2 int)
}
```

### 5. Call from Interface Code

CLI example:
```go
func runMyCommand(cmd *cobra.Command, args []string) error {
    // ... command logic
    telemetryClient.TrackMyNewEvent("value", 42)
}
```

TUI example:
```go
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // ... on user action
    m.telemetry.TrackMyNewEvent("value", 42)
}
```

MCP example:
```go
func (s *Server) handleMyTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    start := time.Now()
    // ... handler logic
    s.telemetry.TrackMCPToolCalled("my_tool", time.Since(start).Milliseconds(), true)
}
```

## Privacy Considerations

### What We Track

- Feature usage frequency
- Error types (not messages)
- Session duration
- Navigation patterns
- Search query lengths (not content)

### What We Never Track

- Personal information
- IP addresses
- Skill content
- Repository URLs
- File paths
- Custom skill names
- User input text

### Data Minimization

All tracking methods use sanitized, aggregated data:

```go
// Good: Track query length, not content
props["query_length"] = len(query)

// Bad: Don't do this
// props["query"] = query  // Could contain sensitive data
```

## Testing

For tests, use the noopClient:

```go
func TestMyFeature(t *testing.T) {
    tc := telemetry.New(nil)  // Returns noopClient when disabled
    // ... test with tc
}
```

Or mock the interface:

```go
type mockTelemetry struct {
    events []string
}

func (m *mockTelemetry) TrackSkillInstalled(...) {
    m.events = append(m.events, "skill_installed")
}
```
