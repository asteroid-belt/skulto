package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/mark3labs/mcp-go/mcp"
)

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
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Paths   []string `json:"paths,omitempty"` // Symlink paths created
}

// toSkillResponse converts a models.Skill to SkillResponse.
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
		IsInstalled: skill.IsInstalled,
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

func (s *Server) handleSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, ok := req.Params.Arguments["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	limit := 20
	if l, ok := req.Params.Arguments["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 100 {
			limit = 100
		}
	}

	skills, err := s.db.SearchSkills(query, limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	results := make([]SkillResponse, 0, len(skills))
	for i := range skills {
		results = append(results, toSkillResponse(&skills[i], false))
	}

	data, err := json.Marshal(results)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleGetSkill handles the skulto_get_skill tool.
func (s *Server) handleGetSkill(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	slug, ok := req.Params.Arguments["slug"].(string)
	if !ok || slug == "" {
		return mcp.NewToolResultError("slug parameter is required"), nil
	}

	skill, err := s.db.GetSkillBySlug(slug)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get skill: %v", err)), nil
	}
	if skill == nil {
		return mcp.NewToolResultError(fmt.Sprintf("skill not found: %s", slug)), nil
	}

	now := time.Now()
	skill.ViewedAt = &now
	_ = s.db.UpdateSkill(skill)

	resp := toSkillResponse(skill, true)

	data, err := json.Marshal(resp)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal skill: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleListSkills handles the skulto_list_skills tool.
func (s *Server) handleListSkills(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	limit := 20
	if l, ok := req.Params.Arguments["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 100 {
			limit = 100
		}
	}

	offset := 0
	if o, ok := req.Params.Arguments["offset"].(float64); ok && o >= 0 {
		offset = int(o)
	}

	skills, err := s.db.ListSkills(limit, offset)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list skills: %v", err)), nil
	}

	results := make([]SkillResponse, 0, len(skills))
	for i := range skills {
		results = append(results, toSkillResponse(&skills[i], false))
	}

	data, err := json.Marshal(results)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleBrowseTags handles the skulto_browse_tags tool.
func (s *Server) handleBrowseTags(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	category := ""
	if c, ok := req.Params.Arguments["category"].(string); ok {
		category = c
	}

	tags, err := s.db.ListTags(category)
	if err != nil {
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
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleGetStats handles the skulto_get_stats tool.
func (s *Server) handleGetStats(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stats, err := s.db.GetStats()
	if err != nil {
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
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal stats: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleGetRecent handles the skulto_get_recent tool.
func (s *Server) handleGetRecent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	limit := 10
	if l, ok := req.Params.Arguments["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 50 {
			limit = 50
		}
	}

	skills, err := s.db.GetRecentSkills(limit)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get recent skills: %v", err)), nil
	}

	results := make([]SkillResponse, 0, len(skills))
	for i := range skills {
		results = append(results, toSkillResponse(&skills[i], false))
	}

	data, err := json.Marshal(results)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}

// handleInstall handles the skulto_install tool.
// Uses internal/installer to create actual symlinks to AI tool directories.
func (s *Server) handleInstall(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	slug, ok := req.Params.Arguments["slug"].(string)
	if !ok || slug == "" {
		return mcp.NewToolResultError("slug parameter is required"), nil
	}

	skill, err := s.db.GetSkillBySlug(slug)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get skill: %v", err)), nil
	}
	if skill == nil {
		return mcp.NewToolResultError(fmt.Sprintf("skill not found: %s", slug)), nil
	}

	fullSkill, err := s.db.GetSkill(skill.ID)
	if err != nil || fullSkill == nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to load skill details: %v", err)), nil
	}

	source := fullSkill.Source
	if source == nil && fullSkill.SourceID != nil {
		source, _ = s.db.GetSource(*fullSkill.SourceID)
	}

	if source == nil {
		return mcp.NewToolResultError("skill has no source repository - cannot install"), nil
	}

	userState, err := s.db.GetUserState()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get user settings: %v", err)), nil
	}

	platforms := userState.GetAITools()
	if len(platforms) == 0 {
		return mcp.NewToolResultError("no AI platforms configured - run 'skulto' TUI to select platforms"), nil
	}

	locations := make([]installer.InstallLocation, 0, len(platforms))
	for _, platformName := range platforms {
		platform := installer.PlatformFromString(platformName)
		if platform == "" {
			continue
		}
		loc, err := installer.NewInstallLocation(platform, installer.ScopeGlobal)
		if err != nil {
			continue
		}
		locations = append(locations, loc)
	}

	if len(locations) == 0 {
		return mcp.NewToolResultError("no valid installation locations found"), nil
	}

	if err := s.installer.InstallTo(ctx, fullSkill, source, locations); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to install: %v", err)), nil
	}

	installedLocs, _ := s.installer.GetInstallLocations(fullSkill.ID)
	paths := make([]string, 0, len(installedLocs))
	for _, loc := range installedLocs {
		paths = append(paths, loc.GetSkillPath(fullSkill.Slug))
	}

	result := InstallResult{
		Success: true,
		Message: fmt.Sprintf("Skill '%s' installed to %d platform(s)", fullSkill.Title, len(paths)),
		Paths:   paths,
	}

	data, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(data)), nil
}

