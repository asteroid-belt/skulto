package components

import (
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// LocationCheckbox represents a single platform+scope checkbox.
type LocationCheckbox struct {
	Platform     installer.Platform
	Scope        installer.InstallScope
	DisplayName  string // e.g., "Claude Code - Global"
	Description  string // e.g., "~/.claude/skills/"
	Checked      bool   // Current state (user's selection)
	WasInstalled bool   // Original state (from DB)
}

// ManageSkillDialog allows editing installation locations for a skill.
type ManageSkillDialog struct {
	skill        installer.InstalledSkillSummary
	checkboxes   []LocationCheckbox
	currentIndex int
	width        int
	cancelled    bool
	confirmed    bool
	platforms    []installer.Platform // User's configured platforms
}

// NewManageSkillDialog creates a new dialog for managing skill installation locations.
func NewManageSkillDialog(skill installer.InstalledSkillSummary, platforms []installer.Platform) *ManageSkillDialog {
	dialog := &ManageSkillDialog{
		skill:     skill,
		platforms: platforms,
	}
	dialog.buildCheckboxes()
	return dialog
}

// buildCheckboxes creates location checkboxes for each platform x scope combination.
func (d *ManageSkillDialog) buildCheckboxes() {
	d.checkboxes = make([]LocationCheckbox, 0, len(d.platforms)*2)

	for _, platform := range d.platforms {
		info := platform.Info()

		// Check if platform is installed at global scope
		globalInstalled := d.isInstalled(platform, installer.ScopeGlobal)
		d.checkboxes = append(d.checkboxes, LocationCheckbox{
			Platform:     platform,
			Scope:        installer.ScopeGlobal,
			DisplayName:  "Global",
			Description:  "~/" + info.SkillsPath + "/",
			Checked:      globalInstalled,
			WasInstalled: globalInstalled,
		})

		// Check if platform is installed at project scope
		projectInstalled := d.isInstalled(platform, installer.ScopeProject)
		d.checkboxes = append(d.checkboxes, LocationCheckbox{
			Platform:     platform,
			Scope:        installer.ScopeProject,
			DisplayName:  "Project",
			Description:  "./" + info.SkillsPath + "/",
			Checked:      projectInstalled,
			WasInstalled: projectInstalled,
		})
	}
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

// SetWidth sets the dialog width.
func (d *ManageSkillDialog) SetWidth(w int) {
	d.width = w
}

// HandleKey processes keyboard input for the dialog.
func (d *ManageSkillDialog) HandleKey(key string) {
	totalItems := len(d.checkboxes)
	if totalItems == 0 {
		return
	}

	switch key {
	case "up", "k":
		d.currentIndex--
		if d.currentIndex < 0 {
			d.currentIndex = totalItems - 1
		}
	case "down", "j":
		d.currentIndex++
		if d.currentIndex >= totalItems {
			d.currentIndex = 0
		}
	case "space", " ":
		if d.currentIndex >= 0 && d.currentIndex < totalItems {
			d.checkboxes[d.currentIndex].Checked = !d.checkboxes[d.currentIndex].Checked
		}
	case "enter":
		d.confirmed = true
	case "esc":
		d.cancelled = true
	case "a":
		// Select all
		for i := range d.checkboxes {
			d.checkboxes[i].Checked = true
		}
	case "n":
		// Select none
		for i := range d.checkboxes {
			d.checkboxes[i].Checked = false
		}
	}
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
			// New installation
			toInstall = append(toInstall, loc)
		} else if !cb.Checked && cb.WasInstalled {
			// Removal
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
	d.cancelled = false
	d.confirmed = false
	// Reset checkboxes to original state
	for i := range d.checkboxes {
		d.checkboxes[i].Checked = d.checkboxes[i].WasInstalled
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

	// Group checkboxes by platform
	var platformSections []string
	checkboxIdx := 0

	for _, platform := range d.platforms {
		info := platform.Info()

		// Platform header
		platformHeaderStyle := lipgloss.NewStyle().
			Foreground(textColor).
			Bold(true).
			MarginTop(1)
		platformHeader := platformHeaderStyle.Render(info.Name)

		// Render checkboxes for this platform (global and project)
		var checkboxViews []string
		for i := 0; i < 2 && checkboxIdx < len(d.checkboxes); i++ {
			cb := d.checkboxes[checkboxIdx]
			if cb.Platform != platform {
				break
			}

			isCurrent := checkboxIdx == d.currentIndex

			// Checkbox icon
			var checkbox string
			if cb.Checked {
				checkbox = lipgloss.NewStyle().Foreground(successColor).Render("\u2611")
			} else {
				checkbox = lipgloss.NewStyle().Foreground(mutedColor).Render("\u2610")
			}

			// Selection indicator
			indicator := "  "
			if isCurrent {
				indicator = lipgloss.NewStyle().Foreground(goldColor).Render("\u25b8 ")
			}

			// Checkbox label
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

			checkboxViews = append(checkboxViews, optStyle.Render(line))
			checkboxIdx++
		}

		section := lipgloss.JoinVertical(lipgloss.Left,
			platformHeader,
			lipgloss.JoinVertical(lipgloss.Left, checkboxViews...),
		)
		platformSections = append(platformSections, section)
	}

	options := lipgloss.JoinVertical(lipgloss.Left, platformSections...)

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginTop(1).
		Italic(true)

	footerText := "Space toggle \u2022 a all \u2022 n none \u2022 Enter save \u2022 Esc cancel"
	footer := footerStyle.Render(footerText)

	// Dialog container
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Width(dialogWidth)

	content := lipgloss.JoinVertical(lipgloss.Left, title, subtitle, "", options, "", footer)

	return dialogStyle.Render(content)
}

// CenteredView renders the dialog centered within the given dimensions.
func (d *ManageSkillDialog) CenteredView(width, height int) string {
	dialog := d.View()
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, dialog)
}
