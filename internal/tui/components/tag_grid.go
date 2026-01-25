package components

import (
	"fmt"

	"github.com/asteroid-belt/skulto/internal/models"
	"github.com/charmbracelet/lipgloss"
)

// TagGrid displays tags in a horizontal flow grid with keyboard navigation.
type TagGrid struct {
	tags         []models.Tag
	selectedIdx  int
	width        int
	height       int
	scrollOffset int // Vertical scroll (in rows)
	focused      bool

	// Calculated layout
	tagWidths []int // Width of each rendered tag
	rowStarts []int // Index of first tag in each row
}

// NewTagGrid creates a new tag grid component.
func NewTagGrid() *TagGrid {
	return &TagGrid{
		selectedIdx: 0,
	}
}

// SetTags updates the tags to display.
func (tg *TagGrid) SetTags(tags []models.Tag) {
	tg.tags = tags
	tg.selectedIdx = 0
	tg.scrollOffset = 0
	tg.calculateLayout()
}

// SetSize sets the available width and height for rendering.
func (tg *TagGrid) SetSize(width, height int) {
	tg.width = width
	tg.height = height
	tg.calculateLayout()
}

// SetFocused sets the focus state.
func (tg *TagGrid) SetFocused(focused bool) {
	tg.focused = focused
}

// IsFocused returns whether the grid is focused.
func (tg *TagGrid) IsFocused() bool {
	return tg.focused
}

// GetSelectedTag returns the currently selected tag.
func (tg *TagGrid) GetSelectedTag() *models.Tag {
	if tg.selectedIdx >= 0 && tg.selectedIdx < len(tg.tags) {
		return &tg.tags[tg.selectedIdx]
	}
	return nil
}

// GetSelectedIndex returns the current selection index.
func (tg *TagGrid) GetSelectedIndex() int {
	return tg.selectedIdx
}

// TagCount returns the number of tags.
func (tg *TagGrid) TagCount() int {
	return len(tg.tags)
}

// calculateLayout computes tag widths and row positions.
func (tg *TagGrid) calculateLayout() {
	if len(tg.tags) == 0 || tg.width == 0 {
		tg.tagWidths = nil
		tg.rowStarts = nil
		return
	}

	tg.tagWidths = make([]int, len(tg.tags))
	tg.rowStarts = []int{0}

	availableWidth := tg.width - 4 // Account for margins
	currentRowWidth := 0
	tagMargin := 2 // Space between tags

	for i, tag := range tg.tags {
		// Calculate rendered width: "name (count)" + padding
		tagText := fmt.Sprintf("%s (%d)", tag.Name, tag.Count)
		tagWidth := len(tagText) + 2 + tagMargin // +2 for padding, +margin

		tg.tagWidths[i] = tagWidth

		// Check if tag fits on current row
		if currentRowWidth+tagWidth > availableWidth && currentRowWidth > 0 {
			// Start new row
			tg.rowStarts = append(tg.rowStarts, i)
			currentRowWidth = tagWidth
		} else {
			currentRowWidth += tagWidth
		}
	}
}

// getSelectedRow returns the row index of the currently selected tag.
func (tg *TagGrid) getSelectedRow() int {
	for row := len(tg.rowStarts) - 1; row >= 0; row-- {
		if tg.selectedIdx >= tg.rowStarts[row] {
			return row
		}
	}
	return 0
}

// getRowBounds returns the start and end indices for a row.
func (tg *TagGrid) getRowBounds(row int) (start, end int) {
	if row < 0 || row >= len(tg.rowStarts) {
		return 0, 0
	}
	start = tg.rowStarts[row]
	if row+1 < len(tg.rowStarts) {
		end = tg.rowStarts[row+1]
	} else {
		end = len(tg.tags)
	}
	return start, end
}

// MoveLeft moves selection left.
func (tg *TagGrid) MoveLeft() {
	if tg.selectedIdx > 0 {
		tg.selectedIdx--
		tg.adjustScroll()
	}
}

// MoveRight moves selection right.
func (tg *TagGrid) MoveRight() {
	if tg.selectedIdx < len(tg.tags)-1 {
		tg.selectedIdx++
		tg.adjustScroll()
	}
}

// MoveUp moves selection up one row. Returns true if already at top row.
func (tg *TagGrid) MoveUp() bool {
	currentRow := tg.getSelectedRow()
	if currentRow == 0 {
		return true // Signal to move focus to search bar
	}

	// Find position within current row
	rowStart, _ := tg.getRowBounds(currentRow)
	posInRow := tg.selectedIdx - rowStart

	// Move to previous row, same position or last tag in row
	prevRowStart, prevRowEnd := tg.getRowBounds(currentRow - 1)
	prevRowSize := prevRowEnd - prevRowStart

	if posInRow >= prevRowSize {
		tg.selectedIdx = prevRowEnd - 1 // Last tag in previous row
	} else {
		tg.selectedIdx = prevRowStart + posInRow
	}

	tg.adjustScroll()
	return false
}

