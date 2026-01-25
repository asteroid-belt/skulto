package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

// RenderMarkdown renders markdown content using Glamour for terminal display.
// Returns a slice of lines ready for display.
func RenderMarkdown(content string, theme *Theme) []string {
	if content == "" {
		return []string{}
	}

	// Create a glamour renderer with auto style detection
	// This will automatically pick the best rendering style for the terminal
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		// Fallback to plain text if rendering fails
		return strings.Split(content, "\n")
	}

	// Render the markdown
	rendered, err := renderer.Render(content)
	if err != nil {
		// Fallback to plain text if rendering fails
		return strings.Split(content, "\n")
	}

	// Split into lines for scrolling
	lines := strings.Split(rendered, "\n")

	// Remove trailing empty lines
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}

// StripMarkdown removes markdown formatting from content for plain text display.
func StripMarkdown(content string) string {
	// Simple approach: render and strip ANSI codes
	lines := RenderMarkdown(content, nil)
	result := strings.Join(lines, "\n")

	// Remove ANSI escape codes
	result = stripANSI(result)
	return result
}

// stripANSI removes ANSI escape codes from a string.
func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false

	for _, ch := range s {
		if ch == '\x1b' {
			inEscape = true
		} else if inEscape && ch == 'm' {
			inEscape = false
		} else if !inEscape {
			result.WriteRune(ch)
		}
	}

	return result.String()
}
