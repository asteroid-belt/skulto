// Package main provides the skulto-mcp server for Claude Code integration.
//
// skulto-mcp exposes the Skulto skills database via the Model Context Protocol,
// enabling Claude Code to search, browse, and install skills from the marketplace.
//
// Usage:
//
//	skulto-mcp [flags]
//
// The server communicates via JSON-RPC 2.0 over stdio (stdin/stdout).
// Configure in Claude Code via ~/.claude.json or .mcp.json.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/mcp"
	"github.com/asteroid-belt/skulto/pkg/version"
)

func main() {
	// Handle --version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("skulto-mcp %s\n", version.Version)
		os.Exit(0)
	}

	// Handle --help flag
	if len(os.Args) > 1 && (os.Args[1] == "--help" || os.Args[1] == "-h") {
		printHelp()
		os.Exit(0)
	}

	// Setup context with cancellation on interrupt
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Load config and initialize database
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	paths := config.GetPaths(cfg)
	database, err := db.New(db.DefaultConfig(paths.Database))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = database.Close()
	}()

	// Create and run MCP server
	server := mcp.NewServer(database, cfg)
	if err := server.Serve(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	help := `skulto-mcp - MCP server for Skulto skills marketplace

USAGE:
    skulto-mcp [FLAGS]

FLAGS:
    -h, --help       Print this help message
    -v, --version    Print version information

DESCRIPTION:
    skulto-mcp is a Model Context Protocol (MCP) server that exposes the
    Skulto skills database to Claude Code and other MCP-compatible clients.

    The server communicates via JSON-RPC 2.0 over stdio (stdin/stdout).

CONFIGURATION:
    Add to ~/.claude.json for user-level access:

    {
      "mcpServers": {
        "skulto": {
          "type": "stdio",
          "command": "skulto-mcp"
        }
      }
    }

    Or add to .mcp.json in your project root for project-level access.

TOOLS PROVIDED:
    skulto_search        Search skills with full-text search
    skulto_get_skill     Get detailed skill information
    skulto_list_skills   List all skills with pagination
    skulto_browse_tags   List tags by category
    skulto_get_stats     Get database statistics
    skulto_get_recent    Get recently viewed skills
    skulto_install       Install a skill to Claude
    skulto_uninstall     Uninstall a skill from Claude
    skulto_bookmark      Bookmark/unbookmark a skill
    skulto_get_bookmarks Get bookmarked skills

RESOURCES PROVIDED:
    skulto://skill/{slug}           Skill markdown content
    skulto://skill/{slug}/metadata  Skill metadata as JSON

MORE INFO:
    https://github.com/asteroid-belt-llc/skulto
`
	fmt.Print(help)
}
