package components

import (
	"fmt"

	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmChangesDialog shows pending install/uninstall changes for a skill.
type ConfirmChangesDialog struct {
	skillSlug      string
	toInstall      []installer.InstallLocation
	toUninstall    []installer.InstallLocation
	doNotShowAgain bool
	onDoNotShow    bool // Cursor on "do not show" checkbox
	confirmed      bool
	cancelled      bool
	width          int
}

// NewConfirmChangesDialog creates a new confirmation dialog for pending changes.
func NewConfirmChangesDialog(skillSlug string, toInstall, toUninstall []installer.InstallLocation) *ConfirmChangesDialog {
	return &ConfirmChangesDialog{
		skillSlug:   skillSlug,
		toInstall:   toInstall,
		toUninstall: toUninstall,
	}
}

// SetWidth sets the dialog width.
func (d *ConfirmChangesDialog) SetWidth(w int) {
	d.width = w
}

// HandleKey processes keyboard input for the dialog.
func (d *ConfirmChangesDialog) HandleKey(key string) {
	switch key {
	case "up", "k":
		// Navigate to/from "do not show" checkbox
		d.onDoNotShow = false
	case "down", "j":
		// Navigate to/from "do not show" checkbox
		d.onDoNotShow = true
	case "space", " ":
		// Toggle "do not show" checkbox when on it
		if d.onDoNotShow {
			d.doNotShowAgain = !d.doNotShowAgain
		}
	case "enter":
		d.confirmed = true
	case "esc":
		d.cancelled = true
	}
}

// IsConfirmed returns true if the user confirmed the changes.
func (d *ConfirmChangesDialog) IsConfirmed() bool {
	return d.confirmed
}

// IsCancelled returns true if the user cancelled the dialog.
func (d *ConfirmChangesDialog) IsCancelled() bool {
	return d.cancelled
}

// DoNotShowAgain returns the "do not show" preference.
func (d *ConfirmChangesDialog) DoNotShowAgain() bool {
	return d.doNotShowAgain
}

// Reset clears the dialog state for reuse.
func (d *ConfirmChangesDialog) Reset() {
	d.doNotShowAgain = false
	d.onDoNotShow = false
	d.confirmed = false
	d.cancelled = false
}

// formatLocationPath formats a location path for display.
// Global: ~/.<platform>/skills/
// Project: ./.<platform>/skills/
func formatLocationPath(loc installer.InstallLocation) string {
	info := loc.Platform.Info()
	if loc.Scope == installer.ScopeGlobal {
		return "~/" + info.SkillsPath + "/"
	}
	return "./" + info.SkillsPath + "/"
}

// formatLocationLine formats a single location for display.
// Format: {Platform.Info().Name} - {Scope} ({path})
func formatLocationLine(loc installer.InstallLocation) string {
	info := loc.Platform.Info()
	var scopeName string
	if loc.Scope == installer.ScopeGlobal {
		scopeName = "Global"
	} else {
		scopeName = "Project"
	}
	return info.Name + " - " + scopeName + " (" + formatLocationPath(loc) + ")"
}

// View renders the dialog.
func (d *ConfirmChangesDialog) View() string {
	dialogWidth := d.width
	if dialogWidth == 0 || dialogWidth > 70 {
		dialogWidth = 70
	}
	if dialogWidth < 55 {
		dialogWidth = 55
	}
	contentWidth := dialogWidth - 6

	// Colors from theme
	accentColor := theme.Current.Primary
	goldColor := theme.Current.Accent
	mutedColor := theme.Current.TextMuted
	textColor := theme.Current.Text
	successColor := theme.Current.Success
	errorColor := theme.Current.Error
	selectedBgColor := theme.Current.Surface

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginBottom(1)

	title := titleStyle.Render("Confirm Changes")

	// Description
	descStyle := lipgloss.NewStyle().
		Foreground(textColor).
		Width(contentWidth).
		Align(lipgloss.Left).
		MarginBottom(1)

	desc := descStyle.Render("The following changes will be made to '" + d.skillSlug + "':")

	var parts []string
	parts = append(parts, title, desc)

	// INSTALL TO section (if any)
	if len(d.toInstall) > 0 {
		sectionStyle := lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true).
			MarginTop(1)

		parts = append(parts, sectionStyle.Render("INSTALL TO:"))

		for _, loc := range d.toInstall {
			lineStyle := lipgloss.NewStyle().
				Foreground(successColor).
				PaddingLeft(2)
			parts = append(parts, lineStyle.Render("+ "+formatLocationLine(loc)))
		}
	}

	// UNINSTALL FROM section (if any)
	if len(d.toUninstall) > 0 {
		sectionStyle := lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true).
			MarginTop(1)

		parts = append(parts, sectionStyle.Render("UNINSTALL FROM:"))

		for _, loc := range d.toUninstall {
			lineStyle := lipgloss.NewStyle().
				Foreground(errorColor).
				PaddingLeft(2)
			parts = append(parts, lineStyle.Render("- "+formatLocationLine(loc)))
		}
	}

	// "Do not show again" checkbox
	var checkbox string
	if d.doNotShowAgain {
		checkbox = lipgloss.NewStyle().Foreground(successColor).Render("☑")
	} else {
		checkbox = lipgloss.NewStyle().Foreground(mutedColor).Render("☐")
	}

	checkboxLabelStyle := lipgloss.NewStyle().Foreground(textColor)
	if d.onDoNotShow {
		checkboxLabelStyle = checkboxLabelStyle.Foreground(goldColor).Bold(true)
	}

	checkboxLine := checkbox + " " + checkboxLabelStyle.Render("Do not show this confirmation again")

	checkboxStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Padding(0, 1).
		MarginTop(1)

	if d.onDoNotShow {
		checkboxStyle = checkboxStyle.Background(selectedBgColor)
	}

	parts = append(parts, checkboxStyle.Render(checkboxLine))

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginTop(1).
		Italic(true)

	footer := footerStyle.Render("Enter confirm  •  Esc cancel")

	parts = append(parts, footer)

	// Dialog container with border
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Width(dialogWidth)

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)

	return dialogStyle.Render(content)
}

