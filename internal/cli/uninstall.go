package cli

import (
	"fmt"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var (
	uninstallYes bool
	uninstallAll bool
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <slug>",
	Short: "Uninstall a skill from AI tool directories",
	Long: `Uninstall a skill by removing its symlinks from AI tool directories.

Interactive Mode (default):
  Shows installed locations and lets you select which to remove.

Non-Interactive Mode (-y):
  Removes from all installed locations.

Examples:
  # Interactive uninstall - choose which locations
  skulto uninstall docker-expert

  # Remove from all locations
  skulto uninstall docker-expert -y

  # Same as -y, explicit flag
  skulto uninstall docker-expert --all`,
	Args: cobra.ExactArgs(1),
	RunE: runUninstall,
}

func init() {
	uninstallCmd.Flags().BoolVarP(&uninstallYes, "yes", "y", false,
		"Skip interactive prompts, remove from all locations")
	uninstallCmd.Flags().BoolVarP(&uninstallAll, "all", "a", false,
		"Remove from all installed locations (same as -y)")
}

func runUninstall(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	slug := args[0]

	// Load config and database
	cfg, err := config.Load()
	if err != nil {
		return trackCLIError("uninstall", fmt.Errorf("load config: %w", err))
	}

	paths := config.GetPaths(cfg)
	database, err := db.New(db.DefaultConfig(paths.Database))
	if err != nil {
		return trackCLIError("uninstall", fmt.Errorf("initialize database: %w", err))
	}
	defer func() { _ = database.Close() }()

	// Create install service
	service := installer.NewInstallService(database, cfg, telemetryClient)

	// Get current install locations
	locations, err := service.GetInstallLocations(ctx, slug)
	if err != nil {
		return trackCLIError("uninstall", fmt.Errorf("get install locations: %w", err))
	}

	if len(locations) == 0 {
		fmt.Printf("Skill '%s' is not installed anywhere.\n", slug)
		return nil
	}

	fmt.Printf("Found %d installation(s) of '%s':\n", len(locations), slug)
	for i, loc := range locations {
		fmt.Printf("  %d. %s (%s)\n", i+1, loc.Platform, loc.Scope)
	}
	fmt.Println()

	// Determine which locations to uninstall
	var toUninstall []installer.InstallLocation
	if uninstallYes || uninstallAll {
		// Non-interactive: remove all
		toUninstall = locations
	} else {
		// Interactive: show location selector
		if !isInteractive() {
			return trackCLIError("uninstall", fmt.Errorf("interactive mode requires a terminal, use -y flag"))
		}
		toUninstall, err = runLocationSelector(locations)
		if err != nil {
			return trackCLIError("uninstall", fmt.Errorf("location selection: %w", err))
		}
	}

	if len(toUninstall) == 0 {
		fmt.Println("No locations selected for removal.")
		return nil
	}

	// Perform uninstallation
	fmt.Printf("Uninstalling from %d location(s)...\n", len(toUninstall))
	if err := service.Uninstall(ctx, slug, toUninstall); err != nil {
		return trackCLIError("uninstall", fmt.Errorf("uninstall failed: %w", err))
	}

	// Print results
	for _, loc := range toUninstall {
		fmt.Printf("  ✓ Removed from %s (%s)\n", loc.Platform, loc.Scope)
	}
	fmt.Println("\nDone!")

	return nil
}

// runLocationSelector shows interactive location selection for uninstall.
func runLocationSelector(locations []installer.InstallLocation) ([]installer.InstallLocation, error) {
	// Build options
	options := make([]huh.Option[int], 0, len(locations))
	for i, loc := range locations {
		label := fmt.Sprintf("%s (%s)", loc.Platform, loc.Scope)
		options = append(options, huh.NewOption(label, i))
	}

	var selectedIndices []int
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[int]().
				Title("Select locations to uninstall from").
				Description("Space to toggle, Enter to confirm").
				Options(options...).
				Value(&selectedIndices),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	// Convert indices to locations
	result := make([]installer.InstallLocation, 0, len(selectedIndices))
	for _, idx := range selectedIndices {
		if idx >= 0 && idx < len(locations) {
			result = append(result, locations[idx])
		}
	}

	return result, nil
}
