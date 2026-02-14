package views

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/telemetry"
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ManageSection represents the active section in the manage view.
type ManageSection int

const (
	ManageSectionInstalled ManageSection = iota
	ManageSectionDiscovered
)

// ManageAction represents actions from the manage view.
type ManageAction int

const (
	ManageActionNone ManageAction = iota
	ManageActionBack
	ManageActionSelectSkill
	ManageActionSelectDiscovery
	ManageActionSave
)

// ManageSkillsLoadedMsg is sent when skills data has been loaded.
type ManageSkillsLoadedMsg struct {
	Skills []installer.InstalledSkillSummary
	Err    error
}

// DiscoveriesLoadedMsg is sent when discovered skills data has been loaded.
type DiscoveriesLoadedMsg struct {
	Skills []models.DiscoveredSkill
	Err    error
}

// ManageView displays installed skills with their locations for management.
type ManageView struct {
	db             *db.DB
	cfg            *config.Config
	installService *installer.InstallService
	telemetry      telemetry.Client

	// Data - Installed skills
	skills    []installer.InstalledSkillSummary
	loading   bool
	loadError error

	// Data - Discovered skills
	discoveries      []models.DiscoveredSkill
	discoveriesError error

	// UI State - Section
	section ManageSection

	// UI State - Installed section
	selectedIdx  int
	scrollOffset int

	// UI State - Discovered section
	discoveredIdx    int
	discoveredScroll int

	// UI State - Common
	width  int
	height int
}

