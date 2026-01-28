package views

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/telemetry"
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ManageAction represents actions from the manage view.
type ManageAction int

const (
	ManageActionNone ManageAction = iota
	ManageActionBack
	ManageActionSelectSkill
)

// ManageSkillsLoadedMsg is sent when skills data has been loaded.
type ManageSkillsLoadedMsg struct {
	Skills []installer.InstalledSkillSummary
	Err    error
}

// ManageView displays installed skills with their locations for management.
type ManageView struct {
	db             *db.DB
	cfg            *config.Config
	installService *installer.InstallService
	telemetry      telemetry.Client

	// Data
	skills    []installer.InstalledSkillSummary
	loading   bool
	loadError error

	// UI State
	selectedIdx  int
	scrollOffset int
	width        int
	height       int
}

// NewManageView creates a new manage view.
func NewManageView(database *db.DB, conf *config.Config, installSvc *installer.InstallService, tel telemetry.Client) *ManageView {
	return &ManageView{
		db:             database,
		cfg:            conf,
		installService: installSvc,
		telemetry:      tel,
		loading:        true,
		selectedIdx:    0,
		scrollOffset:   0,
		width:          80,
		height:         24,
	}
}

// SetSize sets the width and height of the view.
func (mv *ManageView) SetSize(width, height int) {
	mv.width = width
	mv.height = height
}

// Init initializes the manage view and loads data.
func (mv *ManageView) Init() tea.Cmd {
	mv.loading = true
	mv.loadError = nil
	mv.scrollOffset = 0

	return func() tea.Msg {
		skills, err := mv.installService.GetInstalledSkillsSummary(context.Background())
		return ManageSkillsLoadedMsg{Skills: skills, Err: err}
	}
}

// HandleManageSkillsLoaded handles the ManageSkillsLoadedMsg.
func (mv *ManageView) HandleManageSkillsLoaded(msg ManageSkillsLoadedMsg) {
	if msg.Err != nil {
		mv.loadError = msg.Err
	} else {
		mv.skills = msg.Skills
	}
	mv.loading = false
}

// Update handles keyboard input and returns the action to perform.
func (mv *ManageView) Update(key string) (ManageAction, tea.Cmd) {
	switch key {
	case "q", "esc":
		return ManageActionBack, nil

	case "j", "down":
		if mv.selectedIdx < len(mv.skills)-1 {
			mv.selectedIdx++
			mv.adjustScrollForSelection()
		}
		return ManageActionNone, nil

	case "k", "up":
		if mv.selectedIdx > 0 {
			mv.selectedIdx--
			mv.adjustScrollForSelection()
		}
		return ManageActionNone, nil

	case "d":
		// Down half page
		pageSize := max(1, (mv.height-8)/2)
		mv.selectedIdx = min(mv.selectedIdx+pageSize, len(mv.skills)-1)
		mv.adjustScrollForSelection()
		return ManageActionNone, nil

	case "u":
		// Up half page
		pageSize := max(1, (mv.height-8)/2)
		mv.selectedIdx = max(mv.selectedIdx-pageSize, 0)
		mv.adjustScrollForSelection()
		return ManageActionNone, nil

	case "g":
		mv.selectedIdx = 0
		mv.scrollOffset = 0
		return ManageActionNone, nil

	case "G":
		mv.selectedIdx = max(0, len(mv.skills)-1)
		mv.adjustScrollForSelection()
		return ManageActionNone, nil

	case "enter":
		if len(mv.skills) > 0 && mv.selectedIdx >= 0 && mv.selectedIdx < len(mv.skills) {
			return ManageActionSelectSkill, nil
		}
		return ManageActionNone, nil
	}

	return ManageActionNone, nil
}

// GetSelectedSkill returns the currently selected skill.
func (mv *ManageView) GetSelectedSkill() *installer.InstalledSkillSummary {
	if mv.selectedIdx >= 0 && mv.selectedIdx < len(mv.skills) {
		return &mv.skills[mv.selectedIdx]
	}
	return nil
}

// RefreshSkills reloads the skills list.
func (mv *ManageView) RefreshSkills() tea.Cmd {
	mv.loading = true
	return mv.Init()
}

// View renders the manage view.
func (mv *ManageView) View() string {
	if mv.loading {
		return mv.renderLoading()
	}

	if mv.loadError != nil {
		return mv.renderError()
	}

	header := mv.renderHeader()
	content := mv.renderSkillsList()
	footer := mv.renderFooter()

	return lipgloss.JoinVertical(
		lipgloss.Top,
		header,
		"",
		content,
		"",
		footer,
	)
}

