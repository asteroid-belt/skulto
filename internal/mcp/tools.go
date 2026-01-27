package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// Tool definitions for Skulto MCP server.

// searchTool returns the skulto_search tool definition.
func searchTool() mcp.Tool {
	return mcp.NewTool("skulto_search",
		mcp.WithDescription("Search skills using full-text search with BM25 ranking. Returns skills matching the query ordered by relevance."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query - supports partial matching and multiple terms"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results to return (default: 20, max: 100)"),
		),
	)
}

// getSkillTool returns the skulto_get_skill tool definition.
func getSkillTool() mcp.Tool {
	return mcp.NewTool("skulto_get_skill",
		mcp.WithDescription("Get detailed information about a skill including full content, tags, source repository, and auxiliary files."),
		mcp.WithString("slug",
			mcp.Required(),
			mcp.Description("The skill's unique slug identifier"),
		),
	)
}

// listSkillsTool returns the skulto_list_skills tool definition.
func listSkillsTool() mcp.Tool {
	return mcp.NewTool("skulto_list_skills",
		mcp.WithDescription("List all skills with pagination. Returns skills ordered by most recently updated."),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results to return (default: 20, max: 100)"),
		),
		mcp.WithNumber("offset",
			mcp.Description("Number of results to skip for pagination (default: 0)"),
		),
	)
}

// browseTagsTool returns the skulto_browse_tags tool definition.
func browseTagsTool() mcp.Tool {
	return mcp.NewTool("skulto_browse_tags",
		mcp.WithDescription("List available tags for filtering skills. Tags are organized by category: language, framework, tool, concept, domain."),
		mcp.WithString("category",
			mcp.Description("Filter by category: language, framework, tool, concept, domain (optional - returns all if not specified)"),
		),
	)
}

// getStatsTool returns the skulto_get_stats tool definition.
func getStatsTool() mcp.Tool {
	return mcp.NewTool("skulto_get_stats",
		mcp.WithDescription("Get database statistics including total skills, tags, sources, and cache size."),
	)
}

// getRecentTool returns the skulto_get_recent tool definition.
func getRecentTool() mcp.Tool {
	return mcp.NewTool("skulto_get_recent",
		mcp.WithDescription("Get recently viewed skills ordered by view time."),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results to return (default: 10, max: 50)"),
		),
	)
}

// installTool returns the skulto_install tool definition.
func installTool() mcp.Tool {
	return mcp.NewTool("skulto_install",
		mcp.WithDescription("Install a skill to Claude Code. Creates a symlink in the global Claude skills directory."),
		mcp.WithString("slug",
			mcp.Required(),
			mcp.Description("The skill's unique slug identifier"),
		),
		mcp.WithArray("platforms",
			mcp.Description("Platforms to install to. Options: claude, cursor, windsurf, copilot, codex, opencode. Default: user's configured platforms."),
		),
		mcp.WithString("scope",
			mcp.Description("Installation scope: 'global' (user-wide) or 'project' (current directory). Default: global."),
		),
	)
}

// uninstallTool returns the skulto_uninstall tool definition.
func uninstallTool() mcp.Tool {
	return mcp.NewTool("skulto_uninstall",
		mcp.WithDescription("Uninstall a skill from Claude Code. Removes the symlink from the skills directory."),
		mcp.WithString("slug",
			mcp.Required(),
			mcp.Description("The skill's unique slug identifier"),
		),
		mcp.WithArray("platforms",
			mcp.Description("Platforms to uninstall from. Options: claude, cursor, windsurf, copilot, codex, opencode. Default: all installed locations."),
		),
		mcp.WithString("scope",
			mcp.Description("Scope to uninstall from: 'global', 'project', or 'all'. Default: all."),
		),
	)
}

// favoriteTool returns the skulto_favorite tool definition.
func favoriteTool() mcp.Tool {
	return mcp.NewTool("skulto_favorite",
		mcp.WithDescription("Add or remove a skill from your favorites."),
		mcp.WithString("slug",
			mcp.Required(),
			mcp.Description("The skill's unique slug identifier"),
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Action to perform: 'add' or 'remove'"),
		),
	)
}

// getFavoritesTool returns the skulto_get_favorites tool definition.
func getFavoritesTool() mcp.Tool {
	return mcp.NewTool("skulto_get_favorites",
		mcp.WithDescription("Get your favorite skills."),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results to return (default: 50, max: 100)"),
		),
	)
}
