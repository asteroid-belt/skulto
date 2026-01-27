package cli

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/scraper"
	"github.com/asteroid-belt/skulto/internal/tui/components"
)

var removeForce bool

var removeCmd = &cobra.Command{
	Use:     "remove [repository]",
	Aliases: []string{"rm"},
	Short:   "Remove a skill repository (alias: rm)",
	Long: `Remove a skill repository from Skulto.

This will:
  - Uninstall all skills from the repository (remove symlinks)
  - Delete skill records from the database
  - Remove the repository from the database
  - Delete the local git clone

Supports multiple formats:
  - owner/repo                    (short format)
  - https://github.com/owner/repo
  - https://github.com/owner/repo.git
  - git@github.com:owner/repo.git

If no repository is specified, an interactive selection dialog will be shown.

Examples:
  # Remove using short format
  skulto remove asteroid-belt/skills

  # Remove using full URL
  skulto remove https://github.com/asteroid-belt/skills

  # Remove with force (skip confirmation)
  skulto remove asteroid-belt/skills --force

  # Interactive selection
  skulto remove`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRemove,
}

func init() {
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Skip confirmation prompt")
}

func runRemove(cmd *cobra.Command, args []string) error {
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

	var source *models.Source

	if len(args) == 0 {
		// Interactive selection mode
		source, err = selectRepositoryInteractive(database)
		if err != nil {
			return err
		}
		if source == nil {
			fmt.Println("No repository selected. Aborting.")
			return nil
		}
	} else {
		// Parse repository URL
		repoURL := args[0]
		fmt.Printf("Parsing repository URL: %s\n", repoURL)

		parsed, err := scraper.ParseRepositoryURL(repoURL)
		if err != nil {
			return fmt.Errorf("invalid repository URL: %w", err)
		}

		source, err = database.GetSource(parsed.ID)
		if err != nil {
			return fmt.Errorf("failed to check source: %w", err)
		}
		if source == nil {
			return fmt.Errorf("repository %s not found in database", parsed.ID)
		}
	}

	// Show confirmation unless --force is set
	if !removeForce {
		confirmed := showRemoveConfirmation(source)
		if !confirmed {
			fmt.Println("Aborting.")
			return nil
		}
	}

	// Execute removal
	return executeRemoval(ctx, cfg, database, source)
}

// executeRemoval performs the actual repository removal.
func executeRemoval(ctx context.Context, cfg *config.Config, database *db.DB, source *models.Source) error {
	fmt.Printf("\nRemoving repository: %s\n", source.ID)

	// Step 1: Get all skills from this repository
	fmt.Println("\n[1/4] Finding skills from repository...")
	skills, err := database.GetSkillsBySourceID(source.ID)
	if err != nil {
		return fmt.Errorf("failed to get skills: %w", err)
	}
	fmt.Printf("      Found %d skills\n", len(skills))

	// Step 2: Uninstall all skills (remove symlinks)
	if len(skills) > 0 {
		fmt.Println("\n[2/4] Uninstalling skills...")
		inst := installer.New(database, cfg)

		var uninstallErrors []error
		for _, skill := range skills {
			if skill.IsInstalled {
				if err := inst.UninstallAll(ctx, &skill); err != nil {
					uninstallErrors = append(uninstallErrors, fmt.Errorf("%s: %w", skill.Slug, err))
				} else {
					fmt.Printf("      Uninstalled: %s\n", skill.Slug)
				}
			}
		}

		if len(uninstallErrors) > 0 {
			fmt.Printf("      Warning: %d skills had uninstall errors\n", len(uninstallErrors))
			for _, e := range uninstallErrors {
				fmt.Printf("        - %v\n", e)
			}
		}
	} else {
		fmt.Println("\n[2/4] No skills to uninstall")
	}

	// Step 3: Delete skills and source from database
	fmt.Println("\n[3/4] Removing from database...")
	count, err := database.HardDeleteSkillsBySource(source.ID)
	if err != nil {
		return fmt.Errorf("failed to delete skills: %w", err)
	}
	fmt.Printf("      Deleted %d skill records\n", count)

	if err := database.HardDeleteSource(source.ID); err != nil {
		return fmt.Errorf("failed to delete source: %w", err)
	}
	fmt.Println("      Deleted source record")

	// Step 4: Remove git clone from disk
	fmt.Println("\n[4/4] Removing local git clone...")
	paths := config.GetPaths(cfg)
	repoManager := scraper.NewRepositoryManager(paths.Repositories, cfg.GitHub.Token)
	if err := repoManager.RemoveRepository(source.Owner, source.Repo); err != nil {
		fmt.Printf("      Warning: failed to remove git clone: %v\n", err)
	} else {
		fmt.Println("      Removed git clone")
	}

	fmt.Printf("\nRepository %s removed successfully!\n", source.ID)

	// Track telemetry event
	telemetryClient.TrackRepoRemoved(source.ID, int(source.SkillCount))

	return nil
}

