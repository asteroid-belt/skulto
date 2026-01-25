// Package theme provides color theming for the TUI.
package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme defines the color palette for the TUI.
type Theme struct {
	// Primary colors
	Primary   lipgloss.AdaptiveColor
	Secondary lipgloss.AdaptiveColor
	Accent    lipgloss.AdaptiveColor

	// Background colors
	Background lipgloss.AdaptiveColor
	Surface    lipgloss.AdaptiveColor
	Overlay    lipgloss.AdaptiveColor

	// Text colors
	Text          lipgloss.AdaptiveColor
	TextMuted     lipgloss.AdaptiveColor
	TextHighlight lipgloss.AdaptiveColor

	// Semantic colors
	Success lipgloss.AdaptiveColor
	Warning lipgloss.AdaptiveColor
	Error   lipgloss.AdaptiveColor
	Info    lipgloss.AdaptiveColor

	// Tag category colors
	TagLanguage  lipgloss.AdaptiveColor
	TagFramework lipgloss.AdaptiveColor
	TagTool      lipgloss.AdaptiveColor
	TagConcept   lipgloss.AdaptiveColor
	TagDomain    lipgloss.AdaptiveColor
	TagMine      lipgloss.AdaptiveColor
}

// PunkTheme is the default skull-themed color scheme.
var PunkTheme = Theme{
	Primary:   lipgloss.AdaptiveColor{Light: "#8B0000", Dark: "#DC143C"}, // Crimson
	Secondary: lipgloss.AdaptiveColor{Light: "#6B3FA0", Dark: "#9B59B6"}, // Purple
	Accent:    lipgloss.AdaptiveColor{Light: "#B8860B", Dark: "#F1C40F"}, // Gold

	Background: lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#0D0D0D"}, // White / Near black
	Surface:    lipgloss.AdaptiveColor{Light: "#F5F5F5", Dark: "#1A1A1A"}, // Light gray / Dark gray
	Overlay:    lipgloss.AdaptiveColor{Light: "#E5E5E5", Dark: "#2D2D2D"}, // Medium gray

	Text:          lipgloss.AdaptiveColor{Light: "#1A1A1A", Dark: "#E5E5E5"}, // Dark / Light
	TextMuted:     lipgloss.AdaptiveColor{Light: "#6B6B6B", Dark: "#6B6B6B"}, // Gray (same)
	TextHighlight: lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}, // Black / White

	Success: lipgloss.AdaptiveColor{Light: "#008000", Dark: "#00FF41"}, // Green / Matrix green
	Warning: lipgloss.AdaptiveColor{Light: "#CC5500", Dark: "#FF6B35"}, // Orange / Fire orange
	Error:   lipgloss.AdaptiveColor{Light: "#CC0033", Dark: "#FF0040"}, // Red / Hot red
	Info:    lipgloss.AdaptiveColor{Light: "#0088CC", Dark: "#00D4FF"}, // Blue / Cyan

	TagLanguage:  lipgloss.AdaptiveColor{Light: "#6B3FA0", Dark: "#8B5CF6"}, // Purple
	TagFramework: lipgloss.AdaptiveColor{Light: "#C41E7A", Dark: "#EC4899"}, // Pink
	TagTool:      lipgloss.AdaptiveColor{Light: "#0D8A5E", Dark: "#10B981"}, // Emerald
	TagConcept:   lipgloss.AdaptiveColor{Light: "#B87A00", Dark: "#F59E0B"}, // Amber
	TagDomain:    lipgloss.AdaptiveColor{Light: "#1E5FAA", Dark: "#3B82F6"}, // Blue
	TagMine:      lipgloss.AdaptiveColor{Light: "#8B0000", Dark: "#DC143C"}, // Crimson
}

