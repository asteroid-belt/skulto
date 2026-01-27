package prompts

import (
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/charmbracelet/huh"
)

// BuildScopeOptions creates huh options for installation scopes.
func BuildScopeOptions() []huh.Option[string] {
	return []huh.Option[string]{
		huh.NewOption("Global (~/.claude/skills/) - Available everywhere", string(installer.ScopeGlobal)),
		huh.NewOption("Project (./.claude/skills/) - Current directory only", string(installer.ScopeProject)),
	}
}

// ParseScopeStrings converts string slice to InstallScope slice.
// Invalid scope strings are ignored.
func ParseScopeStrings(scopeStrs []string) []installer.InstallScope {
	scopes := make([]installer.InstallScope, 0, len(scopeStrs))
	for _, s := range scopeStrs {
		scope := installer.InstallScope(s)
		if scope == installer.ScopeGlobal || scope == installer.ScopeProject {
			scopes = append(scopes, scope)
		}
	}
	return scopes
}

// RunScopeSelector shows interactive scope selection.
// Returns selected scope strings.
func RunScopeSelector(preselected []string) ([]string, error) {
	options := BuildScopeOptions()

	// Default to global if no preselection
	if len(preselected) == 0 {
		preselected = []string{string(installer.ScopeGlobal)}
	}

	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select installation scope").
				Description("Space to toggle, Enter to confirm").
				Options(options...).
				Value(&selected),
		),
	)

	// Set initial selection
	selected = preselected

	if err := form.Run(); err != nil {
		return nil, err
	}

	return selected, nil
}