// showRemoveConfirmation displays a confirmation prompt and returns true if confirmed.
func showRemoveConfirmation(source *models.Source) bool {
	fmt.Printf("\nYou are about to remove repository: %s\n", source.ID)
	fmt.Printf("  Skills: %d\n", source.SkillCount)
	if source.LastScrapedAt != nil {
		fmt.Printf("  Last synced: %s\n", source.LastScrapedAt.Format(time.RFC3339))
	}
	fmt.Println("\nThis will:")
	fmt.Println("  - Uninstall all skills from this repository")
	fmt.Println("  - Delete all skill records from the database")
	fmt.Println("  - Remove the repository from the database")
	fmt.Println("  - Delete the local git clone")
	fmt.Println("\nThis action cannot be undone.")
	fmt.Print("\nAre you sure? [y/N]: ")

	var response string
	_, _ = fmt.Scanln(&response)

	return response == "y" || response == "Y" || response == "yes" || response == "Yes"
}

// repoSelectModel wraps the dialog for standalone TUI execution
type repoSelectModel struct {
	dialog *components.RepoSelectDialog
	width  int
	height int
}

func (m repoSelectModel) Init() tea.Cmd {
	return nil
}

func (m repoSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.dialog.SetWidth(msg.Width - 4)
		return m, nil

	case tea.KeyMsg:
		m.dialog.Update(msg)
		if m.dialog.IsConfirmed() || m.dialog.IsCancelled() {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m repoSelectModel) View() string {
	return m.dialog.OverlayView("", m.width, m.height)
}

// selectRepositoryInteractive shows an interactive dialog to select a repository.
func selectRepositoryInteractive(database *db.DB) (*models.Source, error) {
	// Get list of sources
	sources, err := database.ListSources()
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	if len(sources) == 0 {
		fmt.Println("No repositories found.")
		return nil, nil
	}

	// Build options with installed/not-installed counts
	options := make([]components.RepoOption, len(sources))
	for i := range sources {
		// Get skills for this source to count installed/not-installed
		skills, err := database.GetSkillsBySourceID(sources[i].ID)
		if err != nil {
			// Fall back to just total count on error
			options[i] = components.RepoOption{
				Source:      &sources[i],
				Title:       sources[i].ID,
				Description: fmt.Sprintf("%d skills", sources[i].SkillCount),
			}
			continue
		}

		var installedCount, notInstalledCount int
		for _, skill := range skills {
			if skill.IsInstalled {
				installedCount++
			} else {
				notInstalledCount++
			}
		}

		options[i] = components.RepoOption{
			Source:            &sources[i],
			Title:             sources[i].ID,
			InstalledCount:    installedCount,
			NotInstalledCount: notInstalledCount,
		}
	}

	// Create dialog with options
	dialog := components.NewRepoSelectDialogWithOptions(options)

	// Run TUI
	model := repoSelectModel{
		dialog: dialog,
		width:  80,
		height: 24,
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("TUI error: %w", err)
	}

	m := finalModel.(repoSelectModel)
	if m.dialog.IsCancelled() {
		return nil, nil
	}

	return m.dialog.GetSelection(), nil
}
