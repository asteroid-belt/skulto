package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/detect"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/scraper"
	"github.com/mark3labs/mcp-go/mcp"
)

// Pagination constants for MCP tool handlers.
const (
	defaultSearchLimit    = 20
	maxSearchLimit        = 100
	defaultListLimit      = 20
	maxListLimit          = 100
	defaultRecentLimit    = 10
	maxRecentLimit        = 50
	defaultFavoritesLimit = 50
	maxFavoritesLimit     = 100
)

// parseLimit extracts and validates a limit parameter from MCP tool arguments.
// Returns defaultVal if not present, caps at maxVal if exceeded.
func parseLimit(arguments map[string]interface{}, defaultVal, maxVal int) int {
	if l, ok := arguments["limit"].(float64); ok && l > 0 {
		limit := int(l)
		if limit > maxVal {
			return maxVal
		}
		return limit
	}
	return defaultVal
}

// trackToolCall is a helper to track MCP tool invocations.
func (s *Server) trackToolCall(toolName string, start time.Time, success bool) {
	if s.telemetry != nil {
		durationMs := time.Since(start).Milliseconds()
		s.telemetry.TrackMCPToolCalled(toolName, durationMs, success)
	}
}

// SkillResponse represents a skill in MCP tool responses.
type SkillResponse struct {
	ID          string          `json:"id"`
	Slug        string          `json:"slug"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Summary     string          `json:"summary,omitempty"`
	Content     string          `json:"content,omitempty"`
	Author      string          `json:"author,omitempty"`
	Difficulty  string          `json:"difficulty,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
	Source      *SourceResponse `json:"source,omitempty"`
	Stars       int             `json:"stars"`
	IsInstalled bool            `json:"is_installed"`
	Rank        float64         `json:"rank,omitempty"`
}

// SourceResponse represents a source repository in MCP responses.
type SourceResponse struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
	URL   string `json:"url"`
}

// TagResponse represents a tag in MCP tool responses.
type TagResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Category string `json:"category"`
	Color    string `json:"color,omitempty"`
	Count    int    `json:"count"`
}

// StatsResponse represents database statistics.
type StatsResponse struct {
	TotalSkills  int64 `json:"total_skills"`
	TotalTags    int64 `json:"total_tags"`
	TotalSources int64 `json:"total_sources"`
	Installed    int64 `json:"installed_count"`
}

// InstallResult represents the result of an install/uninstall operation.
type InstallResult struct {
	Success           bool                   `json:"success"`
	Message           string                 `json:"message"`
	Paths             []string               `json:"paths,omitempty"`              // Symlink paths created
	NeedsSelection    bool                   `json:"needs_selection,omitempty"`    // True when the LLM should ask the user to choose platforms
	DetectedPlatforms []DetectedPlatformInfo `json:"detected_platforms,omitempty"` // Available platforms for selection
}

// DetectedPlatformInfo describes a detected platform returned to the LLM for user selection.
type DetectedPlatformInfo struct {
	ID   string `json:"id"`   // Platform identifier (e.g. "cursor")
	Name string `json:"name"` // Human-readable name (e.g. "Cursor")
}

// CheckSkillResponse represents an installed skill in the check response.
type CheckSkillResponse struct {
	Slug      string                  `json:"slug"`
	Title     string                  `json:"title"`
	Locations []CheckLocationResponse `json:"locations"`
}

// CheckLocationResponse represents an installation location.
type CheckLocationResponse struct {
	Platform string `json:"platform"`
	Scope    string `json:"scope"`
}

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

// toSkillResponse converts a models.Skill to SkillResponse.
// Note: IsInstalled uses the skill record's legacy field.
// Use toSkillResponseWithDB for accurate installed status from skill_installations.
func toSkillResponse(skill *models.Skill, includeContent bool) SkillResponse {
	resp := SkillResponse{
		ID:          skill.ID,
		Slug:        skill.Slug,
		Title:       skill.Title,
		Description: skill.Description,
		Summary:     skill.Summary,
		Author:      skill.Author,
		Difficulty:  skill.Difficulty,
		Stars:       skill.Stars,
		IsInstalled: skill.IsInstalled, // Legacy field - may be stale
	}

	if includeContent {
		resp.Content = skill.Content
	}

	for _, tag := range skill.Tags {
		resp.Tags = append(resp.Tags, tag.Name)
	}

	if skill.Source != nil {
		resp.Source = &SourceResponse{
			Owner: skill.Source.Owner,
			Repo:  skill.Source.Repo,
			URL:   skill.Source.URL,
		}
	}

	return resp
}

