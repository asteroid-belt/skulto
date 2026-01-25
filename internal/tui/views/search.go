package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/search"
	"github.com/asteroid-belt/skulto/internal/telemetry"
	"github.com/asteroid-belt/skulto/internal/tui/components"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MinSearchChars is the minimum number of characters required to trigger a search.
// This prevents overwhelming the UI with results from overly broad queries.
const MinSearchChars = 3

// FocusArea indicates which part of the search view has keyboard focus.
type FocusArea int

const (
	FocusSearchBar FocusArea = iota
	FocusTagGrid
	FocusResults
)

// SearchView displays search results for a query.
type SearchView struct {
	db        *db.DB
	cfg       *config.Config
	searchBar *components.SearchBar
	searchSvc *search.Service
	telemetry telemetry.Client

	query         string
	results       *search.SearchResults
	legacyResults []models.Skill // Fallback when search service unavailable
	selectedIdx   int
	scrollOffset  int // Maintains scroll position in results
	loading       bool
	lastSearchAt  time.Time
	searchDelay   time.Duration

	// Unified result list for all search results
	resultList *components.UnifiedResultList

	// Tag browsing mode
	tagGrid   *components.TagGrid
	focusArea FocusArea
	allTags   []models.Tag

	// Telemetry tracking state
	lastTrackedQuery    string
	lastSearchTrackTime time.Time
	searchTrackThrottle time.Duration
	wasInTagMode        bool

	width  int
	height int
}

// NewSearchView creates a new search view.
// The searchSvc parameter can be nil - in that case, legacy FTS-only search is used.
func NewSearchView(database *db.DB, conf *config.Config, searchSvc *search.Service) *SearchView {
	// Create a searchbar with default styles
	searchBar := components.NewSearchBar()

	return &SearchView{
		db:                  database,
		cfg:                 conf,
		searchBar:           searchBar,
		searchSvc:           searchSvc,
		selectedIdx:         0,
		loading:             false,
		searchDelay:         200 * time.Millisecond,
		searchTrackThrottle: 2 * time.Second, // Throttle search tracking to once per 2 seconds
		resultList:          components.NewUnifiedResultList(),
		tagGrid:             components.NewTagGrid(),
		focusArea:           FocusSearchBar,
	}
}

// Init initializes the search view and resets state.
func (sv *SearchView) Init(tc telemetry.Client) {
	sv.telemetry = tc
	sv.searchBar.Focus()
	sv.searchBar.Clear()
	sv.query = ""
	sv.results = nil
	sv.legacyResults = nil
	sv.selectedIdx = 0
	sv.scrollOffset = 0
	sv.resultList.Clear()
	sv.focusArea = FocusSearchBar
	sv.tagGrid.SetFocused(false)

	// Reset telemetry tracking state
	sv.lastTrackedQuery = ""
	sv.wasInTagMode = false

	// Load all tags for browsing
	sv.loadAllTags()
}

// loadAllTags fetches all tags from the database.
func (sv *SearchView) loadAllTags() {
	tags, err := sv.db.ListTags("")
	if err == nil {
		sv.allTags = tags
		sv.tagGrid.SetTags(tags)
	}
}

// Update handles user input and search logic.
// Returns (shouldGoBack, tagSelected, cmd).
// tagSelected is true when user presses Enter on a tag (app.go should navigate to TagView).
func (sv *SearchView) Update(key string) (bool, bool, tea.Cmd) {
	// Check if we're in tag browsing mode (query < 3 chars)
	inTagMode := len(sv.query) < MinSearchChars

	if inTagMode {
		return sv.updateTagMode(key)
	}
	return sv.updateSearchMode(key)
}

// updateTagMode handles input when displaying tags (query < 3 chars).
func (sv *SearchView) updateTagMode(key string) (bool, bool, tea.Cmd) {
	switch sv.focusArea {
	case FocusSearchBar:
		return sv.updateSearchBarInTagMode(key)
	case FocusTagGrid:
		return sv.updateTagGrid(key)
	}
	return false, false, nil
}

