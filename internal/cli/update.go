package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/scraper"
	"github.com/asteroid-belt/skulto/internal/security"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"up"},
	Short:   "Pull repositories, scan for threats, and report changes (alias: up)",
	Long: `Update pulls all registered skill repositories, runs security scans
on new and updated skills, and displays a summary of what changed.

This is equivalent to running:
  skulto pull
  skulto scan --pending

But with enhanced reporting of updated skills.

Examples:
  # Update all repositories and scan new skills
  skulto update

  # Update and scan ALL skills (not just new/updated)
  skulto update --scan-all`,
	Args: cobra.NoArgs,
	RunE: runUpdate,
}

var updateScanAll bool

func init() {
	updateCmd.Flags().BoolVar(&updateScanAll, "scan-all", false,
		"Scan all skills, not just newly updated ones")
}

// SkillChange tracks what changed for a skill during update.
type SkillChange struct {
	Skill      models.Skill
	ChangeType string // "new", "updated"
}

// UpdateResult tracks the results of an update operation.
type UpdateResult struct {
	// Pull phase results
	ReposSynced   int
	ReposErrored  int
	SkillsNew     int
	SkillsUpdated int
	UpdatedSkills []models.Skill
	Changes       []SkillChange

	// Scan phase results
	SkillsScanned   int
	ThreatsCritical int
	ThreatsHigh     int
	ThreatsMedium   int
	ThreatsLow      int
	SkillsClean     int
}

func runUpdate(cmd *cobra.Command, args []string) error {
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

	result := &UpdateResult{}

	// Phase 1: Pull repositories
	fmt.Println("🔄 [1/3] Pulling skill repositories...")
	fmt.Println()

	if err := runUpdatePull(ctx, cfg, database, result); err != nil {
		return err
	}

	// Phase 2: Security scan
	fmt.Println()
	fmt.Println("🔒 [2/3] Scanning for security threats...")
	fmt.Println()

	if err := runUpdateScan(ctx, database, result); err != nil {
		return err
	}

	// Phase 3: Report
	fmt.Println()
	fmt.Println("📋 [3/3] Update Summary")
	fmt.Println()

	printUpdateReport(result)

	return nil
}

func runUpdatePull(ctx context.Context, cfg *config.Config, database *db.DB, result *UpdateResult) error {
	sources, err := database.ListSources()
	if err != nil {
		return fmt.Errorf("list sources: %w", err)
	}

	if len(sources) == 0 {
		fmt.Println("   📦 No repositories configured. Use 'skulto add <repo>' to add one.")
		return nil
	}

	// Create scraper
	scraperCfg := scraper.ScraperConfig{
		Token:        cfg.GitHub.Token,
		DataDir:      cfg.BaseDir,
		RepoCacheTTL: cfg.GitHub.RepoCacheTTL,
		UseGitClone:  cfg.GitHub.UseGitClone,
	}
	s := scraper.NewScraperWithConfig(scraperCfg, database)

	// Track skills before pull to detect updates
	skillsBefore := make(map[string]string) // ID -> ContentHash
	allSkills, _ := database.GetAllSkills()
	for _, skill := range allSkills {
		skillsBefore[skill.ID] = skill.ContentHash
	}

	// Initialize progress bar
	progress := NewProgressBar(len(sources), 15)

	// Sync each repository
	for i, source := range sources {
		repoName := fmt.Sprintf("%s/%s", source.Owner, source.Repo)
		progress.Update(i+1, repoName)
		ClearLine()
		fmt.Print("   " + progress.Render())

		syncCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		scrapeResult, err := s.ScrapeRepository(syncCtx, source.Owner, source.Repo)
		cancel()

		if err != nil {
			ClearLine()
			fmt.Printf("   ❌ %s: %v\n", repoName, err)
			result.ReposErrored++
			continue
		}

		result.ReposSynced++
		result.SkillsNew += scrapeResult.SkillsNew
		result.SkillsUpdated += scrapeResult.SkillsUpdated

		// Track telemetry per repo
		telemetryClient.TrackRepoSynced(source.ID, scrapeResult.SkillsNew, 0, scrapeResult.SkillsUpdated)
	}

	// Final progress
	ClearLine()
	fmt.Println("   ✓ Pull complete")

	// Sync install state
	fmt.Println("   🔍 Reconciling install state...")
	inst := installer.New(database, cfg)
	if err := inst.SyncInstallState(ctx); err != nil {
		fmt.Printf("   ⚠️  Warning: %v\n", err)
	} else {
		fmt.Println("   ✓ Install state reconciled")
	}

	// Collect updated skills for reporting
	allSkillsAfter, _ := database.GetAllSkills()
	for _, skill := range allSkillsAfter {
		oldHash, existed := skillsBefore[skill.ID]
		if !existed {
			result.UpdatedSkills = append(result.UpdatedSkills, skill)
			result.Changes = append(result.Changes, SkillChange{
				Skill:      skill,
				ChangeType: "new",
			})
		} else if oldHash != skill.ContentHash {
			result.UpdatedSkills = append(result.UpdatedSkills, skill)
			result.Changes = append(result.Changes, SkillChange{
				Skill:      skill,
				ChangeType: "updated",
			})
		}
	}

	return nil
}

