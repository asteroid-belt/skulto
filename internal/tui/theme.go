// Package tui contains the Bubble Tea user interface components.
package tui

import (
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// Theme is an alias for theme.Theme for backwards compatibility.
type Theme = theme.Theme

// Re-export theme variables for backwards compatibility.
var (
	PunkTheme    = theme.PunkTheme
	NeonTheme    = theme.NeonTheme
	BloodTheme   = theme.BloodTheme
	CurrentTheme = &theme.Current
)

// GetTagColor is an alias for theme.GetTagColor for backwards compatibility.
func GetTagColor(category string) lipgloss.AdaptiveColor {
	return theme.GetTagColor(category)
}
