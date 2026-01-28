package views

import (
	"fmt"
	"sort"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PrimarySkillsFetchedMsg is sent when primary skills have been fetched from the repo.
type PrimarySkillsFetchedMsg struct {
	Skills []models.Skill
	Err    error
}

// SkillItem represents a skill in the checklist with selection state.
type SkillItem struct {
	Skill            models.Skill
	Selected         bool
	AlreadyInstalled bool
}

// OnboardingSkillsView displays skills from the primary repo in a checklist.
type OnboardingSkillsView struct {
	cfg *config.Config
	db  *db.DB

	// Skills lists - separated by installation status
	newSkills       []SkillItem
	installedSkills []SkillItem

	// Navigation state
	currentIndex   int  // Current selection index across both sections
	inNewSection   bool // Whether cursor is in the new skills section
	currentSection int  // 0 = new skills, 1 = already installed

	// Loading/error state
	loading  bool
	err      error
	errorMsg string

	// Dimensions
	width  int
	height int
}

// NewOnboardingSkillsView creates a new onboarding skills view.
func NewOnboardingSkillsView(conf *config.Config, database *db.DB) *OnboardingSkillsView {
	return &OnboardingSkillsView{
		cfg:          conf,
		db:           database,
		loading:      true,
		inNewSection: true,
	}
}

// Init resets the view state.
func (v *OnboardingSkillsView) Init() {
	v.newSkills = nil
	v.installedSkills = nil
	v.currentIndex = 0
	v.currentSection = 0
	v.inNewSection = true
	v.loading = true
	v.err = nil
	v.errorMsg = ""
}

// SetSize sets the width and height of the view.
func (v *OnboardingSkillsView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// HandleSkillsFetched processes the fetched skills and categorizes them.
func (v *OnboardingSkillsView) HandleSkillsFetched(skills []models.Skill, err error) {
	v.loading = false

	if err != nil {
		v.err = err
		v.errorMsg = err.Error()
		return
	}

	// Sort skills alphabetically by title
	sort.Slice(skills, func(i, j int) bool {
		return skills[i].Title < skills[j].Title
	})

	// Categorize skills into new and already installed
	v.newSkills = nil
	v.installedSkills = nil

	for _, skill := range skills {
		item := SkillItem{
			Skill:            skill,
			AlreadyInstalled: skill.IsInstalled,
		}

		if skill.IsInstalled {
			// Already installed skills are unchecked by default
			item.Selected = false
			v.installedSkills = append(v.installedSkills, item)
		} else {
			// New skills are pre-selected (checked by default)
			item.Selected = true
			v.newSkills = append(v.newSkills, item)
		}
	}

	// Start in the new skills section if there are new skills
	if len(v.newSkills) > 0 {
		v.currentSection = 0
		v.inNewSection = true
		v.currentIndex = 0
	} else if len(v.installedSkills) > 0 {
		v.currentSection = 1
		v.inNewSection = false
		v.currentIndex = 0
	}
}

// Update handles key input. Returns (done, skipped, cmd).
func (v *OnboardingSkillsView) Update(key string) (bool, bool, tea.Cmd) {
	// If loading or error, only allow skip/continue
	if v.loading {
		return false, false, nil
	}

	if v.err != nil {
		switch key {
		case "enter":
			return true, false, nil
		case "esc":
			return true, true, nil
		}
		return false, false, nil
	}

	totalNew := len(v.newSkills)
	totalInstalled := len(v.installedSkills)
	totalItems := totalNew + totalInstalled

	if totalItems == 0 {
		// No skills to show, allow continue/skip
		switch key {
		case "enter":
			return true, false, nil
		case "esc":
			return true, true, nil
		}
		return false, false, nil
	}

	switch key {
	case "up", "k":
		v.moveUp()
	case "down", "j":
		v.moveDown()
	case "space", " ":
		v.toggleSelection()
	case "a":
		// Select all new skills
		for i := range v.newSkills {
			v.newSkills[i].Selected = true
		}
	case "n":
		// Deselect all
		for i := range v.newSkills {
			v.newSkills[i].Selected = false
		}
		for i := range v.installedSkills {
			v.installedSkills[i].Selected = false
		}
	case "enter":
		// Continue with selection
		return true, false, nil
	case "esc":
		// Skip
		return true, true, nil
	}

	return false, false, nil
}

// moveUp moves the cursor up, crossing section boundaries.
func (v *OnboardingSkillsView) moveUp() {
	if v.currentSection == 0 {
		// In new skills section
		if v.currentIndex > 0 {
			v.currentIndex--
		}
	} else {
		// In installed skills section
		if v.currentIndex > 0 {
			v.currentIndex--
		} else if len(v.newSkills) > 0 {
			// Move to new skills section
			v.currentSection = 0
			v.inNewSection = true
			v.currentIndex = len(v.newSkills) - 1
		}
	}
}

// moveDown moves the cursor down, crossing section boundaries.
func (v *OnboardingSkillsView) moveDown() {
	if v.currentSection == 0 {
		// In new skills section
		if v.currentIndex < len(v.newSkills)-1 {
			v.currentIndex++
		} else if len(v.installedSkills) > 0 {
			// Move to installed skills section
			v.currentSection = 1
			v.inNewSection = false
			v.currentIndex = 0
		}
	} else {
		// In installed skills section
		if v.currentIndex < len(v.installedSkills)-1 {
			v.currentIndex++
		}
	}
}

// toggleSelection toggles the selection of the current item.
func (v *OnboardingSkillsView) toggleSelection() {
	if v.currentSection == 0 && len(v.newSkills) > 0 {
		v.newSkills[v.currentIndex].Selected = !v.newSkills[v.currentIndex].Selected
	} else if v.currentSection == 1 && len(v.installedSkills) > 0 {
		v.installedSkills[v.currentIndex].Selected = !v.installedSkills[v.currentIndex].Selected
	}
}

// GetSelectedSkills returns the list of selected new skills to install.
func (v *OnboardingSkillsView) GetSelectedSkills() []models.Skill {
	var selected []models.Skill
	for _, item := range v.newSkills {
		if item.Selected {
			selected = append(selected, item.Skill)
		}
	}
	return selected
}

// GetReplaceSkills returns the list of already-installed skills selected for replacement.
func (v *OnboardingSkillsView) GetReplaceSkills() []models.Skill {
	var replace []models.Skill
	for _, item := range v.installedSkills {
		if item.Selected {
			replace = append(replace, item.Skill)
		}
	}
	return replace
}

// View renders the onboarding skills view.
func (v *OnboardingSkillsView) View() string {
	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current.Primary).
		MarginBottom(1)

	title := titleStyle.Render("Select Skills to Install")

	// Subtitle
	subtitleStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		MarginBottom(2)

	subtitle := subtitleStyle.Render("Curated skills from the Asteroid Belt team")

	// Skill highlights
	highlightStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		MarginBottom(1)

	highlights := highlightStyle.Render(
		"• superplan + superbuild — rigorous quality gates, definitions of done,\n" +
			"  and enforced autonomy with sub-agents\n" +
			"• teach — interactively learn any technical concept, codebase, or spec\n" +
			"• agentsmd-generator — generate a comprehensive AGENTS.md for your project",
	)

	// Calculate content width for wrapping (maxWidth - padding - border)
	maxWidth := v.width * 90 / 100
	if maxWidth < 50 {
		maxWidth = 50
	}
	contentWidth := maxWidth - 8 // 3*2 padding + 2 border

	// Content based on state
	var content string

	if v.loading {
		content = v.renderLoading()
	} else if v.err != nil {
		content = v.renderError(contentWidth)
	} else {
		content = v.renderChecklist()
	}

	// Instructions
	instructionStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		MarginTop(2)

	var instructions string
	if v.loading {
		instructions = instructionStyle.Render("Loading skills...")
	} else if v.err != nil {
		instructions = instructionStyle.Render("Enter to continue  •  Esc to skip")
	} else {
		instructions = instructionStyle.Render(
			"↑/↓ or j/k to navigate  •  Space to toggle  •  A all  •  N none  •  Enter to continue  •  Esc to skip",
		)
	}

	// Combine content
	fullContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		highlights,
		"",
		content,
		"",
		instructions,
	)

	// Create bordered dialog with responsive width
	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Current.Primary).
		Padding(2, 3).
		MaxWidth(maxWidth)

	renderedDialog := dialog.Render(fullContent)

	// Center the dialog both vertically and horizontally
	dialogWidth := lipgloss.Width(renderedDialog)
	paddingLeft := (v.width - dialogWidth) / 2
	if paddingLeft < 0 {
		paddingLeft = 0
	}

	dialogHeight := lipgloss.Height(renderedDialog)
	paddingTop := (v.height - dialogHeight) / 2
	if paddingTop < 1 {
		paddingTop = 1
	}

	return lipgloss.NewStyle().
		Padding(paddingTop, paddingLeft).
		Render(renderedDialog)
}

