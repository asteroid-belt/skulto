# Feature Plan: `skulto_add` MCP Tool

## Summary

Add a new `skulto_add` MCP tool that adds a skill repository by URL, reusing the same logic as `skulto add` CLI and the TUI's 'a' button. The tool parses the URL, checks for duplicates, inserts the source into the database, scrapes all skills, and returns the list of scraped skills so the user can immediately install any of them.

## Requirements

- **MVP**: Always scrape immediately after adding (no `--no-sync` equivalent)
- **Duplicates**: Reject if source already exists (matches CLI behavior)
- **Scope**: Only the MCP tool — no changes to CLI or TUI
- **URL formats**: `owner/repo`, `https://github.com/owner/repo`, `.git` variants, SSH variants
- **Response**: Include list of scraped skill names/slugs so user can install immediately

## Tech Stack

- **Language**: Go 1.25+
- **MCP Library**: `mcp-go` (`github.com/mark3labs/mcp-go`)
- **Database**: SQLite via `internal/db`
- **Scraper**: `internal/scraper`

## Architecture

The new tool follows the exact pattern of all existing MCP tools:

```
tools.go:  addTool() → defines skulto_add with "url" param
handlers.go: handleAdd() → parses URL, checks dups, upserts, scrapes, lists skills
server.go: registerTools() → s.server.AddTool(addTool(), s.handleAdd)
```

### Data Flow

```
MCP Client calls skulto_add(url: "owner/repo")
    ↓
handleAdd():
  1. Parse URL via scraper.ParseRepositoryURL()
  2. Check for existing source via s.db.GetSource()
  3. Insert source via s.db.UpsertSource()
  4. Create scraper with s.cfg GitHub config
  5. ScrapeRepository() with 5-minute timeout
  6. Query s.db.GetSkillsBySourceID() to get scraped skills
  7. Return AddResult JSON with skills list
    ↓
MCP Client receives:
{
  "success": true,
  "message": "Repository 'owner/repo' added with 3 skills",
  "source": {"owner": "owner", "repo": "repo", "url": "..."},
  "skills_found": 3,
  "skills": [
    {"slug": "react-hooks", "title": "React Hooks Best Practices"},
    {"slug": "go-patterns", "title": "Go Design Patterns"},
    {"slug": "tdd-guide", "title": "Test-Driven Development Guide"}
  ]
}
```

### Response Types

```go
// AddResult represents the result of adding a repository.
type AddResult struct {
    Success     bool              `json:"success"`
    Message     string            `json:"message"`
    Source      *SourceResponse   `json:"source,omitempty"`
    SkillsFound int               `json:"skills_found"`
    Skills      []AddSkillResult  `json:"skills,omitempty"`
}

// AddSkillResult is a minimal skill reference returned after adding a repo.
type AddSkillResult struct {
    Slug  string `json:"slug"`
    Title string `json:"title"`
}
```

## Implementation Phases

| Phase | Name | Depends On | Parallel With | Estimate | Status |
|-------|------|------------|---------------|----------|--------|
| 1 | Add tool definition + handler + registration | None | - | 3 | ✅ |
| 2 | Add tests | Phase 1 | - | 3 | ✅ |

**Total estimate: 5 points** (some tasks overlap)

---

## Phase 1: Tool Definition, Handler & Registration (Est: 3)

### Task 1.1: Add tool definition — `internal/mcp/tools.go` (MODIFY)

Add at end of file:

```go
// addTool returns the skulto_add tool definition.
func addTool() mcp.Tool {
	return mcp.NewTool("skulto_add",
		mcp.WithDescription("Add a skill repository and sync its skills. Supports multiple URL formats: owner/repo, https://github.com/owner/repo, https://github.com/owner/repo.git, git@github.com:owner/repo.git"),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("Repository URL in any supported format (e.g., 'owner/repo' or full GitHub URL)"),
		),
	)
}
```

### Task 1.2: Add response types and handler — `internal/mcp/handlers.go` (MODIFY)

Add `AddResult`, `AddSkillResult` structs and `handleAdd` method. The handler reuses the same logic as `cli/add.go:runAdd()` and `tui/app.go:addSourceCmd()`:

