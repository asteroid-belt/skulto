package components

import (
	"testing"

	"github.com/asteroid-belt/skulto/internal/search"
	"github.com/stretchr/testify/assert"
)

func TestRenderSnippet_EmptyText(t *testing.T) {
	snippet := search.Snippet{Text: ""}

	result := RenderSnippet(snippet, 80)

	assert.Empty(t, result)
}

func TestRenderSnippet_NoHighlights(t *testing.T) {
	snippet := search.Snippet{
		Text:       "This is some content without highlights",
		Highlights: nil,
	}

	result := RenderSnippet(snippet, 80)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "without highlights")
}

func TestRenderSnippet_WithHighlights(t *testing.T) {
	snippet := search.Snippet{
		Text: "testing React components",
		Highlights: []search.Highlight{
			{Start: 8, End: 13}, // "React"
		},
	}

	result := RenderSnippet(snippet, 80)

	assert.NotEmpty(t, result)
	// The result should contain ANSI escape codes for styling
	assert.Contains(t, result, "testing")
	assert.Contains(t, result, "components")
}

func TestRenderSnippet_MultipleHighlights(t *testing.T) {
	snippet := search.Snippet{
		Text: "React testing with React components",
		Highlights: []search.Highlight{
			{Start: 0, End: 5},   // first "React"
			{Start: 19, End: 24}, // second "React"
		},
	}

	result := RenderSnippet(snippet, 80)

	assert.NotEmpty(t, result)
}

func TestRenderSnippet_WithEllipsisPrefix(t *testing.T) {
	// This simulates what ExtractSnippets produces when the snippet
	// is extracted from the middle of content. The text already has
	// "..." prefix and highlight positions are adjusted for it.
	snippet := search.Snippet{
		Text: "...content with hello in it...",
		Highlights: []search.Highlight{
			{Start: 16, End: 21}, // "hello" - position accounts for "..." prefix
		},
	}

	result := RenderSnippet(snippet, 80)

	assert.NotEmpty(t, result)
	// The highlight should correctly extract "hello" from position 16-21
	// which is "hello" in "...content with hello in it..."
	assert.Contains(t, result, "content")
	assert.Contains(t, result, "in it")
}

func TestRenderSnippet_InvalidHighlightBounds(t *testing.T) {
	snippet := search.Snippet{
		Text: "short",
		Highlights: []search.Highlight{
			{Start: -5, End: 100}, // Out of bounds
		},
	}

	// Should not panic, should handle gracefully
	result := RenderSnippet(snippet, 80)
	assert.NotEmpty(t, result)
}

func TestRenderSnippet_OverlappingHighlights(t *testing.T) {
	snippet := search.Snippet{
		Text: "test content here",
		Highlights: []search.Highlight{
			{Start: 0, End: 10},
			{Start: 5, End: 15}, // Overlaps with previous
		},
	}

	// Should not panic
	result := RenderSnippet(snippet, 80)
	assert.NotEmpty(t, result)
}

func TestRenderSnippets_Empty(t *testing.T) {
	result := RenderSnippets(nil, 80)
	assert.Empty(t, result)

	result = RenderSnippets([]search.Snippet{}, 80)
	assert.Empty(t, result)
}

func TestRenderSnippets_Multiple(t *testing.T) {
	snippets := []search.Snippet{
		{Text: "First snippet"},
		{Text: "Second snippet"},
	}

	result := RenderSnippets(snippets, 80)

	assert.NotEmpty(t, result)
	assert.Contains(t, result, "First snippet")
	assert.Contains(t, result, "Second snippet")
}

func TestRenderSkillWithSnippets(t *testing.T) {
	skillContent := "React Testing Guide\nLearn to test React components"
	snippets := []search.Snippet{
		{Text: "testing components"},
	}

	result := RenderSkillWithSnippets(skillContent, snippets, 80)

	assert.Contains(t, result, "React Testing Guide")
	assert.Contains(t, result, "testing components")
}

func TestRenderSkillWithSnippets_NoSnippets(t *testing.T) {
	skillContent := "React Testing Guide"

	result := RenderSkillWithSnippets(skillContent, nil, 80)

	assert.Equal(t, "React Testing Guide", result)
}

func TestTruncateSnippet(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "needs truncation",
			input:    "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "zero max length",
			input:    "hello",
			maxLen:   0,
			expected: "",
		},
		{
			name:     "very short max",
			input:    "hello",
			maxLen:   2,
			expected: "he",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateSnippet(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultSnippetStyles(t *testing.T) {
	styles := DefaultSnippetStyles()

	// Just ensure styles are created without error
	assert.NotNil(t, styles.Normal)
	assert.NotNil(t, styles.Highlight)
	assert.NotNil(t, styles.Ellipsis)
}