// updateSearchBarInTagMode handles input when search bar is focused in tag mode.
func (sv *SearchView) updateSearchBarInTagMode(key string) (bool, bool, tea.Cmd) {
	switch key {
	case "tab", "down":
		// Move focus to tag grid
		sv.focusArea = FocusTagGrid
		sv.searchBar.Blur()
		sv.tagGrid.SetFocused(true)
		// Track tag browsing entry
		if !sv.wasInTagMode {
			sv.telemetry.TrackTagBrowsingEntered(len(sv.allTags))
			sv.wasInTagMode = true
		}
		return false, false, nil

	case "esc":
		return true, false, nil // Return to home

	case "enter":
		// No action when search bar focused with < 3 chars
		return false, false, nil

	default:
		// Send to search bar
		keyMsg := sv.stringToKeyMsg(key)
		cmd := sv.searchBar.HandleKey(keyMsg)
		sv.UpdateQuery(sv.searchBar.Value())
		return false, false, cmd
	}
}

// updateTagGrid handles input when tag grid is focused.
func (sv *SearchView) updateTagGrid(key string) (bool, bool, tea.Cmd) {
	switch key {
	case "left", "h":
		sv.tagGrid.MoveLeft()
		return false, false, nil

	case "right", "l", "tab":
		sv.tagGrid.MoveRight()
		return false, false, nil

	case "up", "k":
		atTop := sv.tagGrid.MoveUp()
		if atTop {
			// Move focus back to search bar
			sv.focusArea = FocusSearchBar
			sv.searchBar.Focus()
			sv.tagGrid.SetFocused(false)
		}
		return false, false, nil

	case "down", "j":
		sv.tagGrid.MoveDown()
		return false, false, nil

	case "enter":
		// Signal that a tag was selected
		if tag := sv.tagGrid.GetSelectedTag(); tag != nil {
			sv.telemetry.TrackTagSelected(tag.Name)
			return false, true, nil
		}
		return false, false, nil

	case "esc":
		return true, false, nil

	default:
		// Printable character: refocus search bar and type
		if len(key) == 1 && key[0] >= 32 && key[0] <= 126 {
			sv.focusArea = FocusSearchBar
			sv.searchBar.Focus()
			sv.tagGrid.SetFocused(false)
			keyMsg := sv.stringToKeyMsg(key)
			cmd := sv.searchBar.HandleKey(keyMsg)
			sv.UpdateQuery(sv.searchBar.Value())
			return false, false, cmd
		}
		return false, false, nil
	}
}

// updateSearchMode handles input when displaying search results (query >= 3 chars).
func (sv *SearchView) updateSearchMode(key string) (bool, bool, tea.Cmd) {
	// Keep existing search mode logic
	if sv.searchBar.Focused() {
		switch key {
		case "up", "down":
			if key == "up" {
				sv.navigateUp()
			} else {
				sv.navigateDown()
			}
			return false, false, nil

		case "tab":
			sv.toggleExpand()
			return false, false, nil

		case "enter":
			return false, false, nil

		case "esc":
			return true, false, nil

		default:
			keyMsg := sv.stringToKeyMsg(key)
			cmd := sv.searchBar.HandleKey(keyMsg)
			sv.UpdateQuery(sv.searchBar.Value())
			return false, false, cmd
		}
	}

	switch key {
	case "up", "k":
		sv.navigateUp()
		return false, false, nil

	case "down", "j":
		sv.navigateDown()
		return false, false, nil

	case "tab":
		sv.toggleExpand()
		return false, false, nil

	case "enter":
		return false, false, nil

	case "esc":
		return true, false, nil

	default:
		sv.searchBar.Focus()
		keyMsg := sv.stringToKeyMsg(key)
		cmd := sv.searchBar.HandleKey(keyMsg)
		sv.UpdateQuery(sv.searchBar.Value())
		return false, false, cmd
	}
}

// navigateUp moves selection up in the result list.
func (sv *SearchView) navigateUp() {
	if sv.results != nil {
		sv.resultList.MoveUp()
	} else {
		// Legacy mode
		if sv.selectedIdx > 0 {
			sv.selectedIdx--
			sv.adjustScroll()
		}
	}
}

