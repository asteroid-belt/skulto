// Skulto - Cross-Platform AI Skills Management
//
// A punk-rock themed, offline-first CLI tool for managing AI coding skills
// across multiple platforms (Claude, Cursor, Copilot, Codex, OpenCode, Windsurf).
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/asteroid-belt/skulto/internal/cli"
	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/telemetry"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Load config and open database for persistent tracking ID
	cfg, err := config.Load()
	if err != nil {
		os.Exit(1)
	}

	paths := config.GetPaths(cfg)
	database, err := db.New(db.DefaultConfig(paths.Database))
	if err != nil {
		os.Exit(1)
	}
	defer func() {
		_ = database.Close()
	}()

	// Use persistent tracking ID from database
	telemetryClient := telemetry.New(database)
	defer telemetryClient.Close()

	if err := cli.Execute(ctx, telemetryClient); err != nil {
		os.Exit(1)
	}
}
