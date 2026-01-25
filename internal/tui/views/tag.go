package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/tui/components"
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TagView displays all skills for a selected tag
type TagView struct {
	db  *db.DB
	cfg *config.Config

	// Data
	tag            *models.Tag
	allSkills      []models.Skill // All skills (sorted alphabetically)
	filteredSkills []models.Skill // Filtered based on search
	selectedIdx    int
	loading        bool
	searchActive   bool // Whether search mode is on
	searchQuery    string
	searchBar      *components.SearchBar

	// Layout
	width  int
	height int

	// Viewport (for scrolling if many skills)
	scrollOffset int
}

// SkillsLoadedMsg is sent when skills finish loading
type SkillsLoadedMsg struct {
	tag    *models.Tag
	skills []models.Skill
	err    error
}

// NewTagView creates a new tag view
func NewTagView(database *db.DB, conf *config.Config) *TagView {
	return &TagView{
		db:          database,
		cfg:         conf,
		selectedIdx: 0,
		loading:     true,
		searchBar:   components.NewSearchBar(),
	}
}

// Init initializes the tag view
func (tv *TagView) Init() {
	tv.selectedIdx = 0
	tv.scrollOffset = 0
	tv.loading = true
	tv.tag = nil
	tv.allSkills = nil
	tv.filteredSkills = nil
	tv.searchActive = false
	tv.searchQuery = ""
	tv.searchBar.Clear()
}

// SetTag sets the tag and loads skills asynchronously
func (tv *TagView) SetTag(tag *models.Tag) tea.Cmd {
	tv.tag = tag
	tv.loading = true
	tv.selectedIdx = 0
	tv.scrollOffset = 0
	tv.searchActive = false
	tv.searchQuery = ""
	tv.searchBar.Clear()
	tv.allSkills = nil
	tv.filteredSkills = nil

	return func() tea.Msg {
		skills, err := tv.db.GetSkillsByTag(tag.Slug, 100, 0)
		return SkillsLoadedMsg{
			tag:    tag,
			skills: skills,
			err:    err,
		}
	}
}

// HandleSkillsLoaded processes the async load result
func (tv *TagView) HandleSkillsLoaded(msg SkillsLoadedMsg) {
	tv.loading = false // Always stop loading indicator
	if msg.err == nil {
		// Sort skills alphabetically by title
		sort.Slice(msg.skills, func(i, j int) bool {
			return strings.ToLower(msg.skills[i].Title) < strings.ToLower(msg.skills[j].Title)
		})
		tv.allSkills = msg.skills
		tv.filteredSkills = msg.skills
	} else {
		// On error, set empty skills (will show "No skills found" message)
		tv.allSkills = nil
		tv.filteredSkills = nil
	}
}

// Update handles user input
// Returns (shouldGoBack, shouldOpenDetail)
func (tv *TagView) Update(key string) (bool, bool) {
	// If search is active, handle search input
	if tv.searchActive && key != "esc" && key != "enter" {
		return tv.updateSearch(key)
	}

	switch key {
	case "up", "k":
		if tv.selectedIdx > 0 {
			tv.selectedIdx--
			tv.adjustScroll()
		}
		return false, false

	case "down", "j":
		if tv.selectedIdx < len(tv.filteredSkills)-1 {
			tv.selectedIdx++
			tv.adjustScroll()
		}
		return false, false

	case "enter":
		return false, true // Signal to open detail view

	case "esc":
		// If search active, clear it; otherwise go back
		if tv.searchActive {
			tv.searchActive = false
			tv.searchQuery = ""
			tv.searchBar.Clear()
			tv.filteredSkills = tv.allSkills
			tv.selectedIdx = 0
			tv.scrollOffset = 0
			return false, false
		}
		return true, false // Go back to home

	case "/":
		// Start search mode
		tv.searchActive = true
		tv.searchQuery = ""
		tv.searchBar.Clear()
		tv.searchBar.Focus()
		return false, false

	default:
		return false, false
	}
}

// updateSearch handles input while search is active
func (tv *TagView) updateSearch(key string) (bool, bool) {
	switch key {
	case "up", "k":
		if tv.selectedIdx > 0 {
			tv.selectedIdx--
			tv.adjustScroll()
		}
		return false, false

	case "down", "j":
		if tv.selectedIdx < len(tv.filteredSkills)-1 {
			tv.selectedIdx++
			tv.adjustScroll()
		}
		return false, false

	default:
		// Delegate to search bar for text input
		keyMsg := tv.stringToKeyMsg(key)
		tv.searchBar.HandleKey(keyMsg)

		// Update search query and filter results
		newQuery := tv.searchBar.Value()
		if newQuery != tv.searchQuery {
			tv.searchQuery = newQuery
			tv.filterSkills()
			tv.selectedIdx = 0
			tv.scrollOffset = 0
		}
		return false, false
	}
}

