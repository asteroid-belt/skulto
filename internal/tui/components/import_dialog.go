// Package components provides TUI components for skulto.
package components

import (
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ImportDialogAction represents the action selected by the user.
type ImportDialogAction int

const (
	// ImportDialogCancel indicates the user cancelled the import.
	ImportDialogCancel ImportDialogAction = iota
	// ImportDialogImport indicates the user wants to import the skill.
	ImportDialogImport
	// ImportDialogSkip indicates the user wants to skip this skill (conflict mode).
	ImportDialogSkip
	// ImportDialogRename indicates the user wants to rename and import (conflict mode).
	ImportDialogRename
	// ImportDialogReplace indicates the user wants to replace the existing skill (conflict mode).
	ImportDialogReplace
)

// ImportDialog is a dialog for confirming skill import operations.
// In normal mode, it shows Import/Cancel options.
// In conflict mode, it shows Rename/Skip/Replace options.
type ImportDialog struct {
	skillName   string
	skillPath   string
	hasConflict bool
	selectedIdx int
	width       int

	// Rename mode state
	renaming bool
	newName  string

	// Dialog result state
	confirmed bool
	cancelled bool
}

// NewImportDialog creates a new import dialog.
func NewImportDialog(name, path string, hasConflict bool) *ImportDialog {
	return &ImportDialog{
		skillName:   name,
		skillPath:   path,
		hasConflict: hasConflict,
		selectedIdx: 0,
	}
}

// SetWidth sets the dialog width.
func (d *ImportDialog) SetWidth(w int) {
	d.width = w
}

// Update handles keyboard input for the dialog.
func (d *ImportDialog) Update(msg tea.KeyMsg) {
	if d.renaming {
		d.handleRenameInput(msg)
		return
	}

	optionCount := d.getOptionCount()

	switch msg.Type {
	case tea.KeyRight, tea.KeyTab:
		d.selectedIdx = (d.selectedIdx + 1) % optionCount
	case tea.KeyLeft, tea.KeyShiftTab:
		d.selectedIdx = (d.selectedIdx - 1 + optionCount) % optionCount
	case tea.KeyEnter:
		d.handleEnter()
	case tea.KeyEsc:
		d.cancelled = true
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "l":
			d.selectedIdx = (d.selectedIdx + 1) % optionCount
		case "h":
			d.selectedIdx = (d.selectedIdx - 1 + optionCount) % optionCount
		}
	}
}

// HandleKey processes string key for compatibility.
func (d *ImportDialog) HandleKey(key string) {
	optionCount := d.getOptionCount()

	switch key {
	case "right", "l":
		d.selectedIdx = (d.selectedIdx + 1) % optionCount
	case "left", "h":
		d.selectedIdx = (d.selectedIdx - 1 + optionCount) % optionCount
	case "enter":
		d.handleEnter()
	case "esc":
		d.cancelled = true
	}
}

func (d *ImportDialog) handleEnter() {
	if d.hasConflict && d.selectedIdx == 0 {
		// Rename option selected - enter rename mode
		d.renaming = true
		d.newName = ""
		return
	}
	d.confirmed = true
}

func (d *ImportDialog) handleRenameInput(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyEnter:
		if d.newName != "" {
			d.confirmed = true
		}
	case tea.KeyEsc:
		d.renaming = false
		d.newName = ""
	case tea.KeyBackspace:
		if len(d.newName) > 0 {
			d.newName = d.newName[:len(d.newName)-1]
		}
	case tea.KeyRunes:
		d.newName += string(msg.Runes)
	}
}

func (d *ImportDialog) getOptionCount() int {
	if d.hasConflict {
		return 3 // Rename, Skip, Replace
	}
	return 2 // Import, Cancel
}

// GetAction returns the current action based on selection.
func (d *ImportDialog) GetAction() ImportDialogAction {
	if d.hasConflict {
		switch d.selectedIdx {
		case 0:
			return ImportDialogRename
		case 1:
			return ImportDialogSkip
		case 2:
			return ImportDialogReplace
		}
	} else {
		switch d.selectedIdx {
		case 0:
			return ImportDialogImport
		case 1:
			return ImportDialogCancel
		}
	}
	return ImportDialogCancel
}

// IsConfirmed returns true if the user confirmed their selection.
func (d *ImportDialog) IsConfirmed() bool {
	return d.confirmed
}

// IsCancelled returns true if the user cancelled the dialog.
func (d *ImportDialog) IsCancelled() bool {
	return d.cancelled
}

