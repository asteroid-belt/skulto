package views

import (
	"fmt"
	"sort"

	"github.com/asteroid-belt/skulto/internal/config"
	"github.com/asteroid-belt/skulto/internal/detect"
	"github.com/asteroid-belt/skulto/internal/installer"
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// itemKind identifies the type of a display item.
type itemKind int

const (
	itemHeader       itemKind = iota // Group header ("Detected on your system")
	itemAgent                        // Selectable agent entry
	itemSeparator                    // Visual separator between groups
	itemToggleHeader                 // Collapsible group header (interactive)
)

// displayItem represents a single row in the grouped list.
type displayItem struct {
	kind     itemKind
	platform installer.Platform // Only set for itemAgent
	label    string             // Display text for headers
}

// OnboardingToolsView displays AI tool detection and selection
// with grouped display: detected agents at top, all others below in a collapsible group.
type OnboardingToolsView struct {
	cfg              *config.Config
	detectionResults []detect.DetectionResult
	selectedTools    map[installer.Platform]bool
	currentSelection int

	// Grouped display
	detectedAgents []installer.Platform // Detected, shown first
	allAgents      []installer.Platform // All others, alphabetical
	displayItems   []displayItem        // Flattened for rendering
	group2Expanded bool                 // Whether "All Agents" group is expanded

	// Scrolling
	scrollOffset int

	width  int
	height int
}

// NewOnboardingToolsView creates a new onboarding tools view.
func NewOnboardingToolsView(conf *config.Config) *OnboardingToolsView {
	return &OnboardingToolsView{
		cfg:           conf,
		selectedTools: make(map[installer.Platform]bool),
	}
}

// Init initializes the view and detects available tools.
func (v *OnboardingToolsView) Init() {
	v.detectionResults = detect.DetectAll()
	v.selectedTools = make(map[installer.Platform]bool)
	v.detectedAgents = nil
	v.allAgents = nil
	v.scrollOffset = 0
	v.group2Expanded = false

	// Partition platforms into detected vs all-others
	detectedMap := make(map[installer.Platform]bool)
	for _, result := range v.detectionResults {
		if result.Detected {
			detectedMap[result.Platform] = true
			v.detectedAgents = append(v.detectedAgents, result.Platform)
		}
	}

	// All others, sorted alphabetically by display name
	for _, p := range installer.AllPlatforms() {
		if !detectedMap[p] {
			v.allAgents = append(v.allAgents, p)
		}
	}
	sort.Slice(v.allAgents, func(i, j int) bool {
		return v.allAgents[i].Info().Name < v.allAgents[j].Info().Name
	})

	// Auto-expand group 2 if nothing was detected
	if len(v.detectedAgents) == 0 {
		v.group2Expanded = true
	}

	// Build display items
	v.buildDisplayItems()

	// Position cursor on first selectable item
	v.currentSelection = v.firstSelectableIndex()
}

// buildDisplayItems creates the flattened list for rendering.
func (v *OnboardingToolsView) buildDisplayItems() {
	v.displayItems = nil

	if len(v.detectedAgents) > 0 {
		v.displayItems = append(v.displayItems, displayItem{
			kind: itemHeader, label: "Detected on your system",
		})
		for _, p := range v.detectedAgents {
			v.displayItems = append(v.displayItems, displayItem{
				kind: itemAgent, platform: p,
			})
		}
		v.displayItems = append(v.displayItems, displayItem{kind: itemSeparator})
	}

	// Collapsible "All Agents" group
	label := fmt.Sprintf("All Agents (%d)", len(v.allAgents))
	v.displayItems = append(v.displayItems, displayItem{
		kind: itemToggleHeader, label: label,
	})

	if v.group2Expanded {
		for _, p := range v.allAgents {
			v.displayItems = append(v.displayItems, displayItem{
				kind: itemAgent, platform: p,
			})
		}
	}
}

// firstSelectableIndex returns the index of the first interactive item.
func (v *OnboardingToolsView) firstSelectableIndex() int {
	for i, item := range v.displayItems {
		if item.kind == itemAgent || item.kind == itemToggleHeader {
			return i
		}
	}
	return 0
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
		v.moveCursor(-1)
	case "down", "j":
		v.moveCursor(1)
	case "space":
		if v.currentSelection >= 0 && v.currentSelection < len(v.displayItems) {
			item := v.displayItems[v.currentSelection]
			switch item.kind {
			case itemAgent:
				v.selectedTools[item.platform] = !v.selectedTools[item.platform]
			case itemToggleHeader:
				v.group2Expanded = !v.group2Expanded
				v.buildDisplayItems()
				// Keep cursor on toggle header
				for i, di := range v.displayItems {
					if di.kind == itemToggleHeader {
						v.currentSelection = i
						break
					}
				}
				v.ensureVisible()
			}
		}
	case "enter":
		return true, false
	case "esc":
		return true, true
	}
	return false, false
}

// moveCursor moves the cursor by delta, skipping non-interactive items.
func (v *OnboardingToolsView) moveCursor(delta int) {
	next := v.currentSelection + delta
	for next >= 0 && next < len(v.displayItems) {
		kind := v.displayItems[next].kind
		if kind == itemAgent || kind == itemToggleHeader {
			v.currentSelection = next
			v.ensureVisible()
			return
		}
		next += delta
	}
}

// ensureVisible adjusts scroll offset so the cursor is visible.
func (v *OnboardingToolsView) ensureVisible() {
	vpHeight := v.calcViewportHeight()
	if vpHeight <= 0 {
		return
	}
	if v.currentSelection < v.scrollOffset {
		v.scrollOffset = v.currentSelection
	}
	if v.currentSelection >= v.scrollOffset+vpHeight {
		v.scrollOffset = v.currentSelection - vpHeight + 1
	}
}

