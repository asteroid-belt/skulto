package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/discovery"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/manifest"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var saveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save project skills to skulto.json",
	Long: `Save currently installed project-scope skills to skulto.json.

This creates a manifest file that can be checked into version control,
allowing teammates to sync the same skills with 'skulto sync'.

Only project-scope installations for the current directory are saved.
Local-only skills (without a source repository) are skipped.

Examples:
  skulto save
  git add skulto.json && git commit -m "chore: add skulto manifest"`,
	Args: cobra.NoArgs,
	RunE: runSave,
}

func runSave(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return trackCLIError("save", fmt.Errorf("get working directory: %w", err))
	}

	cfg, err := config.Load()
	if err != nil {
		return trackCLIError("save", fmt.Errorf("load config: %w", err))
	}

	paths := config.GetPaths(cfg)
	database, err := db.New(db.DefaultConfig(paths.Database))
	if err != nil {
		return trackCLIError("save", fmt.Errorf("initialize database: %w", err))
	}
	defer func() { _ = database.Close() }()

	// Reconcile project skills before querying installations
	inst := installer.New(database, cfg)
	reconcileResult, _ := inst.ReconcileProjectSkills(cwd)
	if reconcileResult != nil && len(reconcileResult.Reconciled) > 0 {
		reconStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
		reconSkillStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
		reconPlatStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		fmt.Printf("%s %d project skill(s)\n", reconStyle.Render("RECONCILED"), len(reconcileResult.Reconciled))
		for _, r := range reconcileResult.Reconciled {
			fmt.Printf("  %s  %s\n", reconSkillStyle.Render(r.Slug), reconPlatStyle.Render(string(r.Platform)))
		}
		fmt.Println()
	}

	installations, err := database.GetProjectInstallations(cwd)
	if err != nil {
		return trackCLIError("save", fmt.Errorf("query installations: %w", err))
	}

	if len(installations) == 0 {
		fmt.Println("No project-scope skills installed for this directory.")
		fmt.Println()
		fmt.Println("Install skills with project scope first:")
		fmt.Println("  skulto install <slug> -s project")
		return nil
	}

	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	var entries []manifest.SkillEntry
	seen := make(map[string]bool)
	for _, inst := range installations {
		if seen[inst.SkillID] {
			continue
		}
		seen[inst.SkillID] = true

		skill, err := database.GetSkill(inst.SkillID)
		if err != nil || skill == nil {
			continue
		}

		localOnly := skill.SourceID == nil || skill.Source == nil
		if localOnly {
			fmt.Printf("  %s %s (local-only, no source repository)\n",
				warnStyle.Render("SKIP"), skill.Slug)
		}

		sourceName := ""
		if skill.Source != nil {
			sourceName = skill.Source.FullName
		}
		entries = append(entries, manifest.SkillEntry{
			Slug:       skill.Slug,
			SourceName: sourceName,
			LocalOnly:  localOnly,
		})
	}

	mf, skippedLocal := manifest.BuildFromSkills(entries)

	// Read existing manifest to compare and determine version
	existing, err := manifest.Read(cwd)
	if err != nil {
		return trackCLIError("save", fmt.Errorf("read existing manifest: %w", err))
	}

	// Check for unmanaged skills in project platform dirs
	if reconcileResult != nil && len(reconcileResult.Unmanaged) > 0 {
		newUnmanaged := filterIgnored(reconcileResult.Unmanaged, existing)
		if len(newUnmanaged) > 0 {
			if isInteractive() {
				handleUnmanagedSkills(newUnmanaged, mf, cwd, database, cfg)
			} else {
				fmt.Printf("Found %d unmanaged skill(s). Run `skulto save` interactively to ingest.\n\n",
					len(newUnmanaged))
			}
		}
	}

	// Merge ignored from existing manifest (preserve previously ignored)
	if existing != nil {
		for _, ig := range existing.Ignored {
			found := false
			for _, mig := range mf.Ignored {
				if mig == ig {
					found = true
					break
				}
			}
			if !found {
				mf.Ignored = append(mf.Ignored, ig)
			}
		}
	}

	if existing != nil && manifest.ManifestEqual(existing, mf) {
		infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		fmt.Printf("%s (version %d)\n", infoStyle.Render("No changes to skulto.json"), existing.Version)
		return nil
	}

	if existing != nil {
		mf.Version = existing.Version + 1
	} else {
		mf.Version = 1
	}

	if err := manifest.Write(cwd, mf); err != nil {
		return trackCLIError("save", fmt.Errorf("write manifest: %w", err))
	}

	fmt.Println()
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	slugStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	sourceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	fmt.Printf("%s Saved to %s (version %d)\n\n", successStyle.Render("SAVED"), manifest.FileName, mf.Version)

	for _, slug := range mf.SortedSlugs() {
		source := mf.Skills[slug]
		fmt.Printf("  %s  %s\n", slugStyle.Render(slug), sourceStyle.Render(source))
	}

	fmt.Printf("\n%d skill(s) saved", mf.SkillCount())
	if skippedLocal > 0 {
		fmt.Printf(", %d local-only skill(s) skipped", skippedLocal)
	}
	fmt.Println()

	telemetryClient.TrackManifestSaved(mf.SkillCount(), "cli")

	return nil
}

