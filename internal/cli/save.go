package cli

import (
	"fmt"
	"os"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/manifest"
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

	if err := manifest.Write(cwd, mf); err != nil {
		return trackCLIError("save", fmt.Errorf("write manifest: %w", err))
	}

	fmt.Println()
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	slugStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	sourceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	fmt.Printf("%s Saved to %s\n\n", successStyle.Render("SAVED"), manifest.FileName)

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