// NewManageView creates a new manage view.
func NewManageView(database *db.DB, conf *config.Config, installSvc *installer.InstallService, tel telemetry.Client) *ManageView {
	return &ManageView{
		db:               database,
		cfg:              conf,
		installService:   installSvc,
		telemetry:        tel,
		loading:          true,
		section:          ManageSectionInstalled,
		selectedIdx:      0,
		scrollOffset:     0,
		discoveredIdx:    0,
		discoveredScroll: 0,
		width:            80,
		height:           24,
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

	loadInstalled := func() tea.Msg {
		skills, err := mv.installService.GetInstalledSkillsSummary(context.Background())
		return ManageSkillsLoadedMsg{Skills: skills, Err: err}
	}

	loadDiscoveries := func() tea.Msg {
		skills, err := mv.db.ListDiscoveredSkills()
		return DiscoveriesLoadedMsg{Skills: skills, Err: err}
	}

	return tea.Batch(loadInstalled, loadDiscoveries)
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

// HandleDiscoveriesLoaded handles the DiscoveriesLoadedMsg.
func (mv *ManageView) HandleDiscoveriesLoaded(msg DiscoveriesLoadedMsg) {
	if msg.Err != nil {
		mv.discoveriesError = msg.Err
	} else {
		mv.discoveries = msg.Skills
	}
}

// LoadDiscoveries returns a command to load discovered skills from the database.
func (mv *ManageView) LoadDiscoveries() tea.Cmd {
	return func() tea.Msg {
		skills, err := mv.db.ListDiscoveredSkills()
		return DiscoveriesLoadedMsg{Skills: skills, Err: err}
	}
}

// GetCurrentSection returns the currently active section.
func (mv *ManageView) GetCurrentSection() ManageSection {
	return mv.section
}

// GetSelectedIndex returns the selection index for the current section.
func (mv *ManageView) GetSelectedIndex() int {
	if mv.section == ManageSectionDiscovered {
		return mv.discoveredIdx
	}
	return mv.selectedIdx
}

// GetDiscoveries returns the list of discovered skills.
func (mv *ManageView) GetDiscoveries() []models.DiscoveredSkill {
	return mv.discoveries
}

// Update handles keyboard input and returns the action to perform.
func (mv *ManageView) Update(key string) (ManageAction, tea.Cmd) {
	switch key {
	case "q", "esc":
		return ManageActionBack, nil

	case "tab":
		// Toggle between sections
		if mv.section == ManageSectionInstalled {
			mv.section = ManageSectionDiscovered
		} else {
			mv.section = ManageSectionInstalled
		}
		return ManageActionNone, nil

	case "j", "down":
		mv.navigateDown()
		return ManageActionNone, nil

	case "k", "up":
		mv.navigateUp()
		return ManageActionNone, nil

	case "d":
		// Down half page
		pageSize := max(1, (mv.height-8)/2)
		mv.navigateDownBy(pageSize)
		return ManageActionNone, nil

	case "u":
		// Up half page
		pageSize := max(1, (mv.height-8)/2)
		mv.navigateUpBy(pageSize)
		return ManageActionNone, nil

	case "g":
		mv.navigateToStart()
		return ManageActionNone, nil

	case "G":
		mv.navigateToEnd()
		return ManageActionNone, nil

	case "enter":
		switch mv.section {
		case ManageSectionInstalled:
			if len(mv.skills) > 0 && mv.selectedIdx >= 0 && mv.selectedIdx < len(mv.skills) {
				return ManageActionSelectSkill, nil
			}
		case ManageSectionDiscovered:
			if len(mv.discoveries) > 0 && mv.discoveredIdx >= 0 && mv.discoveredIdx < len(mv.discoveries) {
				return ManageActionSelectDiscovery, nil
			}
		default:
			return ManageActionNone, nil
		}

	case "S":
		return ManageActionSave, nil
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

// GetSelectedDiscovery returns the currently selected discovered skill.
func (mv *ManageView) GetSelectedDiscovery() *models.DiscoveredSkill {
	if mv.discoveredIdx >= 0 && mv.discoveredIdx < len(mv.discoveries) {
		return &mv.discoveries[mv.discoveredIdx]
	}
	return nil
}

// navigateDown moves selection down by one in the current section.
func (mv *ManageView) navigateDown() {
	if mv.section == ManageSectionDiscovered {
		if mv.discoveredIdx < len(mv.discoveries)-1 {
			mv.discoveredIdx++
			mv.adjustDiscoveredScroll()
		}
	} else {
		if mv.selectedIdx < len(mv.skills)-1 {
			mv.selectedIdx++
			mv.adjustScrollForSelection()
		}
	}
}

// navigateUp moves selection up by one in the current section.
func (mv *ManageView) navigateUp() {
	if mv.section == ManageSectionDiscovered {
		if mv.discoveredIdx > 0 {
			mv.discoveredIdx--
			mv.adjustDiscoveredScroll()
		}
	} else {
		if mv.selectedIdx > 0 {
			mv.selectedIdx--
			mv.adjustScrollForSelection()
		}
	}
}

// navigateDownBy moves selection down by n items in the current section.
func (mv *ManageView) navigateDownBy(n int) {
	if mv.section == ManageSectionDiscovered {
		mv.discoveredIdx = min(mv.discoveredIdx+n, len(mv.discoveries)-1)
		mv.adjustDiscoveredScroll()
	} else {
		mv.selectedIdx = min(mv.selectedIdx+n, len(mv.skills)-1)
		mv.adjustScrollForSelection()
	}
}

// navigateUpBy moves selection up by n items in the current section.
func (mv *ManageView) navigateUpBy(n int) {
	if mv.section == ManageSectionDiscovered {
		mv.discoveredIdx = max(mv.discoveredIdx-n, 0)
		mv.adjustDiscoveredScroll()
	} else {
		mv.selectedIdx = max(mv.selectedIdx-n, 0)
		mv.adjustScrollForSelection()
	}
}

// navigateToStart moves to the first item in the current section.
func (mv *ManageView) navigateToStart() {
	if mv.section == ManageSectionDiscovered {
		mv.discoveredIdx = 0
		mv.discoveredScroll = 0
	} else {
		mv.selectedIdx = 0
		mv.scrollOffset = 0
	}
}

// navigateToEnd moves to the last item in the current section.
func (mv *ManageView) navigateToEnd() {
	if mv.section == ManageSectionDiscovered {
		mv.discoveredIdx = max(0, len(mv.discoveries)-1)
		mv.adjustDiscoveredScroll()
	} else {
		mv.selectedIdx = max(0, len(mv.skills)-1)
		mv.adjustScrollForSelection()
	}
}

// adjustDiscoveredScroll adjusts scroll offset for discovered section.
func (mv *ManageView) adjustDiscoveredScroll() {
	maxVisibleItems := mv.getMaxVisibleItems()

	// Scroll down if selected item is below visible area
	if mv.discoveredIdx >= mv.discoveredScroll+maxVisibleItems {
		mv.discoveredScroll = mv.discoveredIdx - maxVisibleItems + 1
	}
	// Scroll up if selected item is above visible area
	if mv.discoveredIdx < mv.discoveredScroll {
		mv.discoveredScroll = mv.discoveredIdx
	}
	// Clamp scroll offset
	maxScroll := len(mv.discoveries) - maxVisibleItems
	if maxScroll < 0 {
		maxScroll = 0
	}
	if mv.discoveredScroll > maxScroll {
		mv.discoveredScroll = maxScroll
	}
	if mv.discoveredScroll < 0 {
		mv.discoveredScroll = 0
	}
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

	sectionTabs := mv.renderSectionTabs()
	header := mv.renderHeader()
	content := mv.renderContent()
	footer := mv.renderFooter()

	return lipgloss.JoinVertical(
		lipgloss.Top,
		sectionTabs,
		header,
		"",
		content,
		"",
		footer,
	)
}

// renderSectionTabs renders the section tab bar.
func (mv *ManageView) renderSectionTabs() string {
	activeStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Primary).
		Bold(true).
		Underline(true)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted)

	installedLabel := fmt.Sprintf("INSTALLED (%d)", len(mv.skills))
	discoveredLabel := fmt.Sprintf("DISCOVERED (%d)", len(mv.discoveries))

	var installedTab, discoveredTab string
	if mv.section == ManageSectionInstalled {
		installedTab = activeStyle.Render(installedLabel)
		discoveredTab = inactiveStyle.Render(discoveredLabel)
	} else {
		installedTab = inactiveStyle.Render(installedLabel)
		discoveredTab = activeStyle.Render(discoveredLabel)
	}

	tabStyle := lipgloss.NewStyle().
		MarginLeft(2).
		MarginTop(1)

	return tabStyle.Render(installedTab + "    " + discoveredTab)
}

// renderContent renders the appropriate content based on the current section.
func (mv *ManageView) renderContent() string {
	if mv.section == ManageSectionDiscovered {
		return mv.renderDiscoveredList()
	}
	return mv.renderSkillsList()
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
	// Column headers
	headerStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Text).
		Bold(true).
		MarginLeft(2)

	var columnHeader string
	var separatorWidth int

	if mv.section == ManageSectionDiscovered {
		skillWidth := mv.getDiscoveredColumnWidth()
		columnHeader = fmt.Sprintf("  %-*s  %-10s  %s", skillWidth, "NAME", "PLATFORM", "SCOPE")
		separatorWidth = min(mv.width-4, skillWidth+2+10+2+20)
	} else {
		skillWidth := mv.getSkillColumnWidth()
		columnHeader = fmt.Sprintf("  %-*s  %s", skillWidth, "SKILL", "INSTALLED LOCATIONS")
		separatorWidth = min(mv.width-4, skillWidth+2+40)
	}
	header := headerStyle.Render(columnHeader)

	// Separator line
	separatorStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		MarginLeft(2)

	separator := separatorStyle.Render(strings.Repeat("─", separatorWidth))

	return lipgloss.JoinVertical(lipgloss.Top, header, separator)
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

// renderDiscoveredList renders the discovered skills list.
func (mv *ManageView) renderDiscoveredList() string {
	if len(mv.discoveries) == 0 {
		return mv.renderDiscoveredEmptyState()
	}

	maxVisibleItems := mv.getMaxVisibleItems()
	nameWidth := mv.getDiscoveredColumnWidth()

	// Calculate visible range
	startIdx := mv.discoveredScroll
	endIdx := startIdx + maxVisibleItems
	if endIdx > len(mv.discoveries) {
		endIdx = len(mv.discoveries)
	}

	var lines []string
	for i := startIdx; i < endIdx; i++ {
		discovery := mv.discoveries[i]
		isSelected := i == mv.discoveredIdx
		line := mv.renderDiscoveredRow(discovery, nameWidth, isSelected)
		lines = append(lines, line)
	}

	// Add scroll indicators
	if endIdx < len(mv.discoveries) {
		moreStyle := lipgloss.NewStyle().
			Foreground(theme.Current.TextMuted).
			Italic(true).
			MarginLeft(4)
		lines = append(lines, moreStyle.Render(fmt.Sprintf("↓ %d more...", len(mv.discoveries)-endIdx)))
	}

	return strings.Join(lines, "\n")
}

// renderDiscoveredRow renders a single discovered skill row.
func (mv *ManageView) renderDiscoveredRow(discovery models.DiscoveredSkill, nameWidth int, isSelected bool) string {
	// Selection indicator
	indicator := "  "
	if isSelected {
		indicator = "▸ "
	}

	// Name styling
	nameStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Warning) // Use warning color for discovered (external) skills

	if isSelected {
		nameStyle = nameStyle.Bold(true)
	}

	name := nameStyle.Render(fmt.Sprintf("%-*s", nameWidth, discovery.Name))

	// Platform styling
	platformStyle := lipgloss.NewStyle().Foreground(theme.Current.Info)
	platform := platformStyle.Render(fmt.Sprintf("%-10s", discovery.Platform))

	// Scope styling
	var scopeStyle lipgloss.Style
	switch discovery.Scope {
	case "global":
		scopeStyle = lipgloss.NewStyle().Foreground(theme.Current.Success)
	case "project":
		scopeStyle = lipgloss.NewStyle().Foreground(theme.Current.Info)
	default:
		scopeStyle = lipgloss.NewStyle().Foreground(theme.Current.TextMuted)
	}
	scope := scopeStyle.Render(discovery.Scope)

	// Build row
	rowStyle := lipgloss.NewStyle().MarginLeft(2)
	if isSelected {
		rowStyle = rowStyle.Background(theme.Current.Surface)
	}

	return rowStyle.Render(indicator + name + "  " + platform + "  " + scope)
}

// renderDiscoveredEmptyState renders the empty state for discovered skills.
func (mv *ManageView) renderDiscoveredEmptyState() string {
	emptyStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		Italic(true).
		MarginLeft(4).
		MarginTop(2)

	helpStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		MarginLeft(4).
		MarginTop(1)

	return emptyStyle.Render("No external skills discovered.") + "\n" +
		helpStyle.Render("Skills not managed by Skulto will appear here when found.")
}

// getDiscoveredColumnWidth calculates the width for the name column in discovered section.
func (mv *ManageView) getDiscoveredColumnWidth() int {
	minWidth := len("NAME")
	maxWidth := 30

	for _, d := range mv.discoveries {
		if len(d.Name) > minWidth {
			minWidth = len(d.Name)
		}
	}

	if minWidth > maxWidth {
		return maxWidth
	}
	return minWidth
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

	// Add local badge if this is a local skill
	if skill.IsLocal {
		localBadge := lipgloss.NewStyle().
			Foreground(theme.Current.Warning).
			Render(" [local]")
		skillName += localBadge
	}

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
	// Count display based on current section
	countStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Primary).
		Bold(true).
		MarginLeft(2)

	count := ""
	if mv.section == ManageSectionDiscovered {
		if len(mv.discoveries) > 0 {
			count = countStyle.Render(fmt.Sprintf("%d skill(s) discovered", len(mv.discoveries)))
		}
	} else {
		if len(mv.skills) > 0 {
			count = countStyle.Render(fmt.Sprintf("%d skill(s) installed", len(mv.skills)))
		}
	}

	// Navigation help - include Tab for section switching
	helpStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		MarginLeft(2).
		MarginTop(1)

	enterAction := "edit"
	if mv.section == ManageSectionDiscovered {
		enterAction = "import"
	}
	help := helpStyle.Render(fmt.Sprintf("↑↓ navigate • Tab switch sections • Enter %s • S save • Esc back", enterAction))

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
			{Key: "Tab", Description: "Switch sections"},
			{Key: "↑↓, k/j", Description: "Navigate skills"},
			{Key: "d/u", Description: "Page down/up"},
			{Key: "g/G", Description: "Jump to start/end"},
			{Key: "Enter", Description: "Edit skill locations"},
			{Key: "S", Description: "Save project skills to skulto.json"},
			{Key: "Esc, q", Description: "Return to Home"},
		},
	}
}
