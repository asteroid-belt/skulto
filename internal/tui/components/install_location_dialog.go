package components

import (
	"fmt"

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

// dialogItemKind identifies the type of a display item in the dialog.
type dialogItemKind int

const (
	dkOption    dialogItemKind = iota // Selectable location option
	dkHeader                          // Non-interactive group header
	dkToggle                          // Collapsible group header (interactive)
	dkRemember                        // Remember locations checkbox
	dkSeparator                       // Visual separator
)

// dialogDisplayItem represents a single row in the dialog's display list.
type dialogDisplayItem struct {
	kind      dialogItemKind
	optionIdx int    // Index into options slice (for dkOption)
	label     string // Display text (for headers/toggle)
}

// InstallLocationDialog presents installation location choices with multi-select
// and collapsible groups for preferred vs other agents.
type InstallLocationDialog struct {
	options           []LocationOption
	width             int
	height            int
	cancelled         bool
	confirmed         bool
	platforms         []installer.Platform
	rememberLocations bool

	// Collapsible groups
	preferredCount int  // options[0:preferredCount] = group 1 (preferred)
	group2Expanded bool // Whether "Other Agents" group is expanded

	// Display items for navigation
	displayItems []dialogDisplayItem
	currentIndex int // Index into displayItems
	scrollOffset int // Scroll offset for viewport
}

// NewInstallLocationDialog creates a new install location dialog.
// All platforms are in group 1 (no collapsible group 2).
func NewInstallLocationDialog(platforms []installer.Platform) *InstallLocationDialog {
	dialog := &InstallLocationDialog{
		platforms: platforms,
		options:   make([]LocationOption, 0),
	}
	dialog.buildOptions()
	dialog.preferredCount = len(dialog.options) // All in group 1
	dialog.buildDisplayItems()
	return dialog
}

// buildOptions creates location options for each platform x scope combination.
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

// buildDisplayItems creates the flattened display item list for navigation and rendering.
func (d *InstallLocationDialog) buildDisplayItems() {
	d.displayItems = nil

	hasGroups := d.preferredCount > 0 && d.preferredCount < len(d.options)

	if hasGroups {
		// Group 1: Preferred agents
		d.displayItems = append(d.displayItems, dialogDisplayItem{
			kind: dkHeader, label: "Your Agents",
		})
		for i := 0; i < d.preferredCount; i++ {
			d.displayItems = append(d.displayItems, dialogDisplayItem{
				kind: dkOption, optionIdx: i,
			})
		}

		d.displayItems = append(d.displayItems, dialogDisplayItem{kind: dkSeparator})

		// Group 2: Other agents (collapsible)
		otherPlatforms := d.countOtherPlatforms()
		label := fmt.Sprintf("Other Agents (%d)", otherPlatforms)
		d.displayItems = append(d.displayItems, dialogDisplayItem{
			kind: dkToggle, label: label,
		})

		if d.group2Expanded {
			for i := d.preferredCount; i < len(d.options); i++ {
				d.displayItems = append(d.displayItems, dialogDisplayItem{
					kind: dkOption, optionIdx: i,
				})
			}
		}
	} else {
		// Single group: all options
		for i := range d.options {
			d.displayItems = append(d.displayItems, dialogDisplayItem{
				kind: dkOption, optionIdx: i,
			})
		}
	}

	// Remember option always at end
	d.displayItems = append(d.displayItems, dialogDisplayItem{
		kind: dkRemember,
	})
}

// countOtherPlatforms returns the number of unique platforms in group 2.
func (d *InstallLocationDialog) countOtherPlatforms() int {
	seen := make(map[installer.Platform]bool)
	for i := d.preferredCount; i < len(d.options); i++ {
		seen[d.options[i].Location.Platform] = true
	}
	return len(seen)
}

// isInteractiveDialogItem returns true if the item can be navigated to.
func isInteractiveDialogItem(kind dialogItemKind) bool {
	return kind == dkOption || kind == dkToggle || kind == dkRemember
}

// SetWidth sets the dialog width.
func (d *InstallLocationDialog) SetWidth(w int) {
	d.width = w
}

// SetHeight sets the dialog height for viewport calculations.
func (d *InstallLocationDialog) SetHeight(h int) {
	d.height = h
}

// Update handles keyboard input for the dialog.
func (d *InstallLocationDialog) Update(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyUp, tea.KeyShiftTab:
		d.moveCursor(-1)
	case tea.KeyDown, tea.KeyTab:
		d.moveCursor(1)
	case tea.KeySpace:
		d.handleToggle()
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
				d.moveCursor(1)
			case "k":
				d.moveCursor(-1)
			case " ":
				d.handleToggle()
			case "a":
				for i := range d.options {
					d.options[i].Selected = true
				}
			case "n":
				for i := range d.options {
					d.options[i].Selected = false
				}
			case "g":
				for i := range d.options {
					d.options[i].Selected = d.options[i].Location.Scope == installer.ScopeGlobal
				}
			case "p":
				for i := range d.options {
					d.options[i].Selected = d.options[i].Location.Scope == installer.ScopeProject
				}
			case "r":
				d.rememberLocations = !d.rememberLocations
			}
		}
	}
}