// filterIgnored removes entries whose Name appears in the manifest's ignored list.
func filterIgnored(unmanaged []installer.UnmanagedEntry, existing *manifest.ManifestFile) []installer.UnmanagedEntry {
	if existing == nil {
		return unmanaged
	}
	ignoredSet := make(map[string]bool, len(existing.Ignored))
	for _, s := range existing.Ignored {
		ignoredSet[s] = true
	}
	var filtered []installer.UnmanagedEntry
	for _, entry := range unmanaged {
		if !ignoredSet[entry.Name] {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// handleUnmanagedSkills prompts the user about each unmanaged skill.
func handleUnmanagedSkills(unmanaged []installer.UnmanagedEntry, mf *manifest.ManifestFile, cwd string, database *db.DB, cfg *config.Config) {
	reader := bufio.NewReader(os.Stdin)
	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))

	for _, entry := range unmanaged {
		fmt.Printf("\n%s %s (%s)\n", promptStyle.Render("Found unmanaged skill:"), entry.Name, entry.Platform)
		fmt.Print("  [y] Ingest into skulto  [N] Skip  [i] Ignore permanently\n  Choice [N]: ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "y", "yes":
			if err := safeIngestSkill(entry, cwd, database, cfg); err != nil {
				fmt.Printf("  Failed to ingest %s: %v\n", entry.Name, err)
			} else {
				fmt.Printf("  %s %s\n", successStyle.Render("Ingested"), entry.Name)
			}
		case "i", "ignore":
			mf.Ignored = append(mf.Ignored, entry.Name)
			fmt.Printf("  Added %s to ignored list\n", entry.Name)
		default:
			// N or empty — skip
		}
	}
}

// safeIngestSkill ingests an unmanaged skill using the discovery ingestion pipeline.
// Uses IngestOptions to set the correct BasePath for project-scope installations.
func safeIngestSkill(entry installer.UnmanagedEntry, cwd string, database *db.DB, cfg *config.Config) error {
	svc := discovery.NewIngestionService(database, cfg)

	ds := &models.DiscoveredSkill{
		Platform: string(entry.Platform),
		Scope:    "project",
		Path:     entry.Path,
		Name:     entry.Name,
	}
	ds.ID = ds.GenerateID()

	// Pass BasePath = cwd so the installation record is findable by GetProjectInstallations(cwd)
	opts := &discovery.IngestOptions{
		BasePath: cwd,
	}

	result, err := svc.IngestSkill(context.TODO(), ds, opts)
	if err != nil {
		return fmt.Errorf("ingestion failed: %w", err)
	}
	if result.ScanHasWarning {
		fmt.Printf("    ⚠ %-8s %s\n", result.ScanThreatLevel, result.Skill.Slug)
	}

	return nil
}
