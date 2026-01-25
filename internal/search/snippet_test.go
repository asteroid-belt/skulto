package search

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractSnippets_BasicMatch(t *testing.T) {
	content := "This is a guide about testing React components with Jest and React Testing Library."
	query := "testing React"

	snippets := ExtractSnippets(content, query, 3)

	assert.NotEmpty(t, snippets)
	assert.Contains(t, snippets[0].Text, "testing")
	assert.NotEmpty(t, snippets[0].Highlights)
}

func TestExtractSnippets_NoMatch(t *testing.T) {
	content := "This is a guide about Python programming."
	query := "javascript"

	snippets := ExtractSnippets(content, query, 3)

	// Should return fallback snippet
	assert.Len(t, snippets, 1)
	assert.Empty(t, snippets[0].Highlights) // No highlights in fallback
}

func TestExtractSnippets_EmptyContent(t *testing.T) {
	snippets := ExtractSnippets("", "query", 3)
	assert.Empty(t, snippets)
}

func TestExtractSnippets_EmptyQuery(t *testing.T) {
	snippets := ExtractSnippets("some content", "", 3)
	assert.Empty(t, snippets)
}

func TestExtractSnippets_MultipleMatches(t *testing.T) {
	content := `React is a JavaScript library. React components are reusable.
	Testing React is important. React Testing Library helps test React apps.`
	query := "React"

	snippets := ExtractSnippets(content, query, 3)

	assert.NotEmpty(t, snippets)
	// Should find multiple occurrences
	for _, s := range snippets {
		assert.Contains(t, strings.ToLower(s.Text), "react")
	}
}

func TestExtractSnippets_LongContent(t *testing.T) {
	// Create content with a match somewhere in the middle
	content := strings.Repeat("Lorem ipsum dolor sit amet. ", 50) +
		"This section talks about React testing. " +
		strings.Repeat("More filler content here. ", 50)
	query := "React testing"

	snippets := ExtractSnippets(content, query, 1)

	assert.NotEmpty(t, snippets)
	assert.Contains(t, strings.ToLower(snippets[0].Text), "react")
}

func TestExtractQueryTerms(t *testing.T) {
	tests := []struct {
		query    string
		expected []string
	}{
		{"testing React", []string{"testing", "react"}},
		{"a React", []string{"react"}},                        // "a" is too short
		{"React.js library", []string{"react.js", "library"}}, // Dots are preserved within words
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			terms := extractQueryTerms(tt.query)
			assert.Equal(t, tt.expected, terms)
		})
	}
}

func TestHighlightText(t *testing.T) {
	snippet := Snippet{
		Text: "Testing React components is fun",
		Highlights: []Highlight{
			{Start: 0, End: 7},  // "Testing"
			{Start: 8, End: 13}, // "React"
		},
	}

	result := HighlightText(snippet)

	assert.Contains(t, result, "**Testing**")
	assert.Contains(t, result, "**React**")
}

func TestHighlightText_NoHighlights(t *testing.T) {
	snippet := Snippet{
		Text:       "Plain text without highlights",
		Highlights: nil,
	}

	result := HighlightText(snippet)
	assert.Equal(t, snippet.Text, result)
}

func TestCreateFallbackSnippet(t *testing.T) {
	// Short content
	short := "Short content"
	snippet := createFallbackSnippet(short)
	assert.Equal(t, short, snippet.Text)

	// Long content
	long := strings.Repeat("This is some long content. ", 20)
	snippet = createFallbackSnippet(long)
	assert.Less(t, len(snippet.Text), len(long))
	assert.True(t, strings.HasSuffix(snippet.Text, "..."))
}
