package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles contains all reusable Lipgloss styles for the TUI.
type Styles struct {
	// Container styles
	Container   lipgloss.Style
	Box         lipgloss.Style
	SelectedBox lipgloss.Style

	// Header styles
	Header        lipgloss.Style
	HeaderTitle   lipgloss.Style
	HeaderVersion lipgloss.Style

	// Search styles
	SearchBox         lipgloss.Style
	SearchInput       lipgloss.Style
	SearchPlaceholder lipgloss.Style

	// List styles
	List             lipgloss.Style
	ListItem         lipgloss.Style
	ListItemSelected lipgloss.Style
	ListItemFocused  lipgloss.Style

	// Footer styles
	Footer       lipgloss.Style
	FooterLeft   lipgloss.Style
	FooterCenter lipgloss.Style
	FooterRight  lipgloss.Style

	// Text styles
	Normal    lipgloss.Style
	Muted     lipgloss.Style
	Bold      lipgloss.Style
	Highlight lipgloss.Style

	// Status indicators
	StatusOK      lipgloss.Style
	StatusWarning lipgloss.Style
	StatusError   lipgloss.Style
	StatusInfo    lipgloss.Style

	// Tag styles
	Tag          lipgloss.Style
	TagLanguage  lipgloss.Style
	TagFramework lipgloss.Style
	TagTool      lipgloss.Style
	TagConcept   lipgloss.Style

	// Skill item styles
	SkillTitle lipgloss.Style
	SkillDesc  lipgloss.Style
	SkillMeta  lipgloss.Style
}

// DefaultStyles returns the default Lipgloss styles using the current theme.
func DefaultStyles() Styles {
	theme := CurrentTheme

	return Styles{
		// Container styles
		Container: lipgloss.NewStyle().
			Background(theme.Background),

		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Surface).
			Padding(1).
			Background(theme.Surface),

		SelectedBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Primary).
			Padding(1).
			Background(theme.Surface),

		// Header styles
		Header: lipgloss.NewStyle().
			Background(theme.Background).
			Padding(1),

		HeaderTitle: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			Width(20).
			AlignHorizontal(lipgloss.Center),

		HeaderVersion: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Italic(true),

		// Search styles
		SearchBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Accent).
			Padding(0, 1).
			Background(theme.Surface),

		SearchInput: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.Surface),

		SearchPlaceholder: lipgloss.NewStyle().
			Foreground(theme.TextMuted),

		// List styles
		List: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Surface).
			Padding(1).
			Background(theme.Surface),

		ListItem: lipgloss.NewStyle().
			Foreground(theme.Text).
			Padding(0, 1),

		ListItemSelected: lipgloss.NewStyle().
			Foreground(theme.TextHighlight).
			Background(theme.Overlay).
			Padding(0, 1).
			Bold(true),

		ListItemFocused: lipgloss.NewStyle().
			Foreground(theme.Accent).
			Padding(0, 1).
			Bold(true),

		// Footer styles
		Footer: lipgloss.NewStyle().
			Background(theme.Surface).
			Foreground(theme.TextMuted).
			Padding(0, 1),

		FooterLeft: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Align(lipgloss.Left),

		FooterCenter: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Align(lipgloss.Center),

		FooterRight: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Align(lipgloss.Right),

		// Text styles
		Normal: lipgloss.NewStyle().
			Foreground(theme.Text),

		Muted: lipgloss.NewStyle().
			Foreground(theme.TextMuted),

		Bold: lipgloss.NewStyle().
			Foreground(theme.Text).
			Bold(true),

		Highlight: lipgloss.NewStyle().
			Foreground(theme.Accent).
			Bold(true),

		// Status indicators
		StatusOK: lipgloss.NewStyle().
			Foreground(theme.Success).
			Bold(true),

		StatusWarning: lipgloss.NewStyle().
			Foreground(theme.Warning).
			Bold(true),

		StatusError: lipgloss.NewStyle().
			Foreground(theme.Error).
			Bold(true),

		StatusInfo: lipgloss.NewStyle().
			Foreground(theme.Info).
			Bold(true),

		// Tag styles
		Tag: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.Overlay).
			Padding(0, 1).
			Margin(0, 1, 0, 0),

		TagLanguage: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.TagLanguage).
			Padding(0, 1).
			Margin(0, 1, 0, 0),

		TagFramework: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.TagFramework).
			Padding(0, 1).
			Margin(0, 1, 0, 0),

		TagTool: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.TagTool).
			Padding(0, 1).
			Margin(0, 1, 0, 0),

		TagConcept: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.TagConcept).
			Padding(0, 1).
			Margin(0, 1, 0, 0),

		// Skill item styles
		SkillTitle: lipgloss.NewStyle().
			Foreground(theme.TextHighlight).
			Bold(true),

		SkillDesc: lipgloss.NewStyle().
			Foreground(theme.Text),

		SkillMeta: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Italic(true),
	}
}
