package cli

import (
	"fmt"
	"time"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/security"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan skills for security threats",
	Long: `Scan skills in the database for prompt injection and dangerous code patterns.

Examples:
  skulto scan --all              # Scan all skills
  skulto scan --skill abc123     # Scan specific skill by ID
  skulto scan --source owner/repo  # Scan skills from a source
  skulto scan --pending          # Scan only unscanned skills`,
	RunE: runScan,
}

var (
	scanAll     bool
	scanSkillID string
	scanSource  string
	scanPending bool
)

func init() {
	scanCmd.Flags().BoolVar(&scanAll, "all", false, "Scan all skills in database")
	scanCmd.Flags().StringVar(&scanSkillID, "skill", "", "Scan specific skill by ID")
	scanCmd.Flags().StringVar(&scanSource, "source", "", "Scan skills from specific source (owner/repo)")
	scanCmd.Flags().BoolVar(&scanPending, "pending", false, "Scan only unscanned skills")
}

// Color styles for CLI output
var (
	criticalStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true)
	highStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C00"))
	mediumStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700"))
	lowStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	cleanStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
)

func runScan(cmd *cobra.Command, args []string) error {
	start := time.Now()

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

	scanner := security.NewScanner()

	var skills []models.Skill

	switch {
	case scanAll:
		skills, err = database.GetAllSkills()
	case scanSkillID != "":
		skill, err := database.GetSkill(scanSkillID)
		if err != nil || skill == nil {
			return fmt.Errorf("skill not found: %s", scanSkillID)
		}
		skills = []models.Skill{*skill}
	case scanSource != "":
		skills, err = database.GetSkillsBySourceID(scanSource)
	case scanPending:
		skills, err = database.GetPendingSkills()
	default:
		return fmt.Errorf("specify --all, --skill, --source, or --pending")
	}

	if err != nil {
		return err
	}

	if len(skills) == 0 {
		fmt.Println("No skills to scan.")
		return nil
	}

	fmt.Printf("Scanning %d skill(s) for security threats...\n\n", len(skills))

	warningCount := 0
	criticalCount := 0
	highCount := 0
	mediumCount := 0
	lowCount := 0

	for i := range skills {
		skill := &skills[i]
		result := scanner.ScanAndClassify(skill)

		if err := database.UpdateSkillSecurity(skill); err != nil {
			fmt.Println(errorStyle.Render(fmt.Sprintf("Error updating %s: %v", skill.Slug, err)))
			continue
		}

		printScanResult(result, i+1, len(skills))

		if result.HasWarning {
			warningCount++
			switch result.ThreatLevel {
			case models.ThreatLevelCritical:
				criticalCount++
			case models.ThreatLevelHigh:
				highCount++
			case models.ThreatLevelMedium:
				mediumCount++
			case models.ThreatLevelLow:
				lowCount++
			}
		}
	}

	fmt.Println()
	fmt.Printf("Completed in %v\n", time.Since(start).Round(time.Millisecond))

	if warningCount > 0 {
		fmt.Println()
		fmt.Println(highStyle.Render(fmt.Sprintf("Found %d skill(s) with security warnings:", warningCount)))
		if criticalCount > 0 {
			fmt.Println(criticalStyle.Render(fmt.Sprintf("   CRITICAL: %d", criticalCount)))
		}
		if highCount > 0 {
			fmt.Println(highStyle.Render(fmt.Sprintf("   HIGH:     %d", highCount)))
		}
		if mediumCount > 0 {
			fmt.Println(mediumStyle.Render(fmt.Sprintf("   MEDIUM:   %d", mediumCount)))
		}
		if lowCount > 0 {
			fmt.Println(lowStyle.Render(fmt.Sprintf("   LOW:      %d", lowCount)))
		}
	} else {
		fmt.Println()
		fmt.Println(cleanStyle.Render("All skills clean - no threats detected"))
	}

	return nil
}

func printScanResult(result *security.ScanResult, current, total int) {
	prefix := fmt.Sprintf("[%d/%d]", current, total)

	if result.HasWarning {
		var style lipgloss.Style
		switch result.ThreatLevel {
		case models.ThreatLevelCritical:
			style = criticalStyle
		case models.ThreatLevelHigh:
			style = highStyle
		case models.ThreatLevelMedium:
			style = mediumStyle
		default:
			style = lowStyle
		}

		fmt.Printf("%s %s %s [%s]\n",
			prefix,
			style.Render("WARNING"),
			result.SkillSlug,
			result.ThreatLevel,
		)

		if result.ThreatSummary != "" {
			fmt.Printf("    %s\n", result.ThreatSummary)
		}
	} else {
		fmt.Println(cleanStyle.Render(fmt.Sprintf("%s CLEAN   %s", prefix, result.SkillSlug)))
	}
}
