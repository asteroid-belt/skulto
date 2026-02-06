package views

import (
	"fmt"
	"strings"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/db"
	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/telemetry"
	"github.com/asteroid-belt/skulto/internal/tui/components"
	"github.com/asteroid-belt/skulto/internal/tui/design"
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	"github.com/asteroid-belt/skulto/pkg/version"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HomeAction represents an action requested by the home view.
type HomeAction int

const (
	HomeActionNone HomeAction = iota
	HomeActionSearch
	HomeActionAddSource
	HomeActionSettings
	HomeActionSelectTag
	HomeActionSelectSkill
)

// Home view layout constants.
const (
	homeWelcomeLines  = 5  // Welcome section with margins
	homeTagsLines     = 4  // Tags section with title and margin
	homeFooterLines   = 1  // Footer
	homePaddingLines  = 2  // Extra padding
	homeLinesPerItem  = 2  // Each skill item takes ~2 lines
	homeTitleLines    = 2  // Title + margin
	homeMaxItems      = 10 // Maximum visible items
	homeMinItems      = 2  // Minimum visible items
	homeLoadedMaxCap  = 5  // Cap for loaded skills column
	homeApproxHeader  = 10 // Approximate header height for scroll calculations
)

// HomeView displays the dashboard with welcome message, recent skills, top tags, and installed skills.
type HomeView struct {
	db        *db.DB
	cfg       *config.Config
	telemetry telemetry.Client

	recentSkills []models.Skill
	topTags      []models.Tag
	loadedSkills []models.Skill

	width           int
	height          int
	selectedIdx     int // Index within the current column
	selectedColumn  int // 0 = loaded, 1 = recent, 2 = tags
	tagScrollOffset int // Horizontal scroll offset for tags
	loading         bool

	// Scroll offsets for skill columns
	loadedScrollOffset int
	recentScrollOffset int

	// Header and footer state
	skillCount int64
	tagCount   int64
	pulling    bool
	animTick   int

	// Pull progress state
	pullProgress int
	pullTotal    int
	pullRepoName string

	// Indexing state for semantic search
	indexing         bool
	indexingProgress string

	// Security scan progress state
	scanning     bool
	scanProgress int
	scanTotal    int
	scanRepoName string

	// Discovery badge count
	discoveryCount int64
}

// NewHomeView creates a new home view.
func NewHomeView(database *db.DB, conf *config.Config) *HomeView {
	return &HomeView{
		db:          database,
		cfg:         conf,
		selectedIdx: 0,
		loading:     true,
		skillCount:  0,
		tagCount:    0,
		pulling:     false,
		animTick:    0,
	}
}

// SetStats sets the skill and tag counts for the footer.
func (hv *HomeView) SetStats(skillCount, tagCount int64) {
	hv.skillCount = skillCount
	hv.tagCount = tagCount
}

// SetPulling updates the pulling state for the footer.
func (hv *HomeView) SetPulling(pulling bool) {
	hv.pulling = pulling
	if !pulling {
		// Reset progress when done
		hv.pullProgress = 0
		hv.pullTotal = 0
		hv.pullRepoName = ""
	}
}

// SetPullProgress updates the pull progress state.
func (hv *HomeView) SetPullProgress(completed, total int, repoName string) {
	hv.pullProgress = completed
	hv.pullTotal = total
	hv.pullRepoName = repoName
}

// IsPulling returns whether a pull is in progress.
func (hv *HomeView) IsPulling() bool {
	return hv.pulling
}

// SetIndexing updates the indexing state for the footer.
func (hv *HomeView) SetIndexing(indexing bool, progress string) {
	hv.indexing = indexing
	hv.indexingProgress = progress
}

// IsIndexing returns whether indexing is in progress.
func (hv *HomeView) IsIndexing() bool {
	return hv.indexing
}

// SetScanProgress updates the security scan progress state.
func (hv *HomeView) SetScanProgress(scanned, total int, repoName string) {
	hv.scanning = true
	hv.scanProgress = scanned
	hv.scanTotal = total
	hv.scanRepoName = repoName
}

// ClearScanProgress clears the security scan progress state.
func (hv *HomeView) ClearScanProgress() {
	hv.scanning = false
	hv.scanProgress = 0
	hv.scanTotal = 0
	hv.scanRepoName = ""
}

// SetDiscoveryCount sets the count of undismissed discovered skills.
func (hv *HomeView) SetDiscoveryCount(count int64) {
	hv.discoveryCount = count
}

// GetDiscoveryCount returns the count of undismissed discovered skills.
func (hv *HomeView) GetDiscoveryCount() int64 {
	return hv.discoveryCount
}

// UpdateAnimation advances the animation frame for the header.
func (hv *HomeView) UpdateAnimation() {
	hv.animTick++
}

// Init loads initial data and sets the telemetry client.
func (hv *HomeView) Init(tc telemetry.Client) {
	hv.telemetry = tc

	// Load recent skills
	skills, err := hv.db.GetRecentSkills(5)
	if err == nil {
		hv.recentSkills = skills
	}

	// Load top tags (excluding "mine" tag which is always zero)
	tags, err := hv.db.GetTopTags(11) // Fetch one extra in case "mine" is included
	if err == nil {
		hv.topTags = FilterOutMineTag(tags, 10)
	}

	// Load installed skills
	installed, err := hv.db.GetInstalled()
	if err == nil {
		hv.loadedSkills = installed
	}

	hv.loading = false
}

// Update handles user input and returns the action to perform.
func (hv *HomeView) Update(key string) HomeAction {
	switch key {
	case "up", "k":
		if hv.selectedIdx > 0 {
			hv.selectedIdx--
			hv.adjustScrollForSelection()
		}
		return HomeActionNone

	case "down", "j":
		// Get max index for current column
		var maxIdx int
		switch hv.selectedColumn {
		case 0:
			maxIdx = len(hv.loadedSkills) - 1
		case 1:
			maxIdx = len(hv.recentSkills) - 1
		case 2:
			// tags
			maxIdx = len(hv.topTags) - 1
		}

		if hv.selectedIdx < maxIdx {
			hv.selectedIdx++
			hv.adjustScrollForSelection()
		}
		return HomeActionNone

	case "left", "h":
		// Switch between columns
		switch hv.selectedColumn {
		case 1:
			// Recent -> Loaded
			hv.selectedColumn = 0
			hv.selectedIdx = min(hv.selectedIdx, len(hv.loadedSkills)-1)
			if hv.selectedIdx < 0 {
				hv.selectedIdx = 0
			}
			hv.adjustScrollForSelection()
		case 2:
			// Tags -> Recent
			hv.selectedColumn = 1
			hv.selectedIdx = min(hv.selectedIdx, len(hv.recentSkills)-1)
			if hv.selectedIdx < 0 {
				hv.selectedIdx = 0
			}
			hv.adjustScrollForSelection()
		}
		return HomeActionNone

	case "right", "l":
		// Switch between columns
		if hv.selectedColumn == 0 {
			// Loaded -> Recent
			hv.selectedColumn = 1
			hv.selectedIdx = min(hv.selectedIdx, len(hv.recentSkills)-1)
			if hv.selectedIdx < 0 {
				hv.selectedIdx = 0
			}
			hv.adjustScrollForSelection()
		} else if hv.selectedColumn == 1 && len(hv.topTags) > 0 {
			// Recent -> Tags (only if we have tags)
			hv.selectedColumn = 2
			hv.selectedIdx = min(hv.selectedIdx, len(hv.topTags)-1)
			if hv.selectedIdx < 0 {
				hv.selectedIdx = 0
			}
		}
		return HomeActionNone

	case "/":
		return HomeActionSearch

	case "a":
		return HomeActionAddSource

	case "s":
		return HomeActionSettings

	case "enter":
		// Check if tag is selected first
		if hv.selectedColumn == 2 && len(hv.topTags) > 0 {
			return HomeActionSelectTag
		}
		// Otherwise check if skill is selected
		if skill := hv.GetSelectedSkill(); skill != nil {
			return HomeActionSelectSkill
		}
		return HomeActionNone

	default:
		return HomeActionNone
	}
}

// calculateMaxVisibleItems returns the max visible items for a given header height.
func (hv *HomeView) calculateMaxVisibleItems(headerLineCount int) int {
	fixedHeight := headerLineCount + homeWelcomeLines + homeTagsLines + homeFooterLines + homePaddingLines
	availableForSkills := hv.height - fixedHeight
	maxVisibleItems := (availableForSkills - homeTitleLines) / homeLinesPerItem
	if maxVisibleItems < homeMinItems {
		maxVisibleItems = homeMinItems
	}
	if maxVisibleItems > homeMaxItems {
		maxVisibleItems = homeMaxItems
	}
	return maxVisibleItems
}

// adjustScrollForSelection adjusts scroll offset to keep the selected item visible.
func (hv *HomeView) adjustScrollForSelection() {
	maxVisibleItems := hv.calculateMaxVisibleItems(homeApproxHeader)

	switch hv.selectedColumn {
	case 0: // Loaded skills — always cap at 5 visible items
		loadedMax := maxVisibleItems
		if loadedMax > homeLoadedMaxCap {
			loadedMax = homeLoadedMaxCap
		}
		// Scroll down if selected item is below visible area
		if hv.selectedIdx >= hv.loadedScrollOffset+loadedMax {
			hv.loadedScrollOffset = hv.selectedIdx - loadedMax + 1
		}
		// Scroll up if selected item is above visible area
		if hv.selectedIdx < hv.loadedScrollOffset {
			hv.loadedScrollOffset = hv.selectedIdx
		}
		// Clamp scroll offset
		maxScroll := len(hv.loadedSkills) - loadedMax
		if maxScroll < 0 {
			maxScroll = 0
		}
		if hv.loadedScrollOffset > maxScroll {
			hv.loadedScrollOffset = maxScroll
		}
		if hv.loadedScrollOffset < 0 {
			hv.loadedScrollOffset = 0
		}

	case 1: // Recent skills
		// Scroll down if selected item is below visible area
		if hv.selectedIdx >= hv.recentScrollOffset+maxVisibleItems {
			hv.recentScrollOffset = hv.selectedIdx - maxVisibleItems + 1
		}
		// Scroll up if selected item is above visible area
		if hv.selectedIdx < hv.recentScrollOffset {
			hv.recentScrollOffset = hv.selectedIdx
		}
		// Clamp scroll offset
		maxScroll := len(hv.recentSkills) - maxVisibleItems
		if maxScroll < 0 {
			maxScroll = 0
		}
		if hv.recentScrollOffset > maxScroll {
			hv.recentScrollOffset = maxScroll
		}
		if hv.recentScrollOffset < 0 {
			hv.recentScrollOffset = 0
		}
	}
}

// GetSelectedSkill returns the currently selected skill.
func (hv *HomeView) GetSelectedSkill() *models.Skill {
	switch hv.selectedColumn {
	case 0:
		// Loaded skills column
		if hv.selectedIdx >= 0 && hv.selectedIdx < len(hv.loadedSkills) {
			return &hv.loadedSkills[hv.selectedIdx]
		}
	case 1:
		// Recent skills column
		if hv.selectedIdx >= 0 && hv.selectedIdx < len(hv.recentSkills) {
			return &hv.recentSkills[hv.selectedIdx]
		}
	}
	return nil
}

// GetSelectedTag returns the currently selected tag.
func (hv *HomeView) GetSelectedTag() *models.Tag {
	if hv.selectedColumn == 2 {
		if hv.selectedIdx >= 0 && hv.selectedIdx < len(hv.topTags) {
			return &hv.topTags[hv.selectedIdx]
		}
	}
	return nil
}

// View renders the home view with header, content, and footer.
func (hv *HomeView) View() string {
	if hv.loading {
		return hv.renderLoading()
	}

	// Calculate layout dimensions
	headerView := hv.renderHeader()
	headerLineCount := strings.Count(headerView, "\n") + 1

	// Calculate max visible items using shared helper
	maxVisibleItems := hv.calculateMaxVisibleItems(headerLineCount)

	// Build main content with dynamic item limits
	content := []string{
		"",
		hv.renderWelcome(),
		"",
		hv.renderSkillsColumnsWithLimit(maxVisibleItems),
		"",
		hv.renderTopTags(hv.selectedColumn == 2),
		"",
	}
	mainContent := strings.Join(content, "\n")

	// Calculate remaining space
	mainLines := strings.Split(mainContent, "\n")

	// Calculate space available for main content
	availableMainHeight := hv.height - homeFooterLines - headerLineCount - 1
	if availableMainHeight < 0 {
		availableMainHeight = 0
	}

	// Pad main content to fill available space
	for len(mainLines) < availableMainHeight {
		mainLines = append(mainLines, "")
	}

	// Calculate footer height to extend to bottom
	footerHeight := hv.height - len(mainLines) - headerLineCount - 1
	if footerHeight < 1 {
		footerHeight = 1
	}

	// Render footer with extended height
	footerView := hv.renderFooterWithHeight(footerHeight)

	// Build final layout
	paddedContent := strings.Join(mainLines, "\n")
	layout := lipgloss.JoinVertical(
		lipgloss.Top,
		paddedContent,
		footerView,
	)

	// Prepend header
	layout = headerView + "\n" + layout

	return layout
}

// SetSize updates the view dimensions.
func (hv *HomeView) SetSize(w, h int) {
	hv.width = w
	hv.height = h
}

// renderSkillsColumnsWithLimit renders skill columns with a maximum number of visible items.
func (hv *HomeView) renderSkillsColumnsWithLimit(maxItems int) string {
	loadedActive := hv.selectedColumn == 0
	recentActive := hv.selectedColumn == 1

	loadedSection := hv.renderLoadedSkillsWithLimit(loadedActive, maxItems)
	recentSection := hv.renderRecentSkillsWithLimit(recentActive, maxItems)

	// Calculate column width (split available width, accounting for padding)
	colWidth := (hv.width - 4) / 2
	if colWidth < 30 {
		colWidth = 30
	}

	// Apply width constraint to both sections
	loadedStyle := lipgloss.NewStyle().Width(colWidth)
	recentStyle := lipgloss.NewStyle().Width(colWidth)

	loadedStyledSection := loadedStyle.Render(loadedSection)
	recentStyledSection := recentStyle.Render(recentSection)

	// Join horizontally with spacing (loaded first, then recent)
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		loadedStyledSection,
		"  ", // Add spacing between columns
		recentStyledSection,
	)
}