// renderLoading renders the loading state.
func (v *OnboardingSkillsView) renderLoading() string {
	loadingStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Accent).
		Italic(true)

	return loadingStyle.Render("Fetching skills from primary repository...")
}

// renderError renders the error state with text wrapping.
func (v *OnboardingSkillsView) renderError(maxWidth int) string {
	errorStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Error).
		Width(maxWidth)

	return errorStyle.Render(fmt.Sprintf("Error: %s", v.errorMsg))
}

// renderChecklist renders the skills checklist with sections.
func (v *OnboardingSkillsView) renderChecklist() string {
	if len(v.newSkills) == 0 && len(v.installedSkills) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(theme.Current.TextMuted).
			Italic(true)
		return emptyStyle.Render("No skills found in the primary repository.")
	}

	var sections []string

	// Render "New Skills" section
	if len(v.newSkills) > 0 {
		sectionHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Current.Accent).
			MarginBottom(1).
			Render("New Skills")

		var items []string
		for i, item := range v.newSkills {
			isSelected := v.currentSection == 0 && v.currentIndex == i
			items = append(items, v.renderSkillItem(item, isSelected, false))
		}

		sectionContent := lipgloss.JoinVertical(lipgloss.Left, items...)
		sections = append(sections, lipgloss.JoinVertical(lipgloss.Left, sectionHeader, sectionContent))
	}

	// Render "Already Installed" section
	if len(v.installedSkills) > 0 {
		sectionHeader := lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Current.TextMuted).
			MarginTop(1).
			MarginBottom(1).
			Render("Already Installed")

		var items []string
		for i, item := range v.installedSkills {
			isSelected := v.currentSection == 1 && v.currentIndex == i
			items = append(items, v.renderSkillItem(item, isSelected, true))
		}

		sectionContent := lipgloss.JoinVertical(lipgloss.Left, items...)
		sections = append(sections, lipgloss.JoinVertical(lipgloss.Left, sectionHeader, sectionContent))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderSkillItem renders a single skill item in the checklist.
