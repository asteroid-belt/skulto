package prompts

import (
	"fmt"
	"strings"

	"github.com/asteroid-belt/skulto/internal/installer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// platformItemKind identifies the type of a display item.
type platformItemKind int

const (
	pikAgent        platformItemKind = iota // Selectable platform entry
	pikHeader                               // Non-interactive group header
	pikSeparator                            // Visual separator
	pikToggleHeader                         // Collapsible group toggle
)

// platformItem represents a single row in the grouped list.
type platformItem struct {
	kind     platformItemKind
	platform installer.DetectedPlatform
	label    string
}

// platformSelectorModel is a Bubble Tea model for the grouped platform selector.
type platformSelectorModel struct {
	// Data
	detected    []installer.DetectedPlatform // Detected platforms (group 1)
	others      []installer.DetectedPlatform // Non-detected platforms (group 2)
	installed   map[string]bool              // Already-installed platform IDs
	selected    map[string]bool              // Currently selected platform IDs
	displayItems []platformItem

	// UI state
	cursor         int
	group2Expanded bool
	cancelled      bool
	confirmed      bool
}

func newPlatformSelectorModel(platforms []installer.DetectedPlatform, installedLocations []installer.InstallLocation, preselected []string) platformSelectorModel {
	installed := make(map[string]bool)
	for _, loc := range installedLocations {
		installed[string(loc.Platform)] = true
	}

	var detected, others []installer.DetectedPlatform
	for _, p := range platforms {
		if installed[p.ID] {
			continue // Skip already-installed
		}
		if p.Detected {
			detected = append(detected, p)
		} else {
			others = append(others, p)
		}
	}

	// Build selected set: only pre-select if explicitly provided
	selected := make(map[string]bool)
	for _, id := range preselected {
		if !installed[id] {
			selected[id] = true
		}
	}

	m := platformSelectorModel{
		detected:  detected,
		others:    others,
		installed: installed,
		selected:  selected,
	}
	m.buildDisplayItems()
	m.cursor = m.firstInteractive()
	return m
}

func (m *platformSelectorModel) buildDisplayItems() {
	m.displayItems = nil

	if len(m.detected) > 0 {
		m.displayItems = append(m.displayItems, platformItem{
			kind: pikHeader, label: "Detected on your system",
		})
		for _, p := range m.detected {
			m.displayItems = append(m.displayItems, platformItem{
				kind: pikAgent, platform: p,
			})
		}
		m.displayItems = append(m.displayItems, platformItem{kind: pikSeparator})
	}

	if len(m.others) > 0 {
		label := fmt.Sprintf("Other Agents (%d)", len(m.others))
		m.displayItems = append(m.displayItems, platformItem{
			kind: pikToggleHeader, label: label,
		})

		if m.group2Expanded {
			for _, p := range m.others {
				m.displayItems = append(m.displayItems, platformItem{
					kind: pikAgent, platform: p,
				})
			}
		}
	}
}

func (m *platformSelectorModel) firstInteractive() int {
	for i, item := range m.displayItems {
		if item.kind == pikAgent || item.kind == pikToggleHeader {
			return i
		}
	}
	return 0
}

func (m platformSelectorModel) Init() tea.Cmd {
	return nil
}

func (m platformSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)
		case " ", "x":
			if m.cursor >= 0 && m.cursor < len(m.displayItems) {
				item := m.displayItems[m.cursor]
				switch item.kind {
				case pikAgent:
					m.selected[item.platform.ID] = !m.selected[item.platform.ID]
				case pikToggleHeader:
					m.group2Expanded = !m.group2Expanded
					m.buildDisplayItems()
					// Keep cursor on toggle
					for i, di := range m.displayItems {
						if di.kind == pikToggleHeader {
							m.cursor = i
							break
						}
					}
				}
			}
		case "a":
			// Select all visible
			for _, item := range m.displayItems {
				if item.kind == pikAgent {
					m.selected[item.platform.ID] = true
				}
			}
		case "n":
			// Select none
			for k := range m.selected {
				m.selected[k] = false
			}
		case "enter":
			m.confirmed = true
			return m, tea.Quit
		case "ctrl+c", "q":
			m.cancelled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *platformSelectorModel) moveCursor(delta int) {
	next := m.cursor + delta
	for next >= 0 && next < len(m.displayItems) {
		kind := m.displayItems[next].kind
		if kind == pikAgent || kind == pikToggleHeader {
			m.cursor = next
			return
		}
		next += delta
	}
}