// calcViewportHeight returns the number of display items visible at once.
func (v *OnboardingToolsView) calcViewportHeight() int {
	vpHeight := v.height - 14 // Account for title, subtitle, instructions, border chrome
	if vpHeight < 10 {
		vpHeight = 10
	}
	return vpHeight
}

// GetSelectedPlatforms returns the list of selected platforms.
func (v *OnboardingToolsView) GetSelectedPlatforms() []installer.Platform {
	var platforms []installer.Platform
	// Return from both groups regardless of collapse state
	for _, p := range v.detectedAgents {
		if v.selectedTools[p] {
			platforms = append(platforms, p)
		}
	}
	for _, p := range v.allAgents {
		if v.selectedTools[p] {
			platforms = append(platforms, p)
		}
	}
	return platforms
}

// GetPlatformName returns the human-readable name for a platform.
func (v *OnboardingToolsView) GetPlatformName(platform installer.Platform) string {
	info := platform.Info()
	if info.Name != "" {
		return info.Name
	}
	return string(platform)
}

// View renders the onboarding tools view.
func (v *OnboardingToolsView) View() string {
	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current.Primary).
		MarginBottom(1)

	title := titleStyle.Render("Select AI Tools to Sync")

	// Subtitle
	subtitleStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		MarginBottom(1)

	subtitle := subtitleStyle.Render("Choose which platforms to export your skills to")

	// Build visible display items with scrolling
	vpHeight := v.calcViewportHeight()
	start := v.scrollOffset
	if start < 0 {
		start = 0
	}
	end := start + vpHeight
	if end > len(v.displayItems) {
		end = len(v.displayItems)
	}

	// Render items
	var itemViews []string
	for i := start; i < end; i++ {
		item := v.displayItems[i]
		switch item.kind {
		case itemHeader:
			headerStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(theme.Current.Primary).
				MarginTop(1)
			itemViews = append(itemViews, headerStyle.Render(item.label))

		case itemSeparator:
			sepStyle := lipgloss.NewStyle().
				Foreground(theme.Current.TextMuted)
			itemViews = append(itemViews, sepStyle.Render("───"))

		case itemToggleHeader:
			arrow := "▶"
			if v.group2Expanded {
				arrow = "▼"
			}
			headerStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(theme.Current.Primary).
				MarginTop(1)
			if i == v.currentSelection {
				headerStyle = headerStyle.
					Background(theme.Current.Surface).
					Foreground(theme.Current.Accent).
					Padding(0, 1)
			}
			itemViews = append(itemViews, headerStyle.Render(arrow+" "+item.label))

		case itemAgent:
			selected := v.selectedTools[item.platform]

			var checkbox string
			if selected {
				checkbox = "☑"
			} else {
				checkbox = "☐"
			}

			info := item.platform.Info()
			platformName := info.Name
			if platformName == "" {
				platformName = string(item.platform)
			}

			// Highlight current selection
			itemStyle := lipgloss.NewStyle()
			if i == v.currentSelection {
				itemStyle = itemStyle.
					Background(theme.Current.Surface).
					Foreground(theme.Current.Accent).
					Bold(true).
					Padding(0, 1)
			}

			line := checkbox + " " + platformName
			itemViews = append(itemViews, itemStyle.Render(line))
		}
	}

	// Scroll indicators
	var scrollIndicator string
	if start > 0 {
		scrollIndicator = lipgloss.NewStyle().
			Foreground(theme.Current.TextMuted).
			Render("  ↑ more above")
	}
	var scrollBottom string
	if end < len(v.displayItems) {
		scrollBottom = lipgloss.NewStyle().
			Foreground(theme.Current.TextMuted).
			Render("  ↓ more below")
	}

	platformsContent := lipgloss.JoinVertical(lipgloss.Left, itemViews...)

	// Instructions
	instructionStyle := lipgloss.NewStyle().
		Foreground(theme.Current.TextMuted).
		MarginTop(1)

	instructions := instructionStyle.Render(
		"↑/↓ or j/k to navigate  •  Space to toggle  •  Enter to confirm  •  Esc to skip",
	)

	// Combine content
	parts := []string{title, subtitle, ""}
	if scrollIndicator != "" {
		parts = append(parts, scrollIndicator)
	}
	parts = append(parts, platformsContent)
	if scrollBottom != "" {
		parts = append(parts, scrollBottom)
	}
	parts = append(parts, "", instructions)

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// Create bordered dialog with responsive width
	maxWidth := v.width * 90 / 100
	if maxWidth < 50 {
		maxWidth = 50
	}

	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Current.Primary).
		Padding(2, 3).
		MaxWidth(maxWidth)

	renderedDialog := dialog.Render(content)

	// Center the dialog
	dialogWidth := lipgloss.Width(renderedDialog)
	paddingLeft := (v.width - dialogWidth) / 2
	if paddingLeft < 0 {
		paddingLeft = 0
	}

	paddingTop := (v.height - lipgloss.Height(renderedDialog)) / 2
	if paddingTop < 1 {
		paddingTop = 1
	}

	return lipgloss.NewStyle().
		Padding(paddingTop, paddingLeft).
		Render(renderedDialog)
}

// GetKeyboardCommands returns the keyboard commands for this view.
func (ov *OnboardingToolsView) GetKeyboardCommands() ViewCommands {
	return ViewCommands{
		ViewName: "Onboarding",
		Commands: []Command{
			{Key: "↑↓, k/j", Description: "Navigate tool options"},
			{Key: "Space", Description: "Toggle tool selection"},
			{Key: "Enter", Description: "Confirm selection"},
			{Key: "Esc", Description: "Skip onboarding"},
		},
	}
}
