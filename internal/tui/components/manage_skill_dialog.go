package components

import (
	"fmt"

	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// LocationCheckbox represents a single platform+scope checkbox.
type LocationCheckbox struct {
	Platform     installer.Platform
	Scope        installer.InstallScope
	DisplayName  string // e.g., "Global"
	Description  string // e.g., "~/.claude/skills/"
	Checked      bool   // Current state (user's selection)
	WasInstalled bool   // Original state (from DB)
}

// manageItemKind identifies the type of a display item in the manage dialog.
type manageItemKind int

const (
	mkCheckbox  manageItemKind = iota // Selectable checkbox
	mkHeader                          // Non-interactive group header (platform name)
	mkGroupHdr                        // Non-interactive section header ("Installed", etc.)
	mkToggle                          // Collapsible group toggle
	mkSeparator                       // Visual separator
)

// manageDisplayItem represents a single row in the dialog's display list.
type manageDisplayItem struct {
	kind        manageItemKind
	checkboxIdx int    // Index into checkboxes slice (for mkCheckbox)
	label       string // Display text (for headers/toggle)
}

// ManageSkillDialog allows editing installation locations for a skill.
type ManageSkillDialog struct {
	skill      installer.InstalledSkillSummary
	checkboxes []LocationCheckbox
	platforms  []installer.Platform // User's configured platforms

	// Display items for navigation
	displayItems   []manageDisplayItem
	currentIndex   int // Index into displayItems
	scrollOffset   int
	width          int
	height         int

	// Collapsible groups
	installedCount int  // checkboxes[0:installedCount] = group 1
	group2Expanded bool // Whether "Other Agents" group is expanded

	cancelled bool
	confirmed bool
}

// NewManageSkillDialog creates a new dialog for managing skill installation locations.
// Installed platforms appear at top, others are in a collapsed group.
func NewManageSkillDialog(skill installer.InstalledSkillSummary, platforms []installer.Platform) *ManageSkillDialog {
	dialog := &ManageSkillDialog{
		skill:     skill,
		platforms: platforms,
	}
	dialog.buildCheckboxes()
	dialog.buildDisplayItems()
	// Start cursor on first interactive item
	for i, item := range dialog.displayItems {
		if isManageInteractive(item.kind) {
			dialog.currentIndex = i
			break
		}
	}
	return dialog
}

// buildCheckboxes creates location checkboxes, installed platforms first.
func (d *ManageSkillDialog) buildCheckboxes() {
	d.checkboxes = make([]LocationCheckbox, 0, len(d.platforms)*2)

	// First pass: installed platforms
	for _, platform := range d.platforms {
		globalInstalled := d.isInstalled(platform, installer.ScopeGlobal)
		projectInstalled := d.isInstalled(platform, installer.ScopeProject)
		if !globalInstalled && !projectInstalled {
			continue
		}
		d.addPlatformCheckboxes(platform, globalInstalled, projectInstalled)
	}
	d.installedCount = len(d.checkboxes)

	// Second pass: non-installed platforms
	for _, platform := range d.platforms {
		globalInstalled := d.isInstalled(platform, installer.ScopeGlobal)
		projectInstalled := d.isInstalled(platform, installer.ScopeProject)
		if globalInstalled || projectInstalled {
			continue
		}
		d.addPlatformCheckboxes(platform, false, false)
	}
}

// addPlatformCheckboxes adds global + project checkboxes for a platform.
func (d *ManageSkillDialog) addPlatformCheckboxes(platform installer.Platform, globalInstalled, projectInstalled bool) {
	info := platform.Info()

	d.checkboxes = append(d.checkboxes, LocationCheckbox{
		Platform:     platform,
		Scope:        installer.ScopeGlobal,
		DisplayName:  "Global",
		Description:  "~/" + info.SkillsPath + "/",
		Checked:      globalInstalled,
		WasInstalled: globalInstalled,
	})

	d.checkboxes = append(d.checkboxes, LocationCheckbox{
		Platform:     platform,
		Scope:        installer.ScopeProject,
		DisplayName:  "Project",
		Description:  "./" + info.SkillsPath + "/",
		Checked:      projectInstalled,
		WasInstalled: projectInstalled,
	})
}

// isInstalled checks if the skill is installed at the given platform and scope.
func (d *ManageSkillDialog) isInstalled(platform installer.Platform, scope installer.InstallScope) bool {
	scopes, exists := d.skill.Locations[platform]
	if !exists {
		return false
	}
	for _, s := range scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// buildDisplayItems creates the flattened display item list.
func (d *ManageSkillDialog) buildDisplayItems() {
	d.displayItems = nil

	hasOthers := d.installedCount < len(d.checkboxes)

	// Group 1: Installed platforms
	if d.installedCount > 0 {
		d.displayItems = append(d.displayItems, manageDisplayItem{
			kind: mkGroupHdr, label: "Installed",
		})

		var lastPlatform installer.Platform
		for i := 0; i < d.installedCount; i++ {
			cb := d.checkboxes[i]
			if cb.Platform != lastPlatform {
				d.displayItems = append(d.displayItems, manageDisplayItem{
					kind: mkHeader, label: cb.Platform.Info().Name,
				})
				lastPlatform = cb.Platform
			}
			d.displayItems = append(d.displayItems, manageDisplayItem{
				kind: mkCheckbox, checkboxIdx: i,
			})
		}
	}

	// Group 2: Other platforms (collapsible)
	if hasOthers {
		if d.installedCount > 0 {
			d.displayItems = append(d.displayItems, manageDisplayItem{kind: mkSeparator})
		}

		otherCount := d.countOtherPlatforms()
		label := fmt.Sprintf("Other Agents (%d)", otherCount)
		d.displayItems = append(d.displayItems, manageDisplayItem{
			kind: mkToggle, label: label,
		})

		if d.group2Expanded {
			var lastPlatform installer.Platform
			for i := d.installedCount; i < len(d.checkboxes); i++ {
				cb := d.checkboxes[i]
				if cb.Platform != lastPlatform {
					d.displayItems = append(d.displayItems, manageDisplayItem{
						kind: mkHeader, label: cb.Platform.Info().Name,
					})
					lastPlatform = cb.Platform
				}
				d.displayItems = append(d.displayItems, manageDisplayItem{
					kind: mkCheckbox, checkboxIdx: i,
				})
			}
		}
	}

	// If no installed platforms, auto-expand others
	if d.installedCount == 0 && hasOthers && !d.group2Expanded {
		d.group2Expanded = true
		d.buildDisplayItems()
	}
}

// countOtherPlatforms returns the number of unique platforms in group 2.
func (d *ManageSkillDialog) countOtherPlatforms() int {
	seen := make(map[installer.Platform]bool)
	for i := d.installedCount; i < len(d.checkboxes); i++ {
		seen[d.checkboxes[i].Platform] = true
	}
	return len(seen)
}

// isManageInteractive returns true if the item kind can be navigated to.
func isManageInteractive(kind manageItemKind) bool {
	return kind == mkCheckbox || kind == mkToggle
}

// SetWidth sets the dialog width.
func (d *ManageSkillDialog) SetWidth(w int) {
	d.width = w
}

// SetHeight sets the dialog height.
func (d *ManageSkillDialog) SetHeight(h int) {
	d.height = h
}

// HandleKey processes keyboard input for the dialog.
func (d *ManageSkillDialog) HandleKey(key string) {
	switch key {
	case "up", "k":
		d.moveCursor(-1)
	case "down", "j":
		d.moveCursor(1)
	case "space", " ":
		d.handleToggle()
	case "enter":
		d.confirmed = true
	case "esc":
		d.cancelled = true
	case "a":
		// Select all visible
		for idx := range d.displayItems {
			item := d.displayItems[idx]
			if item.kind == mkCheckbox {
				d.checkboxes[item.checkboxIdx].Checked = true
			}
		}
	case "n":
		// Select none visible
		for idx := range d.displayItems {
			item := d.displayItems[idx]
			if item.kind == mkCheckbox {
				d.checkboxes[item.checkboxIdx].Checked = false
			}
		}
	}
}

// moveCursor moves cursor by delta, skipping non-interactive items, with wrapping.
func (d *ManageSkillDialog) moveCursor(delta int) {
	next := d.currentIndex + delta
	for next >= 0 && next < len(d.displayItems) {
		if isManageInteractive(d.displayItems[next].kind) {
			d.currentIndex = next
			d.ensureVisible()
			return
		}
		next += delta
	}
	// Wrap around
	if delta > 0 {
		for i := 0; i < len(d.displayItems); i++ {
			if isManageInteractive(d.displayItems[i].kind) {
				d.currentIndex = i
				d.ensureVisible()
				return
			}
		}
	} else {
		for i := len(d.displayItems) - 1; i >= 0; i-- {
			if isManageInteractive(d.displayItems[i].kind) {
				d.currentIndex = i
				d.ensureVisible()
				return
			}
		}
	}
}

// handleToggle handles space on the current display item.
func (d *ManageSkillDialog) handleToggle() {
	if d.currentIndex < 0 || d.currentIndex >= len(d.displayItems) {
		return
	}
	item := d.displayItems[d.currentIndex]
	switch item.kind {
	case mkCheckbox:
		d.checkboxes[item.checkboxIdx].Checked = !d.checkboxes[item.checkboxIdx].Checked
	case mkToggle:
		d.group2Expanded = !d.group2Expanded
		d.buildDisplayItems()
		// Keep cursor on toggle
		for i, di := range d.displayItems {
			if di.kind == mkToggle {
				d.currentIndex = i
				break
			}
		}
		d.ensureVisible()
	}
}

// ensureVisible adjusts scroll offset so the cursor is visible.
// When scrolling up, includes preceding non-interactive headers for context.
func (d *ManageSkillDialog) ensureVisible() {
	vpHeight := d.calcViewportHeight()
	if vpHeight <= 0 || vpHeight >= len(d.displayItems) {
		d.scrollOffset = 0
		return
	}
	if d.currentIndex < d.scrollOffset {
		// Include preceding non-interactive items (headers) so they stay visible
		target := d.currentIndex
		for target > 0 && !isManageInteractive(d.displayItems[target-1].kind) {
			target--
		}
		d.scrollOffset = target
	}
	if d.currentIndex >= d.scrollOffset+vpHeight {
		d.scrollOffset = d.currentIndex - vpHeight + 1
	}
}

// calcViewportHeight returns the number of display items visible at once.
func (d *ManageSkillDialog) calcViewportHeight() int {
	if d.height <= 0 {
		return len(d.displayItems)
	}
	// Account for title, subtitle, footer, borders, padding (~12 lines)
	vpLines := d.height - 12
	if vpLines < 8 {
		vpLines = 8
	}
	// ~2 lines per item average (checkbox lines + platform headers)
	vpItems := vpLines / 2
	if vpItems < 5 {
		vpItems = 5
	}
	return vpItems
}

// IsConfirmed returns true if the user confirmed their selection.
func (d *ManageSkillDialog) IsConfirmed() bool {
	return d.confirmed
}

// IsCancelled returns true if the user cancelled the dialog.
func (d *ManageSkillDialog) IsCancelled() bool {
	return d.cancelled
}

// GetSkill returns the skill being edited.
func (d *ManageSkillDialog) GetSkill() *installer.InstalledSkillSummary {
	return &d.skill
}

// GetChanges returns locations to install and uninstall based on checkbox changes.
func (d *ManageSkillDialog) GetChanges() (toInstall, toUninstall []installer.InstallLocation) {
	for _, cb := range d.checkboxes {
		loc, err := installer.NewInstallLocation(cb.Platform, cb.Scope)
		if err != nil {
			continue
		}

		if cb.Checked && !cb.WasInstalled {
			toInstall = append(toInstall, loc)
		} else if !cb.Checked && cb.WasInstalled {
			toUninstall = append(toUninstall, loc)
		}
	}
	return toInstall, toUninstall
}

// HasChanges returns true if any checkbox state differs from the original.
func (d *ManageSkillDialog) HasChanges() bool {
	for _, cb := range d.checkboxes {
		if cb.Checked != cb.WasInstalled {
			return true
		}
	}
	return false
}

// HasRemovals returns true if any installed location is now unchecked.
func (d *ManageSkillDialog) HasRemovals() bool {
	for _, cb := range d.checkboxes {
		if cb.WasInstalled && !cb.Checked {
			return true
		}
	}
	return false
}

// Reset clears the dialog state for reuse.
func (d *ManageSkillDialog) Reset() {
	d.currentIndex = 0
	d.scrollOffset = 0
	d.cancelled = false
	d.confirmed = false
	d.group2Expanded = false
	// Reset checkboxes to original state
	for i := range d.checkboxes {
		d.checkboxes[i].Checked = d.checkboxes[i].WasInstalled
	}
	d.buildDisplayItems()
	// Reset cursor to first interactive
	for i, item := range d.displayItems {
		if isManageInteractive(item.kind) {
			d.currentIndex = i
			break
		}
	}
}

// View renders the dialog.
func (d *ManageSkillDialog) View() string {
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

	title := titleStyle.Render("Edit: " + d.skill.Slug)

	// Subtitle
	subtitleStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginBottom(1)

	subtitle := subtitleStyle.Render("Select installation locations:")

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
		case mkGroupHdr:
			hdrStyle := lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true).
				Width(contentWidth).
				MarginTop(1)
			itemViews = append(itemViews, hdrStyle.Render(item.label))

		case mkHeader:
			platformHeaderStyle := lipgloss.NewStyle().
				Foreground(textColor).
				Bold(true).
				MarginTop(1)
			itemViews = append(itemViews, platformHeaderStyle.Render(item.label))

		case mkSeparator:
			sepStyle := lipgloss.NewStyle().
				Foreground(mutedColor).
				Width(contentWidth)
			itemViews = append(itemViews, sepStyle.Render("───"))

		case mkToggle:
			arrow := "▶"
			if d.group2Expanded {
				arrow = "▼"
			}
			indicator := "  "
			if isCurrent {
				indicator = "▸ "
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
			itemViews = append(itemViews, toggleStyle.Render(indicator+arrow+" "+item.label))

		case mkCheckbox:
			cb := d.checkboxes[item.checkboxIdx]

			// Checkbox icon
			var checkbox string
			if cb.Checked {
				checkbox = lipgloss.NewStyle().Foreground(successColor).Render("☑")
			} else {
				checkbox = lipgloss.NewStyle().Foreground(mutedColor).Render("☐")
			}

			// Selection indicator
			indicator := "  "
			if isCurrent {
				indicator = lipgloss.NewStyle().Foreground(goldColor).Render("▸ ")
			}

			// Label styling
			nameStyle := lipgloss.NewStyle().Foreground(textColor)
			descStyle := lipgloss.NewStyle().Foreground(mutedColor).Italic(true)

			if isCurrent {
				nameStyle = nameStyle.Foreground(goldColor).Bold(true)
			}

			line := indicator + checkbox + " " + nameStyle.Render(cb.DisplayName)
			line += "    " + descStyle.Render("("+cb.Description+")")

			optStyle := lipgloss.NewStyle().
				Width(contentWidth).
				PaddingLeft(2)

			if isCurrent {
				optStyle = optStyle.Background(selectedBgColor)
			}

			itemViews = append(itemViews, optStyle.Render(line))
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

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginTop(1).
		Italic(true)

	footerText := "Space toggle • a all • n none • Enter save • Esc cancel"
	footer := footerStyle.Render(footerText)

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
	parts = append(parts, "", footer)

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)

	return dialogStyle.Render(content)
}

// CenteredView renders the dialog centered within the given dimensions.
func (d *ManageSkillDialog) CenteredView(width, height int) string {
	d.height = height
	dialog := d.View()
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, dialog)
}
