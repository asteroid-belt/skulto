package cli

import (
	"fmt"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/favorites"
	"github.com/spf13/cobra"
)

var favoritesCmd = &cobra.Command{
	Use:   "favorites",
	Short: "Manage favorite skills",
	Long: `Manage your favorite skills.

Favorites persist across database resets and are stored in ~/.skulto/favorites.json.

Subcommands:
  add <slug>     Add a skill to favorites
  remove <slug>  Remove a skill from favorites
  list           List all favorite skills`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var favoritesAddCmd = &cobra.Command{
	Use:   "add <slug>",
	Short: "Add a skill to favorites",
	Long:  `Add a skill to your favorites by its slug.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runFavoritesAdd,
}

var favoritesRemoveCmd = &cobra.Command{
	Use:   "remove <slug>",
	Short: "Remove a skill from favorites",
	Long:  `Remove a skill from your favorites by its slug.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runFavoritesRemove,
}

var favoritesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all favorite skills",
	Long:  `List all skills you have marked as favorites.`,
	Args:  cobra.NoArgs,
	RunE:  runFavoritesList,
}

func init() {
	favoritesCmd.AddCommand(favoritesAddCmd)
	favoritesCmd.AddCommand(favoritesRemoveCmd)
	favoritesCmd.AddCommand(favoritesListCmd)
}

func runFavoritesAdd(cmd *cobra.Command, args []string) error {
	slug := args[0]

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return trackCLIError("favorites add", fmt.Errorf("load config: %w", err))
	}

	paths := config.GetPaths(cfg)

	// Initialize database to verify skill exists
	database, err := db.New(db.DefaultConfig(paths.Database))
	if err != nil {
		return trackCLIError("favorites add", fmt.Errorf("initialize database: %w", err))
	}
	defer func() { _ = database.Close() }()

	// Verify skill exists
	skill, err := database.GetSkillBySlug(slug)
	if err != nil {
		return trackCLIError("favorites add", fmt.Errorf("lookup skill: %w", err))
	}
	if skill == nil {
		return trackCLIError("favorites add", fmt.Errorf("skill not found: %s", slug))
	}

	// Initialize favorites store
	store := favorites.NewStore(paths.Favorites)
	if err := store.Load(); err != nil {
		return trackCLIError("favorites add", fmt.Errorf("load favorites: %w", err))
	}

	// Check if already a favorite
	if store.IsFavorite(slug) {
		fmt.Printf("'%s' is already a favorite.\n", skill.Title)
		return nil
	}

	// Add to favorites
	if err := store.Add(slug); err != nil {
		return trackCLIError("favorites add", fmt.Errorf("add favorite: %w", err))
	}

	telemetryClient.TrackFavoriteAdded(slug)
	fmt.Printf("Added '%s' to favorites.\n", skill.Title)
	return nil
}

func runFavoritesRemove(cmd *cobra.Command, args []string) error {
	slug := args[0]

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return trackCLIError("favorites remove", fmt.Errorf("load config: %w", err))
	}

	paths := config.GetPaths(cfg)

	// Initialize favorites store
	store := favorites.NewStore(paths.Favorites)
	if err := store.Load(); err != nil {
		return trackCLIError("favorites remove", fmt.Errorf("load favorites: %w", err))
	}

	// Check if it's a favorite
	if !store.IsFavorite(slug) {
		fmt.Printf("'%s' is not in your favorites.\n", slug)
		return nil
	}

	// Remove from favorites
	if err := store.Remove(slug); err != nil {
		return trackCLIError("favorites remove", fmt.Errorf("remove favorite: %w", err))
	}

	telemetryClient.TrackFavoriteRemoved(slug)
	fmt.Printf("Removed '%s' from favorites.\n", slug)
	return nil
}

func runFavoritesList(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return trackCLIError("favorites list", fmt.Errorf("load config: %w", err))
	}

	paths := config.GetPaths(cfg)

	// Initialize favorites store
	store := favorites.NewStore(paths.Favorites)
	if err := store.Load(); err != nil {
		return trackCLIError("favorites list", fmt.Errorf("load favorites: %w", err))
	}

	favs := store.List()
	telemetryClient.TrackFavoritesListed(len(favs))

	if len(favs) == 0 {
		fmt.Println("No favorites yet.")
		fmt.Println("\nUse 'skulto favorites add <slug>' to add a skill to favorites.")
		return nil
	}

	// Initialize database to get skill titles
	database, err := db.New(db.DefaultConfig(paths.Database))
	if err != nil {
		// Fallback: show slugs only
		fmt.Printf("FAVORITES (%d skills)\n", len(favs))
		fmt.Println("──────────────────────────────────────────────────")
		for _, fav := range favs {
			fmt.Printf("  %s (added %s)\n", fav.Slug, formatTimeSince(fav.AddedAt))
		}
		return nil
	}
	defer func() { _ = database.Close() }()

	fmt.Printf("FAVORITES (%d skills)\n", len(favs))
	fmt.Println("──────────────────────────────────────────────────")

	for _, fav := range favs {
		skill, err := database.GetSkillBySlug(fav.Slug)
		if err != nil || skill == nil {
			// Skill might have been deleted from DB
			fmt.Printf("  %s (not in database)\n", fav.Slug)
			continue
		}

		installedIndicator := ""
		if skill.IsInstalled {
			installedIndicator = " [installed]"
		}

		fmt.Printf("  %s%s\n", skill.Title, installedIndicator)
		fmt.Printf("    slug: %s | added: %s\n", fav.Slug, formatTimeSince(fav.AddedAt))
	}

	return nil
}