// NeonTheme is an alternative synthwave-inspired color scheme.
var NeonTheme = Theme{
	Primary:   lipgloss.AdaptiveColor{Light: "#AA00AA", Dark: "#FF00FF"}, // Magenta
	Secondary: lipgloss.AdaptiveColor{Light: "#008B8B", Dark: "#00FFFF"}, // Cyan
	Accent:    lipgloss.AdaptiveColor{Light: "#B8B800", Dark: "#FFFF00"}, // Yellow

	Background: lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#000000"},
	Surface:    lipgloss.AdaptiveColor{Light: "#F0F0F0", Dark: "#111111"},
	Overlay:    lipgloss.AdaptiveColor{Light: "#E0E0E0", Dark: "#222222"},

	Text:          lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"},
	TextMuted:     lipgloss.AdaptiveColor{Light: "#888888", Dark: "#888888"},
	TextHighlight: lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"},

	Success: lipgloss.AdaptiveColor{Light: "#228B22", Dark: "#39FF14"},
	Warning: lipgloss.AdaptiveColor{Light: "#CC7700", Dark: "#FF9500"},
	Error:   lipgloss.AdaptiveColor{Light: "#CC0022", Dark: "#FF073A"},
	Info:    lipgloss.AdaptiveColor{Light: "#0077BB", Dark: "#00BFFF"},

	TagLanguage:  lipgloss.AdaptiveColor{Light: "#AA00AA", Dark: "#FF00FF"},
	TagFramework: lipgloss.AdaptiveColor{Light: "#008B8B", Dark: "#00FFFF"},
	TagTool:      lipgloss.AdaptiveColor{Light: "#228B22", Dark: "#39FF14"},
	TagConcept:   lipgloss.AdaptiveColor{Light: "#CC7700", Dark: "#FF9500"},
	TagDomain:    lipgloss.AdaptiveColor{Light: "#0077BB", Dark: "#00BFFF"},
	TagMine:      lipgloss.AdaptiveColor{Light: "#AA00AA", Dark: "#FF00FF"},
}

// BloodTheme is a dark red and black theme.
var BloodTheme = Theme{
	Primary:   lipgloss.AdaptiveColor{Light: "#660000", Dark: "#8B0000"},
	Secondary: lipgloss.AdaptiveColor{Light: "#8B1A2B", Dark: "#C41E3A"},
	Accent:    lipgloss.AdaptiveColor{Light: "#B8860B", Dark: "#FFD700"},

	Background: lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#0A0000"},
	Surface:    lipgloss.AdaptiveColor{Light: "#FFF5F5", Dark: "#1A0000"},
	Overlay:    lipgloss.AdaptiveColor{Light: "#FFE5E5", Dark: "#2D0000"},

	Text:          lipgloss.AdaptiveColor{Light: "#1A0000", Dark: "#EEEEEE"},
	TextMuted:     lipgloss.AdaptiveColor{Light: "#666666", Dark: "#666666"},
	TextHighlight: lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"},

	Success: lipgloss.AdaptiveColor{Light: "#006600", Dark: "#00AA00"},
	Warning: lipgloss.AdaptiveColor{Light: "#CC8800", Dark: "#FFAA00"},
	Error:   lipgloss.AdaptiveColor{Light: "#CC0000", Dark: "#FF0000"},
	Info:    lipgloss.AdaptiveColor{Light: "#0077AA", Dark: "#00AAFF"},

	TagLanguage:  lipgloss.AdaptiveColor{Light: "#7700AA", Dark: "#AA00FF"},
	TagFramework: lipgloss.AdaptiveColor{Light: "#AA0077", Dark: "#FF00AA"},
	TagTool:      lipgloss.AdaptiveColor{Light: "#00AA00", Dark: "#00FF00"},
	TagConcept:   lipgloss.AdaptiveColor{Light: "#CC8800", Dark: "#FFAA00"},
	TagDomain:    lipgloss.AdaptiveColor{Light: "#0077AA", Dark: "#00AAFF"},
	TagMine:      lipgloss.AdaptiveColor{Light: "#660000", Dark: "#8B0000"},
}

// Current is the active theme (can be changed at runtime).
var Current = PunkTheme

// GetTagColor returns the appropriate AdaptiveColor for a tag category.
func GetTagColor(category string) lipgloss.AdaptiveColor {
	switch category {
	case "language":
		return Current.TagLanguage
	case "framework":
		return Current.TagFramework
	case "tool":
		return Current.TagTool
	case "concept":
		return Current.TagConcept
	case "domain":
		return Current.TagDomain
	case "mine":
		return Current.TagMine
	default:
		return Current.Primary
	}
}
