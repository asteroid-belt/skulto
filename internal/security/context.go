package security

// ContextMatch represents a match of an allowlist pattern near a threat.
type ContextMatch struct {
	AllowlistID  string
	Type         ContextType
	MatchedText  string
	Position     int
	DistanceFrom int
}

// ContextAnalyzer analyzes content for mitigating context around threats.
type ContextAnalyzer struct {
	patterns []AllowlistPattern
	window   int
}

// NewContextAnalyzer creates a new ContextAnalyzer with default settings.
func NewContextAnalyzer() *ContextAnalyzer {
	return &ContextAnalyzer{
		patterns: AllowlistPatterns,
		window:   DefaultProximityWindow,
	}
}

// NewContextAnalyzerWithWindow creates a new ContextAnalyzer with a custom proximity window.
func NewContextAnalyzerWithWindow(window int) *ContextAnalyzer {
	return &ContextAnalyzer{
		patterns: AllowlistPatterns,
		window:   window,
	}
}

// FindContext searches for allowlist patterns within the proximity window of a threat.
// threatStart and threatEnd are the character positions of the threat in content.
func (ca *ContextAnalyzer) FindContext(content string, threatStart, threatEnd int) []ContextMatch {
	var matches []ContextMatch

	contentLen := len(content)
	if contentLen == 0 {
		return matches
	}

	// Clamp threat positions to valid range
	if threatStart < 0 {
		threatStart = 0
	}
	if threatStart > contentLen {
		threatStart = contentLen
	}
	if threatEnd < threatStart {
		threatEnd = threatStart
	}
	if threatEnd > contentLen {
		threatEnd = contentLen
	}

	// Calculate search bounds
	searchStart := threatStart - ca.window
	if searchStart < 0 {
		searchStart = 0
	}
	searchEnd := threatEnd + ca.window
	if searchEnd > contentLen {
		searchEnd = contentLen
	}

	// Ensure valid slice bounds
	if searchStart >= searchEnd {
		return matches
	}

	// Extract the search region
	searchRegion := content[searchStart:searchEnd]

	// Search for each allowlist pattern
	for _, pattern := range ca.patterns {
		locs := pattern.Regex.FindAllStringIndex(searchRegion, -1)
		for _, loc := range locs {
			// Calculate absolute position in original content
			absPos := searchStart + loc[0]
			matchedText := searchRegion[loc[0]:loc[1]]

			// Calculate distance from threat
			var distance int
			if absPos < threatStart {
				// Pattern is before the threat
				distance = threatStart - (absPos + len(matchedText))
				if distance < 0 {
					distance = 0
				}
			} else if absPos >= threatEnd {
				// Pattern is after the threat
				distance = absPos - threatEnd
			} else {
				// Pattern overlaps with threat
				distance = 0
			}

			matches = append(matches, ContextMatch{
				AllowlistID:  pattern.ID,
				Type:         pattern.Type,
				MatchedText:  matchedText,
				Position:     absPos,
				DistanceFrom: distance,
			})
		}
	}

	return matches
}

// CalculateMitigation calculates the total mitigation score from context matches.
// The score is the sum of MitigationWeight for all unique context types found.
func (ca *ContextAnalyzer) CalculateMitigation(matches []ContextMatch) int {
	total := 0
	for _, match := range matches {
		total += match.Type.MitigationWeight()
	}
	return total
}
