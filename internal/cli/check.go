package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/spf13/cobra"
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
		fmt.Println("No skills installed.")
		fmt.Println("\nUse 'skulto install <slug>' to install a skill.")
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
	fmt.Printf("%-*s  INSTALLED LOCATIONS\n", skillWidth, "SKILL")
	fmt.Println(strings.Repeat("-", skillWidth+2+30))

	// Print rows
	for _, s := range skills {
		locations := formatLocations(s.Locations)
		fmt.Printf("%-*s  %s\n", skillWidth, s.Slug, locations)
	}
}

// formatLocations formats the platform-scope map as a readable string.
// Example: "claude (global), cursor (global + project)"
func formatLocations(locations map[installer.Platform][]installer.InstallScope) string {
	if len(locations) == 0 {
		return ""
	}

	// Get sorted platform names
	platforms := make([]string, 0, len(locations))
	for p := range locations {
		platforms = append(platforms, string(p))
	}
	sort.Strings(platforms)

	// Build formatted strings
	parts := make([]string, 0, len(platforms))
	for _, pStr := range platforms {
		p := installer.Platform(pStr)
		scopes := locations[p]
		scopeStr := formatScopes(scopes)
		parts = append(parts, fmt.Sprintf("%s (%s)", pStr, scopeStr))
	}

	return strings.Join(parts, ", ")
}

// formatScopes formats scopes as "global", "project", or "global + project".
func formatScopes(scopes []installer.InstallScope) string {
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
		return "global + project"
	case hasGlobal:
		return "global"
	case hasProject:
		return "project"
	default:
		return ""
	}
}