// adjustScrollForSelection adjusts scroll offset to keep the selected item visible.
func (mv *ManageView) adjustScrollForSelection() {
	maxVisibleItems := mv.getMaxVisibleItems()

	// Scroll down if selected item is below visible area
	if mv.selectedIdx >= mv.scrollOffset+maxVisibleItems {
		mv.scrollOffset = mv.selectedIdx - maxVisibleItems + 1
	}
	// Scroll up if selected item is above visible area
	if mv.selectedIdx < mv.scrollOffset {
		mv.scrollOffset = mv.selectedIdx
	}
	// Clamp scroll offset
	maxScroll := len(mv.skills) - maxVisibleItems
	if maxScroll < 0 {
		maxScroll = 0
	}
	if mv.scrollOffset > maxScroll {
		mv.scrollOffset = maxScroll
	}
	if mv.scrollOffset < 0 {
		mv.scrollOffset = 0
	}
}

// getMaxVisibleItems calculates how many items can be shown.
func (mv *ManageView) getMaxVisibleItems() int {
	// Header (2) + blank + content area + blank + footer (2) + margins
	fixedHeight := 8
	availableHeight := mv.height - fixedHeight
	// Each item may take 1-3 lines depending on location wrapping
	// Use 2 as average for multi-platform skills
	maxItems := availableHeight / 2
	if maxItems < 3 {
		maxItems = 3
	}
	if maxItems > 20 {
		maxItems = 20
	}
	return maxItems
}

// renderLoading shows a loading message.
func (mv *ManageView) renderLoading() string {
	loadingStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Accent).
		Bold(true).
		MarginLeft(2).
		MarginTop(2)

	return loadingStyle.Render("Loading installed skills...")
}

// renderError shows an error message.
func (mv *ManageView) renderError() string {
	errorStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Error).
		Bold(true).
		MarginLeft(2).
		MarginTop(2)

	return errorStyle.Render(fmt.Sprintf("Error: %v", mv.loadError))
}

// renderHeader renders the header section.
func (mv *ManageView) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Info).
		Bold(true).
		MarginLeft(2).
		MarginTop(1)

	title := titleStyle.Render("MANAGE INSTALLED SKILLS")

	// Column headers
	headerStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Text).
		Bold(true).
		MarginLeft(2)

	skillWidth := mv.getSkillColumnWidth()
	columnHeader := fmt.Sprintf("  %-*s  %s", skillWidth, "SKILL", "INSTALLED LOCATIONS")
	header := headerStyle.Render(columnHeader)

	// Separator line
	separatorStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		MarginLeft(2)

	separator := separatorStyle.Render(strings.Repeat("─", min(mv.width-4, skillWidth+2+40)))

	return lipgloss.JoinVertical(lipgloss.Top, title, header, separator)
}

// renderSkillsList renders the skills list with scrolling.
func (mv *ManageView) renderSkillsList() string {
	if len(mv.skills) == 0 {
		return mv.renderEmptyState()
	}

	maxVisibleItems := mv.getMaxVisibleItems()
	skillWidth := mv.getSkillColumnWidth()

	// Calculate visible range
	startIdx := mv.scrollOffset
	endIdx := startIdx + maxVisibleItems
	if endIdx > len(mv.skills) {
		endIdx = len(mv.skills)
	}

	var lines []string
	for i := startIdx; i < endIdx; i++ {
		skill := mv.skills[i]
		isSelected := i == mv.selectedIdx
		line := mv.renderSkillRow(skill, skillWidth, isSelected)
		lines = append(lines, line)
	}

	// Add scroll indicators
	if endIdx < len(mv.skills) {
		moreStyle := lipgloss.NewStyle().
			Foreground(theme.Current.TextMuted).
			Italic(true).
			MarginLeft(4)
		lines = append(lines, moreStyle.Render(fmt.Sprintf("↓ %d more...", len(mv.skills)-endIdx)))
	}

	return strings.Join(lines, "\n")
}

// renderSkillRow renders a single skill row with wrapped locations.
func (mv *ManageView) renderSkillRow(skill installer.InstalledSkillSummary, skillWidth int, isSelected bool) string {
	// Selection indicator
	indicator := "  "
	if isSelected {
		indicator = "▸ "
	}

	// Skill name styling
	skillStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Secondary)

	if isSelected {
		skillStyle = skillStyle.Bold(true)
	}

	skillName := skillStyle.Render(fmt.Sprintf("%-*s", skillWidth, skill.Slug))

	// Locations formatting - wrap to multiple lines if needed
	locationParts := mv.formatLocationParts(skill.Locations)
	// Prefix width: indicator(2) + skillName(skillWidth) + gap(2) + marginLeft(2)
	prefixWidth := 2 + skillWidth + 2
	// Available width for locations
	locWidth := mv.width - prefixWidth - 4 // margin
	if locWidth < 30 {
		locWidth = 30
	}

	// Build wrapped location lines
	var locLines []string
	var currentLine string
	mutedStyle := lipgloss.NewStyle().Foreground(theme.Current.TextMuted)

	for i, part := range locationParts {
		sep := ""
		if i > 0 {
			sep = ", "
		}
		candidate := currentLine + sep + part
		// Rough length estimate (strip ANSI for length check)
		plainLen := len(stripLocationPart(candidate))
		if plainLen > locWidth && currentLine != "" {
			locLines = append(locLines, currentLine)
			currentLine = part
		} else {
			if currentLine == "" {
				currentLine = part
			} else {
				currentLine += mutedStyle.Render(", ") + part
			}
		}
	}
	if currentLine != "" {
		locLines = append(locLines, currentLine)
	}

	// Build row
	rowStyle := lipgloss.NewStyle().MarginLeft(2)
	if isSelected {
		rowStyle = rowStyle.Background(theme.Current.Surface)
	}

	if len(locLines) <= 1 {
		loc := ""
		if len(locLines) == 1 {
			loc = locLines[0]
		}
		return rowStyle.Render(indicator + skillName + "  " + loc)
	}

	// Multi-line: first line with skill name, subsequent lines indented
	padding := strings.Repeat(" ", prefixWidth)
	var rows []string
	rows = append(rows, indicator+skillName+"  "+locLines[0])
	for _, line := range locLines[1:] {
		rows = append(rows, padding+line)
	}
	return rowStyle.Render(strings.Join(rows, "\n"))
}

