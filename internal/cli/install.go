package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/asteroid-belt/skulto/internal/cli/prompts"
	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/scraper"
	"github.com/asteroid-belt/skulto/internal/security"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	installPlatforms []string
	installScope     string
	installYes       bool
)

var installCmd = &cobra.Command{
	Use:     "install <slug|url>",
	Aliases: []string{"i"},
	Short:   "Install a skill to AI tool directories (alias: i)",
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

Supported platforms (-p flag):
  claude, cursor, copilot, codex, opencode, windsurf,
  amp, kimi-cli, antigravity, moltbot, cline, codebuddy,
  command-code, continue, crush, droid, gemini-cli, goose,
  junie, kilo, kiro-cli, kode, mcpjam, mux, openhands,
  pi, qoder, qwen-code, roo, trae, zencoder, neovate, pochi

Examples:
  # Interactive install
  skulto install docker-expert

  # Non-interactive with defaults
  skulto install docker-expert -y

  # Install to specific platform
  skulto install docker-expert -p claude -y

  # Install to multiple platforms and project scope
  skulto install docker-expert -p claude -p cursor -s project -y

  # Install to a new agent
  skulto install docker-expert -p cline -p roo -y

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

	// Validate platform flags early
	if len(installPlatforms) > 0 {
		if err := validatePlatformFlags(installPlatforms); err != nil {
			return trackCLIError("install", err)
		}
	}

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

// selectPlatformsAndScope resolves the platforms and scopes for installation.
// It shows interactive prompts when needed and returns the resolved options.
// If all platforms are already installed for the given slug, it prints a message and returns nil opts.
func selectPlatformsAndScope(service *installer.InstallService, ctx context.Context, slug string) (*installer.InstallOptions, error) {
	// Detect platforms
	platforms, err := service.DetectPlatforms(ctx)
	if err != nil {
		return nil, fmt.Errorf("detect platforms: %w", err)
	}

	// Get current install locations for this skill
	var installedLocations []installer.InstallLocation
	if slug != "" {
		installedLocations, err = service.GetInstallLocations(ctx, slug)
		if err != nil {
			installedLocations = nil
		}
	}

	// Determine selected platforms
	selectedPlatforms := installPlatforms

	if !installYes && len(selectedPlatforms) == 0 {
		if !isInteractive() {
			return nil, fmt.Errorf("interactive mode requires a terminal, use -y flag")
		}

		result, err := prompts.RunGroupedPlatformSelector(platforms, installedLocations, installPlatforms)
		if err != nil {
			return nil, fmt.Errorf("platform selection: %w", err)
		}

		if result.AllAlreadyInstalled {
			if slug != "" {
				fmt.Printf("✓ %s is already installed to all detected platforms.\n\n", slug)
				fmt.Println("Installed locations:")
				for _, loc := range installedLocations {
					fmt.Printf("  • %s (%s)\n", loc.Platform, loc.Scope)
				}
				fmt.Printf("\nTo install to additional platforms, use: skulto install %s -p <platform>\n", slug)
				fmt.Printf("To remove from locations, use: skulto uninstall %s\n", slug)
			}
			return nil, nil // All installed, nothing to do
		}

		selectedPlatforms = result.Selected
	}

	if len(selectedPlatforms) == 0 {
		selectedPlatforms = prompts.GetDefaultSelectablePlatforms(platforms, installedLocations)
	}
	if len(selectedPlatforms) == 0 {
		fmt.Println("No platforms selected. Nothing to install.")
		return nil, nil
	}

	// Determine scope
	var scopes []installer.InstallScope
	if installScope != "" {
		scopes = prompts.ParseScopeStrings([]string{installScope})
	} else if !installYes {
		if isInteractive() {
			scopeStrs, err := prompts.RunScopeSelector(nil)
			if err != nil {
				return nil, fmt.Errorf("scope selection: %w", err)
			}
			scopes = prompts.ParseScopeStrings(scopeStrs)
		}
	}
	if len(scopes) == 0 {
		scopes = []installer.InstallScope{installer.ScopeGlobal}
	}

	return &installer.InstallOptions{
		Platforms: selectedPlatforms,
		Scopes:    scopes,
		Confirm:   true,
	}, nil
}

func runInstallBySlug(ctx context.Context, service *installer.InstallService, slug string) error {
	opts, err := selectPlatformsAndScope(service, ctx, slug)
	if err != nil {
		return trackCLIError("install", err)
	}
	if opts == nil {
		return nil // All installed or no platforms selected
	}

	return executeInstall(ctx, service, slug, *opts)
}

// executeInstall performs the actual installation with pre-resolved options.
func executeInstall(ctx context.Context, service *installer.InstallService, slug string, opts installer.InstallOptions) error {
	// Get current install locations for comparison
	installedLocations, _ := service.GetInstallLocations(ctx, slug)

	fmt.Printf("Installing %s...\n", slug)
	result, err := service.Install(ctx, slug, opts)
	if err != nil {
		return trackCLIError("install", fmt.Errorf("install failed: %w", err))
	}

	newInstalls := 0
	alreadyInstalledCount := 0
	for _, loc := range result.Locations {
		wasAlreadyInstalled := false
		for _, existingLoc := range installedLocations {
			if existingLoc.Platform == loc.Platform && existingLoc.Scope == loc.Scope {
				wasAlreadyInstalled = true
				break
			}
		}
		if wasAlreadyInstalled {
			fmt.Printf("  ○ %s (%s) - already installed\n", loc.Platform, loc.Scope)
			alreadyInstalledCount++
		} else {
			fmt.Printf("  ✓ %s (%s)\n", loc.Platform, loc.Scope)
			newInstalls++
		}
	}

	if newInstalls == 0 {
		fmt.Println("\nNo new installations performed. All locations were already installed.")
	} else {
		if alreadyInstalledCount > 0 {
			fmt.Printf("\nDone! Installed to %d new location(s). %d location(s) were already installed.\n", newInstalls, alreadyInstalledCount)
		} else {
			fmt.Printf("\nDone! Installed to %d location(s).\n", newInstalls)
		}
		fmt.Println("\nRestart your agent for the skill to take effect.")
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

	// Check if source already exists with skills
	existing, _ := database.GetSource(source.ID)
	needsScrape := existing == nil

	if existing != nil {
		existingSkills, _ := database.GetSkillsBySourceID(source.ID)
		if len(existingSkills) == 0 {
			needsScrape = true
		} else {
			fmt.Printf("Repository %s is already in the database.\n", source.FullName)
		}
	}

	if needsScrape {
		if existing == nil {
			fmt.Printf("Adding repository %s...\n", source.FullName)
		} else {
			fmt.Printf("Re-syncing repository %s...\n", source.FullName)
		}
		if err := database.UpsertSource(source); err != nil {
			return trackCLIError("install", fmt.Errorf("add repository: %w", err))
		}

		scraperCfg := scraper.ScraperConfig{
			Token:        cfg.GitHub.Token,
			DataDir:      cfg.BaseDir,
			RepoCacheTTL: cfg.GitHub.RepoCacheTTL,
			UseGitClone:  cfg.GitHub.UseGitClone,
		}
		s := scraper.NewScraperWithConfig(scraperCfg, database)

		_, err = s.ScrapeRepository(ctx, source.Owner, source.Repo)
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

	fmt.Printf("Found %d skill(s).\n\n", len(skills))

	// Security scan all skills
	hasThreats, err := scanSkillsForInstall(database, skills)
	if err != nil {
		return trackCLIError("install", fmt.Errorf("security scan: %w", err))
	}

	// If threats found, prompt for confirmation (unless -y)
	if hasThreats && !installYes {
		if !isInteractive() {
			fmt.Println()
			fmt.Println("Threats detected. Use -y to install anyway, or run interactively to confirm.")
			_ = removeSourceAndSkills(database, source.ID)
			return trackCLIError("install", fmt.Errorf("security threats detected, installation blocked"))
		}

		fmt.Println()
		fmt.Print("Install anyway? [y/N] ")
		var answer string
		_, _ = fmt.Scanln(&answer)
		if answer != "y" && answer != "Y" {
			fmt.Println("Installation cancelled.")
			_ = removeSourceAndSkills(database, source.ID)
			return nil
		}
	}

	fmt.Println()

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
		// Pre-select skills that are already installed
		var installedSlugs []string
		for _, skill := range skills {
			installations, _ := database.GetInstallations(skill.ID)
			if len(installations) > 0 {
				installedSlugs = append(installedSlugs, skill.Slug)
			}
		}
		selectedSlugs, err = prompts.RunSkillSelector(skills, installedSlugs)
		if err != nil {
			return trackCLIError("install", fmt.Errorf("skill selection: %w", err))
		}
	}

	if len(selectedSlugs) == 0 {
		fmt.Println("No skills selected.")
		return nil
	}

	// Select platforms and scope ONCE for all skills
	opts, err := selectPlatformsAndScope(service, ctx, "")
	if err != nil {
		return trackCLIError("install", err)
	}
	if opts == nil {
		return nil
	}

	// Install each selected skill with smart skip for already-installed skills
	type installSummary struct {
		slug    string
		action  string // "installed", "added", "skipped", "error"
		details string
	}

	var summaries []installSummary
	skipAllInstalled := false
	interactive := isInteractive()
	reader := bufio.NewReader(os.Stdin)

	// Build the set of selected platform+scope pairs for comparison
	type platformScope struct {
		platform string
		scope    installer.InstallScope
	}
	selectedSet := make(map[platformScope]bool)
	for _, p := range opts.Platforms {
		for _, s := range opts.Scopes {
			selectedSet[platformScope{p, s}] = true
		}
	}

	for _, slug := range selectedSlugs {
		// Check existing installations
		existingLocs, _ := service.GetInstallLocations(ctx, slug)

		if len(existingLocs) == 0 {
			// Fresh skill — install directly
			if err := executeInstall(ctx, service, slug, *opts); err != nil {
				summaries = append(summaries, installSummary{slug, "error", err.Error()})
			} else {
				locs := formatOptLocations(opts.Platforms, opts.Scopes)
				summaries = append(summaries, installSummary{slug, "installed", locs})
			}
			continue
		}

		// Already installed somewhere — check which selected locations are new
		existingSet := make(map[platformScope]bool)
		for _, loc := range existingLocs {
			existingSet[platformScope{string(loc.Platform), loc.Scope}] = true
		}

		// Find genuinely new locations
		var newPlatforms []string
		var newScopes []installer.InstallScope
		newScopesSet := make(map[installer.InstallScope]bool)
		newPlatformsSet := make(map[string]bool)
		for ps := range selectedSet {
			if !existingSet[ps] {
				if !newPlatformsSet[ps.platform] {
					newPlatformsSet[ps.platform] = true
					newPlatforms = append(newPlatforms, ps.platform)
				}
				if !newScopesSet[ps.scope] {
					newScopesSet[ps.scope] = true
					newScopes = append(newScopes, ps.scope)
				}
			}
		}

		if len(newPlatforms) == 0 {
			// All selected locations already installed — auto-skip
			summaries = append(summaries, installSummary{slug, "skipped", "already installed at all selected locations"})
			continue
		}

		// Has existing installs + new locations to add — prompt unless skipping all
		if skipAllInstalled {
			summaries = append(summaries, installSummary{slug, "skipped", "already installed (skip all)"})
			continue
		}

		if interactive && !installYes {
			// Show existing locations
			var existingParts []string
			for _, loc := range existingLocs {
				existingParts = append(existingParts, fmt.Sprintf("%s (%s)", loc.Platform, loc.Scope))
			}
			fmt.Printf("\n%s is already installed to:\n", slug)
			for _, part := range existingParts {
				fmt.Printf("  • %s\n", part)
			}
			fmt.Print("\nAlso install to your selected locations? [y/N/s] ")

			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))

			switch answer {
			case "s":
				skipAllInstalled = true
				summaries = append(summaries, installSummary{slug, "skipped", "already installed (skip all)"})
				continue
			case "y":
				// Proceed with new locations only
			default:
				// N or empty — skip
				summaries = append(summaries, installSummary{slug, "skipped", "already installed"})
				continue
			}
		} else {
			// Non-interactive: default to skip
			summaries = append(summaries, installSummary{slug, "skipped", "already installed"})
			continue
		}

		// Install to new locations only
		newOpts := installer.InstallOptions{
			Platforms: newPlatforms,
			Scopes:    newScopes,
			Confirm:   opts.Confirm,
		}
		if err := executeInstall(ctx, service, slug, newOpts); err != nil {
			summaries = append(summaries, installSummary{slug, "error", err.Error()})
		} else {
			locs := formatOptLocations(newPlatforms, newScopes)
			summaries = append(summaries, installSummary{slug, "added", locs})
		}
	}

	// Print final summary
	fmt.Println()
	var installed, skipped, errored []installSummary
	for _, s := range summaries {
		switch s.action {
		case "installed", "added":
			installed = append(installed, s)
		case "skipped":
			skipped = append(skipped, s)
		case "error":
			errored = append(errored, s)
		}
	}

	if len(installed) > 0 {
		fmt.Printf("Installed %d skill(s):\n", len(installed))
		for _, s := range installed {
			fmt.Printf("  ✓ %s → %s\n", s.slug, s.details)
		}
	}
	if len(skipped) > 0 {
		fmt.Printf("Skipped %d skill(s):\n", len(skipped))
		for _, s := range skipped {
			fmt.Printf("  ○ %s (%s)\n", s.slug, s.details)
		}
	}
	if len(errored) > 0 {
		fmt.Printf("Failed %d skill(s):\n", len(errored))
		for _, s := range errored {
			fmt.Printf("  ✗ %s: %s\n", s.slug, s.details)
		}
	}

	return nil
}

