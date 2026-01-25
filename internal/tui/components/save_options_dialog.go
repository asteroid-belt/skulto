package components

import (
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SaveDestination represents where the skill should be saved.
type SaveDestination int

const (
	// SaveToDatabase saves the skill to the database only.
	SaveToDatabase SaveDestination = iota
	// SaveToFiles saves the skill to the local filesystem only.
	SaveToFiles
	// SaveToBoth saves to both database and filesystem.
	SaveToBoth
)

// SaveOption represents a single save destination option.
type SaveOption struct {
	Destination SaveDestination
	Title       string
	Description string
	Icon        string
}

// SaveOptionsDialog presents save destination choices with polished styling.
type SaveOptionsDialog struct {
	options       []SaveOption
	selectedIndex int
	width         int
	cancelled     bool
	confirmed     bool
}

// NewSaveOptionsDialog creates a new save options dialog.
func NewSaveOptionsDialog() *SaveOptionsDialog {
	return &SaveOptionsDialog{
		options: []SaveOption{
			{
				Destination: SaveToBoth,
				Title:       "Both",
				Description: "Save to database and local files (recommended)",
				Icon:        "󰣀",
			},
			{
				Destination: SaveToDatabase,
				Title:       "Database Only",
				Description: "Save to Skulto's internal database for quick access",
				Icon:        "󰆼",
			},
			{
				Destination: SaveToFiles,
				Title:       "Local Files Only",
				Description: "Save to .skulto/skills/ as markdown files",
				Icon:        "󰈙",
			},
		},
		selectedIndex: 0, // Default to "Both"
		cancelled:     false,
		confirmed:     false,
	}
}

// SetWidth sets the dialog width.
func (d *SaveOptionsDialog) SetWidth(w int) {
	d.width = w
}

// Update handles keyboard input for the dialog.
func (d *SaveOptionsDialog) Update(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyUp, tea.KeyShiftTab:
		d.selectedIndex--
		if d.selectedIndex < 0 {
			d.selectedIndex = len(d.options) - 1
		}
	case tea.KeyDown, tea.KeyTab:
		d.selectedIndex++
		if d.selectedIndex >= len(d.options) {
			d.selectedIndex = 0
		}
	case tea.KeyEnter:
		d.confirmed = true
	case tea.KeyEsc:
		d.cancelled = true
	default:
		// Handle vim-style navigation
		if msg.Type == tea.KeyRunes {
			switch string(msg.Runes) {
			case "j":
				d.selectedIndex++
				if d.selectedIndex >= len(d.options) {
					d.selectedIndex = 0
				}
			case "k":
				d.selectedIndex--
				if d.selectedIndex < 0 {
					d.selectedIndex = len(d.options) - 1
				}
			case "1":
				d.selectedIndex = 0
				d.confirmed = true
			case "2":
				d.selectedIndex = 1
				d.confirmed = true
			case "3":
				d.selectedIndex = 2
				d.confirmed = true
			}
		}
	}
}

// HandleKey processes string key for compatibility.
func (d *SaveOptionsDialog) HandleKey(key string) {
	switch key {
	case "up", "k":
		d.selectedIndex--
		if d.selectedIndex < 0 {
			d.selectedIndex = len(d.options) - 1
		}
	case "down", "j":
		d.selectedIndex++
		if d.selectedIndex >= len(d.options) {
			d.selectedIndex = 0
		}
	case "enter":
		d.confirmed = true
	case "esc":
		d.cancelled = true
	case "1":
		d.selectedIndex = 0
		d.confirmed = true
	case "2":
		d.selectedIndex = 1
		d.confirmed = true
	case "3":
		d.selectedIndex = 2
		d.confirmed = true
	}
}

// IsConfirmed returns true if the user confirmed their selection.
func (d *SaveOptionsDialog) IsConfirmed() bool {
	return d.confirmed
}

// IsCancelled returns true if the user cancelled the dialog.
func (d *SaveOptionsDialog) IsCancelled() bool {
	return d.cancelled
}

// GetSelection returns the selected save destination.
func (d *SaveOptionsDialog) GetSelection() SaveDestination {
	if d.selectedIndex >= 0 && d.selectedIndex < len(d.options) {
		return d.options[d.selectedIndex].Destination
	}
	return SaveToBoth
}

// Reset clears the dialog state for reuse.
func (d *SaveOptionsDialog) Reset() {
	d.selectedIndex = 0
	d.cancelled = false
	d.confirmed = false
}

