package views

import (
	"os"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/charmbracelet/lipgloss"
)

// OnboardingSetupView displays environment variable setup guidance.
type OnboardingSetupView struct {
	cfg    *config.Config
	width  int
	height int
}

// NewOnboardingSetupView creates a new onboarding setup view.
func NewOnboardingSetupView(conf *config.Config) *OnboardingSetupView {
	return &OnboardingSetupView{
		cfg: conf,
	}
}

// Init initializes the view.
func (v *OnboardingSetupView) Init() {
	// No state to reset
}

// SetSize sets the width and height of the view.
func (v *OnboardingSetupView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Update handles key input. Returns (should continue, was skipped).
func (v *OnboardingSetupView) Update(key string) (bool, bool) {
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

// View renders the onboarding setup view.
func (v *OnboardingSetupView) View() string {
	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		MarginBottom(1)

	title := titleStyle.Render("⚙️  Environment Setup")

	// Subtitle
	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		MarginBottom(2)

	subtitle := subtitleStyle.Render("Configure API access for enhanced functionality")

	// Environment variable status
	githubStatus := v.checkEnvVar("GITHUB_TOKEN")
	openaiStatus := v.checkEnvVar("OPENAI_API_KEY")

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250"))

	statusContent := "Environment Variables:\n\n"

	// GitHub Token status
	if githubStatus {
		statusContent += "✅ GITHUB_TOKEN is set\n"
	} else {
		statusContent += "❌ GITHUB_TOKEN is not set\n"
	}

	// OpenAI API Key status
	if openaiStatus {
		statusContent += "✅ OPENAI_API_KEY is set\n"
	} else {
		statusContent += "❌ OPENAI_API_KEY is not set\n"
	}

	statusContent += "\n"

	status := statusStyle.Render(statusContent)

	// Instructions
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	instructions := instructionStyle.Render(
		"GITHUB_TOKEN (required for higher rate limits):\n" +
			"  export GITHUB_TOKEN=\"your_token_here\"\n" +
			"  Get token at: https://github.com/settings/tokens\n\n" +
			"OPENAI_API_KEY (required for semantic search indexing):\n" +
			"  export OPENAI_API_KEY=\"your_api_key_here\"\n" +
			"  Get key at: https://platform.openai.com/api-keys\n\n" +
			"Note: You can continue without these, but with limited features.\n" +
			"You can set them later in your shell configuration.",
	)

	// Combine content
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		status,
		instructions,
	)

	// Create bordered dialog with responsive width
	maxWidth := v.width * 80 / 100
	if maxWidth < 50 {
		maxWidth = 50
	}

	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(2, 3).
		MaxWidth(maxWidth)

	renderedDialog := dialog.Render(content)

	// Footer with instructions
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		MarginTop(2)

	footer := footerStyle.Render("Press Enter to continue or Esc to skip")

	// Center the dialog both vertically and horizontally
	dialogWidth := lipgloss.Width(renderedDialog)
	paddingLeft := (v.width - dialogWidth) / 2
	if paddingLeft < 0 {
		paddingLeft = 0
	}

	paddingTop := (v.height - 24) / 2
	if paddingTop < 1 {
		paddingTop = 1
	}

	return lipgloss.NewStyle().
		Padding(paddingTop, paddingLeft).
		Render(lipgloss.JoinVertical(
			lipgloss.Center,
			renderedDialog,
			footer,
		))
}

// checkEnvVar checks if an environment variable is set.
func (v *OnboardingSetupView) checkEnvVar(name string) bool {
	_, exists := os.LookupEnv(name)
	return exists
}

// GetKeyboardCommands returns the keyboard commands for this view.
func (ov *OnboardingSetupView) GetKeyboardCommands() ViewCommands {
	return ViewCommands{
		ViewName: "Onboarding",
		Commands: []Command{
			{Key: "Enter", Description: "Continue to tool selection"},
			{Key: "Esc", Description: "Skip onboarding"},
		},
	}
}
