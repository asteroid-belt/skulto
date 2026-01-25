package components

import (
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmDialog is a simple yes/no confirmation dialog.
type ConfirmDialog struct {
	title    string
	message  string
	selected bool // false = no, true = yes
}

// NewConfirmDialog creates a new confirmation dialog.
func NewConfirmDialog(title, message string) *ConfirmDialog {
	return &ConfirmDialog{
		title:    title,
		message:  message,
		selected: false, // Default to "No"
	}
}

// SelectYes selects the "Yes" option.
func (c *ConfirmDialog) SelectYes() {
	c.selected = true
}

// SelectNo selects the "No" option.
func (c *ConfirmDialog) SelectNo() {
	c.selected = false
}

// IsYesSelected returns whether "Yes" is selected.
func (c *ConfirmDialog) IsYesSelected() bool {
	return c.selected
}

// Toggle switches between Yes and No.
func (c *ConfirmDialog) Toggle() {
	c.selected = !c.selected
}

// View renders the confirmation dialog.
func (c *ConfirmDialog) View() string {
	yesStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		Padding(0, 2)

	noStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		Padding(0, 2)

	if c.selected {
		yesStyle = yesStyle.
			Background(theme.Current.Accent).
			Foreground(theme.Current.Background).
			Bold(true)
	} else {
		noStyle = noStyle.
			Background(theme.Current.Accent).
			Foreground(theme.Current.Background).
			Bold(true)
	}

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Left,
		"[ ",
		yesStyle.Render("Yes"),
		" ] [ ",
		noStyle.Render("No"),
		" ]",
	)

	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Current.Primary).
		Padding(1, 2).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Center,
				lipgloss.NewStyle().Bold(true).Foreground(theme.Current.Text).Render(c.title),
				"",
				c.message,
				"",
				buttons,
			),
		)

	return dialog
}

// CenteredView renders the dialog centered on the screen.
func (c *ConfirmDialog) CenteredView(width, height int) string {
	dialog := c.View()
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, dialog)
}
