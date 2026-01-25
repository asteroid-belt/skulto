package components

import (
	"fmt"
	"strings"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/asteroid-belt/skulto/internal/search"
	"github.com/charmbracelet/lipgloss"
)

// MatchType indicates how a result matched the query.
type MatchType int

const (
	MatchTypeName    MatchType = iota // Matched title or tag
	MatchTypeContent                  // Matched content/description
)

// UnifiedResultItem represents a single search result with its metadata.
type UnifiedResultItem struct {
	Skill     models.Skill
	MatchType MatchType
	Snippets  []search.Snippet
	Expanded  bool // Whether snippets are shown (for content matches)
}

// UnifiedResultList renders a unified list of search results.
type UnifiedResultList struct {
	Items        []UnifiedResultItem
	Selected     int
	scrollOffset int
	viewportSize int // Number of items visible at once
	width        int
}

// NewUnifiedResultList creates a new unified result list.
func NewUnifiedResultList() *UnifiedResultList {
	return &UnifiedResultList{
		Items:        []UnifiedResultItem{},
		Selected:     -1,
		scrollOffset: 0,
		viewportSize: 10, // Default, will be adjusted by SetSize
	}
}

// SetItems updates the list with new results.
func (ul *UnifiedResultList) SetItems(items []UnifiedResultItem) {
	ul.Items = items
	ul.scrollOffset = 0
	if len(items) > 0 && ul.Selected < 0 {
		ul.Selected = 0
	}
	if ul.Selected >= len(items) {
		ul.Selected = len(items) - 1
	}
}

// Clear removes all items.
func (ul *UnifiedResultList) Clear() {
	ul.Items = []UnifiedResultItem{}
	ul.Selected = -1
	ul.scrollOffset = 0
}

// SetSize sets the available width and viewport height (in lines).
func (ul *UnifiedResultList) SetSize(width, viewportLines int) {
	ul.width = width
	// Each collapsed item ~3 lines, expanded ~6+ lines
	// Use conservative estimate for viewport sizing
	ul.viewportSize = max(1, viewportLines/3)
}

// MoveUp moves selection up.
func (ul *UnifiedResultList) MoveUp() bool {
	if ul.Selected > 0 {
		ul.Selected--
		ul.adjustScroll()
		return true
	}
	return false
}

// MoveDown moves selection down.
func (ul *UnifiedResultList) MoveDown() bool {
	if ul.Selected < len(ul.Items)-1 {
		ul.Selected++
		ul.adjustScroll()
		return true
	}
	return false
}

// ToggleExpand toggles the expanded state of the selected item.
func (ul *UnifiedResultList) ToggleExpand() bool {
	if ul.Selected >= 0 && ul.Selected < len(ul.Items) {
		item := &ul.Items[ul.Selected]
		// Only content matches have snippets to expand
		if item.MatchType == MatchTypeContent && len(item.Snippets) > 0 {
			item.Expanded = !item.Expanded
			return true
		}
	}
	return false
}

// GetSelectedItem returns the currently selected item.
func (ul *UnifiedResultList) GetSelectedItem() *UnifiedResultItem {
	if ul.Selected >= 0 && ul.Selected < len(ul.Items) {
		return &ul.Items[ul.Selected]
	}
	return nil
}

// adjustScroll ensures the selected item is visible.
func (ul *UnifiedResultList) adjustScroll() {
	if ul.Selected < ul.scrollOffset {
		ul.scrollOffset = ul.Selected
	}
	if ul.Selected >= ul.scrollOffset+ul.viewportSize {
		ul.scrollOffset = ul.Selected - ul.viewportSize + 1
	}
	// Clamp
	maxOffset := len(ul.Items) - ul.viewportSize
	if maxOffset < 0 {
		maxOffset = 0
	}
	if ul.scrollOffset > maxOffset {
		ul.scrollOffset = maxOffset
	}
	if ul.scrollOffset < 0 {
		ul.scrollOffset = 0
	}
}

// View renders the list.
func (ul *UnifiedResultList) View() string {
	if len(ul.Items) == 0 {
		return ""
	}

	var parts []string

	// Styles
	nameTagBadge := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#2ECC71")).
		Bold(true)
	contentBadge := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3498DB")).
		Bold(true)
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)
	selectedTitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F1C40F")).
		Bold(true)
	sourceStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B6B6B"))
	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA"))
	snippetIndicator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B6B6B")).
		Italic(true)
	itemStyle := lipgloss.NewStyle().
		MarginLeft(1).
		Width(ul.width - 2)

	// Scroll indicator at top
	if ul.scrollOffset > 0 {
		scrollStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B6B6B")).
			MarginLeft(2)
		parts = append(parts, scrollStyle.Render(fmt.Sprintf("↑ %d more above", ul.scrollOffset)))
	}

	// Render visible items
	endIdx := ul.scrollOffset + ul.viewportSize
	if endIdx > len(ul.Items) {
		endIdx = len(ul.Items)
	}

	for i := ul.scrollOffset; i < endIdx; i++ {
		item := ul.Items[i]
		isSelected := i == ul.Selected

		var lines []string

		// Line 1: Badge + Title + Source
		var badge string
		if item.MatchType == MatchTypeName {
			badge = nameTagBadge.Render("[name]")
		} else {
			badge = contentBadge.Render("[content]")
		}

		titleText := item.Skill.Title
		var title string
		if isSelected {
			title = selectedTitleStyle.Render("› " + titleText)
		} else {
			title = titleStyle.Render("  " + titleText)
		}

		// Add source info if available
		var sourceInfo string
		if item.Skill.Source != nil && item.Skill.Source.Owner != "" && item.Skill.Source.Repo != "" {
			sourceInfo = sourceStyle.Render(fmt.Sprintf(" (%s/%s)", item.Skill.Source.Owner, item.Skill.Source.Repo))
		}
		lines = append(lines, badge+" "+title+sourceInfo)

		// Line 2: Description (truncated)
		desc := truncate(item.Skill.Description, ul.width-8)
		lines = append(lines, "    "+descStyle.Render(desc))

		// Line 3+: Snippets (if content match)
		if item.MatchType == MatchTypeContent && len(item.Snippets) > 0 {
			if item.Expanded {
				// Show expanded snippets
				lines = append(lines, "    "+snippetIndicator.Render("▼ Matching context:"))
				for _, snippet := range item.Snippets {
					rendered := RenderSnippet(snippet, ul.width-10)
					lines = append(lines, "      "+rendered)
				}
			} else {
				// Show collapse indicator
				lines = append(lines, "    "+snippetIndicator.Render(
					fmt.Sprintf("▶ %d matching snippet%s (Tab to expand)",
						len(item.Snippets),
						map[bool]string{true: "", false: "s"}[len(item.Snippets) == 1])))
			}
		}

		parts = append(parts, itemStyle.Render(strings.Join(lines, "\n")))
	}

	// Scroll indicator at bottom
	remaining := len(ul.Items) - endIdx
	if remaining > 0 {
		scrollStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B6B6B")).
			MarginLeft(2)
		parts = append(parts, scrollStyle.Render(fmt.Sprintf("↓ %d more below", remaining)))
	}

	return strings.Join(parts, "\n")
}

// TotalCount returns total number of items.
func (ul *UnifiedResultList) TotalCount() int {
	return len(ul.Items)
}