// renderLoading shows a loading message.
func (hv *HomeView) renderLoading() string {
	loadingStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Accent).
		Bold(true).
		Align(lipgloss.Center).
		Width(hv.width)

	skull := design.SkultoLogo
	return loadingStyle.Render(skull + "\n\nLoading skills...")
}

// renderWelcome renders the welcome section.
func (hv *HomeView) renderWelcome() string {
	welcomeStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Primary).
		Bold(true).
		MarginLeft(1).
		MarginRight(2).
		MarginTop(1).
		MarginBottom(1)

	msgStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Text).
		MarginLeft(1).
		MarginRight(2)

	// Secondary line with slightly muted text
	secondaryStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		MarginLeft(1).
		MarginRight(2).
		MarginBottom(1)

	// Build manage text with optional discovery badge
	manageText := "m (manage)"
	if hv.discoveryCount > 0 {
		badgeStyle := lipgloss.NewStyle().
			Foreground(theme.Current.Warning).
			Bold(true)
		manageText = fmt.Sprintf("m (manage %s)", badgeStyle.Render(fmt.Sprintf("(%d)", hv.discoveryCount)))
	}

	return welcomeStyle.Render("Welcome to SKULTO") + "\n" +
		msgStyle.Render(fmt.Sprintf("/ (search) • ↑↓ (nav) • %s • p (pull) • q (quit)", manageText)) + "\n" +
		secondaryStyle.Render("a (add repo) • s (settings) • n (new skill) • ? (help)")
}

