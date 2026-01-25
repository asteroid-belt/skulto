package views

import (
	"fmt"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/scraper"
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AddSourceView displays a form for adding a new source repository.
type AddSourceView struct {
	db      *db.DB
	cfg     *config.Config
	input   textinput.Model
	repoURL string // Stores the submitted repository URL
	error   string // Validation error if any
	width   int
	height  int
}

// NewAddSourceView creates a new add source view.
func NewAddSourceView(database *db.DB, conf *config.Config) *AddSourceView {
	ti := textinput.New()
	ti.Placeholder = "owner/repo"
	ti.CharLimit = 100
	ti.Width = 40

	// Style the input
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Current.Text)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Current.TextMuted)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(theme.Current.Accent)
	ti.PromptStyle = lipgloss.NewStyle().Foreground(theme.Current.Primary)

	return &AddSourceView{
		db:    database,
		cfg:   conf,
		input: ti,
	}
}

// Init initializes the view and focuses the input.
func (asv *AddSourceView) Init() {
	asv.input.Reset()
	asv.input.Focus()
	asv.repoURL = ""
	asv.error = ""
}

// SetSize sets the width and height of the view.
func (asv *AddSourceView) SetSize(width, height int) {
	asv.width = width
	asv.height = height
}

// Update handles key input. Returns (shouldGoBack, wasSuccessful).
func (asv *AddSourceView) Update(key string) (bool, bool) {
	switch key {
	case "esc":
		return true, false

	case "enter":
		repoURL := asv.input.Value()
		if repoURL == "" {
			asv.error = "Repository URL cannot be empty"
			return false, false
		}

		// Validate the repository URL
		_, err := scraper.ParseRepositoryURL(repoURL)
		if err != nil {
			asv.error = fmt.Sprintf("Invalid repository URL: %v", err)
			return false, false
		}

		asv.repoURL = repoURL
		return true, true

	default:
		// Convert string key to tea.KeyMsg for textinput
		keyMsg := asv.stringToKeyMsg(key)

		// Pass to textinput for proper cursor handling
		asv.input, _ = asv.input.Update(keyMsg)

		// Clear error when user starts typing
		if asv.error != "" {
			asv.error = ""
		}
		return false, false
	}
}

// stringToKeyMsg converts a string key representation to a tea.KeyMsg.
func (asv *AddSourceView) stringToKeyMsg(key string) tea.KeyMsg {
	switch key {
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "home":
		return tea.KeyMsg{Type: tea.KeyHome}
	case "end":
		return tea.KeyMsg{Type: tea.KeyEnd}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "delete":
		return tea.KeyMsg{Type: tea.KeyDelete}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "space":
		return tea.KeyMsg{Type: tea.KeySpace}
	default:
		// For regular characters, create a Runes message
		if len(key) == 1 {
			return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		}
		// Unknown key, return empty
		return tea.KeyMsg{}
	}
}

// GetRepositoryURL returns the validated repository URL.
func (asv *AddSourceView) GetRepositoryURL() string {
	return asv.repoURL
}

// View renders the add source view.
func (asv *AddSourceView) View() string {
	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Primary).
		Bold(true).
		MarginBottom(1)
	title := titleStyle.Render("Add Source Repository")

	// Input label
	labelStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Text).
		MarginBottom(1)
	label := labelStyle.Render("Repository URL (owner/repo):")

	// Input field
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Current.Accent).
		Padding(0, 1)
	inputField := inputStyle.Render(asv.input.View())

	// Example text
	exampleStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		Italic(true).
		MarginBottom(1)
	example := exampleStyle.Render("Example: anthropics/anthropic-cookbook")

	// Warning message
	warningStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Warning).
		MarginTop(1).
		MarginBottom(1)
	warning := warningStyle.Render(
		"⚠ For private repositories, set GITHUB_TOKEN environment variable",
	)

	// Build content parts
	contentParts := []string{
		title,
		"",
		label,
		inputField,
		example,
		warning,
	}

	// Add error message if present
	if asv.error != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(theme.Current.Error).
			MarginTop(1).
			Bold(true)
		contentParts = append(contentParts, errorStyle.Render("✗ "+asv.error))
	}

	// Buttons
	confirmStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Background).
		Background(theme.Current.Success).
		Padding(0, 2).
		Bold(true)
	cancelStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Background).
		Background(theme.Current.Accent).
		Padding(0, 2).
		Bold(true)

	buttons := lipgloss.JoinHorizontal(
		lipgloss.Center,
		"[ ",
		confirmStyle.Render("Enter"),
		" ] [ ",
		cancelStyle.Render("Esc"),
		" ]",
	)

	buttonStyle := lipgloss.NewStyle().
		MarginTop(1)

	buttonsRendered := buttonStyle.Render(buttons)
	contentParts = append(contentParts, "", buttonsRendered)

	// Combine content
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		contentParts...,
	)

	// Create bordered dialog
	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Current.Primary).
		Padding(2, 3)

	renderedDialog := dialog.Render(content)

	// Center the dialog
	dialogWidth := lipgloss.Width(renderedDialog)
	paddingLeft := (asv.width - dialogWidth) / 2
	if paddingLeft < 0 {
		paddingLeft = 0
	}

	paddingTop := (asv.height - 20) / 2
	if paddingTop < 1 {
		paddingTop = 1
	}

	return lipgloss.NewStyle().
		Padding(paddingTop, paddingLeft).
		Render(renderedDialog)
}

// GetKeyboardCommands returns the keyboard commands for this view.
func (asv *AddSourceView) GetKeyboardCommands() ViewCommands {
	return ViewCommands{
		ViewName: "Add Source",
		Commands: []Command{
			{Key: "Text input", Description: "Enter repository URL (owner/repo format)"},
			{Key: "Enter", Description: "Submit and add repository"},
			{Key: "Esc", Description: "Cancel and return to home"},
		},
	}
}