func runUpdateScan(_ context.Context, database *db.DB, result *UpdateResult) error {
	var skills []models.Skill
	var err error

	if updateScanAll {
		skills, err = database.GetAllSkills()
	} else {
		skills, err = database.GetPendingSkills()
		if len(skills) == 0 {
			// Also scan any skills that were just updated
			for _, skill := range result.UpdatedSkills {
				if skill.SecurityStatus == models.SecurityStatusPending {
					skills = append(skills, skill)
				}
			}
		}
	}

	if err != nil {
		return fmt.Errorf("get skills for scan: %w", err)
	}

	if len(skills) == 0 {
		fmt.Println("   ✓ No skills require scanning")
		return nil
	}

	scanner := security.NewScanner()

	// Initialize progress bar
	progress := NewProgressBar(len(skills), 15)

	for i := range skills {
		skill := &skills[i]

		// Update progress
		progress.Update(i+1, skill.Slug)
		ClearLine()
		fmt.Print("   " + progress.RenderScan())

		scanResult := scanner.ScanSkill(skill)

		// Update skill in database
		now := time.Now()
		skill.SecurityStatus = models.SecurityStatusClean
		skill.ThreatLevel = scanResult.MaxThreatLevel()
		skill.ThreatSummary = scanResult.ThreatSummary
		skill.ScannedAt = &now
		skill.ContentHash = skill.ComputeContentHash()

		if scanResult.HasWarning {
			skill.SecurityStatus = models.SecurityStatusQuarantined
		}

		if err := database.UpdateSkillSecurity(skill); err != nil {
			ClearLine()
			fmt.Printf("   ❌ Error updating %s: %v\n", skill.Slug, err)
			continue
		}

		result.SkillsScanned++

		// Track threat levels
		if scanResult.HasWarning {
			switch scanResult.ThreatLevel {
			case models.ThreatLevelCritical:
				result.ThreatsCritical++
			case models.ThreatLevelHigh:
				result.ThreatsHigh++
			case models.ThreatLevelMedium:
				result.ThreatsMedium++
			case models.ThreatLevelLow:
				result.ThreatsLow++
			}
		} else {
			result.SkillsClean++
		}
	}

	// Final progress
	ClearLine()
	fmt.Println("   ✓ Scan complete")

	return nil
}

func printUpdateReport(result *UpdateResult) {
	// Summary box
	fmt.Println("   ┌─────────────────────────────────────")
	fmt.Println("   │ REPOSITORIES")
	fmt.Printf("   │   Synced:  %d\n", result.ReposSynced)
	if result.ReposErrored > 0 {
		fmt.Printf("   │   Errors:  %s\n", errorStyle.Render(fmt.Sprintf("%d", result.ReposErrored)))
	}
	fmt.Println("   │")
	fmt.Println("   │ SKILLS")
	fmt.Printf("   │   New:     %d\n", result.SkillsNew)
	fmt.Printf("   │   Updated: %d\n", result.SkillsUpdated)
	fmt.Println("   │")

	// Scan summary
	fmt.Println("   │ SECURITY SCAN")
	fmt.Printf("   │   Scanned: %d\n", result.SkillsScanned)
	fmt.Printf("   │   Clean:   %s\n", cleanStyle.Render(fmt.Sprintf("%d", result.SkillsClean)))

	totalThreats := result.ThreatsCritical + result.ThreatsHigh + result.ThreatsMedium + result.ThreatsLow
	if totalThreats > 0 {
		fmt.Printf("   │   Threats: %d\n", totalThreats)
		if result.ThreatsCritical > 0 {
			fmt.Printf("   │     %s\n", criticalStyle.Render(fmt.Sprintf("CRITICAL: %d", result.ThreatsCritical)))
		}
		if result.ThreatsHigh > 0 {
			fmt.Printf("   │     %s\n", highStyle.Render(fmt.Sprintf("HIGH: %d", result.ThreatsHigh)))
		}
		if result.ThreatsMedium > 0 {
			fmt.Printf("   │     %s\n", mediumStyle.Render(fmt.Sprintf("MEDIUM: %d", result.ThreatsMedium)))
		}
		if result.ThreatsLow > 0 {
			fmt.Printf("   │     %s\n", lowStyle.Render(fmt.Sprintf("LOW: %d", result.ThreatsLow)))
		}
	}

	fmt.Println("   └─────────────────────────────────────")

	// List updated skills grouped by change type
	if len(result.Changes) > 0 {
		fmt.Println()
		fmt.Println("   CHANGED SKILLS:")

		// Group by change type
		newSkills := []SkillChange{}
		updatedSkills := []SkillChange{}
		for _, change := range result.Changes {
			if change.ChangeType == "new" {
				newSkills = append(newSkills, change)
			} else {
				updatedSkills = append(updatedSkills, change)
			}
		}

		if len(newSkills) > 0 {
			fmt.Println()
			fmt.Printf("   New (%d):\n", len(newSkills))
			for _, change := range newSkills {
				threatIndicator := getThreatIndicator(change.Skill.ThreatLevel)
				fmt.Printf("   • %s%s\n", change.Skill.Title, threatIndicator)
			}
		}

		if len(updatedSkills) > 0 {
			fmt.Println()
			fmt.Printf("   Updated (%d):\n", len(updatedSkills))
			for _, change := range updatedSkills {
				threatIndicator := getThreatIndicator(change.Skill.ThreatLevel)
				fmt.Printf("   • %s%s\n", change.Skill.Title, threatIndicator)
			}
		}
	}

	fmt.Println()
	fmt.Println("✓ Update complete!")
}

func getThreatIndicator(level models.ThreatLevel) string {
	switch level {
	case models.ThreatLevelCritical:
		return criticalStyle.Render(" [CRITICAL]")
	case models.ThreatLevelHigh:
		return highStyle.Render(" [HIGH]")
	case models.ThreatLevelMedium:
		return mediumStyle.Render(" [MEDIUM]")
	case models.ThreatLevelLow:
		return lowStyle.Render(" [LOW]")
	default:
		return ""
	}
}