// renderRecentSkillsWithLimit renders recent skills with scrolling support.
func (hv *HomeView) renderRecentSkillsWithLimit(columnActive bool, maxItems int) string {
	// Make the title brighter if this column is active
	titleColor := theme.Current.TextMuted
	if columnActive {
		titleColor = theme.Current.Accent
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(titleColor).
		Bold(true).
		MarginLeft(1).
		MarginTop(1)

	total := len(hv.recentSkills)
	titleText := "Recently Viewed Skills"
	if total > maxItems {
		titleText = fmt.Sprintf("Recently Viewed (%d/%d)", min(hv.recentScrollOffset+maxItems, total), total)
	}
	title := titleStyle.Render(titleText)

	if total == 0 {
		return title + "\n" + hv.renderEmptyState("No recently viewed skills")
	}

	// Calculate visible range
	startIdx := hv.recentScrollOffset
	endIdx := startIdx + maxItems
	if endIdx > total {
		endIdx = total
	}
	if startIdx > endIdx {
		startIdx = endIdx
	}

	var items []string
	for i := startIdx; i < endIdx; i++ {
		skill := hv.recentSkills[i]
		if columnActive && i == hv.selectedIdx {
			items = append(items, components.RenderSelectedSkill(skill, components.SimpleStyle))
		} else {
			items = append(items, components.RenderSkillItem(skill, components.SimpleStyle))
		}
	}

	// Add scroll indicators
	result := title + "\n" + strings.Join(items, "\n")
	if endIdx < total {
		moreStyle := lipgloss.NewStyle().
			Foreground(theme.Current.TextMuted).
			Italic(true).
			MarginLeft(2)
		result += "\n" + moreStyle.Render(fmt.Sprintf("↓ %d more...", total-endIdx))
	}

	return result
}

// renderLoadedSkillsWithLimit renders installed skills with scrolling support.
// Always shows at most homeLoadedMaxCap items to keep the layout consistent with recently viewed.
func (hv *HomeView) renderLoadedSkillsWithLimit(columnActive bool, maxItems int) string {
	if maxItems > homeLoadedMaxCap {
		maxItems = homeLoadedMaxCap
	}
	// Make the title brighter if this column is active
	titleColor := theme.Current.TextMuted
	if columnActive {
		titleColor = theme.Current.Accent
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(titleColor).
		Bold(true).
		MarginLeft(1).
		MarginTop(1)

	total := len(hv.loadedSkills)
	titleText := "Installed Skills"
	if total > maxItems {
		titleText = fmt.Sprintf("Installed Skills (%d/%d)", min(hv.loadedScrollOffset+maxItems, total), total)
	}
	title := titleStyle.Render(titleText)

	if total == 0 {
		return title + "\n" + hv.renderEmptyState("No skills installed")
	}

	// Calculate visible range
	startIdx := hv.loadedScrollOffset
	endIdx := startIdx + maxItems
	if endIdx > total {
		endIdx = total
	}
	if startIdx > endIdx {
		startIdx = endIdx
	}

	var items []string
	for i := startIdx; i < endIdx; i++ {
		skill := hv.loadedSkills[i]
		if columnActive && i == hv.selectedIdx {
			items = append(items, components.RenderSelectedSkill(skill, components.SimpleStyle))
		} else {
			items = append(items, components.RenderSkillItem(skill, components.SimpleStyle))
		}
	}

	// Add scroll indicators
	result := title + "\n" + strings.Join(items, "\n")
	if endIdx < total {
		moreStyle := lipgloss.NewStyle().
			Foreground(theme.Current.TextMuted).
			Italic(true).
			MarginLeft(2)
		result += "\n" + moreStyle.Render(fmt.Sprintf("↓ %d more...", total-endIdx))
	}

	return result
}

// renderTopTags renders the top tags section.
func (hv *HomeView) renderTopTags(isActive bool) string {
	if len(hv.topTags) == 0 {
		return ""
	}

	// Make the title brighter if this column is active
	titleColor := theme.Current.TextMuted
	if isActive {
		titleColor = theme.Current.Accent
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(titleColor).
		Bold(true).
		Padding(0, 0, 1, 0).
		MarginLeft(2).
		MarginTop(1)

	title := titleStyle.Render("Top Tags")

	// Render all tags and calculate their widths
	var renderedTags []string
	var tagWidths []int
	for i, tag := range hv.topTags {
		var rendered string
		if isActive && i == hv.selectedIdx {
			// Render as selected - inverted colors
			tagStyle := lipgloss.NewStyle().
				Background(theme.Current.TextHighlight).
				Foreground(theme.Current.Background).
				Padding(0, 1).
				Margin(0, 1, 0, 0).
				Bold(true)
			rendered = tagStyle.Render(fmt.Sprintf("%s (%d)", tag.Name, tag.Count))
		} else if tag.ID == "mine" {
			// Special styling for "mine" tag - bold with contrasting foreground
			tagStyle := lipgloss.NewStyle().
				Background(theme.GetTagColor(tag.Category)).
				Foreground(theme.Current.Background).
				Bold(true).
				Padding(0, 1).
				Margin(0, 1, 0, 0)
			rendered = tagStyle.Render(fmt.Sprintf("%s (%d)", tag.Name, tag.Count))
		} else {
			// Normal rendering with contrasting foreground for bright backgrounds
			tagStyle := lipgloss.NewStyle().
				Background(theme.GetTagColor(tag.Category)).
				Foreground(theme.Current.Background).
				Padding(0, 1).
				Margin(0, 1, 0, 0)
			rendered = tagStyle.Render(fmt.Sprintf("%s (%d)", tag.Name, tag.Count))
		}
		renderedTags = append(renderedTags, rendered)
		tagWidths = append(tagWidths, lipgloss.Width(rendered))
	}

	// Calculate available width for tags (account for margins and scroll indicators)
	availableWidth := hv.width - 8 // Left margin + potential scroll indicators
	if availableWidth < 20 {
		availableWidth = 20
	}

	// Calculate cumulative positions for each tag
	positions := make([]int, len(tagWidths)+1)
	for i, w := range tagWidths {
		positions[i+1] = positions[i] + w
	}

	// Adjust scroll offset to keep selected tag visible (only when tags section is active)
	if isActive && len(hv.topTags) > 0 {
		selectedStart := positions[hv.selectedIdx]
		selectedEnd := positions[hv.selectedIdx+1]

		// If selected tag is before visible area, scroll left
		if selectedStart < hv.tagScrollOffset {
			hv.tagScrollOffset = selectedStart
		}
		// If selected tag is after visible area, scroll right
		if selectedEnd > hv.tagScrollOffset+availableWidth {
			hv.tagScrollOffset = selectedEnd - availableWidth
		}
	}

	// Ensure scroll offset is valid
	if hv.tagScrollOffset < 0 {
		hv.tagScrollOffset = 0
	}
	totalWidth := positions[len(positions)-1]
	if hv.tagScrollOffset > totalWidth-availableWidth {
		hv.tagScrollOffset = totalWidth - availableWidth
	}
	if hv.tagScrollOffset < 0 {
		hv.tagScrollOffset = 0
	}

	// Build visible tags string
	var visibleTags []string
	for i, rendered := range renderedTags {
		tagStart := positions[i]
		tagEnd := positions[i+1]

		// Skip tags completely before visible area
		if tagEnd <= hv.tagScrollOffset {
			continue
		}
		// Stop if tag starts after visible area
		if tagStart >= hv.tagScrollOffset+availableWidth {
			break
		}
		visibleTags = append(visibleTags, rendered)
	}

	// Add scroll indicators
	var result string
	leftIndicator := ""
	rightIndicator := ""

	if hv.tagScrollOffset > 0 {
		leftIndicator = "◀ "
	}
	if hv.tagScrollOffset+availableWidth < totalWidth {
		rightIndicator = " ▶"
	}

	result = leftIndicator + strings.Join(visibleTags, "") + rightIndicator

	return title + "\n" + lipgloss.NewStyle().MarginLeft(2).Render(result)
}

// renderEmptyState renders an empty state message.
func (hv *HomeView) renderEmptyState(msg string) string {
	emptyStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		Italic(true).
		MarginLeft(2).
		MarginTop(2)

	return emptyStyle.Render(msg)
}

// renderHeader renders the animated skull header with app title and version.
func (hv *HomeView) renderHeader() string {
	// Choose logo based on available width (logo is 48 chars, need some padding)
	var skull string
	if hv.width >= 56 {
		skull = design.SkultoLogo
	} else {
		skull = design.SkultoLogoMinimal
	}

	// Format skull with punk rock colors
	skullStyle := lipgloss.NewStyle().
		Foreground(theme.Current.Primary).
		Bold(true)

	// Format version
	versionStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		Italic(true).
		PaddingTop(1)

	ver := versionStyle.Render(version.Version)

	// Combine skull and title
	header := lipgloss.JoinVertical(lipgloss.Center, skullStyle.Render(skull), ver)

	// Center the header
	headerStyle := lipgloss.NewStyle().
		Width(hv.width).
		Align(lipgloss.Center).
		Padding(1, 0)

	return headerStyle.Render(header)
}

// renderFooterWithHeight renders the footer and extends it to fill the given height.
func (hv *HomeView) renderFooterWithHeight(height int) string {
	// Left: stats
	leftStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted)

	left := leftStyle.Render(fmt.Sprintf("[ %d skills • %d tags ]",
		hv.skillCount, hv.tagCount))

	// Center: pull status, scan status, or indexing indicator
	center := ""

	// Override with pull progress bar if pulling
	if hv.pulling {
		center = hv.renderPullProgress()
	}

	// Override with scan progress if scanning (shows alongside or instead of pull)
	if hv.scanning {
		center = hv.renderScanProgress()
	}

	// Override with indexing indicator if indexing
	if hv.indexing {
		indexStyle := lipgloss.NewStyle().
			Foreground(theme.Current.Accent).
			Bold(true)
		center = indexStyle.Render(hv.indexingProgress)
	}

	// Build footer with left-aligned stats and centered status
	footerWidth := hv.width
	leftWidth := lipgloss.Width(left)
	centerWidth := lipgloss.Width(center)

	// Calculate spacing to center the status text
	// Total available space after left = footerWidth - leftWidth - 2 (padding)
	availableForCenter := footerWidth - leftWidth - 2
	leftPadding := (availableForCenter - centerWidth) / 2
	if leftPadding < 1 {
		leftPadding = 1
	}

	footerStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		Padding(0, 1).
		Width(footerWidth).
		Height(height).
		AlignVertical(lipgloss.Bottom)

	footer := lipgloss.JoinHorizontal(lipgloss.Center,
		left,
		lipgloss.NewStyle().Width(leftPadding).Render(""),
		center,
	)

	return footerStyle.Render(footer)
}

// progressBarWidth is the width of progress bars in characters.
const progressBarWidth = 15

// renderProgressBar renders a progress bar with the given configuration.
func (hv *HomeView) renderProgressBar(current, total int, icon string, iconColor, barColor lipgloss.AdaptiveColor, suffix string) string {
	percent := float64(current) / float64(total)
	filled := int(float64(progressBarWidth) * percent)
	empty := progressBarWidth - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	iconStyle := lipgloss.NewStyle().Foreground(iconColor).Bold(true)
	barStyle := lipgloss.NewStyle().Foreground(barColor)
	countStyle := lipgloss.NewStyle().Foreground(theme.Current.TextMuted)

	return iconStyle.Render(icon+" ") +
		barStyle.Render("["+bar+"]") +
		countStyle.Render(fmt.Sprintf(" %d/%d ", current, total)) +
		iconStyle.Render(suffix)
}

// renderPullProgress renders the pull progress bar.
func (hv *HomeView) renderPullProgress() string {
	if hv.pullTotal == 0 {
		pullStyle := lipgloss.NewStyle().
			Foreground(theme.Current.Success).
			Bold(true)
		return pullStyle.Render("⚡ Starting pull...")
	}

	return hv.renderProgressBar(
		hv.pullProgress, hv.pullTotal,
		"⚡", theme.Current.Success, theme.Current.Accent,
		hv.pullRepoName,
	)
}

// renderScanProgress renders the security scan progress bar.
func (hv *HomeView) renderScanProgress() string {
	if hv.scanTotal == 0 {
		scanStyle := lipgloss.NewStyle().
			Foreground(theme.Current.Error).
			Bold(true)
		return scanStyle.Render("🔒 Starting security scan...")
	}

	// Truncate repo name if too long
	repoName := hv.scanRepoName
	if len(repoName) > 25 {
		repoName = repoName[:22] + "..."
	}

	return hv.renderProgressBar(
		hv.scanProgress, hv.scanTotal,
		"🔒", theme.Current.Error, theme.Current.Warning,
		"Scanning "+repoName,
	)
}

// HandleMouse handles mouse events for scrolling.
func (hv *HomeView) HandleMouse(msg tea.MouseMsg) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if hv.selectedIdx > 0 {
			hv.selectedIdx--
		}
	case tea.MouseButtonWheelDown:
		var maxIdx int
		switch hv.selectedColumn {
		case 0:
			maxIdx = len(hv.loadedSkills) - 1
		case 1:
			maxIdx = len(hv.recentSkills) - 1
		case 2:
			maxIdx = len(hv.topTags) - 1
		}
		if hv.selectedIdx < maxIdx {
			hv.selectedIdx++
		}
	}
}

// GetKeyboardCommands returns the keyboard commands for this view.
func (hv *HomeView) GetKeyboardCommands() ViewCommands {
	return ViewCommands{
		ViewName: "Home",
		Commands: []Command{
			{Key: "↑↓, k/j", Description: "Navigate items within column"},
			{Key: "←→, h/l", Description: "Switch columns (loaded, recent, tags)"},
			{Key: "Enter", Description: "Select skill or tag"},
			{Key: "/", Description: "Search skills"},
			{Key: "m", Description: "Manage installed skills"},
			{Key: "a", Description: "Add new source repository"},
			{Key: "p", Description: "Pull latest from seed repositories"},
			{Key: "s", Description: "Open settings"},
			{Key: "n", Description: "New skill - create a skill from a prompt"},
		},
	}
}

