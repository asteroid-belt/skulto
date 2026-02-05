package views

import (
	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// OnboardingSkillsIntroView displays the "What are Agent Skills?" onboarding screen.
type OnboardingSkillsIntroView struct {
	cfg    *config.Config
	width  int
	height int
}

// NewOnboardingSkillsIntroView creates a new onboarding skills intro view.
func NewOnboardingSkillsIntroView(conf *config.Config) *OnboardingSkillsIntroView {
	return &OnboardingSkillsIntroView{
		cfg: conf,
	}
}

// Init initializes the view.
func (v *OnboardingSkillsIntroView) Init() {
	// No state to reset
}

// SetSize sets the width and height of the view.
func (v *OnboardingSkillsIntroView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Update handles key input. Returns (should continue, was skipped).
func (v *OnboardingSkillsIntroView) Update(key string) (bool, bool) {
	switch key {
	case "enter":
		return true, false
	case "esc":
		return true, true
	}
	return false, false
}

// View renders the "What are Agent Skills?" onboarding view.
func (v *OnboardingSkillsIntroView) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current.Primary).
		MarginBottom(1)

	title := titleStyle.Render("What are Agent Skills?")

	subtitleStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		MarginBottom(2)

	subtitle := subtitleStyle.Render("A simple, open format for extending AI agent capabilities")

	descStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Text)

	description := descStyle.Render(
		"Agent Skills are folders of instructions, scripts, and\n" +
			"resources that AI coding agents can discover and use to\n" +
			"work more accurately and efficiently. A skill is a folder\n" +
			"containing a SKILL.md file with instructions that tell an\n" +
			"agent how to perform a specific task.\n" +
			"\n" +
			"Skills let you package domain expertise, repeatable\n" +
			"workflows, and new capabilities into portable, reusable\n" +
			"packages that work across 30+ agent products.\n" +
			"\n" +
			"Learn more at agentskills.io",
	)

	instructionStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Accent).
		MarginTop(2).
		MarginBottom(1)

	instructions := instructionStyle.Render(
		"Press Enter to continue or Esc to skip onboarding",
	)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		description,
		instructions,
	)

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
func (v *OnboardingSkillsIntroView) GetKeyboardCommands() ViewCommands {
	return ViewCommands{
		ViewName: "Onboarding",
		Commands: []Command{
			{Key: "Enter", Description: "Continue to next step"},
			{Key: "Esc", Description: "Skip onboarding"},
		},
	}
}
