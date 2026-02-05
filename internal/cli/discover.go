package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/discovery"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/spf13/cobra"
)

var (
	discoverProjectOnly bool
	discoverGlobalOnly  bool
)

// discoverCmd is the discover command instance registered with root.
var discoverCmd = newDiscoverCmd()

// newDiscoverCmd creates a new discover command.
// This is a factory function to support testing.
func newDiscoverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover unmanaged skills in platform directories",
		Long: `Discover unmanaged skill directories in platform skill folders
(e.g., .claude/skills/, .cursor/skills/).

Discovered skills are directories that are not symlinks and thus
not managed by skulto or other tools.

Use 'skulto ingest <name>' to import discovered skills.

Flags:
  --project   Discover only in project-level directories (current working directory)
  --global    Discover only in global directories (home directory)

If neither flag is specified, both project and global directories are scanned.`,
		RunE: runDiscover,
	}

	cmd.Flags().BoolVar(&discoverProjectOnly, "project", false, "Discover only in project-level directories")
	cmd.Flags().BoolVar(&discoverGlobalOnly, "global", false, "Discover only in global directories")

	return cmd
}

// discoveredSkillDisplay wraps a discovered skill with display info.
type discoveredSkillDisplay struct {
	skill        models.DiscoveredSkill
	platformName string
}

func runDiscover(cmd *cobra.Command, args []string) error {
	// Validate flag combination
	if discoverProjectOnly && discoverGlobalOnly {
		return trackCLIError("discover", fmt.Errorf("cannot specify both --project and --global"))
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return trackCLIError("discover", fmt.Errorf("load config: %w", err))
	}

	// Open database
	paths := config.GetPaths(cfg)
	database, err := db.New(db.DefaultConfig(paths.Database))
	if err != nil {
		return trackCLIError("discover", fmt.Errorf("initialize database: %w", err))
	}
	defer func() { _ = database.Close() }()

	// Create scanner service
	scanner := discovery.NewScannerService()

	// Get platform configurations
	service := installer.NewInstallService(database, cfg, telemetryClient)
	platforms, err := service.DetectPlatforms(cmd.Context())
	if err != nil {
		return trackCLIError("discover", fmt.Errorf("detect platforms: %w", err))
	}

	// Build list of directories to scan
	var allDiscovered []discoveredSkillDisplay

	// Determine which scopes to scan
	scanGlobal := !discoverProjectOnly
	scanProject := !discoverGlobalOnly

	for _, platform := range platforms {
		if !platform.Detected {
			continue
		}

		info := installer.PlatformFromString(platform.ID).Info()

		// Scan global directories
		if scanGlobal && info.GlobalDir != "" {
			globalPath := expandPath(info.GlobalDir)
			discovered, err := scanner.ScanDirectory(globalPath, platform.ID, string(installer.ScopeGlobal))
			if err == nil {
				for _, d := range discovered {
					allDiscovered = append(allDiscovered, discoveredSkillDisplay{
						skill:        d,
						platformName: platform.Name,
					})
				}
			}
		}

		// Scan project directories
		if scanProject && info.SkillsPath != "" {
			cwd, err := os.Getwd()
			if err == nil {
				projectPath := cwd + "/" + info.SkillsPath
				discovered, err := scanner.ScanDirectory(projectPath, platform.ID, string(installer.ScopeProject))
				if err == nil {
					for _, d := range discovered {
						allDiscovered = append(allDiscovered, discoveredSkillDisplay{
							skill:        d,
							platformName: platform.Name,
						})
					}
				}
			}
		}
	}

	// Store discovered skills in database
	for _, d := range allDiscovered {
		if err := database.UpsertDiscoveredSkill(&d.skill); err != nil {
			// Log but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to store discovered skill %s: %v\n", d.skill.Name, err)
		}
	}

	// Track telemetry
	telemetryClient.TrackSkillsDiscovered(len(allDiscovered), scanGlobal, scanProject)

	// Display results
	if len(allDiscovered) == 0 {
		fmt.Println("No unmanaged skills discovered.")
		fmt.Println()
		fmt.Println("All skill directories are either symlinks (managed by skulto or other tools) or empty.")
		return nil
	}

	fmt.Printf("Discovered %d unmanaged skill(s):\n\n", len(allDiscovered))

	// Use tabwriter for aligned output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tPLATFORM\tSCOPE\tPATH")
	_, _ = fmt.Fprintln(w, "----\t--------\t-----\t----")

	for _, d := range allDiscovered {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", d.skill.Name, d.platformName, d.skill.Scope, d.skill.Path)
	}
	_ = w.Flush()

	fmt.Println()
	fmt.Println("To import a discovered skill, use:")
	fmt.Println("  skulto ingest <name>")

	return nil
}

// expandPath expands ~ to the user's home directory.
func expandPath(path string) string {
	if len(path) >= 2 && path[0] == '~' && path[1] == '/' {
		home, _ := os.UserHomeDir()
		return home + path[1:]
	}
	return path
}
