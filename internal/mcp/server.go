// Package mcp provides the Model Context Protocol server for Skulto.
//
// This package implements an MCP server that exposes the Skulto skills database
// to Claude Code and other MCP-compatible clients. It reuses the existing
// internal/db and internal/installer packages to ensure consistent behavior
// with the TUI and CLI.
package mcp

import (
	"context"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/favorites"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/telemetry"
	"github.com/asteroid-belt/skulto/pkg/version"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the MCP server with Skulto-specific functionality.
type Server struct {
	db             *db.DB
	cfg            *config.Config
	installer      *installer.Installer      // Reuse existing installer for symlink operations
	installService *installer.InstallService // Unified install service
	favorites      *favorites.Store          // Favorites store (persists across DB resets)
	server         *server.MCPServer
	telemetry      telemetry.Client
}

// NewServer creates a new MCP server instance.
func NewServer(database *db.DB, cfg *config.Config, favStore *favorites.Store, tc telemetry.Client) *Server {
	s := &Server{
		db:             database,
		cfg:            cfg,
		installer:      installer.New(database, cfg),                   // Same installer used by TUI
		installService: installer.NewInstallService(database, cfg, tc), // Unified service with telemetry
		favorites:      favStore,
		telemetry:      tc,
	}

	// Create MCP server with capabilities
	s.server = server.NewMCPServer(
		"skulto",
		version.Version,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false), // subscribe=false for now
	)

	// Register tools and resources
	s.registerTools()
	s.registerResources()

	return s
}

// Serve starts the MCP server over stdio.
func (s *Server) Serve(ctx context.Context) error {
	// Sync install state on startup - same as TUI does
	// This ensures DB reflects actual symlinks on disk
	_ = s.installer.SyncInstallState(ctx) // Best-effort, don't fail server start

	return server.ServeStdio(s.server)
}

// registerTools adds all Skulto tools to the MCP server.
func (s *Server) registerTools() {
	// Core tools (Phase 1A)
	s.server.AddTool(searchTool(), s.handleSearch)
	s.server.AddTool(getSkillTool(), s.handleGetSkill)
	s.server.AddTool(listSkillsTool(), s.handleListSkills)

	// Browse tools (Phase 1B)
	s.server.AddTool(browseTagsTool(), s.handleBrowseTags)
	s.server.AddTool(getStatsTool(), s.handleGetStats)
	s.server.AddTool(getRecentTool(), s.handleGetRecent)

	// User state tools (Phase 2)
	s.server.AddTool(installTool(), s.handleInstall)
	s.server.AddTool(uninstallTool(), s.handleUninstall)
	s.server.AddTool(favoriteTool(), s.handleFavorite)
	s.server.AddTool(getFavoritesTool(), s.handleGetFavorites)

	// Check tool (shows installed skills)
	s.server.AddTool(checkTool(), s.handleCheck)

	// Repository management
	s.server.AddTool(addTool(), s.handleAdd)
}

// registerResources adds all Skulto resources to the MCP server.
func (s *Server) registerResources() {
	// Register resource templates for skill content and metadata
	s.server.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"skulto://skill/{slug}",
			"Skill content",
			mcp.WithTemplateDescription("Full markdown content of a skill"),
			mcp.WithTemplateMIMEType("text/markdown"),
		),
		s.handleSkillContentResource,
	)

	s.server.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"skulto://skill/{slug}/metadata",
			"Skill metadata",
			mcp.WithTemplateDescription("JSON metadata for a skill including tags, source, and stats"),
			mcp.WithTemplateMIMEType("application/json"),
		),
		s.handleSkillMetadataResource,
	)
}