// stringToKeyMsg converts a string key representation to a tea.KeyMsg
func (tv *TagView) stringToKeyMsg(key string) tea.KeyMsg {
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
		if len(key) == 1 {
			return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		}
		return tea.KeyMsg{}
	}
}

// filterSkills filters allSkills based on searchQuery
func (tv *TagView) filterSkills() {
	if tv.searchQuery == "" {
		tv.filteredSkills = tv.allSkills
		return
	}

	query := strings.ToLower(tv.searchQuery)
	var filtered []models.Skill

	for _, skill := range tv.allSkills {
		// Search in title, description, author, and category
		if strings.Contains(strings.ToLower(skill.Title), query) ||
			strings.Contains(strings.ToLower(skill.Description), query) ||
			strings.Contains(strings.ToLower(skill.Author), query) ||
			strings.Contains(strings.ToLower(skill.Category), query) {
			filtered = append(filtered, skill)
		}
	}

	tv.filteredSkills = filtered
}

// GetSelectedSkill returns the currently selected skill
func (tv *TagView) GetSelectedSkill() *models.Skill {
	if tv.selectedIdx >= 0 && tv.selectedIdx < len(tv.filteredSkills) {
		return &tv.filteredSkills[tv.selectedIdx]
	}
	return nil
}

// View renders the tag view with fixed header and scrollable items.
func (tv *TagView) View() string {
	if tv.loading {
		return tv.renderLoading()
	}

	if len(tv.filteredSkills) == 0 {
		if tv.searchActive {
			return tv.renderTagHeader() + "\n\n" + tv.searchBar.View() + "\n\n" + tv.renderEmpty()
		}
		return tv.renderTagHeader() + "\n\n" + tv.renderEmpty()
	}

	// Build layout: fixed header + search bar (optional) + scrollable items + fixed footer
	var layoutParts []string

	// Add fixed tag header (tag title + blanks + count = 4 lines)
	layoutParts = append(layoutParts, tv.renderTagHeader())
	layoutParts = append(layoutParts, "")
	tagHeaderLines := 4

	// Add search bar if search is active (3 lines: label + box + blank)
	searchBarLines := 0
	if tv.searchActive {
		layoutParts = append(layoutParts, tv.searchBar.View())
		layoutParts = append(layoutParts, "")
		searchBarLines = 3
	}

	// Render scrollable items
	itemsContent := tv.renderSkillsOnly()
	itemsLines := strings.Split(itemsContent, "\n")

	// Render footer (1 line for scroll info)
	footerView := tv.renderScrollFooter()
	footerHeight := 1

	// Pad items to fill available space (leaving room for all fixed sections)
	availableItemsHeight := tv.height - tagHeaderLines - footerHeight - searchBarLines
	if availableItemsHeight < 0 {
		availableItemsHeight = 0
	}

	for len(itemsLines) < availableItemsHeight {
		itemsLines = append(itemsLines, "")
	}

	// Build final layout
	layoutParts = append(layoutParts, strings.Join(itemsLines, "\n"))
	return strings.Join(layoutParts, "\n") + "\n" + footerView
}

// renderTagHeader renders the fixed tag header with title and count
func (tv *TagView) renderTagHeader() string {
	// Tag title
	tagTitleStyle := lipgloss.NewStyle().
		Foreground(theme.GetTagColor(tv.tag.Category)).
		Bold(true).
		MarginLeft(1)

	tagTitle := tagTitleStyle.Render(tv.tag.Name)

	// Result count style
	countStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		Italic(true).
		MarginLeft(1)

	// Show range if not all results visible
	var countText string
	if len(tv.filteredSkills) > 0 {
		countText = fmt.Sprintf("Found %d results:", len(tv.filteredSkills))
	} else {
		countText = "No results"
	}
	count := countStyle.Render(countText)

	return tagTitle + "\n\n" + count
}

// SetSize updates dimensions
func (tv *TagView) SetSize(w, h int) {
	tv.width = w
	tv.height = h
	tv.searchBar.SetWidth(w - 4)
}

// adjustScroll ensures selected item is visible
func (tv *TagView) adjustScroll() {
	// Fixed sections:
	// - Search bar (if active): 3 lines
	// - Tag header: 4 lines
	// - Blank: 1 line
	// - Footer: 1 line
	// Each result takes ~4 lines
	tagHeaderLines := 4
	blankLine := 1
	footerLine := 1
	resultLines := 4

	searchBarLines := 0
	if tv.searchActive {
		searchBarLines = 3
	}

	availableHeight := tv.height - tagHeaderLines - blankLine - footerLine - searchBarLines
	if availableHeight < resultLines {
		availableHeight = resultLines
	}

	visibleResults := availableHeight / resultLines
	if visibleResults < 1 {
		visibleResults = 1
	}

	// Scroll down if selected is below viewport
	if tv.selectedIdx >= tv.scrollOffset+visibleResults {
		tv.scrollOffset = tv.selectedIdx - visibleResults + 1
	}

	// Scroll up if selected is above viewport
	if tv.selectedIdx < tv.scrollOffset {
		tv.scrollOffset = tv.selectedIdx
	}
}