// navigateDown moves selection down in the result list.
func (sv *SearchView) navigateDown() {
	if sv.results != nil {
		sv.resultList.MoveDown()
	} else {
		// Legacy mode
		maxIdx := len(sv.legacyResults) - 1
		if sv.selectedIdx < maxIdx {
			sv.selectedIdx++
			sv.adjustScroll()
		}
	}
}

// toggleExpand toggles the expansion of the selected item's snippets.
func (sv *SearchView) toggleExpand() {
	if sv.results != nil {
		sv.resultList.ToggleExpand()
	}
}

// stringToKeyMsg converts a string key representation to a tea.KeyMsg.
// This bridges the gap between app.go's string-based key passing
// and bubbletea's KeyMsg-based input handling.
func (sv *SearchView) stringToKeyMsg(key string) tea.KeyMsg {
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
		// For regular characters, create a Runes message
		if len(key) == 1 {
			return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		}
		// Unknown key, return empty
		return tea.KeyMsg{}
	}
}

// AppendValue adds a character to the search bar.
func (sv *SearchView) AppendValue(s string) {
	sv.searchBar.AppendValue(s)
}

// UpdateQuery updates the search query and triggers a search.
// Search only executes when query length >= MinSearchChars.
// Optimized to skip if query hasn't changed (e.g. cursor movement).
func (sv *SearchView) UpdateQuery(q string) {
	// Optimization: If query hasn't changed (e.g. only cursor moved), do nothing
	if sv.query == q {
		return
	}

	sv.query = q

	// Clear results if query is empty or below minimum
	if len(q) < MinSearchChars {
		sv.clearResults()
		// Stay in search bar focus when clearing
		if sv.focusArea == FocusResults {
			sv.focusArea = FocusSearchBar
		}
		return
	}

	// Entering search mode - ensure focus is on search bar
	sv.focusArea = FocusSearchBar
	sv.tagGrid.SetFocused(false)
	sv.searchBar.Focus()

	// Debounce search
	if time.Since(sv.lastSearchAt) < sv.searchDelay {
		return
	}

	sv.PerformSearch()
}

// clearResults clears all search results.
func (sv *SearchView) clearResults() {
	sv.results = nil
	sv.legacyResults = nil
	sv.selectedIdx = 0
	sv.resultList.Clear()
}

// PerformSearch executes the search query.
func (sv *SearchView) PerformSearch() {
	if len(sv.query) < MinSearchChars {
		sv.clearResults()
		return
	}

	sv.loading = true
	sv.lastSearchAt = time.Now()

	// Use search service if available (hybrid FTS + semantic search)
	if sv.searchSvc != nil {
		sv.performHybridSearch()
	} else {
		sv.performLegacySearch()
	}

	sv.loading = false
	sv.selectedIdx = 0
	sv.scrollOffset = 0 // Reset scroll position for new results
}

// performHybridSearch uses the search service for hybrid FTS + semantic search.
func (sv *SearchView) performHybridSearch() {
	ctx := context.Background()

	// Determine if semantic search should be enabled
	semanticEnabled := sv.searchSvc.HasVectorStore()

	opts := search.SearchOptions{
		Limit:           50,
		Threshold:       0.5, // Slightly lower threshold for better recall
		IncludeFTS:      true,
		IncludeSemantic: semanticEnabled,
	}

	results, err := sv.searchSvc.Search(ctx, sv.query, opts)
	if err != nil || results == nil {
		// Fall back to legacy search on error
		sv.performLegacySearch()
		return
	}

	sv.results = results
	sv.legacyResults = nil
	sv.updateResultList()

	// Track search performed (throttled to avoid excessive events)
	if time.Since(sv.lastSearchTrackTime) >= sv.searchTrackThrottle {
		resultCount := 0
		if results != nil {
			resultCount = len(results.TitleMatches) + len(results.ContentMatches)
		}
		sv.telemetry.TrackSearchPerformed(sv.query, resultCount, "hybrid")
		sv.lastSearchTrackTime = time.Now()
		sv.lastTrackedQuery = sv.query
	}
}

