package search

import (
	"strings"
	"unicode"
)

const (
	// DefaultSnippetLength is the target length for snippets.
	DefaultSnippetLength = 150
	// DefaultContextWindow is the number of characters before/after a match.
	DefaultContextWindow = 50
	// MaxSnippets is the maximum number of snippets to extract per skill.
	MaxSnippets = 3
)

// ExtractSnippets extracts relevant text snippets from content based on the query.
// Returns up to maxSnippets snippets with highlighted query matches.
func ExtractSnippets(content, query string, maxSnippets int) []Snippet {
	if content == "" || query == "" {
		return nil
	}

	if maxSnippets <= 0 {
		maxSnippets = MaxSnippets
	}

	// Normalize for matching
	contentLower := strings.ToLower(content)
	queryTerms := extractQueryTerms(query)

	if len(queryTerms) == 0 {
		return nil
	}

	// Find all match positions
	matches := findMatches(contentLower, queryTerms)
	if len(matches) == 0 {
		// No exact matches found, return first portion of content
		return []Snippet{createFallbackSnippet(content)}
	}

	// Cluster nearby matches into snippets
	snippets := clusterMatches(content, matches, maxSnippets)

	return snippets
}

// matchPosition represents a found query term match.
type matchPosition struct {
	start int
	end   int
	term  string
}

// extractQueryTerms splits a query into searchable terms.
func extractQueryTerms(query string) []string {
	words := strings.Fields(strings.ToLower(query))
	var terms []string

	for _, word := range words {
		// Remove punctuation and filter short words
		cleaned := strings.TrimFunc(word, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})
		if len(cleaned) >= 2 {
			terms = append(terms, cleaned)
		}
	}

	return terms
}

// findMatches finds all positions where query terms appear in content.
func findMatches(contentLower string, terms []string) []matchPosition {
	var matches []matchPosition
	seen := make(map[int]bool) // Avoid overlapping matches

	for _, term := range terms {
		pos := 0
		for {
			idx := strings.Index(contentLower[pos:], term)
			if idx == -1 {
				break
			}

			start := pos + idx
			end := start + len(term)

			// Skip if this position overlaps with an existing match
			if !seen[start] {
				matches = append(matches, matchPosition{
					start: start,
					end:   end,
					term:  term,
				})
				seen[start] = true
			}

			pos = start + 1
		}
	}

	// Sort by position
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].start < matches[i].start {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	return matches
}

// clusterMatches groups nearby matches into snippets.
func clusterMatches(content string, matches []matchPosition, maxSnippets int) []Snippet {
	var snippets []Snippet

	usedPositions := make(map[int]bool)

	for _, match := range matches {
		if len(snippets) >= maxSnippets {
			break
		}

		// Skip if this match is already covered by a previous snippet
		if usedPositions[match.start] {
			continue
		}

		// Calculate snippet bounds
		start := match.start - DefaultContextWindow
		if start < 0 {
			start = 0
		}

		end := match.end + DefaultContextWindow
		if end > len(content) {
			end = len(content)
		}

		// Expand to word boundaries
		start = expandToWordBoundary(content, start, -1)
		end = expandToWordBoundary(content, end, 1)

		// Collect all matches within this snippet
		var highlights []Highlight
		for _, m := range matches {
			if m.start >= start && m.end <= end {
				highlights = append(highlights, Highlight{
					Start: m.start - start,
					End:   m.end - start,
				})
				usedPositions[m.start] = true
			}
		}

		snippetText := content[start:end]

		// Add ellipsis indicators
		if start > 0 {
			snippetText = "..." + snippetText
			// Adjust highlight positions
			for i := range highlights {
				highlights[i].Start += 3
				highlights[i].End += 3
			}
		}
		if end < len(content) {
			snippetText = snippetText + "..."
		}

		snippets = append(snippets, Snippet{
			Text:       snippetText,
			Highlights: highlights,
		})
	}

	return snippets
}

// expandToWordBoundary expands a position to the nearest word boundary.
// direction: -1 for backward, 1 for forward.
func expandToWordBoundary(content string, pos, direction int) int {
	if direction < 0 {
		// Expand backward to start of word
		for pos > 0 && !unicode.IsSpace(rune(content[pos-1])) {
			pos--
		}
	} else {
		// Expand forward to end of word
		for pos < len(content) && !unicode.IsSpace(rune(content[pos])) {
			pos++
		}
	}
	return pos
}

// createFallbackSnippet creates a snippet from the beginning of content
// when no query matches are found.
func createFallbackSnippet(content string) Snippet {
	maxLen := DefaultSnippetLength
	if len(content) <= maxLen {
		return Snippet{Text: content}
	}

	// Find a good break point
	end := maxLen
	for end > maxLen-30 && end < len(content) && !unicode.IsSpace(rune(content[end])) {
		end--
	}

	return Snippet{
		Text: content[:end] + "...",
	}
}

// HighlightText applies highlighting markers to text based on highlights.
// Returns the text with **bold** markers around highlighted sections.
func HighlightText(snippet Snippet) string {
	if len(snippet.Highlights) == 0 {
		return snippet.Text
	}

	// Sort highlights by position (should already be sorted)
	result := strings.Builder{}
	lastEnd := 0

	for _, h := range snippet.Highlights {
		if h.Start > lastEnd && h.Start <= len(snippet.Text) {
			result.WriteString(snippet.Text[lastEnd:h.Start])
		}
		if h.Start < len(snippet.Text) && h.End <= len(snippet.Text) {
			result.WriteString("**")
			result.WriteString(snippet.Text[h.Start:h.End])
			result.WriteString("**")
		}
		lastEnd = h.End
	}

	if lastEnd < len(snippet.Text) {
		result.WriteString(snippet.Text[lastEnd:])
	}

	return result.String()
}
