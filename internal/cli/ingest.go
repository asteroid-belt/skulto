package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
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

	if ingestAll {
		return runBulkIngest(database)
	}

	return runSingleIngest(database, skillName)
}

// runBulkIngest handles the --all flag case.
func runBulkIngest(database *db.DB) error {
	var skills []struct {
		Name  string
		Scope string
		Path  string
	}

	// Get discovered skills filtered by scope if specified
	if ingestProjectOnly {
		discovered, err := database.ListDiscoveredSkillsByScope("project")
		if err != nil {
			return trackCLIError("ingest", fmt.Errorf("list discovered skills: %w", err))
		}
		for _, d := range discovered {
			skills = append(skills, struct {
				Name  string
				Scope string
				Path  string
			}{d.Name, d.Scope, d.Path})
		}
	} else if ingestGlobalOnly {
		discovered, err := database.ListDiscoveredSkillsByScope("global")
		if err != nil {
			return trackCLIError("ingest", fmt.Errorf("list discovered skills: %w", err))
		}
		for _, d := range discovered {
			skills = append(skills, struct {
				Name  string
				Scope string
				Path  string
			}{d.Name, d.Scope, d.Path})
		}
	} else {
		discovered, err := database.ListDiscoveredSkills()
		if err != nil {
			return trackCLIError("ingest", fmt.Errorf("list discovered skills: %w", err))
		}
		for _, d := range discovered {
			skills = append(skills, struct {
				Name  string
				Scope string
				Path  string
			}{d.Name, d.Scope, d.Path})
		}
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

	// Placeholder: IngestionService will be implemented in Phase 4
	fmt.Println("Ingestion service not yet implemented.")
	fmt.Println("The following operations would be performed:")
	for _, s := range skills {
		fmt.Printf("  1. Copy %s to .skulto/skills/%s/\n", s.Path, s.Name)
		fmt.Printf("  2. Create symlink at original location pointing to skulto storage\n")
	}

	return nil
}

// runSingleIngest handles ingestion of a single skill by name.
func runSingleIngest(database *db.DB, skillName string) error {
	// Determine scope to search
	var scope string
	if ingestProjectOnly {
		scope = "project"
	} else if ingestGlobalOnly {
		scope = "global"
	}

	var skill *struct {
		Name  string
		Scope string
		Path  string
	}

	// Find the discovered skill
	if scope != "" {
		discovered, err := database.GetDiscoveredSkillByName(skillName, scope)
		if err != nil {
			return trackCLIError("ingest", fmt.Errorf("skill '%s' not found in %s scope", skillName, scope))
		}
		skill = &struct {
			Name  string
			Scope string
			Path  string
		}{discovered.Name, discovered.Scope, discovered.Path}
	} else {
		// Try project scope first, then global
		discovered, err := database.GetDiscoveredSkillByName(skillName, "project")
		if err != nil {
			discovered, err = database.GetDiscoveredSkillByName(skillName, "global")
			if err != nil {
				return trackCLIError("ingest", fmt.Errorf("skill '%s' not found. Run 'skulto discover' to scan for unmanaged skills", skillName))
			}
		}
		skill = &struct {
			Name  string
			Scope string
			Path  string
		}{discovered.Name, discovered.Scope, discovered.Path}
	}

	fmt.Printf("Found discovered skill: %s (%s scope)\n", skill.Name, skill.Scope)
	fmt.Printf("  Path: %s\n\n", skill.Path)

	// TODO: Check for conflicts with existing skulto-managed skills
	// This would require checking if .skulto/skills/<name> already exists

	// Placeholder: IngestionService will be implemented in Phase 4
	fmt.Println("Ingestion service not yet implemented.")
	fmt.Println("The following operations would be performed:")
	fmt.Printf("  1. Copy %s to .skulto/skills/%s/\n", skill.Path, skill.Name)
	fmt.Printf("  2. Create symlink at original location pointing to skulto storage\n")

	return nil
}

// promptConflictResolution prompts the user to resolve a naming conflict.
// nolint:unused // Stub for Phase 4 IngestionService implementation
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