// CenteredView renders the dialog centered within the given dimensions.
func (d *ConfirmChangesDialog) CenteredView(width, height int) string {
	dialog := d.View()
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, dialog)
}

// HasChanges returns true if there are any changes to confirm.
func (d *ConfirmChangesDialog) HasChanges() bool {
	return len(d.toInstall) > 0 || len(d.toUninstall) > 0
}

// GetToInstall returns the locations to install to.
func (d *ConfirmChangesDialog) GetToInstall() []installer.InstallLocation {
	return d.toInstall
}

// GetToUninstall returns the locations to uninstall from.
func (d *ConfirmChangesDialog) GetToUninstall() []installer.InstallLocation {
	return d.toUninstall
}

// SetChanges updates the pending changes.
func (d *ConfirmChangesDialog) SetChanges(toInstall, toUninstall []installer.InstallLocation) {
	d.toInstall = toInstall
	d.toUninstall = toUninstall
}

// SetSkillSlug updates the skill slug displayed in the dialog.
func (d *ConfirmChangesDialog) SetSkillSlug(slug string) {
	d.skillSlug = slug
}

// Summary returns a brief summary of changes.
func (d *ConfirmChangesDialog) Summary() string {
	if len(d.toInstall) == 0 && len(d.toUninstall) == 0 {
		return "no changes"
	}

	var result string
	if len(d.toInstall) > 0 {
		result = fmt.Sprintf("+%d", len(d.toInstall))
	}
	if len(d.toUninstall) > 0 {
		if result != "" {
			result += " "
		}
		result += fmt.Sprintf("-%d", len(d.toUninstall))
	}
	return result
}