// performLegacySearch uses database FTS search directly.
func (sv *SearchView) performLegacySearch() {
	results, err := sv.db.SearchSkills(sv.query, 50)
	if err != nil {
		sv.legacyResults = nil
	} else {
		sv.legacyResults = results
	}
	sv.results = nil
	sv.resultList.Clear()

	// Track search performed (throttled to avoid excessive events)
	if time.Since(sv.lastSearchTrackTime) >= sv.searchTrackThrottle {
		resultCount := len(sv.legacyResults)
		sv.telemetry.TrackSearchPerformed(sv.query, resultCount, "fts")
		sv.lastSearchTrackTime = time.Now()
		sv.lastTrackedQuery = sv.query
	}
}

// updateResultList populates the unified result list from search results.
func (sv *SearchView) updateResultList() {
	if sv.results == nil {
		return
	}

	var items []components.UnifiedResultItem

	// Add name/tag matches first (they appear at top)
	for _, match := range sv.results.TitleMatches {
		items = append(items, components.UnifiedResultItem{
			Skill:     match.Skill,
			MatchType: components.MatchTypeName,
			Snippets:  nil,
			Expanded:  false,
		})
	}

	// Add content matches
	for _, match := range sv.results.ContentMatches {
		items = append(items, components.UnifiedResultItem{
			Skill:     match.Skill,
			MatchType: components.MatchTypeContent,
			Snippets:  match.Snippets,
			Expanded:  false,
		})
	}

	sv.resultList.SetItems(items)
}

// adjustScroll ensures the selected item remains visible
func (sv *SearchView) adjustScroll() {
	// Fixed sections:
	// - Search header: 2 lines
	// - Search bar: 3 lines
	// - Footer: 1 line
	// Each result takes ~4 lines
	searchHeaderLines := 2
	searchBarLines := 3
	footerLine := 1
	resultLines := 4

	availableHeight := max(resultLines, sv.height-searchHeaderLines-searchBarLines-footerLine)
	visibleResults := max(1, availableHeight/resultLines)

	// Scroll down if selected is below viewport
	if sv.selectedIdx >= sv.scrollOffset+visibleResults {
		sv.scrollOffset = sv.selectedIdx - visibleResults + 1
	}

	// Scroll up if selected is above viewport
	if sv.selectedIdx < sv.scrollOffset {
		sv.scrollOffset = sv.selectedIdx
	}
}

// GetSearchBar returns the search bar component.
func (sv *SearchView) GetSearchBar() *components.SearchBar {
	return sv.searchBar
}

// GetSelectedSkill returns the currently selected skill from search results.
func (sv *SearchView) GetSelectedSkill() *models.Skill {
	// Unified result list mode
	if sv.results != nil {
		item := sv.resultList.GetSelectedItem()
		if item != nil {
			return &item.Skill
		}
		return nil
	}

	// Legacy mode
	if sv.selectedIdx >= 0 && sv.selectedIdx < len(sv.legacyResults) {
		return &sv.legacyResults[sv.selectedIdx]
	}
	return nil
}

// GetSelectedTag returns the currently selected tag (when in tag mode).
func (sv *SearchView) GetSelectedTag() *models.Tag {
	if len(sv.query) < MinSearchChars {
		return sv.tagGrid.GetSelectedTag()
	}
	return nil
}