// View renders the dialog.
func (d *SaveOptionsDialog) View() string {
	// Determine dialog width
	dialogWidth := d.width
	if dialogWidth == 0 || dialogWidth > 60 {
		dialogWidth = 60
	}
	if dialogWidth < 45 {
		dialogWidth = 45
	}
	contentWidth := dialogWidth - 6 // Account for border and padding

	// Colors from theme
	accentColor := theme.Current.Primary
	goldColor := theme.Current.Accent
	mutedColor := theme.Current.TextMuted
	textColor := theme.Current.Text
	successColor := theme.Current.Success
	selectedBgColor := theme.Current.Surface

	// Title style
	titleStyle := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginBottom(1)

	// Subtitle style
	subtitleStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Italic(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginBottom(1)

	// Render title section
	title := titleStyle.Render("Save Skill")
	subtitle := subtitleStyle.Render("Where would you like to save your skill?")

	// Render options
	var optionViews []string

	for i, opt := range d.options {
		isSelected := i == d.selectedIndex

		// Option container style
		optionStyle := lipgloss.NewStyle().
			Width(contentWidth).
			Padding(0, 1).
			MarginTop(0).
			MarginBottom(0)

		if isSelected {
			optionStyle = optionStyle.
				Background(selectedBgColor).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(goldColor)
		} else {
			optionStyle = optionStyle.
				Border(lipgloss.RoundedBorder()).
				BorderForeground(mutedColor)
		}

		// Number indicator
		numStyle := lipgloss.NewStyle().
			Foreground(mutedColor).
			Width(3)
		if isSelected {
			numStyle = numStyle.Foreground(goldColor).Bold(true)
		}
		num := numStyle.Render(string(rune('1'+i)) + ".")

		// Icon
		iconStyle := lipgloss.NewStyle().
			Width(3).
			Foreground(mutedColor)
		if isSelected {
			iconStyle = iconStyle.Foreground(accentColor)
		}
		icon := iconStyle.Render(opt.Icon)

		// Title
		titleTextStyle := lipgloss.NewStyle().
			Foreground(textColor).
			Bold(isSelected)
		if isSelected {
			titleTextStyle = titleTextStyle.Foreground(goldColor)
		}
		optTitle := titleTextStyle.Render(opt.Title)

		// Recommended badge for "Both"
		var badge string
		if opt.Destination == SaveToBoth {
			badgeStyle := lipgloss.NewStyle().
				Foreground(successColor).
				MarginLeft(1)
			badge = badgeStyle.Render("(recommended)")
		}

		// Description
		descStyle := lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Width(contentWidth - 8).
			PaddingLeft(6)
		desc := descStyle.Render(opt.Description)

		// Selection indicator
		indicator := "  "
		if isSelected {
			indicatorStyle := lipgloss.NewStyle().
				Foreground(goldColor).
				Bold(true)
			indicator = indicatorStyle.Render("▶ ")
		}

		// Compose option header line
		headerLine := lipgloss.JoinHorizontal(
			lipgloss.Left,
			indicator,
			num,
			icon,
			" ",
			optTitle,
			badge,
		)

		// Compose full option
		optionContent := lipgloss.JoinVertical(
			lipgloss.Left,
			headerLine,
			desc,
		)

		optionViews = append(optionViews, optionStyle.Render(optionContent))
	}

	// Join options
	options := lipgloss.JoinVertical(lipgloss.Left, optionViews...)

	// Footer with keyboard hints
	footerStyle := lipgloss.NewStyle().
		Foreground(mutedColor).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginTop(1).
		Italic(true)

	footer := footerStyle.Render("↑/↓ or j/k: navigate  •  Enter: confirm  •  Esc: cancel")

	// Main dialog container
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Width(dialogWidth)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		"",
		options,
		"",
		footer,
	)

	return dialogStyle.Render(content)
}

// CenteredView renders the dialog centered within the given dimensions.
func (d *SaveOptionsDialog) CenteredView(width, height int) string {
	dialog := d.View()

	dialogHeight := lipgloss.Height(dialog)
	dialogWidth := lipgloss.Width(dialog)

	// Calculate centering offsets
	topPadding := max(0, (height-dialogHeight)/2)
	leftPadding := max(0, (width-dialogWidth)/2)

	// Create dimmed background
	bgStyle := lipgloss.NewStyle().
		Width(width).
		Height(height).
		Background(theme.Current.Background)

	// Create the centered dialog
	centeredStyle := lipgloss.NewStyle().
		PaddingTop(topPadding).
		PaddingLeft(leftPadding)

	return bgStyle.Render(centeredStyle.Render(dialog))
}

// OverlayView renders the dialog as an overlay on top of existing content.
func (d *SaveOptionsDialog) OverlayView(background string, width, height int) string {
	dialog := d.View()

	// Use lipgloss.Place to center the dialog over a blank canvas
	// This properly handles ANSI escape codes and visual width
	centered := lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)

	return centered
}
