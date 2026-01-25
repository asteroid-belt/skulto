package views

import (
	"strings"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/telemetry"
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// HelpView displays a list of available commands and keybindings.
type HelpView struct {
	db           *db.DB
	cfg          *config.Config
	width        int
	height       int
	viewCommands ViewCommands
	telemetry    telemetry.Client
}

// Command represents a single keyboard command.
type Command struct {
	Key         string
	Description string
}

// ViewCommands represents commands for a specific view.
type ViewCommands struct {
	ViewName string
	Commands []Command
}

// NewHelpView creates a new help view.
func NewHelpView(database *db.DB, conf *config.Config) *HelpView {
	return &HelpView{
		db:  database,
		cfg: conf,
	}
}

// Init initializes the view and sets the telemetry client.
func (hv *HelpView) Init(tc telemetry.Client) {
	hv.telemetry = tc
}

// SetSize sets the width and height of the view.
func (hv *HelpView) SetSize(width, height int) {
	hv.width = width
	hv.height = height
}

// SetViewCommands sets the commands from the calling view.
func (hv *HelpView) SetViewCommands(commands ViewCommands) {
	hv.viewCommands = commands
	hv.telemetry.TrackHelpViewed(commands.ViewName)
}

// Update handles key input. Returns (shouldGoBack, unused).
func (hv *HelpView) Update(key string) (bool, bool) {
	switch key {
	case "esc", "?", "q":
		return true, false
	default:
		return false, false
	}
}

// View renders the help view.
func (hv *HelpView) View() string {
	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Accent).
		Bold(true).
		MarginLeft(1).
		MarginTop(1).
		MarginBottom(1)

	title := titleStyle.Render("Help - Available Commands")

	// Get commands
	globalCommands, viewCommands, viewTitle := hv.getCommands()

	// Section header style
	sectionHeaderStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Primary).
		Bold(true).
		MarginLeft(1).
		MarginTop(1)

	// Render global commands section
	globalHeader := sectionHeaderStyle.Render("Global Commands")
	globalTable := hv.renderCommandTable(globalCommands)

	// Render view-specific commands section
	viewHeader := sectionHeaderStyle.Render(viewTitle + " Commands")
	viewTable := hv.renderCommandTable(viewCommands)

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		Italic(true).
		MarginTop(1).
		MarginLeft(1)

	footer := footerStyle.Render("Press Esc, ?, or q to close")

	// Combine all sections
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		globalHeader,
		globalTable,
		"",
		viewHeader,
		viewTable,
		"",
		footer,
	)

	// Add padding
	paddedContent := lipgloss.NewStyle().
		Padding(1, 2).
		Render(content)

	return paddedContent
}

// getCommands returns global and view-specific commands.
func (hv *HelpView) getCommands() (globalCommands []Command, viewCommands []Command, viewTitle string) {
	// Global commands (work everywhere)
	globalCommands = []Command{
		{Key: "q, ctrl+c", Description: "Quit application"},
		{Key: "ctrl+r", Description: "Reset database (with confirmation)"},
		{Key: "?", Description: "Show this help screen"},
	}

	// Use commands passed from the view
	viewCommands = hv.viewCommands.Commands
	viewTitle = hv.viewCommands.ViewName

	return globalCommands, viewCommands, viewTitle
}

// renderCommandTable renders the commands as a formatted table.
func (hv *HelpView) renderCommandTable(commands []Command) string {
	if len(commands) == 0 {
		return ""
	}

	// Calculate column widths
	maxKeyLen := 0
	for _, cmd := range commands {
		if len(cmd.Key) > maxKeyLen {
			maxKeyLen = len(cmd.Key)
		}
	}

	// Add some padding
	keyColWidth := maxKeyLen + 2
	descColWidth := hv.width - keyColWidth - 6 // Account for borders and padding

	if descColWidth < 20 {
		descColWidth = 20
	}

	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Primary).
		Bold(true).
		Padding(0, 1)

	keyHeader := headerStyle.Render("Key")
	descHeader := headerStyle.Render("Description")

	headerLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		keyHeader,
		"  ",
		descHeader,
	)

	// Separator
	separatorStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted)

	separator := separatorStyle.Render(strings.Repeat("─", hv.width-4))

	// Rows
	var rows []string
	rows = append(rows, headerLine)
	rows = append(rows, separator)

	keyStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Accent).
		Bold(true).
		Padding(0, 1).
		Width(keyColWidth)

	descStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Text).
		Padding(0, 1).
		Width(descColWidth)

	for _, cmd := range commands {
		keyCell := keyStyle.Render(cmd.Key)
		descCell := descStyle.Render(cmd.Description)

		row := lipgloss.JoinHorizontal(
			lipgloss.Left,
			keyCell,
			descCell,
		)

		rows = append(rows, row)
	}

	return strings.Join(rows, "\n")
}