// MoveDown moves selection down one row.
func (tg *TagGrid) MoveDown() {
	currentRow := tg.getSelectedRow()
	totalRows := len(tg.rowStarts)

	if currentRow >= totalRows-1 {
		return // Already at bottom row
	}

	// Find position within current row
	rowStart, _ := tg.getRowBounds(currentRow)
	posInRow := tg.selectedIdx - rowStart

	// Move to next row, same position or last tag in row
	nextRowStart, nextRowEnd := tg.getRowBounds(currentRow + 1)
	nextRowSize := nextRowEnd - nextRowStart

	if posInRow >= nextRowSize {
		tg.selectedIdx = nextRowEnd - 1 // Last tag in next row
	} else {
		tg.selectedIdx = nextRowStart + posInRow
	}

	tg.adjustScroll()
}

// adjustScroll ensures the selected tag is visible.
func (tg *TagGrid) adjustScroll() {
	if tg.height == 0 {
		return
	}

	rowHeight := 1 // Each row takes 1 line
	visibleRows := max(1, tg.height/rowHeight)
	currentRow := tg.getSelectedRow()

	// Scroll down if selection is below viewport
	if currentRow >= tg.scrollOffset+visibleRows {
		tg.scrollOffset = currentRow - visibleRows + 1
	}

	// Scroll up if selection is above viewport
	if currentRow < tg.scrollOffset {
		tg.scrollOffset = currentRow
	}
}

// View renders the tag grid.
func (tg *TagGrid) View() string {
	if len(tg.tags) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B6B6B")).
			Italic(true).
			MarginLeft(2)
		return emptyStyle.Render("No tags available")
	}

	// Render rows
	var rows []string
	totalRows := len(tg.rowStarts)

	// Calculate visible rows (reserve 2 lines for scroll indicators)
	rowHeight := 1
	availableHeight := max(1, tg.height-2)
	visibleRows := max(1, availableHeight/rowHeight)
	startRow := tg.scrollOffset
	endRow := min(startRow+visibleRows, totalRows)

	for row := startRow; row < endRow; row++ {
		rowStart, rowEnd := tg.getRowBounds(row)
		var rowTags []string

		for i := rowStart; i < rowEnd; i++ {
			tag := tg.tags[i]
			tagText := fmt.Sprintf("%s (%d)", tag.Name, tag.Count)

			var tagStyle lipgloss.Style
			if tg.focused && i == tg.selectedIdx {
				// Selected tag
				tagStyle = lipgloss.NewStyle().
					Background(lipgloss.Color("#FFFFFF")).
					Foreground(lipgloss.Color("#000000")).
					Bold(true).
					Padding(0, 1).
					Margin(0, 1, 0, 0)
			} else {
				// Normal tag with category color
				tagStyle = lipgloss.NewStyle().
					Background(getTagColorForGrid(tag.Category)).
					Foreground(lipgloss.Color("#000000")).
					Padding(0, 1).
					Margin(0, 1, 0, 0)
			}

			rowTags = append(rowTags, tagStyle.Render(tagText))
		}

		rowContent := lipgloss.JoinHorizontal(lipgloss.Top, rowTags...)
		rows = append(rows, lipgloss.NewStyle().MarginLeft(2).Render(rowContent))
	}

	// Add scroll indicators
	var result string
	if tg.scrollOffset > 0 {
		result = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B6B6B")).MarginLeft(2).Render("↑ more tags above") + "\n"
	}

	result += lipgloss.JoinVertical(lipgloss.Left, rows...)

	if endRow < totalRows {
		result += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#6B6B6B")).MarginLeft(2).Render("↓ more tags below")
	}

	return result
}

// getTagColorForGrid returns the color for a tag category.
func getTagColorForGrid(category string) lipgloss.Color {
	switch category {
	case "language":
		return lipgloss.Color("#A855F7") // Purple
	case "framework":
		return lipgloss.Color("#EC4899") // Pink
	case "tool":
		return lipgloss.Color("#10B981") // Emerald
	case "concept":
		return lipgloss.Color("#F59E0B") // Amber
	case "domain":
		return lipgloss.Color("#3B82F6") // Blue
	case "mine":
		return lipgloss.Color("#DC143C") // Crimson
	default:
		return lipgloss.Color("#6B7280") // Gray
	}
}
