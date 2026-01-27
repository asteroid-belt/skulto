// Package prompts provides interactive CLI prompt components using charmbracelet/huh.
package prompts

import (
	"fmt"

	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/charmbracelet/huh"
)

// BuildPlatformOptions creates huh options from detected platforms.
// Detected platforms are marked with a checkmark indicator.
func BuildPlatformOptions(platforms []installer.DetectedPlatform) []huh.Option[string] {
	options := make([]huh.Option[string], 0, len(platforms))
	for _, p := range platforms {
		var label string
		if p.Detected {
			label = fmt.Sprintf("%s (%s) ✓ detected", p.Name, p.Path)
		} else {
			label = fmt.Sprintf("%s (%s)", p.Name, p.Path)
		}
		options = append(options, huh.NewOption(label, p.ID))
	}
	return options
}

// FilterSelectedPlatforms returns only platforms matching selected IDs.
func FilterSelectedPlatforms(all []installer.DetectedPlatform, selected []string) []installer.DetectedPlatform {
	selectedMap := make(map[string]bool, len(selected))
	for _, s := range selected {
		selectedMap[s] = true
	}

	result := make([]installer.DetectedPlatform, 0, len(selected))
	for _, p := range all {
		if selectedMap[p.ID] {
			result = append(result, p)
		}
	}
	return result
}

// GetDefaultSelectedPlatforms returns IDs of all detected platforms.
func GetDefaultSelectedPlatforms(platforms []installer.DetectedPlatform) []string {
	var defaults []string
	for _, p := range platforms {
		if p.Detected {
			defaults = append(defaults, p.ID)
		}
	}
	return defaults
}

// RunPlatformSelector shows interactive platform selection.
// Returns selected platform IDs.
func RunPlatformSelector(platforms []installer.DetectedPlatform, preselected []string) ([]string, error) {
	options := BuildPlatformOptions(platforms)

	// Use detected platforms as defaults if no preselection
	if len(preselected) == 0 {
		preselected = GetDefaultSelectedPlatforms(platforms)
	}

	// Initialize selected with preselected values BEFORE creating form
	// (form binds to &selected, so we must not reassign after binding)
	selected := make([]string, len(preselected))
	copy(selected, preselected)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select platforms to install to").
				Description("Space to toggle, Enter to confirm").
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return selected, nil
}