// renderLoading shows loading state
func (tv *TagView) renderLoading() string {
	loadingStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Accent).
		Bold(true).
		Align(lipgloss.Center).
		Width(tv.width)

	return loadingStyle.Render("Loading skills for tag...")
}

// renderEmpty shows empty state
func (tv *TagView) renderEmpty() string {
	emptyStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		Italic(true).
		Align(lipgloss.Center).
		Width(tv.width).
		MarginTop(2)

	return emptyStyle.Render("No skills found for this tag")
}

// renderScrollFooter shows scrolling information at the bottom.
func (tv *TagView) renderScrollFooter() string {
	tagHeaderLines := 4
	blankLine := 1
	footerLine := 1
	resultLines := 4

	searchBarLines := 0
	if tv.searchActive {
		searchBarLines = 3
	}

	availableHeight := tv.height - tagHeaderLines - blankLine - footerLine - searchBarLines
	if availableHeight < resultLines {
		availableHeight = resultLines
	}

	maxVisible := availableHeight / resultLines
	if maxVisible < 1 {
		maxVisible = 1
	}

	startIdx := tv.scrollOffset
	endIdx := startIdx + maxVisible
	if endIdx > len(tv.filteredSkills) {
		endIdx = len(tv.filteredSkills)
	}

	footerStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		Padding(0, 1).
		Width(tv.width)

	var footerText string
	if len(tv.filteredSkills) > maxVisible {
		// Show range and scroll indicators
		footerText = fmt.Sprintf("Showing %d-%d of %d  (↑↓ to scroll, / filter, esc to back)", startIdx+1, endIdx, len(tv.filteredSkills))
	} else {
		// Show total count
		footerText = fmt.Sprintf("Found %d result%s  (/ filter, esc to back)", len(tv.filteredSkills), map[bool]string{true: "", false: "s"}[len(tv.filteredSkills) == 1])
	}

	return footerStyle.Render(footerText)
}

// renderSkillsOnly shows only the skill list items (without header)
func (tv *TagView) renderSkillsOnly() string {
	// Calculate maximum visible results based on viewport height
	// Layout: search bar (optional, 3) + tag header (fixed, 4) + blank (1) + items + footer (1)
	tagHeaderLines := 4 // fixed header
	blankLine := 1      // blank line between header and items
	footerLine := 1     // scroll footer
	resultLines := 4    // Approximate lines per result item

	searchBarLines := 0
	if tv.searchActive {
		searchBarLines = 3
	}

	availableHeight := tv.height - tagHeaderLines - blankLine - footerLine - searchBarLines
	if availableHeight < resultLines {
		availableHeight = resultLines
	}

	maxVisible := availableHeight / resultLines
	if maxVisible < 1 {
		maxVisible = 1
	}
	if maxVisible > len(tv.filteredSkills) {
		maxVisible = len(tv.filteredSkills)
	}

	// Use scroll offset maintained by adjustScroll()
	startIdx := tv.scrollOffset
	endIdx := startIdx + maxVisible

	// Ensure bounds are valid
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx > len(tv.filteredSkills) {
		endIdx = len(tv.filteredSkills)
	}
	if startIdx > endIdx {
		startIdx = endIdx
	}

	var items []string
	for i := startIdx; i < endIdx; i++ {
		skill := tv.filteredSkills[i]
		if i == tv.selectedIdx {
			items = append(items, components.RenderSelectedSkill(skill, components.DetailedStyle))
		} else {
			items = append(items, components.RenderSkillItem(skill, components.DetailedStyle))
		}
	}

	return strings.Join(items, "\n")
}

// HandleMouse handles mouse events for scrolling.
func (tv *TagView) HandleMouse(msg tea.MouseMsg) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if tv.selectedIdx > 0 {
			tv.selectedIdx--
			tv.adjustScroll()
		}
	case tea.MouseButtonWheelDown:
		if tv.selectedIdx < len(tv.filteredSkills)-1 {
			tv.selectedIdx++
			tv.adjustScroll()
		}
	}
}

// GetKeyboardCommands returns the keyboard commands for this view.
func (tv *TagView) GetKeyboardCommands() ViewCommands {
	return ViewCommands{
		ViewName: "Tag",
		Commands: []Command{
			{Key: "↑↓, k/j", Description: "Navigate skills in tag"},
			{Key: "/", Description: "Filter skills within tag"},
			{Key: "Enter", Description: "Select skill to view details"},
			{Key: "Esc", Description: "Go back to home or clear filter"},
		},
	}
}
