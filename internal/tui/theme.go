// Package tui contains the Bubble Tea user interface components.
package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme defines the color palette for the TUI.
type Theme struct {
	// Primary colors
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color

	// Background colors
	Background lipgloss.Color
	Surface    lipgloss.Color
	Overlay    lipgloss.Color

	// Text colors
	Text          lipgloss.Color
	TextMuted     lipgloss.Color
	TextHighlight lipgloss.Color

	// Semantic colors
	Success lipgloss.Color
	Warning lipgloss.Color
	Error   lipgloss.Color
	Info    lipgloss.Color

	// Tag category colors
	TagLanguage  lipgloss.Color
	TagFramework lipgloss.Color
	TagTool      lipgloss.Color
	TagConcept   lipgloss.Color
}

// PunkTheme is the default skull-themed color scheme.
var PunkTheme = Theme{
	Primary:   lipgloss.Color("#DC143C"), // Crimson
	Secondary: lipgloss.Color("#9B59B6"), // Purple
	Accent:    lipgloss.Color("#F1C40F"), // Gold

	Background: lipgloss.Color("#0D0D0D"), // Near black
	Surface:    lipgloss.Color("#1A1A1A"), // Dark gray
	Overlay:    lipgloss.Color("#2D2D2D"), // Medium gray

	Text:          lipgloss.Color("#E5E5E5"), // Off-white
	TextMuted:     lipgloss.Color("#6B6B6B"), // Gray
	TextHighlight: lipgloss.Color("#FFFFFF"), // Pure white

	Success: lipgloss.Color("#00FF41"), // Matrix green
	Warning: lipgloss.Color("#FF6B35"), // Fire orange
	Error:   lipgloss.Color("#FF0040"), // Hot red
	Info:    lipgloss.Color("#00D4FF"), // Cyan

	TagLanguage:  lipgloss.Color("#8B5CF6"), // Purple
	TagFramework: lipgloss.Color("#EC4899"), // Pink
	TagTool:      lipgloss.Color("#10B981"), // Emerald
	TagConcept:   lipgloss.Color("#F59E0B"), // Amber
}

// NeonTheme is an alternative synthwave-inspired color scheme.
var NeonTheme = Theme{
	Primary:   lipgloss.Color("#FF00FF"), // Magenta
	Secondary: lipgloss.Color("#00FFFF"), // Cyan
	Accent:    lipgloss.Color("#FFFF00"), // Yellow

	Background: lipgloss.Color("#000000"), // Pure black
	Surface:    lipgloss.Color("#111111"), // Very dark gray
	Overlay:    lipgloss.Color("#222222"), // Dark gray

	Text:          lipgloss.Color("#FFFFFF"), // White
	TextMuted:     lipgloss.Color("#888888"), // Medium gray
	TextHighlight: lipgloss.Color("#FFFFFF"), // Bright white

	Success: lipgloss.Color("#39FF14"), // Lime green
	Warning: lipgloss.Color("#FF9500"), // Orange
	Error:   lipgloss.Color("#FF073A"), // Red
	Info:    lipgloss.Color("#00BFFF"), // Sky blue

	TagLanguage:  lipgloss.Color("#FF00FF"), // Magenta
	TagFramework: lipgloss.Color("#00FFFF"), // Cyan
	TagTool:      lipgloss.Color("#39FF14"), // Lime
	TagConcept:   lipgloss.Color("#FF9500"), // Orange
}

// BloodTheme is a dark red and black theme.
var BloodTheme = Theme{
	Primary:   lipgloss.Color("#8B0000"), // Dark red
	Secondary: lipgloss.Color("#C41E3A"), // Crimson
	Accent:    lipgloss.Color("#FFD700"), // Gold

	Background: lipgloss.Color("#0A0000"), // Almost black
	Surface:    lipgloss.Color("#1A0000"), // Very dark red
	Overlay:    lipgloss.Color("#2D0000"), // Dark red

	Text:          lipgloss.Color("#EEEEEE"), // Off white
	TextMuted:     lipgloss.Color("#666666"), // Gray
	TextHighlight: lipgloss.Color("#FFFFFF"), // White

	Success: lipgloss.Color("#00AA00"), // Green
	Warning: lipgloss.Color("#FFAA00"), // Orange
	Error:   lipgloss.Color("#FF0000"), // Red
	Info:    lipgloss.Color("#00AAFF"), // Cyan

	TagLanguage:  lipgloss.Color("#AA00FF"), // Purple
	TagFramework: lipgloss.Color("#FF00AA"), // Pink
	TagTool:      lipgloss.Color("#00FF00"), // Lime
	TagConcept:   lipgloss.Color("#FFAA00"), // Orange
}

// CurrentTheme is the active theme (can be changed at runtime).
var CurrentTheme = PunkTheme
