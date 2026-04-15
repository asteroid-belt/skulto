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
	"github.com/asteroid-belt/skulto/internal/manifest"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/scraper"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var syncYes bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Install skills from skulto.json",
	Long: `Install all skills listed in skulto.json.

Reads the project manifest and installs any skills not already present.
If a source repository is not in your database, you will be prompted to add it.

This is equivalent to running 'skulto install' with no arguments.

Examples:
  skulto sync
  skulto sync -y        # non-interactive: detected platforms, global scope
  skulto install`,
	Args: cobra.NoArgs,
	RunE: runSync,
}

func init() {
	syncCmd.Flags().BoolVarP(&syncYes, "yes", "y", false, "Skip interactive prompts; use detected platforms with global scope")
}

func runSync(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	cwd, err := os.Getwd()
	if err != nil {
		return trackCLIError("sync", fmt.Errorf("get working directory: %w", err))
	}

	mf, err := manifest.Read(cwd)
	if err != nil {
		return trackCLIError("sync", fmt.Errorf("read manifest: %w", err))
	}
	if mf == nil {
		fmt.Println("No skulto.json found in the current directory.")
		fmt.Println()
		fmt.Println("Create one with:")
		fmt.Println("  skulto save")
		return nil
	}

	if mf.SkillCount() == 0 {
		fmt.Println("skulto.json has no skills listed.")
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return trackCLIError("sync", fmt.Errorf("load config: %w", err))
	}

	paths := config.GetPaths(cfg)
	database, err := db.New(db.DefaultConfig(paths.Database))
	if err != nil {
		return trackCLIError("sync", fmt.Errorf("initialize database: %w", err))
	}
	defer func() { _ = database.Close() }()

	service := installer.NewInstallService(database, cfg, telemetryClient)

	headerStyle := lipgloss.NewStyle().Bold(true)
	fmt.Printf("%s (%d skills)\n", headerStyle.Render("SYNCING from skulto.json"), mf.SkillCount())
	fmt.Println(strings.Repeat("\u2500", 50))

	reader := bufio.NewReader(os.Stdin)

	skippedSources, err := syncEnsureSources(ctx, mf, database, cfg, reader, syncYes)
	if err != nil {
		return trackCLIError("sync", err)
	}

	skillsToInstall, skippedSkills := syncResolveSkills(mf, database, skippedSources)

	if len(skillsToInstall) == 0 {
		fmt.Println("\nNo skills to install.")
		if skippedSkills > 0 {
			fmt.Printf("(%d skill(s) skipped)\n", skippedSkills)
		}
		return nil
	}

	fmt.Printf("\n%d skill(s) to install. Select where to install them:\n\n", len(skillsToInstall))

	plan, err := buildSyncPlan(ctx, service, database, syncYes)
	if err != nil {
		return trackCLIError("sync", err)
	}
	if len(plan) == 0 {
		fmt.Println("Cancelled.")
		return nil
	}

	installed, skipped, errored := syncInstallSkills(ctx, skillsToInstall, service, plan, reader, cwd, syncYes)

	fmt.Println()
	fmt.Println(strings.Repeat("\u2500", 50))
	fmt.Printf("Done! Installed: %d, Skipped: %d", installed, skipped)
	if errored > 0 {
		fmt.Printf(", Errors: %d", errored)
	}
	if skippedSkills > 0 {
		fmt.Printf(", Not found: %d", skippedSkills)
	}
	fmt.Println()

	telemetryClient.TrackManifestSynced(mf.SkillCount(), installed, skipped)

	return nil
}

// syncEnsureSources checks that all sources referenced in the manifest exist in the database.
// For missing sources, it prompts the user to add them. Returns a set of skipped source names.
func syncEnsureSources(
	ctx context.Context,
	mf *manifest.ManifestFile,
	database *db.DB,
	cfg *config.Config,
	reader *bufio.Reader,
	yes bool,
) (map[string]bool, error) {
	sourceSkills := make(map[string][]string)
	for slug, source := range mf.Skills {
		sourceSkills[source] = append(sourceSkills[source], slug)
	}

	skippedSources := make(map[string]bool)

	for sourceName := range sourceSkills {
		source, err := database.GetSource(sourceName)
		if err != nil || source == nil {
			skipped, err := syncPromptAddSource(ctx, sourceName, database, cfg, reader, yes)
			if err != nil {
				return nil, err
			}
			if skipped {
				skippedSources[sourceName] = true
			}
		}
	}

	return skippedSources, nil
}

