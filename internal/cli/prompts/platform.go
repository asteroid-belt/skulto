// Package prompts provides interactive CLI prompt components using charmbracelet/huh.
package prompts

import (
	"fmt"

	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/charmbracelet/huh"
)

// PlatformSelectorResult contains the result of platform selection.
type PlatformSelectorResult struct {
	Selected           []string // Platform IDs selected for installation
	AlreadyInstalled   []string // Platform IDs that were already installed
	AllAlreadyInstalled bool     // True if all platforms are already installed
}

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

// BuildSelectablePlatformOptions creates huh options excluding already-installed platforms.
// Returns options for platforms that can be selected and a list of installed platform info.
func BuildSelectablePlatformOptions(platforms []installer.DetectedPlatform, installedLocations []installer.InstallLocation) ([]huh.Option[string], []installer.DetectedPlatform) {
	// Build map of installed platform IDs
	installedMap := make(map[string]bool)
	for _, loc := range installedLocations {
		installedMap[string(loc.Platform)] = true
	}

	var selectableOptions []huh.Option[string]
	var installedPlatforms []installer.DetectedPlatform

	for _, p := range platforms {
		if installedMap[p.ID] {
			installedPlatforms = append(installedPlatforms, p)
		} else {
			var label string
			if p.Detected {
				label = fmt.Sprintf("%s (%s) ✓ detected", p.Name, p.Path)
			} else {
				label = fmt.Sprintf("%s (%s)", p.Name, p.Path)
			}
			selectableOptions = append(selectableOptions, huh.NewOption(label, p.ID))
		}
	}

	return selectableOptions, installedPlatforms
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

// GetDefaultSelectablePlatforms returns IDs of detected platforms that are not already installed.
func GetDefaultSelectablePlatforms(platforms []installer.DetectedPlatform, installedLocations []installer.InstallLocation) []string {
	// Build map of installed platform IDs
	installedMap := make(map[string]bool)
	for _, loc := range installedLocations {
		installedMap[string(loc.Platform)] = true
	}

	var defaults []string
	for _, p := range platforms {
		if p.Detected && !installedMap[p.ID] {
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

// RunPlatformSelectorWithInstalled shows interactive platform selection,
// accounting for already-installed locations.
// Prints informational text about installed platforms and only allows
// selection of platforms where the skill is not yet installed.
func RunPlatformSelectorWithInstalled(platforms []installer.DetectedPlatform, installedLocations []installer.InstallLocation, preselected []string) (*PlatformSelectorResult, error) {
	selectableOptions, installedPlatforms := BuildSelectablePlatformOptions(platforms, installedLocations)

	result := &PlatformSelectorResult{
		AlreadyInstalled: make([]string, 0, len(installedPlatforms)),
	}

	// Collect installed platform IDs
	for _, p := range installedPlatforms {
		result.AlreadyInstalled = append(result.AlreadyInstalled, p.ID)
	}

	// If all platforms are already installed, return early
	if len(selectableOptions) == 0 {
		result.AllAlreadyInstalled = true
		return result, nil
	}

	// Print installed locations info
	if len(installedPlatforms) > 0 {
		fmt.Println("Already installed:")
		for _, p := range installedPlatforms {
			fmt.Printf("  • %s (%s)\n", p.Name, p.Path)
		}
		fmt.Println()
	}

	// Use detected platforms (excluding installed) as defaults if no preselection
	if len(preselected) == 0 {
		preselected = GetDefaultSelectablePlatforms(platforms, installedLocations)
	} else {
		// Filter preselected to only include selectable platforms
		installedMap := make(map[string]bool)
		for _, p := range installedPlatforms {
			installedMap[p.ID] = true
		}
		var filteredPreselected []string
		for _, id := range preselected {
			if !installedMap[id] {
				filteredPreselected = append(filteredPreselected, id)
			}
		}
		preselected = filteredPreselected
	}

	// Initialize selected with preselected values BEFORE creating form
	selected := make([]string, len(preselected))
	copy(selected, preselected)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select platforms to install to").
				Description("Space to toggle, Enter to confirm").
				Options(selectableOptions...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	result.Selected = selected
	return result, nil
}