// Styles
var (
	psTitle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00d4aa"))
	psDescription = lipgloss.NewStyle().Foreground(lipgloss.Color("#888"))
	psDetected    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00d4aa"))
	psNormal      = lipgloss.NewStyle()
	psCursorStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00d4aa"))
	psSeparator   = lipgloss.NewStyle().Foreground(lipgloss.Color("#555"))
	psHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00d4aa"))
	psToggle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00d4aa"))
)

func (m platformSelectorModel) View() string {
	var b strings.Builder

	b.WriteString(psTitle.Render("Select platforms to install to"))
	b.WriteString("\n")
	b.WriteString(psDescription.Render("Space to toggle, Enter to confirm"))
	b.WriteString("\n")

	for i, item := range m.displayItems {
		isCursor := i == m.cursor

		switch item.kind {
		case pikHeader:
			b.WriteString(psHeaderStyle.Render(item.label))
			b.WriteString("\n")

		case pikSeparator:
			b.WriteString(psSeparator.Render("───"))
			b.WriteString("\n")

		case pikToggleHeader:
			arrow := "▶"
			if m.group2Expanded {
				arrow = "▼"
			}
			cursor := "  "
			if isCursor {
				cursor = "> "
			}
			line := cursor + arrow + " " + item.label
			if isCursor {
				b.WriteString(psCursorStyle.Render(line))
			} else {
				b.WriteString(psToggle.Render(line))
			}
			b.WriteString("\n")

		case pikAgent:
			check := "•"
			if m.selected[item.platform.ID] {
				check = "✓"
			}

			cursor := "  "
			if isCursor {
				cursor = "> "
			}

			label := item.platform.Name + " (" + item.platform.Path + ")"
			if item.platform.Detected {
				label += " ✓ detected"
			}

			line := cursor + check + " " + label
			if isCursor {
				b.WriteString(psCursorStyle.Render(line))
			} else if item.platform.Detected && m.selected[item.platform.ID] {
				b.WriteString(psDetected.Render(line))
			} else {
				b.WriteString(psNormal.Render(line))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(psDescription.Render("a: select all • n: select none • q: cancel"))

	return b.String()
}

// RunGroupedPlatformSelector shows the grouped platform selector with collapsible sections.
// Detected platforms appear at the top in teal. Others are in a collapsed group.
func RunGroupedPlatformSelector(platforms []installer.DetectedPlatform, installedLocations []installer.InstallLocation, preselected []string) (*PlatformSelectorResult, error) {
	// Build installed info for the result
	installedMap := make(map[string]bool)
	var installedPlatformIDs []string
	for _, loc := range installedLocations {
		id := string(loc.Platform)
		if !installedMap[id] {
			installedMap[id] = true
			installedPlatformIDs = append(installedPlatformIDs, id)
		}
	}

	// Check if all platforms are already installed
	selectableCount := 0
	for _, p := range platforms {
		if !installedMap[p.ID] {
			selectableCount++
		}
	}
	if selectableCount == 0 {
		return &PlatformSelectorResult{
			AlreadyInstalled:    installedPlatformIDs,
			AllAlreadyInstalled: true,
		}, nil
	}

	// Print installed locations info
	if len(installedPlatformIDs) > 0 {
		fmt.Println("Already installed:")
		for _, p := range platforms {
			if installedMap[p.ID] {
				fmt.Printf("  • %s (%s)\n", p.Name, p.Path)
			}
		}
		fmt.Println()
	}

	model := newPlatformSelectorModel(platforms, installedLocations, preselected)

	// If no detected and no others, nothing to show
	if len(model.detected) == 0 && len(model.others) == 0 {
		return &PlatformSelectorResult{
			AlreadyInstalled:    installedPlatformIDs,
			AllAlreadyInstalled: true,
		}, nil
	}

	// Auto-expand if nothing is detected
	if len(model.detected) == 0 {
		model.group2Expanded = true
		model.buildDisplayItems()
		model.cursor = model.firstInteractive()
	}

	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	final := finalModel.(platformSelectorModel)
	if final.cancelled {
		return nil, fmt.Errorf("cancelled")
	}

	var selected []string
	for id, sel := range final.selected {
		if sel {
			selected = append(selected, id)
		}
	}

	return &PlatformSelectorResult{
		Selected:         selected,
		AlreadyInstalled: installedPlatformIDs,
	}, nil
}
