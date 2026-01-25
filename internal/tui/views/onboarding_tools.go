package views

import (
	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/detect"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/charmbracelet/lipgloss"
)

// OnboardingToolsView displays AI tool detection and selection.
type OnboardingToolsView struct {
	cfg              *config.Config
	detectionResults []detect.DetectionResult
	selectedTools    map[installer.Platform]bool
	currentSelection int
	platformOrder    []installer.Platform
	width            int
	height           int
}

// NewOnboardingToolsView creates a new onboarding tools view.
func NewOnboardingToolsView(conf *config.Config) *OnboardingToolsView {
	return &OnboardingToolsView{
		cfg:           conf,
		selectedTools: make(map[installer.Platform]bool),
		platformOrder: []installer.Platform{
			installer.PlatformClaude,
			installer.PlatformCursor,
			installer.PlatformCopilot,
			installer.PlatformCodex,
			installer.PlatformOpenCode,
			installer.PlatformWindsurf,
		},
	}
}

// Init initializes the view and detects available tools.
func (v *OnboardingToolsView) Init() {
	v.detectionResults = detect.DetectAll()
	v.selectedTools = make(map[installer.Platform]bool)
	v.currentSelection = 0

	// Auto-select detected tools
	for _, result := range v.detectionResults {
		if result.Detected {
			v.selectedTools[result.Platform] = true
		}
	}

	// If no tools detected, select Claude by default
	if len(v.selectedTools) == 0 {
		v.selectedTools[installer.PlatformClaude] = true
	}
}

// SetSize sets the width and height of the view.
func (v *OnboardingToolsView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Update handles key input. Returns (should continue, was skipped).
func (v *OnboardingToolsView) Update(key string) (bool, bool) {
	switch key {
	case "up", "k":
		if v.currentSelection > 0 {
			v.currentSelection--
		}
	case "down", "j":
		if v.currentSelection < len(v.platformOrder)-1 {
			v.currentSelection++
		}
	case "space", "enter":
		// Toggle selection for current tool
		platform := v.platformOrder[v.currentSelection]
		v.selectedTools[platform] = !v.selectedTools[platform]
		// Ensure at least one tool is selected
		if v.hasAnySelected() == 0 {
			v.selectedTools[platform] = true
		}
	case "c":
		// Complete onboarding (confirm selection)
		return true, false
	case "esc":
		// Skip onboarding
		return true, true
	}
	return false, false
}

// GetSelectedPlatforms returns the list of selected platforms.
func (v *OnboardingToolsView) GetSelectedPlatforms() []installer.Platform {
	var platforms []installer.Platform
	for _, platform := range v.platformOrder {
		if v.selectedTools[platform] {
			platforms = append(platforms, platform)
		}
	}
	return platforms
}

// GetPlatformName returns the human-readable name for a platform (public interface).
func (v *OnboardingToolsView) GetPlatformName(platform installer.Platform) string {
	return v.platformName(platform)
}

// View renders the onboarding tools view.
func (v *OnboardingToolsView) View() string {
	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		MarginBottom(1)

	title := titleStyle.Render("🔧 Select AI Tools to Sync")

	// Subtitle
	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		MarginBottom(2)

	subtitle := subtitleStyle.Render("Choose which platforms to export your skills to")

	// Build platform selection list
	var platformItems []string
	for i, platform := range v.platformOrder {
		selected := v.selectedTools[platform]
		detected := v.isDetected(platform)

		var checkbox string
		if selected {
			checkbox = "☑"
		} else {
			checkbox = "☐"
		}

		var status string
		if detected {
			status = " ✓ installed"
		} else {
			status = " (not installed)"
		}

		platformName := v.platformName(platform)

		// Highlight current selection
		itemStyle := lipgloss.NewStyle()
		if i == v.currentSelection {
			itemStyle = itemStyle.
				Background(lipgloss.Color("238")).
				Foreground(lipgloss.Color("220")).
				Bold(true).
				Padding(0, 1)
		}

		item := itemStyle.Render(checkbox + " " + platformName + status)
		platformItems = append(platformItems, item)
	}

	platformsContent := lipgloss.JoinVertical(lipgloss.Left, platformItems...)

	// Instructions
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		MarginTop(2)

	instructions := instructionStyle.Render(
		"↑/↓ or j/k to navigate  •  Space/Enter to toggle  •  C to confirm or Esc to skip",
	)

	// Combine content
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		"",
		platformsContent,
		"",
		instructions,
	)

	// Create bordered dialog with responsive width
	maxWidth := v.width * 90 / 100
	if maxWidth < 50 {
		maxWidth = 50
	}

	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
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

// Helper functions

// isDetected checks if a platform was detected.
func (v *OnboardingToolsView) isDetected(platform installer.Platform) bool {
	for _, result := range v.detectionResults {
		if result.Platform == platform && result.Detected {
			return true
		}
	}
	return false
}

// platformName returns the human-readable name for a platform.
func (v *OnboardingToolsView) platformName(platform installer.Platform) string {
	switch platform {
	case installer.PlatformClaude:
		return "Claude Code"
	case installer.PlatformCursor:
		return "Cursor"
	case installer.PlatformCopilot:
		return "GitHub Copilot"
	case installer.PlatformCodex:
		return "OpenAI Codex"
	case installer.PlatformOpenCode:
		return "OpenCode"
	case installer.PlatformWindsurf:
		return "Windsurf"
	default:
		return string(platform)
	}
}

// hasAnySelected checks if at least one tool is selected.
func (v *OnboardingToolsView) hasAnySelected() int {
	count := 0
	for _, selected := range v.selectedTools {
		if selected {
			count++
		}
	}
	return count
}

// GetKeyboardCommands returns the keyboard commands for this view.
func (ov *OnboardingToolsView) GetKeyboardCommands() ViewCommands {
	return ViewCommands{
		ViewName: "Onboarding",
		Commands: []Command{
			{Key: "↑↓, k/j", Description: "Navigate tool options"},
			{Key: "Space, Enter", Description: "Toggle tool selection"},
			{Key: "c", Description: "Complete onboarding"},
			{Key: "Esc", Description: "Skip onboarding"},
		},
	}
}
