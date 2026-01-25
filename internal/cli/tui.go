package cli

import (
	"fmt"
	"path/filepath"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/log"
	"github.com/asteroid-belt/skulto/internal/search"
	"github.com/asteroid-belt/skulto/internal/telemetry"
	"github.com/asteroid-belt/skulto/internal/tui"
	"github.com/asteroid-belt/skulto/internal/vector"
	"github.com/asteroid-belt/skulto/pkg/version"
	"github.com/spf13/cobra"
)

// runTUI executes the TUI when no subcommand is specified.
func runTUI(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Initialize logger
	if err := log.Init(cfg.BaseDir); err != nil {
		return fmt.Errorf("initialize logger: %w", err)
	}
	defer func() {
		_ = log.Close()
	}()

	// Print banner
	printBanner()

	// Print config info
	paths := config.GetPaths(cfg)
	log.Printf("\n\U0001F4C1 Base directory: %s\n", cfg.BaseDir)
	log.Printf("\U0001F4C1 Database: %s\n", paths.Database)
	log.Printf("\U0001F4C1 Log file: %s/skulto.log\n", cfg.BaseDir)

	if cfg.GitHub.Token != "" {
		log.Println("\U0001F511 GitHub token: configured")
	} else {
		log.Println("\U0001F511 GitHub token: not set (set GITHUB_TOKEN for higher rate limits)")
	}

	// Initialize database
	log.Println("\n\U0001F4CA Initializing database...")
	database, err := db.New(db.DefaultConfig(paths.Database))
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Errorf("\U000026A0\U0000FE0F  Failed to close database: %v\n", err)
		}
	}()

	// Get stats
	stats, err := database.GetStats()
	if err != nil {
		return fmt.Errorf("get stats: %w", err)
	}

	log.Printf("   Skills indexed: %d\n", stats.TotalSkills)
	log.Printf("   Tags: %d\n", stats.TotalTags)
	log.Printf("   Sources: %d\n", stats.TotalSources)
	if stats.CacheSizeBytes > 0 {
		log.Printf("   Database size: %.2f KB\n", float64(stats.CacheSizeBytes)/1024)
	}

	// Initialize vector store and background indexer if API key is set
	var bgIndexer *search.BackgroundIndexer
	if cfg.Embedding.APIKey != "" {
		log.Println("\n\U0001F50D Semantic search: enabled (OPENAI_API_KEY found)")

		vectorDir := cfg.Embedding.DataDir
		if vectorDir == "" {
			vectorDir = filepath.Join(cfg.BaseDir, "vectors")
		}

		store, err := vector.New(vector.Config{
			DataDir:   vectorDir,
			OpenAIKey: cfg.Embedding.APIKey,
			Model:     cfg.Embedding.Model,
		})
		if err != nil {
			log.Printf("   \U000026A0\U0000FE0F  Could not initialize vector store: %v\n", err)
			log.Println("   Falling back to keyword-only search")
		} else {
			bgIndexer = search.NewBackgroundIndexer(
				database,
				store,
				search.DefaultIndexerConfig(),
			)

			pending, _ := bgIndexer.GetPendingCount()
			if pending > 0 {
				log.Printf("   Found %d skills to index for semantic search\n", pending)
			} else {
				log.Println("   All skills already indexed")
			}
		}
	} else {
		log.Println("\n\U0001F50D Semantic search: disabled (set OPENAI_API_KEY to enable)")
	}

	// Telemetry status
	if telemetry.IsEnabled() {
		log.Println("\n\U0001F4CA Telemetry: ON (set SKULTO_TELEMETRY_TRACKING_ENABLED=false to disable)")
		log.Printf("   Anon ID: %s\n", database.GetOrCreateTrackingID())
	} else {
		log.Println("\n\U0001F4CA Telemetry: OFF")
	}

	log.Println("\n\U0001F480 Launching Skulto TUI...")
	log.Println("   Press / to search, \u2193 to browse, q to quit")

	return tui.RunWithIndexer(database, cfg, bgIndexer, telemetryClient)
}

func printBanner() {
	banner := `
   ╔═══════════════════════════════════════════════════════╗
   ║    ███████╗██╗  ██╗██╗   ██╗██╗  ████████╗ ██████╗    ║
   ║    ██╔════╝██║ ██╔╝██║   ██║██║  ╚══██╔══╝██╔═══██╗   ║
   ║    ███████╗█████╔╝ ██║   ██║██║     ██║   ██║   ██║   ║
   ║    ╚════██║██╔═██╗ ██║   ██║██║     ██║   ██║   ██║   ║
   ║    ███████║██║  ██╗╚██████╔╝███████╗██║   ╚██████╔╝   ║
   ║    ╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝    ╚═════╝    ║
   ╠═══════════════════════════════════════════════════════╣
   ║          CROSS-PLATFORM AI SKILLS MANAGEMENT          ║
   ╚═══════════════════════════════════════════════════════╝
`
	fmt.Print(banner)
	fmt.Printf("   Version: %s\n", version.Short())
}