// syncPromptAddSource prompts the user to add a missing source repository.
// Returns true if the source was skipped (not added).
func syncPromptAddSource(
	ctx context.Context,
	sourceName string,
	database *db.DB,
	cfg *config.Config,
	reader *bufio.Reader,
	yes bool,
) (bool, error) {
	fmt.Printf("\nSource '%s' not found in your database.\n", sourceName)

	if !yes {
		fmt.Print("Add it? [y/N] ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Printf("  Skipping all skills from %s\n", sourceName)
			return true, nil
		}
	} else {
		fmt.Println("Auto-adding (via --yes)")
	}

	fmt.Printf("Adding %s...\n", sourceName)

	parts := strings.SplitN(sourceName, "/", 2)
	if len(parts) != 2 {
		fmt.Printf("  Invalid source format: %s (expected owner/repo)\n", sourceName)
		return true, nil
	}

	scraperCfg := scraper.ScraperConfig{
		Token:        cfg.GitHub.Token,
		DataDir:      cfg.BaseDir,
		RepoCacheTTL: cfg.GitHub.RepoCacheTTL,
		UseGitClone:  cfg.GitHub.UseGitClone,
	}
	sc := scraper.NewScraperWithConfig(scraperCfg, database)

	if _, err := sc.ScrapeRepository(ctx, parts[0], parts[1]); err != nil {
		fmt.Printf("  Failed to add source: %v\n", err)
		return true, nil
	}

	fmt.Printf("  Added %s successfully.\n", sourceName)
	return false, nil
}

// syncResolveSkills resolves manifest slugs to database skills, skipping any from skipped sources
// or not found in the database. Returns the skills to install and the count of skipped skills.
func syncResolveSkills(
	mf *manifest.ManifestFile,
	database *db.DB,
	skippedSources map[string]bool,
) ([]*models.Skill, int) {
	var skillsToInstall []*models.Skill
	var skippedSkills int

	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	for _, slug := range mf.SortedSlugs() {
		sourceName := mf.Skills[slug]

		if skippedSources[sourceName] {
			skippedSkills++
			continue
		}

		skill, err := database.GetSkillBySlug(slug)
		if err != nil || skill == nil {
			fmt.Printf("  %s Skill '%s' not found in database\n", warnStyle.Render("WARN"), slug)
			skippedSkills++
			continue
		}

		if skill.Source != nil && skill.Source.FullName != sourceName {
			fmt.Printf("  %s Skill '%s' found but from different source (%s, expected %s)\n",
				warnStyle.Render("WARN"), slug, skill.Source.FullName, sourceName)
			skippedSkills++
			continue
		}

		skillsToInstall = append(skillsToInstall, skill)
	}

	return skillsToInstall, skippedSkills
}

// syncInstallSkills installs the resolved skills against every entry in the plan.
// Each plan entry is a (platforms, scope) InstallOptions; multi-entry plans occur
// when the user has remembered heterogeneous pairs (e.g. claude=project + cursor=global).
func syncInstallSkills(
	ctx context.Context,
	skills []*models.Skill,
	service *installer.InstallService,
	plan []installer.InstallOptions,
	reader *bufio.Reader,
	cwd string,
	yes bool,
) (installed, skipped, errored int) {
	fmt.Println()
	fmt.Println(strings.Repeat("\u2500", 50))

	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	skipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	skipAll := false

	for _, skill := range skills {
		locations, _ := service.GetInstallLocations(ctx, skill.Slug)
		if syncPlanFullyInstalled(locations, plan, cwd) {
			fmt.Printf("  %s %s (already installed)\n", skipStyle.Render("o"), skill.Slug)
			skipped++
			continue
		}

		relevantLocations := syncFilterRelevantLocations(locations, cwd)

		if len(relevantLocations) > 0 && !skipAll && !yes {
			fmt.Printf("\n  '%s' is already installed at some locations.\n", skill.Slug)
			fmt.Print("  Also install to your selected locations? [y/N/s(kip all)] ")

			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))

			if answer == "s" {
				skipAll = true
				fmt.Printf("  %s %s (skipped)\n", skipStyle.Render("o"), skill.Slug)
				skipped++
				continue
			}
			if answer != "y" && answer != "yes" {
				fmt.Printf("  %s %s (skipped)\n", skipStyle.Render("o"), skill.Slug)
				skipped++
				continue
			}
		}

		var lastErr error
		entryOK := 0
		for _, opts := range plan {
			if _, err := service.Install(ctx, skill.Slug, opts); err != nil {
				lastErr = err
				continue
			}
			entryOK++
		}

		if entryOK == 0 {
			fmt.Printf("  %s %s: %v\n", errorStyle.Render("x"), skill.Slug, lastErr)
			errored++
			continue
		}

		fmt.Printf("  %s %s\n", successStyle.Render("v"), skill.Slug)
		installed++
	}

	return installed, skipped, errored
}