// HandleKey processes string key for compatibility.
func (d *InstallLocationDialog) HandleKey(key string) {
	switch key {
	case "up", "k":
		d.moveCursor(-1)
	case "down", "j":
		d.moveCursor(1)
	case "space", " ":
		d.handleToggle()
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

// moveCursor moves the cursor by delta, skipping non-interactive items.
func (d *InstallLocationDialog) moveCursor(delta int) {
	next := d.currentIndex + delta
	for next >= 0 && next < len(d.displayItems) {
		if isInteractiveDialogItem(d.displayItems[next].kind) {
			d.currentIndex = next
			d.ensureVisible()
			return
		}
		next += delta
	}
	// Wrap around
	if delta > 0 {
		for i := 0; i < len(d.displayItems); i++ {
			if isInteractiveDialogItem(d.displayItems[i].kind) {
				d.currentIndex = i
				d.ensureVisible()
				return
			}
		}
	} else {
		for i := len(d.displayItems) - 1; i >= 0; i-- {
			if isInteractiveDialogItem(d.displayItems[i].kind) {
				d.currentIndex = i
				d.ensureVisible()
				return
			}
		}
	}
}

// handleToggle handles space/enter on the current display item.
func (d *InstallLocationDialog) handleToggle() {
	if d.currentIndex < 0 || d.currentIndex >= len(d.displayItems) {
		return
	}
	item := d.displayItems[d.currentIndex]
	switch item.kind {
	case dkOption:
		d.options[item.optionIdx].Selected = !d.options[item.optionIdx].Selected
	case dkToggle:
		d.group2Expanded = !d.group2Expanded
		d.buildDisplayItems()
		// Keep cursor on toggle header
		for i, di := range d.displayItems {
			if di.kind == dkToggle {
				d.currentIndex = i
				break
			}
		}
		d.ensureVisible()
	case dkRemember:
		d.rememberLocations = !d.rememberLocations
	}
}

// ensureVisible adjusts scroll offset so the cursor is visible.
func (d *InstallLocationDialog) ensureVisible() {
	vpHeight := d.calcViewportHeight()
	if vpHeight <= 0 || vpHeight >= len(d.displayItems) {
		d.scrollOffset = 0
		return
	}
	if d.currentIndex < d.scrollOffset {
		d.scrollOffset = d.currentIndex
	}
	if d.currentIndex >= d.scrollOffset+vpHeight {
		d.scrollOffset = d.currentIndex - vpHeight + 1
	}
}

// calcViewportHeight returns the number of display items visible at once.
func (d *InstallLocationDialog) calcViewportHeight() int {
	if d.height <= 0 {
		return len(d.displayItems) // No height constraint, show all
	}
	// Each option takes ~3 lines (name + desc + gap), headers take ~1-2 lines
	// Account for title, subtitle, footer, borders, padding (~14 lines)
	vpLines := d.height - 14
	if vpLines < 10 {
		vpLines = 10
	}
	// Estimate ~2 lines per display item on average
	vpItems := vpLines / 2
	if vpItems < 5 {
		vpItems = 5
	}
	return vpItems
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
	d.rememberLocations = false
	d.scrollOffset = 0
	// Reset selections to default (global selected, project not)
	for i := range d.options {
		d.options[i].Selected = d.options[i].Location.Scope == installer.ScopeGlobal
	}
	// Collapse group 2 on reset
	d.group2Expanded = false
	d.buildDisplayItems()
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

	// Viewport for scrolling
	vpHeight := d.calcViewportHeight()
	start := d.scrollOffset
	if start < 0 {
		start = 0
	}
	end := start + vpHeight
	if end > len(d.displayItems) {
		end = len(d.displayItems)
	}

	// Render display items
	var itemViews []string

	for idx := start; idx < end; idx++ {
		item := d.displayItems[idx]
		isCurrent := idx == d.currentIndex

		switch item.kind {
		case dkHeader:
			headerStyle := lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true).
				Width(contentWidth).
				MarginTop(1)
			itemViews = append(itemViews, headerStyle.Render(item.label))

		case dkSeparator:
			sepStyle := lipgloss.NewStyle().
				Foreground(mutedColor).
				Width(contentWidth)
			itemViews = append(itemViews, sepStyle.Render("───"))

		case dkToggle:
			arrow := "▶"
			if d.group2Expanded {
				arrow = "▼"
			}
			toggleStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(accentColor).
				Width(contentWidth).
				MarginTop(1)
			if isCurrent {
				toggleStyle = toggleStyle.
					Foreground(goldColor).
					Background(selectedBgColor).
					Padding(0, 1)
			}
			itemViews = append(itemViews, toggleStyle.Render(arrow+" "+item.label))

		case dkOption:
			opt := d.options[item.optionIdx]

			var checkbox string
			if opt.Selected {
				checkbox = lipgloss.NewStyle().Foreground(successColor).Render("☑")
			} else {
				checkbox = lipgloss.NewStyle().Foreground(mutedColor).Render("☐")
			}

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

			itemViews = append(itemViews, optStyle.Render(optContent))

		case dkRemember:
			var rememberCheckbox string
			if d.rememberLocations {
				rememberCheckbox = lipgloss.NewStyle().Foreground(successColor).Render("☑")
			} else {
				rememberCheckbox = lipgloss.NewStyle().Foreground(mutedColor).Render("☐")
			}

			rememberNameStyle := lipgloss.NewStyle().Foreground(textColor)
			rememberDescStyle := lipgloss.NewStyle().Foreground(mutedColor).Italic(true)

			if isCurrent {
				rememberNameStyle = rememberNameStyle.Foreground(goldColor).Bold(true)
			}

			rememberLine := rememberCheckbox + " " + rememberNameStyle.Render("Remember these locations")
			rememberDesc := "    " + rememberDescStyle.Render("Skip this dialog for future installs")

			rememberContent := lipgloss.JoinVertical(lipgloss.Left, rememberLine, rememberDesc)

			rememberStyle := lipgloss.NewStyle().
				Width(contentWidth).
				Padding(0, 1).
				MarginTop(1)

			if isCurrent {
				rememberStyle = rememberStyle.Background(selectedBgColor)
			}

			itemViews = append(itemViews, rememberStyle.Render(rememberContent))
		}
	}

	options := lipgloss.JoinVertical(lipgloss.Left, itemViews...)

	// Scroll indicators
	var scrollUp string
	if start > 0 {
		scrollUp = lipgloss.NewStyle().
			Foreground(mutedColor).
			Width(contentWidth).
			Align(lipgloss.Center).
			Render("↑ more above")
	}
	var scrollDown string
	if end < len(d.displayItems) {
		scrollDown = lipgloss.NewStyle().
			Foreground(mutedColor).
			Width(contentWidth).
			Align(lipgloss.Center).
			Render("↓ more below")
	}

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

	parts := []string{title, subtitle, ""}
	if scrollUp != "" {
		parts = append(parts, scrollUp)
	}
	parts = append(parts, options)
	if scrollDown != "" {
		parts = append(parts, scrollDown)
	}
	if validationMsg != "" {
		parts = append(parts, "", validationMsg)
	}
	parts = append(parts, "", footer)

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)

	return dialogStyle.Render(content)
}

