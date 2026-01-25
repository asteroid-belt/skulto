package security

import (
	"testing"
)

func TestAllowlistPatternsNotEmpty(t *testing.T) {
	if len(AllowlistPatterns) < 15 {
		t.Errorf("Expected at least 15 allowlist patterns, got %d", len(AllowlistPatterns))
	}
}

func TestAllAllowlistPatternsCompile(t *testing.T) {
	for _, pattern := range AllowlistPatterns {
		t.Run(pattern.ID, func(t *testing.T) {
			if pattern.ID == "" {
				t.Error("Pattern ID is empty")
			}
			if pattern.Name == "" {
				t.Error("Pattern Name is empty")
			}
			if pattern.Type == "" {
				t.Error("Pattern Type is empty")
			}
			if pattern.Regex == nil {
				t.Error("Pattern Regex is nil (failed to compile)")
			}
		})
	}
}

func TestDefensiveContextPatterns(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "defend against attack",
			input:    "How to defend against SQL injection attacks",
			expected: true,
		},
		{
			name:     "protect from threats",
			input:    "protect from malicious actors",
			expected: true,
		},
		{
			name:     "security best practice",
			input:    "This is a security best practice",
			expected: true,
		},
		{
			name:     "security best practices plural",
			input:    "Follow these security best practices",
			expected: true,
		},
		{
			name:     "never do",
			input:    "never do this in production",
			expected: true,
		},
		{
			name:     "don't use",
			input:    "don't use eval() with user input",
			expected: true,
		},
		{
			name:     "input validation",
			input:    "Always perform input validation",
			expected: true,
		},
		{
			name:     "sanitizing input",
			input:    "sanitizing input before processing",
			expected: true,
		},
		{
			name:     "no defensive context",
			input:    "This is just regular content without defensive patterns",
			expected: false,
		},
	}

	defensivePatterns := make([]AllowlistPattern, 0)
	for _, p := range AllowlistPatterns {
		if p.Type == ContextDefensive {
			defensivePatterns = append(defensivePatterns, p)
		}
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matched := false
			for _, pattern := range defensivePatterns {
				if pattern.Regex.MatchString(tc.input) {
					matched = true
					break
				}
			}
			if matched != tc.expected {
				t.Errorf("Input %q: expected match=%v, got match=%v", tc.input, tc.expected, matched)
			}
		})
	}
}

func TestEducationalContextPatterns(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "common vulnerability",
			input:    "This is a common vulnerability in web applications",
			expected: true,
		},
		{
			name:     "known attack",
			input:    "A known attack vector for this system",
			expected: true,
		},
		{
			name:     "OWASP reference",
			input:    "According to OWASP guidelines",
			expected: true,
		},
		{
			name:     "CVE reference",
			input:    "This is related to CVE-2024-1234",
			expected: true,
		},
		{
			name:     "CVE with longer number",
			input:    "Fixed in response to CVE-2023-12345",
			expected: true,
		},
		{
			name:     "CWE reference",
			input:    "CWE-79 Cross-site Scripting",
			expected: true,
		},
		{
			name:     "security testing",
			input:    "During security testing we found",
			expected: true,
		},
		{
			name:     "penetration testing",
			input:    "penetration testing revealed",
			expected: true,
		},
		{
			name:     "detecting threats",
			input:    "Methods for detecting the threat",
			expected: true,
		},
		{
			name:     "understanding vulnerability",
			input:    "understanding the vulnerability helps",
			expected: true,
		},
		{
			name:     "no educational context",
			input:    "Just some random content here",
			expected: false,
		},
	}

	educationalPatterns := make([]AllowlistPattern, 0)
	for _, p := range AllowlistPatterns {
		if p.Type == ContextEducational {
			educationalPatterns = append(educationalPatterns, p)
		}
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			matched := false
			for _, pattern := range educationalPatterns {
				if pattern.Regex.MatchString(tc.input) {
					matched = true
					break
				}
			}
			if matched != tc.expected {
				t.Errorf("Input %q: expected match=%v, got match=%v", tc.input, tc.expected, matched)
			}
		})
	}
}

func TestMitigationWeight(t *testing.T) {
	tests := []struct {
		contextType ContextType
		expected    int
	}{
		{ContextDefensive, 3},
		{ContextEducational, 2},
		{ContextDocumentation, 1},
		{ContextType("unknown"), 0},
		{ContextType(""), 0},
	}

	for _, tc := range tests {
		t.Run(string(tc.contextType), func(t *testing.T) {
			got := tc.contextType.MitigationWeight()
			if got != tc.expected {
				t.Errorf("MitigationWeight() for %q: expected %d, got %d", tc.contextType, tc.expected, got)
			}
		})
	}
}
