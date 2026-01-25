package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ProgressBar renders a CLI progress bar similar to the TUI Home view.
type ProgressBar struct {
	completed int
	total     int
	label     string
	width     int
}

// NewProgressBar creates a new progress bar with the specified total and width.
func NewProgressBar(total int, width int) *ProgressBar {
	if width <= 0 {
		width = 15
	}
	return &ProgressBar{
		total: total,
		width: width,
	}
}

// Update sets the current progress and label.
func (p *ProgressBar) Update(completed int, label string) {
	p.completed = completed
	p.label = label
}

// Render returns the formatted progress bar string for pull operations.
func (p *ProgressBar) Render() string {
	if p.total == 0 {
		return ""
	}

	percent := float64(p.completed) / float64(p.total)
	filled := int(float64(p.width) * percent)
	empty := p.width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	// Styles matching TUI Home view
	progressStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981")).
		Bold(true)

	barStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#10B981"))

	countStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B6B6B"))

	return progressStyle.Render("⚡ ") +
		barStyle.Render("["+bar+"]") +
		countStyle.Render(fmt.Sprintf(" %d/%d ", p.completed, p.total)) +
		progressStyle.Render(p.label)
}

// RenderScan returns a scan-themed progress bar (amber color).
func (p *ProgressBar) RenderScan() string {
	if p.total == 0 {
		return ""
	}

	percent := float64(p.completed) / float64(p.total)
	filled := int(float64(p.width) * percent)
	empty := p.width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	scanStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F59E0B")).
		Bold(true)

	barStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F59E0B"))

	countStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B6B6B"))

	return scanStyle.Render("🔒 ") +
		barStyle.Render("["+bar+"]") +
		countStyle.Render(fmt.Sprintf(" %d/%d ", p.completed, p.total)) +
		scanStyle.Render(p.label)
}

// ClearLine clears the current line for in-place progress updates.
func ClearLine() {
	fmt.Print("\r\033[K")
}
