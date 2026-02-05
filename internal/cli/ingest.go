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
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/spf13/cobra"
)

var (
	ingestAll         bool
	ingestProjectOnly bool
	ingestGlobalOnly  bool
)

// ingestCmd is the ingest command instance registered with root.
var ingestCmd = newIngestCmd()

// newIngestCmd creates a new ingest command.
// This is a factory function to support testing.
func newIngestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingest [skill-name]",
		Short: "Import a discovered skill into skulto management",
		Long: `Import a discovered (unmanaged) skill into skulto management.

The skill will be copied to .skulto/skills/ and a symlink will be
created at the original location.

Use 'skulto discover' to list available discovered skills.

Flags:
  --all       Import all discovered skills
  --project   Import only from project scope
  --global    Import only from global scope

If neither --project nor --global is specified, both scopes are considered.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runIngest,
	}

	cmd.Flags().BoolVar(&ingestAll, "all", false, "Import all discovered skills")
	cmd.Flags().BoolVar(&ingestProjectOnly, "project", false, "Import only from project scope")
	cmd.Flags().BoolVar(&ingestGlobalOnly, "global", false, "Import only from global scope")

	return cmd
}

func runIngest(cmd *cobra.Command, args []string) error {
	var skillName string
	if len(args) > 0 {
		skillName = args[0]
	}

	// Validate: need either skillName or --all
	if skillName == "" && !ingestAll {
		return trackCLIError("ingest", fmt.Errorf("skill name required (or use --all flag)"))
	}

	// Validate: cannot specify both --project and --global
	if ingestProjectOnly && ingestGlobalOnly {
		return trackCLIError("ingest", fmt.Errorf("cannot specify both --project and --global"))
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return trackCLIError("ingest", fmt.Errorf("load config: %w", err))
	}

	// Open database
	paths := config.GetPaths(cfg)
	database, err := db.New(db.DefaultConfig(paths.Database))
	if err != nil {
		return trackCLIError("ingest", fmt.Errorf("initialize database: %w", err))
	}
	defer func() { _ = database.Close() }()

	// Create ingestion service
	ingestionSvc := discovery.NewIngestionService(database, cfg)

	if ingestAll {
		return runBulkIngest(database, cfg, ingestionSvc)
	}

	return runSingleIngest(database, cfg, ingestionSvc, skillName)
}

// runBulkIngest handles the --all flag case.
func runBulkIngest(database *db.DB, cfg *config.Config, ingestionSvc *discovery.IngestionService) error {
	var skills []models.DiscoveredSkill

	// Get discovered skills filtered by scope if specified
	if ingestProjectOnly {
		discovered, err := database.ListDiscoveredSkillsByScope("project")
		if err != nil {
			return trackCLIError("ingest", fmt.Errorf("list discovered skills: %w", err))
		}
		skills = discovered
	} else if ingestGlobalOnly {
		discovered, err := database.ListDiscoveredSkillsByScope("global")
		if err != nil {
			return trackCLIError("ingest", fmt.Errorf("list discovered skills: %w", err))
		}
		skills = discovered
	} else {
		discovered, err := database.ListDiscoveredSkills()
		if err != nil {
			return trackCLIError("ingest", fmt.Errorf("list discovered skills: %w", err))
		}
		skills = discovered
	}

	if len(skills) == 0 {
		fmt.Println("No discovered skills to import.")
		fmt.Println("Run 'skulto discover' to scan for unmanaged skills.")
		return nil
	}

	fmt.Printf("Found %d discovered skill(s) to import:\n\n", len(skills))
	for _, s := range skills {
		fmt.Printf("  - %s (%s)\n", s.Name, s.Scope)
	}
	fmt.Println()

	ctx := context.Background()
	var successCount, skipCount, errorCount int

	for i := range skills {
		skill := &skills[i]

		// Check for name conflicts
		hasConflict, err := ingestionSvc.CheckNameConflict(skill.Name, skill.Scope)
		if err != nil {
			fmt.Printf("  Error checking conflict for %s: %v\n", skill.Name, err)
			errorCount++
			continue
		}

		if hasConflict {
			action, newName, err := promptConflictResolution(skill.Name)
			if err != nil {
				fmt.Printf("  Error reading input for %s: %v\n", skill.Name, err)
				errorCount++
				continue
			}

			switch action {
			case "skip":
				fmt.Printf("  Skipped: %s\n", skill.Name)
				skipCount++
				continue
			case "rename":
				skill.Name = newName
			case "replace":
				// Remove existing and continue with ingestion
				paths := config.GetPaths(cfg)
				destPath := paths.Skills
				if skill.Scope == "project" {
					cwd, _ := os.Getwd()
					destPath = cwd + "/.skulto/skills"
				}
				_ = os.RemoveAll(destPath + "/" + skill.Name)
			}
		}

		// Perform ingestion
		result, err := ingestionSvc.IngestSkill(ctx, skill)
		if err != nil {
			fmt.Printf("  Error ingesting %s: %v\n", skill.Name, err)
			errorCount++
			continue
		}

		fmt.Printf("  Imported: %s -> %s\n", result.Name, result.DestPath)
		telemetryClient.TrackSkillIngested(skill.Name, skill.Scope)
		successCount++
	}

	fmt.Printf("\nImport complete: %d imported, %d skipped, %d errors\n", successCount, skipCount, errorCount)
	return nil
}

// runSingleIngest handles ingestion of a single skill by name.
func runSingleIngest(database *db.DB, cfg *config.Config, ingestionSvc *discovery.IngestionService, skillName string) error {
	// Determine scope to search
	var scope string
	if ingestProjectOnly {
		scope = "project"
	} else if ingestGlobalOnly {
		scope = "global"
	}

	var skill *models.DiscoveredSkill

	// Find the discovered skill
	if scope != "" {
		discovered, err := database.GetDiscoveredSkillByName(skillName, scope)
		if err != nil {
			return trackCLIError("ingest", fmt.Errorf("skill '%s' not found in %s scope", skillName, scope))
		}
		skill = discovered
	} else {
		// Try project scope first, then global
		discovered, err := database.GetDiscoveredSkillByName(skillName, "project")
		if err != nil {
			discovered, err = database.GetDiscoveredSkillByName(skillName, "global")
			if err != nil {
				return trackCLIError("ingest", fmt.Errorf("skill '%s' not found. Run 'skulto discover' to scan for unmanaged skills", skillName))
			}
		}
		skill = discovered
	}

	fmt.Printf("Found discovered skill: %s (%s scope)\n", skill.Name, skill.Scope)
	fmt.Printf("  Path: %s\n\n", skill.Path)

	// Check for name conflicts
	hasConflict, err := ingestionSvc.CheckNameConflict(skill.Name, skill.Scope)
	if err != nil {
		return trackCLIError("ingest", fmt.Errorf("check conflict: %w", err))
	}

	if hasConflict {
		action, newName, err := promptConflictResolution(skill.Name)
		if err != nil {
			return trackCLIError("ingest", fmt.Errorf("read input: %w", err))
		}

		switch action {
		case "skip":
			fmt.Printf("Skipped: %s\n", skill.Name)
			return nil
		case "rename":
			skill.Name = newName
		case "replace":
			// Remove existing and continue with ingestion
			paths := config.GetPaths(cfg)
			destPath := paths.Skills
			if skill.Scope == "project" {
				cwd, _ := os.Getwd()
				destPath = cwd + "/.skulto/skills"
			}
			_ = os.RemoveAll(destPath + "/" + skill.Name)
		}
	}

	// Perform ingestion
	ctx := context.Background()
	result, err := ingestionSvc.IngestSkill(ctx, skill)
	if err != nil {
		return trackCLIError("ingest", fmt.Errorf("ingest skill: %w", err))
	}

	telemetryClient.TrackSkillIngested(skill.Name, skill.Scope)
	fmt.Printf("Imported: %s -> %s\n", result.Name, result.DestPath)
	fmt.Println("Original location now points to skulto-managed skill via symlink.")
	return nil
}

// promptConflictResolution prompts the user to resolve a naming conflict.
func promptConflictResolution(name string) (action, newName string, err error) {
	fmt.Printf("Skill '%s' already exists in .skulto/skills/\n", name)
	fmt.Print("  [r]ename  [s]kip  [R]eplace: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}

	input = strings.TrimSpace(strings.ToLower(input))

	switch input {
	case "r", "rename":
		fmt.Print("Enter new name: ")
		newName, err = reader.ReadString('\n')
		if err != nil {
			return "", "", err
		}
		return "rename", strings.TrimSpace(newName), nil
	case "s", "skip":
		return "skip", "", nil
	case "replace":
		return "replace", "", nil
	default:
		return "skip", "", nil
	}
}
