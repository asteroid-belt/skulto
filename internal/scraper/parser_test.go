package scraper

import (
	"strings"
	"testing"
)

// Test fixtures inlined to avoid external file dependencies
const validSkillMD = `---
name: Advanced Python Testing with pytest
description: Write comprehensive pytest tests with fixtures, markers, and plugins for production-grade Python applications.
metadata:
  version: 1.0.0
  author: Alice Developer
  license: MIT
tags:
  - python
  - testing
  - pytest
---

# Advanced Python Testing with pytest

This skill teaches you how to write production-grade tests using pytest.

## Prerequisites

- Python 3.8+
- Basic understanding of unit testing

## Getting Started

Install pytest:

` + "```bash" + `
pip install pytest
` + "```" + `

## Writing Your First Test

` + "```python" + `
def test_addition():
    assert 1 + 1 == 2
` + "```" + `
`

const minimalSkillMD = `# Getting Started with Docker

Docker is a platform for developing, shipping, and running applications in containers.

## What is Docker?

Docker allows you to package your application and its dependencies into a container.

## Installation

Follow the official Docker installation guide for your operating system.
`

const invalidSkillMD = `---
name: ""
description: Missing title edge case
---

Some content without an H1 heading.

## No H1 Heading

This skill has an H2 heading but no H1 heading, and the frontmatter name is empty.
`

// TestNewSkillParser tests parser creation
func TestNewSkillParser(t *testing.T) {
	parser := NewSkillParser()
	if parser == nil {
		t.Fatal("NewSkillParser returned nil")
	}
	if parser.md == nil {
		t.Fatal("Parser markdown instance is nil")
	}
}

// TestParseWithFullFrontmatter tests parsing a skill with complete YAML frontmatter
func TestParseWithFullFrontmatter(t *testing.T) {
	content := validSkillMD

	parser := NewSkillParser()
	skillFile := &SkillFile{
		ID:       "abc123def456",
		Path:     ".claude/SKILL.md",
		RepoName: "example/repo",
		Owner:    "example",
		Repo:     "repo",
		URL:      "https://github.com/example/repo",
		SHA:      "abc123",
	}

	skill, err := parser.Parse(content, skillFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify core fields
	if skill.ID != skillFile.ID {
		t.Errorf("Expected ID %q, got %q", skillFile.ID, skill.ID)
	}

	if skill.Title != "Advanced Python Testing with pytest" {
		t.Errorf("Expected title 'Advanced Python Testing with pytest', got %q", skill.Title)
	}

	if skill.Description != "Write comprehensive pytest tests with fixtures, markers, and plugins for production-grade Python applications." {
		t.Errorf("Expected description from frontmatter, got %q", skill.Description)
	}

	// Verify metadata fields
	if skill.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %q", skill.Version)
	}

	if skill.Author != "Alice Developer" {
		t.Errorf("Expected author 'Alice Developer', got %q", skill.Author)
	}

	if skill.License != "MIT" {
		t.Errorf("Expected license 'MIT', got %q", skill.License)
	}

	// Verify slug generation
	if skill.Slug != "advanced-python-testing-with-pytest" {
		t.Errorf("Expected slug 'advanced-python-testing-with-pytest', got %q", skill.Slug)
	}

	// Verify embedding ID generation
	if len(skill.EmbeddingID) != 16 {
		t.Errorf("Expected embedding ID length 16, got %d", len(skill.EmbeddingID))
	}

	// Verify source ID is set correctly
	if skill.SourceID == nil || *skill.SourceID != "example/repo" {
		t.Errorf("Expected source ID 'example/repo'")
	}
}