// IsRenaming returns true if in rename input mode.
func (d *ImportDialog) IsRenaming() bool {
	return d.renaming
}

// GetNewName returns the new name entered in rename mode.
func (d *ImportDialog) GetNewName() string {
	return d.newName
}

// Reset clears the dialog state for reuse.
func (d *ImportDialog) Reset() {
	d.selectedIdx = 0
	d.confirmed = false
	d.cancelled = false
	d.renaming = false
	d.newName = ""
}

// View renders the dialog.
func (d *ImportDialog) View() string {
	// Determine dialog width
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
	warningColor := theme.Current.Warning

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginBottom(1)

	var title string
	if d.hasConflict {
		title = titleStyle.Render("Name Conflict")
	} else {
		title = titleStyle.Render("Import Skill")
	}

	// Skill info
	skillNameStyle := lipgloss.NewStyle().
		Foreground(goldColor).
		Bold(true)

	pathStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true)

	infoStyle := lipgloss.NewStyle().
		Foreground(textColor).
		Width(contentWidth).
		Align(lipgloss.Center)

	skillInfo := infoStyle.Render(skillNameStyle.Render(d.skillName))
	pathInfo := pathStyle.Render(d.skillPath)

	var parts []string
	parts = append(parts, title, skillInfo, pathInfo)

	// Conflict warning
	if d.hasConflict {
		warningStyle := lipgloss.NewStyle().
			Foreground(warningColor).
			Width(contentWidth).
			Align(lipgloss.Center).
			MarginTop(1)
		parts = append(parts, warningStyle.Render("A skill with this name already exists"))
	}

	// Rename input mode
	if d.renaming {
		parts = append(parts, d.renderRenameInput(contentWidth))
	} else {
		parts = append(parts, d.renderButtons(contentWidth))
	}

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginTop(1).
		Italic(true)

	var footerText string
	if d.renaming {
		footerText = "Enter confirm  |  Esc cancel"
	} else {
		footerText = "Left/Right navigate  |  Enter select  |  Esc cancel"
	}
	parts = append(parts, footerStyle.Render(footerText))

	// Dialog container
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Width(dialogWidth)

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)

	return dialogStyle.Render(content)
}

func (d *ImportDialog) renderButtons(contentWidth int) string {
	goldColor := theme.Current.Accent
	mutedColor := theme.Current.TextMuted
	textColor := theme.Current.Text
	selectedBgColor := theme.Current.Surface

	var options []string
	if d.hasConflict {
		options = []string{"Rename", "Skip", "Replace"}
	} else {
		options = []string{"Import", "Cancel"}
	}

	var buttons []string
	for i, opt := range options {
		isSelected := i == d.selectedIdx

		btnStyle := lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(textColor)

		if isSelected {
			btnStyle = btnStyle.
				Background(selectedBgColor).
				Foreground(goldColor).
				Bold(true)
		} else {
			btnStyle = btnStyle.Foreground(mutedColor)
		}

		indicator := "  "
		if isSelected {
			indicator = lipgloss.NewStyle().Foreground(goldColor).Render("> ")
		}

		buttons = append(buttons, indicator+btnStyle.Render(opt))
	}

	buttonRow := lipgloss.JoinHorizontal(lipgloss.Center, buttons...)

	containerStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginTop(1)

	return containerStyle.Render(buttonRow)
}

func (d *ImportDialog) renderRenameInput(contentWidth int) string {
	goldColor := theme.Current.Accent
	textColor := theme.Current.Text
	mutedColor := theme.Current.TextMuted

	labelStyle := lipgloss.NewStyle().
		Foreground(textColor).
		MarginTop(1)

	inputStyle := lipgloss.NewStyle().
		Foreground(goldColor).
		Bold(true).
		Border(lipgloss.NormalBorder()).
		BorderForeground(goldColor).
		Padding(0, 1).
		Width(contentWidth - 4)

	displayValue := d.newName
	if displayValue == "" {
		displayValue = lipgloss.NewStyle().Foreground(mutedColor).Italic(true).Render("Enter new name...")
	} else {
		displayValue += lipgloss.NewStyle().Foreground(goldColor).Render("_")
	}

	containerStyle := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center)

	return containerStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Center,
			labelStyle.Render("New name:"),
			inputStyle.Render(displayValue),
		),
	)
}

// CenteredView renders the dialog centered within the given dimensions.
func (d *ImportDialog) CenteredView(width, height int) string {
	dialog := d.View()
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, dialog)
}
