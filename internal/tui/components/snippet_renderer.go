package components

import (
	"strings"

	"github.com/asteroid-belt/skulto/internal/search"
	"github.com/asteroid-belt/skulto/internal/tui/theme"
	"github.com/charmbracelet/lipgloss"
)

// SnippetStyles holds the styles for rendering snippets.
type SnippetStyles struct {
	Normal    lipgloss.Style
	Highlight lipgloss.Style
	Ellipsis  lipgloss.Style
}

// DefaultSnippetStyles returns the default snippet styles.
func DefaultSnippetStyles() SnippetStyles {
	return SnippetStyles{
		Normal: lipgloss.NewStyle().
			Foreground(theme.Current.TextMuted),
		Highlight: lipgloss.NewStyle().
			Foreground(theme.Current.Accent).
			Bold(true),
		Ellipsis: lipgloss.NewStyle().
			Foreground(theme.Current.TextMuted),
	}
}

// RenderSnippet renders a snippet with highlights for TUI display.
func RenderSnippet(snippet search.Snippet, width int) string {
	return RenderSnippetWithStyles(snippet, width, DefaultSnippetStyles())
}

// RenderSnippetWithStyles renders a snippet with custom styles.
// Note: The snippet text from search.ExtractSnippets already includes "..." prefix/suffix
// when appropriate, with highlight positions adjusted accordingly. We render it as-is.
func RenderSnippetWithStyles(snippet search.Snippet, width int, styles SnippetStyles) string {
	if snippet.Text == "" {
		return ""
	}

	text := snippet.Text

	// If no highlights, return plain styled text
	if len(snippet.Highlights) == 0 {
		return styles.Normal.Render(truncateSnippet(text, width))
	}

	// Build string with highlights applied
	var result strings.Builder

	lastEnd := 0
	for _, h := range snippet.Highlights {
		// Validate bounds
		start := h.Start
		end := h.End
		if start < 0 {
			start = 0
		}
		if end > len(text) {
			end = len(text)
		}
		if start >= end {
			continue
		}

		// Render text before highlight
		if start > lastEnd && lastEnd < len(text) {
			beforeEnd := min(start, len(text))
			result.WriteString(styles.Normal.Render(text[lastEnd:beforeEnd]))
		}

		// Render highlighted text
		result.WriteString(styles.Highlight.Render(text[start:end]))
		lastEnd = end
	}

	// Render remaining text after last highlight
	if lastEnd < len(text) {
		result.WriteString(styles.Normal.Render(text[lastEnd:]))
	}

	return result.String()
}

// RenderSnippets renders multiple snippets, joined with newlines.
func RenderSnippets(snippets []search.Snippet, width int) string {
	if len(snippets) == 0 {
		return ""
	}

	var parts []string
	for _, snippet := range snippets {
		rendered := RenderSnippet(snippet, width)
		if rendered != "" {
			parts = append(parts, rendered)
		}
	}

	return strings.Join(parts, "\n")
}

// RenderSkillWithSnippets renders a skill title/description with its snippets below.
func RenderSkillWithSnippets(skillContent string, snippets []search.Snippet, width int) string {
	var parts []string
	parts = append(parts, skillContent)

	if len(snippets) > 0 {
		snippetStyle := lipgloss.NewStyle().
			MarginLeft(2).
			Width(width - 4)

		for _, snippet := range snippets {
			rendered := RenderSnippet(snippet, width-4)
			if rendered != "" {
				parts = append(parts, snippetStyle.Render(rendered))
			}
		}
	}

	return strings.Join(parts, "\n")
}

// truncateSnippet truncates a string to maxLen, adding ellipsis if needed.
func truncateSnippet(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
