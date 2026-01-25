package components

import (
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
		Foreground(lipgloss.Color("240")).
		Padding(0, 2)

	noStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 2)

	if c.selected {
		yesStyle = yesStyle.
			Background(lipgloss.Color("220")).
			Foreground(lipgloss.Color("0")).
			Bold(true)
	} else {
		noStyle = noStyle.
			Background(lipgloss.Color("220")).
			Foreground(lipgloss.Color("0")).
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
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Center,
				lipgloss.NewStyle().Bold(true).Render(c.title),
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