// stripLocationPart returns a rough plain-text length estimate by removing ANSI escape sequences.
func stripLocationPart(s string) string {
	// Simple strip: remove \x1b[...m sequences
	result := make([]byte, 0, len(s))
	inEsc := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if s[i] == 'm' {
				inEsc = false
			}
			continue
		}
		result = append(result, s[i])
	}
	return string(result)
}

// formatLocationParts returns individually styled "platform (scope)" strings.
func (mv *ManageView) formatLocationParts(locations map[installer.Platform][]installer.InstallScope) []string {
	if len(locations) == 0 {
		return nil
	}

	// Get sorted platform names
	platforms := make([]string, 0, len(locations))
	for p := range locations {
		platforms = append(platforms, string(p))
	}
	sort.Strings(platforms)

	// Build formatted strings with styling
	parts := make([]string, 0, len(platforms))
	for _, pStr := range platforms {
		p := installer.Platform(pStr)
		scopes := locations[p]
		scopeStr, scopeStyle := mv.formatScopes(scopes)
		styledScope := scopeStyle.Render(scopeStr)
		parts = append(parts, fmt.Sprintf("%s (%s)", pStr, styledScope))
	}

	return parts
}

// formatScopes formats scopes and returns appropriate style.
func (mv *ManageView) formatScopes(scopes []installer.InstallScope) (string, lipgloss.Style) {
	hasGlobal := false
	hasProject := false

	for _, s := range scopes {
		switch s {
		case installer.ScopeGlobal:
			hasGlobal = true
		case installer.ScopeProject:
			hasProject = true
		}
	}

	// Colors matching check.go CLI
	globalStyle := lipgloss.NewStyle().Foreground(theme.Current.Success)
	projectStyle := lipgloss.NewStyle().Foreground(theme.Current.Info)
	bothStyle := lipgloss.NewStyle().Foreground(theme.Current.Accent)
	mutedStyle := lipgloss.NewStyle().Foreground(theme.Current.TextMuted)

	switch {
	case hasGlobal && hasProject:
		return "global + project", bothStyle
	case hasGlobal:
		return "global", globalStyle
	case hasProject:
		return "project", projectStyle
	default:
		return "", mutedStyle
	}
}

// renderEmptyState renders the empty state.
func (mv *ManageView) renderEmptyState() string {
	emptyStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		Italic(true).
		MarginLeft(4).
		MarginTop(2)

	helpStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		MarginLeft(4).
		MarginTop(1)

	return emptyStyle.Render("No skills installed.") + "\n" +
		helpStyle.Render("Use 'skulto install <slug>' or browse skills from Home.")
}

// renderFooter renders the footer with navigation help.
func (mv *ManageView) renderFooter() string {
	// Count display
	countStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Primary).
		Bold(true).
		MarginLeft(2)

	count := ""
	if len(mv.skills) > 0 {
		count = countStyle.Render(fmt.Sprintf("%d skill(s) installed", len(mv.skills)))
	}

	// Navigation help
	helpStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		MarginLeft(2).
		MarginTop(1)

	help := helpStyle.Render("↑↓ navigate • Enter edit • Esc back")

	return lipgloss.JoinVertical(lipgloss.Top, count, help)
}

// getSkillColumnWidth calculates the width for the skill name column.
func (mv *ManageView) getSkillColumnWidth() int {
	minWidth := len("SKILL")
	maxWidth := 30

	for _, s := range mv.skills {
		if len(s.Slug) > minWidth {
			minWidth = len(s.Slug)
		}
	}

	if minWidth > maxWidth {
		return maxWidth
	}
	return minWidth
}

// GetKeyboardCommands returns the keyboard commands for this view.
func (mv *ManageView) GetKeyboardCommands() ViewCommands {
	return ViewCommands{
		ViewName: "Manage",
		Commands: []Command{
			{Key: "↑↓, k/j", Description: "Navigate skills"},
			{Key: "d/u", Description: "Page down/up"},
			{Key: "g/G", Description: "Jump to start/end"},
			{Key: "Enter", Description: "Edit skill locations"},
			{Key: "Esc, q", Description: "Return to Home"},
		},
	}
}
