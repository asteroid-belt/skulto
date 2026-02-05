package cli

import (
	"fmt"
	"strings"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <skill-slug>",
	Short: "Show detailed information about a skill",
	Long: `Display detailed information about a specific skill.

The skill can be identified by its slug (e.g., 'commit-message-generator').`,
	Args: cobra.ExactArgs(1),
	RunE: runInfo,
}

func runInfo(cmd *cobra.Command, args []string) error {
	slug := args[0]

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return trackCLIError("info", fmt.Errorf("load config: %w", err))
	}

	// Initialize database
	paths := config.GetPaths(cfg)
	database, err := db.New(db.DefaultConfig(paths.Database))
	if err != nil {
		return trackCLIError("info", fmt.Errorf("initialize database: %w", err))
	}
	defer func() { _ = database.Close() }()

	// Find skill by slug
	skill, err := database.GetSkillBySlug(slug)
	if err != nil {
		return trackCLIError("info", fmt.Errorf("skill not found: %w", err))
	}

	if skill == nil {
		return trackCLIError("info", fmt.Errorf("skill '%s' not found", slug))
	}

	// Track the info view
	telemetryClient.TrackSkillViewed(skill.Slug, skill.Category, skill.IsLocal)

	// Display skill information
	fmt.Printf("Skill: %s\n", skill.Title)
	fmt.Printf("Slug: %s\n", skill.Slug)
	fmt.Printf("Category: %s\n", skill.Category)
	fmt.Printf("Local: %v\n", skill.IsLocal)

	if skill.Description != "" {
		fmt.Printf("\nDescription:\n  %s\n", skill.Description)
	}

	if len(skill.Tags) > 0 {
		tagNames := make([]string, len(skill.Tags))
		for i, tag := range skill.Tags {
			tagNames[i] = tag.Name
		}
		fmt.Printf("\nTags: %s\n", strings.Join(tagNames, ", "))
	}

	if skill.Source != nil {
		fmt.Printf("\nSource: %s/%s\n", skill.Source.Owner, skill.Source.Repo)
	}

	// Use skill_installations as source of truth for installed status
	hasInstallations, _ := database.HasInstallations(skill.ID)
	fmt.Printf("\nInstalled: %v\n", hasInstallations)

	return nil
}
