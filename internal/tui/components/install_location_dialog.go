package components

import (
	"github.com/asteroid-belt/skulto/internal/detect"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// LocationOption represents a single installation location option.
type LocationOption struct {
	Location    installer.InstallLocation
	DisplayName string
	Description string
	Selected    bool
}

// InstallLocationDialog presents installation location choices with multi-select.
type InstallLocationDialog struct {
	options           []LocationOption
	currentIndex      int
	width             int
	cancelled         bool
	confirmed         bool
	platforms         []installer.Platform // User's selected platforms
	rememberLocations bool                 // If true, cache these locations for future installs
	onRememberOption  bool                 // True when cursor is on the "remember" checkbox
}

// NewInstallLocationDialog creates a new install location dialog.
func NewInstallLocationDialog(platforms []installer.Platform) *InstallLocationDialog {
	dialog := &InstallLocationDialog{
		platforms: platforms,
		options:   make([]LocationOption, 0),
	}
	dialog.buildOptions()
	return dialog
}

// buildOptions creates location options for each platform × scope combination.
func (d *InstallLocationDialog) buildOptions() {
	d.options = make([]LocationOption, 0, len(d.platforms)*2)

	for _, platform := range d.platforms {
		info := platform.Info()

		// Global option
		globalLoc, err := installer.NewInstallLocation(platform, installer.ScopeGlobal)
		if err == nil {
			d.options = append(d.options, LocationOption{
				Location:    globalLoc,
				DisplayName: info.Name + " - Global",
				Description: "~/" + info.SkillsPath + "/",
				Selected:    true, // Default to global selected
			})
		}

		// Project option
		projectLoc, err := installer.NewInstallLocation(platform, installer.ScopeProject)
		if err == nil {
			d.options = append(d.options, LocationOption{
				Location:    projectLoc,
				DisplayName: info.Name + " - Project",
				Description: "./" + info.SkillsPath + "/",
				Selected:    false, // Default to project not selected
			})
		}
	}
}

// SetWidth sets the dialog width.
func (d *InstallLocationDialog) SetWidth(w int) {
	d.width = w
}

// Update handles keyboard input for the dialog.
func (d *InstallLocationDialog) Update(msg tea.KeyMsg) {
	// Total items = options + 1 (remember checkbox)
	totalItems := len(d.options) + 1

	switch msg.Type {
	case tea.KeyUp, tea.KeyShiftTab:
		d.currentIndex--
		if d.currentIndex < 0 {
			d.currentIndex = totalItems - 1
		}
		d.onRememberOption = d.currentIndex == len(d.options)
	case tea.KeyDown, tea.KeyTab:
		d.currentIndex++
		if d.currentIndex >= totalItems {
			d.currentIndex = 0
		}
		d.onRememberOption = d.currentIndex == len(d.options)
	case tea.KeySpace:
		// Toggle selection
		if d.onRememberOption {
			d.rememberLocations = !d.rememberLocations
		} else if d.currentIndex >= 0 && d.currentIndex < len(d.options) {
			d.options[d.currentIndex].Selected = !d.options[d.currentIndex].Selected
		}
	case tea.KeyEnter:
		if d.hasAnySelected() {
			d.confirmed = true
		}
	case tea.KeyEsc:
		d.cancelled = true
	default:
		if msg.Type == tea.KeyRunes {
			switch string(msg.Runes) {
			case "j":
				d.currentIndex++
				if d.currentIndex >= totalItems {
					d.currentIndex = 0
				}
				d.onRememberOption = d.currentIndex == len(d.options)
			case "k":
				d.currentIndex--
				if d.currentIndex < 0 {
					d.currentIndex = totalItems - 1
				}
				d.onRememberOption = d.currentIndex == len(d.options)
			case " ":
				if d.onRememberOption {
					d.rememberLocations = !d.rememberLocations
				} else if d.currentIndex >= 0 && d.currentIndex < len(d.options) {
					d.options[d.currentIndex].Selected = !d.options[d.currentIndex].Selected
				}
			case "a":
				// Select all
				for i := range d.options {
					d.options[i].Selected = true
				}
			case "n":
				// Select none (then user must select at least one)
				for i := range d.options {
					d.options[i].Selected = false
				}
			case "g":
				// Select all global locations
				for i := range d.options {
					d.options[i].Selected = d.options[i].Location.Scope == installer.ScopeGlobal
				}
			case "p":
				// Select all project locations
				for i := range d.options {
					d.options[i].Selected = d.options[i].Location.Scope == installer.ScopeProject
				}
			case "r":
				// Toggle remember locations
				d.rememberLocations = !d.rememberLocations
			}
		}
	}
}

// HandleKey processes string key for compatibility.
func (d *InstallLocationDialog) HandleKey(key string) {
	// Total items = options + 1 (remember checkbox)
	totalItems := len(d.options) + 1

	switch key {
	case "up", "k":
		d.currentIndex--
		if d.currentIndex < 0 {
			d.currentIndex = totalItems - 1
		}
		d.onRememberOption = d.currentIndex == len(d.options)
	case "down", "j":
		d.currentIndex++
		if d.currentIndex >= totalItems {
			d.currentIndex = 0
		}
		d.onRememberOption = d.currentIndex == len(d.options)
	case "space", " ":
		if d.onRememberOption {
			d.rememberLocations = !d.rememberLocations
		} else if d.currentIndex >= 0 && d.currentIndex < len(d.options) {
			d.options[d.currentIndex].Selected = !d.options[d.currentIndex].Selected
		}
	case "enter":
		if d.hasAnySelected() {
			d.confirmed = true
		}
	case "esc":
		d.cancelled = true
	case "r":
		d.rememberLocations = !d.rememberLocations
	}
}

// hasAnySelected returns true if at least one option is selected.
func (d *InstallLocationDialog) hasAnySelected() bool {
	for _, opt := range d.options {
		if opt.Selected {
			return true
		}
	}
	return false
}

// IsConfirmed returns true if the user confirmed their selection.
func (d *InstallLocationDialog) IsConfirmed() bool {
	return d.confirmed
}

// IsCancelled returns true if the user cancelled the dialog.
func (d *InstallLocationDialog) IsCancelled() bool {
	return d.cancelled
}

// GetSelectedLocations returns the selected installation locations.
func (d *InstallLocationDialog) GetSelectedLocations() []installer.InstallLocation {
	var locations []installer.InstallLocation
	for _, opt := range d.options {
		if opt.Selected {
			locations = append(locations, opt.Location)
		}
	}
	return locations
}

// ShouldRememberLocations returns true if user wants to cache locations for future installs.
func (d *InstallLocationDialog) ShouldRememberLocations() bool {
	return d.rememberLocations
}

// Reset clears the dialog state for reuse.
func (d *InstallLocationDialog) Reset() {
	d.currentIndex = 0
	d.cancelled = false
	d.confirmed = false
	d.onRememberOption = false
	d.rememberLocations = false
	// Reset selections to default (global selected, project not)
	for i := range d.options {
		d.options[i].Selected = d.options[i].Location.Scope == installer.ScopeGlobal
	}
}

// View renders the dialog.
func (d *InstallLocationDialog) View() string {
	dialogWidth := d.width
	if dialogWidth == 0 || dialogWidth > 65 {
		dialogWidth = 65
	}
	if dialogWidth < 50 {
		dialogWidth = 50
	}
	contentWidth := dialogWidth - 6

	// Colors from theme
	accentColor := theme.Current.Primary
	goldColor := theme.Current.Accent
	mutedColor := theme.Current.TextMuted
	textColor := theme.Current.Text
	successColor := theme.Current.Success
	selectedBgColor := theme.Current.Surface

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginBottom(1)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginBottom(1)

	title := titleStyle.Render("Install Location")
	subtitle := subtitleStyle.Render("Select where to install this skill (multi-select)")

	// Render options
	var optionViews []string

	for i, opt := range d.options {
		isCurrent := i == d.currentIndex

		// Checkbox
		var checkbox string
		if opt.Selected {
			checkbox = lipgloss.NewStyle().Foreground(successColor).Render("☑")
		} else {
			checkbox = lipgloss.NewStyle().Foreground(mutedColor).Render("☐")
		}

		// Option text
		nameStyle := lipgloss.NewStyle().Foreground(textColor)
		descStyle := lipgloss.NewStyle().Foreground(mutedColor).Italic(true)

		if isCurrent {
			nameStyle = nameStyle.Foreground(goldColor).Bold(true)
		}

		line := checkbox + " " + nameStyle.Render(opt.DisplayName)
		desc := "    " + descStyle.Render(opt.Description)

		optContent := lipgloss.JoinVertical(lipgloss.Left, line, desc)

		optStyle := lipgloss.NewStyle().
			Width(contentWidth).
			Padding(0, 1)

		if isCurrent {
			optStyle = optStyle.Background(selectedBgColor)
		}

		optionViews = append(optionViews, optStyle.Render(optContent))
	}

	options := lipgloss.JoinVertical(lipgloss.Left, optionViews...)

	// Remember locations checkbox
	var rememberCheckbox string
	if d.rememberLocations {
		rememberCheckbox = lipgloss.NewStyle().Foreground(successColor).Render("☑")
	} else {
		rememberCheckbox = lipgloss.NewStyle().Foreground(mutedColor).Render("☐")
	}

	rememberNameStyle := lipgloss.NewStyle().Foreground(textColor)
	rememberDescStyle := lipgloss.NewStyle().Foreground(mutedColor).Italic(true)

	if d.onRememberOption {
		rememberNameStyle = rememberNameStyle.Foreground(goldColor).Bold(true)
	}

	rememberLine := rememberCheckbox + " " + rememberNameStyle.Render("Remember these locations")
	rememberDesc := "    " + rememberDescStyle.Render("Skip this dialog for future installs")

	rememberContent := lipgloss.JoinVertical(lipgloss.Left, rememberLine, rememberDesc)

	rememberStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Padding(0, 1).
		MarginTop(1)

	if d.onRememberOption {
		rememberStyle = rememberStyle.Background(selectedBgColor)
	}

	rememberOption := rememberStyle.Render(rememberContent)

	// Validation message
	var validationMsg string
	if !d.hasAnySelected() {
		validationStyle := lipgloss.NewStyle().
			Foreground(theme.Current.Error).
			Italic(true).
			Width(contentWidth).
			Align(lipgloss.Center)
		validationMsg = validationStyle.Render("Select at least one location")
	}

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginTop(1).
		Italic(true)

	footerLine1 := "↑/↓: navigate  •  Space: toggle  •  Enter: confirm"
	footerLine2 := "a: all  •  n: none  •  g: global  •  p: project  •  r: remember"
	footer := footerStyle.Render(footerLine1 + "\n" + footerLine2)

	// Dialog container
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Width(dialogWidth)

	parts := []string{title, subtitle, "", options, rememberOption}
	if validationMsg != "" {
		parts = append(parts, "", validationMsg)
	}
	parts = append(parts, "", footer)

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)

	return dialogStyle.Render(content)
}

