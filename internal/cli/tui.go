package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/discovery"
	"github.com/asteroid-belt/skulto/internal/installer"
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
	log.Printf("\n📁 Base directory: %s\n", cfg.BaseDir)
	log.Printf("📁 Database: %s\n", paths.Database)
	log.Printf("🗂️  Log file: %s/skulto.log\n", cfg.BaseDir)

	if cfg.GitHub.Token != "" {
		log.Println("\nGitHub token: configured")
	} else {
		log.Println("\nGitHub token: not set (set GITHUB_TOKEN for higher rate limits)")
	}

	// Initialize database
	log.Println("\nInitializing database...")
	database, err := db.New(db.DefaultConfig(paths.Database))
	if err != nil {
		return fmt.Errorf("\ninitialize database: %w", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Errorf("\nFailed to close database: %v\n", err)
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
		log.Println("\nSemantic search: enabled (OPENAI_API_KEY found)")

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
			log.Printf("   Could not initialize vector store: %v\n", err)
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
		log.Println("\nSemantic search: disabled (set OPENAI_API_KEY to enable)")
	}

	// Telemetry status
	if telemetry.IsEnabled() {
		log.Println("\n📞 Telemetry: ON (set SKULTO_TELEMETRY_TRACKING_ENABLED=false to disable)")
		log.Printf("   Anon ID: %s\n", database.GetOrCreateTrackingID())
	} else {
		log.Println("\nTelemetry: OFF")
	}

	// Enforce installer path policy migrations before discovery scans.
	installService := installer.NewInstallService(database, cfg, telemetryClient)
	cwd, _ := os.Getwd()
	if err := installService.EnsurePathPolicy(context.Background(), cwd); err != nil {
		log.Printf("\nPath policy migration skipped: %v\n", err)
	}

	// Scan for unmanaged skills before showing notification
	scanDiscoveredSkills(database, cfg)

	// Show startup notification for unnotified discovered skills
	showStartupNotification(database, os.Stdout)

	log.Println("\n🚀 Launching Skulto TUI...")
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

// scanDiscoveredSkills scans platform directories for unmanaged skills.
// This runs synchronously before the TUI launches to populate discovered_skills.
func scanDiscoveredSkills(database *db.DB, cfg *config.Config) {
	if database == nil || cfg == nil {
		return
	}

	scanner := discovery.NewScannerService()
	service := installer.NewInstallService(database, cfg, telemetryClient)

	// Detect available platforms
	ctx := context.Background()
	platforms, err := service.DetectPlatforms(ctx)
	if err != nil {
		return
	}

	for _, platform := range platforms {
		if !platform.Detected {
			continue
		}

		info := installer.PlatformFromString(platform.ID).Info()

		// Scan global directories
		if info.GlobalDir != "" {
			globalPath := expandPath(info.GlobalDir)
			discovered, err := scanner.ScanDirectory(globalPath, platform.ID, string(installer.ScopeGlobal))
			if err == nil {
				for _, d := range discovered {
					_ = database.UpsertDiscoveredSkill(&d)
				}
			}
		}

		// Scan project directories
		if info.SkillsPath != "" {
			cwd, err := os.Getwd()
			if err == nil {
				projectPath := filepath.Join(cwd, info.SkillsPath)
				discovered, err := scanner.ScanDirectory(projectPath, platform.ID, string(installer.ScopeProject))
				if err == nil {
					for _, d := range discovered {
						_ = database.UpsertDiscoveredSkill(&d)
					}
				}
			}
		}
	}
}
