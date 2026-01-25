package cli

import (
	"fmt"
	"time"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured source repositories",
	Long: `List all source repositories configured in Skulto.

Shows repository URLs, skill counts (installed/not installed), and last sync times.`,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return trackCLIError("list", fmt.Errorf("load config: %w", err))
	}

	// Initialize database
	paths := config.GetPaths(cfg)
	database, err := db.New(db.DefaultConfig(paths.Database))
	if err != nil {
		return trackCLIError("list", fmt.Errorf("initialize database: %w", err))
	}
	defer func() { _ = database.Close() }()

	// Get all sources
	sources, err := database.ListSources()
	if err != nil {
		return trackCLIError("list", fmt.Errorf("list sources: %w", err))
	}

	// Calculate totals for telemetry
	totalSkillCount := 0
	for _, source := range sources {
		totalSkillCount += int(source.SkillCount)
	}

	// Track the list operation
	telemetryClient.TrackRepoListed(len(sources), totalSkillCount)

	// Display results
	if len(sources) == 0 {
		fmt.Println("No source repositories configured.")
		fmt.Println("\nUse 'skulto add <github-url>' to add a repository.")
		return nil
	}

	fmt.Printf("REPOSITORIES (%d sources)\n", len(sources))
	fmt.Println("──────────────────────────────────────────────────")

	for _, source := range sources {
		// Get skills for this source to count installed vs not installed
		skills, err := database.GetSkillsBySourceID(source.ID)

		var installedCount, notInstalledCount int
		if err == nil {
			for _, skill := range skills {
				if skill.IsInstalled {
					installedCount++
				} else {
					notInstalledCount++
				}
			}
		} else {
			// Fall back to total count if we can't get skills
			notInstalledCount = int(source.SkillCount)
		}

		// Format last sync time
		syncStatus := "never"
		if source.LastScrapedAt != nil && !source.LastScrapedAt.IsZero() {
			syncStatus = formatTimeSince(*source.LastScrapedAt)
		}

		fmt.Printf("  ✓ %s\n", source.FullName)
		fmt.Printf("    %d installed, %d not installed\n", installedCount, notInstalledCount)
		fmt.Printf("    Last synced: %s\n", syncStatus)
		fmt.Println()
	}

	return nil
}

// formatTimeSince formats a duration since a time in a human-readable way.
func formatTimeSince(t time.Time) string {
	diff := time.Since(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("2006-01-02")
	}
}