// CenteredView renders the dialog centered within the given dimensions.
func (d *InstallLocationDialog) CenteredView(width, height int) string {
	dialog := d.View()
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, dialog)
}

// NewInstallLocationDialogWithPrefs creates a dialog with merged preferences + detection.
// Saved agents and newly detected agents are pre-selected (global scope).
func NewInstallLocationDialogWithPrefs(platforms []installer.Platform, savedPrefs []string, detectionResults []detect.DetectionResult) *InstallLocationDialog {
	// Merge and reorder platforms
	merged := mergePreferencesWithDetected(platforms, savedPrefs, detectionResults)

	dialog := &InstallLocationDialog{
		platforms: merged,
		options:   make([]LocationOption, 0),
	}
	dialog.buildOptions()

	// Pre-select: saved and detected platforms get global selected
	savedSet := make(map[string]bool)
	for _, s := range savedPrefs {
		savedSet[s] = true
	}
	detectedSet := make(map[string]bool)
	for _, d := range detectionResults {
		if d.Detected {
			detectedSet[string(d.Platform)] = true
		}
	}

	for i := range dialog.options {
		platformID := string(dialog.options[i].Location.Platform)
		isPreferred := savedSet[platformID] || detectedSet[platformID]
		if dialog.options[i].Location.Scope == installer.ScopeGlobal {
			dialog.options[i].Selected = isPreferred
		} else {
			dialog.options[i].Selected = false
		}
	}

	return dialog
}

// mergePreferencesWithDetected reorders platforms:
// 1. Saved preferences (enabled) first
// 2. Newly detected (not in saved) second
// 3. All others last
func mergePreferencesWithDetected(allPlatforms []installer.Platform, saved []string, detected []detect.DetectionResult) []installer.Platform {
	savedSet := make(map[string]bool)
	for _, s := range saved {
		savedSet[s] = true
	}
	detectedSet := make(map[string]bool)
	for _, d := range detected {
		if d.Detected {
			detectedSet[string(d.Platform)] = true
		}
	}

	var result []installer.Platform
	// Saved first (in platform order)
	for _, p := range allPlatforms {
		if savedSet[string(p)] {
			result = append(result, p)
		}
	}
	// Newly detected (not saved)
	for _, p := range allPlatforms {
		if detectedSet[string(p)] && !savedSet[string(p)] {
			result = append(result, p)
		}
	}
	// Everything else
	for _, p := range allPlatforms {
		if !savedSet[string(p)] && !detectedSet[string(p)] {
			result = append(result, p)
		}
	}
	return result
}