// View renders the search view with fixed header and footer.
func (sv *SearchView) View() string {
	// Build layout: fixed header + search bar + scrollable results + fixed footer
	var layoutParts []string

	// Add fixed search header (2 lines for title + blank)
	layoutParts = append(layoutParts, "")
	searchHeaderLines := 1

	// Add search bar (3 lines: label + box + blank)
	layoutParts = append(layoutParts, sv.searchBar.View())
	layoutParts = append(layoutParts, "")
	searchBarLines := 3

	// Handle different states
	if sv.loading {
		layoutParts = append(layoutParts, sv.renderLoading())
		return strings.Join(layoutParts, "\n")
	}

	// Show tags when query is below threshold
	if len(sv.query) < MinSearchChars {
		return sv.renderTagBrowsingView(layoutParts, searchHeaderLines, searchBarLines)
	}

	// Check if we have any results
	hasResults := sv.hasAnyResults()
	if !hasResults {
		var emptyMsg string
		if len(sv.query) == 0 {
			emptyMsg = "Start typing to search..."
		} else if len(sv.query) < MinSearchChars {
			emptyMsg = fmt.Sprintf("Type at least %d characters to search...", MinSearchChars)
		} else {
			emptyMsg = "No results found for: " + sv.query
		}
		layoutParts = append(layoutParts, sv.renderEmptyState(emptyMsg))
		return strings.Join(layoutParts, "\n")
	}

	// Render scrollable results
	resultsContent := sv.renderResultsOnly()
	resultsLines := strings.Split(resultsContent, "\n")

	// Render footer (1 line for scroll info)
	footerView := sv.renderSearchFooter()
	footerHeight := 1

	// Pad results to fill available space (leaving room for all fixed sections)
	availableResultsHeight := max(0, sv.height-searchHeaderLines-searchBarLines-footerHeight)

	for len(resultsLines) < availableResultsHeight {
		resultsLines = append(resultsLines, "")
	}

	// Build final layout
	layoutParts = append(layoutParts, strings.Join(resultsLines, "\n"))
	return strings.Join(layoutParts, "\n") + "\n" + footerView
}

// hasAnyResults returns true if there are any search results.
func (sv *SearchView) hasAnyResults() bool {
	if sv.results != nil {
		return len(sv.results.TitleMatches) > 0 || len(sv.results.ContentMatches) > 0
	}
	return len(sv.legacyResults) > 0
}

// SetSize updates the view dimensions.
func (sv *SearchView) SetSize(w, h int) {
	sv.width = w
	sv.height = h
	sv.searchBar.SetWidth(w - 4)
	// Reserve space for search bar (3 lines) + footer (1 line) + title (1 line)
	tagGridHeight := h - 5
	sv.tagGrid.SetSize(w, tagGridHeight)
}

// renderLoading shows a loading indicator.
func (sv *SearchView) renderLoading() string {
	loadingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F1C40F")).
		Italic(true).
		MarginLeft(2)

	return loadingStyle.Render("Searching...")
}

// renderResultsOnly renders only the search results items (without count)
func (sv *SearchView) renderResultsOnly() string {
	// Unified result list mode
	if sv.results != nil {
		// Set size for proper viewport calculation
		headerLines := 4 // padding + search bar + padding
		footerLines := 1
		availableHeight := sv.height - headerLines - footerLines
		sv.resultList.SetSize(sv.width, availableHeight)
		return sv.resultList.View()
	}

	// Legacy mode
	return sv.renderLegacyResults()
}

// renderLegacyResults renders results in the legacy FTS-only mode.
func (sv *SearchView) renderLegacyResults() string {
	if len(sv.legacyResults) == 0 {
		return ""
	}

	// Calculate maximum visible results based on viewport height
	searchHeaderLines := 2
	searchBarLines := 3
	footerLine := 1
	resultLines := 4 // Approximate lines per result item

	availableHeight := max(resultLines, sv.height-searchHeaderLines-searchBarLines-footerLine)
	maxVisible := min(len(sv.legacyResults), max(1, availableHeight/resultLines))

	// Use scroll offset maintained by adjustScroll()
	startIdx := sv.scrollOffset
	endIdx := startIdx + maxVisible

	// Ensure bounds are valid
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx > len(sv.legacyResults) {
		endIdx = len(sv.legacyResults)
	}
	if startIdx > endIdx {
		startIdx = endIdx
	}

	var items []string
	for i := startIdx; i < endIdx; i++ {
		skill := sv.legacyResults[i]
		if i == sv.selectedIdx {
			items = append(items, components.RenderSelectedSkill(skill, components.DetailedStyle))
		} else {
			items = append(items, components.RenderSkillItem(skill, components.DetailedStyle))
		}
	}

	return strings.Join(items, "\n")
}

