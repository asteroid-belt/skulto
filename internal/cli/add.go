package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/scraper"
	"github.com/spf13/cobra"
)

var addNoSync bool

var addCmd = &cobra.Command{
	Use:     "add <repository_url>",
	Aliases: []string{"a"},
	Short:   "Add a skill repository (alias: a)",
	Long: `Add a skill repository to Skulto and sync its skills.

Supports multiple URL formats:
  - owner/repo                    (short format)
  - https://github.com/owner/repo
  - https://github.com/owner/repo.git
  - git@github.com:owner/repo.git

Examples:
  # Add using short format
  skulto add asteroid-belt/skills

  # Add using full URL
  skulto add https://github.com/asteroid-belt/skills

  # Add without syncing (manual sync later)
  skulto add asteroid-belt/skills --no-sync`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().BoolVar(&addNoSync, "no-sync", false, "Don't clone and sync skills immediately")
}

func runAdd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	repoURL := args[0]

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

	fmt.Printf("\U0001F4E6 Parsing repository URL: %s\n", repoURL)
	source, err := scraper.ParseRepositoryURL(repoURL)
	if err != nil {
		return fmt.Errorf("invalid repository URL: %w", err)
	}

	existing, err := database.GetSource(source.ID)
	if err != nil {
		return fmt.Errorf("failed to check existing source: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("repository %s already exists in database", source.ID)
	}

	fmt.Println("\n\U0001F4DD Adding repository to database...")
	if err := database.UpsertSource(source); err != nil {
		return fmt.Errorf("failed to add source: %w", err)
	}
	fmt.Println("   \u2713 Repository added")

	if !addNoSync {
		fmt.Println("\n\U0001F504 Cloning and syncing repository...")

		scraperCfg := scraper.ScraperConfig{
			Token:        cfg.GitHub.Token,
			DataDir:      cfg.BaseDir,
			RepoCacheTTL: cfg.GitHub.RepoCacheTTL,
			UseGitClone:  cfg.GitHub.UseGitClone,
		}
		s := scraper.NewScraperWithConfig(scraperCfg, database)

		syncCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		result, err := s.ScrapeRepository(syncCtx, source.Owner, source.Repo)
		if err != nil {
			return fmt.Errorf("failed to sync %s: %w", source.ID, err)
		}

		fmt.Printf("   \u2713 Skills found: %d\n", result.SkillsNew)
	} else {
		fmt.Println("\n\u23ED\uFE0F  Skipping sync (--no-sync specified)")
		fmt.Println("   Run 'skulto' and press 'p' to sync later")
	}

	fmt.Printf("\n\U0001F480 Repository %s added successfully!\n", source.ID)

	// Track telemetry event
	skillCount := 0
	if !addNoSync {
		skills, _ := database.GetSkillsBySourceID(source.ID)
		skillCount = len(skills)
	}
	telemetryClient.TrackRepoAdded(source.ID, skillCount)

	return nil
}