// buildSyncPlan decides where the manifest's skills should be installed.
//
// Non-interactive (--yes or piped stdin):
//  1. If the user has opted into remembered install locations, honor each
//     enabled (platform, scope) pair individually. This mirrors `install -y`.
//  2. Otherwise fall back to detected platforms at **project scope**, because
//     skulto.json is a project manifest by design (see cert 2u).
//
// Interactive: prompt for platforms and scopes as before, returning a single
// combined plan entry.
func buildSyncPlan(ctx context.Context, service *installer.InstallService, database *db.DB, yes bool) ([]installer.InstallOptions, error) {
	platforms, err := service.DetectPlatforms(ctx)
	if err != nil {
		return nil, fmt.Errorf("detect platforms: %w", err)
	}

	if yes || !isInteractive() {
		if remember, _ := database.GetRememberInstallLocations(); remember {
			saved, err := database.GetEnabledAgentScopes()
			if err == nil && len(saved) > 0 {
				var plan []installer.InstallOptions
				for platformID, scopeStr := range saved {
					scope := installer.InstallScope(scopeStr)
					if !scope.IsValid() {
						scope = installer.ScopeProject
					}
					plan = append(plan, installer.InstallOptions{
						Platforms: []string{platformID},
						Scopes:    []installer.InstallScope{scope},
						Confirm:   true,
					})
				}
				return plan, nil
			}
		}

		var selected []string
		for _, p := range platforms {
			if p.Detected {
				selected = append(selected, p.ID)
			}
		}
		if len(selected) == 0 {
			return nil, fmt.Errorf("no platforms detected, use interactive mode")
		}
		return []installer.InstallOptions{{
			Platforms: selected,
			Scopes:    []installer.InstallScope{installer.ScopeProject},
			Confirm:   true,
		}}, nil
	}

	result, err := prompts.RunGroupedPlatformSelector(platforms, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("platform selection: %w", err)
	}

	if len(result.Selected) == 0 {
		return nil, nil
	}

	scopeStrs, err := prompts.RunScopeSelector(nil)
	if err != nil {
		return nil, fmt.Errorf("scope selection: %w", err)
	}
	scopes := prompts.ParseScopeStrings(scopeStrs)
	if len(scopes) == 0 {
		scopes = []installer.InstallScope{installer.ScopeProject}
	}

	return []installer.InstallOptions{{
		Platforms: result.Selected,
		Scopes:    scopes,
		Confirm:   true,
	}}, nil
}

// syncPlanFullyInstalled returns true when the skill is already present at
// every (platform, scope) pair the plan would install to. Project-scoped
// installs match cwd; global-scoped installs match the user's home dir.
func syncPlanFullyInstalled(existing []installer.InstallLocation, plan []installer.InstallOptions, cwd string) bool {
	home, _ := os.UserHomeDir()

	for _, opts := range plan {
		for _, p := range opts.Platforms {
			for _, s := range opts.Scopes {
				expectedBase := home
				if s == installer.ScopeProject {
					expectedBase = cwd
				}

				found := false
				for _, loc := range existing {
					if string(loc.Platform) == p && loc.Scope == s && loc.BasePath == expectedBase {
						found = true
						break
					}
				}
				if !found {
					return false
				}
			}
		}
	}
	return true
}

// syncFilterRelevantLocations filters install locations to only those matching
// the current working directory (for project scope) or home directory (for global scope).
func syncFilterRelevantLocations(locations []installer.InstallLocation, cwd string) []installer.InstallLocation {
	home, _ := os.UserHomeDir()

	var relevant []installer.InstallLocation
	for _, loc := range locations {
		expectedBase := home
		if loc.Scope == installer.ScopeProject {
			expectedBase = cwd
		}
		if loc.BasePath == expectedBase {
			relevant = append(relevant, loc)
		}
	}
	return relevant
}