// renderSearchFooter shows scrolling information at the bottom.
func (sv *SearchView) renderSearchFooter() string {
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B6B6B")).
		Padding(0, 1).
		Width(sv.width)

	var footerText string

	// Unified result list mode
	if sv.results != nil {
		totalHits := len(sv.results.TitleMatches) + len(sv.results.ContentMatches)
		nameCount := len(sv.results.TitleMatches)
		contentCount := len(sv.results.ContentMatches)

		if totalHits == 0 {
			footerText = "No results  (esc to back)"
		} else {
			footerText = fmt.Sprintf("%d result%s (%d name, %d content)  Tab: expand snippets • esc: back",
				totalHits,
				map[bool]string{true: "", false: "s"}[totalHits == 1],
				nameCount,
				contentCount)
		}
	} else {
		// Legacy mode
		totalResults := len(sv.legacyResults)
		if totalResults == 0 {
			footerText = "No results  (esc to back)"
		} else {
			footerText = fmt.Sprintf("Found %d result%s  (↑↓ to scroll, esc to back)",
				totalResults,
				map[bool]string{true: "", false: "s"}[totalResults == 1])
		}
	}

	return footerStyle.Render(footerText)
}

// renderEmptyState renders an empty state message.
func (sv *SearchView) renderEmptyState(msg string) string {
	emptyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B6B6B")).
		Italic(true).
		MarginLeft(2)

	return emptyStyle.Render(msg)
}

// renderTagBrowsingView renders the tag browsing mode.
func (sv *SearchView) renderTagBrowsingView(layoutParts []string, _, _ int) string {
	// Title for tag section
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F1C40F")).
		Bold(true).
		MarginLeft(2)

	layoutParts = append(layoutParts, titleStyle.Render("Browse All Tags"))
	layoutParts = append(layoutParts, "")

	// Tag grid
	layoutParts = append(layoutParts, sv.tagGrid.View())

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B6B6B")).
		Padding(0, 1).
		Width(sv.width)

	var footerText string
	if sv.focusArea == FocusTagGrid {
		footerText = fmt.Sprintf("%d tags • ←→: navigate • Enter: view tag • ↑: back to search • Esc: home",
			len(sv.allTags))
	} else {
		footerText = fmt.Sprintf("%d tags • Tab/↓: browse tags • Type to search • Esc: home",
			len(sv.allTags))
	}

	// Calculate padding needed
	content := strings.Join(layoutParts, "\n")
	contentLines := strings.Count(content, "\n") + 1
	footerHeight := 1
	padding := sv.height - contentLines - footerHeight
	if padding > 0 {
		layoutParts = append(layoutParts, strings.Repeat("\n", padding-1))
	}

	return strings.Join(layoutParts, "\n") + "\n" + footerStyle.Render(footerText)
}

// HandleMouse handles mouse events for scrolling.
func (sv *SearchView) HandleMouse(msg tea.MouseMsg) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		sv.navigateUp()
	case tea.MouseButtonWheelDown:
		sv.navigateDown()
	}
}

// GetKeyboardCommands returns the keyboard commands for this view.
func (sv *SearchView) GetKeyboardCommands() ViewCommands {
	if len(sv.query) < MinSearchChars {
		return ViewCommands{
			ViewName: "Search (Tag Browse)",
			Commands: []Command{
				{Key: "Tab/↓", Description: "Browse tags"},
				{Key: "←→", Description: "Navigate tags"},
				{Key: "↑", Description: "Back to search"},
				{Key: "Enter", Description: "View tag skills"},
				{Key: "Esc", Description: "Return to home"},
			},
		}
	}
	return ViewCommands{
		ViewName: "Search",
		Commands: []Command{
			{Key: "↑↓", Description: "Navigate results"},
			{Key: "Tab", Description: "Expand/collapse snippets"},
			{Key: "Enter", Description: "View skill details"},
			{Key: "Esc", Description: "Return to home"},
		},
	}
}
