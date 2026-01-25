package views

import (
	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// OnboardingIntroView displays the onboarding introduction.
type OnboardingIntroView struct {
	cfg    *config.Config
	width  int
	height int
}

// NewOnboardingIntroView creates a new onboarding intro view.
func NewOnboardingIntroView(conf *config.Config) *OnboardingIntroView {
	return &OnboardingIntroView{
		cfg: conf,
	}
}

// Init initializes the view.
func (v *OnboardingIntroView) Init() {
	// No state to reset
}

// SetSize sets the width and height of the view.
func (v *OnboardingIntroView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Update handles key input. Returns (should continue, was skipped).
func (v *OnboardingIntroView) Update(key string) (bool, bool) {
	switch key {
	case "enter":
		// Continue to next phase
		return true, false
	case "esc":
		// Skip onboarding
		return true, true
	}
	return false, false
}

// View renders the onboarding intro view.
func (v *OnboardingIntroView) View() string {
	// Title with punk styling (bold, crimson red)
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current.Primary).
		MarginBottom(1)

	title := titleStyle.Render("🎸 Welcome to Skulto")

	// Subtitle
	subtitleStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		MarginBottom(2)

	subtitle := subtitleStyle.Render("Your personal AI developer tool skill library")

	// Description sections with consistent styling
	descStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Text)

	features := descStyle.Render(
		"Skulto is a powerful skill management system that helps you:\n" +
			"\n" +
			"• Scrape and index AI developer tool skills from GitHub\n" +
			"• Search and browse thousands of community-curated skills\n" +
			"• Translate skills to 6 AI platforms:\n" +
			"  - Claude Code (.claude/)\n" +
			"  - Cursor (.cursor/)\n" +
			"  - GitHub Copilot (.github/)\n" +
			"  - OpenAI Codex (.codex/)\n" +
			"  - OpenCode (.opencode/)\n" +
			"  - Windsurf (.windsurf/)\n" +
			"• Sync skills across your developer tools\n" +
			"• Keep your skill library in sync with seed repositories",
	)

	// Instructions
	instructionStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Accent).
		MarginTop(2).
		MarginBottom(1)

	instructions := instructionStyle.Render(
		"Let's set up your Skulto experience!\n\n" +
			"Press Enter to continue or Esc to skip onboarding",
	)

	// Combine content
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		features,
		instructions,
	)

	// Create bordered dialog with responsive width
	maxWidth := v.width * 80 / 100
	if maxWidth < 50 {
		maxWidth = 50
	}

	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Current.Primary).
		Padding(2, 3).
		MaxWidth(maxWidth)

	renderedDialog := dialog.Render(content)

	// Center the dialog both vertically and horizontally
	dialogWidth := lipgloss.Width(renderedDialog)
	paddingLeft := (v.width - dialogWidth) / 2
	if paddingLeft < 0 {
		paddingLeft = 0
	}

	paddingTop := (v.height - 20) / 2
	if paddingTop < 1 {
		paddingTop = 1
	}

	return lipgloss.NewStyle().
		Padding(paddingTop, paddingLeft).
		Render(renderedDialog)
}

// GetKeyboardCommands returns the keyboard commands for this view.
func (ov *OnboardingIntroView) GetKeyboardCommands() ViewCommands {
	return ViewCommands{
		ViewName: "Onboarding",
		Commands: []Command{
			{Key: "Enter", Description: "Continue to next step"},
			{Key: "Esc", Description: "Skip onboarding"},
		},
	}
}