// TestParseWithoutFrontmatter tests parsing a skill without YAML frontmatter
func TestParseWithoutFrontmatter(t *testing.T) {
	content := minimalSkillMD

	parser := NewSkillParser()
	skillFile := &SkillFile{
		ID:       "xyz789abc",
		Path:     "SKILL.md",
		RepoName: "docker/docs",
		Owner:    "docker",
		Repo:     "docs",
		URL:      "https://github.com/docker/docs",
		SHA:      "xyz789",
	}

	skill, err := parser.Parse(content, skillFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should extract title from H1 heading
	if skill.Title != "Getting Started with Docker" {
		t.Errorf("Expected title 'Getting Started with Docker', got %q", skill.Title)
	}

	// Should extract description from content
	if skill.Description == "" {
		t.Error("Expected description to be extracted from content, got empty string")
	}

	// Should generate slug from extracted title
	if skill.Slug != "getting-started-with-docker" {
		t.Errorf("Expected slug 'getting-started-with-docker', got %q", skill.Slug)
	}

	// Metadata fields should be empty since there's no frontmatter
	if skill.Version != "" {
		t.Errorf("Expected empty version, got %q", skill.Version)
	}

	if skill.Author != "" {
		t.Errorf("Expected empty author, got %q", skill.Author)
	}

	if skill.License != "" {
		t.Errorf("Expected empty license, got %q", skill.License)
	}
}

// TestParseWithInvalidFrontmatter tests handling of edge cases
func TestParseWithInvalidFrontmatter(t *testing.T) {
	content := invalidSkillMD

	parser := NewSkillParser()
	skillFile := &SkillFile{
		ID:       "edge123case",
		Path:     "skill.md",
		RepoName: "test/edge",
		Owner:    "test",
		Repo:     "edge",
		URL:      "https://github.com/test/edge",
		SHA:      "edge123",
	}

	skill, err := parser.Parse(content, skillFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should fall back to extracting H2 heading when name is empty
	if skill.Title != "No H1 Heading" {
		t.Errorf("Expected title 'No H1 Heading', got %q", skill.Title)
	}

	// Should extract description from frontmatter even if name is empty
	if skill.Description != "Missing title edge case" {
		t.Errorf("Expected description 'Missing title edge case', got %q", skill.Description)
	}
}

// TestGenerateSlug tests slug generation from various titles
func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Advanced Python Testing with pytest",
			expected: "advanced-python-testing-with-pytest",
		},
		{
			input:    "Getting Started with Docker",
			expected: "getting-started-with-docker",
		},
		{
			input:    "C++ Memory Management & Smart Pointers",
			expected: "c-memory-management-smart-pointers",
		},
		{
			input:    "This is a Very Long Title That Should Be Truncated to Fifty Characters Max",
			expected: "this-is-a-very-long-title-that-should-be-truncated",
		},
		{
			input:    "Title    with    extra    spaces",
			expected: "title-with-extra-spaces",
		},
		{
			input:    "---dashes---at---edges---",
			expected: "dashes-at-edges",
		},
		{
			input:    "UPPERCASE TO LOWERCASE",
			expected: "uppercase-to-lowercase",
		},
		{
			input:    "Numbers123AndSpecial!@#$%Chars",
			expected: "numbers123andspecialchars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := generateSlug(tt.input)
			if result != tt.expected {
				t.Errorf("generateSlug(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestGenerateSkillID tests ID generation consistency
func TestGenerateSkillID(t *testing.T) {
	// Same input should produce same ID
	id1 := generateSkillID("owner/repo", "path/to/skill.md")
	id2 := generateSkillID("owner/repo", "path/to/skill.md")

	if id1 != id2 {
		t.Errorf("generateSkillID produced different IDs for same input: %s vs %s", id1, id2)
	}

	// Different input should produce different ID
	id3 := generateSkillID("different/repo", "path/to/skill.md")
	if id1 == id3 {
		t.Errorf("generateSkillID produced same ID for different input")
	}

	// ID should be 16 characters
	if len(id1) != 16 {
		t.Errorf("generateSkillID produced ID of length %d, expected 16", len(id1))
	}

	// ID should be hex string
	for _, ch := range id1 {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			t.Errorf("generateSkillID produced non-hex character: %c", ch)
		}
	}
}

// TestExtractFirstHeading tests heading extraction
func TestExtractFirstHeading(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "H1 heading",
			content:  "# Main Title\n\nSome content",
			expected: "Main Title",
		},
		{
			name:     "H2 heading",
			content:  "## Secondary Title\n\nSome content",
			expected: "Secondary Title",
		},
		{
			name:     "H1 preferred over H2",
			content:  "## Not Main\n# Main Title\n## Secondary",
			expected: "Not Main",
		},
		{
			name:     "Heading with special characters",
			content:  "# Title with C++ & @Special #Chars!",
			expected: "Title with C++ & @Special #Chars!",
		},
		{
			name:     "No heading",
			content:  "Just some text without headings",
			expected: "Untitled Skill",
		},
		{
			name:     "Frontmatter skip",
			content:  "---\nname: frontmatter\n---\n# Real Title",
			expected: "Real Title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFirstHeading(tt.content)
			if result != tt.expected {
				t.Errorf("extractFirstHeading: got %q, expected %q", result, tt.expected)
			}
		})
	}
}