```go
// AddResult represents the result of adding a repository.
type AddResult struct {
	Success     bool             `json:"success"`
	Message     string           `json:"message"`
	Source      *SourceResponse  `json:"source,omitempty"`
	SkillsFound int              `json:"skills_found"`
	Skills      []AddSkillResult `json:"skills,omitempty"`
}

// AddSkillResult is a minimal skill reference returned after adding a repo.
type AddSkillResult struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

// handleAdd handles the skulto_add tool.
func (s *Server) handleAdd(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, ok := req.Params.Arguments["url"].(string)
	if !ok || url == "" {
		return mcp.NewToolResultError("url parameter is required"), nil
	}

	// Parse and validate the repository URL
	source, err := scraper.ParseRepositoryURL(url)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid repository URL: %v", err)), nil
	}

	// Check if source already exists
	existing, err := s.db.GetSource(source.ID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to check existing source: %v", err)), nil
	}
	if existing != nil {
		return mcp.NewToolResultError(fmt.Sprintf("repository %s already exists", source.ID)), nil
	}

	// Add source to database
	if err := s.db.UpsertSource(source); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to add source: %v", err)), nil
	}

	// Create scraper and sync
	scraperCfg := scraper.ScraperConfig{
		Token:        s.cfg.GitHub.Token,
		DataDir:      s.cfg.BaseDir,
		RepoCacheTTL: s.cfg.GitHub.RepoCacheTTL,
		UseGitClone:  s.cfg.GitHub.UseGitClone,
	}
	sc := scraper.NewScraperWithConfig(scraperCfg, s.db)

	syncCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	result, err := sc.ScrapeRepository(syncCtx, source.Owner, source.Repo)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to sync %s: %v", source.ID, err)), nil
	}

	// Fetch the scraped skills to include in response
	skills, err := s.db.GetSkillsBySourceID(source.ID)
	var skillResults []AddSkillResult
	if err == nil {
		skillResults = make([]AddSkillResult, 0, len(skills))
		for _, skill := range skills {
			skillResults = append(skillResults, AddSkillResult{
				Slug:  skill.Slug,
				Title: skill.Title,
			})
		}
	}

	addResult := AddResult{
		Success: true,
		Message: fmt.Sprintf("Repository '%s/%s' added with %d skills", source.Owner, source.Repo, result.SkillsNew),
		Source: &SourceResponse{
			Owner: source.Owner,
			Repo:  source.Repo,
			URL:   source.URL,
		},
		SkillsFound: result.SkillsNew,
		Skills:      skillResults,
	}

	data, _ := json.Marshal(addResult)
	return mcp.NewToolResultText(string(data)), nil
}
```

New import needed: `"github.com/asteroid-belt/skulto/internal/scraper"`

### Task 1.3: Register tool — `internal/mcp/server.go:registerTools()` (MODIFY)

Add to `registerTools()`:

```go
// Repository management
s.server.AddTool(addTool(), s.handleAdd)
```

### Definition of Done — Phase 1
- [x] Code compiles (`go build ./...`)
- [x] Code passes linter
- [x] Code passes formatter (`gofmt`)
- [x] No new warnings introduced
- [x] **CHECKPOINT: Phase 1 complete**

---

## Phase 2: Tests (Est: 3)

### Task 2.1: Add handler tests — `internal/mcp/handlers_test.go` (MODIFY)

Tests to add, following the existing test patterns (using `setupTestDB`, `seedTestSkills`, direct handler calls):

```go
func TestHandleAdd(t *testing.T) {
	// Test 1: Missing url parameter → error
	// Test 2: Empty url parameter → error
	// Test 3: Invalid url format → error
	// Test 4: Duplicate source → error "already exists"
	// Test 5: Valid URL adds source to database (mock-free: just verify DB state)
}
```

**Note**: The scraper calls GitHub API, so full integration tests would require network access or mocking. The tests should focus on:
- Parameter validation (missing/empty/invalid URL)
- Duplicate detection (insert a source first, then try to add the same one)
- URL parsing validation (various formats)

For the scrape step, we accept that the handler will return an error about GitHub access in test environments — this matches how the TUI and CLI handle it (no scraper mocking exists in the codebase).

### Definition of Done — Phase 2
- [x] All new tests pass (`go test ./internal/mcp/...`)
- [x] All existing tests still pass
- [x] Code passes linter and formatter
- [x] **CHECKPOINT: Complete**

---

## Conventional Commit Message

```
feat(mcp): add skulto_add tool for adding skill repositories

Adds a new MCP tool that allows adding skill repositories via the MCP
protocol, reusing the same URL parsing and scraping logic as the CLI
`skulto add` command and TUI 'a' button. Returns list of scraped skills
so the user can immediately install them.

Files changed:
- internal/mcp/tools.go (MODIFY)
- internal/mcp/handlers.go (MODIFY)
- internal/mcp/server.go (MODIFY)
- internal/mcp/handlers_test.go (MODIFY)
```

---

## Execution Options

```
PLAN COMPLETE: docs/skulto-add-mcp-tool-plan.md
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EXECUTION OPTIONS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Option 1: Execute Now (This Session)
  Run `/superbuild docs/skulto-add-mcp-tool-plan.md`

Option 2: Execute in Fresh Session
  Start new session and run `/superbuild docs/skulto-add-mcp-tool-plan.md`

Option 3: Review First
  Read through the plan, suggest modifications, then execute

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Which option would you like?
```
