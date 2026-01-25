package security

import (
	"testing"
)

func TestNewContextAnalyzer(t *testing.T) {
	ca := NewContextAnalyzer()

	if ca == nil {
		t.Fatal("NewContextAnalyzer() returned nil")
	}

	if ca.window != DefaultProximityWindow {
		t.Errorf("Expected window=%d, got %d", DefaultProximityWindow, ca.window)
	}

	if len(ca.patterns) != len(AllowlistPatterns) {
		t.Errorf("Expected %d patterns, got %d", len(AllowlistPatterns), len(ca.patterns))
	}
}

func TestNewContextAnalyzerWithWindow(t *testing.T) {
	customWindow := 500
	ca := NewContextAnalyzerWithWindow(customWindow)

	if ca == nil {
		t.Fatal("NewContextAnalyzerWithWindow() returned nil")
	}

	if ca.window != customWindow {
		t.Errorf("Expected window=%d, got %d", customWindow, ca.window)
	}

	if len(ca.patterns) != len(AllowlistPatterns) {
		t.Errorf("Expected %d patterns, got %d", len(AllowlistPatterns), len(ca.patterns))
	}
}

func TestFindContext_NoMatch(t *testing.T) {
	ca := NewContextAnalyzer()

	content := "This is some content with a potential threat pattern here that has no mitigating context around it."
	threatStart := 40
	threatEnd := 54

	matches := ca.FindContext(content, threatStart, threatEnd)

	if len(matches) != 0 {
		t.Errorf("Expected no matches, got %d: %+v", len(matches), matches)
	}
}

func TestFindContext_DefensiveNearby(t *testing.T) {
	ca := NewContextAnalyzer()

	// Content with defensive context near a threat
	content := "To defend against injection attacks, you should always validate input. Here is the dangerous pattern: exec(userInput). Always sanitize user data."
	threatStart := 85
	threatEnd := 101

	matches := ca.FindContext(content, threatStart, threatEnd)

	if len(matches) == 0 {
		t.Fatal("Expected to find defensive context matches, got none")
	}

	// Check that we found the "defend against" pattern
	foundDefendAgainst := false
	for _, match := range matches {
		if match.AllowlistID == "DEF-001" {
			foundDefendAgainst = true
			if match.Type != ContextDefensive {
				t.Errorf("Expected ContextDefensive, got %s", match.Type)
			}
		}
	}

	if !foundDefendAgainst {
		t.Error("Expected to find DEF-001 (defend against) pattern")
	}
}

func TestFindContext_MultipleMatches(t *testing.T) {
	ca := NewContextAnalyzer()

	// Content with multiple context patterns
	content := "This is a common vulnerability (CVE-2024-5678) that we need to defend against. Security best practices recommend input validation to protect from this threat."
	threatStart := 50
	threatEnd := 70

	matches := ca.FindContext(content, threatStart, threatEnd)

	if len(matches) < 2 {
		t.Errorf("Expected at least 2 matches, got %d", len(matches))
	}

	// Check for different types of matches
	hasEducational := false
	hasDefensive := false
	for _, match := range matches {
		if match.Type == ContextEducational {
			hasEducational = true
		}
		if match.Type == ContextDefensive {
			hasDefensive = true
		}
	}

	if !hasEducational {
		t.Error("Expected to find educational context (CVE reference)")
	}
	if !hasDefensive {
		t.Error("Expected to find defensive context")
	}
}

func TestFindContext_WindowBoundary(t *testing.T) {
	ca := NewContextAnalyzerWithWindow(50)

	// Context pattern is outside the window
	content := "Security best practices are important. " +
		"Here is a lot of text that creates distance between the context and the threat location. " +
		"More filler text to ensure we exceed the window size. " +
		"The threat appears here."
	threatStart := 180
	threatEnd := 200

	matches := ca.FindContext(content, threatStart, threatEnd)

	// With a small window, the "security best practices" at the start should not be found
	for _, match := range matches {
		if match.AllowlistID == "DEF-002" {
			t.Error("Should not find DEF-002 pattern outside window")
		}
	}
}

func TestCalculateMitigation(t *testing.T) {
	ca := NewContextAnalyzer()

	tests := []struct {
		name     string
		matches  []ContextMatch
		expected int
	}{
		{
			name:     "no matches",
			matches:  []ContextMatch{},
			expected: 0,
		},
		{
			name: "single defensive",
			matches: []ContextMatch{
				{AllowlistID: "DEF-001", Type: ContextDefensive},
			},
			expected: 3,
		},
		{
			name: "single educational",
			matches: []ContextMatch{
				{AllowlistID: "EDU-001", Type: ContextEducational},
			},
			expected: 2,
		},
		{
			name: "single documentation",
			matches: []ContextMatch{
				{AllowlistID: "DOC-001", Type: ContextDocumentation},
			},
			expected: 1,
		},
		{
			name: "mixed matches",
			matches: []ContextMatch{
				{AllowlistID: "DEF-001", Type: ContextDefensive},
				{AllowlistID: "EDU-001", Type: ContextEducational},
				{AllowlistID: "DOC-001", Type: ContextDocumentation},
			},
			expected: 6, // 3 + 2 + 1
		},
		{
			name: "multiple same type",
			matches: []ContextMatch{
				{AllowlistID: "DEF-001", Type: ContextDefensive},
				{AllowlistID: "DEF-002", Type: ContextDefensive},
			},
			expected: 6, // 3 + 3
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ca.CalculateMitigation(tc.matches)
			if got != tc.expected {
				t.Errorf("CalculateMitigation(): expected %d, got %d", tc.expected, got)
			}
		})
	}
}

func TestFindContext_DistanceCalculation(t *testing.T) {
	ca := NewContextAnalyzer()

	// "defend against" appears before the threat
	content := "We defend against attacks. The threat is here."
	threatStart := 28
	threatEnd := 44

	matches := ca.FindContext(content, threatStart, threatEnd)

	for _, match := range matches {
		if match.AllowlistID == "DEF-001" {
			// The pattern "defend against" ends around position 17
			// Threat starts at 28, so distance should be around 11
			if match.DistanceFrom < 0 {
				t.Errorf("Distance should not be negative, got %d", match.DistanceFrom)
			}
		}
	}
}