// TestExtractDescription tests description extraction
func TestExtractDescription(t *testing.T) {
	tests := []struct {
		name    string
		content string
		minLen  int
		hasText string
	}{
		{
			name:    "Extract first paragraph",
			content: "# Title\n\nFirst line of description. Second line. Third line.",
			minLen:  10,
			hasText: "First line",
		},
		{
			name:    "Stop at code block",
			content: "# Title\n\nDescription starts here.\n```python\ncode\n```",
			minLen:  5,
			hasText: "Description",
		},
		{
			name:    "Stop at list",
			content: "# Title\n\nDescription text here.\n- List item\n- Another item",
			minLen:  5,
			hasText: "Description",
		},
		{
			name:    "Handle empty content",
			content: "# Title",
			minLen:  0,
		},
		{
			name:    "Truncate long description",
			content: "# Title\n\n" + strings.Repeat("A", 300),
			minLen:  197,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractDescription(tt.content)
			if len(result) < tt.minLen {
				t.Errorf("extractDescription: got length %d, expected at least %d", len(result), tt.minLen)
			}
			if tt.hasText != "" && !strings.Contains(result, tt.hasText) {
				t.Errorf("extractDescription: expected to contain %q, got %q", tt.hasText, result)
			}
		})
	}
}

// TestParseIntegration tests complete parsing workflow with all fixture types
func TestParseIntegration(t *testing.T) {
	fixtures := map[string]string{
		"valid_skill.md":   validSkillMD,
		"minimal_skill.md": minimalSkillMD,
		"invalid_skill.md": invalidSkillMD,
	}

	parser := NewSkillParser()

	for name, content := range fixtures {
		t.Run(name, func(t *testing.T) {
			skillFile := &SkillFile{
				ID:       generateSkillID("test/repo", name),
				Path:     name,
				RepoName: "test/repo",
				Owner:    "test",
				Repo:     "repo",
				URL:      "https://github.com/test/repo",
				SHA:      "abc123",
			}

			skill, err := parser.Parse(content, skillFile)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Basic validation
			if skill.ID == "" {
				t.Error("Skill ID is empty")
			}

			if skill.Title == "" {
				t.Error("Skill title is empty")
			}

			if skill.Slug == "" {
				t.Error("Skill slug is empty")
			}

			if skill.Content != content {
				t.Error("Skill content doesn't match input")
			}

			if skill.EmbeddingID == "" {
				t.Error("Skill embedding ID is empty")
			}
		})
	}
}

// BenchmarkGenerateSlug benchmarks slug generation
func BenchmarkGenerateSlug(b *testing.B) {
	title := "Advanced Python Testing with pytest and fixtures"
	for i := 0; i < b.N; i++ {
		generateSlug(title)
	}
}

// BenchmarkGenerateSkillID benchmarks ID generation
func BenchmarkGenerateSkillID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateSkillID("owner/repo", "path/to/skill.md")
	}
}

// BenchmarkParse benchmarks full parsing
func BenchmarkParse(b *testing.B) {
	content := validSkillMD

	parser := NewSkillParser()
	skillFile := &SkillFile{
		ID:       "benchmark123",
		Path:     "SKILL.md",
		RepoName: "test/repo",
		Owner:    "test",
		Repo:     "repo",
		URL:      "https://github.com/test/repo",
		SHA:      "abc123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(content, skillFile)
	}
}