// handleUninstall handles the skulto_uninstall tool.
// Uses internal/installer to remove symlinks from all AI tool directories.
func (s *Server) handleUninstall(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	slug, ok := req.Params.Arguments["slug"].(string)
	if !ok || slug == "" {
		return mcp.NewToolResultError("slug parameter is required"), nil
	}

	skill, err := s.db.GetSkillBySlug(slug)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get skill: %v", err)), nil
	}
	if skill == nil {
		return mcp.NewToolResultError(fmt.Sprintf("skill not found: %s", slug)), nil
	}

	if err := s.installer.UninstallAll(ctx, skill); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to uninstall: %v", err)), nil
	}

	result := InstallResult{
		Success: true,
		Message: fmt.Sprintf("Skill '%s' uninstalled from all platforms", skill.Title),
	}

	data, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(data)), nil
}

// handleBookmark handles the skulto_bookmark tool.
func (s *Server) handleBookmark(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	slug, ok := req.Params.Arguments["slug"].(string)
	if !ok || slug == "" {
		return mcp.NewToolResultError("slug parameter is required"), nil
	}

	action, ok := req.Params.Arguments["action"].(string)
	if !ok || action == "" {
		return mcp.NewToolResultError("action parameter is required"), nil
	}

	if action != "add" && action != "remove" {
		return mcp.NewToolResultError("action must be 'add' or 'remove'"), nil
	}

	skill, err := s.db.GetSkillBySlug(slug)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get skill: %v", err)), nil
	}
	if skill == nil {
		return mcp.NewToolResultError(fmt.Sprintf("skill not found: %s", slug)), nil
	}

	var opErr error
	var message string

	if action == "add" {
		opErr = s.db.AddInstalled(skill.ID)
		message = fmt.Sprintf("Skill '%s' bookmarked", skill.Title)
	} else {
		opErr = s.db.RemoveInstalled(skill.ID)
		message = fmt.Sprintf("Skill '%s' removed from bookmarks", skill.Title)
	}

	if opErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to %s bookmark: %v", action, opErr)), nil
	}

	result := InstallResult{
		Success: true,
		Message: message,
	}

	data, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(data)), nil
}

// handleGetBookmarks handles the skulto_get_bookmarks tool.
func (s *Server) handleGetBookmarks(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	limit := 50
	if l, ok := req.Params.Arguments["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 100 {
			limit = 100
		}
	}

	skills, err := s.db.GetInstalled()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get bookmarks: %v", err)), nil
	}

	// Apply limit
	if len(skills) > limit {
		skills = skills[:limit]
	}

	results := make([]SkillResponse, 0, len(skills))
	for i := range skills {
		results = append(results, toSkillResponse(&skills[i], false))
	}

	data, err := json.Marshal(results)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal results: %v", err)), nil
	}

	return mcp.NewToolResultText(string(data)), nil
}
