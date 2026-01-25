package components

import (
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SearchBar is an input component for searching skills.
type SearchBar struct {
	input textinput.Model
}

// NewSearchBar creates a new search bar component.
func NewSearchBar() *SearchBar {
	ti := textinput.New()
	ti.Placeholder = "Start typing..."
	ti.CharLimit = 100
	ti.Width = 50

	// Style the input with theme colors
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Current.Text)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Current.TextMuted)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(theme.Current.Accent)
	ti.PromptStyle = lipgloss.NewStyle().Foreground(theme.Current.Primary)

	return &SearchBar{
		input: ti,
	}
}

// Init initializes the search bar (enables focus).
func (sb *SearchBar) Init() textinput.Model {
	sb.input.Focus()
	return sb.input
}

// Update handles messages for the search bar.
func (sb *SearchBar) Update(msg interface{}) (string, bool) {
	var cmd textinput.Model
	var ok bool

	if cmd, ok = msg.(textinput.Model); ok {
		sb.input = cmd
	}

	return sb.input.Value(), true
}

// View renders the search bar.
func (sb *SearchBar) View() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Current.Accent).
		Padding(0, 1)

	input := boxStyle.Render(sb.input.View())
	return input
}

// HandleKey passes a key message to the underlying textinput.
// This enables full cursor support including left/right navigation,
// home/end, and proper backspace/delete at cursor position.
// Returns a tea.Cmd that MUST be executed by the parent for cursor blink.
func (sb *SearchBar) HandleKey(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	sb.input, cmd = sb.input.Update(msg)
	return cmd
}

// Focus sets focus on the search bar.
func (sb *SearchBar) Focus() {
	sb.input.Focus()
}

// Blur removes focus from the search bar.
func (sb *SearchBar) Blur() {
	sb.input.Blur()
}

// Focused returns true if the search bar has focus.
func (sb *SearchBar) Focused() bool {
	return sb.input.Focused()
}

// Value returns the current search query.
func (sb *SearchBar) Value() string {
	return sb.input.Value()
}

// SetValue sets the search query.
func (sb *SearchBar) SetValue(v string) {
	sb.input.SetValue(v)
}

// Clear clears the search query.
func (sb *SearchBar) Clear() {
	sb.input.Reset()
}

// Model returns the underlying textinput model for tea.Update.
func (sb *SearchBar) Model() textinput.Model {
	return sb.input
}

// SetModel updates the underlying textinput model.
func (sb *SearchBar) SetModel(m textinput.Model) {
	sb.input = m
}

// SetWidth sets the width of the search bar.
func (sb *SearchBar) SetWidth(w int) {
	// Reduce width for padding and borders
	if w > 4 {
		sb.input.Width = w - 4
	}
}

// AppendValue appends a string to the search query.
func (sb *SearchBar) AppendValue(s string) {
	current := sb.input.Value()
	sb.input.SetValue(current + s)
}

// DeleteChar removes the last character from the search query.
func (sb *SearchBar) DeleteChar() {
	current := sb.input.Value()
	if len(current) > 0 {
		// Handle multi-byte UTF-8 characters properly
		runes := []rune(current)
		sb.input.SetValue(string(runes[:len(runes)-1]))
	}
}
