package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/asteroid-belt/skulto/internal/cli/prompts"
	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/scraper"
	"github.com/spf13/cobra"
)

var (
	installPlatforms []string
	installScope     string
	installYes       bool
)

var installCmd = &cobra.Command{
	Use:   "install <slug|url>",
	Short: "Install a skill to AI tool directories",
	Long: `Install a skill by slug or from a repository URL.

The install command creates symlinks in your AI tool directories,
making skills available to Claude, Cursor, Windsurf, and other tools.

Interactive Mode (default):
  Shows a multi-select prompt for platforms and scopes.
  Detected platforms are pre-selected.

Non-Interactive Mode (-y):
  Installs to detected platforms with global scope.
  Use -p and -s flags to override defaults.

URL Mode:
  When given a URL or owner/repo format, auto-adds the repository
  and shows a skill picker to select which skills to install.

Examples:
  # Interactive install
  skulto install docker-expert

  # Non-interactive with defaults
  skulto install docker-expert -y

  # Install to specific platform
  skulto install docker-expert -p claude -y

  # Install to multiple platforms and project scope
  skulto install docker-expert -p claude -p cursor -s project -y

  # Install from repository URL
  skulto install https://github.com/owner/skills

  # Install from short format
  skulto install owner/skills`,
	Args: cobra.ExactArgs(1),
	RunE: runInstall,
}

func init() {
	installCmd.Flags().StringArrayVarP(&installPlatforms, "platform", "p", nil,
		"Platform to install to (repeatable: -p claude -p cursor)")
	installCmd.Flags().StringVarP(&installScope, "scope", "s", "",
		"Installation scope: global or project")
	installCmd.Flags().BoolVarP(&installYes, "yes", "y", false,
		"Skip interactive prompts, use defaults")
}

func runInstall(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	input := args[0]

	// Load config and database
	cfg, err := config.Load()
	if err != nil {
		return trackCLIError("install", fmt.Errorf("load config: %w", err))
	}

	paths := config.GetPaths(cfg)
	database, err := db.New(db.DefaultConfig(paths.Database))
	if err != nil {
		return trackCLIError("install", fmt.Errorf("initialize database: %w", err))
	}
	defer func() { _ = database.Close() }()

	// Create install service
	service := installer.NewInstallService(database, cfg, telemetryClient)

	// Check if input is URL or slug
	if isURL(input) {
		return runInstallFromURL(ctx, service, database, cfg, input)
	}

	return runInstallBySlug(ctx, service, input)
}

func runInstallBySlug(ctx context.Context, service *installer.InstallService, slug string) error {
	// Detect platforms
	platforms, err := service.DetectPlatforms(ctx)
	if err != nil {
		return trackCLIError("install", fmt.Errorf("detect platforms: %w", err))
	}

	// Determine selected platforms
	selectedPlatforms := installPlatforms
	if !installYes && len(selectedPlatforms) == 0 {
		// Interactive mode - show platform selector
		if !isInteractive() {
			return trackCLIError("install", fmt.Errorf("interactive mode requires a terminal, use -y flag"))
		}
		selectedPlatforms, err = prompts.RunPlatformSelector(platforms, installPlatforms)
		if err != nil {
			return trackCLIError("install", fmt.Errorf("platform selection: %w", err))
		}
	}
	if len(selectedPlatforms) == 0 {
		// Default to detected platforms
		selectedPlatforms = prompts.GetDefaultSelectedPlatforms(platforms)
	}
	if len(selectedPlatforms) == 0 {
		return trackCLIError("install", fmt.Errorf("no platforms selected or detected"))
	}

	// Determine scope
	var scopes []installer.InstallScope
	if installScope != "" {
		scopes = prompts.ParseScopeStrings([]string{installScope})
	} else if !installYes {
		// Interactive mode - show scope selector
		if isInteractive() {
			scopeStrs, err := prompts.RunScopeSelector(nil)
			if err != nil {
				return trackCLIError("install", fmt.Errorf("scope selection: %w", err))
			}
			scopes = prompts.ParseScopeStrings(scopeStrs)
		}
	}
	if len(scopes) == 0 {
		scopes = []installer.InstallScope{installer.ScopeGlobal}
	}

	// Perform installation
	opts := installer.InstallOptions{
		Platforms: selectedPlatforms,
		Scopes:    scopes,
		Confirm:   true,
	}

	fmt.Printf("Installing %s...\n", slug)
	result, err := service.Install(ctx, slug, opts)
	if err != nil {
		return trackCLIError("install", fmt.Errorf("install failed: %w", err))
	}

	// Print results
	if len(result.Locations) == 0 {
		fmt.Println("  No installations performed.")
	} else {
		for _, loc := range result.Locations {
			fmt.Printf("  ✓ %s (%s)\n", loc.Platform, loc.Scope)
		}
		fmt.Printf("\nDone! Installed to %d location(s).\n", len(result.Locations))
	}

	return nil
}