// CenteredView renders the dialog centered within the given dimensions.
func (d *InstallLocationDialog) CenteredView(width, height int) string {
	d.height = height
	dialog := d.View()
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, dialog)
}

// NewInstallLocationDialogWithPrefs creates a dialog with merged preferences + detection.
// savedScopes maps platform agent_id → preferred scope ("global" or "project").
// Group 1 = preferred (saved + detected), Group 2 = all others (collapsed).
func NewInstallLocationDialogWithPrefs(platforms []installer.Platform, savedScopes map[string]string, detectionResults []detect.DetectionResult) *InstallLocationDialog {
	// Build plain saved list for merge ordering
	var plainSaved []string
	for p := range savedScopes {
		plainSaved = append(plainSaved, p)
	}

	// Merge and reorder platforms
	merged := mergePreferencesWithDetected(platforms, plainSaved, detectionResults)

	dialog := &InstallLocationDialog{
		platforms: merged,
		options:   make([]LocationOption, 0),
	}
	dialog.buildOptions()

	// Determine preferred count: saved + detected platforms
	detectedSet := make(map[string]bool)
	for _, dr := range detectionResults {
		if dr.Detected {
			detectedSet[string(dr.Platform)] = true
		}
	}

	// Count options that belong to preferred platforms
	preferredCount := 0
	for _, opt := range dialog.options {
		platformID := string(opt.Location.Platform)
		_, isSaved := savedScopes[platformID]
		if isSaved || detectedSet[platformID] {
			preferredCount++
		} else {
			break // Options are ordered: preferred first, then others
		}
	}
	dialog.preferredCount = preferredCount

	// Pre-select: use saved scope for saved platforms, global for detected-only
	for i := range dialog.options {
		platformID := string(dialog.options[i].Location.Platform)
		optScope := string(dialog.options[i].Location.Scope)

		if prefScope, isSaved := savedScopes[platformID]; isSaved {
			// Match the saved scope for this platform
			dialog.options[i].Selected = (optScope == prefScope)
		} else if detectedSet[platformID] {
			// Detected but not saved: default to global
			dialog.options[i].Selected = (dialog.options[i].Location.Scope == installer.ScopeGlobal)
		} else {
			dialog.options[i].Selected = false
		}
	}

	dialog.buildDisplayItems()
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
