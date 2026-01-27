package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// Styles for check command output
var (
	checkHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.AdaptiveColor{Light: "#1A1A1A", Dark: "#E5E5E5"})

	checkSkillStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#6B3FA0", Dark: "#9B59B6"})

	checkGlobalStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#008000", Dark: "#00FF41"})

	checkProjectStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#0088CC", Dark: "#00D4FF"})

	checkBothStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#B8860B", Dark: "#F1C40F"})

	checkMutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#6B6B6B", Dark: "#6B6B6B"})

	checkCountStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#8B0000", Dark: "#DC143C"}).
			Bold(true)
)

var checkCmd = &cobra.Command{
	Use:     "check",
	Aliases: []string{"ck"},
	Short:   "Show installed skills and their locations (alias: ck)",
	Args:    cobra.NoArgs,
	RunE:    runCheck,
}

func runCheck(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return trackCLIError("check", fmt.Errorf("load config: %w", err))
	}

	// Initialize database
	paths := config.GetPaths(cfg)
	database, err := db.New(db.DefaultConfig(paths.Database))
	if err != nil {
		return trackCLIError("check", fmt.Errorf("initialize database: %w", err))
	}
	defer func() { _ = database.Close() }()

	// Create install service
	svc := installer.NewInstallService(database, cfg, telemetryClient)

	// Get installed skills summary
	skills, err := svc.GetInstalledSkillsSummary(cmd.Context())
	if err != nil {
		return trackCLIError("check", fmt.Errorf("get installed skills: %w", err))
	}

	// Print table
	printCheckTable(skills)

	return nil
}

// printCheckTable displays installed skills in a formatted table.
func printCheckTable(skills []installer.InstalledSkillSummary) {
	if len(skills) == 0 {
		fmt.Println(checkMutedStyle.Render("No skills installed."))
		fmt.Println()
		fmt.Println(checkMutedStyle.Render("Use 'skulto install <slug>' to install a skill."))
		return
	}

	// Calculate column widths
	skillWidth := len("SKILL")
	for _, s := range skills {
		if len(s.Slug) > skillWidth {
			skillWidth = len(s.Slug)
		}
	}

	// Print header
	header := fmt.Sprintf("%-*s  %s", skillWidth, "SKILL", "INSTALLED LOCATIONS")
	fmt.Println(checkHeaderStyle.Render(header))
	fmt.Println(checkMutedStyle.Render(strings.Repeat("─", skillWidth+2+40)))

	// Print rows
	for _, s := range skills {
		skillName := checkSkillStyle.Render(fmt.Sprintf("%-*s", skillWidth, s.Slug))
		locations := formatLocationsStyled(s.Locations)
		fmt.Printf("%s  %s\n", skillName, locations)
	}

	// Print count
	fmt.Println()
	fmt.Println(checkCountStyle.Render(fmt.Sprintf("%d skill(s) installed", len(skills))))
}

// formatLocationsStyled formats the platform-scope map with lipgloss styling.
// Example: "claude (global), cursor (global + project)"
func formatLocationsStyled(locations map[installer.Platform][]installer.InstallScope) string {
	if len(locations) == 0 {
		return ""
	}

	// Get sorted platform names
	platforms := make([]string, 0, len(locations))
	for p := range locations {
		platforms = append(platforms, string(p))
	}
	sort.Strings(platforms)

	// Build formatted strings with styling
	parts := make([]string, 0, len(platforms))
	for _, pStr := range platforms {
		p := installer.Platform(pStr)
		scopes := locations[p]
		scopeStr, style := formatScopesStyled(scopes)
		styledScope := style.Render(scopeStr)
		parts = append(parts, fmt.Sprintf("%s (%s)", pStr, styledScope))
	}

	return strings.Join(parts, checkMutedStyle.Render(", "))
}

// formatScopesStyled formats scopes and returns appropriate style.
// Returns: scope string, style to use
func formatScopesStyled(scopes []installer.InstallScope) (string, lipgloss.Style) {
	hasGlobal := false
	hasProject := false

	for _, s := range scopes {
		switch s {
		case installer.ScopeGlobal:
			hasGlobal = true
		case installer.ScopeProject:
			hasProject = true
		}
	}

	switch {
	case hasGlobal && hasProject:
		return "global + project", checkBothStyle
	case hasGlobal:
		return "global", checkGlobalStyle
	case hasProject:
		return "project", checkProjectStyle
	default:
		return "", checkMutedStyle
	}
}
