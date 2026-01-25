package components

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/asteroid-belt/skulto/internal/models"
)

// RepoOption represents a single repository option.
type RepoOption struct {
	Source            *models.Source
	Title             string
	Description       string
	InstalledCount    int
	NotInstalledCount int
}

// RepoSelectDialog presents repository choices for selection.
type RepoSelectDialog struct {
	options       []RepoOption
	selectedIndex int
	width         int
	cancelled     bool
	confirmed     bool
}

// NewRepoSelectDialog creates a new repository selection dialog.
func NewRepoSelectDialog(sources []models.Source) *RepoSelectDialog {
	options := make([]RepoOption, len(sources))
	for i := range sources {
		options[i] = RepoOption{
			Source:      &sources[i],
			Title:       sources[i].ID,
			Description: fmt.Sprintf("%d skills", sources[i].SkillCount),
		}
	}

	return &RepoSelectDialog{
		options:       options,
		selectedIndex: 0,
		cancelled:     false,
		confirmed:     false,
	}
}

// NewRepoSelectDialogWithOptions creates a dialog with pre-built options.
func NewRepoSelectDialogWithOptions(options []RepoOption) *RepoSelectDialog {
	return &RepoSelectDialog{
		options:       options,
		selectedIndex: 0,
		cancelled:     false,
		confirmed:     false,
	}
}

// SetWidth sets the dialog width.
func (d *RepoSelectDialog) SetWidth(w int) {
	d.width = w
}

// Update handles keyboard input for the dialog.
func (d *RepoSelectDialog) Update(msg tea.KeyMsg) {
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
		// Handle vim-style navigation and number shortcuts
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
			default:
				// Number shortcuts (1-9)
				if len(msg.Runes) == 1 {
					r := msg.Runes[0]
					if r >= '1' && r <= '9' {
						idx := int(r - '1')
						if idx < len(d.options) {
							d.selectedIndex = idx
							d.confirmed = true
						}
					}
				}
			}
		}
	}
}

// HandleKey processes string key for compatibility.
func (d *RepoSelectDialog) HandleKey(key string) {
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
	}
}

// IsConfirmed returns true if the user confirmed their selection.
func (d *RepoSelectDialog) IsConfirmed() bool {
	return d.confirmed
}

// IsCancelled returns true if the user cancelled the dialog.
func (d *RepoSelectDialog) IsCancelled() bool {
	return d.cancelled
}

// GetSelection returns the selected source.
func (d *RepoSelectDialog) GetSelection() *models.Source {
	if d.selectedIndex >= 0 && d.selectedIndex < len(d.options) {
		return d.options[d.selectedIndex].Source
	}
	return nil
}

// Reset clears the dialog state for reuse.
func (d *RepoSelectDialog) Reset() {
	d.selectedIndex = 0
	d.cancelled = false
	d.confirmed = false
}

// View renders the dialog.
func (d *RepoSelectDialog) View() string {
	// Determine dialog width
	dialogWidth := d.width
	if dialogWidth == 0 || dialogWidth > 70 {
		dialogWidth = 70
	}
	if dialogWidth < 50 {
		dialogWidth = 50
	}
	contentWidth := dialogWidth - 6

	// Colors from design system
	goldColor := lipgloss.Color("#F1C40F")
	mutedColor := lipgloss.Color("#6B6B6B")
	textColor := lipgloss.Color("#E5E5E5")
	warningColor := lipgloss.Color("#E74C3C")
	selectedBgColor := lipgloss.Color("#1A1A2E")

	// Title style
	titleStyle := lipgloss.NewStyle().
		Foreground(warningColor).
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
	title := titleStyle.Render("Remove Repository")
	subtitle := subtitleStyle.Render("Select a repository to remove")

	// Render options
	var optionViews []string

	maxVisible := 8
	startIdx := 0
	if d.selectedIndex >= maxVisible {
		startIdx = d.selectedIndex - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(d.options) {
		endIdx = len(d.options)
	}

	for i := startIdx; i < endIdx; i++ {
		opt := d.options[i]
		isSelected := i == d.selectedIndex

		// Option container style
		optionStyle := lipgloss.NewStyle().
			Width(contentWidth).
			Padding(0, 1)

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
		num := numStyle.Render(fmt.Sprintf("%d.", i+1))

		// Title
		titleTextStyle := lipgloss.NewStyle().
			Foreground(textColor).
			Bold(isSelected)
		if isSelected {
			titleTextStyle = titleTextStyle.Foreground(goldColor)
		}
		optTitle := titleTextStyle.Render(opt.Title)

		// Description with installed/not-installed counts
		var desc string
		if opt.InstalledCount > 0 || opt.NotInstalledCount > 0 {
			// Show detailed counts with colors
			installedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")) // Green
			notInstalledStyle := lipgloss.NewStyle().Foreground(mutedColor)

			installedText := installedStyle.Render(fmt.Sprintf("%d installed", opt.InstalledCount))
			notInstalledText := notInstalledStyle.Render(fmt.Sprintf("%d not installed", opt.NotInstalledCount))

			descStyle := lipgloss.NewStyle().MarginLeft(1)
			desc = descStyle.Render(fmt.Sprintf("(%s, %s)", installedText, notInstalledText))
		} else {
			// Fallback to simple description
			descStyle := lipgloss.NewStyle().
				Foreground(mutedColor).
				Italic(true).
				MarginLeft(1)
			desc = descStyle.Render(fmt.Sprintf("(%s)", opt.Description))
		}

		// Selection indicator
		indicator := "  "
		if isSelected {
			indicatorStyle := lipgloss.NewStyle().
				Foreground(goldColor).
				Bold(true)
			indicator = indicatorStyle.Render("> ")
		}

		// Compose option line
		headerLine := lipgloss.JoinHorizontal(
			lipgloss.Left,
			indicator,
			num,
			" ",
			optTitle,
			desc,
		)

		optionViews = append(optionViews, optionStyle.Render(headerLine))
	}

	// Show scroll indicator if needed
	if len(d.options) > maxVisible {
		scrollInfo := lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true).
			Width(contentWidth).
			Align(lipgloss.Center).
			Render(fmt.Sprintf("Showing %d-%d of %d", startIdx+1, endIdx, len(d.options)))
		optionViews = append(optionViews, scrollInfo)
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

	footer := footerStyle.Render("up/down or j/k: navigate  |  Enter: confirm  |  Esc: cancel")

	// Warning message
	warningStyle := lipgloss.NewStyle().
		Foreground(warningColor).
		Width(contentWidth).
		Align(lipgloss.Center).
		MarginTop(1).
		Bold(true)
	warning := warningStyle.Render("! This action cannot be undone")

	// Main dialog container
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(warningColor).
		Padding(1, 2).
		Width(dialogWidth)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		"",
		options,
		warning,
		"",
		footer,
	)

	return dialogStyle.Render(content)
}

// OverlayView renders the dialog centered over existing content.
func (d *RepoSelectDialog) OverlayView(background string, width, height int) string {
	dialog := d.View()

	centered := lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)

	return centered
}