func runInstallFromURL(ctx context.Context, service *installer.InstallService, database *db.DB, cfg *config.Config, url string) error {
	fmt.Printf("Fetching skills from %s...\n", url)

	// Parse and add repository if needed
	source, err := scraper.ParseRepositoryURL(url)
	if err != nil {
		return trackCLIError("install", fmt.Errorf("invalid repository URL: %w", err))
	}

	// Check if source exists
	existing, _ := database.GetSource(source.ID)
	if existing == nil {
		// Add the source
		fmt.Printf("Adding repository %s...\n", source.FullName)
		if err := database.UpsertSource(source); err != nil {
			return trackCLIError("install", fmt.Errorf("add repository: %w", err))
		}

		// Sync the repository
		scraperCfg := scraper.ScraperConfig{
			Token:        cfg.GitHub.Token,
			DataDir:      cfg.BaseDir,
			RepoCacheTTL: cfg.GitHub.RepoCacheTTL,
			UseGitClone:  cfg.GitHub.UseGitClone,
		}
		s := scraper.NewScraperWithConfig(scraperCfg, database)

		_, err := s.ScrapeRepository(ctx, source.Owner, source.Repo)
		if err != nil {
			return trackCLIError("install", fmt.Errorf("sync repository: %w", err))
		}
	}

	// Get skills from source
	skills, err := database.GetSkillsBySourceID(source.ID)
	if err != nil {
		return trackCLIError("install", fmt.Errorf("get skills: %w", err))
	}

	if len(skills) == 0 {
		fmt.Println("No skills found in repository.")
		return nil
	}

	fmt.Printf("Found %d skills.\n\n", len(skills))

	// Show skill selector if not -y
	var selectedSlugs []string
	if installYes {
		// Install all skills
		for _, s := range skills {
			selectedSlugs = append(selectedSlugs, s.Slug)
		}
	} else {
		if !isInteractive() {
			return trackCLIError("install", fmt.Errorf("interactive mode requires a terminal, use -y flag"))
		}
		selectedSlugs, err = prompts.RunSkillSelector(skills, nil)
		if err != nil {
			return trackCLIError("install", fmt.Errorf("skill selection: %w", err))
		}
	}

	if len(selectedSlugs) == 0 {
		fmt.Println("No skills selected.")
		return nil
	}

	// Install each selected skill
	var installErrors []error
	for _, slug := range selectedSlugs {
		if err := runInstallBySlug(ctx, service, slug); err != nil {
			installErrors = append(installErrors, fmt.Errorf("%s: %w", slug, err))
		}
	}

	if len(installErrors) > 0 {
		fmt.Printf("\n%d skill(s) failed to install:\n", len(installErrors))
		for _, e := range installErrors {
			fmt.Printf("  ✗ %v\n", e)
		}
	}

	return nil
}

// isURL checks if the input looks like a URL or owner/repo format.
func isURL(s string) bool {
	// Explicit URLs
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return true
	}
	// owner/repo format (but not ./path or ../path)
	if strings.Contains(s, "/") && !strings.HasPrefix(s, ".") {
		return true
	}
	return false
}

// isInteractive checks if we're running in an interactive terminal.
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}