func (v *OnboardingSkillsView) renderSkillItem(item SkillItem, isSelected bool, isInstalled bool) string {
	// Checkbox
	var checkbox string
	if item.Selected {
		checkbox = "☑"
	} else {
		checkbox = "☐"
	}

	// Skill title
	title := item.Skill.Title
	if title == "" {
		title = item.Skill.Slug
	}

	// Add "(replace?)" suffix for already installed skills
	if isInstalled {
		title = title + " (replace?)"
	}

	// Build the full line
	line := fmt.Sprintf("%s %s", checkbox, title)

	// Apply styling based on selection
	itemStyle := lipgloss.NewStyle()
	if isSelected {
		itemStyle = itemStyle.
			Background(theme.Current.Surface).
			Foreground(theme.Current.Accent).
			Bold(true).
			Padding(0, 1)
	} else if isInstalled {
		itemStyle = itemStyle.
			Foreground(theme.Current.TextMuted)
	}

	return itemStyle.Render(line)
}

// GetKeyboardCommands returns the keyboard commands for this view.
func (v *OnboardingSkillsView) GetKeyboardCommands() ViewCommands {
	return ViewCommands{
		ViewName: "Skills Onboarding",
		Commands: []Command{
			{Key: "↑↓, k/j", Description: "Navigate skills"},
			{Key: "Space", Description: "Toggle selection"},
			{Key: "a", Description: "Select all new skills"},
			{Key: "n", Description: "Deselect all skills"},
			{Key: "Enter", Description: "Continue with selection"},
			{Key: "Esc", Description: "Skip skills selection"},
		},
	}
}
