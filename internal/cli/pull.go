package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/scraper"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:     "pull",
	Aliases: []string{"p"},
	Short:   "Pull and sync all skill repositories (alias: p)",
	Long: `Pull and sync all skill repositories, then reconcile installed skills.

This command:
  1. Clones/updates all registered skill repositories
  2. Scans AI tool directories to detect installed skills
  3. Reconciles database state with filesystem reality

Examples:
  # Pull all repositories and sync install state
  skulto pull`,
	Args: cobra.NoArgs,
	RunE: runPull,
}

func runPull(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	paths := config.GetPaths(cfg)
	database, err := db.New(db.DefaultConfig(paths.Database))
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer func() {
		_ = database.Close()
	}()

	// Get all sources
	sources, err := database.ListSources()
	if err != nil {
		return fmt.Errorf("failed to get sources: %w", err)
	}

	if len(sources) == 0 {
		fmt.Println("📦 No repositories configured. Use 'skulto add <repo>' to add one.")
		return nil
	}

	fmt.Println("🔄 Pulling skill repositories...")
	fmt.Println()

	// Create scraper
	scraperCfg := scraper.ScraperConfig{
		Token:        cfg.GitHub.Token,
		DataDir:      cfg.BaseDir,
		RepoCacheTTL: cfg.GitHub.RepoCacheTTL,
		UseGitClone:  cfg.GitHub.UseGitClone,
	}
	s := scraper.NewScraperWithConfig(scraperCfg, database)

	// Initialize progress bar
	progress := NewProgressBar(len(sources), 15)
	reposErrored := 0

	// Sync each repository
	for i, source := range sources {
		repoName := fmt.Sprintf("%s/%s", source.Owner, source.Repo)
		progress.Update(i+1, repoName)
		ClearLine()
		fmt.Print("   " + progress.Render())

		syncCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		result, err := s.ScrapeRepository(syncCtx, source.Owner, source.Repo)
		cancel()

		if err != nil {
			ClearLine()
			fmt.Printf("   ❌ %s: %v\n", repoName, err)
			reposErrored++
			continue
		}

		// Track telemetry per repo
		telemetryClient.TrackRepoSynced(source.ID, result.SkillsNew, 0, result.SkillsUpdated)
	}

	// Final progress
	ClearLine()
	if reposErrored > 0 {
		fmt.Printf("   ✓ Pull complete (%d errors)\n", reposErrored)
	} else {
		fmt.Println("   ✓ Pull complete")
	}

	// Sync install state
	fmt.Println("\n🔍 Scanning AI tool directories for installed skills...")

	inst := installer.New(database, cfg)
	if err := inst.SyncInstallState(ctx); err != nil {
		fmt.Printf("⚠️  Install sync warning: %v\n", err)
	} else {
		fmt.Println("   ✓ Install state reconciled")
	}

	fmt.Println("\n⚡ Pull complete!")

	return nil
}