// formatOptLocations formats platform+scope pairs for display.
func formatOptLocations(platforms []string, scopes []installer.InstallScope) string {
	var parts []string
	for _, p := range platforms {
		for _, s := range scopes {
			parts = append(parts, fmt.Sprintf("%s (%s)", p, s))
		}
	}
	return strings.Join(parts, ", ")
}

// validatePlatformFlags validates that all platform flag values are valid platform IDs or aliases.
func validatePlatformFlags(platforms []string) error {
	for _, p := range platforms {
		if installer.PlatformFromStringOrAlias(p) == "" {
			return fmt.Errorf("unknown platform %q. Run 'skulto install --help' for valid platforms", p)
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

// scanSkillsForInstall scans all skills for security threats and prints a report.
// Returns true if any threats were found.
func scanSkillsForInstall(database *db.DB, skills []models.Skill) (bool, error) {
	fmt.Println("Scanning skills for security threats...")
	fmt.Println()

	scanner := security.NewScanner()
	hasThreats := false

	categoriesChecked := []string{
		"Frontmatter injection",
		"Dangerous shell patterns",
		"External references",
		"Encoded payloads",
	}

	var threatResults []security.ScanResult
	for i := range skills {
		skill := &skills[i]
		result := scanner.ScanAndClassify(skill)

		if err := database.UpdateSkillSecurity(skill); err != nil {
			fmt.Printf("  Error scanning %s: %v\n", skill.Slug, err)
			continue
		}

		printScanResult(result, i+1, len(skills))

		if result.HasWarning {
			hasThreats = true
			threatResults = append(threatResults, *result)
		}
	}

	fmt.Println()

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#555")).
		Padding(1, 2)

	if !hasThreats {
		var content string
		content += cleanStyle.Render("PASSED") + " - No threats detected\n\n"
		content += "Checked:\n"
		for _, cat := range categoriesChecked {
			content += fmt.Sprintf("  %s  %s\n", cleanStyle.Render("✓"), cat)
		}
		fmt.Println(borderStyle.Render(content))
	} else {
		totalThreats := 0
		critCount, highCount, medCount, lowCount := 0, 0, 0, 0
		for _, r := range threatResults {
			totalThreats += r.TotalMatchCount()
			switch r.ThreatLevel {
			case models.ThreatLevelCritical:
				critCount++
			case models.ThreatLevelHigh:
				highCount++
			case models.ThreatLevelMedium:
				medCount++
			case models.ThreatLevelLow:
				lowCount++
			}
		}

		var content string
		content += highStyle.Render(fmt.Sprintf("⚠ %d RISKY PATTERNS", totalThreats)) + "\n\n"
		if critCount > 0 {
			content += criticalStyle.Render(fmt.Sprintf("  CRITICAL  %d skill(s)", critCount)) + "\n"
		}
		if highCount > 0 {
			content += highStyle.Render(fmt.Sprintf("  HIGH      %d skill(s)", highCount)) + "\n"
		}
		if medCount > 0 {
			content += mediumStyle.Render(fmt.Sprintf("  MEDIUM    %d skill(s)", medCount)) + "\n"
		}
		if lowCount > 0 {
			content += lowStyle.Render(fmt.Sprintf("  LOW       %d skill(s)", lowCount)) + "\n"
		}
		fmt.Println(borderStyle.Render(content))
	}

	return hasThreats, nil
}

// removeSourceAndSkills removes a source and all its skills from the database.
// Used when a user declines to install after security threats are found.
func removeSourceAndSkills(database *db.DB, sourceID string) error {
	skills, err := database.GetSkillsBySourceID(sourceID)
	if err != nil {
		return fmt.Errorf("get skills for cleanup: %w", err)
	}

	for _, skill := range skills {
		_ = database.RemoveAllInstallations(skill.ID)
	}

	if _, err := database.HardDeleteSkillsBySource(sourceID); err != nil {
		return fmt.Errorf("remove skills: %w", err)
	}

	if err := database.HardDeleteSource(sourceID); err != nil {
		return fmt.Errorf("remove source: %w", err)
	}

	return nil
}