// toSkillResponseWithDB converts a models.Skill to SkillResponse using
// skill_installations as the source of truth for installed status.
func toSkillResponseWithDB(skill *models.Skill, includeContent bool, database *db.DB) SkillResponse {
	resp := toSkillResponse(skill, includeContent)
	// Override IsInstalled with accurate value from skill_installations
	if database != nil {
		hasInstalls, _ := database.HasInstallations(skill.ID)
		resp.IsInstalled = hasInstalls
	}
	return resp
}

func (s *Server) handleSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	query, ok := req.Params.Arguments["query"].(string)
	if !ok || query == "" {
		s.trackToolCall("skulto_search", start, false)
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	limit := parseLimit(req.Params.Arguments, defaultSearchLimit, maxSearchLimit)

	skills, err := s.db.SearchSkills(query, limit)
	if err != nil {
		s.trackToolCall("skulto_search", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	results := make([]SkillResponse, 0, len(skills))
	for i := range skills {
		results = append(results, toSkillResponseWithDB(&skills[i], false, s.db))
	}

	data, err := json.Marshal(results)
	if err != nil {
		s.trackToolCall("skulto_search", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	// Track search telemetry
	if s.telemetry != nil {
		s.telemetry.TrackSearchPerformed(query, len(skills), "mcp")
	}

	s.trackToolCall("skulto_search", start, true)
	return mcp.NewToolResultText(string(data)), nil
}

// handleGetSkill handles the skulto_get_skill tool.
func (s *Server) handleGetSkill(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	slug, ok := req.Params.Arguments["slug"].(string)
	if !ok || slug == "" {
		s.trackToolCall("skulto_get_skill", start, false)
		return mcp.NewToolResultError("slug parameter is required"), nil
	}

	skill, err := s.db.GetSkillBySlug(slug)
	if err != nil {
		s.trackToolCall("skulto_get_skill", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get skill: %v", err)), nil
	}
	if skill == nil {
		s.trackToolCall("skulto_get_skill", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("skill not found: %s", slug)), nil
	}

	now := time.Now()
	skill.ViewedAt = &now
	_ = s.db.UpdateSkill(skill)

	resp := toSkillResponseWithDB(skill, true, s.db)

	data, err := json.Marshal(resp)
	if err != nil {
		s.trackToolCall("skulto_get_skill", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal skill: %v", err)), nil
	}

	// Track skill viewed telemetry
	if s.telemetry != nil {
		s.telemetry.TrackSkillViewed(skill.Slug, skill.Category, skill.IsLocal)
	}

	s.trackToolCall("skulto_get_skill", start, true)
	return mcp.NewToolResultText(string(data)), nil
}

// handleListSkills handles the skulto_list_skills tool.
func (s *Server) handleListSkills(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	limit := parseLimit(req.Params.Arguments, defaultListLimit, maxListLimit)

	offset := 0
	if o, ok := req.Params.Arguments["offset"].(float64); ok && o >= 0 {
		offset = int(o)
	}

	skills, err := s.db.ListSkills(limit, offset)
	if err != nil {
		s.trackToolCall("skulto_list_skills", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to list skills: %v", err)), nil
	}

	results := make([]SkillResponse, 0, len(skills))
	for i := range skills {
		results = append(results, toSkillResponseWithDB(&skills[i], false, s.db))
	}

	data, err := json.Marshal(results)
	if err != nil {
		s.trackToolCall("skulto_list_skills", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	// Track skills listed telemetry
	if s.telemetry != nil {
		s.telemetry.TrackSkillsListed(len(skills), "mcp")
	}

	s.trackToolCall("skulto_list_skills", start, true)
	return mcp.NewToolResultText(string(data)), nil
}

// handleBrowseTags handles the skulto_browse_tags tool.
func (s *Server) handleBrowseTags(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	category := ""
	if c, ok := req.Params.Arguments["category"].(string); ok {
		category = c
	}

	tags, err := s.db.ListTags(category)
	if err != nil {
		s.trackToolCall("skulto_browse_tags", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to list tags: %v", err)), nil
	}

	results := make([]TagResponse, 0, len(tags))
	for _, tag := range tags {
		results = append(results, TagResponse{
			ID:       tag.ID,
			Name:     tag.Name,
			Slug:     tag.Slug,
			Category: tag.Category,
			Color:    tag.Color,
			Count:    tag.Count,
		})
	}

	data, err := json.Marshal(results)
	if err != nil {
		s.trackToolCall("skulto_browse_tags", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	s.trackToolCall("skulto_browse_tags", start, true)
	return mcp.NewToolResultText(string(data)), nil
}

// handleGetStats handles the skulto_get_stats tool.
func (s *Server) handleGetStats(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	stats, err := s.db.GetStats()
	if err != nil {
		s.trackToolCall("skulto_get_stats", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get stats: %v", err)), nil
	}

	installed, err := s.db.CountInstalled()
	if err != nil {
		installed = 0 // Non-fatal
	}

	resp := StatsResponse{
		TotalSkills:  stats.TotalSkills,
		TotalTags:    stats.TotalTags,
		TotalSources: stats.TotalSources,
		Installed:    installed,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		s.trackToolCall("skulto_get_stats", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal stats: %v", err)), nil
	}

	// Track stats viewed telemetry
	if s.telemetry != nil {
		s.telemetry.TrackStatsViewed()
	}

	s.trackToolCall("skulto_get_stats", start, true)
	return mcp.NewToolResultText(string(data)), nil
}

// handleGetRecent handles the skulto_get_recent tool.
func (s *Server) handleGetRecent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	limit := parseLimit(req.Params.Arguments, defaultRecentLimit, maxRecentLimit)

	skills, err := s.db.GetRecentSkills(limit)
	if err != nil {
		s.trackToolCall("skulto_get_recent", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get recent skills: %v", err)), nil
	}

	results := make([]SkillResponse, 0, len(skills))
	for i := range skills {
		results = append(results, toSkillResponseWithDB(&skills[i], false, s.db))
	}

	data, err := json.Marshal(results)
	if err != nil {
		s.trackToolCall("skulto_get_recent", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	// Track recent skills viewed telemetry
	if s.telemetry != nil {
		s.telemetry.TrackRecentSkillsViewed(len(skills))
	}

	s.trackToolCall("skulto_get_recent", start, true)
	return mcp.NewToolResultText(string(data)), nil
}

// handleInstall handles the skulto_install tool.
// Uses InstallService for unified installation across all platforms.
func (s *Server) handleInstall(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	slug, ok := req.Params.Arguments["slug"].(string)
	if !ok || slug == "" {
		s.trackToolCall("skulto_install", start, false)
		return mcp.NewToolResultError("slug parameter is required"), nil
	}

	// Parse optional platforms array
	var platforms []string
	if platformsArg, ok := req.Params.Arguments["platforms"].([]interface{}); ok {
		for _, p := range platformsArg {
			if ps, ok := p.(string); ok && ps != "" {
				platforms = append(platforms, ps)
			}
		}
	}

	// When no platforms specified, use detection to find available platforms
	if len(platforms) == 0 {
		detected := detect.DetectAll()
		var found []DetectedPlatformInfo
		for _, d := range detected {
			if d.Detected {
				info := d.Platform.Info()
				found = append(found, DetectedPlatformInfo{
					ID:   string(d.Platform),
					Name: info.Name,
				})
			}
		}

		switch len(found) {
		case 0:
			// No platforms detected — fall through with nil platforms (uses user config / defaults)
		case 1:
			// Single platform detected — use it directly
			platforms = []string{found[0].ID}
		default:
			// Multiple platforms detected — return them for user selection
			result := InstallResult{
				Success:           false,
				NeedsSelection:    true,
				Message:           "Multiple platforms detected. Please ask the user which platform(s) to install to, then re-call with the chosen platforms.",
				DetectedPlatforms: found,
			}
			data, _ := json.Marshal(result)
			// Track as success since this is expected behavior requiring user input
			s.trackToolCall("skulto_install", start, true)
			return mcp.NewToolResultText(string(data)), nil
		}
	}

	// Parse optional scope - default to project for MCP (local to current workspace)
	scopes := []installer.InstallScope{installer.ScopeProject}
	if scopeArg, ok := req.Params.Arguments["scope"].(string); ok && scopeArg != "" {
		scope := installer.InstallScope(scopeArg)
		if scope == installer.ScopeGlobal || scope == installer.ScopeProject {
			scopes = []installer.InstallScope{scope}
		}
	}

	// Build install options
	opts := installer.InstallOptions{
		Platforms: platforms,
		Scopes:    scopes,
		Confirm:   true,
	}

	// Use InstallService for unified behavior (telemetry tracked via InstallService)
	result, err := s.installService.Install(ctx, slug, opts)
	if err != nil {
		s.trackToolCall("skulto_install", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to install: %v", err)), nil
	}

	// Build paths from locations
	paths := make([]string, 0, len(result.Locations))
	for _, loc := range result.Locations {
		paths = append(paths, loc.GetSkillPath(result.Skill.Slug))
	}

	installResult := InstallResult{
		Success: true,
		Message: fmt.Sprintf("Skill '%s' installed to %d platform(s). Restart your agent for the skill to take effect.", result.Skill.Title, len(paths)),
		Paths:   paths,
	}

	data, _ := json.Marshal(installResult)
	s.trackToolCall("skulto_install", start, true)
	return mcp.NewToolResultText(string(data)), nil
}

// handleUninstall handles the skulto_uninstall tool.
// Uses InstallService for unified uninstallation across all platforms.
func (s *Server) handleUninstall(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	slug, ok := req.Params.Arguments["slug"].(string)
	if !ok || slug == "" {
		s.trackToolCall("skulto_uninstall", start, false)
		return mcp.NewToolResultError("slug parameter is required"), nil
	}

	// Parse optional platforms array
	var platforms []string
	if platformsArg, ok := req.Params.Arguments["platforms"].([]interface{}); ok {
		for _, p := range platformsArg {
			if ps, ok := p.(string); ok && ps != "" {
				platforms = append(platforms, ps)
			}
		}
	}

	// Parse optional scope
	scopeArg, _ := req.Params.Arguments["scope"].(string)

	// Get current install locations
	locations, err := s.installService.GetInstallLocations(ctx, slug)
	if err != nil {
		s.trackToolCall("skulto_uninstall", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get install locations: %v", err)), nil
	}

	if len(locations) == 0 {
		s.trackToolCall("skulto_uninstall", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("skill '%s' is not installed anywhere", slug)), nil
	}

	// Filter locations if platforms or scope specified
	var toUninstall []installer.InstallLocation
	if len(platforms) > 0 || (scopeArg != "" && scopeArg != "all") {
		platformSet := make(map[string]bool)
		for _, p := range platforms {
			platformSet[p] = true
		}

		for _, loc := range locations {
			// Filter by platform if specified
			if len(platforms) > 0 && !platformSet[string(loc.Platform)] {
				continue
			}
			// Filter by scope if specified (and not "all")
			if scopeArg != "" && scopeArg != "all" && string(loc.Scope) != scopeArg {
				continue
			}
			toUninstall = append(toUninstall, loc)
		}
	} else {
		// No filters - uninstall from all
		toUninstall = locations
	}

	if len(toUninstall) == 0 {
		s.trackToolCall("skulto_uninstall", start, false)
		return mcp.NewToolResultError("no matching installation locations found"), nil
	}

	// Perform uninstallation (telemetry tracked via InstallService)
	if err := s.installService.Uninstall(ctx, slug, toUninstall); err != nil {
		s.trackToolCall("skulto_uninstall", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to uninstall: %v", err)), nil
	}

	result := InstallResult{
		Success: true,
		Message: fmt.Sprintf("Skill '%s' uninstalled from %d location(s)", slug, len(toUninstall)),
	}

	data, _ := json.Marshal(result)
	s.trackToolCall("skulto_uninstall", start, true)
	return mcp.NewToolResultText(string(data)), nil
}

// handleFavorite handles the skulto_favorite tool.
func (s *Server) handleFavorite(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	slug, ok := req.Params.Arguments["slug"].(string)
	if !ok || slug == "" {
		s.trackToolCall("skulto_favorite", start, false)
		return mcp.NewToolResultError("slug parameter is required"), nil
	}

	action, ok := req.Params.Arguments["action"].(string)
	if !ok || action == "" {
		s.trackToolCall("skulto_favorite", start, false)
		return mcp.NewToolResultError("action parameter is required"), nil
	}

	if action != "add" && action != "remove" {
		s.trackToolCall("skulto_favorite", start, false)
		return mcp.NewToolResultError("action must be 'add' or 'remove'"), nil
	}

	// Verify skill exists in database
	skill, err := s.db.GetSkillBySlug(slug)
	if err != nil {
		s.trackToolCall("skulto_favorite", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get skill: %v", err)), nil
	}
	if skill == nil {
		s.trackToolCall("skulto_favorite", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("skill not found: %s", slug)), nil
	}

	// Check if favorites store is available
	if s.favorites == nil {
		s.trackToolCall("skulto_favorite", start, false)
		return mcp.NewToolResultError("favorites store not initialized"), nil
	}

	var opErr error
	var message string

	if action == "add" {
		opErr = s.favorites.Add(slug)
		message = fmt.Sprintf("Skill '%s' added to favorites", skill.Title)
	} else {
		opErr = s.favorites.Remove(slug)
		message = fmt.Sprintf("Skill '%s' removed from favorites", skill.Title)
	}

	if opErr != nil {
		s.trackToolCall("skulto_favorite", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to %s favorite: %v", action, opErr)), nil
	}

	// Track favorite telemetry
	if s.telemetry != nil {
		if action == "add" {
			s.telemetry.TrackFavoriteAdded(slug)
		} else {
			s.telemetry.TrackFavoriteRemoved(slug)
		}
	}

	result := InstallResult{
		Success: true,
		Message: message,
	}

	data, _ := json.Marshal(result)
	s.trackToolCall("skulto_favorite", start, true)
	return mcp.NewToolResultText(string(data)), nil
}

// handleGetFavorites handles the skulto_get_favorites tool.
func (s *Server) handleGetFavorites(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	limit := parseLimit(req.Params.Arguments, defaultFavoritesLimit, maxFavoritesLimit)

	// Check if favorites store is available
	if s.favorites == nil {
		s.trackToolCall("skulto_get_favorites", start, false)
		return mcp.NewToolResultError("favorites store not initialized"), nil
	}

	// Get favorites from file-based store
	favs := s.favorites.List()

	// Apply limit
	if len(favs) > limit {
		favs = favs[:limit]
	}

	// Look up skill details from database for each favorite
	results := make([]SkillResponse, 0, len(favs))
	for _, fav := range favs {
		skill, err := s.db.GetSkillBySlug(fav.Slug)
		if err != nil || skill == nil {
			// Skill might have been deleted from DB, include minimal info
			results = append(results, SkillResponse{
				Slug:        fav.Slug,
				Title:       fav.Slug, // Use slug as title if skill not found
				Description: "(Skill no longer in database)",
			})
			continue
		}
		results = append(results, toSkillResponseWithDB(skill, false, s.db))
	}

	data, err := json.Marshal(results)
	if err != nil {
		s.trackToolCall("skulto_get_favorites", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	// Track favorites listed telemetry
	if s.telemetry != nil {
		s.telemetry.TrackFavoritesListed(len(results))
	}

	s.trackToolCall("skulto_get_favorites", start, true)
	return mcp.NewToolResultText(string(data)), nil
}

// handleCheck handles the skulto_check tool.
func (s *Server) handleCheck(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	// Get all installed skills with their locations
	summaries, err := s.installService.GetInstalledSkillsSummary(ctx)
	if err != nil {
		s.trackToolCall("skulto_check", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get installed skills: %v", err)), nil
	}

	// Convert to response format
	results := make([]CheckSkillResponse, 0, len(summaries))
	for _, summary := range summaries {
		skillResp := CheckSkillResponse{
			Slug:      summary.Slug,
			Title:     summary.Title,
			Locations: make([]CheckLocationResponse, 0),
		}

		// Iterate through platforms and their scopes
		for platform, scopes := range summary.Locations {
			for _, scope := range scopes {
				skillResp.Locations = append(skillResp.Locations, CheckLocationResponse{
					Platform: string(platform),
					Scope:    string(scope),
				})
			}
		}

		results = append(results, skillResp)
	}

	data, err := json.Marshal(results)
	if err != nil {
		s.trackToolCall("skulto_check", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	// Track installed skills checked telemetry
	if s.telemetry != nil {
		s.telemetry.TrackInstalledSkillsChecked(len(summaries))
	}

	s.trackToolCall("skulto_check", start, true)
	return mcp.NewToolResultText(string(data)), nil
}

// handleAdd handles the skulto_add tool.
func (s *Server) handleAdd(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	start := time.Now()

	url, ok := req.Params.Arguments["url"].(string)
	if !ok || url == "" {
		s.trackToolCall("skulto_add", start, false)
		return mcp.NewToolResultError("url parameter is required"), nil
	}

	// Parse and validate the repository URL
	source, err := scraper.ParseRepositoryURL(url)
	if err != nil {
		s.trackToolCall("skulto_add", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("invalid repository URL: %v", err)), nil
	}

	// Check if source already exists
	existing, err := s.db.GetSource(source.ID)
	if err != nil {
		s.trackToolCall("skulto_add", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("failed to check existing source: %v", err)), nil
	}
	if existing != nil {
		s.trackToolCall("skulto_add", start, false)
		return mcp.NewToolResultError(fmt.Sprintf("repository %s already exists", source.ID)), nil
	}

	// Add source to database
	if err := s.db.UpsertSource(source); err != nil {
		s.trackToolCall("skulto_add", start, false)
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
		s.trackToolCall("skulto_add", start, false)
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

	// Track repo added telemetry
	if s.telemetry != nil {
		s.telemetry.TrackRepoAdded(source.ID, result.SkillsNew)
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
	s.trackToolCall("skulto_add", start, true)
	return mcp.NewToolResultText(string(data)), nil
}
